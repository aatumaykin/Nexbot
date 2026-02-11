package tools

import (
	"context"
	"fmt"
	"time"
)

// SpawnFunc is a function type for spawning subagents.
// This avoids circular import with the subagent package.
type SpawnFunc func(ctx context.Context, parentSession string, task string) (string, error)

// SpawnTool implements the Tool and ContextualTool interfaces for spawning subagents.
// It creates isolated agent instances with their own sessions for parallel task execution.
type SpawnTool struct {
	spawnFunc SpawnFunc
}

// SpawnResult represents the result of spawning a subagent.
type SpawnResult struct {
	ID      string `json:"id"`
	Session string `json:"session"`
}

// SpawnArgs represents the arguments for the spawn tool.
type SpawnArgs struct {
	Task           string `json:"task"`                      // Task description for the subagent
	TimeoutSeconds *int   `json:"timeout_seconds,omitempty"` // Optional timeout in seconds (default: 300)
}

// NewSpawnTool creates a new SpawnTool instance.
// The spawnFunc parameter is used for creating subagents.
func NewSpawnTool(spawnFunc SpawnFunc) *SpawnTool {
	return &SpawnTool{spawnFunc: spawnFunc}
}

// Name returns the tool name.
func (t *SpawnTool) Name() string {
	return "spawn"
}

// Description returns a description of what the tool does.
func (t *SpawnTool) Description() string {
	return "Create a subagent for parallel task execution. The subagent will have its own isolated session and memory."
}

// Parameters returns the JSON Schema for the tool's parameters.
func (t *SpawnTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"task": map[string]any{
				"type":        "string",
				"description": "Task description for the subagent",
			},
			"timeout_seconds": map[string]any{
				"type":        "number",
				"description": "Optional timeout in seconds (default: 300)",
			},
		},
		"required": []string{"task"},
	}
}

// Execute runs the tool with the provided arguments.
// args is a JSON-encoded string containing the tool's input parameters.
// This method is part of the Tool interface and delegates to ExecuteWithContext.
func (t *SpawnTool) Execute(args string) (string, error) {
	return t.ExecuteWithContext(context.Background(), args)
}

// ExecuteWithContext runs the tool with the provided arguments and execution context.
// The context can be used for cancellation, deadlines, and timeout handling.
// This is the preferred method for spawning subagents as it provides better control.
func (t *SpawnTool) ExecuteWithContext(ctx context.Context, args string) (string, error) {
	// Parse arguments
	var spawnArgs SpawnArgs
	if err := parseJSON(args, &spawnArgs); err != nil {
		return "", fmt.Errorf("failed to parse arguments: %w", err)
	}

	// Validate required arguments
	if spawnArgs.Task == "" {
		return "", fmt.Errorf("task is required")
	}

	// Apply timeout to context (default: 300 seconds if not specified)
	timeoutSeconds := 300
	if spawnArgs.TimeoutSeconds != nil {
		if *spawnArgs.TimeoutSeconds <= 0 {
			return "", fmt.Errorf("timeout_seconds must be positive")
		}
		timeoutSeconds = *spawnArgs.TimeoutSeconds
	}
	timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSeconds)*time.Second)
	defer cancel()
	ctx = timeoutCtx

	// Execute task via subagent using "parent" as parent session ID
	// Note: In a future enhancement, this could be the actual parent agent's session ID
	result, err := t.spawnFunc(ctx, "parent", spawnArgs.Task)
	if err != nil {
		return "", fmt.Errorf("failed to execute task via subagent: %w", err)
	}

	// Return result directly
	return result, nil
}

// Ensure SpawnTool implements Tool interface
var _ Tool = (*SpawnTool)(nil)

// Ensure SpawnTool implements ContextualTool interface
var _ ContextualTool = (*SpawnTool)(nil)
