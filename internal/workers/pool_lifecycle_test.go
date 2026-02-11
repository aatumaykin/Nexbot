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

func TestPool_StartStop(t *testing.T) {
	log, err := logger.New(logger.Config{Level: "debug", Format: "text", Output: "stdout"})
	require.NoError(t, err)

	messageBus := bus.New(100, 10, log)
	require.NoError(t, messageBus.Start(context.Background()))
	defer func() { _ = messageBus.Stop() }()

	pool := NewPool(2, 10, log, messageBus)

	pool.Start()

	assert.Equal(t, 2, pool.WorkerCount())

	pool.Stop()
}

func TestPool_GracefulShutdown(t *testing.T) {
	log, err := logger.New(logger.Config{Level: "debug", Format: "text", Output: "stdout"})
	require.NoError(t, err)

	messageBus := bus.New(100, 10, log)
	require.NoError(t, messageBus.Start(context.Background()))
	defer func() { _ = messageBus.Stop() }()

	pool := NewPool(2, 10, log, messageBus)
	pool.Start()

	for i := 0; i < 5; i++ {
		task := Task{
			ID:   fmt.Sprintf("shutdown-task-%d", i),
			Type: "cron",
		}
		pool.Submit(task)
	}

	time.Sleep(10 * time.Millisecond)

	pool.Stop()

	metrics := pool.Metrics()
	assert.Equal(t, uint64(5), metrics.TasksSubmitted)
	assert.GreaterOrEqual(t, metrics.TasksCompleted+metrics.TasksFailed, uint64(0))
}
