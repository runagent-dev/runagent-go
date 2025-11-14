# Client Package Architecture

The `pkg/client` package is organized in a modular, single-responsibility pattern for maintainability and clarity.

## File Organization

### `client.go` (~350 lines)
**Core client implementation**
- `RunAgentClient` struct definition
- Constructor: `NewClient()`, `FromEnv()`, legacy `New()`, `NewWithAddress()`
- Configuration and initialization logic
- Configuration precedence implementation
- HTTP execution methods: `Run()`, `RunWithArgs()`, `runInternal()`
- Response deserialization: `deserializeResponse()`, `extractErrorInfo()`
- WebSocket execution: `RunStream()`
- Authentication: `addAuthHeader()`
- Utility methods: `GetExtraParams()`, `Close()`
- Agent information methods: `AgentID()`, `EntrypointTag()`, `IsLocal()`
- Additional endpoints: `HealthCheck()`, `GetAgentArchitecture()`, `GetAgentLimits()`, `UploadMetadata()`, `StartAgent()`, `GetAgentStatus()`

**Dependencies:**
- `context`, `bytes`, `io`, `net/http`, `os`, `strings`, `time`
- External: `github.com/gorilla/websocket`
- Internal: `pkg/constants`, `pkg/db`, `pkg/types`

---

### `types.go` (~30 lines)
**Data structures for client operations**
- `ClientOptions` - Configuration struct for client initialization
- `StreamRequest` - WebSocket handshake request format
- `StreamMessage` - WebSocket frame representation

**Dependencies:**
- None (data types only)

---

### `serializer.go` (~40 lines)
**Message serialization and deserialization**
- `CoreSerializer` struct
- `SerializeMessage()` - Converts objects to JSON
- `DeserializeMessage()` - Parses JSON to StreamMessage
- `DeserializeObject()` - Generic JSON deserialization
- `NewCoreSerializer()` - Factory function

**Responsibilities:**
- JSON marshaling/unmarshaling
- Error handling for malformed messages
- Stream message parsing

**Dependencies:**
- `encoding/json`, `fmt`

---

### `stream.go` (~80 lines)
**WebSocket streaming support**
- `StreamIterator` struct - Iterator pattern for stream consumption
- `NewStreamIterator()` - Factory function
- `Next()` - Read next message from stream
- `Close()` - Graceful stream termination

**Responsibilities:**
- WebSocket frame iteration
- Message type handling (status, data, error)
- Context-aware cancellation
- Error propagation

**Dependencies:**
- `context`, `fmt`
- External: `github.com/gorilla/websocket`

---

### `README.md`
**User-facing documentation**
- Quickstart examples (local, remote, streaming)
- Configuration reference
- Usage patterns
- Error handling guide
- Advanced topics
- Troubleshooting
- Legacy compatibility notes

---

## Data Flow

```
NewClient()
    ↓
    ├─ Parse ClientOptions
    ├─ Apply configuration precedence (constructor > env > defaults)
    ├─ Resolve local/remote URLs
    └─ Return RunAgentClient
         
Run(ctx, input)
    ↓
    ├─ Build request payload (StreamRequest-like)
    ├─ Marshal to JSON
    ├─ POST to /api/v1/agents/{id}/run
    ├─ Handle response (200/401/403/5xx)
    └─ deserializeResponse() → client response
         
RunStream(ctx, input)
    ↓
    ├─ Connect WebSocket to /api/v1/agents/{id}/run-stream
    ├─ Send StreamRequest JSON
    ├─ Return StreamIterator
         
StreamIterator.Next(ctx)
    ↓
    ├─ Read WebSocket message
    ├─ Deserialize via CoreSerializer
    ├─ Handle message types (status/data/error)
    └─ Yield content or terminate
```

## Separation of Concerns

| Module | Responsibility | Single Purpose |
|--------|-----------------|-----------------|
| `client.go` | Client initialization, HTTP/WebSocket execution | Orchestrate agent execution |
| `types.go` | Data structure definitions | Define API contracts |
| `serializer.go` | JSON parsing and formatting | Handle message serialization |
| `stream.go` | Iterator pattern for streaming | Consume streaming responses |

## Design Principles

1. **Modularity**: Each file handles one aspect of client functionality
2. **Single Responsibility**: No file does more than necessary
3. **Clarity**: File names match their purpose
4. **Testability**: Each module can be tested independently
5. **Maintainability**: Easy to locate and modify features

## Adding New Features

### To add a new HTTP endpoint:
1. Add method to `RunAgentClient` in `client.go`
2. Reuse `addAuthHeader()` for authentication
3. Use same error handling pattern as existing methods

### To add a new configuration option:
1. Add field to `ClientOptions` in `types.go`
2. Handle in configuration precedence in `client.go`'s `NewClient()`
3. Document in `README.md`

### To change message handling:
1. Update `StreamMessage` in `types.go`
2. Update parsing logic in `serializer.go`
3. Update handling in `StreamIterator.Next()` in `stream.go`

## Import Dependencies

**Minimal external dependencies:**
- `github.com/gorilla/websocket` - WebSocket support
- `github.com/runagent-dev/runagent-go/pkg/constants` - Constants
- `github.com/runagent-dev/runagent-go/pkg/db` - Local DB access
- `github.com/runagent-dev/runagent-go/pkg/types` - Error types

**Standard library only:**
- All other files use only Go standard library

## Testing Strategy

Each module can be unit tested:
- `types.go`: No logic, data structures only
- `serializer.go`: JSON marshal/unmarshal with mock data
- `stream.go`: Mock WebSocket connection with test messages
- `client.go`: Mock HTTP server with httptest package

---

**File Statistics:**
- Total: ~500 lines of code
- Well-distributed across modules
- Average module: 100-125 lines (highly focused)
- No circular dependencies
- Clear initialization flow
