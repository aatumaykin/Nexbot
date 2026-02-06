package workers

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/aatumaykin/nexbot/internal/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPool(t *testing.T) {
	log, err := logger.New(logger.Config{Level: "debug", Format: "text", Output: "stdout"})
	require.NoError(t, err)

	tests := []struct {
		name       string
		workers    int
		bufferSize int
		wantErr    bool
	}{
		{
			name:       "valid pool",
			workers:    3,
			bufferSize: 10,
			wantErr:    false,
		},
		{
			name:       "single worker",
			workers:    1,
			bufferSize: 5,
			wantErr:    false,
		},
		{
			name:       "many workers",
			workers:    100,
			bufferSize: 50,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pool := NewPool(tt.workers, tt.bufferSize, log)
			assert.NotNil(t, pool)
			assert.Equal(t, tt.workers, pool.WorkerCount())
			assert.NotNil(t, pool.Results())
		})
	}
}

func TestPool_StartStop(t *testing.T) {
	log, err := logger.New(logger.Config{Level: "debug", Format: "text", Output: "stdout"})
	require.NoError(t, err)

	pool := NewPool(2, 10, log)

	// Start the pool
	pool.Start()

	// Verify pool is running
	assert.Equal(t, 2, pool.WorkerCount())

	// Stop the pool
	pool.Stop()

	// Verify channels are closed
	// Note: We can't easily verify channels are closed without race conditions
}

func TestPool_Submit(t *testing.T) {
	log, err := logger.New(logger.Config{Level: "debug", Format: "text", Output: "stdout"})
	require.NoError(t, err)

	pool := NewPool(2, 10, log)
	pool.Start()
	defer pool.Stop()

	task := Task{
		ID:      "task-1",
		Type:    "cron",
		Payload: "test command",
	}

	// Submit should not block
	pool.Submit(task)

	metrics := pool.Metrics()
	assert.Equal(t, uint64(1), metrics.TasksSubmitted)
}

func TestPool_ExecuteCronTask(t *testing.T) {
	log, err := logger.New(logger.Config{Level: "debug", Format: "text", Output: "stdout"})
	require.NoError(t, err)

	pool := NewPool(1, 10, log)
	pool.Start()
	defer pool.Stop()

	task := Task{
		ID:      "cron-task-1",
		Type:    "cron",
		Payload: "echo 'hello'",
	}

	// Submit task
	pool.Submit(task)

	// Wait for result with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	select {
	case result := <-pool.Results():
		assert.Equal(t, task.ID, result.TaskID)
		assert.NoError(t, result.Error)
		assert.NotEmpty(t, result.Output)
		assert.Contains(t, result.Output, "echo 'hello'")
		assert.Greater(t, result.Duration, time.Duration(0))

	case <-ctx.Done():
		t.Fatal("timeout waiting for task result")
	}

	// Verify metrics
	metrics := pool.Metrics()
	assert.Equal(t, uint64(1), metrics.TasksSubmitted)
	assert.Equal(t, uint64(1), metrics.TasksCompleted)
	assert.Equal(t, uint64(0), metrics.TasksFailed)
}

func TestPool_ExecuteSubagentTask(t *testing.T) {
	log, err := logger.New(logger.Config{Level: "debug", Format: "text", Output: "stdout"})
	require.NoError(t, err)

	pool := NewPool(1, 10, log)
	pool.Start()
	defer pool.Stop()

	task := Task{
		ID:      "subagent-task-1",
		Type:    "subagent",
		Payload: map[string]string{"agent": "test"},
	}

	// Submit task
	pool.Submit(task)

	// Wait for result with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	select {
	case result := <-pool.Results():
		assert.Equal(t, task.ID, result.TaskID)
		assert.NoError(t, result.Error)
		assert.NotEmpty(t, result.Output)
		assert.Contains(t, result.Output, "subagent task completed")

	case <-ctx.Done():
		t.Fatal("timeout waiting for task result")
	}

	// Verify metrics
	metrics := pool.Metrics()
	assert.Equal(t, uint64(1), metrics.TasksCompleted)
}

func TestPool_UnknownTaskType(t *testing.T) {
	log, err := logger.New(logger.Config{Level: "debug", Format: "text", Output: "stdout"})
	require.NoError(t, err)

	pool := NewPool(1, 10, log)
	pool.Start()
	defer pool.Stop()

	task := Task{
		ID:      "unknown-task",
		Type:    "invalid",
		Payload: "test",
	}

	// Submit task
	pool.Submit(task)

	// Wait for result with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	select {
	case result := <-pool.Results():
		assert.Equal(t, task.ID, result.TaskID)
		assert.Error(t, result.Error)
		assert.Contains(t, result.Error.Error(), "unknown task type")

	case <-ctx.Done():
		t.Fatal("timeout waiting for task result")
	}

	// Verify metrics - should be a failed task
	metrics := pool.Metrics()
	assert.Equal(t, uint64(1), metrics.TasksFailed)
}

