package telegram

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/aatumaykin/nexbot/internal/agent/loop"
	"github.com/aatumaykin/nexbot/internal/bus"
	"github.com/aatumaykin/nexbot/internal/config"
	"github.com/aatumaykin/nexbot/internal/logger"
	"github.com/mymmrac/telego"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// Suppress unused import warning for loop (used for nil argument type)
var _ *loop.Loop

// TestConnector_New tests connector creation
func TestConnector_New(t *testing.T) {
	log, _ := logger.New(logger.Config{
		Level:  "debug",
		Format: "text",
		Output: "stdout",
	})

	msgBus := bus.New(100, log)

	cfg := config.TelegramConfig{
		Enabled: true,
		Token:   "test-token",
	}

	conn := New(cfg, log, msgBus)

	if conn == nil {
		t.Fatal("New() returned nil")
	}

	if conn.cfg.Token != "test-token" {
		t.Errorf("Expected token 'test-token', got '%s'", conn.cfg.Token)
	}

	if conn.logger != log {
		t.Error("Logger not set correctly")
	}

	if conn.bus != msgBus {
		t.Error("Message bus not set correctly")
	}
}

// TestConnector_Start_Disabled tests starting connector when disabled
func TestConnector_Start_Disabled(t *testing.T) {
	log, _ := logger.New(logger.Config{
		Level:  "debug",
		Format: "text",
		Output: "stdout",
	})

	msgBus := bus.New(100, log)

	cfg := config.TelegramConfig{
		Enabled: false,
	}

	conn := New(cfg, log, msgBus)
	ctx := context.Background()

	err := conn.Start(ctx)
	if err != nil {
		t.Fatalf("Start() with disabled connector should return nil, got %v", err)
	}

	if conn.bot != nil {
		t.Error("Bot should not be initialized when connector is disabled")
	}
}

// TestConnector_Start_ValidationError tests validation on start
func TestConnector_Start_ValidationError(t *testing.T) {
	log, _ := logger.New(logger.Config{
		Level:  "debug",
		Format: "text",
		Output: "stdout",
	})

	msgBus := bus.New(100, log)

	cfg := config.TelegramConfig{
		Enabled: true,
		Token:   "", // Empty token
	}

	conn := New(cfg, log, msgBus)
	ctx := context.Background()

	err := conn.Start(ctx)
	if err == nil {
		t.Fatal("Start() with empty token should return error")
	}

	if !errors.Is(err, errors.New("telegram token is required")) && err.Error() != "invalid config: telegram token is required" {
		t.Errorf("Expected token required error, got: %v", err)
	}
}

// TestConnector_isAllowedUser tests whitelist checking
func TestConnector_isAllowedUser(t *testing.T) {
	log, _ := logger.New(logger.Config{
		Level:  "debug",
		Format: "text",
		Output: "stdout",
	})

	msgBus := bus.New(100, log)

	tests := []struct {
		name       string
		allowed    []string
		userID     string
		shouldPass bool
	}{
		{
			name:       "empty whitelist allows all",
			allowed:    []string{},
			userID:     "123",
			shouldPass: true,
		},
		{
			name:       "user in whitelist",
			allowed:    []string{"123", "456"},
			userID:     "123",
			shouldPass: true,
		},
		{
			name:       "user not in whitelist",
			allowed:    []string{"123", "456"},
			userID:     "789",
			shouldPass: false,
		},
		{
			name:       "non-empty whitelist without user",
			allowed:    []string{"123", "456"},
			userID:     "",
			shouldPass: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.TelegramConfig{
				AllowedUsers: tt.allowed,
			}

			conn := New(cfg, log, msgBus)
			result := conn.isAllowedUser(tt.userID)

			if result != tt.shouldPass {
				t.Errorf("isAllowedUser(%s) = %v, want %v", tt.userID, result, tt.shouldPass)
			}
		})
	}
}

// TestConnector_handleUpdate_NilMessage tests that nil messages are skipped
func TestConnector_handleUpdate_NilMessage(t *testing.T) {
	log, _ := logger.New(logger.Config{
		Level:  "debug",
		Format: "text",
		Output: "stdout",
	})

	msgBus := bus.New(100, log)
	ctx := context.Background()

	cfg := config.TelegramConfig{}
	conn := New(cfg, log, msgBus)
	conn.ctx = ctx

	// Create an update without a message
	update := telego.Update{}

	// Handle the update - should not return error
	err := conn.handleUpdate(update)
	if err != nil {
		t.Fatalf("handleUpdate() with nil message should return nil, got %v", err)
	}
}

