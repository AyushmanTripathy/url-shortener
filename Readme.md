# URL Shortener

__PS:__ Docs are generated, code is hand-written :)

## Requirements

- 1 billion redirects per day (~11.6k per sec)
- Redirect P99 latency < 100ms (or as fast as I can make it)
- No CDN Caching, Only Anycast DNS + Edge VMs
- No permanent redirects as we need analytics

```sh
# Fetching an alias
GET /<alias>

# Creating an alias
POST /

# Updating an alias
PATCH /
```

## Revisions

### Revision 4 (Edge nodes no longer talk to the DB)

- Edge VMs no longer hold a PostgreSQL connection (`DB_URL` removed from edge config)
- On cache miss, Edge VM calls Master VM over HTTP
- Master VM exposes a new `GET /:alias` endpoint, which returns the destination URL as plain text
- Master VM gets its own in-process cache
    - Read-through on `GET /:alias`
    - Write-through on `POST /` and `PATCH /`
- Cache TTL raised from 10 minutes to 24 hours, as the invalidation events from NATS keep Edge caches fresh
- Reduces DB connections from (edges + master) to just master, and shields the DB from cache-miss storms across edges
- Logging (NATS `LOGS`) and invalidation (NATS `ALIAS_UPDATIONS`) flows are unchanged

```mermaid
flowchart TD
    subgraph Clients["Clients"]
        Admin(["Admin User"])
        Internet(["Internet User"])
    end

    subgraph EdgeRegions["Edge Regions"]
        Redirector["Redirector Service"]
        EdgeCache[("In-Process Cache")]
        Batcher["Batching Service"]
        Redirector <--> EdgeCache
        Redirector -.->|"Buffer log"| Batcher
    end

    subgraph MasterRegion["Master Region"]
        Master["Master Instance (CRUD, Lookup & Ingester)"]
        MasterCache[("In-Process Cache")]
        DB[("Database (PostgreSQL)")]

        subgraph JetStream["NATS JetStream Broker"]
            StreamAlias[["Stream: ALIAS_UPDATIONS (alias.updates)"]]
            StreamLogs[["Stream: LOGS (logs.redirects)"]]
        end

        Master <--> MasterCache
        Master <--> DB
        Master -->|"Publish alias update"| StreamAlias
        StreamLogs -->|"Deliver log batches"| Master
    end

    Admin -->|"POST / PATCH alias"| Master
    Internet -->|"GET /:alias"| Redirector

    Redirector -.->|"HTTP GET /:alias on cache miss"| Master
    Batcher -->|"Batched flush (2s ticker)"| StreamLogs
    StreamAlias -->|"Durable consumer"| Redirector
```

#### Redirection Flow

1. Users on Internet send `GET /:alias`, received by Edge VM.
2. Edge VM checks its local in-process cache (LRU, 100k capacity, 24h TTL).
3. On a cache miss, Edge VM sends `GET /:alias` to Master VM.
4. Master VM checks its own cache and, only if that also misses, queries PostgreSQL and populates its cache.
5. Master VM returns the destination URL in the response body.
6. Edge VM populates its cache and responds with `302 Found`.
7. If Master is unreachable, or returns `5xx` or any status other than `200`/`404`, Edge VM responds `503` and does not cache the result; if Master returns `404` Edge VM caches a negative entry (empty value) and responds `404`. Later requests for that alias are answered `404` from the Edge cache without calling Master, until the entry expires (24h) or a `POST` invalidation event clears it.
8. Log buffering and batched flush to NATS are unchanged from Revision 3.

