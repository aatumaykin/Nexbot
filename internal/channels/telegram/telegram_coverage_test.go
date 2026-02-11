package telegram

import (
	"context"
	"testing"
	"time"

	"github.com/aatumaykin/nexbot/internal/bus"
	"github.com/aatumaykin/nexbot/internal/config"
	"github.com/aatumaykin/nexbot/internal/logger"
	"github.com/mymmrac/telego"
	"github.com/stretchr/testify/require"
)

// TestConnector_sendStartupMessage tests sending startup messages to allowed users
func TestConnector_sendStartupMessage(t *testing.T) {
	log, _ := logger.New(logger.Config{
		Level:  "debug",
		Format: "text",
		Output: "stdout",
	})

	t.Run("no allowed users", func(t *testing.T) {
		cfg := config.TelegramConfig{
			AllowedUsers: []string{},
		}

		conn := New(cfg, log, nil)
		conn.ctx = context.Background()

		// Should not error when no users are configured
		err := conn.sendStartupMessage()
		if err != nil {
			t.Errorf("Expected no error when whitelist is empty, got: %v", err)
		}
	})

	t.Run("only invalid user IDs", func(t *testing.T) {
		cfg := config.TelegramConfig{
			AllowedUsers: []string{"not-a-number", "also-not-a-number"},
		}

		conn := New(cfg, log, nil)
		conn.ctx = context.Background()

		// Should not error even with invalid user IDs (no valid IDs to send to)
		err := conn.sendStartupMessage()
		if err != nil {
			t.Errorf("Expected no error with only invalid user IDs, got: %v", err)
		}
	})

	t.Run("mixed valid and invalid user IDs", func(t *testing.T) {
		cfg := config.TelegramConfig{
			AllowedUsers: []string{"not-a-number", "123", "also-not-a-number"},
		}

		conn := New(cfg, log, nil)
		conn.ctx = context.Background()
		conn.bot = &telego.Bot{} // Set bot but it will panic on send

		// Should not panic even though bot will fail to send
		defer func() {
			if r := recover(); r != nil {
				// Expected - bot is not properly initialized
				t.Logf("Recovered expected panic: %v", r)
			}
		}()

		err := conn.sendStartupMessage()
		// Function logs errors but doesn't return them
		_ = err
	})
}

// TestConnector_registerCommands_NoAllowedUsers tests command registration with empty whitelist
func TestConnector_registerCommands_NoAllowedUsers(t *testing.T) {
	log, _ := logger.New(logger.Config{
		Level:  "debug",
		Format: "text",
		Output: "stdout",
	})

	msgBus := bus.New(100, 10, log)

	cfg := config.TelegramConfig{
		AllowedUsers: []string{}, // Empty whitelist
	}

	conn := New(cfg, log, msgBus)
	conn.ctx = context.Background()

	// Test with nil bot - should return error
	err := conn.registerCommands()
	if err == nil {
		t.Error("Expected error when bot is nil")
	}
}

// TestConnector_handleUpdate_StatusCommand_EmptyWhitelist tests /status with empty whitelist
func TestConnector_handleUpdate_StatusCommand_EmptyWhitelist(t *testing.T) {
	log, _ := logger.New(logger.Config{
		Level:  "debug",
		Format: "text",
		Output: "stdout",
	})

	msgBus := bus.New(100, 10, log)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := msgBus.Start(ctx); err != nil {
		t.Fatalf("Failed to start message bus: %v", err)
	}

	cfg := config.TelegramConfig{
		AllowedUsers: []string{}, // Empty whitelist allows all
	}

	conn := New(cfg, log, msgBus)
	conn.ctx = ctx

	// Subscribe to inbound messages
	inboundCh := msgBus.SubscribeInbound(ctx)

	// Create a test update with /status command
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
			Text: "/status",
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
		if cmd, ok := msg.Metadata["command"].(string); !ok || cmd != "status" {
			t.Errorf("Expected command 'status' in metadata, got %v", msg.Metadata["command"])
		}
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for /status command message")
	}

	t.Cleanup(func() {
		require.NoError(t, msgBus.Stop())
	})
}

