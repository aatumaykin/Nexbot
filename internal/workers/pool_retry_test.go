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

// Custom type for context key to avoid collisions
type contextKey string

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
		time.Sleep(1 * time.Second)
		return "should not reach here", nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

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

	testCtx := context.WithValue(context.Background(), contextKey("testKey"), "value")
	result := pool.executeWithRetry(testCtx, task, executor)

	assert.Equal(t, task.ID, result.TaskID)
	assert.NoError(t, result.Error)

	assert.NotNil(t, passedCtx)
	assert.Equal(t, "value", passedCtx.Value(contextKey("testKey")))
}
