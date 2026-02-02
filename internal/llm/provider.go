package llm

import (
	"context"
)

// Provider defines the interface for LLM (Large Language Model) providers.
// Different LLM providers (OpenAI, Anthropic, Z.ai, etc.) must implement this interface.
type Provider interface {
	// Chat sends a chat completion request to the LLM provider.
	// It takes a context for cancellation/timeout and a ChatRequest with the conversation
	// parameters, and returns a ChatResponse with the model's reply.
	Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error)

	// SupportsToolCalling returns true if the provider supports tool/function calling.
	// This allows the system to know whether to send tool definitions in requests.
	SupportsToolCalling() bool

	// GetDefaultModel returns the default model identifier for this provider.
	// Used when no specific model is requested by the user.
	GetDefaultModel() string
}

// Role represents the role of a message sender in the conversation.
type Role string

const (
	RoleSystem    Role = "system"    // System message provides context/instructions
	RoleUser      Role = "user"      // User message represents user input
	RoleAssistant Role = "assistant" // Assistant message represents model response
	RoleTool      Role = "tool"      // Tool message represents tool execution results
)

// Message represents a single message in the chat conversation.
type Message struct {
	Role    Role   `json:"role"`    // The role of the message sender
	Content string `json:"content"` // The content of the message

	// ToolCallID is set for RoleTool messages to identify which tool call this result is for
	ToolCallID string `json:"tool_call_id,omitempty"`
}

// FinishReason indicates why the model stopped generating tokens.
type FinishReason string

const (
	FinishReasonStop      FinishReason = "stop"       // Model reached a natural stopping point
	FinishReasonLength    FinishReason = "length"     // Model exceeded max tokens
	FinishReasonToolCalls FinishReason = "tool_calls" // Model requested tool calls
	FinishReasonError     FinishReason = "error"      // Generation stopped due to an error
)

// ToolCall represents a requested tool/function call by the model.
type ToolCall struct {
	ID   string `json:"id"`   // Unique identifier for this tool call
	Name string `json:"name"` // Name of the tool/function to call

	// Arguments is a JSON string containing the arguments for the tool call
	Arguments string `json:"arguments"`
}

// Usage tracks token usage information for the request.
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`     // Number of tokens in the prompt
	CompletionTokens int `json:"completion_tokens"` // Number of tokens in the completion
	TotalTokens      int `json:"total_tokens"`      // Total number of tokens used
}

// ChatRequest represents a request to send to the LLM provider for chat completion.
type ChatRequest struct {
	Messages    []Message `json:"messages"`    // The conversation history
	Model       string    `json:"model"`       // The model to use for completion
	Temperature float64   `json:"temperature"` // Sampling temperature (0.0-2.0)
	MaxTokens   int       `json:"max_tokens"`  // Maximum tokens to generate

	// Tools is a list of tools/functions the model can call. Only used if supported.
	Tools []ToolDefinition `json:"tools,omitempty"`
}

// ToolDefinition defines a tool that the model can call.
type ToolDefinition struct {
	Name        string `json:"name"`        // Name of the tool
	Description string `json:"description"` // Description of what the tool does

	// Parameters is a JSON Schema object describing the tool's input parameters
	Parameters map[string]interface{} `json:"parameters"`
}

// ChatResponse represents a response from the LLM provider.
type ChatResponse struct {
	Content      string       `json:"content"`       // The model's text response
	FinishReason FinishReason `json:"finish_reason"` // Reason generation stopped
	ToolCalls    []ToolCall   `json:"tool_calls"`    // Tool calls requested by model
	Usage        Usage        `json:"usage"`         // Token usage information

	// Model is the actual model used for the completion (may differ from request)
	Model string `json:"model"`
}
