package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/aatumaykin/nexbot/internal/logger"
)

const (
	// ZAIEndpoint is the base URL for Z.ai Coding API
	ZAIEndpoint = "https://api.z.ai/api/coding/paas/v4/chat/completions"
	// ZAIRequestTimeout is the default timeout for API requests
	ZAIRequestTimeout = 60 * time.Second
	// ZAIMaxRetries is the maximum number of retry attempts
	ZAIMaxRetries = 3
	// ZAIRetryDelay is the delay between retry attempts
	ZAIRetryDelay = 1 * time.Second
)

// ZAIConfig contains configuration for the Z.ai provider.
type ZAIConfig struct {
	APIKey string `json:"api_key"` // API key for authentication
	Model  string `json:"model"`   // Default model to use (optional, defaults to glm-4.7)
}

// ZAIProvider implements the Provider interface for Z.ai Coding API.
type ZAIProvider struct {
	client *http.Client // HTTP client for API requests
	config ZAIConfig    // Provider configuration
	apiURL string       // API endpoint URL
	logger *logger.Logger
}

// zaiRequest represents the request format for Z.ai API.
type zaiRequest struct {
	Messages    []zaiMessage `json:"messages"`              // Conversation messages
	Model       string       `json:"model"`                 // Model identifier
	Temperature float64      `json:"temperature,omitempty"` // Sampling temperature
	MaxTokens   int          `json:"max_tokens,omitempty"`  // Maximum tokens to generate
	Tools       []zaiTool    `json:"tools,omitempty"`       // Available tools/functions
}

// zaiMessage represents a message in Z.ai API format.
type zaiMessage struct {
	Role    string `json:"role"`    // Role of the message sender
	Content string `json:"content"` // Message content
}

// zaiTool represents a tool definition in Z.ai API format.
type zaiTool struct {
	Type     string                 `json:"type"`     // Always "function"
	Function map[string]interface{} `json:"function"` // Function definition
}

// zaiResponse represents the response format from Z.ai API.
type zaiResponse struct {
	ID      string       `json:"id"`              // Response identifier
	Object  string       `json:"object"`          // Response object type
	Created int64        `json:"created"`         // Unix timestamp
	Model   string       `json:"model"`           // Model used
	Choices []zaiChoice  `json:"choices"`         // Response choices
	Usage   zaiUsage     `json:"usage"`           // Token usage
	Error   *zaiAPIError `json:"error,omitempty"` // API error if present
}

// zaiChoice represents a choice in the response.
type zaiChoice struct {
	Index        int           `json:"index"`                   // Choice index
	Message      zaiMessage    `json:"message"`                 // The generated message
	FinishReason string        `json:"finish_reason,omitempty"` // Reason generation stopped
	ToolCalls    []zaiToolCall `json:"tool_calls,omitempty"`    // Tool calls requested
}

// zaiToolCall represents a tool call in the response.
type zaiToolCall struct {
	ID       string `json:"id"`   // Tool call identifier
	Type     string `json:"type"` // Always "function"
	Function struct {
		Name      string `json:"name"`      // Function name
		Arguments string `json:"arguments"` // Function arguments as JSON string
	} `json:"function"`
}

// zaiUsage represents token usage information.
type zaiUsage struct {
	PromptTokens     int `json:"prompt_tokens"`     // Tokens in prompt
	CompletionTokens int `json:"completion_tokens"` // Tokens in completion
	TotalTokens      int `json:"total_tokens"`      // Total tokens used
}

// zaiAPIError represents an error response from the API.
type zaiAPIError struct {
	Message string `json:"message"` // Error message
	Type    string `json:"type"`    // Error type
	Code    string `json:"code"`    // Error code
}