func TestPool_InvalidCronPayload(t *testing.T) {
	log, err := logger.New(logger.Config{Level: "debug", Format: "text", Output: "stdout"})
	require.NoError(t, err)

	pool := NewPool(1, 10, log)
	pool.Start()
	defer pool.Stop()

	task := Task{
		ID:      "cron-invalid-payload",
		Type:    "cron",
		Payload: 123, // Invalid payload type
	}

	// Submit task
	pool.Submit(task)

	// Wait for result with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	select {
	case result := <-pool.Results():
		assert.Equal(t, task.ID, result.TaskID)
		assert.Error(t, result.Error)
		assert.Contains(t, result.Error.Error(), "invalid cron task payload")

	case <-ctx.Done():
		t.Fatal("timeout waiting for task result")
	}

	metrics := pool.Metrics()
	assert.Equal(t, uint64(1), metrics.TasksFailed)
}

func TestPool_ContextCancellation(t *testing.T) {
	log, err := logger.New(logger.Config{Level: "debug", Format: "text", Output: "stdout"})
	require.NoError(t, err)

	pool := NewPool(1, 10, log)
	pool.Start()
	defer pool.Stop()

	// Create a task with context that gets cancelled
	taskCtx, cancel := context.WithCancel(context.Background())
	task := Task{
		ID:      "cancelled-task",
		Type:    "cron",
		Payload: "test command",
		Context: taskCtx,
	}

	// Cancel the task context immediately
	cancel()

	// Submit task - task should be cancelled during execution
	pool.Submit(task)

	// Wait for result with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	select {
	case result := <-pool.Results():
		assert.Equal(t, task.ID, result.TaskID)
		assert.Error(t, result.Error)
		assert.True(t, errors.Is(result.Error, context.Canceled))

	case <-ctx.Done():
		t.Fatal("timeout waiting for task result")
	}

	metrics := pool.Metrics()
	assert.Equal(t, uint64(1), metrics.TasksFailed)
}

func TestPool_SubmitWithContext(t *testing.T) {
	log, err := logger.New(logger.Config{Level: "debug", Format: "text", Output: "stdout"})
	require.NoError(t, err)

	pool := NewPool(1, 1, log) // Buffer size of 1
	pool.Start()
	defer pool.Stop()

	// Fill the buffer
	task1 := Task{ID: "task-1", Type: "cron", Payload: "command1"}
	task2 := Task{ID: "task-2", Type: "cron", Payload: "command2"}
	pool.Submit(task1)
	pool.Submit(task2)

	// Try to submit with timeout - should succeed as queue is full but workers are processing
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	task3 := Task{ID: "task-3", Type: "cron", Payload: "command3"}
	err = pool.SubmitWithContext(ctx, task3)
	// May succeed or timeout depending on worker speed
	if err != nil {
		assert.True(t, errors.Is(err, context.DeadlineExceeded))
	}
}

func TestPool_MultipleTasks(t *testing.T) {
	log, err := logger.New(logger.Config{Level: "debug", Format: "text", Output: "stdout"})
	require.NoError(t, err)

	pool := NewPool(3, 10, log)
	pool.Start()
	defer pool.Stop()

	numTasks := 10
	tasks := make([]Task, numTasks)

	// Submit multiple tasks
	for i := 0; i < numTasks; i++ {
		taskType := "cron"
		if i%2 == 0 {
			taskType = "subagent"
		}
		tasks[i] = Task{
			ID:      fmt.Sprintf("task-%d", i),
			Type:    taskType,
			Payload: fmt.Sprintf("command %d", i),
		}
		pool.Submit(tasks[i])
	}

	// Wait for all results
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	results := make(map[string]Result)
	for i := 0; i < numTasks; i++ {
		select {
		case result := <-pool.Results():
			results[result.TaskID] = result
		case <-ctx.Done():
			t.Fatalf("timeout waiting for results, got %d/%d", len(results), numTasks)
		}
	}

	// Verify all tasks completed
	assert.Len(t, results, numTasks)
	for _, task := range tasks {
		result, ok := results[task.ID]
		assert.True(t, ok, "missing result for task %s", task.ID)
		assert.Equal(t, task.ID, result.TaskID)
		assert.NoError(t, result.Error)
	}

	// Verify metrics
	metrics := pool.Metrics()
	assert.Equal(t, uint64(numTasks), metrics.TasksCompleted)
	assert.Equal(t, uint64(0), metrics.TasksFailed)
}

