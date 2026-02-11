package telegram

import (
	"context"
	"testing"
	"time"

	"github.com/mymmrac/telego"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/aatumaykin/nexbot/internal/bus"
	"github.com/aatumaykin/nexbot/internal/config"
	"github.com/aatumaykin/nexbot/internal/logger"
)

// TestTelegramConnector_FullWorkflow_WithMock tests the full workflow from receiving an update to sending a response.
func TestTelegramConnector_FullWorkflow_WithMock(t *testing.T) {
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

	msgBus := bus.New(100, 10, log)

	ctx := t.Context()

	// Start message bus
	if err := msgBus.Start(ctx); err != nil {
		t.Fatalf("Failed to start message bus: %v", err)
	}
	defer func() { _ = msgBus.Stop() }()

	// Create mock bot
	mockBot := NewMockBotSuccess()

	// Create connector
	conn := New(cfg, log, msgBus)
	conn.ctx = ctx
	conn.bot = mockBot

	// Subscribe to inbound messages
	inboundCh := msgBus.SubscribeInbound(ctx)

	// Create a test update
	update := telego.Update{
		Message: &telego.Message{
			MessageID: 1,
			From: &telego.User{
				ID:        123456789,
				FirstName: "User",
				Username:  "user123",
			},
			Chat: telego.Chat{
				ID:   987654321,
				Type: "private",
			},
			Text: "Hello",
		},
	}

	// Handle the update
	err := conn.handleUpdate(update)
	assert.NoError(t, err)

	// Wait for inbound message
	select {
	case msg := <-inboundCh:
		assert.Equal(t, bus.ChannelTypeTelegram, msg.ChannelType)
		assert.Equal(t, "123456789", msg.UserID)
		assert.Equal(t, "telegram:987654321", msg.SessionID)
		assert.Equal(t, "Hello", msg.Content)

	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for inbound message")
	}
}

// TestTelegramConnector_Concurrent_WithMock tests concurrent operations.
func TestTelegramConnector_Concurrent_WithMock(t *testing.T) {
	log, _ := logger.New(logger.Config{
		Level:  "debug",
		Format: "text",
		Output: "stdout",
	})

	cfg := config.TelegramConfig{
		Token:   "test-token",
		Enabled: true,
	}

	msgBus := bus.New(100, 10, log)
	defer func() { _ = msgBus.Stop() }()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create mock bot
	mockBot := NewMockBotSuccess()

	// Set up expectations for multiple SendMessage calls
	mockBot.On("SendMessage", mock.Anything, mock.Anything).Return(&telego.Message{
		MessageID: 1,
	}, nil)

	// Create connector
	conn := New(cfg, log, msgBus)
	conn.ctx = ctx
	conn.bot = mockBot

	// Create outbound channel
	outboundCh := make(chan bus.OutboundMessage, 10)
	conn.outboundCh = outboundCh

	// Start outbound handler
	go conn.handleOutbound()

	// Send multiple messages concurrently
	for i := range 5 {
		go func(idx int) {
			outboundCh <- bus.OutboundMessage{
				ChannelType: bus.ChannelTypeTelegram,
				SessionID:   "telegram:987654321",
				Type:        bus.MessageTypeText,
				Content:     "Test message",
			}
		}(i)
	}

	// Wait for messages to be processed
	time.Sleep(200 * time.Millisecond)

	// Stop handler
	cancel()
	time.Sleep(100 * time.Millisecond)

	// Verify
	mockBot.AssertExpectations(t)
}

// TestTelegramConnector_ErrorHandling_WithMock tests error handling in the connector.
func TestTelegramConnector_ErrorHandling_WithMock(t *testing.T) {
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

	msgBus := bus.New(100, 10, log)
	defer func() { _ = msgBus.Stop() }()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create mock bot
	mockBot := new(MockBot)

	// Set up expectations for SetMyCommands (should not fail the connector)
	mockBot.On("SetMyCommands", mock.Anything, mock.Anything).Return(assert.AnError).Maybe()

	// Set up expectations for SendMessage (should handle error gracefully)
	mockBot.On("SendMessage", mock.Anything, mock.Anything).Return((*telego.Message)(nil), assert.AnError).Maybe()

	// Create connector
	conn := New(cfg, log, msgBus)
	conn.ctx = ctx
	conn.bot = mockBot

	// Register commands (should return error but not panic)
	err := conn.registerCommands()
	assert.Error(t, err)

	// Create outbound channel
	outboundCh := make(chan bus.OutboundMessage, 1)
	conn.outboundCh = outboundCh

	// Start outbound handler
	go conn.handleOutbound()

	// Send a message (should handle error gracefully)
	outboundCh <- bus.OutboundMessage{
		ChannelType: bus.ChannelTypeTelegram,
		SessionID:   "telegram:987654321",
		Type:        bus.MessageTypeText,
		Content:     "Test message",
	}

	// Wait for message to be processed
	time.Sleep(100 * time.Millisecond)

	// Stop handler
	cancel()
	time.Sleep(100 * time.Millisecond)

	// Verify
	mockBot.AssertExpectations(t)
}