```mermaid
sequenceDiagram
    autonumber
    actor User as Internet User
    participant Edge as Edge VM
    participant ECache as Edge Cache
    participant Master as Master VM
    participant MCache as Master Cache
    participant DB as Master DB (PostgreSQL)
    participant NATS as NATS JetStream (LOGS)

    User->>+Edge: GET /:alias
    Edge->>ECache: Check alias entry
    alt Edge cache hit (URL)
        ECache-->>Edge: Return destination URL
        Edge-->>User: 302 Found (Location: destination URL)
    else Edge cache hit (negative entry)
        ECache-->>Edge: Return empty value
        Edge-->>User: 404 Not Found
    else Edge cache miss
        Edge->>+Master: HTTP GET /:alias
        Master->>MCache: Check alias entry
        alt Master cache miss
            Master->>+DB: SELECT destination FROM redirects WHERE code = alias
            alt DB error
                DB-->>Master: Error
                Master-->>Edge: 503 (not cached)
            else Row found or not found
                DB-->>-Master: Destination URL or no row
                Master->>MCache: Populate cache (alias -> URL, or empty for no row)
            end
        end
        alt Master unreachable, 5xx or unexpected status
            Edge-->>User: 503 Service Unavailable (not cached)
        else Alias not found
            Master-->>Edge: 404
            Edge->>ECache: Populate negative entry (alias -> empty)
            Edge-->>User: 404 Not Found
        else Alias found
            Master-->>-Edge: 200 (destination URL as body)
            Edge->>ECache: Populate cache (alias -> URL)
            Edge-->>User: 302 Found (Location: destination URL)
        end
    end
    deactivate Edge

    Note over Edge,NATS: Asynchronous log batching (unchanged)
    Edge->>Edge: Buffer log entry in memory
    opt Every 2 seconds (Batch Timer)
        Edge-)NATS: Flush batch to logs.redirects
    end
```

#### Alias Entry Update Flow

1. Admin User performs REST (`POST /` or `PATCH /`) to Master VM.
2. Master VM validates request body (alias length >= 3, letters/digits only, valid destination URL).
3. Master VM inserts (`POST`) or updates (`PATCH`) the alias entry in PostgreSQL; failure returns `422`.
4. Master VM writes the new value into its own cache (write-through).
5. Master VM publishes an invalidation event to `ALIAS_UPDATIONS` (`alias.updates`) for both `POST` and `PATCH`. For `POST` this clears any negatively-cached (`404`) entry on Edges.
6. Master VM responds `201 Created` (`POST`) or `200 OK` (`PATCH`), or `202 Accepted` if the event publish failed.
7. NATS JetStream delivers the event to every Edge VM's durable consumer, which removes the alias from its cache and acks. The next request refetches from Master, which already holds the fresh value.

```mermaid
sequenceDiagram
    autonumber
    actor Admin as Admin User
    participant Master as Master VM
    participant MCache as Master Cache
    participant DB as Master DB (PostgreSQL)
    participant NATS as NATS JetStream (ALIAS_UPDATIONS)
    participant Edge as Edge VM
    participant ECache as Edge Cache

    Admin->>+Master: POST / or PATCH / (alias, redirectsTo)
    Master->>Master: Validate payload
    alt Invalid request
        Master-->>Admin: 400 Bad Request
    else Valid request
        Master->>+DB: INSERT (POST) / UPDATE (PATCH) redirects
        alt DB error
            DB-->>Master: Error
            Master-->>Admin: 422 Unprocessable Entity
        else DB success
            DB-->>-Master: Confirm write
            Master->>MCache: Set alias -> URL (write-through)
            Master-)NATS: Publish alias name to alias.updates
            alt Publish success
                Master-->>Admin: 201 Created (POST) / 200 OK (PATCH)
            else Publish failure
                Master-->>-Admin: 202 Accepted (DB updated, invalidation not sent)
            end
            NATS-)Edge: Deliver message (alias) to durable consumer
            Edge->>ECache: Invalidate alias
            Edge-)NATS: Ack message
        end
    end

    Note over Edge,Master: Next GET on Edge misses cache and refetches the fresh value from Master cache
```

#### Negative caching

- Master `GET /:alias` now returns `404` for an unknown alias (and `503` on a DB error); Edge maps `404` to `404`, and any error or unexpected (non-`200`/`404`) status to `503` without caching.
- Unknown aliases are cached as an empty value on both Edge and Master (`Cache.Get` returns `found` and `valid` separately), so repeated lookups of a missing alias don't reach the DB.

### Revision 3 (Messaging System instead of Queue)

- NATS Messaging system instead of message queue
- with JetStream to add persistence
- for alias updates,
    - MQs (RabbitMQ) require awkward Exchange with Fanout
    - NATS supports multiple consumer delivery on single stream
- for redirect logs,
    - Kafka suits long persistence or multiple consumers on high load
    - Consumer is always single master
    - Long retention is not required
    - Instead, edge nodes will perform batching