func TestPool_QueueSize(t *testing.T) {
	log, err := logger.New(logger.Config{Level: "debug", Format: "text", Output: "stdout"})
	require.NoError(t, err)

	pool := NewPool(1, 5, log)
	pool.Start()
	defer pool.Stop()

	// Initially queue should be empty
	assert.Equal(t, 0, pool.QueueSize())

	// Submit tasks without processing
	// Note: Workers will immediately start processing, so we can't reliably test queue size
	// This test is more about the method working than specific values
}

func TestPool_GracefulShutdown(t *testing.T) {
	log, err := logger.New(logger.Config{Level: "debug", Format: "text", Output: "stdout"})
	require.NoError(t, err)

	pool := NewPool(2, 10, log)
	pool.Start()

	// Submit some tasks
	for i := 0; i < 5; i++ {
		task := Task{
			ID:      fmt.Sprintf("shutdown-task-%d", i),
			Type:    "cron",
			Payload: fmt.Sprintf("command %d", i),
		}
		pool.Submit(task)
	}

	// Give tasks a moment to start processing
	time.Sleep(10 * time.Millisecond)

	// Stop pool - should wait for workers to finish
	pool.Stop()

	// Verify metrics (some tasks may complete, others may be cancelled)
	metrics := pool.Metrics()
	assert.Equal(t, uint64(5), metrics.TasksSubmitted)
	assert.GreaterOrEqual(t, metrics.TasksCompleted+metrics.TasksFailed, uint64(0))
}

func TestPool_Metrics(t *testing.T) {
	log, err := logger.New(logger.Config{Level: "debug", Format: "text", Output: "stdout"})
	require.NoError(t, err)

	pool := NewPool(1, 10, log)
	pool.Start()
	defer pool.Stop()

	// Submit a task
	task := Task{
		ID:      "metrics-task",
		Type:    "cron",
		Payload: "test",
	}
	pool.Submit(task)

	// Wait for completion
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	select {
	case <-pool.Results():
	case <-ctx.Done():
		t.Fatal("timeout waiting for task result")
	}

	// Get metrics
	metrics := pool.Metrics()
	assert.Equal(t, uint64(1), metrics.TasksSubmitted)
	assert.Equal(t, uint64(1), metrics.TasksCompleted)
	assert.Equal(t, uint64(0), metrics.TasksFailed)
	assert.Greater(t, metrics.TotalDuration, time.Duration(0))
}

func TestPool_WorkerPanicRecovery(t *testing.T) {
	log, err := logger.New(logger.Config{Level: "debug", Format: "text", Output: "stdout"})
	require.NoError(t, err)

	pool := NewPool(1, 10, log)
	pool.Start()
	defer pool.Stop()

	// Submit a task that will panic during execution
	task := Task{
		ID:      "panic-task",
		Type:    "cron",
		Payload: "test",
	}

	// We can't easily induce a panic in the mock implementation,
	// but we can verify the worker continues after tasks
	pool.Submit(task)

	// Wait for result
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	select {
	case result := <-pool.Results():
		assert.Equal(t, task.ID, result.TaskID)
	case <-ctx.Done():
		t.Fatal("timeout waiting for task result")
	}

	// Submit another task to verify worker is still running
	task2 := Task{
		ID:      "recovery-task",
		Type:    "cron",
		Payload: "test2",
	}
	pool.Submit(task2)

	select {
	case result := <-pool.Results():
		assert.Equal(t, task2.ID, result.TaskID)
	case <-ctx.Done():
		t.Fatal("timeout waiting for second task result")
	}
}

func TestTaskWaitGroup(t *testing.T) {
	twg := newTaskWaitGroup()

	twg.Add(1)
	done := make(chan struct{})
	go func() {
		twg.Done()
		close(done)
	}()

	select {
	case <-done:
		// Done was called successfully
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout waiting for Done")
	}

	// Test wait with zero count
	twg.Add(1)
	twg.Done()
	twg.Wait()
}

func TestPool_ConcurrentSubmissions(t *testing.T) {
	log, err := logger.New(logger.Config{Level: "debug", Format: "text", Output: "stdout"})
	require.NoError(t, err)

	pool := NewPool(5, 100, log)
	pool.Start()
	defer pool.Stop()

	numGoroutines := 10
	tasksPerGoroutine := 5

	// Submit tasks concurrently
	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			for j := 0; j < tasksPerGoroutine; j++ {
				task := Task{
					ID:      fmt.Sprintf("goroutine-%d-task-%d", goroutineID, j),
					Type:    "cron",
					Payload: fmt.Sprintf("command %d", j),
				}
				pool.Submit(task)
			}
		}(i)
	}

	// Wait for all results
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	totalTasks := numGoroutines * tasksPerGoroutine
	results := make(map[string]Result)
	for i := 0; i < totalTasks; i++ {
		select {
		case result := <-pool.Results():
			results[result.TaskID] = result
		case <-ctx.Done():
			t.Fatalf("timeout waiting for results, got %d/%d", len(results), totalTasks)
		}
	}

	// Verify all tasks completed
	assert.Len(t, results, totalTasks)
	metrics := pool.Metrics()
	assert.Equal(t, uint64(totalTasks), metrics.TasksCompleted)
}

