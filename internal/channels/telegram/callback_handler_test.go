package telegram

import (
	"context"
	"testing"
	"time"

	"github.com/aatumaykin/nexbot/internal/bus"
	"github.com/aatumaykin/nexbot/internal/config"
	"github.com/aatumaykin/nexbot/internal/logger"
	"github.com/mymmrac/telego"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestNewCallbackHandler(t *testing.T) {
	log, err := logger.New(logger.Config{
		Level:  "debug",
		Format: "text",
		Output: "stdout",
	})
	require.NoError(t, err)
	mockBus := bus.New(10, log)

	handler := NewCallbackHandler(nil, log, mockBus)
	assert.NotNil(t, handler)
	assert.NotNil(t, handler.logger)
	assert.NotNil(t, handler.bus)
}

func TestCallbackHandler_Handle_NilCallback(t *testing.T) {
	log, err := logger.New(logger.Config{
		Level:  "debug",
		Format: "text",
		Output: "stdout",
	})
	require.NoError(t, err)
	mockBus := bus.New(10, log)

	handler := NewCallbackHandler(nil, log, mockBus)
	err = handler.Handle(nil)
	assert.NoError(t, err)
}

func TestCallbackHandler_Handle_UnauthorizedUser(t *testing.T) {
	ctx := context.Background()
	log, err := logger.New(logger.Config{
		Level:  "debug",
		Format: "text",
		Output: "stdout",
	})
	require.NoError(t, err)
	mockBus := bus.New(10, log)
	require.NoError(t, mockBus.Start(ctx))

	// Create mock bot
	mockBot := NewMockBotSuccess()
	mockBot.On("AnswerCallbackQuery", mock.Anything, mock.Anything).Return(nil)

	// Create connector with specific allowed users
	connector := &Connector{
		cfg:    config.TelegramConfig{AllowedUsers: []string{"123456"}},
		ctx:    ctx,
		logger: log,
		bus:    mockBus,
		bot:    mockBot,
	}

	handler := NewCallbackHandler(connector, log, mockBus)

	// Create callback query from unauthorized user
	callbackQuery := &telego.CallbackQuery{
		ID: "callback_123",
		From: telego.User{
			ID:       999999,
			Username: "unauthorized",
		},
		Data: "action:test",
		Message: &telego.Message{
			MessageID: 123,
			Chat: telego.Chat{
				ID:   123456789,
				Type: "private",
			},
		},
	}

	err = handler.Handle(callbackQuery)
	assert.NoError(t, err)

	// Wait a bit for async operations
	time.Sleep(10 * time.Millisecond)
	mockBus.Stop()
}

func TestCallbackHandler_Handle_AuthorizedUser(t *testing.T) {
	ctx := context.Background()
	log, err := logger.New(logger.Config{
		Level:  "debug",
		Format: "text",
		Output: "stdout",
	})
	require.NoError(t, err)
	mockBus := bus.New(10, log)
	require.NoError(t, mockBus.Start(ctx))

	// Create mock bot
	mockBot := NewMockBotSuccess()
	mockBot.On("AnswerCallbackQuery", mock.Anything, mock.Anything).Return(nil)

	// Create connector with specific allowed users
	connector := &Connector{
		cfg: config.TelegramConfig{
			AllowedUsers:          []string{"123456"},
			AnswerCallbackTimeout: 5,
		},
		ctx:    ctx,
		logger: log,
		bus:    mockBus,
		bot:    mockBot,
	}

	handler := NewCallbackHandler(connector, log, mockBus)

	// Subscribe to inbound messages to verify publication
	inboundCh := mockBus.SubscribeInbound(ctx)
	defer func() {
		if inboundCh != nil {
			go func() {
				for range inboundCh {
				}
			}()
		}
	}()

	// Create callback query from authorized user
	callbackQuery := &telego.CallbackQuery{
		ID: "callback_123",
		From: telego.User{
			ID:       123456,
			Username: "authorized",
		},
		Data: "action:test",
		Message: &telego.Message{
			MessageID: 123,
			Chat: telego.Chat{
				ID:   123456789,
				Type: "private",
			},
		},
	}

	err = handler.Handle(callbackQuery)
	assert.NoError(t, err)

	// Verify bot.AnswerCallbackQuery was called
	mockBot.AssertCalled(t, "AnswerCallbackQuery", mock.Anything, mock.Anything)

	// Wait for inbound message
	select {
	case msg := <-inboundCh:
		assert.Equal(t, bus.ChannelTypeTelegram, msg.ChannelType)
		assert.Equal(t, "123456", msg.UserID)
		assert.Equal(t, "telegram:123456789", msg.SessionID)
		assert.Equal(t, "action:test", msg.Content)
		assert.Equal(t, "callback", msg.Metadata["message_type"])
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Timeout waiting for inbound message")
	}

	mockBus.Stop()
}

func TestCallbackHandler_Handle_InlineMessage(t *testing.T) {
	ctx := context.Background()
	log, err := logger.New(logger.Config{
		Level:  "debug",
		Format: "text",
		Output: "stdout",
	})
	require.NoError(t, err)
	mockBus := bus.New(10, log)
	require.NoError(t, mockBus.Start(ctx))

	// Create mock bot
	mockBot := NewMockBotSuccess()
	mockBot.On("AnswerCallbackQuery", mock.Anything, mock.Anything).Return(nil)

	// Create connector with specific allowed users
	connector := &Connector{
		cfg: config.TelegramConfig{
			AllowedUsers:          []string{"123456", "789012"},
			AnswerCallbackTimeout: 5,
		},
		ctx:    ctx,
		logger: log,
		bus:    mockBus,
		bot:    mockBot,
	}

	handler := NewCallbackHandler(connector, log, mockBus)

	// Subscribe to inbound messages
	inboundCh := mockBus.SubscribeInbound(ctx)
	defer func() {
		if inboundCh != nil {
			go func() {
				for range inboundCh {
				}
			}()
		}
	}()

	// Create inline callback query (no message)
	callbackQuery := &telego.CallbackQuery{
		ID: "callback_456",
		From: telego.User{
			ID:       789012,
			Username: "inline_user",
		},
		Data:            "inline_action",
		InlineMessageID: "inline_msg_id_123",
	}

	err = handler.Handle(callbackQuery)
	assert.NoError(t, err)

	// Verify session ID uses user ID for inline messages
	select {
	case msg := <-inboundCh:
		assert.Equal(t, "telegram:789012", msg.SessionID)
		assert.Equal(t, "inline_action", msg.Content)
		assert.Equal(t, "inline_msg_id_123", msg.Metadata["inline_message_id"])
		assert.Equal(t, true, msg.Metadata["is_inline"])
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Timeout waiting for inbound message")
	}

	mockBus.Stop()
}