// TestConnector_handleUpdate_RestartCommand_EmptyWhitelist tests /restart with empty whitelist
func TestConnector_handleUpdate_RestartCommand_EmptyWhitelist(t *testing.T) {
	log, _ := logger.New(logger.Config{
		Level:  "debug",
		Format: "text",
		Output: "stdout",
	})

	msgBus := bus.New(100, 10, log)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := msgBus.Start(ctx); err != nil {
		t.Fatalf("Failed to start message bus: %v", err)
	}

	cfg := config.TelegramConfig{
		AllowedUsers: []string{}, // Empty whitelist allows all
	}

	conn := New(cfg, log, msgBus)
	conn.ctx = ctx

	// Subscribe to inbound messages
	inboundCh := msgBus.SubscribeInbound(ctx)

	// Create a test update with /restart command
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
			Text: "/restart",
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
		if cmd, ok := msg.Metadata["command"].(string); !ok || cmd != "restart" {
			t.Errorf("Expected command 'restart' in metadata, got %v", msg.Metadata["command"])
		}
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for /restart command message")
	}

	t.Cleanup(func() {
		require.NoError(t, msgBus.Stop())
	})
}

// TestConnector_handleUpdate_NewCommand_EmptyWhitelist tests /new with empty whitelist
func TestConnector_handleUpdate_NewCommand_EmptyWhitelist(t *testing.T) {
	log, _ := logger.New(logger.Config{
		Level:  "debug",
		Format: "text",
		Output: "stdout",
	})

	msgBus := bus.New(100, 10, log)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := msgBus.Start(ctx); err != nil {
		t.Fatalf("Failed to start message bus: %v", err)
	}

	cfg := config.TelegramConfig{
		AllowedUsers: []string{}, // Empty whitelist allows all
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
		if cmd, ok := msg.Metadata["command"].(string); !ok || cmd != "new_session" {
			t.Errorf("Expected command 'new_session' in metadata, got %v", msg.Metadata["command"])
		}
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for /new command message")
	}

	t.Cleanup(func() {
		require.NoError(t, msgBus.Stop())
	})
}

// TestTypingManager_Start tests Start method with different scenarios
func TestTypingManager_Start(t *testing.T) {
	log, _ := logger.New(logger.Config{
		Level:  "debug",
		Format: "text",
		Output: "stdout",
	})

	t.Run("Start with nil context", func(t *testing.T) {
		tm := NewTypingManager(nil, log)
		// Don't set context - should use background context

		event := bus.NewProcessingStartEvent(
			bus.ChannelTypeTelegram,
			"user1",
			"123",
			nil,
		)

		// Should not panic
		tm.Start(*event)
		time.Sleep(50 * time.Millisecond)

		// Stop it
		endEvent := bus.NewProcessingEndEvent(
			bus.ChannelTypeTelegram,
			"user1",
			"123",
			nil,
		)
		tm.Stop(*endEvent)
	})

	t.Run("Start with same event multiple times", func(t *testing.T) {
		tm := NewTypingManager(nil, log)
		ctx := context.Background()
		tm.SetContext(ctx)

		event := bus.NewProcessingStartEvent(
			bus.ChannelTypeTelegram,
			"user1",
			"456",
			nil,
		)

		// Start typing first time
		tm.Start(*event)
		time.Sleep(50 * time.Millisecond)

		// Start typing again immediately (should not duplicate)
		tm.Start(*event)
		time.Sleep(50 * time.Millisecond)

		// Start typing third time (should still not duplicate)
		tm.Start(*event)
		time.Sleep(50 * time.Millisecond)

		// Stop it
		endEvent := bus.NewProcessingEndEvent(
			bus.ChannelTypeTelegram,
			"user1",
			"456",
			nil,
		)
		tm.Stop(*endEvent)
	})
}

// TestConnector_handleOutbound tests various outbound scenarios
func TestConnector_handleOutbound_Variations(t *testing.T) {
	log, _ := logger.New(logger.Config{
		Level:  "debug",
		Format: "text",
		Output: "stdout",
	})

	msgBus := bus.New(100, 10, log)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := config.TelegramConfig{}

	conn := New(cfg, log, msgBus)
	conn.ctx = ctx

	// Create outbound channel
	outboundCh := make(chan bus.OutboundMessage, 3)
	conn.outboundCh = outboundCh

	// Start outbound handler in goroutine
	go conn.handleOutbound()

	// Send multiple different types of messages
	messages := []bus.OutboundMessage{
		{
			ChannelType: bus.ChannelTypeTelegram,
			UserID:      "user1",
			SessionID:   "telegram:111",
			Content:     "Message 1",
			Timestamp:   time.Now(),
		},
		{
			ChannelType: bus.ChannelTypeTelegram,
			UserID:      "user2",
			SessionID:   "telegram:222",
			Content:     "Message 2 with emoji 🎉",
			Timestamp:   time.Now(),
		},
		{
			ChannelType: bus.ChannelTypeTelegram,
			UserID:      "user3",
			SessionID:   "telegram:333",
			Content:     "Message 3\nwith newlines",
			Timestamp:   time.Now(),
		},
	}

	for _, msg := range messages {
		outboundCh <- msg
	}

	// Wait a bit for processing
	time.Sleep(200 * time.Millisecond)

	// Stop handler
	cancel()
	time.Sleep(100 * time.Millisecond)
}

