package telegram

import (
	"context"
	"fmt"
	"time"

	"github.com/aatumaykin/nexbot/internal/bus"
	"github.com/aatumaykin/nexbot/internal/logger"
	"github.com/mymmrac/telego"
)

// CallbackHandler handles processing of Telegram callback queries from inline keyboards.
type CallbackHandler struct {
	connector *Connector
	logger    *logger.Logger
	bus       *bus.MessageBus
}

// NewCallbackHandler creates a new callback handler.
func NewCallbackHandler(connector *Connector, logger *logger.Logger, bus *bus.MessageBus) *CallbackHandler {
	return &CallbackHandler{
		connector: connector,
		logger:    logger,
		bus:       bus,
	}
}

// Handle processes a Telegram callback query and publishes it to the message bus.
// It answers the callback query to remove the loading animation and creates
// an inbound message with the callback data for processing by the agent.
func (ch *CallbackHandler) Handle(callbackQuery *telego.CallbackQuery) error {
	if callbackQuery == nil {
		return nil
	}

	// Extract user information
	userID := fmt.Sprintf("%d", callbackQuery.From.ID)

	// Check whitelist - block unauthorized users
	if !ch.connector.isAllowedUser(userID) {
		ch.logger.WarnCtx(ch.connector.ctx, "callback query blocked - user not in whitelist",
			logger.Field{Key: "user_id", Value: userID},
			logger.Field{Key: "username", Value: callbackQuery.From.Username})

		// Answer the callback query to remove loading animation
		if ch.connector.bot != nil {
			answerParams := &telego.AnswerCallbackQueryParams{
				CallbackQueryID: callbackQuery.ID,
				Text:            "Sorry, you are not authorized to use this bot.",
				ShowAlert:       true,
			}

			// Use timeout from config
			timeout := time.Duration(ch.connector.cfg.AnswerCallbackTimeout) * time.Second
			ctx, cancel := context.WithTimeout(ch.connector.ctx, timeout)
			defer cancel()

			if err := ch.connector.bot.AnswerCallbackQuery(ctx, answerParams); err != nil {
				ch.logger.ErrorCtx(ch.connector.ctx, "failed to answer callback query for unauthorized user", err)
			}
		}

		return nil
	}

	// Use chat ID or message chat ID as session ID with channel prefix
	var sessionID string
	if callbackQuery.Message != nil {
		chat := callbackQuery.Message.GetChat()
		if chat.ID != 0 {
			sessionID = fmt.Sprintf("telegram:%d", chat.ID)
		}
	}
	if sessionID == "" {
		// Fallback: use user ID if no chat available (e.g., inline mode)
		sessionID = fmt.Sprintf("telegram:%s", userID)
	}

	// Extract metadata from callback query
	metadata := map[string]any{
		"callback_query_id": callbackQuery.ID,
		"username":          callbackQuery.From.Username,
		"first_name":        callbackQuery.From.FirstName,
		"last_name":         callbackQuery.From.LastName,
		"language_code":     callbackQuery.From.LanguageCode,
		"is_inline":         callbackQuery.Message == nil,
	}

	// Include message metadata if available
	if callbackQuery.Message != nil {
		metadata["message_id"] = callbackQuery.Message.GetMessageID()
		chat := callbackQuery.Message.GetChat()
		metadata["chat_id"] = chat.ID
		metadata["chat_type"] = chat.Type
	}

	// Include inline message ID if available
	if callbackQuery.InlineMessageID != "" {
		metadata["inline_message_id"] = callbackQuery.InlineMessageID
	}

	// Create inbound message with callback data as content
	// The callback data contains the button action/value
	content := callbackQuery.Data
	if content == "" {
		content = "" // Empty callback data
	}

	inboundMsg := bus.NewInboundMessage(
		bus.ChannelTypeTelegram,
		userID,
		sessionID,
		content,
		metadata,
	)

	// Mark this as a callback message in metadata
	if inboundMsg.Metadata == nil {
		inboundMsg.Metadata = make(map[string]any)
	}
	inboundMsg.Metadata["message_type"] = "callback"

	// Publish to message bus
	if err := ch.bus.PublishInbound(*inboundMsg); err != nil {
		return fmt.Errorf("failed to publish inbound callback message: %w", err)
	}

	// Answer the callback query to remove the loading animation
	// We answer it immediately to improve user experience
	if ch.connector.bot != nil {
		answerParams := &telego.AnswerCallbackQueryParams{
			CallbackQueryID: callbackQuery.ID,
		}

		// Use timeout from config
		timeout := time.Duration(ch.connector.cfg.AnswerCallbackTimeout) * time.Second
		ctx, cancel := context.WithTimeout(ch.connector.ctx, timeout)
		defer cancel()

		if err := ch.connector.bot.AnswerCallbackQuery(ctx, answerParams); err != nil {
			ch.logger.ErrorCtx(ch.connector.ctx, "failed to answer callback query", err,
				logger.Field{Key: "callback_query_id", Value: callbackQuery.ID},
				logger.Field{Key: "user_id", Value: userID})
		}
	}

	ch.logger.DebugCtx(ch.connector.ctx, "inbound callback message published",
		logger.Field{Key: "user_id", Value: userID},
		logger.Field{Key: "session_id", Value: sessionID},
		logger.Field{Key: "callback_data", Value: content})

	return nil
}