```mermaid
flowchart TD
    subgraph Clients["Clients"]
        Admin(["Admin User"])
        Internet(["Internet User"])
    end

    subgraph EdgeRegions["Edge Regions"]
        Redirector["Redirector Service"]
        Cache[("In-Process Cache")]
        Batcher["Batching Service"]
        Redirector <--> Cache
        Redirector -.->|"Buffer log"| Batcher
    end

    subgraph MasterRegion["Master Region"]
        Master["Master Instance (CRUD & Ingester)"]
        DB[("Database (PostgreSQL)")]

        subgraph JetStream["NATS JetStream Broker"]
            StreamAlias[["Stream: ALIAS_UPDATIONS (alias.updates)"]]
            StreamLogs[["Stream: LOGS (logs.redirects)"]]
        end

        Master <--> DB
        Master -->|"Publish alias update"| StreamAlias
        StreamLogs -->|"Deliver log batches"| Master
    end

    Admin -->|"POST / PATCH alias"| Master
    Internet -->|"GET /:alias"| Redirector

    Redirector -.->|"Query on cache miss"| DB
    Batcher -->|"Batched flush (2s ticker)"| StreamLogs
    StreamAlias -->|"Durable consumer"| Redirector
```

#### Redirection Flow

1. Users on Internet send `GET /:alias`, received by Edge VM.
2. Edge VM checks its local in-process cache (sync.Map, 100k capacity).
3. If not found, Edge VM retrieves the alias entry from Master DB and populates its local cache.
4. Edge VM immediately responds with `302 Found` (with destination URL) to the user.
5. In the background, the request log is buffered in memory.
6. A background ticker (every 2s) prepends a JSON header (`count`, `edge` hostname) and flushes the batched logs to NATS JetStream `logs.redirects`.
7. Master VM consumer receives log batches and performs a single bulk multi-row `INSERT INTO logs`.
8. Master VM acknowledges the message batch to JetStream.

```mermaid
sequenceDiagram
    autonumber
    actor User as Internet User
    participant Edge as Edge VM
    participant Cache as In-Process Cache
    participant DB as Master DB (PostgreSQL)
    participant NATS as NATS JetStream (LOGS)
    participant Master as Master VM

    User->>+Edge: GET /:alias
    Edge->>Cache: Check alias entry
    alt Cache hit
        Cache-->>Edge: Return destination URL
    else Cache miss
        Edge->>+DB: SELECT destination FROM redirects WHERE code = alias
        DB-->>-Edge: Return destination URL
        Edge->>Cache: Populate cache (alias -> URL)
    end
    Edge-->>-User: 302 Found (Location: destination URL)

    Note over Edge,Master: Asynchronous Log Batching & Persistence
    Edge->>Edge: Buffer log entry in memory
    opt Every 2 seconds (Batch Timer)
        Edge-)NATS: Flush batch with header (Count, Edge Name) to logs.redirects
        NATS-)Master: Deliver message batch to Master consumer
        Master->>+DB: Bulk INSERT INTO logs (code, ip, edge, user_agent)
        DB-->>-Master: Confirm batch insert
        Master-)NATS: Ack message
    end
```

#### Alias Entry Update Flow

1. Admin User performs REST (`POST /` or `PATCH /`) to Master VM.
2. Master VM validates request body (alias length >= 3, valid destination URL).
3. Master VM upserts alias entry into Master PostgreSQL DB.
4. Master VM publishes cache invalidation event to NATS JetStream stream `ALIAS_UPDATIONS` on subject `alias.updates`.
5. Master VM responds to Admin User (`200 OK` / `201 Created`, or `202 Accepted` if event publishing failed).
6. NATS JetStream delivers invalidation event to Edge VM's durable consumer.
7. Edge VM removes the alias from its in-process cache and acknowledges the message to JetStream.

