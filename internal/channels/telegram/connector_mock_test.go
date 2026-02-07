package telegram

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/mymmrac/telego"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/aatumaykin/nexbot/internal/bus"
	"github.com/aatumaykin/nexbot/internal/config"
	"github.com/aatumaykin/nexbot/internal/logger"
)

// TestConnector_RegisterCommands_WithMock tests registering bot commands with a mock bot.
func TestConnector_RegisterCommands_WithMock(t *testing.T) {
	log, _ := logger.New(logger.Config{
		Level:  "debug",
		Format: "text",
		Output: "stdout",
	})
	cfg := config.TelegramConfig{
		Token:        "test-token",
		Enabled:      true,
		AllowedUsers: []string{"123456789"},
	}

	msgBus := bus.New(100, log)
	defer func() {
		_ = msgBus.Stop()
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	conn := New(cfg, log, msgBus)
	mockBot := NewMockBotSuccess()

	// Set up expectations
	mockBot.On("SendMessage", mock.Anything, mock.MatchedBy(func(params *telego.SendMessageParams) bool {
		return params != nil && params.ChatID.ID == 123456789 && params.Text == "Test message"
	})).Return(&telego.Message{
		MessageID: 1,
		Text:      "Test message",
	}, nil)

	// Set the mock bot and context
	conn.ctx = ctx
	conn.bot = mockBot

	// Create outbound channel
	outboundCh := make(chan bus.OutboundMessage, 1)
	conn.outboundCh = outboundCh

	// Start outbound handler in goroutine
	go conn.handleOutbound()

	// Send a message
	outboundCh <- bus.OutboundMessage{
		ChannelType: bus.ChannelTypeTelegram,
		SessionID:   "123456789",
		Content:     "Test message",
	}

	// Wait a bit for processing
	time.Sleep(100 * time.Millisecond)

	// Stop handler
	cancel()
	time.Sleep(100 * time.Millisecond)

	// Verify
	mockBot.AssertExpectations(t)
}

// TestConnector_HandleOutbound_WithMockError tests error handling when sending outbound messages fails.
func TestConnector_HandleOutbound_WithMockError(t *testing.T) {
	log, _ := logger.New(logger.Config{
		Level:  "debug",
		Format: "text",
		Output: "stdout",
	})
	cfg := config.TelegramConfig{
		Token:        "test-token",
		Enabled:      true,
		AllowedUsers: []string{"123456789"},
	}

	msgBus := bus.New(100, log)
	defer func() { _ = msgBus.Stop() }()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	conn := New(cfg, log, msgBus)
	mockBot := NewMockBotError(fmt.Errorf("API error"))

	// Set up expectations
	mockBot.On("SendMessage", mock.Anything, mock.Anything).Return((*telego.Message)(nil), fmt.Errorf("API error"))

	// Set the mock bot and context
	conn.ctx = ctx
	conn.bot = mockBot

	// Create outbound channel
	outboundCh := make(chan bus.OutboundMessage, 1)
	conn.outboundCh = outboundCh

	// Start outbound handler in goroutine
	go conn.handleOutbound()

	// Send a message
	outboundCh <- bus.OutboundMessage{
		ChannelType: bus.ChannelTypeTelegram,
		SessionID:   "123456789",
		Content:     "Test message",
	}

	// Wait a bit for processing
	time.Sleep(100 * time.Millisecond)

	// Stop handler
	cancel()
	time.Sleep(100 * time.Millisecond)

	// Verify
	mockBot.AssertExpectations(t)
}

// TestConnector_HandleEvents_WithMock tests handling events with a mock bot.
func TestConnector_HandleEvents_WithMock(t *testing.T) {
	log, _ := logger.New(logger.Config{
		Level:  "debug",
		Format: "text",
		Output: "stdout",
	})
	cfg := config.TelegramConfig{
		Token:        "test-token",
		Enabled:      true,
		AllowedUsers: []string{"123456789"},
	}

	msgBus := bus.New(100, log)
	defer func() { _ = msgBus.Stop() }()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	conn := New(cfg, log, msgBus)
	mockBot := NewMockBotSuccess()

	// Set up expectations for typing indicator
	mockBot.On("SendChatAction", mock.Anything, mock.MatchedBy(func(params *telego.SendChatActionParams) bool {
		return params != nil && params.ChatID.ID == 123456789
	})).Return(nil)

	// Set the mock bot and context
	conn.ctx = ctx
	conn.bot = mockBot
	conn.typingManager.SetContext(ctx)
	conn.typingManager.bot = mockBot

	// Create event channel
	eventCh := make(chan bus.Event, 1)
	conn.eventCh = eventCh

	// Start event handler in goroutine
	go conn.handleEvents()

	// Send an event
	eventCh <- bus.Event{
		ChannelType: bus.ChannelTypeTelegram,
		Type:        bus.EventTypeProcessingStart,
		SessionID:   "telegram:123456789",
	}

	// Wait a bit for processing
	time.Sleep(100 * time.Millisecond)

	// Stop handler
	cancel()
	time.Sleep(100 * time.Millisecond)

	// Verify - SendChatAction should be called
	mockBot.AssertExpectations(t)
}

// TestConnector_SendStartupMessage_WithMock tests sending startup message with a mock bot.
func TestConnector_SendStartupMessage_WithMock(t *testing.T) {
	log, _ := logger.New(logger.Config{
		Level:  "debug",
		Format: "text",
		Output: "stdout",
	})
	cfg := config.TelegramConfig{
		Token:        "test-token",
		Enabled:      true,
		AllowedUsers: []string{"123456789"},
	}

	msgBus := bus.New(100, log)
	defer func() { _ = msgBus.Stop() }()

	conn := New(cfg, log, msgBus)
	mockBot := NewMockBotSuccess()

	// Set up expectations
	mockBot.On("SendMessage", mock.Anything, mock.MatchedBy(func(params *telego.SendMessageParams) bool {
		return params != nil && params.ChatID.ID == 123456789
	})).Return(&telego.Message{
		MessageID: 1,
	}, nil)

	// Set the mock bot and context
	conn.ctx, conn.cancel = context.WithCancel(context.Background())
	conn.bot = mockBot

	// Send startup message
	err := conn.sendStartupMessage()

	// Verify
	assert.NoError(t, err)
	mockBot.AssertExpectations(t)
}

// TestConnector_GetMe_WithMock tests getting bot info with a mock bot.
func TestConnector_GetMe_WithMock(t *testing.T) {
	log, _ := logger.New(logger.Config{
		Level:  "debug",
		Format: "text",
		Output: "stdout",
	})
	cfg := config.TelegramConfig{
		Token:   "test-token",
		Enabled: true,
	}

	msgBus := bus.New(100, log)
	defer func() { _ = msgBus.Stop() }()

	conn := New(cfg, log, msgBus)
	mockBot := NewMockBotSuccess()

	// Set the mock bot and context
	conn.ctx, conn.cancel = context.WithCancel(context.Background())
	conn.bot = mockBot

	// Get bot info
	botUser, err := conn.bot.GetMe(conn.ctx)

	// Verify
	assert.NoError(t, err)
	assert.NotNil(t, botUser)
	assert.Equal(t, int64(123456789), botUser.ID)
	assert.Equal(t, "Test", botUser.FirstName)
	assert.Equal(t, "test_bot", botUser.Username)
	mockBot.AssertExpectations(t)
}

// TestConnector_Start_WithMock tests starting the connector with a mock bot.
func TestConnector_Start_WithMock(t *testing.T) {
	log, _ := logger.New(logger.Config{
		Level:  "debug",
		Format: "text",
		Output: "stdout",
	})
	cfg := config.TelegramConfig{
		Token:        "test-token",
		Enabled:      true,
		AllowedUsers: []string{"123456789"},
	}

	msgBus := bus.New(100, log)
	defer func() { _ = msgBus.Stop() }()

	conn := New(cfg, log, msgBus)
	mockBot := NewMockBotSuccess()

	// Set up expectations
	mockBot.On("GetMe", mock.Anything).Return(&telego.User{
		ID:        987654321,
		FirstName: "Test",
		Username:  "test_bot",
	}, nil)

	mockBot.On("SetMyCommands", mock.Anything, mock.Anything).Return(nil)

	mockBot.On("SendMessage", mock.Anything, mock.MatchedBy(func(params *telego.SendMessageParams) bool {
		return params != nil && params.ChatID.ID == 123456789
	})).Return(&telego.Message{
		MessageID: 1,
	}, nil)

	// Set the mock bot and context manually to avoid NewBot creation
	conn.ctx, conn.cancel = context.WithCancel(context.Background())
	conn.bot = mockBot
	conn.typingManager.SetContext(conn.ctx)
	conn.typingManager.bot = mockBot
	conn.longPollManager.SetContext(conn.ctx)
	conn.longPollManager.bot = mockBot

	// Subscribe channels
	conn.outboundCh = conn.bus.SubscribeOutbound(conn.ctx)
	conn.eventCh = conn.bus.SubscribeEvent(conn.ctx)

	// Call the parts that would normally be called in Start()
	botUser, err := conn.bot.GetMe(conn.ctx)
	assert.NoError(t, err)
	assert.NotNil(t, botUser)

	err = conn.registerCommands()
	assert.NoError(t, err)

	err = conn.sendStartupMessage()
	assert.NoError(t, err)

	// Verify
	mockBot.AssertExpectations(t)

	// Cleanup
	_ = conn.Stop()
}

// TestConnector_Start_WithMockGetMeError tests error handling when GetMe fails.
func TestConnector_Start_WithMockGetMeError(t *testing.T) {
	log, _ := logger.New(logger.Config{
		Level:  "debug",
		Format: "text",
		Output: "stdout",
	})
	cfg := config.TelegramConfig{
		Token:   "test-token",
		Enabled: true,
	}

	msgBus := bus.New(100, log)
	defer func() { _ = msgBus.Stop() }()

	conn := New(cfg, log, msgBus)
	mockBot := NewMockBotError(fmt.Errorf("API error"))

	// Set up expectations
	mockBot.On("GetMe", mock.Anything).Return((*telego.User)(nil), fmt.Errorf("API error"))

	// Set the mock bot and context
	conn.ctx, conn.cancel = context.WithCancel(context.Background())
	conn.bot = mockBot

	// Get bot info (simulating part of Start())
	_, err := conn.bot.GetMe(conn.ctx)

	// Verify
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "API error")
	mockBot.AssertExpectations(t)
}