// NewZAIProvider creates a new ZAIProvider instance.
func NewZAIProvider(cfg ZAIConfig, log *logger.Logger) *ZAIProvider {
	// Set default model if not provided
	if cfg.Model == "" {
		cfg.Model = "glm-4.7"
	}

	return &ZAIProvider{
		client: &http.Client{
			Timeout: ZAIRequestTimeout,
		},
		config: cfg,
		apiURL: ZAIEndpoint,
		logger: log,
	}
}

// Chat sends a chat completion request to Z.ai API.
func (p *ZAIProvider) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	// Map internal request to Z.ai API format
	zaiReq := p.mapChatRequest(req)

	// Convert to JSON
	reqBody, err := json.Marshal(zaiReq)
	if err != nil {
		p.logger.ErrorCtx(ctx, "Failed to marshal Z.ai request", err,
			logger.Field{Key: "model", Value: req.Model})
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Log request (without sensitive data)
	p.logger.DebugCtx(ctx, "Sending request to Z.ai API",
		logger.Field{Key: "model", Value: req.Model},
		logger.Field{Key: "messages_count", Value: len(req.Messages)})

	// Execute request with retries
	zaiResp, err := p.doRequestWithRetry(ctx, reqBody)
	if err != nil {
		return nil, err
	}

	// Map Z.ai response to internal format
	resp := p.mapChatResponse(zaiResp)

	p.logger.InfoCtx(ctx, "Received response from Z.ai API",
		logger.Field{Key: "model", Value: resp.Model},
		logger.Field{Key: "finish_reason", Value: resp.FinishReason},
		logger.Field{Key: "total_tokens", Value: resp.Usage.TotalTokens})

	return resp, nil
}