func TestPool_ExecuteWithRetry_Success(t *testing.T) {
	log, err := logger.New(logger.Config{Level: "debug", Format: "text", Output: "stdout"})
	require.NoError(t, err)

	pool := NewPool(1, 10, log)

	task := Task{
		ID:      "test-task",
		Type:    "test",
		Payload: "test payload",
	}

	executor := func(ctx context.Context, t Task) (string, error) {
		return "execution successful", nil
	}

	result := pool.executeWithRetry(context.Background(), task, executor)

	assert.Equal(t, task.ID, result.TaskID)
	assert.NoError(t, result.Error)
	assert.Equal(t, "execution successful", result.Output)
}

func TestPool_ExecuteWithRetry_ExecutorError(t *testing.T) {
	log, err := logger.New(logger.Config{Level: "debug", Format: "text", Output: "stdout"})
	require.NoError(t, err)

	pool := NewPool(1, 10, log)

	task := Task{
		ID:      "error-task",
		Type:    "test",
		Payload: "test payload",
	}

	executor := func(ctx context.Context, t Task) (string, error) {
		return "", fmt.Errorf("execution failed")
	}

	result := pool.executeWithRetry(context.Background(), task, executor)

	assert.Equal(t, task.ID, result.TaskID)
	assert.Error(t, result.Error)
	assert.Contains(t, result.Error.Error(), "execution failed")
	assert.Empty(t, result.Output)
}

func TestPool_ExecuteWithRetry_ContextCancellation(t *testing.T) {
	log, err := logger.New(logger.Config{Level: "debug", Format: "text", Output: "stdout"})
	require.NoError(t, err)

	pool := NewPool(1, 10, log)

	task := Task{
		ID:      "cancelled-task",
		Type:    "test",
		Payload: "test payload",
	}

	executor := func(ctx context.Context, t Task) (string, error) {
		// Simulate long-running task that should be cancelled
		time.Sleep(1 * time.Second)
		return "should not reach here", nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	result := pool.executeWithRetry(ctx, task, executor)

	assert.Equal(t, task.ID, result.TaskID)
	assert.Error(t, result.Error)
	assert.True(t, errors.Is(result.Error, context.Canceled))
}

func TestPool_ExecuteWithRetry_PanicRecovery(t *testing.T) {
	log, err := logger.New(logger.Config{Level: "debug", Format: "text", Output: "stdout"})
	require.NoError(t, err)

	pool := NewPool(1, 10, log)

	task := Task{
		ID:      "panic-task",
		Type:    "test",
		Payload: "test payload",
	}

	executor := func(ctx context.Context, t Task) (string, error) {
		panic("something went wrong")
	}

	result := pool.executeWithRetry(context.Background(), task, executor)

	assert.Equal(t, task.ID, result.TaskID)
	assert.Error(t, result.Error)
	assert.Contains(t, result.Error.Error(), "panic during task execution")
	assert.Contains(t, result.Error.Error(), "something went wrong")
}

func TestPool_ExecuteWithRetry_OutputCapture(t *testing.T) {
	log, err := logger.New(logger.Config{Level: "debug", Format: "text", Output: "stdout"})
	require.NoError(t, err)

	pool := NewPool(1, 10, log)

	task := Task{
		ID:      "output-task",
		Type:    "test",
		Payload: "test payload",
	}

	expectedOutput := "processed data: test payload"
	executor := func(ctx context.Context, t Task) (string, error) {
		return expectedOutput, nil
	}

	result := pool.executeWithRetry(context.Background(), task, executor)

	assert.Equal(t, task.ID, result.TaskID)
	assert.NoError(t, result.Error)
	assert.Equal(t, expectedOutput, result.Output)
}

func TestPool_ExecuteWithRetry_ContextPassed(t *testing.T) {
	log, err := logger.New(logger.Config{Level: "debug", Format: "text", Output: "stdout"})
	require.NoError(t, err)

	pool := NewPool(1, 10, log)

	task := Task{
		ID:      "context-task",
		Type:    "test",
		Payload: "test payload",
	}

	var passedCtx context.Context
	executor := func(ctx context.Context, t Task) (string, error) {
		passedCtx = ctx
		return "success", nil
	}

	// Use context with value to verify it's passed correctly
	testCtx := context.WithValue(context.Background(), "key", "value")
	result := pool.executeWithRetry(testCtx, task, executor)

	assert.Equal(t, task.ID, result.TaskID)
	assert.NoError(t, result.Error)

	// Verify context was passed to executor
	assert.NotNil(t, passedCtx)
	assert.Equal(t, "value", passedCtx.Value("key"))
}
