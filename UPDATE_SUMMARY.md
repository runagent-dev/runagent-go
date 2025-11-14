# Go SDK Major Update - Endpoint Alignment with Python SDK

## ✅ Changes Completed

### 1. **Endpoint Changes**
   - **Old**: `/api/v1/agents/{agent_id}/execute/{entrypoint_tag}`
   - **New**: `/agents/{agent_id}/run-stream?token={api_key}`
   - Token authentication moved from headers to query parameter

### 2. **Request Format Changes**

#### Old Format (Removed)
```go
type ExecutionRequest struct {
    Action    string
    AgentID   string
    InputData map[string]interface{}
}

type WebSocketMessage struct {
    ID        string
    Type      string
    Timestamp string
    Data      interface{}
    Metadata  map[string]interface{}
    Error     string
}
```

#### New Format (Added)
```go
type StreamRequest struct {
    EntrypointTag  string
    InputArgs      []interface{}
    InputKwargs    map[string]interface{}
    TimeoutSeconds int
    AsyncExecution bool
}

type StreamMessage struct {
    Type                   string
    Status                 string      // "stream_started", "stream_completed"
    Content                interface{} // Actual data chunks
    Error                  string
    InvocationID           string
    MiddlewareInvocationID string
    TotalChunks            int
    ExecutionTime          float64
}
```

### 3. **Response Message Types**

| Type | Usage | Returns |
|------|-------|---------|
| `"status"` | Stream lifecycle management | nil when stream_completed |
| `"data"` | Actual data chunks | content field value |
| `"error"` | Error handling | error with message |

### 4. **Code Updates**

#### RunStream Method
- ✅ Updated endpoint URL to `/agents/{agent_id}/run-stream?token={api_key}`
- ✅ Removed WebSocketMessage wrapper
- ✅ Send StreamRequest directly as JSON
- ✅ Load API key from config for authentication

#### Stream Iterator Next() Method
- ✅ Updated message type handling (status/data/error)
- ✅ Extract content from `msg.Content` instead of `msg.Data`
- ✅ Use `msg.Status` for status checks instead of nested data

#### Serializer Methods
- ✅ Simplified SerializeMessage() for direct JSON
- ✅ Updated DeserializeMessage() to return StreamMessage

### 5. **Build Status**
```
✓ Build successful
No compilation errors
```

## Breaking Changes

| Aspect | Impact | Action |
|--------|--------|--------|
| WebSocket URL | Format completely changed | Update connection logic |
| Request format | No longer wrapped in envelope | Use StreamRequest directly |
| Response format | Message structure simplified | Update message parsing |
| Message types | New type-based handling | Switch on "status"/"data"/"error" |

## Backward Compatibility

❌ **NOT compatible** with old server (requires updated server)
❌ **Old clients** will NOT work with new server

## Files Modified
- `pkg/client/client.go`

## Files Updated
- `CHANGES_SUMMARY.txt` - Detailed changelog

## Testing Recommended
- [ ] Test WebSocket connection with new endpoint
- [ ] Verify StreamRequest JSON format
- [ ] Test stream message handling (status/data/error)
- [ ] Verify token authentication via query parameter
- [ ] Test error handling and stream completion

## Integration Notes
The Go SDK now fully aligns with the Python SDK implementation:
- Same endpoint structure
- Same request/response format
- Same message type handling
- Same authentication mechanism
