package workers

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/aatumaykin/nexbot/internal/bus"
	"github.com/aatumaykin/nexbot/internal/cron"
	"github.com/aatumaykin/nexbot/internal/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Custom type for context key to avoid collisions
type contextKey string

func TestPool_ExecuteWithRetry_Success(t *testing.T) {
	log, err := logger.New(logger.Config{Level: "debug", Format: "text", Output: "stdout"})
	require.NoError(t, err)

	messageBus := bus.New(100, log)
	require.NoError(t, messageBus.Start(context.Background()))
	defer func() { _ = messageBus.Stop() }()

	pool := NewPool(1, 10, log, messageBus)

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

	messageBus := bus.New(100, log)
	require.NoError(t, messageBus.Start(context.Background()))
	defer func() { _ = messageBus.Stop() }()

	pool := NewPool(1, 10, log, messageBus)

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

	messageBus := bus.New(100, log)
	require.NoError(t, messageBus.Start(context.Background()))
	defer func() { _ = messageBus.Stop() }()

	pool := NewPool(1, 10, log, messageBus)

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

	messageBus := bus.New(100, log)
	require.NoError(t, messageBus.Start(context.Background()))
	defer func() { _ = messageBus.Stop() }()

	pool := NewPool(1, 10, log, messageBus)

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

	messageBus := bus.New(100, log)
	require.NoError(t, messageBus.Start(context.Background()))
	defer func() { _ = messageBus.Stop() }()

	pool := NewPool(1, 10, log, messageBus)

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

	messageBus := bus.New(100, log)
	require.NoError(t, messageBus.Start(context.Background()))
	defer func() { _ = messageBus.Stop() }()

	pool := NewPool(1, 10, log, messageBus)

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

func TestPool_ExecuteSendMessage_Success(t *testing.T) {
	log, err := logger.New(logger.Config{Level: "debug", Format: "text", Output: "stdout"})
	require.NoError(t, err)

	ctx := context.Background()

	messageBus := bus.New(100, log)
	require.NoError(t, messageBus.Start(ctx))
	defer func() { _ = messageBus.Stop() }()

	// Subscribe to outbound messages to verify
	outboundCh := messageBus.SubscribeOutbound(ctx)

	pool := NewPool(1, 10, log, messageBus)

	payload := cron.CronTaskPayload{
		Tool:      "send_message",
		SessionID: "telegram:987654321",
		Payload:   map[string]interface{}{"message": "Hello from cron!"},
	}

	task := Task{
		ID:      "send-msg-task",
		Type:    "cron",
		Payload: payload,
	}

	result := pool.executeCronTask(ctx, task)

	assert.Equal(t, task.ID, result.TaskID)
	assert.NoError(t, result.Error)
	assert.Contains(t, result.Output, "message sent to telegram:987654321")

	// Verify outbound message was published
	select {
	case msg := <-outboundCh:
		assert.Equal(t, bus.ChannelTypeTelegram, msg.ChannelType)
		assert.Equal(t, "telegram:987654321", msg.SessionID)
		assert.Equal(t, "Hello from cron!", msg.Content)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Timeout waiting for outbound message")
	}
}

func TestPool_ExecuteAgent_Success(t *testing.T) {
	log, err := logger.New(logger.Config{Level: "debug", Format: "text", Output: "stdout"})
	require.NoError(t, err)

	ctx := context.Background()

	messageBus := bus.New(100, log)
	require.NoError(t, messageBus.Start(ctx))
	defer func() { _ = messageBus.Stop() }()

	// Subscribe to inbound messages to verify
	inboundCh := messageBus.SubscribeInbound(ctx)

	pool := NewPool(1, 10, log, messageBus)

	payload := cron.CronTaskPayload{
		Tool:      "agent",
		SessionID: "telegram:987654321",
		Payload:   map[string]interface{}{"message": "Process this task"},
	}

	task := Task{
		ID:      "agent-task",
		Type:    "cron",
		Payload: payload,
	}

	result := pool.executeCronTask(ctx, task)

	assert.Equal(t, task.ID, result.TaskID)
	assert.NoError(t, result.Error)
	assert.Contains(t, result.Output, "agent message sent to telegram:987654321")

	// Verify inbound message was published
	select {
	case msg := <-inboundCh:
		assert.Equal(t, bus.ChannelTypeTelegram, msg.ChannelType)
		assert.Equal(t, "telegram:987654321", msg.SessionID)
		assert.Equal(t, "Process this task", msg.Content)
		assert.Equal(t, "agent", msg.Metadata["tool"])
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Timeout waiting for inbound message")
	}
}

func TestPool_ExecuteSendMessage_InvalidSessionID(t *testing.T) {
	log, err := logger.New(logger.Config{Level: "debug", Format: "text", Output: "stdout"})
	require.NoError(t, err)

	ctx := context.Background()

	messageBus := bus.New(100, log)
	require.NoError(t, messageBus.Start(ctx))
	defer func() { _ = messageBus.Stop() }()

	pool := NewPool(1, 10, log, messageBus)

	payload := cron.CronTaskPayload{
		Tool:      "send_message",
		SessionID: "invalid_format",
		Payload:   nil,
	}

	task := Task{
		ID:      "invalid-session-task",
		Type:    "cron",
		Payload: payload,
	}

	result := pool.executeCronTask(ctx, task)

	assert.Equal(t, task.ID, result.TaskID)
	assert.Error(t, result.Error)
	assert.Contains(t, result.Error.Error(), "invalid session_id format")
}

func TestPool_ExecuteAgent_InvalidSessionID(t *testing.T) {
	log, err := logger.New(logger.Config{Level: "debug", Format: "text", Output: "stdout"})
	require.NoError(t, err)

	ctx := context.Background()

	messageBus := bus.New(100, log)
	require.NoError(t, messageBus.Start(ctx))
	defer func() { _ = messageBus.Stop() }()

	pool := NewPool(1, 10, log, messageBus)

	payload := cron.CronTaskPayload{
		Tool:      "agent",
		SessionID: "missing_colon",
		Payload:   nil,
	}

	task := Task{
		ID:      "agent-invalid-session",
		Type:    "cron",
		Payload: payload,
	}

	result := pool.executeCronTask(ctx, task)

	assert.Equal(t, task.ID, result.TaskID)
	assert.Error(t, result.Error)
	assert.Contains(t, result.Error.Error(), "invalid session_id format")
}

func TestPool_ExecuteSendMessage_NoMessageContent(t *testing.T) {
	log, err := logger.New(logger.Config{Level: "debug", Format: "text", Output: "stdout"})
	require.NoError(t, err)

	ctx := context.Background()

	messageBus := bus.New(100, log)
	require.NoError(t, messageBus.Start(ctx))
	defer func() { _ = messageBus.Stop() }()

	pool := NewPool(1, 10, log, messageBus)

	payload := cron.CronTaskPayload{
		Tool:      "send_message",
		SessionID: "telegram:987654321",
		Payload:   map[string]interface{}{},
	}

	task := Task{
		ID:      "no-content-task",
		Type:    "cron",
		Payload: payload,
	}

	result := pool.executeCronTask(ctx, task)

	assert.Equal(t, task.ID, result.TaskID)
	assert.Error(t, result.Error)
	assert.Contains(t, result.Error.Error(), "no message content provided")
}

func TestPool_ExecuteSendMessage_ChannelMismatch(t *testing.T) {
	log, err := logger.New(logger.Config{Level: "debug", Format: "text", Output: "stdout"})
	require.NoError(t, err)

	ctx := context.Background()

	messageBus := bus.New(100, log)
	require.NoError(t, messageBus.Start(ctx))
	defer func() { _ = messageBus.Stop() }()

	pool := NewPool(1, 10, log, messageBus)

	payload := cron.CronTaskPayload{
		Tool:      "agent",
		SessionID: "discord:123456", // Wrong channel
		Payload:   map[string]interface{}{"message": "test message"},
	}

	task := Task{
		ID:      "channel-mismatch-task",
		Type:    "cron",
		Payload: payload,
	}

	result := pool.executeCronTask(ctx, task)

	// Note: executeAgent doesn't validate channel mismatch currently
	// This test documents current behavior
	assert.Equal(t, task.ID, result.TaskID)
	assert.NoError(t, result.Error)
}