// TestCommandHandler_HandleCommand_StoppedBus tests command handling when message bus is stopped
func TestCommandHandler_HandleCommand_StoppedBus(t *testing.T) {
	log, _ := logger.New(logger.Config{
		Level:  "debug",
		Format: "text",
		Output: "stdout",
	})

	msgBus := bus.New(100, 10, log)

	// Don't start the bus - PublishInbound should fail
	cmdHandler := NewCommandHandler(log, msgBus)

	msg := &telego.Message{
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
		Text: "/status",
	}

	ctx := context.Background()
	isAllowedFunc := func(userID string) bool { return true }

	// Handle command - should return error because bus is not started
	err := cmdHandler.HandleCommand(ctx, isAllowedFunc, msg, "status", "123456789")
	if err == nil {
		t.Error("Expected error when message bus is not started")
	}

	if err != nil && err.Error() != "failed to publish command message: message bus is not started" {
		t.Logf("Got error: %v", err)
	}
}

// TestConnector_handleUpdate_SpacesOnly tests handling of message with only spaces
func TestConnector_handleUpdate_SpacesOnly(t *testing.T) {
	log, _ := logger.New(logger.Config{
		Level:  "debug",
		Format: "text",
		Output: "stdout",
	})

	msgBus := bus.New(100, 10, log)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := msgBus.Start(ctx); err != nil {
		t.Fatalf("Failed to start message bus: %v", err)
	}

	cfg := config.TelegramConfig{
		AllowedUsers: []string{"123456789"},
	}

	conn := New(cfg, log, msgBus)
	conn.ctx = ctx

	// Create an update with spaces
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
			Text: "   ", // Only spaces
		},
	}

	// Handle the update
	err := conn.handleUpdate(update)
	if err != nil {
		t.Errorf("handleUpdate() with spaces should return nil, got %v", err)
	}

	t.Cleanup(func() {
		require.NoError(t, msgBus.Stop())
	})
}

// TestTypingManager_Send_NilContext tests Send with nil context
func TestTypingManager_Send_NilContext(t *testing.T) {
	log, _ := logger.New(logger.Config{
		Level:  "debug",
		Format: "text",
		Output: "stdout",
	})

	t.Run("Send with nil context", func(t *testing.T) {
		tm := NewTypingManager(nil, log)
		// Don't set context - should use background context

		event := bus.NewProcessingStartEvent(
			bus.ChannelTypeTelegram,
			"user1",
			"123",
			nil,
		)

		// Should not panic with nil context
		tm.Send(*event)
	})
}

// TestConnector_handleOutbound_VeryLongMessage tests handling of very long messages
func TestConnector_handleOutbound_VeryLongMessage(t *testing.T) {
	log, _ := logger.New(logger.Config{
		Level:  "debug",
		Format: "text",
		Output: "stdout",
	})

	msgBus := bus.New(100, 10, log)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := config.TelegramConfig{}

	conn := New(cfg, log, msgBus)
	conn.ctx = ctx
	conn.bot = nil // Set bot to nil

	// Create outbound channel
	outboundCh := make(chan bus.OutboundMessage, 1)
	conn.outboundCh = outboundCh

	// Start outbound handler in goroutine
	go conn.handleOutbound()

	// Send very long message
	longContent := make([]byte, 5000)
	for i := range longContent {
		longContent[i] = 'A' + byte(i%26)
	}

	outboundMsg := bus.OutboundMessage{
		ChannelType: bus.ChannelTypeTelegram,
		UserID:      "123456789",
		SessionID:   "telegram:987654321",
		Content:     string(longContent),
		Timestamp:   time.Now(),
	}

	outboundCh <- outboundMsg

	// Wait a bit for processing
	time.Sleep(100 * time.Millisecond)

	// Stop handler
	cancel()
	time.Sleep(100 * time.Millisecond)
}

// TestConnector_handleUpdate_SpecialCharacters tests handling of messages with special characters
func TestConnector_handleUpdate_SpecialCharacters(t *testing.T) {
	log, _ := logger.New(logger.Config{
		Level:  "debug",
		Format: "text",
		Output: "stdout",
	})

	msgBus := bus.New(100, 10, log)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

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

	// Create a test update with special characters
	specialContent := "Hello! @#$%^&*()_+-=[]{}|;':\",./<>?~`\n\t\r"

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
			Text: specialContent,
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
		if msg.Content != specialContent {
			t.Errorf("Expected content '%s', got '%s'", specialContent, msg.Content)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for inbound message")
	}

	t.Cleanup(func() {
		require.NoError(t, msgBus.Stop())
	})
}

