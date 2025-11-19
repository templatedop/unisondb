#!/bin/bash

# Test script to write data to UnisonDB via Writer API

WRITER_API="http://localhost:8080"

echo "🧪 Testing UnisonDB gRPC Replication Demo"
echo "=========================================="
echo ""

# Check if Writer API is running
echo "🔍 Checking Writer API health..."
if ! curl -s "${WRITER_API}/health" > /dev/null 2>&1; then
    echo "❌ Writer API is not running on ${WRITER_API}"
    echo "   Start it with: cd writer-api && go run main.go"
    exit 1
fi
echo "✅ Writer API is healthy"
echo ""

# Write test data
echo "📝 Writing test data to UnisonDB..."
echo ""

# Test 1: Write a user
echo "Test 1: Writing user:1001"
curl -s -X POST "${WRITER_API}/write" \
  -H "Content-Type: application/json" \
  -d '{
    "key": "user:1001",
    "value": "Alice Johnson"
  }' | jq '.'
echo ""

sleep 1

# Test 2: Write another user
echo "Test 2: Writing user:1002"
curl -s -X POST "${WRITER_API}/write" \
  -H "Content-Type: application/json" \
  -d '{
    "key": "user:1002",
    "value": "Bob Smith"
  }' | jq '.'
echo ""

sleep 1

# Test 3: Write a product
echo "Test 3: Writing product:5001"
curl -s -X POST "${WRITER_API}/write" \
  -H "Content-Type: application/json" \
  -d '{
    "key": "product:5001",
    "value": "Laptop Pro 15 - 16GB RAM"
  }' | jq '.'
echo ""

sleep 1

# Test 4: Write an order
echo "Test 4: Writing order:9001"
curl -s -X POST "${WRITER_API}/write" \
  -H "Content-Type: application/json" \
  -d '{
    "key": "order:9001",
    "value": "Order placed by Alice for Laptop"
  }' | jq '.'
echo ""

sleep 1

# Test 5: Write a counter
echo "Test 5: Writing counter:visits"
curl -s -X POST "${WRITER_API}/write" \
  -H "Content-Type: application/json" \
  -d '{
    "key": "counter:visits",
    "value": "12345"
  }' | jq '.'
echo ""

echo "=========================================="
echo "✅ Test completed!"
echo ""
echo "🔍 Check the logs of gRPC Client 1 and Client 2"
echo "   They should both show the replicated data!"
echo ""
