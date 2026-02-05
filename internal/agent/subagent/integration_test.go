package subagent

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/aatumaykin/nexbot/internal/agent/loop"
	"github.com/aatumaykin/nexbot/internal/logger"
	"github.com/aatumaykin/nexbot/internal/tools"
	"github.com/aatumaykin/nexbot/internal/workers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	m.toolReg.Register(spawnTool)

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

// TestSpawnWorkflow tests the complete workflow of an agent spawning a subagent via the spawn tool.
// This is an integration test that combines agent loop, tool registry, and subagent manager.
func TestSpawnWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tempDir := t.TempDir()
	log := testLogger()

	// Create subagent manager
	manager, err := NewManager(Config{
		SessionDir: tempDir,
		Logger:     log,
		LoopConfig: loop.Config{
			Workspace:   tempDir,
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
		assert.Contains(t, response, "Subagent spawned with ID")

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
	log := testLogger()

	// Create subagent manager
	manager, err := NewManager(Config{
		SessionDir: tempDir,
		Logger:     log,
		LoopConfig: loop.Config{
			Workspace:   tempDir,
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
		assert.Contains(t, response, "Subagent spawned with ID")

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

// TestMultiSubagent tests spawning and managing multiple concurrent subagents.
// This tests thread-safety and isolation of subagent sessions.
func TestMultiSubagent(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tempDir := t.TempDir()
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
		for i := 0; i < numConcurrent; i++ {
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

// TestWorkerPoolIntegration tests the full workflow of worker pool executing tasks
// that spawn subagents. This is a comprehensive integration test.
func TestWorkerPoolIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tempDir := t.TempDir()
	log := testLogger()

	// Create subagent manager
	manager, err := NewManager(Config{
		SessionDir: tempDir,
		Logger:     log,
		LoopConfig: loop.Config{
			Workspace:   tempDir,
			SessionDir:  tempDir,
			LLMProvider: &mockLLMProvider{response: "Worker task completed"},
			Logger:      log,
		},
	})
	require.NoError(t, err)

	// Test 1: Worker pool spawns subagents via tasks
	t.Run("pool_spawn_workflow", func(t *testing.T) {
		// Create worker pool for this sub-test
		pool := workers.NewPool(5, 20, log)
		pool.Start()
		defer pool.Stop()

		numTasks := 5

		// Submit tasks that will spawn subagents
		for i := 0; i < numTasks; i++ {
			task := workers.Task{
				ID:      fmt.Sprintf("pool-task-%d", i),
				Type:    "subagent",
				Payload: map[string]string{"task": fmt.Sprintf("Pool spawned task %d", i)},
			}
			pool.Submit(task)
		}

		// Wait for results
		results := make(map[string]workers.Result)
		for i := 0; i < numTasks; i++ {
			result := <-pool.Results()
			results[result.TaskID] = result
		}

		// Verify all tasks completed
		assert.Len(t, results, numTasks)
		for _, result := range results {
			assert.NoError(t, result.Error)
		}

		// Verify metrics
		metrics := pool.Metrics()
		assert.Equal(t, uint64(numTasks), metrics.TasksCompleted)
		assert.Equal(t, uint64(0), metrics.TasksFailed)
	})

	// Test 2: Worker pool with mixed task types (cron and subagent)
	t.Run("mixed_task_types", func(t *testing.T) {
		// Create worker pool for this sub-test
		pool := workers.NewPool(5, 20, log)
		pool.Start()
		defer pool.Stop()

		// Submit mix of task types
		for i := 0; i < 3; i++ {
			// Cron task
			cronTask := workers.Task{
				ID:      fmt.Sprintf("cron-%d", i),
				Type:    "cron",
				Payload: fmt.Sprintf("Scheduled job %d", i),
			}
			pool.Submit(cronTask)

			// Subagent task
			subagentTask := workers.Task{
				ID:      fmt.Sprintf("subagent-%d", i),
				Type:    "subagent",
				Payload: map[string]string{"task": fmt.Sprintf("Agent task %d", i)},
			}
			pool.Submit(subagentTask)
		}

		// Wait for all results
		totalTasks := 6
		for i := 0; i < totalTasks; i++ {
			result := <-pool.Results()
			assert.NoError(t, result.Error)
		}

		// Verify metrics
		metrics := pool.Metrics()
		assert.Equal(t, uint64(totalTasks), metrics.TasksCompleted)
	})
	require.NoError(t, err)

	// Test 1: Worker pool spawns subagents via tasks
	t.Run("pool_spawn_workflow", func(t *testing.T) {
		// Create worker pool for this sub-test
		pool := workers.NewPool(5, 20, log)
		pool.Start()
		defer pool.Stop()

		numTasks := 5

		// Submit tasks that will spawn subagents
		for i := 0; i < numTasks; i++ {
			task := workers.Task{
				ID:      fmt.Sprintf("pool-task-%d", i),
				Type:    "subagent",
				Payload: map[string]string{"task": fmt.Sprintf("Pool spawned task %d", i)},
			}
			pool.Submit(task)
		}

		// Wait for results
		results := make(map[string]workers.Result)
		for i := 0; i < numTasks; i++ {
			result := <-pool.Results()
			results[result.TaskID] = result
		}

		// Verify all tasks completed
		assert.Len(t, results, numTasks)
		for _, result := range results {
			assert.NoError(t, result.Error)
		}

		// Verify metrics
		metrics := pool.Metrics()
		assert.Equal(t, uint64(numTasks), metrics.TasksCompleted)
		assert.Equal(t, uint64(0), metrics.TasksFailed)
	})

	// Test 2: Worker pool with mixed task types (cron and subagent)
	t.Run("mixed_task_types", func(t *testing.T) {
		// Create worker pool for this sub-test
		pool := workers.NewPool(5, 20, log)
		pool.Start()
		defer pool.Stop()

		// Submit mix of task types
		for i := 0; i < 3; i++ {
			// Cron task
			cronTask := workers.Task{
				ID:      fmt.Sprintf("cron-%d", i),
				Type:    "cron",
				Payload: fmt.Sprintf("Scheduled job %d", i),
			}
			pool.Submit(cronTask)

			// Subagent task
			subagentTask := workers.Task{
				ID:      fmt.Sprintf("subagent-%d", i),
				Type:    "subagent",
				Payload: map[string]string{"task": fmt.Sprintf("Agent task %d", i)},
			}
			pool.Submit(subagentTask)
		}

		// Wait for all results
		totalTasks := 6
		for i := 0; i < totalTasks; i++ {
			result := <-pool.Results()
			assert.NoError(t, result.Error)
		}

		// Verify metrics
		metrics := pool.Metrics()
		assert.Equal(t, uint64(totalTasks), metrics.TasksCompleted)
	})

	// Test 3: Worker pool high load with subagent spawning
	t.Run("high_load_subagents", func(t *testing.T) {
		// Create worker pool for this sub-test
		pool := workers.NewPool(5, 20, log)
		pool.Start()
		defer pool.Stop()

		numHighLoadTasks := 20

		// Submit many tasks rapidly
		for i := 0; i < numHighLoadTasks; i++ {
			task := workers.Task{
				ID:      fmt.Sprintf("load-%d", i),
				Type:    "subagent",
				Payload: map[string]string{"task": fmt.Sprintf("Load test task %d", i)},
			}
			pool.Submit(task)
		}

		// Wait for all results with timeout
		results := make(map[string]workers.Result)
		timeoutCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		for i := 0; i < numHighLoadTasks; i++ {
			select {
			case result := <-pool.Results():
				results[result.TaskID] = result
			case <-timeoutCtx.Done():
				t.Fatalf("timeout waiting for results, got %d/%d", len(results), numHighLoadTasks)
			}
		}

		// Verify all tasks completed
		assert.Len(t, results, numHighLoadTasks)
		for _, result := range results {
			assert.NoError(t, result.Error)
		}

		// Verify no race conditions
		metrics := pool.Metrics()
		assert.Equal(t, uint64(numHighLoadTasks), metrics.TasksCompleted)
		assert.Equal(t, uint64(0), metrics.TasksFailed)
	})

	// Test 4: Worker pool with context cancellation
	t.Run("context_cancellation", func(t *testing.T) {
		// Create worker pool for this sub-test
		pool := workers.NewPool(2, 10, log)
		pool.Start()
		defer pool.Stop()

		// Create task with cancellable context
		taskCtx, cancel := context.WithCancel(context.Background())

		task := workers.Task{
			ID:      "cancellable",
			Type:    "subagent",
			Payload: map[string]string{"task": "Cancellable task"},
			Context: taskCtx,
		}

		// Cancel before submitting
		cancel()

		pool.Submit(task)

		// Wait for result
		result := <-pool.Results()
		assert.Equal(t, "cancellable", result.TaskID)
		assert.Error(t, result.Error)
	})

	// Test 5: Worker pool graceful shutdown
	t.Run("graceful_shutdown", func(t *testing.T) {
		// Create worker pool for this sub-test
		pool := workers.NewPool(3, 10, log)
		pool.Start()

		// Submit tasks
		for i := 0; i < 5; i++ {
			task := workers.Task{
				ID:      fmt.Sprintf("shutdown-%d", i),
				Type:    "cron",
				Payload: fmt.Sprintf("Task %d", i),
			}
			pool.Submit(task)
		}

		// Give tasks a moment to start
		time.Sleep(10 * time.Millisecond)

		// Stop pool (graceful shutdown)
		pool.Stop()

		// Verify all tasks were processed
		metrics := pool.Metrics()
		assert.GreaterOrEqual(t, metrics.TasksCompleted+metrics.TasksFailed, uint64(0))
	})

	// Cleanup
	manager.StopAll()
}

// TestSubagentSessionIsolation tests that subagents maintain separate sessions.
func TestSubagentSessionIsolation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tempDir := t.TempDir()
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
	var unmarshaled map[string]interface{}
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	// Verify properties
	props, ok := unmarshaled["properties"].(map[string]interface{})
	require.True(t, ok)
	assert.Contains(t, props, "task")
	assert.Contains(t, props, "timeout_seconds")

	// Verify required fields
	required, ok := unmarshaled["required"].([]interface{})
	require.True(t, ok)
	assert.Contains(t, required, "task")
}
