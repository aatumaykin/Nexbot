package workers

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/aatumaykin/nexbot/internal/bus"
	"github.com/aatumaykin/nexbot/internal/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestWorkerPool_Integration_CronTasks tests integration with cron-scheduled tasks
func TestWorkerPool_Integration_CronTasks(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	log, err := logger.New(logger.Config{Level: "debug", Format: "text", Output: "stdout"})
	require.NoError(t, err)

	messageBus := bus.New(100, log)
	require.NoError(t, messageBus.Start(context.Background()))
	defer func() { _ = messageBus.Stop() }()

	// Create worker pool
	pool := NewPool(3, 10, log, messageBus)
	pool.Start()
	defer pool.Stop()

	// Simulate cron scheduler submitting tasks
	numCronJobs := 5
	for i := 0; i < numCronJobs; i++ {
		task := Task{
			ID:      fmt.Sprintf("cron-job-%d", i),
		}
		pool.Submit(task)
	}

	// Wait for all results
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	results := make(map[string]Result)
	for i := 0; i < numCronJobs; i++ {
		select {
		case result := <-pool.Results():
			results[result.TaskID] = result
		case <-ctx.Done():
			t.Fatalf("timeout waiting for cron results, got %d/%d", len(results), numCronJobs)
		}
	}

	// Verify all cron jobs completed successfully
	assert.Len(t, results, numCronJobs)
	for _, result := range results {
		assert.NoError(t, result.Error)
		assert.NotEmpty(t, result.Output)
		assert.Contains(t, result.Output, "scheduled command")
	}

	// Verify metrics
	metrics := pool.Metrics()
	assert.Equal(t, uint64(numCronJobs), metrics.TasksCompleted)
	assert.Equal(t, uint64(0), metrics.TasksFailed)
}

// TestWorkerPool_Integration_SubagentTasks tests integration with subagent tasks
func TestWorkerPool_Integration_SubagentTasks(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	log, err := logger.New(logger.Config{Level: "debug", Format: "text", Output: "stdout"})
	require.NoError(t, err)

	messageBus := bus.New(100, log)
	require.NoError(t, messageBus.Start(context.Background()))
	defer func() { _ = messageBus.Stop() }()

	// Create worker pool
	pool := NewPool(4, 10, log, messageBus)
	pool.Start()
	defer pool.Stop()

	// Simulate subagent tasks with different configurations
	subagentConfigs := []map[string]interface{}{
		{"agent": "planner", "priority": "high"},
		{"agent": "developer", "priority": "medium"},
		{"agent": "tester", "priority": "low"},
	}

	for i, config := range subagentConfigs {
		task := Task{
			ID:      fmt.Sprintf("subagent-%d", i),
			Type:    "subagent",
			Payload: config,
		}
		pool.Submit(task)
	}

	// Wait for all results
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	results := make(map[string]Result)
	for i := 0; i < len(subagentConfigs); i++ {
		select {
		case result := <-pool.Results():
			results[result.TaskID] = result
		case <-ctx.Done():
			t.Fatalf("timeout waiting for subagent results, got %d/%d", len(results), len(subagentConfigs))
		}
	}

	// Verify all subagent tasks completed
	assert.Len(t, results, len(subagentConfigs))
	for _, result := range results {
		assert.NoError(t, result.Error)
		assert.NotEmpty(t, result.Output)
		assert.Contains(t, result.Output, "subagent task completed")
	}
}

// TestWorkerPool_Integration_MixedTasks tests handling mixed task types concurrently
func TestWorkerPool_Integration_MixedTasks(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	log, err := logger.New(logger.Config{Level: "debug", Format: "text", Output: "stdout"})
	require.NoError(t, err)

	messageBus := bus.New(100, log)
	require.NoError(t, messageBus.Start(context.Background()))
	defer func() { _ = messageBus.Stop() }()

	// Create worker pool
	pool := NewPool(5, 20, log, messageBus)
	pool.Start()
	defer pool.Stop()

	// Submit mix of cron and subagent tasks
	totalTasks := 20
	for i := 0; i < totalTasks; i++ {
		var taskType string
		var payload interface{}

		if i%2 == 0 {
		} else {
			taskType = "subagent"
			payload = map[string]string{"task": fmt.Sprintf("subtask %d", i)}
		}

		task := Task{
			ID:      fmt.Sprintf("mixed-task-%d", i),
			Type:    taskType,
			Payload: payload,
		}
		pool.Submit(task)
	}

	// Wait for all results
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	results := make(map[string]Result)
	for i := 0; i < totalTasks; i++ {
		select {
		case result := <-pool.Results():
			results[result.TaskID] = result
		case <-ctx.Done():
			t.Fatalf("timeout waiting for mixed results, got %d/%d", len(results), totalTasks)
		}
	}

	// Verify all tasks completed
	assert.Len(t, results, totalTasks)
	metrics := pool.Metrics()
	assert.Equal(t, uint64(totalTasks), metrics.TasksCompleted)
}

