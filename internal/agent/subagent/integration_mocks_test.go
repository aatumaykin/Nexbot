package subagent

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/aatumaykin/nexbot/internal/logger"
	"github.com/aatumaykin/nexbot/internal/tools"
)

// mockAgentLoop is a mock implementation of the agent loop for testing.
// It simulates the main agent loop that can spawn subagents.
type mockAgentLoop struct {
	manager  *Manager
	toolReg  *tools.Registry
	mu       sync.Mutex
	response string
	logger   *logger.Logger
}

// spawnAdapter adapts the Manager.Spawn signature to tools.SpawnFunc.
// It converts the Subagent struct to JSON string format expected by the spawn tool.
func spawnAdapter(manager *Manager) tools.SpawnFunc {
	return func(ctx context.Context, parentSession string, task string) (string, error) {
		subagent, err := manager.Spawn(ctx, parentSession, task)
		if err != nil {
			return "", err
		}

		// Convert subagent to JSON result
		result := map[string]string{
			"id":      subagent.ID,
			"session": subagent.Session,
		}
		data, err := json.Marshal(result)
		if err != nil {
			return "", fmt.Errorf("failed to marshal subagent result: %w", err)
		}
		return string(data), nil
	}
}

// newMockAgentLoop creates a new mock agent loop for integration testing.
func newMockAgentLoop(manager *Manager, logger *logger.Logger) *mockAgentLoop {
	m := &mockAgentLoop{
		manager: manager,
		toolReg: tools.NewRegistry(),
		logger:  logger,
	}

	// Register spawn tool with adapter
	spawnTool := tools.NewSpawnTool(spawnAdapter(manager))
	if err := m.toolReg.Register(spawnTool); err != nil {
		panic(fmt.Sprintf("Failed to register spawn tool: %v", err))
	}

	return m
}

// processMessage simulates processing a message through the agent loop.
// It handles tool calls and returns a response.
func (m *mockAgentLoop) processMessage(ctx context.Context, message string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if message contains spawn tool call
	if containsSpawnToolCall(message) {
		// Parse and execute spawn tool call
		toolCall := tools.ToolCall{
			ID:        "test-call",
			Name:      "spawn",
			Arguments: extractSpawnArgs(message),
		}

		result, err := tools.ExecuteToolCall(m.toolReg, toolCall)
		if err != nil {
			return "", err
		}
		return result.Content, nil
	}

	// Return mock response for regular messages
	if m.response != "" {
		return m.response, nil
	}
	return "Mock agent response", nil
}

// containsSpawnToolCall checks if a message contains a spawn tool call.
func containsSpawnToolCall(message string) bool {
	return len(message) > 0 // Simplified check for testing
}

// extractSpawnArgs extracts spawn tool arguments from a message.
func extractSpawnArgs(message string) string {
	// Simplified extraction for testing
	return fmt.Sprintf(`{"task": "%s"}`, message)
}
