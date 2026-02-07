package loop

import (
	"context"
	"time"

	"github.com/aatumaykin/nexbot/internal/llm"
	"github.com/aatumaykin/nexbot/internal/logger"
	"github.com/aatumaykin/nexbot/internal/tools"
)

// ToolExecutor handles the execution of tool calls requested by the LLM.
type ToolExecutor struct {
	logger *logger.Logger
	tools  *tools.Registry
}

// NewToolExecutor creates a new ToolExecutor.
func NewToolExecutor(logger *logger.Logger, toolsRegistry *tools.Registry) *ToolExecutor {
	return &ToolExecutor{
		logger: logger,
		tools:  toolsRegistry,
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

// ProcessToolCalls executes all tool calls and returns their results.
func (te *ToolExecutor) ProcessToolCalls(ctx context.Context, toolCalls []tools.ToolCall) ([]tools.ToolResult, error) {
	results := make([]tools.ToolResult, len(toolCalls))

	for i, toolCall := range toolCalls {
		result := te.ExecuteToolCall(ctx, toolCall)
		results[i] = result
	}

	return results, nil
}

// ExecuteToolCall executes a single tool call with context and logging.
func (te *ToolExecutor) ExecuteToolCall(ctx context.Context, toolCall tools.ToolCall) tools.ToolResult {
	te.logger.DebugCtx(ctx, "executing tool",
		logger.Field{Key: "tool_name", Value: toolCall.Name},
		logger.Field{Key: "tool_call_id", Value: toolCall.ID})

	start := time.Now()
	result, _ := tools.ExecuteToolCallWithContext(te.tools, toolCall, ctx, nil)

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
