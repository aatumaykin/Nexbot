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

func TestPool_ExecuteCronTask(t *testing.T) {
	log, err := logger.New(logger.Config{Level: "debug", Format: "text", Output: "stdout"})
	require.NoError(t, err)

	messageBus := bus.New(100, log)
	require.NoError(t, messageBus.Start(context.Background()))
	defer func() { _ = messageBus.Stop() }()

	pool := NewPool(1, 10, log, messageBus)
	pool.Start()
	defer pool.Stop()

	task := Task{
		ID:      "cron-task-1",
		Type:    "cron",
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

	messageBus := bus.New(100, log)
	require.NoError(t, messageBus.Start(context.Background()))
	defer func() { _ = messageBus.Stop() }()

	pool := NewPool(1, 10, log, messageBus)
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

	messageBus := bus.New(100, log)
	require.NoError(t, messageBus.Start(context.Background()))
	defer func() { _ = messageBus.Stop() }()

	pool := NewPool(1, 10, log, messageBus)
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

	messageBus := bus.New(100, log)
	require.NoError(t, messageBus.Start(context.Background()))
	defer func() { _ = messageBus.Stop() }()

	pool := NewPool(1, 10, log, messageBus)
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

func TestPool_MultipleTasks(t *testing.T) {
	log, err := logger.New(logger.Config{Level: "debug", Format: "text", Output: "stdout"})
	require.NoError(t, err)

	messageBus := bus.New(100, log)
	require.NoError(t, messageBus.Start(context.Background()))
	defer func() { _ = messageBus.Stop() }()

	pool := NewPool(3, 10, log, messageBus)
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
		payload := interface{}(fmt.Sprintf("command %d", i))
		if taskType == "cron" {
		}
		tasks[i] = Task{
			ID:      fmt.Sprintf("task-%d", i),
			Type:    taskType,
			Payload: payload,
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