```mermaid
sequenceDiagram
    autonumber
    actor Admin as Admin User
    participant Master as Master VM
    participant DB as Master DB (PostgreSQL)
    participant NATS as NATS JetStream (ALIAS_UPDATIONS)
    participant Edge as Edge VM

    Admin->>+Master: POST / or PATCH / (alias, redirectsTo)
    Master->>Master: Validate payload (length >= 3, valid URL)
    alt Invalid request
        Master-->>Admin: 400 Bad Request
    else Valid request
        Master->>+DB: Upsert alias entry into redirects table
        DB-->>-Master: Confirm upsert
        Master-)NATS: Publish alias name to alias.updates
        alt JetStream publish success
            Master-->>Admin: 200 OK / 201 Created
        else JetStream publish failure
            Master-->>Admin: 202 Accepted (DB updated, invalidation queued)
        end

        NATS-)Edge: Deliver message (alias) to durable consumer
        Edge->>Edge: Invalidate alias from local cache
        Edge-)NATS: Ack message
    end
```

### Revision 2 (Message Queue instead of WebSockets)

- 2 queues for redirect logs & alias updates in Master region
- For redirect logs,
    - log redundancy in case of Master node failure
    - batching of redirect logs before write
- For alias updates,
    - Needs Exchange with Fanout strategy

```mermaid
flowchart TD
    subgraph Clients["Clients"]
        User(["Admin User"])
        Internet(["Internet User"])
    end

    subgraph EdgeRegions["Edge Regions"]
        Redirector["Redirector Instances"]
        Cache[("In-Process Cache")]
        Redirector <--> Cache
    end

    subgraph MasterRegion["Master Region"]
        Master["Master Instance (CRUD)"]
        DB[("Database (PostgreSQL)")]

        subgraph Queues["Message Queues (RabbitMQ)"]
            LogQueue[["Redirect Log Queue"]]
            UpdateExchange[["Alias Update Fanout Exchange"]]
        end

        Master <--> DB
        Master -->|"Publish alias updates"| UpdateExchange
        LogQueue -->|"Deliver batched logs"| Master
    end

    User -->|"POST / PATCH alias"| Master
    Internet -->|"GET /:alias"| Redirector
    Redirector -->|"302 Found"| Internet

    Redirector -.->|"Retrieve on cache miss"| DB
    Redirector -->|"Push request logs"| LogQueue
    UpdateExchange -->|"Fanout cache invalidations"| Redirector
```

#### Redirection Flow

1. Users on Internet `GET /:alias`, received by Edge VM.
2. Edge VM checks its cache for the alias entry.
3. If not found, Edge VM retrieves alias entry from Master DB and populates local cache.
4. Edge VM responds with 302 redirect to Internet traffic.
5. Edge VM pushes log to Message queue asynchronously.
6. Master VM receives logs from Message Queue in batches.
7. Master VM writes these batched logs to Master DB.

```mermaid
sequenceDiagram
    autonumber
    actor User as Internet User
    participant Edge as Edge VM
    participant Cache as In-Process Cache
    participant DB as Master DB (PostgreSQL)
    participant Queue as Redirect Log Queue
    participant Master as Master VM

    User->>+Edge: GET /:alias
    Edge->>Cache: Check alias entry
    alt Cache hit
        Cache-->>Edge: Return destination URL
    else Cache miss
        Edge->>+DB: Retrieve alias entry
        DB-->>-Edge: Return destination URL
        Edge->>Cache: Populate cache (alias -> URL)
    end
    Edge-->>-User: 302 Found (Redirect to destination)

    Note over Edge,Master: Asynchronous logging decoupled from redirect
    Edge-)Queue: Push request log
    Queue->>+Master: Deliver logs in batches
    Master->>+DB: Write log batch
    DB-->>-Master: Confirm batch write
    Master-->>-Queue: Ack batch
```

#### Alias Entry Update Flow

1. Admin User performs REST (POST/PATCH) to Master VM.
2. Master VM validates request body and upserts to Master DB.
3. Master VM pushes alias cache deletion event to Message queue (Fanout Exchange).
4. Master VM responds to Admin User immediately.
5. Edge VMs receive deletion event from Fanout Exchange.
6. Edge VMs delete cache entry and ack to message queue.

```mermaid
sequenceDiagram
    autonumber
    actor Admin as Admin User
    participant Master as Master VM
    participant DB as Master DB (PostgreSQL)
    participant Exchange as Alias Update Fanout Exchange
    participant Edge as Edge VMs

    Admin->>+Master: POST / PATCH alias entry
    Master->>Master: Validate request body
    alt Invalid request
        Master-->>Admin: 400 Bad Request (Validation error)
    else Valid request
        Master->>+DB: Upsert alias entry
        DB-->>-Master: Confirm upsert
        Master-)Exchange: Publish cache deletion event
        Master-->>-Admin: 200 OK / 201 Created

        Note over Exchange,Edge: Exchange broadcasts deletion event to all Edge queues
        Exchange-)Edge: Deliver deletion event
        Edge->>Edge: Delete alias from cache
        Edge-->>Exchange: Acknowledge deletion event
    end
```

