package workers

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/aatumaykin/nexbot/internal/bus"
	"github.com/aatumaykin/nexbot/internal/cron"
	"github.com/aatumaykin/nexbot/internal/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPool_WorkerPanicRecovery(t *testing.T) {
	log, err := logger.New(logger.Config{Level: "debug", Format: "text", Output: "stdout"})
	require.NoError(t, err)

	messageBus := bus.New(100, 10, log)
	require.NoError(t, messageBus.Start(context.Background()))
	defer func() { _ = messageBus.Stop() }()

	pool := NewPool(1, 10, log, messageBus)
	pool.Start()
	defer pool.Stop()

	task := Task{
		ID:   "panic-task",
		Type: "cron",
	}

	pool.Submit(task)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	select {
	case result := <-pool.Results():
		assert.Equal(t, task.ID, result.TaskID)
	case <-ctx.Done():
		t.Fatal("timeout waiting for task result")
	}

	task2 := Task{
		ID:   "recovery-task",
		Type: "cron",
	}
	pool.Submit(task2)

	select {
	case result := <-pool.Results():
		assert.Equal(t, task2.ID, result.TaskID)
	case <-ctx.Done():
		t.Fatal("timeout waiting for second task result")
	}
}

func TestPool_ConcurrentSubmissions(t *testing.T) {
	log, err := logger.New(logger.Config{Level: "debug", Format: "text", Output: "stdout"})
	require.NoError(t, err)

	messageBus := bus.New(100, 10, log)
	require.NoError(t, messageBus.Start(context.Background()))
	defer func() { _ = messageBus.Stop() }()

	pool := NewPool(5, 100, log, messageBus)
	pool.Start()
	defer pool.Stop()

	numGoroutines := 10
	tasksPerGoroutine := 5

	for i := range numGoroutines {
		go func(goroutineID int) {
			for j := range tasksPerGoroutine {
				task := Task{
					ID:   fmt.Sprintf("goroutine-%d-task-%d", goroutineID, j),
					Type: "cron",
					Payload: cron.CronTaskPayload{
						Tool:      "send_message",
						Payload:   map[string]any{"message": fmt.Sprintf("message %d", j)},
						SessionID: fmt.Sprintf("telegram:goroutine-%d-task-%d", goroutineID, j),
					},
				}
				pool.Submit(task)
			}
		}(i)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	totalTasks := numGoroutines * tasksPerGoroutine
	results := make(map[string]Result)
	for range totalTasks {
		select {
		case result := <-pool.Results():
			results[result.TaskID] = result
		case <-ctx.Done():
			t.Fatalf("timeout waiting for results, got %d/%d", len(results), totalTasks)
		}
	}

	assert.Len(t, results, totalTasks)
	metrics := pool.Metrics()
	assert.Equal(t, uint64(totalTasks), metrics.TasksCompleted)
}
