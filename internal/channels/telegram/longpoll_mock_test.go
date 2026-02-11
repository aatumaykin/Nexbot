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

// TestLongPollManager_Start_WithMockUpdates tests LongPollManager with mock updates.
func TestLongPollManager_Start_WithMockUpdates(t *testing.T) {
	log, _ := logger.New(logger.Config{
		Level:  "debug",
		Format: "text",
		Output: "stdout",
	})

	msgBus := bus.New(100, 10, log)
	defer func() { _ = msgBus.Stop() }()

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
			Text: "Test message",
		},
	}

	// Create mock bot with the update
	mockBot, _ := NewMockBotWithUpdates(update)

	// Create connector
	conn := New(config.TelegramConfig{}, log, msgBus)
	conn.updateHandler = NewUpdateHandler(conn, log, msgBus)

	// Create LongPollManager
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	lp := NewLongPollManager(conn, mockBot, log)
	lp.SetContext(ctx)

	// Start long polling in goroutine
	done := make(chan bool)
	go func() {
		lp.Start()
		done <- true
	}()

	// Wait for processing
	select {
	case <-done:
	// OK
	case <-time.After(500 * time.Millisecond):
		cancel()
	}

	// Verify
	mockBot.AssertExpectations(t)
}

// TestLongPollManager_Start_WithCancelledContext tests LongPollManager with cancelled context.
func TestLongPollManager_Start_WithCancelledContext(t *testing.T) {
	log, _ := logger.New(logger.Config{
		Level:  "debug",
		Format: "text",
		Output: "stdout",
	})

	// Create mock bot that returns a channel
	mockBot := new(MockBot)
	updateCh := make(chan telego.Update)
	close(updateCh)
	mockBot.On("UpdatesViaLongPolling", mock.Anything, mock.Anything, mock.Anything).Return(updateCh, nil)

	// Create LongPollManager with already cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	lp := NewLongPollManager(nil, nil, log)
	lp.SetContext(ctx)
	lp.SetBot(mockBot)

	// Start long polling (should return immediately due to cancelled context)
	lp.Start()

	// Verify
	mockBot.AssertExpectations(t)
}

// TestLongPollManager_Start_WithClosedChannel tests LongPollManager when the update channel is closed.
func TestLongPollManager_Start_WithClosedChannel(t *testing.T) {
	log, _ := logger.New(logger.Config{
		Level:  "debug",
		Format: "text",
		Output: "stdout",
	})

	// Create mock bot that returns a closed channel
	updateCh := make(chan telego.Update)
	close(updateCh)

	mockBot := new(MockBot)
	mockBot.On("UpdatesViaLongPolling", mock.Anything, mock.Anything, mock.Anything).Return(updateCh, nil)

	// Create LongPollManager
	ctx := t.Context()

	lp := NewLongPollManager(nil, nil, log)
	lp.SetContext(ctx)
	lp.SetBot(mockBot)

	// Start long polling
	done := make(chan bool)
	go func() {
		lp.Start()
		done <- true
	}()

	// Wait for completion
	select {
	case <-done:
	// OK
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Long poll manager did not exit with closed channel")
	}

	// Verify
	mockBot.AssertExpectations(t)
}

// TestLongPollManager_Start_WithMultipleUpdates tests LongPollManager with multiple updates.
func TestLongPollManager_Start_WithMultipleUpdates(t *testing.T) {
	log, _ := logger.New(logger.Config{
		Level:  "debug",
		Format: "text",
		Output: "stdout",
	})

	msgBus := bus.New(100, 10, log)
	defer func() { _ = msgBus.Stop() }()

	// Create multiple test updates
	updates := []telego.Update{
		{
			Message: &telego.Message{
				MessageID: 1,
				From: &telego.User{
					ID:        123456789,
					FirstName: "User1",
				},
				Chat: telego.Chat{
					ID:   987654321,
					Type: "private",
				},
				Text: "Message 1",
			},
		},
		{
			Message: &telego.Message{
				MessageID: 2,
				From: &telego.User{
					ID:        123456789,
					FirstName: "User1",
				},
				Chat: telego.Chat{
					ID:   987654321,
					Type: "private",
				},
				Text: "Message 2",
			},
		},
		{
			Message: &telego.Message{
				MessageID: 3,
				From: &telego.User{
					ID:        123456789,
					FirstName: "User1",
				},
				Chat: telego.Chat{
					ID:   987654321,
					Type: "private",
				},
				Text: "Message 3",
			},
		},
	}

	// Create mock bot with the updates
	mockBot, _ := NewMockBotWithUpdates(updates...)

	// Create connector
	conn := New(config.TelegramConfig{}, log, msgBus)
	conn.updateHandler = NewUpdateHandler(conn, log, msgBus)

	// Create LongPollManager
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	lp := NewLongPollManager(conn, mockBot, log)
	lp.SetContext(ctx)

	// Start long polling in goroutine
	done := make(chan bool)
	go func() {
		lp.Start()
		done <- true
	}()

	// Wait for processing
	select {
	case <-done:
	// OK
	case <-time.After(500 * time.Millisecond):
		cancel()
	}

	// Verify
	mockBot.AssertExpectations(t)
}