### Revision 1 (WebSockets & In-process caching)

- In-Process cache following Cache-aside

```mermaid
flowchart TD
    subgraph Clients["Clients"]
        User(["Admin User"])
        Internet(["Internet User"])
    end

    subgraph EdgeRegions["Edge Regions"]
        Redirector["Redirector Instances"]
        Cache[("In-Process Cache")]
        Redirector <--> Cache
    end

    subgraph MasterRegion["Master Region"]
        Master["Master Instance (CRUD)"]
        DB[("Database (PostgreSQL)")]
        Master <--> DB
    end

    User -->|"POST / PATCH alias"| Master
    Internet -->|"GET /:alias"| Redirector
    Redirector -->|"302 Found"| Internet

    Redirector -.->|"Retrieve on cache miss"| DB
    Redirector -->|"Stream logs over WebSocket"| Master
    Master -->|"Broadcast cache invalidation over WebSocket"| Redirector
```

#### Redirection Flow

1. Users on Internet `GET /:alias`, received by Edge VM.
2. Edge VM checks its cache for the alias entry.
3. If not found, Edge VM retrieves alias entry from Master DB and populates local cache.
4. Edge VM responds with 302 redirect to Internet traffic.
5. Edge VM sends log for this request to Master VM through WebSockets.
6. Master VM writes these logs in batches to Master DB.

```mermaid
sequenceDiagram
    autonumber
    actor User as Internet User
    participant Edge as Edge VM
    participant Cache as In-Process Cache
    participant DB as Master DB (PostgreSQL)
    participant Master as Master VM

    User->>+Edge: GET /:alias
    Edge->>Cache: Check alias entry
    alt Cache hit
        Cache-->>Edge: Return destination URL
    else Cache miss
        Edge->>+DB: Retrieve alias entry
        DB-->>-Edge: Return destination URL
        Edge->>Cache: Populate cache (alias -> URL)
    end
    Edge-->>-User: 302 Found (Redirect to destination)

    Edge-)Master: Stream request log over WebSocket
    Master->>+DB: Write request logs in batches
    DB-->>-Master: Confirm batch write
```

#### Alias Entry Update Flow

1. Admin User performs REST (POST/PATCH) to Master VM.
2. Master VM validates request body and upserts to Master DB.
3. Master VM sends alias cache deletion event to Edge VMs via WebSockets.
4. Edge VMs send ack and Master VM marks the update as complete.

```mermaid
sequenceDiagram
    autonumber
    actor Admin as Admin User
    participant Master as Master VM
    participant DB as Master DB (PostgreSQL)
    participant Edge as Edge VMs

    Admin->>+Master: POST / PATCH alias entry
    Master->>Master: Validate request body
    alt Invalid request
        Master-->>Admin: 400 Bad Request (Validation error)
    else Valid request
        Master->>+DB: Upsert alias entry
        DB-->>-Master: Confirm upsert
        Master->>Edge: Broadcast alias cache deletion over WebSockets
        Edge->>Edge: Delete alias from cache
        Edge-->>Master: Acknowledge cache deletion
        Note over Master,Edge: Master blocks until all Edge VMs acknowledge
        Master-->>-Admin: 200 OK (Update complete)
    end
```

### Revision 0

Single Go executable with SQLite.

### Planned Optimizations

- [X] Batch logs in edge
- [X] Header to replace duplicated payload from logs
- [X] Replace JSON serialization for logs
- [X] Reduce Edge database calls with requests to Master
- [ ] NATs network partitioning
- [ ] Cache miss leading to initial request brust
- [ ] Handling 500s from master, without brusts
- [ ] Workaround for NATS 1mb message size restriction
- [ ] Persist batched logs on disk and retry to handle NATS network partition
- [ ] Seprate LSM Style DB for redirect logs
- [ ] Prefiling Edge cache?
