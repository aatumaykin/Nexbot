package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// contextKey is the type for context keys to avoid collisions
type contextKey struct{}

var (
	sessionIDKey      contextKey = struct{}{}
	secretResolverKey contextKey = struct{}{}
)

// Error codes for tool execution
const (
	// Validation errors
	ErrCodeRequiredField = "required_field"
	ErrCodeInvalidFormat = "invalid_format"
	ErrCodeInvalidValue  = "invalid_value"
	ErrCodePathTraversal = "path_traversal"

	// Permission errors
	ErrCodePermissionDenied = "permission_denied"
	ErrCodeReadOnly         = "read_only"

	// Not found errors
	ErrCodeFileNotFound = "file_not_found"
	ErrCodeDirNotFound  = "dir_not_found"
	ErrCodeToolNotFound = "tool_not_found"

	// Execution errors
	ErrCodeExecutionFailed = "execution_failed"
	ErrCodeTimeout         = "timeout"
	ErrCodeCancelled       = "cancelled"

	// Rate limit errors
	ErrCodeRateLimitExceeded = "rate_limit_exceeded"

	// Storage errors
	ErrCodeStorageWriteFailed = "storage_write_failed"
	ErrCodeStorageReadFailed  = "storage_read_failed"

	// Disabled errors
	ErrCodeToolDisabled = "tool_disabled"
)

// Tool defines the interface that all tools must implement.
// A tool represents a function that can be called by the LLM agent.
type Tool interface {
	// Name returns the unique name of the tool.
	// This name is used to identify the tool in the function calling API.
	Name() string

	// Description returns a human-readable description of what the tool does.
	// This description helps the LLM understand when and how to use the tool.
	Description() string

	// Parameters returns a JSON Schema object describing the tool's input parameters.
	// The schema follows OpenAI function calling format.
	Parameters() map[string]interface{}

	// Execute runs the tool with the provided arguments.
	// args is a JSON-encoded string containing the tool's input parameters.
	Execute(args string) (string, error)
}

// ContextualTool is an optional interface that tools can implement to receive execution context.
// If a tool implements this interface, ExecuteWithContext will be called instead of Execute.
type ContextualTool interface {
	Tool

	// ExecuteWithContext runs the tool with the provided arguments and execution context.
	// The context can be used for cancellation, deadlines, and timeout handling.
	ExecuteWithContext(ctx context.Context, args string) (string, error)
}

// SecretAwareTool is an optional interface that tools can implement to receive secret resolver.
// If a tool implements this interface, SetSecretResolver will be called before execution.
type SecretAwareTool interface {
	Tool

	// SetSecretResolver sets the secret resolver function.
	// The resolver function takes (sessionID, text) and returns text with secrets resolved.
	SetSecretResolver(resolver func(string, string) string)
}

// Registry manages the collection of available tools.
// It provides thread-safe operations for registering and retrieving tools.
type Registry struct {
	mu    sync.RWMutex
	tools map[string]Tool
}

// NewRegistry creates a new empty tool registry.
func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]Tool),
	}
}

// Register adds a tool to the registry.
// If a tool with the same name already exists, it will be replaced.
func (r *Registry) Register(tool Tool) error {
	if tool == nil {
		return fmt.Errorf("cannot register nil tool")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	name := tool.Name()
	if name == "" {
		return fmt.Errorf("tool name cannot be empty")
	}

	r.tools[name] = tool
	return nil
}

// Get retrieves a tool by its name.
// Returns the tool and true if found, nil and false otherwise.
func (r *Registry) Get(name string) (Tool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tool, ok := r.tools[name]
	return tool, ok
}

// List returns all registered tools as a slice.
// The order of tools is not guaranteed.
func (r *Registry) List() []Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tools := make([]Tool, 0, len(r.tools))
	for _, tool := range r.tools {
		tools = append(tools, tool)
	}

	return tools
}

// ToSchema converts the registered tools to OpenAI-compatible function definitions.
// This returns a slice of ToolDefinition that can be sent to LLM providers.
func (r *Registry) ToSchema() []ToolDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()

	schemas := make([]ToolDefinition, 0, len(r.tools))
	for _, tool := range r.tools {
		schemas = append(schemas, ToolDefinition{
			Name:        tool.Name(),
			Description: tool.Description(),
			Parameters:  tool.Parameters(),
		})
	}

	return schemas
}

// ToolDefinition represents a tool definition in OpenAI function calling format.
type ToolDefinition struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// ToolCall represents a tool call request from the LLM.
type ToolCall struct {
	ID   string `json:"id"`
	Name string `json:"name"`

	// Arguments is a JSON string containing the tool's input parameters.
	Arguments string `json:"arguments"`
}

