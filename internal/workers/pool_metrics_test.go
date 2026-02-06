package workers

import (
	"context"
	"testing"
	"time"

	"github.com/aatumaykin/nexbot/internal/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPool_QueueSize(t *testing.T) {
	log, err := logger.New(logger.Config{Level: "debug", Format: "text", Output: "stdout"})
	require.NoError(t, err)

	pool := NewPool(1, 5, log)
	pool.Start()
	defer pool.Stop()

	assert.Equal(t, 0, pool.QueueSize())
}

func TestPool_Metrics(t *testing.T) {
	log, err := logger.New(logger.Config{Level: "debug", Format: "text", Output: "stdout"})
	require.NoError(t, err)

	pool := NewPool(1, 10, log)
	pool.Start()
	defer pool.Stop()

	task := Task{
		ID:      "metrics-task",
		Type:    "cron",
		Payload: "test",
	}
	pool.Submit(task)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	select {
	case <-pool.Results():
	case <-ctx.Done():
		t.Fatal("timeout waiting for task result")
	}

	metrics := pool.Metrics()
	assert.Equal(t, uint64(1), metrics.TasksSubmitted)
	assert.Equal(t, uint64(1), metrics.TasksCompleted)
	assert.Equal(t, uint64(0), metrics.TasksFailed)
	assert.Greater(t, metrics.TotalDuration, time.Duration(0))
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
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout waiting for Done")
	}

	twg.Add(1)
	twg.Done()
	twg.Wait()
}