// TestTypingManager_Start_CancelledContext tests Start with already cancelled context
func TestTypingManager_Start_CancelledContext(t *testing.T) {
	log, _ := logger.New(logger.Config{
		Level:  "debug",
		Format: "text",
		Output: "stdout",
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	tm := NewTypingManager(nil, log)
	tm.SetContext(ctx)

	event := bus.NewProcessingStartEvent(
		bus.ChannelTypeTelegram,
		"user1",
		"123",
		nil,
	)

	// Should not panic even with cancelled context
	tm.Start(*event)
	time.Sleep(50 * time.Millisecond)

	// Try to stop (should not error)
	endEvent := bus.NewProcessingEndEvent(
		bus.ChannelTypeTelegram,
		"user1",
		"123",
		nil,
	)
	tm.Stop(*endEvent)
}

// TestConnector_isAllowedUser_TableDriven tests isAllowedUser with various scenarios
func TestConnector_isAllowedUser_TableDriven(t *testing.T) {
	log, _ := logger.New(logger.Config{
		Level:  "debug",
		Format: "text",
		Output: "stdout",
	})

	msgBus := bus.New(100, 10, log)

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
			allowed:    []string{"123", "456", "789"},
			userID:     "456",
			shouldPass: true,
		},
		{
			name:       "user not in whitelist",
			allowed:    []string{"123", "456"},
			userID:     "789",
			shouldPass: false,
		},
		{
			name:       "empty user ID with non-empty whitelist",
			allowed:    []string{"123", "456"},
			userID:     "",
			shouldPass: false,
		},
		{
			name:       "string user ID in whitelist",
			allowed:    []string{"alice", "bob"},
			userID:     "alice",
			shouldPass: true,
		},
		{
			name:       "numeric user ID as string",
			allowed:    []string{"123456789"},
			userID:     "123456789",
			shouldPass: true,
		},
		{
			name:       "negative numeric user ID",
			allowed:    []string{"-123"},
			userID:     "-123",
			shouldPass: true,
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
				t.Errorf("isAllowedUser(%s) = %v, want %v (whitelist: %v)",
					tt.userID, result, tt.shouldPass, tt.allowed)
			}
		})
	}
}

// TestTypingManager_Stop tests Stop with various scenarios
func TestTypingManager_Stop(t *testing.T) {
	log, _ := logger.New(logger.Config{
		Level:  "debug",
		Format: "text",
		Output: "stdout",
	})

	t.Run("Stop non-existent typing", func(t *testing.T) {
		tm := NewTypingManager(nil, log)

		event := bus.NewProcessingEndEvent(
			bus.ChannelTypeTelegram,
			"user1",
			"999", // Not started
			nil,
		)

		// Should not panic when stopping non-existent typing
		tm.Stop(*event)
	})

	t.Run("Stop typing that was never started", func(t *testing.T) {
		tm := NewTypingManager(nil, log)

		// Try to stop without starting
		event := bus.NewProcessingEndEvent(
			bus.ChannelTypeTelegram,
			"user1",
			"123",
			nil,
		)

		tm.Stop(*event)
	})
}

// TestLongPollManager tests LongPollManager methods
func TestLongPollManager(t *testing.T) {
	log, _ := logger.New(logger.Config{
		Level:  "debug",
		Format: "text",
		Output: "stdout",
	})

	cfg := config.TelegramConfig{}
	conn := New(cfg, log, nil)

	t.Run("SetContext", func(t *testing.T) {
		lpm := conn.longPollManager
		if lpm == nil {
			t.Fatal("LongPollManager is nil")
		}

		testCtx := context.Background()
		lpm.SetContext(testCtx)

		if lpm.ctx != testCtx {
			t.Error("Context was not set correctly")
		}
	})

	t.Run("SetBot", func(t *testing.T) {
		lpm := conn.longPollManager
		if lpm == nil {
			t.Fatal("LongPollManager is nil")
		}

		// We can't create a real bot without a token, so we just test that the method doesn't panic
		lpm.SetBot(nil)

		if lpm.bot != nil {
			t.Error("Bot should be nil after setting nil")
		}
	})
}

