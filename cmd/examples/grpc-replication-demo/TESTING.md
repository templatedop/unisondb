# Testing Guide for gRPC Replication Demo

This guide provides comprehensive testing instructions to verify the demo works correctly.

## Prerequisites Check

Before testing, verify you have:

```bash
# Check Go version (need 1.24+)
go version

# Check you're in the repository root
pwd  # Should show /path/to/unisondb

# Verify files exist
ls cmd/examples/grpc-replication-demo/
```

## Step 1: Build All Components

```bash
# Build UnisonDB
go build -o unisondb ./cmd/unisondb

# Verify binary created
./unisondb --version

# Build Writer API
cd cmd/examples/grpc-replication-demo/writer-api
go build -o writer-api main.go
cd ../../../..

# Build gRPC Client 1
cd cmd/examples/grpc-replication-demo/grpc-client1
go build -o grpc-client1 main.go
cd ../../../..

# Build gRPC Client 2
cd cmd/examples/grpc-replication-demo/grpc-client2
go build -o grpc-client2 main.go
cd ../../../..
```

## Step 2: Clean Test Environment

```bash
# Remove old data directories
rm -rf /tmp/unisondb/grpc-demo
rm -rf /tmp/unisondb-client1
rm -rf /tmp/unisondb-client2

# Kill any existing processes
pkill -9 unisondb || true
pkill -9 writer-api || true
pkill -9 grpc-client || true
```

## Step 3: Start UnisonDB Primary

In **Terminal 1**:

```bash
./unisondb --config cmd/examples/grpc-replication-demo/configs/primary.toml replicator
```

**Expected Output:**
```
[unisondb] Starting replicator mode
[unisondb] HTTP server listening on :8001
[unisondb] gRPC server listening on :4001
[unisondb] Namespace registered: demo
```

**Test Endpoints:**
```bash
# In another terminal, test HTTP endpoint
curl http://localhost:8001/api/v1/demo/offset

# Test direct PUT (this is what Writer API will do)
curl -X PUT http://localhost:8001/api/v1/demo/kv/test123 \
  -H "Content-Type: application/json" \
  -d '{"value":"dGVzdA=="}'

# Verify you get success response
# {"success":true}

# Test GET
curl http://localhost:8001/api/v1/demo/kv/test123

# Should return: {"value":"dGVzdA==","found":true}
```

## Step 4: Start gRPC Client 1

In **Terminal 2**:

```bash
cd cmd/examples/grpc-replication-demo/grpc-client1
./grpc-client1
```

**Expected Output:**
```
🚀 Starting GRPC-CLIENT-1
🔗 Upstream: localhost:4001
📁 Data directory: /tmp/unisondb-client1
📚 Namespace: demo
✅ Local replica engine initialized
✅ Connected to upstream UnisonDB
✅ Relayer configured
🔄 Starting replication stream for namespace 'demo'...
📊 Monitoring incoming data from UnisonDB...
─────────────────────────────────────────────
[GRPC-CLIENT-1] 📍 Waiting for first record...
```

## Step 5: Start gRPC Client 2

In **Terminal 3**:

```bash
cd cmd/examples/grpc-replication-demo/grpc-client2
./grpc-client2
```

**Expected Output:**
```
🚀 Starting GRPC-CLIENT-2
🔗 Upstream: localhost:4001
📁 Data directory: /tmp/unisondb-client2
📚 Namespace: demo
✅ Local replica engine initialized
✅ Connected to upstream UnisonDB
✅ Relayer configured
🔄 Starting replication stream for namespace 'demo'...
📊 Monitoring incoming data from UnisonDB...
─────────────────────────────────────────────
[GRPC-CLIENT-2] 📍 Waiting for first record...
```

## Step 6: Start Writer API

In **Terminal 4**:

```bash
cd cmd/examples/grpc-replication-demo/writer-api
./writer-api
```

**Expected Output:**
```
🚀 Writer API starting on port 8080
📝 POST /write - Write data to UnisonDB
❤️  GET /health - Health check
🔗 UnisonDB URL: http://localhost:8001/api/v1
```

**Test Health:**
```bash
# In another terminal
curl http://localhost:8080/health

# Should return: {"service":"writer-api","status":"healthy"}
```

## Step 7: Positive Test Cases

Now let's test the complete flow!

### Test 1: Simple Key-Value Write

