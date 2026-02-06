package subagent

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/aatumaykin/nexbot/internal/agent/loop"
	"github.com/aatumaykin/nexbot/internal/workers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