// TestTypingManager tests TypingManager methods
func TestTypingManager(t *testing.T) {
	log, _ := logger.New(logger.Config{
		Level:  "debug",
		Format: "text",
		Output: "stdout",
	})

	_ = bus.New(100, 10, log)

	t.Run("SetContext", func(t *testing.T) {
		tm := NewTypingManager(nil, log)

		testCtx := context.Background()
		tm.SetContext(testCtx)

		if tm.ctx != testCtx {
			t.Error("Context was not set correctly")
		}
	})

	t.Run("Start with nil bot", func(t *testing.T) {
		tm := NewTypingManager(nil, log)

		ctx := context.Background()
		tm.SetContext(ctx)

		event := bus.NewProcessingStartEvent(
			bus.ChannelTypeTelegram,
			"123456789",
			"telegram:987654321",
			nil,
		)

		// Should not panic even with nil bot
		tm.Start(*event)

		// Wait a bit for the goroutine to start
		time.Sleep(50 * time.Millisecond)

		// Stop it
		endEvent := bus.NewProcessingEndEvent(
			bus.ChannelTypeTelegram,
			"123456789",
			"telegram:987654321",
			nil,
		)
		tm.Stop(*endEvent)
	})

	t.Run("Start and Stop typing indicator", func(t *testing.T) {
		tm := NewTypingManager(nil, log)

		ctx := context.Background()
		tm.SetContext(ctx)

		event := bus.NewProcessingStartEvent(
			bus.ChannelTypeTelegram,
			"123456789",
			"telegram:987654321",
			nil,
		)

		// Start typing
		tm.Start(*event)

		// Wait a bit
		time.Sleep(50 * time.Millisecond)

		// Check that typing cancel function is stored
		tm.typingLock.RLock()
		_, exists := tm.typingCancel[event.SessionID]
		tm.typingLock.RUnlock()

		if !exists {
			t.Error("Typing cancel function should be stored after Start")
		}

		// Stop typing
		endEvent := bus.NewProcessingEndEvent(
			bus.ChannelTypeTelegram,
			"123456789",
			"telegram:987654321",
			nil,
		)
		tm.Stop(*endEvent)

		// Check that typing cancel function is removed
		tm.typingLock.RLock()
		_, exists = tm.typingCancel[event.SessionID]
		tm.typingLock.RUnlock()

		if exists {
			t.Error("Typing cancel function should be removed after Stop")
		}
	})

	t.Run("Start already started typing", func(t *testing.T) {
		tm := NewTypingManager(nil, log)

		ctx := context.Background()
		tm.SetContext(ctx)

		event := bus.NewProcessingStartEvent(
			bus.ChannelTypeTelegram,
			"123456789",
			"telegram:987654321",
			nil,
		)

		// Start typing first time
		tm.Start(*event)
		time.Sleep(50 * time.Millisecond)

		// Start typing again (should not duplicate)
		tm.Start(*event)
		time.Sleep(50 * time.Millisecond)

		// Check that only one typing indicator exists
		tm.typingLock.RLock()
		count := 0
		for range tm.typingCancel {
			count++
		}
		tm.typingLock.RUnlock()

		if count != 1 {
			t.Errorf("Expected 1 typing indicator, got %d", count)
		}

		// Cleanup
		endEvent := bus.NewProcessingEndEvent(
			bus.ChannelTypeTelegram,
			"123456789",
			"telegram:987654321",
			nil,
		)
		tm.Stop(*endEvent)
	})

	t.Run("StopAll typing indicators", func(t *testing.T) {
		tm := NewTypingManager(nil, log)

		ctx := context.Background()
		tm.SetContext(ctx)

		// Start typing for multiple sessions (Telegram requires numeric session IDs)
		event1 := bus.NewProcessingStartEvent(
			bus.ChannelTypeTelegram,
			"user1",
			"telegram:123",
			nil,
		)
		event2 := bus.NewProcessingStartEvent(
			bus.ChannelTypeTelegram,
			"user2",
			"telegram:456",
			nil,
		)
		event3 := bus.NewProcessingStartEvent(
			bus.ChannelTypeTelegram,
			"user3",
			"telegram:789",
			nil,
		)

		tm.Start(*event1)
		tm.Start(*event2)
		tm.Start(*event3)
		time.Sleep(50 * time.Millisecond)

		// Check that all are stored
		tm.typingLock.RLock()
		count := len(tm.typingCancel)
		tm.typingLock.RUnlock()

		if count != 3 {
			t.Errorf("Expected 3 typing indicators, got %d", count)
		}

		// Stop all
		tm.StopAll()

		// Check that all are removed
		tm.typingLock.RLock()
		count = len(tm.typingCancel)
		tm.typingLock.RUnlock()

		if count != 0 {
			t.Errorf("Expected 0 typing indicators after StopAll, got %d", count)
		}
	})

	t.Run("Send typing indicator with invalid session ID", func(t *testing.T) {
		tm := NewTypingManager(nil, log)
		ctx := context.Background()
		tm.SetContext(ctx)

		event := bus.NewProcessingStartEvent(
			bus.ChannelTypeTelegram,
			"user1",
			"invalid",
			nil,
		)

		// Should not panic, just log error
		tm.Send(*event)
	})
}

