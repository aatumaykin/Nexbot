package telegram

import (
	"context"
	"testing"
	"time"

	"github.com/mymmrac/telego"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/aatumaykin/nexbot/internal/bus"
	"github.com/aatumaykin/nexbot/internal/logger"
)

// TestTypingManager_SendMock tests sending typing indicator with mock bot.
func TestTypingManager_SendMock(t *testing.T) {
	log, _ := logger.New(logger.Config{
		Level:  "debug",
		Format: "text",
		Output: "stdout",
	})

	mockBot := NewMockBotSuccess()

	// Set up expectations for typing indicator
	mockBot.On("SendChatAction", mock.Anything, mock.MatchedBy(func(params *telego.SendChatActionParams) bool {
		return params != nil && params.ChatID.ID == 987654321 && params.Action == "typing"
	})).Return(nil)

	// Create TypingManager
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tm := NewTypingManager(mockBot, log)
	tm.SetContext(ctx)

	// Send typing indicator
	event := bus.Event{
		ChannelType: bus.ChannelTypeTelegram,
		Type:        bus.EventTypeProcessingStart,
		SessionID:   "987654321",
		UserID:      "123456789",
	}

	tm.Send(event)

	// Wait a bit
	time.Sleep(50 * time.Millisecond)

	// Verify
	mockBot.AssertExpectations(t)
}

// TestTypingManager_Periodic_WithMock tests periodic typing indicator sending with mock bot.
func TestTypingManager_Periodic_WithMock(t *testing.T) {
	log, _ := logger.New(logger.Config{
		Level:  "debug",
		Format: "text",
		Output: "stdout",
	})

	mockBot := new(MockBot)

	// Set up expectations for multiple typing indicators (should be called at least twice)
	mockBot.On("SendChatAction", mock.Anything, mock.MatchedBy(func(params *telego.SendChatActionParams) bool {
		return params != nil && params.ChatID.ID == 987654321 && params.Action == "typing"
	})).Return(nil).Times(2) // Expect at least 2 calls

	// Create TypingManager
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tm := NewTypingManager(mockBot, log)
	tm.SetContext(ctx)

	// Start typing indicator
	event := bus.Event{
		ChannelType: bus.ChannelTypeTelegram,
		Type:        bus.EventTypeProcessingStart,
		SessionID:   "987654321",
		UserID:      "123456789",
	}

	tm.Start(event)

	// Wait for at least 2 sends (3 second interval)
	time.Sleep(4 * time.Second)

	// Stop typing indicator
	tm.Stop(event)

	// Wait a bit for cleanup
	time.Sleep(100 * time.Millisecond)

	// Verify
	mockBot.AssertExpectations(t)
}

// TestTypingManager_Stop_WithMock tests stopping typing indicator with mock bot.
func TestTypingManager_Stop_WithMock(t *testing.T) {
	log, _ := logger.New(logger.Config{
		Level:  "debug",
		Format: "text",
		Output: "stdout",
	})

	mockBot := new(MockBot)

	// Set up expectations for typing indicator (should be called once before stop)
	mockBot.On("SendChatAction", mock.Anything, mock.MatchedBy(func(params *telego.SendChatActionParams) bool {
		return params != nil && params.ChatID.ID == 987654321 && params.Action == "typing"
	})).Return(nil).Once()

	// Create TypingManager
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tm := NewTypingManager(mockBot, log)
	tm.SetContext(ctx)

	// Start typing indicator
	event := bus.Event{
		ChannelType: bus.ChannelTypeTelegram,
		Type:        bus.EventTypeProcessingStart,
		SessionID:   "987654321",
		UserID:      "123456789",
	}

	tm.Start(event)

	// Wait for initial send
	time.Sleep(100 * time.Millisecond)

	// Stop typing indicator
	tm.Stop(event)

	// Wait a bit for cleanup
	time.Sleep(100 * time.Millisecond)

	// Verify
	mockBot.AssertExpectations(t)
}

// TestTypingManager_StopAll_WithMock tests stopping all typing indicators with mock bot.
func TestTypingManager_StopAll_WithMock(t *testing.T) {
	log, _ := logger.New(logger.Config{
		Level:  "debug",
		Format: "text",
		Output: "stdout",
	})

	mockBot := new(MockBot)

	// Set up expectations for multiple typing indicators
	mockBot.On("SendChatAction", mock.Anything, mock.MatchedBy(func(params *telego.SendChatActionParams) bool {
		return params != nil && params.Action == "typing"
	})).Return(nil).Times(2)

	// Create TypingManager
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tm := NewTypingManager(mockBot, log)
	tm.SetContext(ctx)

	// Start multiple typing indicators
	event1 := bus.Event{
		ChannelType: bus.ChannelTypeTelegram,
		Type:        bus.EventTypeProcessingStart,
		SessionID:   "987654321",
		UserID:      "123456789",
	}

	event2 := bus.Event{
		ChannelType: bus.ChannelTypeTelegram,
		Type:        bus.EventTypeProcessingStart,
		SessionID:   "111222333",
		UserID:      "987654321",
	}

	tm.Start(event1)
	tm.Start(event2)

	// Wait for initial sends
	time.Sleep(100 * time.Millisecond)

	// Stop all typing indicators
	tm.StopAll()

	// Wait a bit for cleanup
	time.Sleep(100 * time.Millisecond)

	// Verify
	mockBot.AssertExpectations(t)
}

