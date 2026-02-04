package tools

import (
	"context"
	"encoding/json"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockSpawnManager is a mock subagent manager for integration testing.
// It avoids circular imports by not importing the actual subagent package.
type mockSpawnManager struct {
	mu            sync.Mutex
	subagents     map[string]*mockSubagent
	subagentCount int
}

type mockSubagent struct {
	ID      string
	Session string
}

func newMockSpawnManager() *mockSpawnManager {
	return &mockSpawnManager{
		subagents: make(map[string]*mockSubagent),
	}
}

func (m *mockSpawnManager) Spawn(ctx context.Context, parentSession string, task string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.subagentCount++
	subagent := &mockSubagent{
		ID:      "mock-subagent-" + string(rune(m.subagentCount)),
		Session: "mock-session-" + string(rune(m.subagentCount)),
	}
	m.subagents[subagent.ID] = subagent

	result := map[string]string{
		"id":      subagent.ID,
		"session": subagent.Session,
	}
	data, _ := json.Marshal(result)
	return string(data), nil
}

func (m *mockSpawnManager) Count() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.subagents)
}

// TestSpawnToolIntegration tests spawn tool with registry and execution flow.
func TestSpawnToolIntegration(t *testing.T) {
	// Create mock spawn manager
	mockMgr := newMockSpawnManager()

	// Create spawn tool
	tool := NewSpawnTool(mockMgr.Spawn)

	// Register tool in registry
	registry := NewRegistry()
	registry.Register(tool)

	// Verify tool is registered
	schemas := registry.ToSchema()
	assert.Len(t, schemas, 1)
	assert.Equal(t, "spawn", schemas[0].Name)

	// Get tool from registry
	retrievedTool, ok := registry.Get("spawn")
	require.True(t, ok)
	assert.Equal(t, tool.Name(), retrievedTool.Name())

	// Execute tool via ExecuteToolCall
	toolCall := ToolCall{
		ID:   "test-call-123",
		Name: "spawn",
		Arguments: `{
			"task": "Test integration task",
			"timeout_seconds": 300
		}`,
	}

	result, err := ExecuteToolCall(registry, toolCall)
	require.NoError(t, err)
	assert.Equal(t, "test-call-123", result.ToolCallID)
	assert.Empty(t, result.Error) // No error expected
	assert.Contains(t, result.Content, "Subagent spawned with ID")

	// Verify subagent was created in mock manager
	assert.Equal(t, 1, mockMgr.Count())
}

// TestSpawnToolIntegrationMultipleCalls tests multiple spawn calls.
func TestSpawnToolIntegrationMultipleCalls(t *testing.T) {
	mockMgr := newMockSpawnManager()
	tool := NewSpawnTool(mockMgr.Spawn)
	registry := NewRegistry()
	registry.Register(tool)

	// Spawn multiple subagents
	for i := 1; i <= 5; i++ {
		taskNum := i
		toolCall := ToolCall{
			ID:        "test-call-" + string(rune('0'+taskNum%10)),
			Name:      "spawn",
			Arguments: `{"task": "Parallel task ` + string(rune('0'+taskNum%10)) + `"}`,
		}

		result, err := ExecuteToolCall(registry, toolCall)
		require.NoError(t, err)
		assert.Empty(t, result.Error)
		assert.Contains(t, result.Content, "Subagent spawned with ID")
	}

	// Verify all subagents were created
	assert.Equal(t, 5, mockMgr.Count())
}

// TestSpawnToolIntegrationWithTimeout tests timeout handling in integration.
func TestSpawnToolIntegrationWithTimeout(t *testing.T) {
	mockMgr := newMockSpawnManager()
	tool := NewSpawnTool(mockMgr.Spawn)
	registry := NewRegistry()
	registry.Register(tool)

	toolCall := ToolCall{
		ID:   "timeout-test-call",
		Name: "spawn",
		Arguments: `{
			"task": "Task with custom timeout",
			"timeout_seconds": 120
		}`,
	}

	result, err := ExecuteToolCall(registry, toolCall)
	require.NoError(t, err)
	assert.Empty(t, result.Error)
	assert.Contains(t, result.Content, "Subagent spawned with ID")
}

