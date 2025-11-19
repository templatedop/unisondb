# UnisonDB gRPC Replication Demo

This example demonstrates:
1. **Writer API** - HTTP API that posts data to UnisonDB
2. **Two gRPC Clients** - Receive replicated data from UnisonDB via gRPC WAL streaming and log it in real-time

## Architecture

```
┌─────────────────┐
│   Writer API    │ (HTTP :8080)
│  (Go Service)   │
└────────┬────────┘
         │ HTTP PUT
         ▼
┌─────────────────┐
│  UnisonDB       │ (HTTP :8001, gRPC :4001)
│   Primary       │
│                 │
└────┬───────┬────┘
     │       │
     │ gRPC  │ gRPC WAL Stream
     │       │
     ▼       ▼
┌─────────┐ ┌─────────┐
│ Client1 │ │ Client2 │
│  :9001  │ │  :9002  │
└─────────┘ └─────────┘
   Logs        Logs
  Changes     Changes
```

## Components

### 1. UnisonDB Primary Server
- Accepts write requests via HTTP API
- Streams WAL records to clients via gRPC
- Ports: HTTP 8001, gRPC 4001

### 2. Writer API
- HTTP server accepting POST requests
- Forwards data to UnisonDB
- Port: 8080

### 3. gRPC Client 1 & Client 2
- Subscribe to UnisonDB WAL stream via gRPC
- Maintain local replicas
- Log all incoming data changes
- Ports: 9001, 9002 (informational)

## Prerequisites

- Go 1.24 or higher
- UnisonDB built (`go build -o unisondb ./cmd/unisondb`)

## Quick Start

### Step 1: Start UnisonDB Primary Server

```bash
# From the repository root
./unisondb --config cmd/examples/grpc-replication-demo/configs/primary.toml replicator
```

You should see:
```
✅ UnisonDB started on HTTP :8001, gRPC :4001
```

### Step 2: Start gRPC Client 1

Open a new terminal:

```bash
# From the repository root
cd cmd/examples/grpc-replication-demo/grpc-client1
go run main.go
```

You should see:
```
🚀 Starting GRPC-CLIENT-1
🔗 Upstream: localhost:4001
✅ Connected to upstream UnisonDB
🔄 Starting replication stream...
```

### Step 3: Start gRPC Client 2

Open another new terminal:

```bash
# From the repository root
cd cmd/examples/grpc-replication-demo/grpc-client2
go run main.go
```

You should see:
```
🚀 Starting GRPC-CLIENT-2
🔗 Upstream: localhost:4001
✅ Connected to upstream UnisonDB
🔄 Starting replication stream...
```

### Step 4: Start Writer API

Open another terminal:

```bash
# From the repository root
cd cmd/examples/grpc-replication-demo/writer-api
go run main.go
```

You should see:
```
🚀 Writer API starting on port 8080
📝 POST /write - Write data to UnisonDB
```

### Step 5: Post Data

Now post some data to the Writer API:

```bash
# Terminal 1: Write some data
curl -X POST http://localhost:8080/write \
  -H "Content-Type: application/json" \
  -d '{
    "key": "user:1001",
    "value": "Alice Johnson"
  }'

# Terminal 2: Write more data
curl -X POST http://localhost:8080/write \
  -H "Content-Type: application/json" \
  -d '{
    "key": "user:1002",
    "value": "Bob Smith"
  }'

# Terminal 3: Write even more
curl -X POST http://localhost:8080/write \
  -H "Content-Type: application/json" \
  -d '{
    "key": "product:5001",
    "value": "Laptop Pro 15"
  }'
```

### Step 6: Observe the Logs

Watch the logs in **gRPC Client 1** and **gRPC Client 2** terminals. You should see:

```
[GRPC-CLIENT-1] 📦 Received WAL batch from UnisonDB
[GRPC-CLIENT-1] 📍 Current offset: segment=0, offset=123
```

```
[GRPC-CLIENT-2] 📦 Received WAL batch from UnisonDB
[GRPC-CLIENT-2] 📍 Current offset: segment=0, offset=123
```

Both clients are receiving the same replicated data in real-time!

## Testing the Setup

### Health Check

```bash
# Check Writer API health
curl http://localhost:8080/health
```

