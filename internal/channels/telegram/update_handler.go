package telegram

import (
	"fmt"

	"github.com/aatumaykin/nexbot/internal/bus"
	"github.com/aatumaykin/nexbot/internal/logger"
	"github.com/mymmrac/telego"
)

// UpdateHandler handles processing of Telegram updates.
type UpdateHandler struct {
	connector *Connector
	logger    *logger.Logger
	bus       *bus.MessageBus
}

// NewUpdateHandler creates a new update handler.
func NewUpdateHandler(connector *Connector, logger *logger.Logger, bus *bus.MessageBus) *UpdateHandler {
	return &UpdateHandler{
		connector: connector,
		logger:    logger,
		bus:       bus,
	}
}

// Handle processes a Telegram update and publishes it to the message bus.
func (uh *UpdateHandler) Handle(update telego.Update) error {
	// Only process message updates
	if update.Message == nil {
		return nil
	}

	msg := update.Message
	if msg.Text == "" {
		// Skip non-text messages (photos, stickers, etc.) for now
		return nil
	}

	// Extract user information
	var userID string
	if msg.From != nil {
		userID = fmt.Sprintf("%d", msg.From.ID)
	}

	// Check for /new command before whitelist check (allow clearing session for authorized users)
	if msg.Text == "/new" {
		return uh.connector.commandHandler.HandleCommand(uh.connector.ctx, uh.connector.isAllowedUser, msg, "new_session", userID)
	}

	// Check for /status command - shows session and bot status (doesn't go to session)
	if msg.Text == "/status" {
		return uh.connector.commandHandler.HandleCommand(uh.connector.ctx, uh.connector.isAllowedUser, msg, "status", userID)
	}

	// Check for /restart command - restarts the bot
	if msg.Text == "/restart" {
		return uh.connector.commandHandler.HandleCommand(uh.connector.ctx, uh.connector.isAllowedUser, msg, "restart", userID)
	}

	// Check whitelist - block unauthorized users
	if !uh.connector.isAllowedUser(userID) {
		uh.logger.WarnCtx(uh.connector.ctx, "message blocked - user not in whitelist",
			logger.Field{Key: "user_id", Value: userID},
			logger.Field{Key: "username", Value: msg.From.Username})

		// Optionally send a message back informing the user
		if msg.Chat.ID != 0 && uh.connector.bot != nil {
			notifyParams := telego.SendMessageParams{
				ChatID: telego.ChatID{ID: msg.Chat.ID},
				Text:   "Sorry, you are not authorized to use this bot.",
			}
			_, err := uh.connector.bot.SendMessage(uh.connector.ctx, &notifyParams)
			if err != nil {
				uh.logger.ErrorCtx(uh.connector.ctx, "failed to send notification", err)
			}
		}

		return nil
	}

	// Use chat ID as session ID
	sessionID := fmt.Sprintf("%d", msg.Chat.ID)

	// Create inbound message
	inboundMsg := bus.NewInboundMessage(
		bus.ChannelTypeTelegram,
		userID,
		sessionID,
		msg.Text,
		map[string]any{
			"message_id":    msg.MessageID,
			"chat_id":       msg.Chat.ID,
			"chat_type":     msg.Chat.Type,
			"username":      msg.From.Username,
			"first_name":    msg.From.FirstName,
			"last_name":     msg.From.LastName,
			"language_code": msg.From.LanguageCode,
		},
	)

	// Publish to message bus
	if err := uh.bus.PublishInbound(*inboundMsg); err != nil {
		return fmt.Errorf("failed to publish inbound message: %w", err)
	}

	uh.logger.DebugCtx(uh.connector.ctx, "inbound message published",
		logger.Field{Key: "user_id", Value: userID},
		logger.Field{Key: "session_id", Value: sessionID},
		logger.Field{Key: "content", Value: msg.Text})

	return nil
}
