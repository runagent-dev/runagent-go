package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/runagent-dev/runagent-go/pkg/constants"
	"github.com/runagent-dev/runagent-go/pkg/db"
	"github.com/runagent-dev/runagent-go/pkg/types"
)

// RunAgentClient represents a RunAgent client following the SDK contract
type RunAgentClient struct {
	agentID       string
	entrypointTag string
	local         bool
	apiKey        string
	baseURL       string
	socketURL     string
	httpClient    *http.Client
	dbService     *db.Service
	serializer    *CoreSerializer
	extraParams   map[string]interface{}
}

// NewClient creates a new RunAgent client with options
// Constructor signature follows: agent_id, entrypoint_tag, optional ClientOptions
func NewClient(agentID, entrypointTag string, opts *ClientOptions) (*RunAgentClient, error) {
	if opts == nil {
		opts = &ClientOptions{}
	}

	client := &RunAgentClient{
		agentID:       agentID,
		entrypointTag: entrypointTag,
		local:         opts.Local,
		serializer:    NewCoreSerializer(),
		extraParams:   opts.ExtraParams,
		httpClient: &http.Client{
			Timeout: 5 * time.Minute,
		},
	}

	// Configuration precedence: constructor args > env vars > defaults
	// 1. Set API key (constructor > env > default)
	if opts.APIKey != "" {
		client.apiKey = opts.APIKey
	} else if envAPIKey := os.Getenv(constants.EnvAPIKey); envAPIKey != "" {
		client.apiKey = envAPIKey
	}

	// 2. Set base URL (constructor > env > default)
	var baseURL string
	if opts.BaseURL != "" {
		baseURL = opts.BaseURL
	} else if envBaseURL := os.Getenv(constants.EnvBaseURL); envBaseURL != "" {
		baseURL = envBaseURL
	} else {
		baseURL = "https://backend.run-agent.ai"
	}

	// Ensure proper URL format
	if !strings.HasPrefix(baseURL, "http://") && !strings.HasPrefix(baseURL, "https://") {
		baseURL = "https://" + baseURL
	}

	// 3. Resolve host/port and construct URLs
	if opts.Host != "" && opts.Port != 0 {
		// Explicit host/port provided
		client.baseURL = fmt.Sprintf("http://%s:%d", opts.Host, opts.Port)
		client.socketURL = fmt.Sprintf("ws://%s:%d", opts.Host, opts.Port)
	} else if opts.Local {
		// Local mode: try DB discovery
		dbService, err := db.NewService("")
		if err != nil {
			return nil, types.NewConnectionError(fmt.Sprintf("failed to initialize local database: %v", err))
		}

		agent, err := dbService.GetAgent(agentID)
		if err != nil {
			dbService.Close()
			return nil, types.NewConnectionError(fmt.Sprintf("failed to get agent from database: %v", err))
		}

		if agent == nil {
			dbService.Close()
			return nil, types.NewValidationError(fmt.Sprintf("Agent %s not found in local database. Start or register the agent locally first.", agentID))
		}

		client.baseURL = fmt.Sprintf("http://%s:%d", agent.Host, agent.Port)
		client.socketURL = fmt.Sprintf("ws://%s:%d", agent.Host, agent.Port)
		client.dbService = dbService
	} else {
		// Remote mode: use base URL
		client.baseURL = baseURL
		// Construct WebSocket URL from base URL
		if strings.HasPrefix(baseURL, "https://") {
			client.socketURL = strings.Replace(baseURL, "https://", "wss://", 1)
		} else {
			client.socketURL = strings.Replace(baseURL, "http://", "ws://", 1)
		}

		// Ensure /api/v1 suffix for remote calls
		if !strings.HasSuffix(client.baseURL, "/api/v1") {
			client.baseURL = strings.TrimSuffix(client.baseURL, "/") + "/api/v1"
			client.socketURL = strings.TrimSuffix(client.socketURL, "/") + "/api/v1"
		}
	}

	return client, nil
}

// New creates a new RunAgent client (legacy compatibility)
func New(agentID, entrypointTag string, local bool) (*RunAgentClient, error) {
	return NewClient(agentID, entrypointTag, &ClientOptions{Local: local})
}

// NewWithAddress creates a client with explicit address (legacy compatibility)
func NewWithAddress(agentID, entrypointTag string, local bool, host string, port int) (*RunAgentClient, error) {
	return NewClient(agentID, entrypointTag, &ClientOptions{
		Local: local,
		Host:  host,
		Port:  port,
	})
}

// FromEnv creates a client from environment variables
func FromEnv(agentID, entrypointTag string) (*RunAgentClient, error) {
	opts := &ClientOptions{
		Local:   strings.ToLower(os.Getenv("RUNAGENT_LOCAL")) == "true",
		APIKey:  os.Getenv(constants.EnvAPIKey),
		BaseURL: os.Getenv(constants.EnvBaseURL),
	}
	return NewClient(agentID, entrypointTag, opts)
}