// TestTypingManager_Send tests the Send method with nil bot
func TestTypingManager_Send(t *testing.T) {
	log, _ := logger.New(logger.Config{
		Level:  "debug",
		Format: "text",
		Output: "stdout",
	})

	t.Run("Send with nil bot", func(t *testing.T) {
		tm := NewTypingManager(nil, log)
		ctx := context.Background()
		tm.SetContext(ctx)

		event := bus.NewProcessingStartEvent(
			bus.ChannelTypeTelegram,
			"123456789",
			"telegram:987654321",
			nil,
		)

		// Should not panic with nil bot
		tm.Send(*event)
	})
}

// TestConnector_handleOutbound_InvalidSessionID tests handling outbound message with invalid session ID
func TestConnector_handleOutbound_InvalidSessionID(t *testing.T) {
	log, _ := logger.New(logger.Config{
		Level:  "debug",
		Format: "text",
		Output: "stdout",
	})

	msgBus := bus.New(100, 10, log)
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

	// Send telegram message with invalid session ID (not a number)
	outboundMsg := bus.OutboundMessage{
		ChannelType: bus.ChannelTypeTelegram,
		UserID:      "123456789",
		SessionID:   "invalid-session-id", // Invalid - not a number
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

// TestConnector_handleOutbound_ClosedChannel tests handling when outbound channel is closed
func TestConnector_handleOutbound_ClosedChannel(t *testing.T) {
	log, _ := logger.New(logger.Config{
		Level:  "debug",
		Format: "text",
		Output: "stdout",
	})

	msgBus := bus.New(100, 10, log)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := config.TelegramConfig{}

	conn := New(cfg, log, msgBus)
	conn.ctx = ctx

	// Create outbound channel and close it immediately
	outboundCh := make(chan bus.OutboundMessage, 1)
	close(outboundCh)
	conn.outboundCh = outboundCh

	// Start outbound handler in goroutine
	go conn.handleOutbound()

	// Wait for handler to detect closed channel
	time.Sleep(100 * time.Millisecond)

	// Stop handler
	cancel()
	time.Sleep(100 * time.Millisecond)
}

// TestConnector_handleOutbound_ContextCancelled tests handling when context is cancelled
func TestConnector_handleOutbound_ContextCancelled(t *testing.T) {
	log, _ := logger.New(logger.Config{
		Level:  "debug",
		Format: "text",
		Output: "stdout",
	})

	msgBus := bus.New(100, 10, log)
	ctx, cancel := context.WithCancel(context.Background())

	cfg := config.TelegramConfig{}

	conn := New(cfg, log, msgBus)
	conn.ctx = ctx

	// Create outbound channel
	outboundCh := make(chan bus.OutboundMessage, 1)
	conn.outboundCh = outboundCh

	// Start outbound handler in goroutine
	go conn.handleOutbound()

	// Cancel context immediately
	cancel()

	// Wait for handler to detect context cancellation
	time.Sleep(100 * time.Millisecond)
}

// TestConnector_registerCommands_NilBot tests command registration with nil bot
func TestConnector_registerCommands_NilBot(t *testing.T) {
	log, _ := logger.New(logger.Config{
		Level:  "debug",
		Format: "text",
		Output: "stdout",
	})

	msgBus := bus.New(100, 10, log)

	cfg := config.TelegramConfig{}
	conn := New(cfg, log, msgBus)
	conn.ctx = context.Background()

	// Test with nil bot - should return error
	err := conn.registerCommands()
	if err == nil {
		t.Error("Expected error when bot is nil")
	}

	// Verify the error message
	if err != nil && err.Error() != "bot is not initialized" {
		t.Errorf("Expected 'bot is not initialized' error, got: %v", err)
	}
}

// TestConnector_handleUpdate_StatusCommand tests /status command handling
func TestConnector_handleUpdate_StatusCommand(t *testing.T) {
	log, _ := logger.New(logger.Config{
		Level:  "debug",
		Format: "text",
		Output: "stdout",
	})

	msgBus := bus.New(100, 10, log)
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

	// Create a test update with /status command
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
			Text: "/status",
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
		if cmd, ok := msg.Metadata["command"].(string); !ok || cmd != "status" {
			t.Errorf("Expected command 'status' in metadata, got %v", msg.Metadata["command"])
		}
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for /status command message")
	}

	t.Cleanup(func() {
		require.NoError(t, msgBus.Stop())
	})
}

// TestConnector_handleUpdate_RestartCommand tests /restart command handling
func TestConnector_handleUpdate_RestartCommand(t *testing.T) {
	log, _ := logger.New(logger.Config{
		Level:  "debug",
		Format: "text",
		Output: "stdout",
	})

	msgBus := bus.New(100, 10, log)
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

	// Create a test update with /restart command
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
			Text: "/restart",
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
		if cmd, ok := msg.Metadata["command"].(string); !ok || cmd != "restart" {
			t.Errorf("Expected command 'restart' in metadata, got %v", msg.Metadata["command"])
		}
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for /restart command message")
	}

	t.Cleanup(func() {
		require.NoError(t, msgBus.Stop())
	})
}

