package telegram

import (
	"context"
	"testing"
	"time"

	"github.com/aatumaykin/nexbot/internal/bus"
	"github.com/aatumaykin/nexbot/internal/config"
	"github.com/aatumaykin/nexbot/internal/logger"
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

	_ = bus.New(100, log)

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
			"987654321",
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
			"987654321",
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
			"987654321",
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
			"987654321",
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
			"987654321",
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
			"987654321",
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
			"123",
			nil,
		)
		event2 := bus.NewProcessingStartEvent(
			bus.ChannelTypeTelegram,
			"user2",
			"456",
			nil,
		)
		event3 := bus.NewProcessingStartEvent(
			bus.ChannelTypeTelegram,
			"user3",
			"789",
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
			"987654321",
			nil,
		)

		// Should not panic with nil bot
		tm.Send(*event)
	})
}