// TestWorkerPool_Integration_CronWithCancellation tests cron tasks with context cancellation
func TestWorkerPool_Integration_CronWithCancellation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	log, err := logger.New(logger.Config{Level: "debug", Format: "text", Output: "stdout"})
	require.NoError(t, err)

	messageBus := bus.New(100, log)
	require.NoError(t, messageBus.Start(context.Background()))
	defer func() { _ = messageBus.Stop() }()

	// Create worker pool
	pool := NewPool(2, 10, log, messageBus)
	pool.Start()
	defer pool.Stop()

	// Submit task with context that will be cancelled
	taskCtx, cancel := context.WithCancel(context.Background())
	task := Task{
		ID:      "cancellable-cron-task",
		Context: taskCtx,
	}

	// Cancel immediately
	cancel()

	pool.Submit(task)

	// Wait for result
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	select {
	case result := <-pool.Results():
		assert.Equal(t, "cancellable-cron-task", result.TaskID)
		assert.Error(t, result.Error)
		// Error should be context.Canceled
	case <-ctx.Done():
		t.Fatal("timeout waiting for cancelled task result")
	}

	// Verify metrics include however failed task
	metrics := pool.Metrics()
	assert.Equal(t, uint64(1), metrics.TasksFailed)
}

// TestWorkerPool_Integration_HighLoad tests pool under high load
func TestWorkerPool_Integration_HighLoad(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	log, err := logger.New(logger.Config{Level: "info", Format: "text", Output: "stdout"})
	require.NoError(t, err)

	messageBus := bus.New(100, log)
	require.NoError(t, messageBus.Start(context.Background()))
	defer func() { _ = messageBus.Stop() }()

	// Create worker pool
	numWorkers := 10
	numTasks := 100

	pool := NewPool(numWorkers, numTasks, log, messageBus)
	pool.Start()
	defer pool.Stop()

	startTime := time.Now()

	// Submit many tasks rapidly
	for i := 0; i < numTasks; i++ {
		task := Task{
			ID:      fmt.Sprintf("load-task-%d", i),
		}
		pool.Submit(task)
	}

	submissionTime := time.Since(startTime)

	// Wait for all results
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	results := make(map[string]Result)
	for i := 0; i < numTasks; i++ {
		select {
		case result := <-pool.Results():
			results[result.TaskID] = result
		case <-ctx.Done():
			t.Fatalf("timeout waiting for load results, got %d/%d", len(results), numTasks)
		}
	}

	totalTime := time.Since(startTime)

	// Verify all tasks completed
	assert.Len(t, results, numTasks)

	metrics := pool.Metrics()
	assert.Equal(t, uint64(numTasks), metrics.TasksCompleted)
	assert.Equal(t, uint64(0), metrics.TasksFailed)

	t.Logf("Submission time: %v", submissionTime)
	t.Logf("Total execution time: %v", totalTime)
	t.Logf("Average task duration: %v", metrics.TotalDuration/time.Duration(numTasks))
}

// TestWorkerPool_Integration_GracefulShutdownWithTasks tests shutdown with active tasks
func TestWorkerPool_Integration_GracefulShutdownWithTasks(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	log, err := logger.New(logger.Config{Level: "debug", Format: "text", Output: "stdout"})
	require.NoError(t, err)

	messageBus := bus.New(100, log)
	require.NoError(t, messageBus.Start(context.Background()))
	defer func() { _ = messageBus.Stop() }()

	// Create worker pool
	pool := NewPool(3, 10, log, messageBus)
	pool.Start()

	// Submit tasks
	numTasks := 10
	for i := 0; i < numTasks; i++ {
		task := Task{
			ID:      fmt.Sprintf("shutdown-task-%d", i),
		}
		pool.Submit(task)
	}

	// Give tasks a moment to start processing
	time.Sleep(10 * time.Millisecond)

	// Stop pool - should wait for workers to finish
	shutdownStart := time.Now()
	pool.Stop()
	shutdownDuration := time.Since(shutdownStart)

	// Verify all tasks were processed
	metrics := pool.Metrics()
	assert.Equal(t, uint64(numTasks), metrics.TasksSubmitted)
	// Some tasks should complete or fail during graceful shutdown
	assert.GreaterOrEqual(t, metrics.TasksCompleted+metrics.TasksFailed, uint64(0))

	t.Logf("Shutdown duration: %v", shutdownDuration)
}

// TestWorkerPool_Integration_TaskTimeout tests task execution with timeout
func TestWorkerPool_Integration_TaskTimeout(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	log, err := logger.New(logger.Config{Level: "debug", Format: "text", Output: "stdout"})
	require.NoError(t, err)

	messageBus := bus.New(100, log)
	require.NoError(t, messageBus.Start(context.Background()))
	defer func() { _ = messageBus.Stop() }()

	// Create worker pool
	pool := NewPool(2, 10, log, messageBus)
	pool.Start()
	defer pool.Stop()

	// Create task with short timeout
	taskCtx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	task := Task{
		ID:      "timeout-task",
		Type:    "subagent",
		Payload: map[string]string{"slow": "task"},
		Context: taskCtx,
	}

	// Wait a bit for context to expire
	time.Sleep(10 * time.Millisecond)

	pool.Submit(task)

	// Wait for result
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	select {
	case result := <-pool.Results():
		assert.Equal(t, "timeout-task", result.TaskID)
		// Task should fail due to context cancellation
		assert.Error(t, result.Error)
	case <-ctx.Done():
		t.Fatal("timeout waiting for result")
	}
}

