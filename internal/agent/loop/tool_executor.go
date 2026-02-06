package loop

import (
	"context"
	"fmt"

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
	te.logger.DebugCtx(ctx, "Executing tool",
		logger.Field{Key: "tool_name", Value: toolCall.Name},
		logger.Field{Key: "tool_call_id", Value: toolCall.ID},
		logger.Field{Key: "arguments", Value: toolCall.Arguments})

	result, err := tools.ExecuteToolCallWithContext(te.tools, toolCall, ctx, nil)
	if err != nil {
		te.logger.ErrorCtx(ctx, "Tool execution failed", err,
			logger.Field{Key: "tool_name", Value: toolCall.Name})
		return tools.ToolResult{
			ToolCallID: toolCall.ID,
			Error:      fmt.Sprintf("Tool execution error: %v", err),
		}
	}

	te.logger.DebugCtx(ctx, "Tool execution result",
		logger.Field{Key: "tool_name", Value: toolCall.Name},
		logger.Field{Key: "success", Value: result.Error == ""},
		logger.Field{Key: "result_length", Value: len(result.Content)},
		logger.Field{Key: "error", Value: result.Error})

	return result
}
