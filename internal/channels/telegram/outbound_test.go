package telegram

import (
	"context"
	"testing"
	"time"

	"github.com/aatumaykin/nexbot/internal/bus"
	"github.com/aatumaykin/nexbot/internal/config"
	"github.com/aatumaykin/nexbot/internal/logger"
	"github.com/stretchr/testify/require"
)

func Test_publishResult_Success(t *testing.T) {
	log, _ := logger.New(logger.Config{
		Level:  "debug",
		Format: "text",
		Output: "stdout",
	})

	msgBus := bus.New(100, 10, log)
	err := msgBus.Start(context.Background())
	require.NoError(t, err)
	defer func() {
		err := msgBus.Stop()
		require.NoError(t, err)
	}()

	ctx := context.Background()

	cfg := config.TelegramConfig{}
	conn := New(cfg, log, msgBus)
	conn.ctx = ctx

	resultCh := msgBus.SubscribeSendResults(ctx)

	correlationID := "test-correlation-123"
	chatID := int64(987654321)
	msg := bus.OutboundMessage{
		CorrelationID: correlationID,
		ChannelType:   bus.ChannelTypeTelegram,
		Content:       "test message",
	}

	go func() {
		conn.publishResult(msg, chatID, true, nil)
	}()

	select {
	case result := <-resultCh:
		require.Equal(t, correlationID, result.CorrelationID)
		require.Equal(t, bus.ChannelTypeTelegram, result.ChannelType)
		require.True(t, result.Success)
		require.Nil(t, result.Error)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout waiting for result")
	}
}

func Test_publishResult_Error(t *testing.T) {
	log, _ := logger.New(logger.Config{
		Level:  "debug",
		Format: "text",
		Output: "stdout",
	})

	msgBus := bus.New(100, 10, log)
	err := msgBus.Start(context.Background())
	require.NoError(t, err)
	defer func() {
		err := msgBus.Stop()
		require.NoError(t, err)
	}()

	ctx := context.Background()

	cfg := config.TelegramConfig{}
	conn := New(cfg, log, msgBus)
	conn.ctx = ctx

	resultCh := msgBus.SubscribeSendResults(ctx)

	correlationID := "test-correlation-456"
	chatID := int64(987654321)
	testErr := testError("test error")
	msg := bus.OutboundMessage{
		CorrelationID: correlationID,
		ChannelType:   bus.ChannelTypeTelegram,
		Content:       "test message",
	}

	go func() {
		conn.publishResult(msg, chatID, false, testErr)
	}()

	select {
	case result := <-resultCh:
		require.Equal(t, correlationID, result.CorrelationID)
		require.Equal(t, bus.ChannelTypeTelegram, result.ChannelType)
		require.False(t, result.Success)
		require.Nil(t, result.Error)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout waiting for result")
	}
}

func Test_sendTextMessage_PublishesResultImmediately(t *testing.T) {
	log, _ := logger.New(logger.Config{
		Level:  "debug",
		Format: "text",
		Output: "stdout",
	})

	msgBus := bus.New(100, 10, log)
	err := msgBus.Start(context.Background())
	require.NoError(t, err)
	defer func() {
		err := msgBus.Stop()
		require.NoError(t, err)
	}()

	ctx := context.Background()

	cfg := config.TelegramConfig{
		SendTimeoutSeconds: 5,
	}

	conn := New(cfg, log, msgBus)
	conn.ctx = ctx

	mockBot := NewMockBotSuccess()
	conn.bot = mockBot

	resultCh := msgBus.SubscribeSendResults(ctx)

	correlationID := "test-correlation-789"
	chatID := int64(987654321)
	msg := bus.OutboundMessage{
		CorrelationID: correlationID,
		ChannelType:   bus.ChannelTypeTelegram,
		Content:       "test message",
		Type:          bus.MessageTypeText,
	}

	go func() {
		conn.sendTextMessage(msg, chatID)
	}()

	select {
	case result := <-resultCh:
		require.Equal(t, correlationID, result.CorrelationID)
		require.Equal(t, bus.ChannelTypeTelegram, result.ChannelType)
		require.True(t, result.Success)
		require.Nil(t, result.Error)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout waiting for result - may indicate delay between send and publish")
	}
}

func Test_sendTextMessage_PublishesErrorResult(t *testing.T) {
	log, _ := logger.New(logger.Config{
		Level:  "debug",
		Format: "text",
		Output: "stdout",
	})

	msgBus := bus.New(100, 10, log)
	err := msgBus.Start(context.Background())
	require.NoError(t, err)
	defer func() {
		err := msgBus.Stop()
		require.NoError(t, err)
	}()

	ctx := context.Background()

	cfg := config.TelegramConfig{
		SendTimeoutSeconds: 5,
	}

	conn := New(cfg, log, msgBus)
	conn.ctx = ctx

	mockBot := NewMockBotError(testError("send error"))
	conn.bot = mockBot

	resultCh := msgBus.SubscribeSendResults(ctx)

	correlationID := "test-correlation-101"
	chatID := int64(987654321)
	msg := bus.OutboundMessage{
		CorrelationID: correlationID,
		ChannelType:   bus.ChannelTypeTelegram,
		Content:       "test message",
		Type:          bus.MessageTypeText,
	}

	go func() {
		conn.sendTextMessage(msg, chatID)
	}()

	select {
	case result := <-resultCh:
		require.Equal(t, correlationID, result.CorrelationID)
		require.Equal(t, bus.ChannelTypeTelegram, result.ChannelType)
		require.False(t, result.Success)
		require.Nil(t, result.Error)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout waiting for result")
	}
}

