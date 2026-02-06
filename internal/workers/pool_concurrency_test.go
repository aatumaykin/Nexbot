package workers

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/aatumaykin/nexbot/internal/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPool_WorkerPanicRecovery(t *testing.T) {
	log, err := logger.New(logger.Config{Level: "debug", Format: "text", Output: "stdout"})
	require.NoError(t, err)

	pool := NewPool(1, 10, log)
	pool.Start()
	defer pool.Stop()

	task := Task{
		ID:      "panic-task",
		Type:    "cron",
		Payload: "test",
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

func TestPool_ConcurrentSubmissions(t *testing.T) {
	log, err := logger.New(logger.Config{Level: "debug", Format: "text", Output: "stdout"})
	require.NoError(t, err)

	pool := NewPool(5, 100, log)
	pool.Start()
	defer pool.Stop()

	numGoroutines := 10
	tasksPerGoroutine := 5

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

	assert.Len(t, results, totalTasks)
	metrics := pool.Metrics()
	assert.Equal(t, uint64(totalTasks), metrics.TasksCompleted)
}
