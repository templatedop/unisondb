# Base64 Encoding Flow Verification

## ✅ Confirmation: Value IS base64 encoded when posting to UnisonDB

This document clarifies how the Writer API handles base64 encoding.

---

## Data Flow

```
┌─────────────────────────────────────────────────────────────────┐
│ STEP 1: User sends PLAIN TEXT to Writer API                    │
└─────────────────────────────────────────────────────────────────┘

curl -X POST http://localhost:8080/write \
  -H "Content-Type: application/json" \
  -d '{
    "key": "user:1",
    "value": "Alice"           👈 PLAIN TEXT
  }'

┌─────────────────────────────────────────────────────────────────┐
│ STEP 2: Writer API receives plain text                         │
└─────────────────────────────────────────────────────────────────┘

// writer-api/main.go:37-38
var req WriteRequest
json.NewDecoder(r.Body).Decode(&req)

// req.Value = "Alice" (plain text)

┌─────────────────────────────────────────────────────────────────┐
│ STEP 3: Writer API ENCODES to base64                           │
└─────────────────────────────────────────────────────────────────┘

// writer-api/main.go:67 ⭐ KEY LINE!
encodedValue := base64.StdEncoding.EncodeToString([]byte(value))

// Input:  "Alice"
// Output: "QWxpY2U="   👈 BASE64 ENCODED!

┌─────────────────────────────────────────────────────────────────┐
│ STEP 4: Writer API creates JSON with BASE64 value              │
└─────────────────────────────────────────────────────────────────┘

// writer-api/main.go:69-71
putReq := UnisonDBPutRequest{
    Value: encodedValue,  // "QWxpY2U=" (base64)
}

// writer-api/main.go:73
body, _ := json.Marshal(putReq)

// Result JSON:
{
  "value": "QWxpY2U="    👈 BASE64 STRING
}

┌─────────────────────────────────────────────────────────────────┐
│ STEP 5: Writer API sends BASE64 to UnisonDB                    │
└─────────────────────────────────────────────────────────────────┘

PUT http://localhost:8001/api/v1/demo/kv/user:1
Content-Type: application/json

{
  "value": "QWxpY2U="    👈 BASE64 ENCODED! ✅
}

┌─────────────────────────────────────────────────────────────────┐
│ STEP 6: UnisonDB receives BASE64 value                         │
└─────────────────────────────────────────────────────────────────┘

// UnisonDB's httpapi/service.go:174-178
value, err := base64.StdEncoding.DecodeString(req.Value)
// Decodes "QWxpY2U=" back to "Alice"
// Stores "Alice" in database
```

---

## Code Evidence

### Writer API Code (writer-api/main.go)

```go
// Line 65-71
func writeToUnisonDB(key, value string) error {
    // ⭐ THIS LINE ENCODES TO BASE64
    encodedValue := base64.StdEncoding.EncodeToString([]byte(value))

    putReq := UnisonDBPutRequest{
        Value: encodedValue,  // ✅ base64 encoded
    }

    body, err := json.Marshal(putReq)
    // ... sends to UnisonDB
}
```

### UnisonDB API Code (internal/services/httpapi/service.go)

```go
// Line 174-178
var req PutKVRequest
json.NewDecoder(r.Body).Decode(&req)

// ⭐ UnisonDB EXPECTS base64 and DECODES it
value, err := base64.StdEncoding.DecodeString(req.Value)

// Stores decoded value
engine.PutKV([]byte(key), value)
```

---

## Verification Test

```bash
# What the user sends
INPUT="Alice"

# What Writer API sends to UnisonDB
ENCODED=$(echo -n "Alice" | base64)
echo "$ENCODED"
# Output: QWxpY2U=

# The JSON sent to UnisonDB
echo "{\"value\":\"$ENCODED\"}"
# Output: {"value":"QWxpY2U="}
```

**Result:** ✅ Value IS base64 encoded when posting to UnisonDB

---

## Why This Design?

### User Experience
- Users send **plain text** (easier to use)
- Writer API **automatically encodes** to base64
- Users don't need to know about base64

### UnisonDB Requirement
- UnisonDB API requires **base64-encoded values**
- UnisonDB **decodes** the base64 internally
- Allows binary data storage

### Example Flow

**Plain Text Input:**
```json
{
  "key": "user:1",
  "value": "Hello World"
}
```

**Sent to UnisonDB:**
```json
{
  "value": "SGVsbG8gV29ybGQ="
}
```

**Stored in UnisonDB:**
```
Raw bytes: "Hello World"
```

---

## Comparison Table

| Step | Component | Format | Example |
|------|-----------|--------|---------|
| 1 | User Input | Plain text | `"Alice"` |
| 2 | Writer API receives | Plain text | `"Alice"` |
| 3 | Writer API encodes | **Base64** | `"QWxpY2U="` |
| 4 | Writer API sends | **Base64** in JSON | `{"value":"QWxpY2U="}` |
| 5 | UnisonDB receives | **Base64** | `"QWxpY2U="` |
| 6 | UnisonDB decodes | Plain text | `"Alice"` |
| 7 | UnisonDB stores | Raw bytes | `"Alice"` |

---

## Common Confusion

### ❌ WRONG Understanding:
"User must send base64 to Writer API"

```bash
# ❌ This is NOT required
curl -X POST http://localhost:8080/write \
  -d '{"key": "user:1", "value": "QWxpY2U="}'  # Don't do this!
```

### ✅ CORRECT Understanding:
"User sends plain text, Writer API encodes to base64"

```bash
# ✅ This is the correct way
curl -X POST http://localhost:8080/write \
  -d '{"key": "user:1", "value": "Alice"}'  # Plain text
```

Writer API **automatically** converts `"Alice"` → `"QWxpY2U="` before sending to UnisonDB.

---

## Test Proof

Run this to verify:

```bash
cd cmd/examples/grpc-replication-demo
./validate-logic.sh
```

**Output:**
```
Test 1: Verify Base64 Encoding
--------------------------------
✅ Base64 encoding correct
   Input: Alice Johnson
   Output: QWxpY2UgSm9obnNvbg==
```

---

## Summary

### Question:
"Is the value base64 encoded when posting to UnisonDB?"

### Answer:
**YES! ✅**

The Writer API **automatically encodes** the plain text value to base64 (line 67) before sending to UnisonDB.

### Code Location:
**File:** `cmd/examples/grpc-replication-demo/writer-api/main.go`
**Line:** 67
```go
encodedValue := base64.StdEncoding.EncodeToString([]byte(value))
```

### UnisonDB API Specification:
**File:** `internal/services/httpapi/service.go`
**Line:** 174
```go
value, err := base64.StdEncoding.DecodeString(req.Value)
```

**Both sides match perfectly! ✅**

---

## Conclusion

The implementation is **100% correct**:

1. ✅ User sends plain text to Writer API
2. ✅ Writer API encodes to base64
3. ✅ Writer API sends base64 to UnisonDB
4. ✅ UnisonDB decodes base64
5. ✅ UnisonDB stores the data

**The value IS base64 encoded when posting to UnisonDB!** 🎉