// TestConnector_handleUpdate_WithChatType tests update with different chat types
func TestConnector_handleUpdate_WithChatType(t *testing.T) {
	log, _ := logger.New(logger.Config{
		Level:  "debug",
		Format: "text",
		Output: "stdout",
	})

	msgBus := bus.New(100, 10, log)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := msgBus.Start(ctx); err != nil {
		t.Fatalf("Failed to start message bus: %v", err)
	}

	chatTypes := []struct {
		chatType string
	}{
		{"private"},
		{"group"},
		{"supergroup"},
		{"channel"},
	}

	for _, tt := range chatTypes {
		t.Run(tt.chatType, func(t *testing.T) {
			cfg := config.TelegramConfig{
				AllowedUsers: []string{"123456789"},
			}

			conn := New(cfg, log, msgBus)
			conn.ctx = ctx

			inboundCh := msgBus.SubscribeInbound(ctx)

			update := telego.Update{
				Message: &telego.Message{
					MessageID: 1,
					From: &telego.User{
						ID:        123456789,
						FirstName: "TestUser",
					},
					Chat: telego.Chat{
						ID:   987654321,
						Type: tt.chatType,
					},
					Text: "Hello, bot!",
				},
			}

			err := conn.handleUpdate(update)
			if err != nil {
				t.Fatalf("handleUpdate() failed: %v", err)
			}

			select {
			case msg := <-inboundCh:
				if msg.Metadata["chat_type"] != tt.chatType {
					t.Errorf("Expected chat_type %s, got %v", tt.chatType, msg.Metadata["chat_type"])
				}
			case <-time.After(1 * time.Second):
				t.Fatal("Timeout waiting for inbound message")
			}
		})
	}

	t.Cleanup(func() {
		require.NoError(t, msgBus.Stop())
	})
}

// TestConnector_handleOutbound_MultipleMessages tests handling multiple outbound messages
func TestConnector_handleOutbound_MultipleMessages(t *testing.T) {
	log, _ := logger.New(logger.Config{
		Level:  "debug",
		Format: "text",
		Output: "stdout",
	})

	msgBus := bus.New(100, 10, log)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := config.TelegramConfig{}

	conn := New(cfg, log, msgBus)
	conn.ctx = ctx

	// Create outbound channel with buffer for multiple messages
	outboundCh := make(chan bus.OutboundMessage, 5)
	conn.outboundCh = outboundCh

	// Start outbound handler in goroutine
	go conn.handleOutbound()

	// Send multiple telegram messages (will fail due to nil bot, but tests flow)
	messages := []string{
		"Message 1",
		"Message 2",
		"Message 3",
	}

	for _, msgContent := range messages {
		outboundMsg := bus.OutboundMessage{
			ChannelType: bus.ChannelTypeTelegram,
			UserID:      "123456789",
			SessionID:   "telegram:987654321",
			Content:     msgContent,
			Timestamp:   time.Now(),
		}
		outboundCh <- outboundMsg
	}

	// Wait a bit for processing
	time.Sleep(200 * time.Millisecond)

	// Stop handler
	cancel()
	time.Sleep(100 * time.Millisecond)
}

// TestConnector_handleEvents_EventTypes tests handling different event types
func TestConnector_handleEvents_EventTypes(t *testing.T) {
	log, _ := logger.New(logger.Config{
		Level:  "debug",
		Format: "text",
		Output: "stdout",
	})

	msgBus := bus.New(100, 10, log)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := config.TelegramConfig{}
	conn := New(cfg, log, msgBus)
	conn.ctx = ctx

	// Create event channel
	eventCh := make(chan bus.Event, 20)
	conn.eventCh = eventCh

	// Start event handler in goroutine
	go conn.handleEvents()

	// Send multiple events
	for i := 0; i < 5; i++ {
		startEvent := bus.NewProcessingStartEvent(
			bus.ChannelTypeTelegram,
			"123456789",
			"telegram:987654321",
			nil,
		)
		eventCh <- *startEvent

		time.Sleep(10 * time.Millisecond)

		endEvent := bus.NewProcessingEndEvent(
			bus.ChannelTypeTelegram,
			"123456789",
			"telegram:987654321",
			nil,
		)
		eventCh <- *endEvent

		time.Sleep(10 * time.Millisecond)
	}

	// Stop handler
	cancel()
	time.Sleep(100 * time.Millisecond)
}