// GetExtraParams returns the extra parameters
func (c *RunAgentClient) GetExtraParams() map[string]interface{} {
	return c.extraParams
}

// Close closes the client and any associated resources
func (c *RunAgentClient) Close() error {
	if c.dbService != nil {
		return c.dbService.Close()
	}
	return nil
}

// Run executes the agent with the given input, returning the result
func (c *RunAgentClient) Run(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	// Build request matching Python SDK format
	request := map[string]interface{}{
		"entrypoint_tag":  c.entrypointTag,
		"input_args":      []interface{}{},
		"input_kwargs":    input,
		"timeout_seconds": 300,
	}

	return c.runInternal(ctx, request)
}

// RunWithArgs executes the agent with positional and keyword arguments
func (c *RunAgentClient) RunWithArgs(ctx context.Context, args []interface{}, kwargs map[string]interface{}) (interface{}, error) {
	request := map[string]interface{}{
		"entrypoint_tag":  c.entrypointTag,
		"input_args":      args,
		"input_kwargs":    kwargs,
		"timeout_seconds": 300,
	}

	return c.runInternal(ctx, request)
}

// runInternal handles the common HTTP request logic for run operations
func (c *RunAgentClient) runInternal(ctx context.Context, request map[string]interface{}) (interface{}, error) {
	requestBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/agents/%s/run", c.baseURL, c.agentID)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	c.addAuthHeader(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, types.NewConnectionError(fmt.Sprintf("Failed to execute request: %v", err))
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Handle HTTP error status codes
	if resp.StatusCode == http.StatusUnauthorized {
		return nil, types.NewAuthenticationError("Invalid or missing API key. Set RUNAGENT_API_KEY environment variable or pass api_key parameter.")
	}
	if resp.StatusCode == http.StatusForbidden {
		return nil, types.NewPermissionError("You do not have permission to access this agent")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, types.NewServerError(fmt.Sprintf("Server returned status %d: %s", resp.StatusCode, string(body)))
	}

	if len(body) == 0 {
		return nil, types.NewServerError("Server returned empty response")
	}

	return c.deserializeResponse(body)
}

// deserializeResponse handles response deserialization according to Python SDK format
func (c *RunAgentClient) deserializeResponse(body []byte) (interface{}, error) {
	var response map[string]interface{}
	if err := json.Unmarshal(body, &response); err != nil {
		return string(body), nil
	}

	// Check for success field
	if success, exists := response["success"]; exists {
		if successBool, ok := success.(bool); ok && !successBool {
			// Extract error info
			errorInfo := extractErrorInfo(response)
			return nil, &types.RunAgentError{
				Type:    "execution",
				Message: errorInfo,
				Code:    "SERVER_ERROR",
			}
		}
	}

	// Try result_data.data first (legacy structured output)
	if resultData, exists := response["result_data"]; exists {
		if resultMap, ok := resultData.(map[string]interface{}); ok {
			if data, exists := resultMap["data"]; exists {
				return data, nil
			}
		}
	}

	// Return output_data if it exists
	if outputData, exists := response["output_data"]; exists {
		return outputData, nil
	}

	// Fall back to data field
	if data, exists := response["data"]; exists {
		return data, nil
	}

	return response, nil
}

// extractErrorInfo extracts error information from response
func extractErrorInfo(response map[string]interface{}) string {
	if errMsg, exists := response["error"]; exists {
		return fmt.Sprintf("%v", errMsg)
	}
	if errMsg, exists := response["message"]; exists {
		return fmt.Sprintf("%v", errMsg)
	}
	return "Unknown error"
}

// addAuthHeader adds authorization header to request
func (c *RunAgentClient) addAuthHeader(req *http.Request) {
	if c.apiKey != "" && !c.local {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
	}
}

