# ==========================================
# STAGE 1: Build the binary
# ==========================================
FROM golang:1.27-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /app

COPY go.mod go.sum* ./
RUN go mod download

COPY . .

# Build the statically linked Go binary
# CGO_ENABLED=0 ensures the binary doesn't rely on host system C libraries
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o master ./cmds/master

# ==========================================
# STAGE 2: Run the binary
# ==========================================
FROM scratch 

WORKDIR /app

COPY --from=builder /app/master .

EXPOSE 8080

CMD ["./master"]
