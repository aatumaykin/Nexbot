package subagent

import (
	"context"
	"github.com/aatumaykin/nexbot/internal/config"
	"github.com/aatumaykin/nexbot/internal/workspace"
	"sync"
	"testing"

	"github.com/aatumaykin/nexbot/internal/agent/loop"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSpawnWorkflow tests the complete workflow of an agent spawning a subagent via the spawn tool.
// This is an integration test that combines agent loop, tool registry, and subagent manager.
func TestSpawnWorkflow(t *testing.T) {
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
			Workspace:   ws,
			SessionDir:  tempDir,
			LLMProvider: &mockLLMProvider{response: "Subagent task completed"},
			Logger:      log,
		},
	})
	require.NoError(t, err)

	// Create mock agent loop
	agentLoop := newMockAgentLoop(manager, log)

	ctx := context.Background()

	// Test 1: Agent spawns subagent via spawn tool
	t.Run("spawn_via_tool", func(t *testing.T) {
		task := "Analyze code quality for the project"
		response, err := agentLoop.processMessage(ctx, task)
		require.NoError(t, err)
		// Verify response is JSON with id and session fields
		assert.Contains(t, response, `"id"`)
		assert.Contains(t, response, `"session"`)

		// Verify subagent was created
		assert.Equal(t, 1, manager.Count())

		// Verify subagent properties
		subagents := manager.List()
		assert.Len(t, subagents, 1)
		assert.NotEmpty(t, subagents[0].ID)
		assert.NotEmpty(t, subagents[0].Session)
	})

	// Test 2: Subagent processes task
	t.Run("subagent_process", func(t *testing.T) {
		subagents := manager.List()
		require.Len(t, subagents, 1)
		subagent := subagents[0]

		// Process a task through the subagent
		response, err := subagent.Process(ctx, "What is the code coverage?")
		require.NoError(t, err)
		assert.Equal(t, "Subagent task completed", response)
	})

	// Test 3: Multiple spawns
	t.Run("multiple_spawns", func(t *testing.T) {
		tasks := []string{
			"Run tests",
			"Check dependencies",
			"Generate documentation",
		}

		for _, task := range tasks {
			_, err := agentLoop.processMessage(ctx, task)
			require.NoError(t, err)
		}

		// Verify all subagents were spawned
		assert.Equal(t, 4, manager.Count()) // 1 initial + 3 new
	})

	// Cleanup
	manager.StopAll()
}

// TestSubagentWithScheduler tests a cron job spawning a subagent for scheduled tasks.
// This integrates the scheduler pattern with subagent management.
func TestSubagentWithScheduler(t *testing.T) {
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
			Workspace:   ws,
			SessionDir:  tempDir,
			LLMProvider: &mockLLMProvider{response: "Scheduled task completed"},
			Logger:      log,
		},
	})
	require.NoError(t, err)

	// Create mock agent loop
	agentLoop := newMockAgentLoop(manager, log)

	ctx := context.Background()

	// Simulate scheduler workflow
	t.Run("scheduled_task_spawn", func(t *testing.T) {
		scheduledTask := "Daily backup and report generation"

		// Spawn subagent for scheduled task
		response, err := agentLoop.processMessage(ctx, scheduledTask)
		require.NoError(t, err)
		// Verify response is JSON with id and session fields
		assert.Contains(t, response, `"id"`)
		assert.Contains(t, response, `"session"`)

		// Verify subagent exists
		assert.Equal(t, 1, manager.Count())

		// Get the spawned subagent
		subagents := manager.List()
		subagent := subagents[0]

		// Simulate subagent executing the scheduled task
		taskResult, err := subagent.Process(ctx, "Execute daily backup")
		require.NoError(t, err)
		assert.Equal(t, "Scheduled task completed", taskResult)
	})

	// Test multiple scheduled tasks
	t.Run("multiple_scheduled_tasks", func(t *testing.T) {
		scheduledTasks := []string{
			"Hourly health check",
			"Daily log rotation",
			"Weekly cleanup",
		}

		var wg sync.WaitGroup
		for _, task := range scheduledTasks {
			wg.Add(1)
			go func(task string) {
				defer wg.Done()
				_, err := agentLoop.processMessage(ctx, task)
				assert.NoError(t, err)
			}(task)
		}
		wg.Wait()

		// Verify all scheduled subagents spawned
		assert.Equal(t, 4, manager.Count()) // 1 initial + 3 new
	})

	// Cleanup
	manager.StopAll()
}
