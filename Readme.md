# URL Shortener

## Requirements

- 1 billion redirects per day (~11.6k per sec)
- Redirect latency < 300ms (or as fast as I can make it)
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
        User["User traffic (Admin)"]
        Internet["Internet traffic"]
    end

    subgraph EdgeRegions["Edge Regions"]
        Redirector["Redirector Instances"]
        Cache["In-Process Caching"]
        Redirector <--> Cache
    end

    subgraph MasterRegion["Master Region"]
        Master["Master Instance (CRUD)"]
        MQ["Message Queue"]
        DB[("Database (PostgreSQL)")]

        Master <--> DB
        Master -->|"Alias updates"| MQ
        MQ -->|"Logs"| Master
    end

    User -->|"HTTPS bitly.com"| Master
    Internet -->|"GET bit.ly/<alias>"| Redirector
    Redirector -.->|"302 Redirect"| Internet

    Redirector -.->|"Retrieve on cache miss"| DB
    Redirector -->|"Logs"| MQ
    MQ -->|"Alias updates"| Redirector
```

#### Redirection Flow

1. Users on Internet `GET bit.ly/<alias>`, received by Edge VM.
2. Edge VM checks its cache for the alias entry.
3. If not found, Edge VM retrieves alias entry from Master DB.
4. Edge VM responds with 302 redirect to Internet traffic.
5. Edge VM pushes log to Message queue.
6. Master VM receives logs from Message Queue in batches.
7. Master VM writes these batched logs to Master DB.

```mermaid
sequenceDiagram
    autonumber
    actor User as Internet User
    participant Edge as Edge VM
    participant Cache as Edge Cache
    participant DB as Master DB
    participant Queue as Redirect Log Queue
    participant Master as Master VM

    User->>Edge: GET bit.ly/<alias>
    Edge->>Cache: Check alias entry
    alt Cache hit
        Cache-->>Edge: Return alias entry
    else Cache miss
        Edge->>DB: Retrieve alias entry
        DB-->>Edge: Return alias entry
    end
    Edge-->>User: 302 redirect to destination
    Edge->>Queue: Push request log
    Queue->>Master: Deliver logs in batches
    Master->>DB: Write log batch
```

#### Alias Entry Update Flow

1. Admin User performs REST (POST/PATCH) to Master VM.
2. Master VM validates request body and upserts to Master DB.
3. Master VM pushes alias cache deletion event to Message queue.
4. Master VM responds to Admin User at this point.
5. Edge VMs receive deletion event from Message queue.
6. Edge VMs delete cache entry and ack to message queue.

```mermaid
sequenceDiagram
    autonumber
    actor Admin as Admin User
    participant Master as Master VM
    participant DB as Master DB
    participant Queue as Alias Update Queue
    participant Edge as Edge VMs

    Admin->>Master: POST/PATCH alias entry
    Master->>Master: Validate request body
    alt Invalid request
        Master-->>Admin: Return validation error
    else Valid request
        Master->>DB: Upsert alias entry
        DB-->>Master: Confirm upsert
        Master->>Queue: Push cache deletion event
        Master-->>Admin: Respond to admin
        Queue->>Edge: Deliver deletion event
        Edge->>Edge: Delete alias from cache
        Edge-->>Queue: Acknowledge deletion event
    end
```

### Revision 1 (WebSockets & In-process caching)

- In-Process cache following Cache-aside

```mermaid
flowchart TD
    subgraph Clients["Clients"]
        User["User traffic (Admin)"]
        Internet["Internet traffic"]
    end

    subgraph EdgeRegions["Edge Regions"]
        Redirector["Redirector Instances"]
        Cache["In-Process Caching"]
        Redirector <--> Cache
    end

    subgraph MasterRegion["Master Region"]
        Master["Master Instance (CRUD)"]
        DB[("Database (PostgreSQL)")]
        Master <--> DB
    end

    User -->|"HTTPS bitly.com"| Master
    Internet -->|"GET bit.ly/<alias>"| Redirector
    Redirector -.->|"302 Redirect"| Internet

    Redirector -.->|"Retrieve on cache miss"| DB
    Redirector <-->|"WebSocket"| Master
```

#### Redirection Flow

1. Users on Internet `GET bit.ly/<alias>`, received by Edge VM.
2. Edge VM checks its cache for the alias entry.
3. If not found, Edge VM retrieves alias entry from Master DB.
4. Edge VM responds with 302 redirect to Internet traffic.
5. Edge VM sends log for this request to Master VM through WebSockets.
6. Master VM writes these logs in batches to Master DB.

```mermaid
sequenceDiagram
    autonumber
    actor User as Internet User
    participant Edge as Edge VM
    participant Cache as Edge Cache
    participant DB as Master DB
    participant Master as Master VM

    User->>Edge: GET bit.ly/<alias>
    Edge->>Cache: Check alias entry
    alt Cache hit
        Cache-->>Edge: Return alias entry
    else Cache miss
        Edge->>DB: Retrieve alias entry
        DB-->>Edge: Return alias entry
    end
    Edge-->>User: 302 redirect to destination
    Edge->>Master: Send request log over WebSockets
    Master->>DB: Write request logs in batches
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
    participant DB as Master DB
    participant Edge as Edge VMs

    Admin->>Master: POST/PATCH alias entry
    Master->>Master: Validate request body
    alt Invalid request
        Master-->>Admin: Return validation error
    else Valid request
        Master->>DB: Upsert alias entry
        DB-->>Master: Confirm upsert
        Master->>Edge: Broadcast alias cache deletion over WebSockets
        Edge->>Edge: Delete alias from cache
        Edge-->>Master: Acknowledge cache deletion
        Master-->>Admin: Mark update complete after all Edge VMs acknowledge
    end
```

### Revision 0

Single Go executable with SQLite.

## Optimizations

- [X] Batch logs in edge
- [X] Header to replace duplicated payload from logs
- [ ] Persist batched logs on disk and retry to handle NATS downtime
