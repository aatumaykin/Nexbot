package loop

import (
	"context"
	"time"

	"github.com/aatumaykin/nexbot/internal/llm"
	"github.com/aatumaykin/nexbot/internal/logger"
	"github.com/aatumaykin/nexbot/internal/secrets"
	"github.com/aatumaykin/nexbot/internal/tools"
)

// ToolExecutor handles the execution of tool calls requested by the LLM.
type ToolExecutor struct {
	logger  *logger.Logger
	tools   *tools.Registry
	secrets *secrets.Store
}

// NewToolExecutor creates a new ToolExecutor.
func NewToolExecutor(logger *logger.Logger, toolsRegistry *tools.Registry, secretsStore *secrets.Store) *ToolExecutor {
	return &ToolExecutor{
		logger:  logger,
		tools:   toolsRegistry,
		secrets: secretsStore,
	}
}

// PrepareToolCalls converts LLM tool calls to internal tool calls format.
func (te *ToolExecutor) PrepareToolCalls(llmToolCalls []llm.ToolCall) []tools.ToolCall {
	if len(llmToolCalls) == 0 {
		return nil
	}

	toolCalls := make([]tools.ToolCall, len(llmToolCalls))
	for i, tc := range llmToolCalls {
		toolCalls[i] = tools.ToolCall{
			ID:        tc.ID,
			Name:      tc.Name,
			Arguments: tc.Arguments,
		}
	}

	return toolCalls
}

// SetSecretsStore sets the secrets store (for tools that need secret resolution).
func (te *ToolExecutor) SetSecretsStore(secretsStore *secrets.Store) {
	te.secrets = secretsStore
}

// GetSecretsStore returns the secrets store.
func (te *ToolExecutor) GetSecretsStore() *secrets.Store {
	return te.secrets
}

// ProcessToolCalls executes all tool calls and returns their results.
func (te *ToolExecutor) ProcessToolCalls(ctx context.Context, toolCalls []tools.ToolCall) ([]tools.ToolResult, error) {
	results := make([]tools.ToolResult, len(toolCalls))

	// Extract sessionID from context
	sessionID := getSessionIDFromContext(ctx)

	// Create secret resolver if secrets store is available
	var secretResolver func(string, string) string
	if te.secrets != nil && sessionID != "" {
		resolver := secrets.NewResolver(te.secrets)
		secretResolver = resolver.Resolve
	}

	for i, toolCall := range toolCalls {
		// Create execution config with secrets support
		cfg := &tools.ExecutionConfig{
			DefaultTimeout: 30 * time.Second,
			SessionID:      sessionID,
			SecretResolver: secretResolver,
		}

		result := te.ExecuteToolCall(ctx, toolCall, cfg)
		results[i] = result
	}

	return results, nil
}

// ExecuteToolCall executes a single tool call with context and logging.
func (te *ToolExecutor) ExecuteToolCall(ctx context.Context, toolCall tools.ToolCall, cfg *tools.ExecutionConfig) tools.ToolResult {
	te.logger.DebugCtx(ctx, "executing tool",
		logger.Field{Key: "tool_name", Value: toolCall.Name},
		logger.Field{Key: "tool_call_id", Value: toolCall.ID},
		logger.Field{Key: "session_id", Value: cfg.SessionID})

	start := time.Now()
	result, _ := tools.ExecuteToolCallWithContext(te.tools, toolCall, ctx, cfg)

	duration := time.Since(start)

	// Логируем результат
	if result.Error != nil {
		te.logger.ErrorCtx(ctx, "tool execution failed", result.Error,
			logger.Field{Key: "tool_name", Value: toolCall.Name},
			logger.Field{Key: "tool_call_id", Value: toolCall.ID},
			logger.Field{Key: "duration_ms", Value: duration.Milliseconds()},
			logger.Field{Key: "timed_out", Value: result.TimedOut})
	} else {
		te.logger.DebugCtx(ctx, "tool execution completed",
			logger.Field{Key: "tool_name", Value: toolCall.Name},
			logger.Field{Key: "tool_call_id", Value: toolCall.ID},
			logger.Field{Key: "duration_ms", Value: duration.Milliseconds()})
	}

	return result
}

// getSessionIDFromContext extracts sessionID from context.
// Uses context value key "session_id".
func getSessionIDFromContext(ctx context.Context) string {
	if sessionID, ok := ctx.Value("session_id").(string); ok {
		return sessionID
	}
	return ""
}