```bash
curl -X POST http://localhost:8080/write \
  -H "Content-Type: application/json" \
  -d '{
    "key": "user:1001",
    "value": "Alice Johnson"
  }'
```

**Expected Response:**
```json
{
  "success": true,
  "message": "Data written to UnisonDB successfully",
  "key": "user:1001"
}
```

**Expected in Terminal 1 (UnisonDB):**
```
[httpapi] PUT /api/v1/demo/kv/user:1001
```

**Expected in Terminal 2 (Client 1):**
```
[GRPC-CLIENT-1] 📦 Received WAL batch from UnisonDB
[GRPC-CLIENT-1] 📍 Current offset: segment=0, offset=X
```

**Expected in Terminal 3 (Client 2):**
```
[GRPC-CLIENT-2] 📦 Received WAL batch from UnisonDB
[GRPC-CLIENT-2] 📍 Current offset: segment=0, offset=X
```

**Expected in Terminal 4 (Writer API):**
```
[WRITER] Successfully wrote key=user:1001, value=Alice Johnson
```

✅ **SUCCESS CRITERIA:** All 4 terminals show activity, both clients log the same offset

### Test 2: Multiple Writes in Sequence

```bash
# Write user data
curl -X POST http://localhost:8080/write \
  -H "Content-Type: application/json" \
  -d '{"key": "user:1002", "value": "Bob Smith"}'

sleep 1

# Write product data
curl -X POST http://localhost:8080/write \
  -H "Content-Type: application/json" \
  -d '{"key": "product:5001", "value": "Laptop Pro 15"}'

sleep 1

# Write order data
curl -X POST http://localhost:8080/write \
  -H "Content-Type: application/json" \
  -d '{"key": "order:9001", "value": "Order by Bob"}'
```

✅ **SUCCESS CRITERIA:**
- All 3 writes succeed
- Both clients log 3 separate WAL batches
- Offsets increment for each client

### Test 3: Rapid Writes (Batch Test)

```bash
for i in {1..10}; do
  curl -X POST http://localhost:8080/write \
    -H "Content-Type: application/json" \
    -d "{\"key\": \"test:$i\", \"value\": \"Test Value $i\"}"
  echo ""
done
```

✅ **SUCCESS CRITERIA:**
- All 10 writes succeed
- Clients show batched WAL records
- Both clients show the same final offset

### Test 4: Special Characters and Encoding

```bash
# Test with special characters
curl -X POST http://localhost:8080/write \
  -H "Content-Type: application/json" \
  -d '{
    "key": "special:1",
    "value": "Hello 世界! Testing émojis 🚀"
  }'

# Test with JSON data
curl -X POST http://localhost:8080/write \
  -H "Content-Type: application/json" \
  -d '{
    "key": "json:data",
    "value": "{\"name\":\"Alice\",\"age\":30}"
  }'
```

✅ **SUCCESS CRITERIA:**
- Special characters handled correctly
- JSON data stored as string
- No encoding errors

### Test 5: Large Value

```bash
# Generate a large value (10KB)
LARGE_VALUE=$(python3 -c "print('x' * 10000)")

curl -X POST http://localhost:8080/write \
  -H "Content-Type: application/json" \
  -d "{
    \"key\": \"large:1\",
    \"value\": \"$LARGE_VALUE\"
  }"
```

✅ **SUCCESS CRITERIA:**
- Large value written successfully
- Both clients receive the data
- No truncation or errors

## Step 8: Verify Data Integrity

After all writes, verify the data is replicated correctly:

```bash
# Check data in primary
curl http://localhost:8001/api/v1/demo/kv/user:1001

# Should return: {"value":"QWxpY2UgSm9obnNvbg==","found":true}
# Decode: echo "QWxpY2UgSm9obnNvbg==" | base64 -d
# Result: Alice Johnson

# Verify in client 1 (it has local replica)
# You can add a GET endpoint to the client to query local data

# Check offset synchronization
curl http://localhost:8001/api/v1/demo/offset
```

## Step 9: Resilience Testing

### Test Restart Client 1

1. Stop Client 1 (Ctrl+C in Terminal 2)
2. Write more data via Writer API
3. Restart Client 1
4. Verify it catches up from its last offset

```bash
# Write while client 1 is down
curl -X POST http://localhost:8080/write \
  -H "Content-Type: application/json" \
  -d '{"key": "resilience:1", "value": "Written while client down"}'

# Restart client 1
cd cmd/examples/grpc-replication-demo/grpc-client1
./grpc-client1
```