// TestConnector_handleUpdate_NilText tests that messages without text are skipped
func TestConnector_handleUpdate_NilText(t *testing.T) {
	log, _ := logger.New(logger.Config{
		Level:  "debug",
		Format: "text",
		Output: "stdout",
	})

	msgBus := bus.New(100, log)
	ctx := context.Background()

	cfg := config.TelegramConfig{}
	conn := New(cfg, log, msgBus)
	conn.ctx = ctx

	// Create an update with message but no text
	update := telego.Update{
		Message: &telego.Message{
			MessageID: 1,
			From: &telego.User{
				ID:        123456789,
				FirstName: "TestUser",
			},
			Chat: telego.Chat{
				ID:   987654321,
				Type: "private",
			},
		},
	}

	// Handle the update - should not return error
	err := conn.handleUpdate(update)
	if err != nil {
		t.Fatalf("handleUpdate() with empty text should return nil, got %v", err)
	}
}

// TestConnector_handleUpdate_Success tests successful update handling
func TestConnector_handleUpdate_Success(t *testing.T) {
	log, _ := logger.New(logger.Config{
		Level:  "debug",
		Format: "text",
		Output: "stdout",
	})

	msgBus := bus.New(100, log)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start message bus
	if err := msgBus.Start(ctx); err != nil {
		t.Fatalf("Failed to start message bus: %v", err)
	}

	cfg := config.TelegramConfig{
		AllowedUsers: []string{"123456789"},
	}

	conn := New(cfg, log, msgBus)
	conn.ctx = ctx

	// Subscribe to inbound messages
	inboundCh := msgBus.SubscribeInbound(ctx)

	// Create a test update
	update := telego.Update{
		Message: &telego.Message{
			MessageID: 1,
			From: &telego.User{
				ID:        123456789,
				FirstName: "TestUser",
				Username:  "test_user",
			},
			Chat: telego.Chat{
				ID:   987654321,
				Type: "private",
			},
			Text: "Hello, bot!",
		},
	}

	// Handle the update
	err := conn.handleUpdate(update)
	if err != nil {
		t.Fatalf("handleUpdate() failed: %v", err)
	}

	// Wait for inbound message
	select {
	case msg := <-inboundCh:
		if msg.ChannelType != bus.ChannelTypeTelegram {
			t.Errorf("Expected channel type telegram, got %s", msg.ChannelType)
		}

		if msg.UserID != "123456789" {
			t.Errorf("Expected user ID '123456789', got '%s'", msg.UserID)
		}

		if msg.SessionID != "telegram:987654321" {
			t.Errorf("Expected session ID 'telegram:987654321', got '%s'", msg.SessionID)
		}

		if msg.Content != "Hello, bot!" {
			t.Errorf("Expected content 'Hello, bot!', got '%s'", msg.Content)
		}

		// Check metadata
		if msg.Metadata["message_id"] != 1 {
			t.Errorf("Expected message_id 1, got %v", msg.Metadata["message_id"])
		}
		if msg.Metadata["chat_id"] != int64(987654321) {
			t.Errorf("Expected chat_id 987654321, got %v", msg.Metadata["chat_id"])
		}
		if msg.Metadata["chat_type"] != "private" {
			t.Errorf("Expected chat_type private, got %v", msg.Metadata["chat_type"])
		}
		if msg.Metadata["username"] != "test_user" {
			t.Errorf("Expected username test_user, got %v", msg.Metadata["username"])
		}

	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for inbound message")
	}

	t.Cleanup(func() {
		require.NoError(t, msgBus.Stop())
	})
}