// TestLongPollManager_Start_WithNilBot tests LongPollManager with nil bot.
func TestLongPollManager_Start_WithNilBot(t *testing.T) {
	log, _ := logger.New(logger.Config{
		Level:  "debug",
		Format: "text",
		Output: "stdout",
	})

	// Create LongPollManager with nil bot
	ctx := t.Context()

	lp := NewLongPollManager(nil, nil, log)
	lp.SetContext(ctx)
	lp.SetBot(nil)

	// Start long polling (should panic due to nil bot)
	assert.Panics(t, func() {
		lp.Start()
	})
}

// TestLongPollManager_Start_WithUpdateError tests LongPollManager when UpdatesViaLongPolling returns an error.
func TestLongPollManager_Start_WithUpdateError(t *testing.T) {
	log, _ := logger.New(logger.Config{
		Level:  "debug",
		Format: "text",
		Output: "stdout",
	})

	// Create mock bot that returns an error (by passing nil channel)
	mockBot := new(MockBot)
	var nilCh chan telego.Update
	mockBot.On("UpdatesViaLongPolling", mock.Anything, mock.Anything, mock.Anything).Return(nilCh, assert.AnError)

	// Create LongPollManager
	ctx := t.Context()

	lp := NewLongPollManager(nil, nil, log)
	lp.SetContext(ctx)
	lp.SetBot(mockBot)

	// Start long polling (should return immediately due to error)
	lp.Start()

	// Verify
	mockBot.AssertExpectations(t)
}

// TestLongPollManager_Start_WithCommandUpdate tests LongPollManager with a command update.
func TestLongPollManager_Start_WithCommandUpdate(t *testing.T) {
	log, _ := logger.New(logger.Config{
		Level:  "debug",
		Format: "text",
		Output: "stdout",
	})

	msgBus := bus.New(100, 10, log)
	defer func() { _ = msgBus.Stop() }()

	// Create a command update
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
			Entities: []telego.MessageEntity{
				{
					Type:   "bot_command",
					Offset: 0,
					Length: 4,
				},
			},
		},
	}

	// Create mock bot with the update
	mockBot, _ := NewMockBotWithUpdates(update)

	// Create connector
	conn := New(config.TelegramConfig{}, log, msgBus)
	conn.updateHandler = NewUpdateHandler(conn, log, msgBus)

	// Create LongPollManager
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	lp := NewLongPollManager(conn, mockBot, log)
	lp.SetContext(ctx)

	// Start long polling in goroutine
	done := make(chan bool)
	go func() {
		lp.Start()
		done <- true
	}()

	// Wait for processing
	select {
	case <-done:
	// OK
	case <-time.After(500 * time.Millisecond):
		cancel()
	}

	// Verify
	mockBot.AssertExpectations(t)
}

// TestLongPollManager_Start_WithNilMessage tests LongPollManager with an update containing nil message.
func TestLongPollManager_Start_WithNilMessage(t *testing.T) {
	log, _ := logger.New(logger.Config{
		Level:  "debug",
		Format: "text",
		Output: "stdout",
	})

	msgBus := bus.New(100, 10, log)
	defer func() { _ = msgBus.Stop() }()

	// Create an update with nil message
	update := telego.Update{
		Message: nil,
	}

	// Create mock bot with the update
	mockBot, _ := NewMockBotWithUpdates(update)

	// Create connector
	conn := New(config.TelegramConfig{}, log, msgBus)
	conn.updateHandler = NewUpdateHandler(conn, log, msgBus)

	// Create LongPollManager
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	lp := NewLongPollManager(conn, mockBot, log)
	lp.SetContext(ctx)

	// Start long polling in goroutine
	done := make(chan bool)
	go func() {
		lp.Start()
		done <- true
	}()

	// Wait for processing
	select {
	case <-done:
	// OK
	case <-time.After(500 * time.Millisecond):
		cancel()
	}

	// Verify
	mockBot.AssertExpectations(t)
}