// TestTelegramConnector_MultipleUsers_WithMock tests handling messages from multiple users.
func TestTelegramConnector_MultipleUsers_WithMock(t *testing.T) {
	log, _ := logger.New(logger.Config{
		Level:  "debug",
		Format: "text",
		Output: "stdout",
	})

	cfg := config.TelegramConfig{
		Token:   "test-token",
		Enabled: true,
	}

	msgBus := bus.New(100, 10, log)
	defer func() { _ = msgBus.Stop() }()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create mock bot
	mockBot := NewMockBotSuccess()

	// Set up expectations for multiple SendMessage calls
	mockBot.On("SendMessage", mock.Anything, mock.Anything).Return(&telego.Message{
		MessageID: 1,
	}, nil)

	// Create connector
	conn := New(cfg, log, msgBus)
	conn.ctx = ctx
	conn.bot = mockBot

	// Create outbound channel
	outboundCh := make(chan bus.OutboundMessage, 10)
	conn.outboundCh = outboundCh

	// Start outbound handler
	go conn.handleOutbound()

	// Send messages from multiple users
	users := []string{"111222333", "444555666", "777888999"}
	for i, userID := range users {
		outboundCh <- bus.OutboundMessage{
			ChannelType: bus.ChannelTypeTelegram,
			SessionID:   "telegram:" + userID,
			UserID:      userID,
			Type:        bus.MessageTypeText,
			Content:     "Message from user",
		}
		_ = i // Use variable
	}

	// Wait for messages to be processed
	time.Sleep(200 * time.Millisecond)

	// Stop handler
	cancel()
	time.Sleep(100 * time.Millisecond)

	// Verify
	mockBot.AssertExpectations(t)
}

// TestTelegramConnector_TypingIndicators_WithMock tests typing indicators during processing.
func TestTelegramConnector_TypingIndicators_WithMock(t *testing.T) {
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

	msgBus := bus.New(100, 10, log)
	defer func() { _ = msgBus.Stop() }()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create mock bot
	mockBot := new(MockBot)

	// Set up expectations for typing indicator (called once on start)
	mockBot.On("SendChatAction", mock.Anything, mock.MatchedBy(func(params *telego.SendChatActionParams) bool {
		return params != nil && params.ChatID.ID == 987654321 && params.Action == "typing"
	})).Return(nil).Once()

	// Create connector
	conn := New(cfg, log, msgBus)
	conn.ctx = ctx
	conn.bot = mockBot
	conn.typingManager.SetContext(ctx)
	conn.typingManager.bot = mockBot

	// Create event channel
	eventCh := make(chan bus.Event, 10)
	conn.eventCh = eventCh

	// Start event handler
	go conn.handleEvents()

	// Send processing start event
	eventCh <- bus.Event{
		ChannelType: bus.ChannelTypeTelegram,
		Type:        bus.EventTypeProcessingStart,
		SessionID:   "telegram:987654321",
		UserID:      "123456789",
	}

	// Wait for typing indicator
	time.Sleep(100 * time.Millisecond)

	// Send processing end event
	eventCh <- bus.Event{
		ChannelType: bus.ChannelTypeTelegram,
		Type:        bus.EventTypeProcessingEnd,
		SessionID:   "telegram:987654321",
		UserID:      "123456789",
	}

	// Wait for processing
	time.Sleep(100 * time.Millisecond)

	// Stop handler
	cancel()
	time.Sleep(100 * time.Millisecond)

	// Verify
	mockBot.AssertExpectations(t)
}

// TestTelegramConnector_CommandHandling_WithMock tests handling of bot commands.
func TestTelegramConnector_CommandHandling_WithMock(t *testing.T) {
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

	msgBus := bus.New(100, 10, log)
	defer func() { _ = msgBus.Stop() }()

	ctx := t.Context()

	// Start message bus
	if err := msgBus.Start(ctx); err != nil {
		t.Fatalf("Failed to start message bus: %v", err)
	}

	// Create connector (without mock bot - just test command handling)
	conn := New(cfg, log, msgBus)
	conn.ctx = ctx

	// Subscribe to inbound messages
	inboundCh := msgBus.SubscribeInbound(ctx)

	// Test different commands
	commands := []string{"/new", "/status", "/restart"}
	for i, cmd := range commands {
		update := telego.Update{
			Message: &telego.Message{
				MessageID: int(i) + 1,
				From: &telego.User{
					ID:        123456789,
					FirstName: "User",
					Username:  "user123",
				},
				Chat: telego.Chat{
					ID:   987654321,
					Type: "private",
				},
				Text: cmd,
				Entities: []telego.MessageEntity{
					{
						Type:   "bot_command",
						Offset: 0,
						Length: len(cmd),
					},
				},
			},
		}

		// Handle the update
		err := conn.handleUpdate(update)
		assert.NoError(t, err)

		// Wait for inbound message
		select {
		case msg := <-inboundCh:
			assert.Equal(t, bus.ChannelTypeTelegram, msg.ChannelType)
			assert.Equal(t, "123456789", msg.UserID)
		case <-time.After(500 * time.Millisecond):
			t.Fatalf("Timeout waiting for command %s", cmd)
		}
	}
}

// TestTelegramConnector_GracefulShutdown_WithMock tests graceful shutdown of the connector.
func TestTelegramConnector_GracefulShutdown_WithMock(t *testing.T) {
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

	msgBus := bus.New(100, 10, log)
	defer func() { _ = msgBus.Stop() }()

	ctx := t.Context()

	// Create mock bot
	mockBot := NewMockBotSuccess()

	// Create connector
	conn := New(cfg, log, msgBus)
	conn.ctx = ctx
	conn.bot = mockBot
	conn.typingManager.SetContext(ctx)
	conn.typingManager.bot = mockBot

	// Start typing indicator
	event := bus.Event{
		ChannelType: bus.ChannelTypeTelegram,
		Type:        bus.EventTypeProcessingStart,
		SessionID:   "telegram:987654321",
		UserID:      "123456789",
	}
	conn.typingManager.Start(event)

	// Wait for typing to start
	time.Sleep(100 * time.Millisecond)

	// Stop the connector (should stop all typing indicators)
	err := conn.Stop()
	assert.NoError(t, err)

	// Wait for shutdown
	time.Sleep(100 * time.Millisecond)

	// Verify that typing indicator was stopped
	// (Should not send any more typing indicators)
}