// TestConnector_handleUpdate_WhitelistBlocked tests update blocking by whitelist
func TestConnector_handleUpdate_WhitelistBlocked(t *testing.T) {
	log, _ := logger.New(logger.Config{
		Level:  "debug",
		Format: "text",
		Output: "stdout",
	})

	msgBus := bus.New(100, log)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start message bus
	if err := msgBus.Start(ctx); err != nil {
		t.Fatalf("Failed to start message bus: %v", err)
	}

	cfg := config.TelegramConfig{
		AllowedUsers: []string{"123"}, // User 456 is not in the list
	}

	conn := New(cfg, log, msgBus)
	conn.ctx = ctx

	// Create a minimal mock bot to avoid nil pointer
	// Note: We can't fully mock telego.Bot, so we'll skip the notification test
	// The notification sending is handled by Telegram API and can't be easily unit tested
	// Integration tests would be needed for full flow testing
	// conn.bot = nil // Keep nil, the notification will error but won't panic

	// Subscribe to inbound messages
	inboundCh := msgBus.SubscribeInbound(ctx)

	// Create a test update with non-authorized user
	update := telego.Update{
		Message: &telego.Message{
			MessageID: 1,
			From: &telego.User{
				ID:        456,
				FirstName: "UnauthorizedUser",
			},
			Chat: telego.Chat{
				ID:   987654321,
				Type: "private",
			},
			Text: "Hello, bot!",
		},
	}

	// Handle the update
	err := conn.handleUpdate(update)
	if err != nil {
		t.Fatalf("handleUpdate() failed: %v", err)
	}

	// Should not receive any inbound message
	select {
	case msg := <-inboundCh:
		t.Errorf("Received unexpected message: %+v", msg)
	case <-time.After(100 * time.Millisecond):
		// Expected: no message received
	}

	t.Cleanup(func() {
		require.NoError(t, msgBus.Stop())
	})
}

// TestConnector_Stop tests graceful shutdown
func TestConnector_Stop(t *testing.T) {
	log, _ := logger.New(logger.Config{
		Level:  "debug",
		Format: "text",
		Output: "stdout",
	})

	msgBus := bus.New(100, log)

	cfg := config.TelegramConfig{
		Enabled: true,
		Token:   "test-token",
	}

	conn := New(cfg, log, msgBus)

	ctx := context.Background()
	conn.ctx, conn.cancel = context.WithCancel(ctx)

	// Set mock bot (methods won't be called in this test)
	mockBot := NewMockBotSuccess()
	conn.bot = mockBot
	outboundCh := make(chan bus.OutboundMessage)
	conn.outboundCh = outboundCh

	err := conn.Stop()
	if err != nil {
		t.Fatalf("Stop() failed: %v", err)
	}

	if conn.bot != nil {
		t.Error("Bot should be nil after Stop")
	}

	if conn.outboundCh != nil {
		t.Error("Outbound channel should be nil after Stop")
	}
}

// TestConnector_handleOutbound_Basic tests outbound message handling (basic)
func TestConnector_handleOutbound_Basic(t *testing.T) {
	log, _ := logger.New(logger.Config{
		Level:  "debug",
		Format: "text",
		Output: "stdout",
	})

	msgBus := bus.New(100, log)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := config.TelegramConfig{}

	conn := New(cfg, log, msgBus)
	conn.ctx = ctx

	// Create outbound channel
	outboundCh := make(chan bus.OutboundMessage, 1)
	conn.outboundCh = outboundCh

	// Start outbound handler in goroutine
	go conn.handleOutbound()

	// Send telegram message (will fail due to nil bot, but tests flow)
	outboundMsg := bus.OutboundMessage{
		ChannelType: bus.ChannelTypeTelegram,
		UserID:      "123456789",
		SessionID:   "telegram:987654321",
		Content:     "Hello from bot!",
		Timestamp:   time.Now(),
	}

	outboundCh <- outboundMsg

	// Wait a bit for processing
	time.Sleep(100 * time.Millisecond)

	// Stop handler
	cancel()
	time.Sleep(100 * time.Millisecond)
}

// TestConnector_handleOutbound_NonTelegramMessage tests that non-telegram messages are ignored
func TestConnector_handleOutbound_NonTelegramMessage(t *testing.T) {
	log, _ := logger.New(logger.Config{
		Level:  "debug",
		Format: "text",
		Output: "stdout",
	})

	msgBus := bus.New(100, log)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := config.TelegramConfig{}

	conn := New(cfg, log, msgBus)
	conn.ctx = ctx

	// Create outbound channel
	outboundCh := make(chan bus.OutboundMessage, 1)
	conn.outboundCh = outboundCh

	// Start outbound handler in goroutine
	go conn.handleOutbound()

	// Send non-telegram message
	outboundMsg := bus.OutboundMessage{
		ChannelType: bus.ChannelTypeDiscord, // Wrong channel type
		UserID:      "123456789",
		SessionID:   "telegram:987654321",
		Content:     "Hello from bot!",
		Timestamp:   time.Now(),
	}

	outboundCh <- outboundMsg

	// Wait for message to be processed
	time.Sleep(100 * time.Millisecond)

	// Stop handler
	cancel()
	time.Sleep(100 * time.Millisecond)
}