### Write Multiple Records

```bash
# Batch write script
for i in {1..10}; do
  curl -X POST http://localhost:8080/write \
    -H "Content-Type: application/json" \
    -d "{\"key\": \"test:$i\", \"value\": \"Test Value $i\"}"
  echo ""
  sleep 0.5
done
```

### Verify Replication

Watch both client terminals - they should both receive all 10 records!

## Advanced Usage

### Custom Upstream Address

```bash
# Connect to a different UnisonDB instance
cd cmd/examples/grpc-replication-demo/grpc-client1
go run main.go -upstream "192.168.1.100:4001"
```

### Custom Namespace

```bash
# Use a different namespace
cd cmd/examples/grpc-replication-demo/grpc-client1
go run main.go -namespace "production"
```

### Custom Data Directory

```bash
# Use a different data directory
cd cmd/examples/grpc-replication-demo/grpc-client1
go run main.go -datadir "/var/lib/unisondb-client1"
```

## How It Works

### 1. Write Flow

1. Client sends POST request to Writer API
2. Writer API encodes value as base64
3. Writer API sends HTTP PUT to UnisonDB
4. UnisonDB writes to WAL and B+Tree storage
5. UnisonDB acknowledges write

### 2. Replication Flow

1. gRPC clients connect to UnisonDB gRPC port (4001)
2. Clients send `StreamWalRecords` request with their current offset
3. UnisonDB streams WAL records in batches
4. Clients receive batches and apply to local replica
5. Clients update their offset and continue streaming

### 3. Real-time Streaming

- Sub-second replication latency
- Automatic reconnection on network failures
- Batched WAL record transmission for efficiency
- Offset tracking for resumable replication

## File Structure

```
grpc-replication-demo/
├── README.md
├── configs/
│   └── primary.toml          # UnisonDB primary config
├── writer-api/
│   └── main.go               # HTTP API for writing
├── grpc-client1/
│   └── main.go               # First gRPC replica client
└── grpc-client2/
    └── main.go               # Second gRPC replica client
```

## API Reference

### Writer API

#### POST /write
Write a key-value pair to UnisonDB

**Request:**
```json
{
  "key": "user:1001",
  "value": "Alice Johnson"
}
```

**Response:**
```json
{
  "success": true,
  "message": "Data written to UnisonDB successfully",
  "key": "user:1001"
}
```

#### GET /health
Health check endpoint

**Response:**
```json
{
  "status": "healthy",
  "service": "writer-api"
}
```

## Troubleshooting

### Client can't connect to UnisonDB

**Error:** `Failed to connect to upstream: connection refused`

**Solution:** Make sure UnisonDB primary is running on port 4001:
```bash
./unisondb --config cmd/examples/grpc-replication-demo/configs/primary.toml replicator
```

### Writer API can't write to UnisonDB

**Error:** `Failed to write to UnisonDB: connection refused`

**Solution:** Make sure UnisonDB HTTP API is running on port 8001. Check the primary.toml configuration.

### Permission denied creating data directory

**Error:** `Failed to create data directory: permission denied`

**Solution:** Run with sudo or change the data directory:
```bash
go run main.go -datadir "$HOME/unisondb-client1"
```

### No logs showing in clients

**Solution:** Make sure you're writing data through the Writer API. The clients only log when they receive WAL records.

## Performance Notes

- Each client maintains an independent local replica
- Replication latency is typically under 100ms
- Can scale to 100+ concurrent replica clients
- WAL batching improves throughput for high write loads

## Next Steps

1. **Add Data Querying**: Extend clients to serve read requests from local replica
2. **Add Metrics**: Integrate Prometheus metrics for monitoring
3. **Add Authentication**: Implement mTLS for gRPC connections
4. **Multi-Namespace**: Replicate multiple namespaces simultaneously
5. **Conflict Resolution**: Implement CRDTs for multi-primary setups

## Related Examples

- [Multi-DC CRDT Example](../crdt-multi-dc/) - Multi-datacenter replication with CRDTs
- [HTTP API Documentation](https://unisondb.io/docs/api/http-api/)
- [Deployment Topologies](https://unisondb.io/docs/deployment/)

## License

Same as UnisonDB - Apache 2.0
