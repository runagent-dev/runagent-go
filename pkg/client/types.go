package client

// ClientOptions holds optional configuration for RunAgentClient
type ClientOptions struct {
	Local       bool
	Host        string
	Port        int
	APIKey      string
	BaseURL     string
	ExtraParams map[string]interface{}
}

// StreamRequest represents a stream execution request
type StreamRequest struct {
	EntrypointTag  string                 `json:"entrypoint_tag"`
	InputArgs      []interface{}          `json:"input_args"`
	InputKwargs    map[string]interface{} `json:"input_kwargs"`
	TimeoutSeconds int                    `json:"timeout_seconds,omitempty"`
	AsyncExecution bool                   `json:"async_execution,omitempty"`
}

// StreamMessage represents a simplified stream message from the server
type StreamMessage struct {
	Type                   string      `json:"type"`
	Status                 string      `json:"status,omitempty"`
	Content                interface{} `json:"content,omitempty"`
	Error                  string      `json:"error,omitempty"`
	InvocationID           string      `json:"invocation_id,omitempty"`
	MiddlewareInvocationID string      `json:"middleware_invocation_id,omitempty"`
	TotalChunks            int         `json:"total_chunks,omitempty"`
	ExecutionTime          float64     `json:"execution_time,omitempty"`
}