// TestConnector_handleUpdate_NewCommand tests /new command handling
func TestConnector_handleUpdate_NewCommand(t *testing.T) {
	log, _ := logger.New(logger.Config{
		Level:  "debug",
		Format: "text",
		Output: "stdout",
	})

	msgBus := bus.New(100, log)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start message bus
	if err := msgBus.Start(ctx); err != nil {
		t.Fatalf("Failed to start message bus: %v", err)
	}

	cfg := config.TelegramConfig{
		AllowedUsers: []string{"123456789"},
	}

	conn := New(cfg, log, msgBus)
	conn.ctx = ctx

	// Subscribe to inbound messages
	inboundCh := msgBus.SubscribeInbound(ctx)

	// Create a test update with /new command
	update := telego.Update{
		Message: &telego.Message{
			MessageID: 1,
			From: &telego.User{
				ID:        123456789,
				FirstName: "TestUser",
				Username:  "test_user",
			},
			Chat: telego.Chat{
				ID:   987654321,
				Type: "private",
			},
			Text: "/new",
		},
	}

	// Handle the update
	err := conn.handleUpdate(update)
	if err != nil {
		t.Fatalf("handleUpdate() failed: %v", err)
	}

	// Wait for inbound message
	select {
	case msg := <-inboundCh:
		if msg.ChannelType != bus.ChannelTypeTelegram {
			t.Errorf("Expected channel type telegram, got %s", msg.ChannelType)
		}

		if msg.UserID != "123456789" {
			t.Errorf("Expected user ID '123456789', got '%s'", msg.UserID)
		}

		if msg.SessionID != "telegram:987654321" {
			t.Errorf("Expected session ID 'telegram:987654321', got '%s'", msg.SessionID)
		}

		if msg.Content != "/new" {
			t.Errorf("Expected content '/new', got '%s'", msg.Content)
		}

		// Check metadata for command
		if cmd, ok := msg.Metadata["command"].(string); !ok || cmd != "new_session" {
			t.Errorf("Expected command 'new_session' in metadata, got %v", msg.Metadata["command"])
		}

		if msg.Metadata["message_id"] != 1 {
			t.Errorf("Expected message_id 1, got %v", msg.Metadata["message_id"])
		}
		if msg.Metadata["chat_id"] != int64(987654321) {
			t.Errorf("Expected chat_id 987654321, got %v", msg.Metadata["chat_id"])
		}
		if msg.Metadata["chat_type"] != "private" {
			t.Errorf("Expected chat_type private, got %v", msg.Metadata["chat_type"])
		}
		if msg.Metadata["username"] != "test_user" {
			t.Errorf("Expected username test_user, got %v", msg.Metadata["username"])
		}

	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for inbound message")
	}

	t.Cleanup(func() {
		require.NoError(t, msgBus.Stop())
	})
}

// TestConnector_handleUpdate_NewCommand_Unauthorized tests that /new command is blocked for unauthorized users
func TestConnector_handleUpdate_NewCommand_Unauthorized(t *testing.T) {
	log, _ := logger.New(logger.Config{
		Level:  "debug",
		Format: "text",
		Output: "stdout",
	})

	msgBus := bus.New(100, log)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start message bus
	if err := msgBus.Start(ctx); err != nil {
		t.Fatalf("Failed to start message bus: %v", err)
	}

	cfg := config.TelegramConfig{
		AllowedUsers: []string{"123"}, // User 456 is not in the list
	}

	conn := New(cfg, log, msgBus)
	conn.ctx = ctx

	// Subscribe to inbound messages
	inboundCh := msgBus.SubscribeInbound(ctx)

	// Create a test update with /new command from unauthorized user
	update := telego.Update{
		Message: &telego.Message{
			MessageID: 1,
			From: &telego.User{
				ID:        456,
				FirstName: "UnauthorizedUser",
			},
			Chat: telego.Chat{
				ID:   987654321,
				Type: "private",
			},
			Text: "/new",
		},
	}

	// Handle the update
	err := conn.handleUpdate(update)
	if err != nil {
		t.Fatalf("handleUpdate() failed: %v", err)
	}

	// Should not receive any inbound message
	select {
	case msg := <-inboundCh:
		t.Errorf("Received unexpected message: %+v", msg)
	case <-time.After(100 * time.Millisecond):
		// Expected: no message received
	}

	t.Cleanup(func() {
		require.NoError(t, msgBus.Stop())
	})
}

