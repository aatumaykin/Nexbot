package subagent

import (
	"context"
	"encoding/json"
	"github.com/aatumaykin/nexbot/internal/config"
	"github.com/aatumaykin/nexbot/internal/workspace"
	"testing"

	"github.com/aatumaykin/nexbot/internal/agent/loop"
	"github.com/aatumaykin/nexbot/internal/tools"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSubagentSessionIsolation tests that subagents maintain separate sessions.
func TestSubagentSessionIsolation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tempDir := t.TempDir()
	ws := workspace.New(config.WorkspaceConfig{Path: tempDir})
	log := testLogger()

	// Create subagent manager
	manager, err := NewManager(Config{
		SessionDir: tempDir,
		Logger:     log,
		LoopConfig: loop.Config{
			Workspace:   tempDir,
			SessionDir:  tempDir,
			LLMProvider: &mockLLMProvider{response: "OK"},
			Logger:      log,
		},
	})
	require.NoError(t, err)

	ctx := context.Background()

	// Spawn two subagents
	sub1, err := manager.Spawn(ctx, "parent", "Task 1")
	require.NoError(t, err)

	sub2, err := manager.Spawn(ctx, "parent", "Task 2")
	require.NoError(t, err)

	// Verify sessions are different
	assert.NotEqual(t, sub1.Session, sub2.Session)
	assert.Contains(t, sub1.Session, SessionIDPrefix)
	assert.Contains(t, sub2.Session, SessionIDPrefix)

	// Verify both subagents can process independently
	resp1, err := sub1.Process(ctx, "First task")
	require.NoError(t, err)
	assert.Equal(t, "OK", resp1)

	resp2, err := sub2.Process(ctx, "Second task")
	require.NoError(t, err)
	assert.Equal(t, "OK", resp2)

	// Cleanup
	manager.StopAll()
}

// TestSubagentLifecycle tests the complete lifecycle of subagents.
func TestSubagentLifecycle(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tempDir := t.TempDir()
	ws := workspace.New(config.WorkspaceConfig{Path: tempDir})
	log := testLogger()

	// Create subagent manager
	manager, err := NewManager(Config{
		SessionDir: tempDir,
		Logger:     log,
		LoopConfig: loop.Config{
			Workspace:   tempDir,
			SessionDir:  tempDir,
			LLMProvider: &mockLLMProvider{response: "Lifecycle OK"},
			Logger:      log,
		},
	})
	require.NoError(t, err)

	ctx := context.Background()

	// Spawn subagent
	sub, err := manager.Spawn(ctx, "parent", "Lifecycle test")
	require.NoError(t, err)
	assert.Equal(t, 1, manager.Count())

	// Process task
	resp, err := sub.Process(ctx, "Process task")
	require.NoError(t, err)
	assert.Equal(t, "Lifecycle OK", resp)

	// Stop subagent
	err = manager.Stop(sub.ID)
	require.NoError(t, err)
	assert.Equal(t, 0, manager.Count())

	// Verify cannot retrieve stopped subagent
	_, err = manager.Get(sub.ID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "subagent not found")

	// Spawn new subagent
	sub2, err := manager.Spawn(ctx, "parent", "New task")
	require.NoError(t, err)
	assert.Equal(t, 1, manager.Count())

	// Verify new subagent has different ID
	assert.NotEqual(t, sub.ID, sub2.ID)

	// Stop all subagents
	manager.StopAll()
	assert.Equal(t, 0, manager.Count())
}

// TestSpawnToolSchemaSerialization tests that the spawn tool schema can be serialized correctly.
func TestSpawnToolSchemaSerialization(t *testing.T) {
	tempDir := t.TempDir()
	ws := workspace.New(config.WorkspaceConfig{Path: tempDir})
	log := testLogger()

	// Create subagent manager
	manager, err := NewManager(Config{
		SessionDir: tempDir,
		Logger:     log,
		LoopConfig: loop.Config{
			Workspace:   tempDir,
			SessionDir:  tempDir,
			LLMProvider: &mockLLMProvider{},
			Logger:      log,
		},
	})
	require.NoError(t, err)

	// Create spawn tool with adapter
	spawnTool := tools.NewSpawnTool(spawnAdapter(manager))

	// Get schema
	schema := spawnTool.Parameters()
	assert.NotNil(t, schema)
	assert.Equal(t, "object", schema["type"])

	// Serialize to JSON
	data, err := json.Marshal(schema)
	require.NoError(t, err)

	// Deserialize and verify
	var unmarshaled map[string]any
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	// Verify properties
	props, ok := unmarshaled["properties"].(map[string]any)
	require.True(t, ok)
	assert.Contains(t, props, "task")
	assert.Contains(t, props, "timeout_seconds")

	// Verify required fields
	required, ok := unmarshaled["required"].([]any)
	require.True(t, ok)
	assert.Contains(t, required, "task")
}