// doRequestWithRetry executes HTTP request with retry logic.
func (p *ZAIProvider) doRequestWithRetry(ctx context.Context, reqBody []byte) (*zaiResponse, error) {
	var lastErr error

	for attempt := 0; attempt < ZAIMaxRetries; attempt++ {
		if attempt > 0 {
			p.logger.DebugCtx(ctx, "Retrying request to Z.ai API",
				logger.Field{Key: "attempt", Value: attempt + 1},
				logger.Field{Key: "max_retries", Value: ZAIMaxRetries})

			// Wait before retrying
			select {
			case <-time.After(ZAIRetryDelay):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}

		zaiResp, err := p.doRequest(ctx, reqBody)
		if err == nil {
			return zaiResp, nil
		}

		lastErr = err

		// Don't retry on context cancellation
		if ctx.Err() != nil {
			return nil, lastErr
		}

		// Don't retry on certain HTTP status codes
		if httpErr, ok := err.(*zaiHTTPError); ok {
			if httpErr.StatusCode == 401 || httpErr.StatusCode == 403 {
				// Authentication errors - don't retry
				p.logger.ErrorCtx(ctx, "Authentication error with Z.ai API", httpErr,
					logger.Field{Key: "status_code", Value: httpErr.StatusCode})
				return nil, lastErr
			}
		}
	}

	return nil, fmt.Errorf("failed after %d attempts: %w", ZAIMaxRetries, lastErr)
}

// doRequest executes a single HTTP request to Z.ai API.
func (p *ZAIProvider) doRequest(ctx context.Context, reqBody []byte) (*zaiResponse, error) {
	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.apiURL, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", p.config.APIKey))

	// Execute request
	httpResp, err := p.client.Do(httpReq)
	if err != nil {
		p.logger.ErrorCtx(ctx, "Failed to execute request to Z.ai API", err)
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer httpResp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		p.logger.ErrorCtx(ctx, "Failed to read response body", err)
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Check HTTP status code
	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		p.logger.ErrorCtx(ctx, "Z.ai API returned error status", nil,
			logger.Field{Key: "status_code", Value: httpResp.StatusCode},
			logger.Field{Key: "response_body", Value: string(respBody)})

		return nil, &zaiHTTPError{
			StatusCode: httpResp.StatusCode,
			Body:       string(respBody),
		}
	}

	// Parse JSON response
	var zaiResp zaiResponse
	if err := json.Unmarshal(respBody, &zaiResp); err != nil {
		p.logger.ErrorCtx(ctx, "Failed to unmarshal Z.ai response", err,
			logger.Field{Key: "response_body", Value: string(respBody)})
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// Check for API error in response
	if zaiResp.Error != nil {
		p.logger.ErrorCtx(ctx, "Z.ai API returned error", nil,
			logger.Field{Key: "error_type", Value: zaiResp.Error.Type},
			logger.Field{Key: "error_code", Value: zaiResp.Error.Code},
			logger.Field{Key: "error_message", Value: zaiResp.Error.Message})
		return nil, fmt.Errorf("API error: %s (code: %s): %s",
			zaiResp.Error.Type, zaiResp.Error.Code, zaiResp.Error.Message)
	}

	return &zaiResp, nil
}

// mapChatRequest maps internal ChatRequest to Z.ai API format.
func (p *ZAIProvider) mapChatRequest(req ChatRequest) zaiRequest {
	messages := make([]zaiMessage, len(req.Messages))
	for i, msg := range req.Messages {
		messages[i] = zaiMessage{
			Role:    string(msg.Role),
			Content: msg.Content,
		}
	}

	zaiReq := zaiRequest{
		Messages:    messages,
		Model:       req.Model,
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
	}

	// Map tools if provided
	if len(req.Tools) > 0 {
		zaiReq.Tools = make([]zaiTool, len(req.Tools))
		for i, tool := range req.Tools {
			zaiReq.Tools[i] = zaiTool{
				Type: "function",
				Function: map[string]interface{}{
					"name":        tool.Name,
					"description": tool.Description,
					"parameters":  tool.Parameters,
				},
			}
		}
	}

	return zaiReq
}

// mapChatResponse maps Z.ai API response to internal ChatResponse format.
func (p *ZAIProvider) mapChatResponse(zaiResp *zaiResponse) *ChatResponse {
	if len(zaiResp.Choices) == 0 {
		return &ChatResponse{
			Content:      "",
			FinishReason: FinishReasonError,
			ToolCalls:    []ToolCall{},
			Usage: Usage{
				PromptTokens:     zaiResp.Usage.PromptTokens,
				CompletionTokens: zaiResp.Usage.CompletionTokens,
				TotalTokens:      zaiResp.Usage.TotalTokens,
			},
			Model: zaiResp.Model,
		}
	}

	choice := zaiResp.Choices[0]

	// Map tool calls if present
	toolCalls := make([]ToolCall, len(choice.ToolCalls))
	for i, tc := range choice.ToolCalls {
		toolCalls[i] = ToolCall{
			ID:        tc.ID,
			Name:      tc.Function.Name,
			Arguments: tc.Function.Arguments,
		}
	}

	return &ChatResponse{
		Content:      choice.Message.Content,
		FinishReason: FinishReason(choice.FinishReason),
		ToolCalls:    toolCalls,
		Usage: Usage{
			PromptTokens:     zaiResp.Usage.PromptTokens,
			CompletionTokens: zaiResp.Usage.CompletionTokens,
			TotalTokens:      zaiResp.Usage.TotalTokens,
		},
		Model: zaiResp.Model,
	}
}

// SupportsToolCalling returns true as Z.ai GLM-4.7 supports tool calling.
func (p *ZAIProvider) SupportsToolCalling() bool {
	return true
}

// GetDefaultModel returns the default model identifier.
func (p *ZAIProvider) GetDefaultModel() string {
	if p.config.Model != "" {
		return p.config.Model
	}
	return "glm-4.7"
}

// zaiHTTPError represents an HTTP error from the API.
type zaiHTTPError struct {
	StatusCode int    // HTTP status code
	Body       string // Response body
}

func (e *zaiHTTPError) Error() string {
	return fmt.Sprintf("HTTP error: status=%d, body=%s", e.StatusCode, e.Body)
}