✅ **SUCCESS CRITERIA:**
- Client 1 reconnects successfully
- Client 1 receives missed records
- Client 1 catches up to Client 2's offset

## Step 10: Performance Testing

Run the automated test script:

```bash
cd /home/user/unisondb
./cmd/examples/grpc-replication-demo/test-write.sh
```

✅ **SUCCESS CRITERIA:**
- All test writes complete successfully
- Both clients log all records
- Response time < 100ms per write

## Negative Test Cases

### Test Invalid Data

```bash
# Missing key
curl -X POST http://localhost:8080/write \
  -H "Content-Type: application/json" \
  -d '{"value": "test"}'

# Expected: {"error":"Key and value are required"}

# Missing value
curl -X POST http://localhost:8080/write \
  -H "Content-Type: application/json" \
  -d '{"key": "test"}'

# Expected: {"error":"Key and value are required"}

# Invalid JSON
curl -X POST http://localhost:8080/write \
  -H "Content-Type: application/json" \
  -d '{invalid json}'

# Expected: {"error":"Invalid request body"}
```

### Test UnisonDB Down

```bash
# Stop UnisonDB (Ctrl+C in Terminal 1)

# Try to write
curl -X POST http://localhost:8080/write \
  -H "Content-Type: application/json" \
  -d '{"key": "test", "value": "data"}'

# Expected: Error indicating connection failure
```

## Validation Checklist

After running all tests, verify:

- [ ] UnisonDB primary accepts HTTP PUT requests
- [ ] Writer API correctly encodes values to base64
- [ ] Writer API forwards requests to UnisonDB
- [ ] gRPC Client 1 receives WAL stream
- [ ] gRPC Client 2 receives WAL stream
- [ ] Both clients show the same offsets
- [ ] Both clients log incoming data
- [ ] Data integrity maintained (write vs read)
- [ ] Resilience: clients can reconnect and catch up
- [ ] Performance: < 100ms per write operation
- [ ] Error handling: invalid data rejected gracefully

## Troubleshooting

### Writer API can't connect to UnisonDB

**Symptom:** `Failed to write to UnisonDB: connection refused`

**Solution:** Verify UnisonDB is running on port 8001:
```bash
curl http://localhost:8001/api/v1/demo/offset
```

### Clients can't connect via gRPC

**Symptom:** `Failed to connect to upstream: connection refused`

**Solution:** Verify gRPC port is listening:
```bash
netstat -an | grep 4001
```

### No logs in clients after writing

**Symptom:** Clients show no activity after writes

**Solution:**
1. Check if clients are connected (should show "Starting replication stream")
2. Verify namespace matches ("demo")
3. Check UnisonDB logs for gRPC connections

### Base64 encoding errors

**Symptom:** `invalid base64 value` errors

**Solution:** Verify Writer API is encoding correctly:
```bash
echo "test" | base64
# Should produce: dGVzdAo=
```

## Expected Architecture Behavior

```
┌─────────────┐
│ Writer API  │  POST {"key":"user:1", "value":"Alice"}
└──────┬──────┘
       │ Encodes to base64: "QWxpY2U="
       ▼
┌──────────────────┐
│  UnisonDB HTTP   │  PUT {"value":"QWxpY2U="}
│  Port 8001       │
└────────┬─────────┘
         │ Writes to WAL
         │ Stores in B+Tree
         ▼
┌──────────────────┐
│  UnisonDB gRPC   │  Streams WAL records
│  Port 4001       │
└────┬───────┬─────┘
     │       │
     │       │ gRPC Stream
     ▼       ▼
┌─────────┐ ┌─────────┐
│ Client1 │ │ Client2 │
│ :9001   │ │ :9002   │
└─────────┘ └─────────┘
    │           │
    ▼           ▼
  Logs        Logs
  "📦 Received WAL batch"
```

## Success Metrics

- ✅ **Latency:** Writer API → Clients < 100ms
- ✅ **Throughput:** 100+ writes/second
- ✅ **Reliability:** Both clients receive 100% of records
- ✅ **Consistency:** Both clients show identical offsets
- ✅ **Resilience:** Clients reconnect and catch up after disconnect

---

Once all tests pass, you have a fully working gRPC replication demo! 🎉