// TestTypingManager_Send_WithMockError tests error handling when sending typing indicator fails.
func TestTypingManager_Send_WithMockError(t *testing.T) {
	log, _ := logger.New(logger.Config{
		Level:  "debug",
		Format: "text",
		Output: "stdout",
	})

	mockBot := NewMockBotSuccess()

	// Set up expectations for typing indicator with error
	mockBot.On("SendChatAction", mock.Anything, mock.Anything).Return(assert.AnError)

	// Create TypingManager
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tm := NewTypingManager(mockBot, log)
	tm.SetContext(ctx)

	// Send typing indicator (should not panic)
	event := bus.Event{
		ChannelType: bus.ChannelTypeTelegram,
		Type:        bus.EventTypeProcessingStart,
		SessionID:   "987654321",
		UserID:      "123456789",
	}

	tm.Send(event)

	// Wait a bit
	time.Sleep(50 * time.Millisecond)

	// Verify
	mockBot.AssertExpectations(t)
}

// TestTypingManager_Start_WithNilBot tests starting typing indicator with nil bot.
func TestTypingManager_Start_WithNilBot(t *testing.T) {
	log, _ := logger.New(logger.Config{
		Level:  "debug",
		Format: "text",
		Output: "stdout",
	})

	// Create TypingManager with nil bot
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tm := NewTypingManager(nil, log)
	tm.SetContext(ctx)

	// Start typing indicator (should not panic)
	event := bus.Event{
		ChannelType: bus.ChannelTypeTelegram,
		Type:        bus.EventTypeProcessingStart,
		SessionID:   "987654321",
		UserID:      "123456789",
	}

	tm.Start(event)

	// Wait a bit
	time.Sleep(100 * time.Millisecond)

	// Stop typing indicator
	tm.Stop(event)
}

// TestTypingManager_StartNilContext tests starting typing indicator with nil context.
func TestTypingManager_StartNilContext(t *testing.T) {
	log, _ := logger.New(logger.Config{
		Level:  "debug",
		Format: "text",
		Output: "stdout",
	})

	mockBot := new(MockBot)

	// Set up expectations (may or may not be called)
	mockBot.On("SendChatAction", mock.Anything, mock.Anything).Return(nil).Maybe()

	// Create TypingManager without setting context
	tm := NewTypingManager(mockBot, log)

	// Start typing indicator (should not panic)
	event := bus.Event{
		ChannelType: bus.ChannelTypeTelegram,
		Type:        bus.EventTypeProcessingStart,
		SessionID:   "987654321",
		UserID:      "123456789",
	}

	tm.Start(event)

	// Wait a bit
	time.Sleep(100 * time.Millisecond)

	// Stop typing indicator
	tm.Stop(event)
}

// TestTypingManager_StartAlreadyStarted tests starting typing indicator that is already started.
func TestTypingManager_StartAlreadyStarted(t *testing.T) {
	log, _ := logger.New(logger.Config{
		Level:  "debug",
		Format: "text",
		Output: "stdout",
	})

	mockBot := new(MockBot)

	// Set up expectations for typing indicator (should be called once despite double Start)
	mockBot.On("SendChatAction", mock.Anything, mock.MatchedBy(func(params *telego.SendChatActionParams) bool {
		return params != nil && params.ChatID.ID == 987654321 && params.Action == "typing"
	})).Return(nil).Once()

	// Create TypingManager
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tm := NewTypingManager(mockBot, log)
	tm.SetContext(ctx)

	// Start typing indicator
	event := bus.Event{
		ChannelType: bus.ChannelTypeTelegram,
		Type:        bus.EventTypeProcessingStart,
		SessionID:   "987654321",
		UserID:      "123456789",
	}

	tm.Start(event)

	// Start again with same event (should not create new goroutine)
	tm.Start(event)

	// Wait a bit
	time.Sleep(100 * time.Millisecond)

	// Stop typing indicator
	tm.Stop(event)

	// Verify
	mockBot.AssertExpectations(t)
}

// TestTypingManager_StopNonExistent tests stopping a non-existent typing indicator.
func TestTypingManager_StopNonExistent(t *testing.T) {
	log, _ := logger.New(logger.Config{
		Level:  "debug",
		Format: "text",
		Output: "stdout",
	})

	mockBot := new(MockBot)

	// Create TypingManager
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tm := NewTypingManager(mockBot, log)
	tm.SetContext(ctx)

	// Stop typing indicator that was never started (should not panic)
	event := bus.Event{
		ChannelType: bus.ChannelTypeTelegram,
		Type:        bus.EventTypeProcessingStart,
		SessionID:   "987654321",
		UserID:      "123456789",
	}

	tm.Stop(event)
}

// TestTypingManager_StartCancelledContext tests starting typing indicator with cancelled context.
func TestTypingManager_StartCancelledContext(t *testing.T) {
	log, _ := logger.New(logger.Config{
		Level:  "debug",
		Format: "text",
		Output: "stdout",
	})

	mockBot := new(MockBot)

	// Set up expectations (SendChatAction may be called once before context cancellation is checked)
	mockBot.On("SendChatAction", mock.Anything, mock.Anything).Return(nil).Maybe()

	// Create TypingManager with cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	tm := NewTypingManager(mockBot, log)
	tm.SetContext(ctx)

	// Start typing indicator (may send once before context check)
	event := bus.Event{
		ChannelType: bus.ChannelTypeTelegram,
		Type:        bus.EventTypeProcessingStart,
		SessionID:   "987654321",
		UserID:      "123456789",
	}

	tm.Start(event)

	// Wait a bit
	time.Sleep(100 * time.Millisecond)

	// Stop typing indicator
	tm.Stop(event)
}