// ToolResult represents the result of executing a tool.
type ToolResult struct {
	ToolCallID string         `json:"tool_call_id"`
	Content    string         `json:"content"`
	Error      *ToolError     `json:"error,omitempty"`
	TimedOut   bool           `json:"timed_out,omitempty"`
	ExitCode   int            `json:"exit_code,omitempty"`
	Details    map[string]any `json:"details,omitempty"`
}

// ExecutionConfig represents the configuration for tool execution.
type ExecutionConfig struct {
	Timeout        time.Duration                       // Timeout for tool execution
	WorkingDir     string                              // Working directory for execution
	DefaultTimeout time.Duration                       // Default timeout if not specified
	SessionID      string                              // Session ID for secret isolation
	SecretResolver func(sessionID, text string) string // Secret resolver function
}

// DefaultExecutionConfig returns the default execution configuration.
func DefaultExecutionConfig() *ExecutionConfig {
	return &ExecutionConfig{
		DefaultTimeout: 30 * time.Second,
	}
}

// ExecuteToolCall executes a tool call using the provided registry.
// It parses the arguments, calls the tool with timeout and context, and returns the result.
func ExecuteToolCall(registry *Registry, tc ToolCall) (ToolResult, error) {
	return ExecuteToolCallWithContext(registry, tc, context.Background(), nil)
}

// ExecuteToolCallWithContext executes a tool call with execution context and configuration.
// It supports timeout, working directory, and secret resolution.
func ExecuteToolCallWithContext(registry *Registry, tc ToolCall, ctx context.Context, cfg *ExecutionConfig) (ToolResult, error) {
	tool, ok := registry.Get(tc.Name)
	if !ok {
		return ToolResult{
			ToolCallID: tc.ID,
			Error: NewNotFoundError(
				ErrCodeToolNotFound,
				fmt.Sprintf("tool not found: %s", tc.Name),
				""),
		}, nil
	}

	// Set secret resolver on tool if it supports it
	if cfg != nil && cfg.SecretResolver != nil {
		if secretAwareTool, ok := tool.(SecretAwareTool); ok {
			secretAwareTool.SetSecretResolver(cfg.SecretResolver)
		}
	}

	// Determine timeout
	var timeout time.Duration
	if cfg != nil {
		timeout = cfg.Timeout
		if timeout == 0 && cfg.DefaultTimeout != 0 {
			timeout = cfg.DefaultTimeout
		}
	}

	// Create execution context with timeout if configured
	execCtx := ctx
	if timeout > 0 {
		var cancel context.CancelFunc
		execCtx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	// Add sessionID and secret resolver to context
	if cfg != nil && cfg.SessionID != "" {
		execCtx = context.WithValue(execCtx, sessionIDKey, cfg.SessionID)
	}
	if cfg != nil && cfg.SecretResolver != nil {
		execCtx = context.WithValue(execCtx, secretResolverKey, cfg.SecretResolver)
	}

	// Create a channel for the result
	type executionResult struct {
		result string
		err    error
	}
	resultChan := make(chan executionResult, 1)

	// Execute the tool
	go func() {
		var res string
		var err error

		// Check if tool implements ContextualTool for context support
		if contextualTool, ok := tool.(ContextualTool); ok {
			res, err = contextualTool.ExecuteWithContext(execCtx, tc.Arguments)
		} else {
			// Fall back to regular Execute (no context support)
			res, err = tool.Execute(tc.Arguments)
		}

		resultChan <- executionResult{result: res, err: err}
	}()

	// Wait for result or timeout
	select {
	case res := <-resultChan:
		if res.err != nil {
			return ToolResult{
				ToolCallID: tc.ID,
				Error: NewExecutionError(
					ErrCodeExecutionFailed,
					res.err.Error(),
					"",
					0),
			}, nil
		}

		return ToolResult{
			ToolCallID: tc.ID,
			Content:    res.result,
		}, nil

	case <-execCtx.Done():
		// Check if it's a timeout
		if execCtx.Err() == context.DeadlineExceeded {
			return ToolResult{
				ToolCallID: tc.ID,
				Error: NewTimeoutError(
					ErrCodeTimeout,
					fmt.Sprintf("tool execution timed out after %v", timeout),
					map[string]any{"arguments": tc.Arguments}),
				TimedOut: true,
			}, nil
		}

		// Other context errors
		return ToolResult{
			ToolCallID: tc.ID,
			Error: NewTimeoutError(
				ErrCodeCancelled,
				fmt.Sprintf("tool execution cancelled: %v", execCtx.Err()),
				map[string]any{"arguments": tc.Arguments}),
		}, nil
	}
}

// ToJSON converts the tool definitions to JSON.
// Useful for debugging or logging.
func (r *Registry) ToJSON() (string, error) {
	schemas := r.ToSchema()
	data, err := json.MarshalIndent(schemas, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal schemas: %w", err)
	}
	return string(data), nil
}