func Test_editMessage_PublishesResultImmediately(t *testing.T) {
	log, _ := logger.New(logger.Config{
		Level:  "debug",
		Format: "text",
		Output: "stdout",
	})

	msgBus := bus.New(100, 10, log)
	err := msgBus.Start(context.Background())
	require.NoError(t, err)
	defer func() {
		err := msgBus.Stop()
		require.NoError(t, err)
	}()

	ctx := context.Background()

	cfg := config.TelegramConfig{
		SendTimeoutSeconds: 5,
	}

	conn := New(cfg, log, msgBus)
	conn.ctx = ctx

	mockBot := NewMockBotSuccess()
	conn.bot = mockBot

	resultCh := msgBus.SubscribeSendResults(ctx)

	correlationID := "test-correlation-202"
	chatID := int64(987654321)
	msg := bus.OutboundMessage{
		CorrelationID: correlationID,
		ChannelType:   bus.ChannelTypeTelegram,
		Content:       "edited message",
		MessageID:     "123",
		Type:          bus.MessageTypeEdit,
	}

	go func() {
		conn.editMessage(msg, chatID)
	}()

	select {
	case result := <-resultCh:
		require.Equal(t, correlationID, result.CorrelationID)
		require.Equal(t, bus.ChannelTypeTelegram, result.ChannelType)
		require.True(t, result.Success)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout waiting for result - may indicate delay between send and publish")
	}
}

func Test_deleteMessage_PublishesResultImmediately(t *testing.T) {
	log, _ := logger.New(logger.Config{
		Level:  "debug",
		Format: "text",
		Output: "stdout",
	})

	msgBus := bus.New(100, 10, log)
	err := msgBus.Start(context.Background())
	require.NoError(t, err)
	defer func() {
		err := msgBus.Stop()
		require.NoError(t, err)
	}()

	ctx := context.Background()

	cfg := config.TelegramConfig{
		SendTimeoutSeconds: 5,
	}

	conn := New(cfg, log, msgBus)
	conn.ctx = ctx

	mockBot := NewMockBotSuccess()
	conn.bot = mockBot

	resultCh := msgBus.SubscribeSendResults(ctx)

	correlationID := "test-correlation-303"
	chatID := int64(987654321)
	msg := bus.OutboundMessage{
		CorrelationID: correlationID,
		ChannelType:   bus.ChannelTypeTelegram,
		MessageID:     "123",
		Type:          bus.MessageTypeDelete,
	}

	go func() {
		conn.deleteMessage(msg, chatID)
	}()

	select {
	case result := <-resultCh:
		require.Equal(t, correlationID, result.CorrelationID)
		require.Equal(t, bus.ChannelTypeTelegram, result.ChannelType)
		require.True(t, result.Success)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout waiting for result - may indicate delay between send and publish")
	}
}

func Test_handleOutbound_ConcurrentMessages(t *testing.T) {
	log, _ := logger.New(logger.Config{
		Level:  "debug",
		Format: "text",
		Output: "stdout",
	})

	msgBus := bus.New(100, 10, log)
	err := msgBus.Start(context.Background())
	require.NoError(t, err)
	defer func() {
		err := msgBus.Stop()
		require.NoError(t, err)
	}()

	ctx := t.Context()

	cfg := config.TelegramConfig{
		SendTimeoutSeconds: 5,
	}

	conn := New(cfg, log, msgBus)
	conn.ctx = ctx

	mockBot := NewMockBotSuccess()
	conn.bot = mockBot

	outboundCh := make(chan bus.OutboundMessage, 10)
	conn.outboundCh = outboundCh

	resultCh := msgBus.SubscribeSendResults(ctx)

	go conn.handleOutbound()

	messageCount := 5
	for i := range messageCount {
		outboundCh <- bus.OutboundMessage{
			CorrelationID: testCorrelationID(i),
			ChannelType:   bus.ChannelTypeTelegram,
			Content:       "concurrent message",
			SessionID:     "telegram:987654321",
			Type:          bus.MessageTypeText,
		}
	}

	results := make(map[string]bool)
	timeout := time.After(1 * time.Second)
	for {
		select {
		case result := <-resultCh:
			results[result.CorrelationID] = true
			if len(results) == messageCount {
				return
			}
		case <-timeout:
			t.Fatalf("timeout waiting for results, received %d/%d", len(results), messageCount)
		}
	}
}

func testCorrelationID(i int) string {
	return "test-correlation-" + string(rune(i))
}

type testError string

func (e testError) Error() string {
	return string(e)
}
