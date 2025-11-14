# RunAgent Go SDK Client

The `RunAgentClient` is the main entry point for interacting with RunAgent. It provides both HTTP (`Run`) and WebSocket (`RunStream`) capabilities for executing agents.

## Installation

Add the runagent-go module to your project:

```bash
go get github.com/runagent-dev/runagent-go
```

## Quickstart

### Local Agent Execution

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/runagent-dev/runagent-go/pkg/client"
)

func main() {
	// Create a client for a locally running agent
	c, err := client.NewClient("my-agent-id", "my-entrypoint", &client.ClientOptions{
		Local: true,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()

	// Execute the agent
	input := map[string]interface{}{
		"prompt": "Hello, agent!",
	}
	result, err := c.Run(context.Background(), input)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(result)
}
```

### Remote Agent Execution

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/runagent-dev/runagent-go/pkg/client"
)

func main() {
	// Set RUNAGENT_API_KEY environment variable before running
	c, err := client.FromEnv("my-agent-id", "my-entrypoint")
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()

	result, err := c.Run(context.Background(), map[string]interface{}{
		"question": "What is 2+2?",
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(result)
}
```

### Streaming Responses

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/runagent-dev/runagent-go/pkg/client"
)

func main() {
	c, err := client.NewClient("my-agent-id", "my-stream-entrypoint", &client.ClientOptions{
		Local: true,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()

	// Stream results from the agent
	iterator, err := c.RunStream(context.Background(), map[string]interface{}{
		"prompt": "Generate a long response...",
	})
	if err != nil {
		log.Fatal(err)
	}

	for {
		chunk, ok, err := iterator.Next(context.Background())
		if err != nil {
			log.Fatal(err)
		}
		if !ok {
			break
		}
		fmt.Println(chunk)
	}
}
```

## Configuration

The `RunAgentClient` follows a strict configuration precedence:

1. **Constructor arguments** (highest priority)
2. **Environment variables**
3. **Library defaults** (lowest priority)

### Configuration Options

The `ClientOptions` struct supports:

- `Local` (bool): Whether to use local agent discovery. Defaults to `false` for remote, `true` for Python CLI.
- `Host` (string): Explicit host address for local agents.
- `Port` (int): Explicit port for local agents.
- `APIKey` (string): API key for remote authentication (overrides `RUNAGENT_API_KEY`).
- `BaseURL` (string): Base URL for remote backend (overrides `RUNAGENT_BASE_URL`).
- `ExtraParams` (map): Metadata for future use, stored but not acted upon.

### Environment Variables

- `RUNAGENT_API_KEY`: API key for remote backend authentication.
- `RUNAGENT_BASE_URL`: Base URL for remote backend (default: `https://backend.run-agent.ai`).
- `RUNAGENT_LOCAL`: Set to `true` to use local agent discovery.

## Usage Patterns

### Pattern 1: Explicit Configuration

```go
c, err := client.NewClient("agent-id", "entrypoint-tag", &client.ClientOptions{
	Host:   "localhost",
	Port:   8450,
	APIKey: "my-secret-key",
	Local:  false,
})
```

### Pattern 2: Environment-Based

```go
// Reads RUNAGENT_API_KEY, RUNAGENT_BASE_URL, RUNAGENT_LOCAL
c, err := client.FromEnv("agent-id", "entrypoint-tag")
```

### Pattern 3: Local with DB Discovery

```go
// Automatically discovers agent from ~/.runagent/runagent_local.db
c, err := client.NewClient("agent-id", "entrypoint-tag", &client.ClientOptions{
	Local: true,
})
```

## Error Handling

The SDK raises language-idiomatic errors with structured information:

```go
result, err := c.Run(ctx, input)
if err != nil {
	// Check error type
	if runAgentErr, ok := err.(*types.RunAgentError); ok {
		switch runAgentErr.Type {
		case "authentication":
			fmt.Println("Auth failed:", runAgentErr.Message)
		case "connection":
			fmt.Println("Connection failed:", runAgentErr.Message)
		case "validation":
			fmt.Println("Input validation failed:", runAgentErr.Message)
		case "server":
			fmt.Println("Server error:", runAgentErr.Message)
		}
	}
}
```

### Common Errors

| Error Type | Code | Meaning |
|------------|------|---------|
| `authentication` | `AUTHENTICATION_ERROR` | Invalid or missing API key |
| `permission` | `PERMISSION_ERROR` | Insufficient permissions for resource |
| `validation` | `VALIDATION_ERROR` | Invalid input or configuration |
| `connection` | `CONNECTION_ERROR` | Network connectivity issue |
| `server` | `SERVER_ERROR` | Server returned error status |
| `database` | `DATABASE_ERROR` | Local database access issue |
| `config` | `CONFIG_ERROR` | Configuration loading issue |

## Advanced Topics

### Custom Base URL

```go
c, err := client.NewClient("agent-id", "entrypoint", &client.ClientOptions{
	BaseURL: "https://custom-backend.example.com",
	APIKey:  "api-key",
})
```

### Explicit Host/Port

```go
c, err := client.NewClient("agent-id", "entrypoint", &client.ClientOptions{
	Host:  "192.168.1.100",
	Port:  8450,
	Local: true,
})
```

### Extra Params (Future Use)

```go
c, err := client.NewClient("agent-id", "entrypoint", &client.ClientOptions{
	ExtraParams: map[string]interface{}{
		"trace_id": "abc123",
		"user_id":  "user-42",
	},
})

// Retrieve them later
params := c.GetExtraParams()
```

## Troubleshooting

### "Agent not found in local database"

**Cause**: Running with `Local: true` but agent is not registered locally.

**Solution**: Start the agent locally first, or set `Local: false` and provide explicit host/port.

### "Invalid or missing API key"

**Cause**: Running remote agent without API key.

**Solution**: Set `RUNAGENT_API_KEY` environment variable or pass `APIKey` in `ClientOptions`.

### "Failed to connect to WebSocket"

**Cause**: Network connectivity issue or WebSocket not available.

**Solution**: Check network connectivity and verify server is running. For remote servers, ensure firewall allows WebSocket connections.

## Legacy Compatibility

The SDK supports legacy factory functions for backward compatibility:

```go
// Old API - still works
c, err := client.New("agent-id", "entrypoint", true)

c, err := client.NewWithAddress("agent-id", "entrypoint", true, "localhost", 8450)
```

However, new code should use `NewClient` with `ClientOptions` for clarity and flexibility.