// TestConnector_handleUpdate_NewCommand_ThenRegularMessage tests that regular messages work after /new command
func TestConnector_handleUpdate_NewCommand_ThenRegularMessage(t *testing.T) {
	log, _ := logger.New(logger.Config{
		Level:  "debug",
		Format: "text",
		Output: "stdout",
	})

	msgBus := bus.New(100, log)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start message bus
	if err := msgBus.Start(ctx); err != nil {
		t.Fatalf("Failed to start message bus: %v", err)
	}

	cfg := config.TelegramConfig{
		AllowedUsers: []string{"123456789"},
	}

	conn := New(cfg, log, msgBus)
	conn.ctx = ctx

	// Subscribe to inbound messages
	inboundCh := msgBus.SubscribeInbound(ctx)

	// First, send /new command
	newCmdUpdate := telego.Update{
		Message: &telego.Message{
			MessageID: 1,
			From: &telego.User{
				ID:        123456789,
				FirstName: "TestUser",
			},
			Chat: telego.Chat{
				ID:   987654321,
				Type: "private",
			},
			Text: "/new",
		},
	}

	err := conn.handleUpdate(newCmdUpdate)
	if err != nil {
		t.Fatalf("handleUpdate() for /new command failed: %v", err)
	}

	// Verify /new command message
	select {
	case msg := <-inboundCh:
		if cmd, ok := msg.Metadata["command"].(string); !ok || cmd != "new_session" {
			t.Errorf("Expected command 'new_session', got %v", msg.Metadata["command"])
		}
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for /new command message")
	}

	// Then, send a regular message
	regularMsgUpdate := telego.Update{
		Message: &telego.Message{
			MessageID: 2,
			From: &telego.User{
				ID:        123456789,
				FirstName: "TestUser",
			},
			Chat: telego.Chat{
				ID:   987654321,
				Type: "private",
			},
			Text: "Hello, bot!",
		},
	}

	err = conn.handleUpdate(regularMsgUpdate)
	if err != nil {
		t.Fatalf("handleUpdate() for regular message failed: %v", err)
	}

	// Verify regular message doesn't have command metadata
	select {
	case msg := <-inboundCh:
		if msg.Content != "Hello, bot!" {
			t.Errorf("Expected content 'Hello, bot!', got '%s'", msg.Content)
		}
		if cmd, ok := msg.Metadata["command"]; ok {
			t.Errorf("Regular message should not have command metadata, got %v", cmd)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for regular message")
	}

	t.Cleanup(func() {
		require.NoError(t, msgBus.Stop())
	})
}

// TestConnector_handleEvents tests event handling for typing indicator
func TestConnector_handleEvents(t *testing.T) {
	log, _ := logger.New(logger.Config{
		Level:  "debug",
		Format: "text",
		Output: "stdout",
	})

	msgBus := bus.New(100, log)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := config.TelegramConfig{}
	conn := New(cfg, log, msgBus)
	conn.ctx = ctx

	// Create event channel
	eventCh := make(chan bus.Event, 10)
	conn.eventCh = eventCh

	// Start event handler in goroutine
	go conn.handleEvents()

	// Send processing start event
	startEvent := bus.NewProcessingStartEvent(
		bus.ChannelTypeTelegram,
		"123456789",
		"987654321",
		map[string]any{"chat_id": int64(987654321)},
	)

	eventCh <- *startEvent

	// Wait a bit for processing
	time.Sleep(100 * time.Millisecond)

	// Send processing end event
	endEvent := bus.NewProcessingEndEvent(
		bus.ChannelTypeTelegram,
		"123456789",
		"987654321",
		nil,
	)

	eventCh <- *endEvent

	// Stop handler
	cancel()
	time.Sleep(100 * time.Millisecond)
}

// TestConnector_handleEvents_NonTelegram tests that non-Telegram events are ignored
func TestConnector_handleEvents_NonTelegram(t *testing.T) {
	log, _ := logger.New(logger.Config{
		Level:  "debug",
		Format: "text",
		Output: "stdout",
	})

	msgBus := bus.New(100, log)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := config.TelegramConfig{}
	conn := New(cfg, log, msgBus)
	conn.ctx = ctx

	// Create event channel
	eventCh := make(chan bus.Event, 10)
	conn.eventCh = eventCh

	// Start event handler in goroutine
	go conn.handleEvents()

	// Send non-Telegram processing start event (Discord)
	startEvent := bus.NewProcessingStartEvent(
		bus.ChannelTypeDiscord, // Wrong channel type
		"123456789",
		"987654321",
		nil,
	)

	eventCh <- *startEvent

	// Wait for processing
	time.Sleep(100 * time.Millisecond)

	// Stop handler
	cancel()
	time.Sleep(100 * time.Millisecond)
}

// TestConnector_sendTypingIndicator_NilBot tests typing indicator with nil bot
func TestConnector_sendTypingIndicator_NilBot(t *testing.T) {
	log, _ := logger.New(logger.Config{
		Level:  "debug",
		Format: "text",
		Output: "stdout",
	})

	msgBus := bus.New(100, log)

	cfg := config.TelegramConfig{}
	conn := New(cfg, log, msgBus)
	conn.ctx = context.Background()
	conn.bot = nil // Set bot to nil

	// Send typing indicator event
	event := bus.NewProcessingStartEvent(
		bus.ChannelTypeTelegram,
		"123456789",
		"987654321",
		nil,
	)

	// Should not panic, just log warning
	conn.sendTypingIndicator(*event)
}

// TestConnector_sendTypingIndicator_InvalidSessionID tests typing indicator with invalid session ID
func TestConnector_sendTypingIndicator_InvalidSessionID(t *testing.T) {
	log, _ := logger.New(logger.Config{
		Level:  "debug",
		Format: "text",
		Output: "stdout",
	})

	msgBus := bus.New(100, log)

	cfg := config.TelegramConfig{}
	conn := New(cfg, log, msgBus)
	conn.ctx = context.Background()

	// Send typing indicator event with invalid session ID
	event := bus.NewProcessingStartEvent(
		bus.ChannelTypeTelegram,
		"123456789",
		"invalid-session-id", // Invalid - not a number
		nil,
	)

	// Should not panic, just log error
	conn.sendTypingIndicator(*event)
}

// TestConnector_validateConfig tests configuration validation
func TestConnector_validateConfig(t *testing.T) {
	log, _ := logger.New(logger.Config{
		Level:  "debug",
		Format: "text",
		Output: "stdout",
	})

	msgBus := bus.New(100, log)

	tests := []struct {
		name      string
		cfg       config.TelegramConfig
		expectErr bool
	}{
		{
			name: "valid config",
			cfg: config.TelegramConfig{
				Token: "valid-token",
			},
			expectErr: false,
		},
		{
			name: "empty token",
			cfg: config.TelegramConfig{
				Token: "",
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conn := New(tt.cfg, log, msgBus)
			err := conn.validateConfig()

			if (err != nil) != tt.expectErr {
				t.Errorf("validateConfig() error = %v, expectErr %v", err, tt.expectErr)
			}
		})
	}
}

// TestConnector_registerCommands tests command registration with Telegram
func TestConnector_registerCommands(t *testing.T) {
	log, _ := logger.New(logger.Config{
		Level:  "debug",
		Format: "text",
		Output: "stdout",
	})

	msgBus := bus.New(100, log)

	cfg := config.TelegramConfig{}
	conn := New(cfg, log, msgBus)
	conn.ctx = context.Background()

	// Test with nil bot - should return error without panic
	err := conn.registerCommands()
	if err == nil {
		t.Error("Expected error when bot is nil")
	}
}

// TestLongPollManager_Start_WithMock tests LongPollManager with mock bot
func TestLongPollManager_Start_WithMock(t *testing.T) {
	log, _ := logger.New(logger.Config{
		Level:  "debug",
		Format: "text",
		Output: "stdout",
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create test update
	update := telego.Update{
		Message: &telego.Message{
			MessageID: 1,
			From: &telego.User{
				ID:        123456789,
				FirstName: "TestUser",
			},
			Chat: telego.Chat{
				ID:   987654321,
				Type: "private",
			},
			Text: "Hello, bot!",
		},
	}

	// Create mock bot with update
	mockBot, _ := NewMockBotWithUpdates(update)
	defer mockBot.AssertExpectations(t)

	// Create LongPollManager
	msgBus := bus.New(100, log)
	cfg := config.TelegramConfig{
		AllowedUsers: []string{"123456789"},
	}

	conn := New(cfg, log, msgBus)
	conn.ctx = ctx

	lpm := NewLongPollManager(conn, mockBot, log)
	lpm.SetContext(ctx)

	// Start long polling in goroutine (should process the update and exit)
	done := make(chan bool)
	go func() {
		lpm.Start()
		done <- true
	}()

	// Wait for long polling to complete (channel is closed after processing all updates)
	select {
	case <-done:
		// Long polling completed
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for long polling to complete")
	}

	// Verify that UpdatesViaLongPolling was called
	mockBot.AssertExpectations(t)
}

// TestConnector_handleOutbound_NewFormat tests session_id parsing with new "channel:chat_id" format
func TestConnector_handleOutbound_NewFormat(t *testing.T) {
	log, _ := logger.New(logger.Config{
		Level:  "debug",
		Format: "text",
		Output: "stdout",
	})

	msgBus := bus.New(100, log)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := config.TelegramConfig{
		SendTimeoutSeconds: 5,
	}

	conn := New(cfg, log, msgBus)
	conn.ctx = ctx

	// Create mock bot
	mockBot := NewMockBotSuccess()
	conn.bot = mockBot

	// Create outbound channel
	outboundCh := make(chan bus.OutboundMessage, 1)
	conn.outboundCh = outboundCh

	// Start outbound handler in goroutine
	go conn.handleOutbound()

	// Send telegram message with new session_id format
	outboundMsg := bus.OutboundMessage{
		ChannelType: bus.ChannelTypeTelegram,
		UserID:      "123456789",
		SessionID:   "telegram:987654321", // New format
		Type:        bus.MessageTypeText,
		Content:     "Hello from bot!",
		Timestamp:   time.Now(),
	}

	outboundCh <- outboundMsg

	// Wait a bit for processing
	time.Sleep(100 * time.Millisecond)

	// Stop handler
	cancel()
	time.Sleep(100 * time.Millisecond)

	// Verify bot was called with correct chat ID
	mockBot.AssertCalled(t, "SendMessage", mock.Anything, mock.MatchedBy(func(params *telego.SendMessageParams) bool {
		return params.ChatID.ID == 987654321 && params.Text == "Hello from bot!"
	}))
	mockBot.AssertCalled(t, "SendMessage", mock.Anything, mock.MatchedBy(func(params *telego.SendMessageParams) bool {
		return params.ChatID.ID == 987654321 && params.Text == "Hello from bot!"
	}))
}

// TestConnector_handleOutbound_InvalidNewFormat tests session_id with invalid new format
func TestConnector_handleOutbound_InvalidNewFormat(t *testing.T) {
	log, _ := logger.New(logger.Config{
		Level:  "debug",
		Format: "text",
		Output: "stdout",
	})

	msgBus := bus.New(100, log)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := config.TelegramConfig{}

	conn := New(cfg, log, msgBus)
	conn.ctx = ctx

	// Create mock bot
	mockBot := NewMockBotSuccess()
	conn.bot = mockBot

	// Create outbound channel
	outboundCh := make(chan bus.OutboundMessage, 1)
	conn.outboundCh = outboundCh

	// Start outbound handler in goroutine
	go conn.handleOutbound()

	// Send telegram message with invalid new format (missing second part)
	outboundMsg := bus.OutboundMessage{
		ChannelType: bus.ChannelTypeTelegram,
		UserID:      "123456789",
		SessionID:   "telegram:", // Invalid - missing chat_id
		Content:     "Hello from bot!",
		Timestamp:   time.Now(),
	}

	outboundCh <- outboundMsg

	// Wait a bit for processing
	time.Sleep(100 * time.Millisecond)

	// Stop handler
	cancel()
	time.Sleep(100 * time.Millisecond)

	// Verify bot was NOT called because session_id is invalid
	mockBot.AssertNotCalled(t, "SendMessage")
}

// TestConnector_handleOutbound_ChannelMismatch tests session_id with non-telegram channel
func TestConnector_handleOutbound_ChannelMismatch(t *testing.T) {
	log, _ := logger.New(logger.Config{
		Level:  "debug",
		Format: "text",
		Output: "stdout",
	})

	msgBus := bus.New(100, log)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := config.TelegramConfig{}

	conn := New(cfg, log, msgBus)
	conn.ctx = ctx

	// Create mock bot
	mockBot := NewMockBotSuccess()
	conn.bot = mockBot

	// Create outbound channel
	outboundCh := make(chan bus.OutboundMessage, 1)
	conn.outboundCh = outboundCh

	// Start outbound handler in goroutine
	go conn.handleOutbound()

	// Send telegram message with discord channel in session_id
	outboundMsg := bus.OutboundMessage{
		ChannelType: bus.ChannelTypeTelegram,
		UserID:      "123456789",
		SessionID:   "discord:987654321", // Wrong channel
		Content:     "Hello from bot!",
		Timestamp:   time.Now(),
	}

	outboundCh <- outboundMsg

	// Wait a bit for processing
	time.Sleep(100 * time.Millisecond)

	// Stop handler
	cancel()
	time.Sleep(100 * time.Millisecond)

	// Verify bot was NOT called because channel mismatch
	mockBot.AssertNotCalled(t, "SendMessage")
}

// TestConnector_handleOutbound_InvalidChatID tests session_id with invalid chat ID
func TestConnector_handleOutbound_InvalidChatID(t *testing.T) {
	log, _ := logger.New(logger.Config{
		Level:  "debug",
		Format: "text",
		Output: "stdout",
	})

	msgBus := bus.New(100, log)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := config.TelegramConfig{}

	conn := New(cfg, log, msgBus)
	conn.ctx = ctx

	// Create mock bot
	mockBot := NewMockBotSuccess()
	conn.bot = mockBot

	// Create outbound channel
	outboundCh := make(chan bus.OutboundMessage, 1)
	conn.outboundCh = outboundCh

	// Start outbound handler in goroutine
	go conn.handleOutbound()

	// Send telegram message with invalid chat_id
	outboundMsg := bus.OutboundMessage{
		ChannelType: bus.ChannelTypeTelegram,
		UserID:      "123456789",
		SessionID:   "telegram:not_a_number", // Invalid chat_id
		Content:     "Hello from bot!",
		Timestamp:   time.Now(),
	}

	outboundCh <- outboundMsg

	// Wait a bit for processing
	time.Sleep(100 * time.Millisecond)

	// Stop handler
	cancel()
	time.Sleep(100 * time.Millisecond)

	// Verify bot was NOT called because chat_id is invalid
	mockBot.AssertNotCalled(t, "SendMessage")
}

// TestTypingManager_Start_WithMock tests TypingManager start with mock bot
func TestTypingManager_Start_WithMock(t *testing.T) {
	log, _ := logger.New(logger.Config{
		Level:  "debug",
		Format: "text",
		Output: "stdout",
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create mock bot
	mockBot := NewMockBotSuccess()
	defer mockBot.AssertExpectations(t)

	// Create TypingManager
	tm := NewTypingManager(mockBot, log)
	tm.SetContext(ctx)

	// Create typing event
	event := bus.NewProcessingStartEvent(
		bus.ChannelTypeTelegram,
		"123456789",
		"telegram:987654321",
		nil,
	)

	// Start typing indicator (sends immediately and then periodically)
	tm.Start(*event)

	// Wait a bit to ensure the first indicator was sent
	time.Sleep(100 * time.Millisecond)

	// Verify that SendChatAction was called at least once
	mockBot.AssertCalled(t, "SendChatAction", ctx, mock.MatchedBy(func(params *telego.SendChatActionParams) bool {
		return params.ChatID.ID == 987654321 && params.Action == telego.ChatActionTyping
	}))

	// Stop typing indicator
	tm.Stop(*event)

	// Wait a bit to ensure goroutine stopped
	time.Sleep(100 * time.Millisecond)

	mockBot.AssertExpectations(t)
}
