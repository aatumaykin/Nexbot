package subagent

import (
	"context"
	"fmt"
	"github.com/aatumaykin/nexbot/internal/config"
	"github.com/aatumaykin/nexbot/internal/workspace"
	"sync"
	"testing"

	"github.com/aatumaykin/nexbot/internal/agent/loop"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMultiSubagent tests spawning and managing multiple concurrent subagents.
// This tests thread-safety and isolation of subagent sessions.
func TestMultiSubagent(t *testing.T) {
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
			LLMProvider: &mockLLMProvider{response: "Task completed"},
			Logger:      log,
		},
	})
	require.NoError(t, err)

	// Create mock agent loop
	agentLoop := newMockAgentLoop(manager, log)

	ctx := context.Background()

	// Test concurrent spawns
	t.Run("concurrent_spawns", func(t *testing.T) {
		numConcurrent := 10
		var wg sync.WaitGroup

		// Spawn multiple subagents concurrently
		for i := range numConcurrent {
			wg.Add(1)
			go func(taskNum int) {
				defer wg.Done()
				task := fmt.Sprintf("Concurrent task %d", taskNum)
				_, err := agentLoop.processMessage(ctx, task)
				assert.NoError(t, err)
			}(i)
		}

		wg.Wait()

		// Verify all subagents were spawned
		assert.Equal(t, numConcurrent, manager.Count())
	})

	// Test subagent isolation
	t.Run("subagent_isolation", func(t *testing.T) {
		subagents := manager.List()
		require.GreaterOrEqual(t, len(subagents), 10)

		// Verify each subagent has unique ID and session
		ids := make(map[string]bool)
		sessions := make(map[string]bool)

		for _, sub := range subagents {
			assert.False(t, ids[sub.ID], "duplicate subagent ID")
			assert.False(t, sessions[sub.Session], "duplicate session ID")

			ids[sub.ID] = true
			sessions[sub.Session] = true

			assert.NotEmpty(t, sub.ID)
			assert.NotEmpty(t, sub.Session)
			assert.Contains(t, sub.Session, SessionIDPrefix)
		}
	})

	// Test concurrent task processing
	t.Run("concurrent_processing", func(t *testing.T) {
		subagents := manager.List()
		require.GreaterOrEqual(t, len(subagents), 5)

		var wg sync.WaitGroup
		for i, subagent := range subagents[:5] {
			wg.Add(1)
			go func(idx int, sub *Subagent) {
				defer wg.Done()
				task := fmt.Sprintf("Process data batch %d", idx)
				_, err := sub.Process(ctx, task)
				assert.NoError(t, err)
			}(i, subagent)
		}

		wg.Wait()
	})

	// Cleanup
	manager.StopAll()
}