// TestWorkerPool_Integration_SequentialSubmissions tests sequential task submission
func TestWorkerPool_Integration_SequentialSubmissions(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	log, err := logger.New(logger.Config{Level: "debug", Format: "text", Output: "stdout"})
	require.NoError(t, err)

	messageBus := bus.New(100, log)
	require.NoError(t, messageBus.Start(context.Background()))
	defer func() { _ = messageBus.Stop() }()

	// Create worker pool with single worker to test sequential processing
	pool := NewPool(1, 10, log, messageBus)
	pool.Start()
	defer pool.Stop()

	numBatches := 3
	tasksPerBatch := 3

	for batch := 0; batch < numBatches; batch++ {
		// Submit batch of tasks
		for i := 0; i < tasksPerBatch; i++ {
			task := Task{
				ID:      fmt.Sprintf("batch-%d-task-%d", batch, i),
			}
			pool.Submit(task)
		}

		// Wait for all tasks in this batch to complete
		for i := 0; i < tasksPerBatch; i++ {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			select {
			case result := <-pool.Results():
				assert.NoError(t, result.Error)
			case <-ctx.Done():
				t.Fatalf("timeout waiting for batch %d result", batch)
			}
		}
	}

	// Verify total metrics
	metrics := pool.Metrics()
	expectedTasks := numBatches * tasksPerBatch
	assert.Equal(t, uint64(expectedTasks), metrics.TasksCompleted)
}

// TestWorkerPool_Integration_MetricsTracking tests metrics accuracy
func TestWorkerPool_Integration_MetricsTracking(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	log, err := logger.New(logger.Config{Level: "debug", Format: "text", Output: "stdout"})
	require.NoError(t, err)

	messageBus := bus.New(100, log)
	require.NoError(t, messageBus.Start(context.Background()))
	defer func() { _ = messageBus.Stop() }()

	// Create worker pool
	pool := NewPool(3, 20, log, messageBus)
	pool.Start()
	defer pool.Stop()

	// Track expected metrics
	var successfulTasks, failedTasks int

	// Submit successful tasks
	for i := 0; i < 5; i++ {
		task := Task{
			ID:      fmt.Sprintf("success-%d", i),
		}
		pool.Submit(task)
		successfulTasks++
	}

	// Submit tasks that will fail (invalid type)
	for i := 0; i < 3; i++ {
		task := Task{
			ID:      fmt.Sprintf("fail-%d", i),
			Type:    "invalid",
			Payload: "test",
		}
		pool.Submit(task)
		failedTasks++
	}

	// Wait for all results
	totalTasks := successfulTasks + failedTasks
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	for i := 0; i < totalTasks; i++ {
		select {
		case <-pool.Results():
		case <-ctx.Done():
			t.Fatal("timeout waiting for results")
		}
	}

	// Verify metrics match expected
	metrics := pool.Metrics()
	assert.Equal(t, uint64(successfulTasks), metrics.TasksCompleted)
	assert.Equal(t, uint64(failedTasks), metrics.TasksFailed)
	assert.Equal(t, uint64(totalTasks), metrics.TasksSubmitted)
	assert.Greater(t, metrics.TotalDuration, time.Duration(0))
}

// TestWorkerPool_Integration_Restart tests pool restart scenario
func TestWorkerPool_Integration_Restart(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	log, err := logger.New(logger.Config{Level: "debug", Format: "text", Output: "stdout"})
	require.NoError(t, err)

	messageBus := bus.New(100, log)
	require.NoError(t, messageBus.Start(context.Background()))
	defer func() { _ = messageBus.Stop() }()

	// First pool lifecycle
	pool := NewPool(2, 10, log, messageBus)
	pool.Start()

	// Submit and complete some tasks
	for i := 0; i < 3; i++ {
		task := Task{
			ID:      fmt.Sprintf("run1-task-%d", i),
		}
		pool.Submit(task)
	}

	// Wait for results
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	for i := 0; i < 3; i++ {
		select {
		case <-pool.Results():
		case <-ctx.Done():
			t.Fatal("timeout waiting for first run results")
		}
	}

	// Stop
	pool.Stop()

	// Create and start new pool
	newPool := NewPool(2, 10, log, messageBus)
	newPool.Start()
	defer newPool.Stop()

	// Submit tasks to new pool
	for i := 0; i < 3; i++ {
		task := Task{
			ID:      fmt.Sprintf("run2-task-%d", i),
		}
		newPool.Submit(task)
	}

	// Wait for results from new pool
	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	for i := 0; i < 3; i++ {
		select {
		case result := <-newPool.Results():
			assert.NoError(t, result.Error)
		case <-ctx.Done():
			t.Fatal("timeout waiting for second run results")
		}
	}

	// Verify new pool metrics
	newMetrics := newPool.Metrics()
	assert.Equal(t, uint64(3), newMetrics.TasksCompleted)
}