// TestSpawnToolIntegrationErrorHandling tests error handling in integration.
func TestSpawnToolIntegrationErrorHandling(t *testing.T) {
	// Create spawn func that returns error
	errorSpawnFunc := func(ctx context.Context, parentSession string, task string) (string, error) {
		return "", assert.AnError
	}

	tool := NewSpawnTool(errorSpawnFunc)
	registry := NewRegistry()
	registry.Register(tool)

	// Test invalid JSON
	toolCall := ToolCall{
		ID:        "error-test-call",
		Name:      "spawn",
		Arguments: `{invalid json}`,
	}

	result, err := ExecuteToolCall(registry, toolCall)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Error)
	assert.Contains(t, result.Error, "parse")

	// Test missing task
	toolCall.Arguments = `{}`
	result, err = ExecuteToolCall(registry, toolCall)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Error)
	assert.Contains(t, result.Error, "required")

	// Test invalid timeout
	toolCall.Arguments = `{"task": "test", "timeout_seconds": -5}`
	result, err = ExecuteToolCall(registry, toolCall)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Error)
	assert.Contains(t, result.Error, "positive")

	// Test spawn error
	toolCall.Arguments = `{"task": "this will error"}`
	result, err = ExecuteToolCall(registry, toolCall)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Error)
	assert.Contains(t, result.Error, "failed to spawn")
}

// TestSpawnToolIntegrationSchema tests schema generation and serialization.
func TestSpawnToolIntegrationSchema(t *testing.T) {
	mockMgr := newMockSpawnManager()
	tool := NewSpawnTool(mockMgr.Spawn)
	registry := NewRegistry()
	registry.Register(tool)

	// Get schema
	schemas := registry.ToSchema()
	require.Len(t, schemas, 1)
	schema := schemas[0]

	// Verify schema fields
	assert.Equal(t, "spawn", schema.Name)
	assert.NotEmpty(t, schema.Description)
	assert.NotNil(t, schema.Parameters)

	// Convert to JSON and back
	data, err := json.Marshal(schema)
	require.NoError(t, err)

	var unmarshaled map[string]interface{}
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	// Verify schema is valid
	assert.Equal(t, "spawn", unmarshaled["name"])

	// Verify parameters are valid
	params, ok := unmarshaled["parameters"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "object", params["type"])

	props, ok := params["properties"].(map[string]interface{})
	require.True(t, ok)
	_, hasTask := props["task"]
	_, hasTimeout := props["timeout_seconds"]
	assert.True(t, hasTask)
	assert.True(t, hasTimeout)
}

// TestSpawnToolIntegrationEmptyTask tests empty task handling.
func TestSpawnToolIntegrationEmptyTask(t *testing.T) {
	mockMgr := newMockSpawnManager()
	tool := NewSpawnTool(mockMgr.Spawn)
	registry := NewRegistry()
	registry.Register(tool)

	toolCall := ToolCall{
		ID:        "empty-task-call",
		Name:      "spawn",
		Arguments: `{"task": ""}`,
	}

	result, err := ExecuteToolCall(registry, toolCall)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Error)
	assert.Contains(t, result.Error, "required")

	// No subagent should be created
	assert.Equal(t, 0, mockMgr.Count())
}

// TestSpawnToolIntegrationToolNotFound tests handling of unknown tools.
func TestSpawnToolIntegrationToolNotFound(t *testing.T) {
	registry := NewRegistry()

	toolCall := ToolCall{
		ID:        "unknown-tool-call",
		Name:      "unknown_tool",
		Arguments: `{}`,
	}

	result, err := ExecuteToolCall(registry, toolCall)
	require.NoError(t, err)
	assert.Equal(t, "unknown-tool-call", result.ToolCallID)
	assert.NotEmpty(t, result.Error)
	assert.Contains(t, result.Error, "not found")
}
