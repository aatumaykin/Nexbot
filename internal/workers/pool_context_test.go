package workers

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/aatumaykin/nexbot/internal/bus"
	"github.com/aatumaykin/nexbot/internal/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPool_ContextCancellation(t *testing.T) {
	log, err := logger.New(logger.Config{Level: "debug", Format: "text", Output: "stdout"})
	require.NoError(t, err)

	messageBus := bus.New(100, log)
	require.NoError(t, messageBus.Start(context.Background()))
	defer func() { _ = messageBus.Stop() }()

	pool := NewPool(1, 10, log, messageBus)
	pool.Start()
	defer pool.Stop()

	// Create a task with context that gets cancelled
	taskCtx, cancel := context.WithCancel(context.Background())
	task := Task{
		ID:      "cancelled-task",
		Type:    "cron",
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
