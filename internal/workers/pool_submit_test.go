package workers

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/aatumaykin/nexbot/internal/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

	pool.Submit(task)

	metrics := pool.Metrics()
	assert.Equal(t, uint64(1), metrics.TasksSubmitted)
}

func TestPool_SubmitWithContext(t *testing.T) {
	log, err := logger.New(logger.Config{Level: "debug", Format: "text", Output: "stdout"})
	require.NoError(t, err)

	pool := NewPool(1, 1, log)
	pool.Start()
	defer pool.Stop()

	task1 := Task{ID: "task-1", Type: "cron", Payload: "command1"}
	task2 := Task{ID: "task-2", Type: "cron", Payload: "command2"}
	pool.Submit(task1)
	pool.Submit(task2)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	task3 := Task{ID: "task-3", Type: "cron", Payload: "command3"}
	err = pool.SubmitWithContext(ctx, task3)
	if err != nil {
		assert.True(t, errors.Is(err, context.DeadlineExceeded))
	}
}