// RunStream executes the agent with streaming response using WebSocket
func (c *RunAgentClient) RunStream(ctx context.Context, input map[string]interface{}) (*StreamIterator, error) {
	// Construct WebSocket URL with query string token for authentication
	wsURL := fmt.Sprintf("%s/agents/%s/run-stream", c.socketURL, c.agentID)

	// Add API key as query parameter if available (fallback for WebSocket)
	if c.apiKey != "" && !c.local {
		wsURL = fmt.Sprintf("%s?token=%s", wsURL, c.apiKey)
	}

	dialer := websocket.Dialer{
		HandshakeTimeout: 30 * time.Second,
	}

	headers := http.Header{
		"User-Agent": []string{"RunAgent-Go/1.0"},
	}

	// Add Authorization header if not local
	if c.apiKey != "" && !c.local {
		headers.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
	}

	conn, _, err := dialer.DialContext(ctx, wsURL, headers)
	if err != nil {
		if strings.Contains(err.Error(), "401") || strings.Contains(err.Error(), "403") {
			return nil, types.NewAuthenticationError("Failed to authenticate WebSocket connection. Check your API key.")
		}
		return nil, types.NewConnectionError(fmt.Sprintf("Failed to connect to WebSocket: %v", err))
	}

	// Send stream request with timeout_seconds
	request := StreamRequest{
		EntrypointTag:  c.entrypointTag,
		InputArgs:      []interface{}{},
		InputKwargs:    input,
		TimeoutSeconds: 600,
		AsyncExecution: false,
	}

	requestData, err := json.Marshal(request)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	if err := conn.WriteMessage(websocket.TextMessage, requestData); err != nil {
		conn.Close()
		return nil, types.NewConnectionError(fmt.Sprintf("failed to send stream request: %v", err))
	}

	return NewStreamIterator(conn, c.serializer), nil
}

// HealthCheck checks if the agent is healthy
func (c *RunAgentClient) HealthCheck(ctx context.Context) (bool, error) {
	url := fmt.Sprintf("%s/health", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false, types.NewConnectionError(fmt.Sprintf("Health check failed: %v", err))
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK, nil
}

// GetAgentArchitecture retrieves the agent's architecture information
func (c *RunAgentClient) GetAgentArchitecture(ctx context.Context) (*types.AgentArchitecture, error) {
	url := fmt.Sprintf("%s/agents/%s/architecture", c.baseURL, c.agentID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.addAuthHeader(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, types.NewConnectionError(fmt.Sprintf("Failed to get architecture: %v", err))
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, types.NewServerError(fmt.Sprintf("Server returned status %d", resp.StatusCode))
	}

	var architecture types.AgentArchitecture
	if err := json.NewDecoder(resp.Body).Decode(&architecture); err != nil {
		return nil, fmt.Errorf("failed to decode architecture: %w", err)
	}

	return &architecture, nil
}

// AgentID returns the agent ID
func (c *RunAgentClient) AgentID() string {
	return c.agentID
}

// EntrypointTag returns the entrypoint tag
func (c *RunAgentClient) EntrypointTag() string {
	return c.entrypointTag
}

// IsLocal returns whether this is a local client
func (c *RunAgentClient) IsLocal() bool {
	return c.local
}

// GetAgentLimits retrieves the current agent limits from the API
func (c *RunAgentClient) GetAgentLimits(ctx context.Context) (map[string]interface{}, error) {
	url := fmt.Sprintf("%s/limits/agents", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.addAuthHeader(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, types.NewConnectionError(fmt.Sprintf("Failed to get limits: %v", err))
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, types.NewServerError(fmt.Sprintf("Server returned status %d", resp.StatusCode))
	}

	var limits map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&limits); err != nil {
		return nil, fmt.Errorf("failed to decode limits: %w", err)
	}

	return limits, nil
}

// UploadMetadata uploads sensitive metadata to the server
func (c *RunAgentClient) UploadMetadata(ctx context.Context, metadata map[string]interface{}) (map[string]interface{}, error) {
	url := fmt.Sprintf("%s/agents/metadata-upload", c.baseURL)

	requestBody, err := json.Marshal(metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metadata: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	c.addAuthHeader(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, types.NewConnectionError(fmt.Sprintf("Failed to upload metadata: %v", err))
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, types.NewServerError(fmt.Sprintf("Server returned status %d", resp.StatusCode))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result, nil
}

// StartAgent starts/deploys an agent on the server
func (c *RunAgentClient) StartAgent(ctx context.Context, cfg map[string]interface{}) (map[string]interface{}, error) {
	url := fmt.Sprintf("%s/agents/%s/start", c.baseURL, c.agentID)

	requestBody, err := json.Marshal(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	c.addAuthHeader(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, types.NewConnectionError(fmt.Sprintf("Failed to start agent: %v", err))
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, types.NewServerError(fmt.Sprintf("Server returned status %d", resp.StatusCode))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result, nil
}

// GetAgentStatus retrieves the status of an agent
func (c *RunAgentClient) GetAgentStatus(ctx context.Context) (map[string]interface{}, error) {
	url := fmt.Sprintf("%s/agents/%s/status", c.baseURL, c.agentID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.addAuthHeader(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, types.NewConnectionError(fmt.Sprintf("Failed to get status: %v", err))
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, types.NewServerError(fmt.Sprintf("Server returned status %d", resp.StatusCode))
	}

	var status map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return nil, fmt.Errorf("failed to decode status: %w", err)
	}

	return status, nil
}