// TestTypingManager_Send_InvalidSessionID tests Send with invalid session ID
func TestTypingManager_Send_InvalidSessionID(t *testing.T) {
	log, _ := logger.New(logger.Config{
		Level:  "debug",
		Format: "text",
		Output: "stdout",
	})

	t.Run("invalid session ID format", func(t *testing.T) {
		tm := NewTypingManager(nil, log)
		ctx := context.Background()
		tm.SetContext(ctx)

		event := bus.NewProcessingStartEvent(
			bus.ChannelTypeTelegram,
			"user1",
			"not-a-number",
			nil,
		)

		// Should not panic, just log error
		tm.Send(*event)
	})

	t.Run("empty session ID", func(t *testing.T) {
		tm := NewTypingManager(nil, log)
		ctx := context.Background()
		tm.SetContext(ctx)

		event := bus.NewProcessingStartEvent(
			bus.ChannelTypeTelegram,
			"user1",
			"",
			nil,
		)

		// Should not panic, just log error
		tm.Send(*event)
	})

	t.Run("negative session ID", func(t *testing.T) {
		tm := NewTypingManager(nil, log)
		ctx := context.Background()
		tm.SetContext(ctx)

		event := bus.NewProcessingStartEvent(
			bus.ChannelTypeTelegram,
			"user1",
			"-123",
			nil,
		)

		// Should not panic, Sscanf will parse negative numbers
		tm.Send(*event)
	})
}

// TestUpdateHandler_Handle_BlockedUserNilBot tests blocked user with nil bot
func TestUpdateHandler_Handle_BlockedUserNilBot(t *testing.T) {
	log, _ := logger.New(logger.Config{
		Level:  "debug",
		Format: "text",
		Output: "stdout",
	})

	msgBus := bus.New(100, 10, log)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := msgBus.Start(ctx); err != nil {
		t.Fatalf("Failed to start message bus: %v", err)
	}

	cfg := config.TelegramConfig{
		AllowedUsers: []string{"123"}, // User 456 is not in the list
	}

	conn := New(cfg, log, msgBus)
	conn.ctx = ctx
	conn.bot = nil // Explicitly nil

	uh := NewUpdateHandler(conn, log, msgBus)

	// Create an update with blocked user
	update := telego.Update{
		Message: &telego.Message{
			MessageID: 1,
			From: &telego.User{
				ID:        456,
				FirstName: "BlockedUser",
				Username:  "blocked_user",
			},
			Chat: telego.Chat{
				ID:   987654321,
				Type: "private",
			},
			Text: "Hello, bot!",
		},
	}

	// Handle the update - should not panic
	err := uh.Handle(update)
	if err != nil {
		t.Errorf("Handle() with blocked user should not return error, got: %v", err)
	}

	t.Cleanup(func() {
		require.NoError(t, msgBus.Stop())
	})
}

// TestConnector_handleOutbound_NilBot tests outbound message handling with nil bot
func TestConnector_handleOutbound_NilBot(t *testing.T) {
	log, _ := logger.New(logger.Config{
		Level:  "debug",
		Format: "text",
		Output: "stdout",
	})

	msgBus := bus.New(100, 10, log)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := config.TelegramConfig{}

	conn := New(cfg, log, msgBus)
	conn.ctx = ctx
	conn.bot = nil // Explicitly nil

	// Create outbound channel
	outboundCh := make(chan bus.OutboundMessage, 1)
	conn.outboundCh = outboundCh

	// Start outbound handler in goroutine
	go conn.handleOutbound()

	// Send telegram message (will be skipped due to nil bot)
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

// TestConnector_handleEvents_ClosedChannel tests event handler with closed channel
func TestConnector_handleEvents_ClosedChannel(t *testing.T) {
	log, _ := logger.New(logger.Config{
		Level:  "debug",
		Format: "text",
		Output: "stdout",
	})

	msgBus := bus.New(100, 10, log)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := config.TelegramConfig{}
	conn := New(cfg, log, msgBus)
	conn.ctx = ctx

	// Create event channel and close it immediately
	eventCh := make(chan bus.Event, 1)
	close(eventCh)
	conn.eventCh = eventCh

	// Start event handler in goroutine
	go conn.handleEvents()

	// Wait for handler to detect closed channel
	time.Sleep(100 * time.Millisecond)

	// Stop handler
	cancel()
	time.Sleep(100 * time.Millisecond)
}
