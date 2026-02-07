// Package telegram provides Telegram Bot integration using the Telego library.
package telegram

import (
	"context"
	"fmt"

	"github.com/aatumaykin/nexbot/internal/bus"
	"github.com/aatumaykin/nexbot/internal/logger"
	"github.com/mymmrac/telego"
)

// CommandHandler handles Telegram bot commands
type CommandHandler struct {
	logger *logger.Logger
	bus    *bus.MessageBus
}

// NewCommandHandler creates a new command handler
func NewCommandHandler(logger *logger.Logger, msgBus *bus.MessageBus) *CommandHandler {
	return &CommandHandler{
		logger: logger,
		bus:    msgBus,
	}
}

// HandleCommand processes a bot command
func (h *CommandHandler) HandleCommand(
	ctx context.Context,
	isAllowedFunc func(userID string) bool,
	msg *telego.Message,
	command, userID string,
) error {
	// Authorization check (extracted once)
	if !isAllowedFunc(userID) {
		h.logger.WarnCtx(ctx, "command blocked - user not in whitelist",
			logger.Field{Key: "user_id", Value: userID},
			logger.Field{Key: "command", Value: "/" + command})
		return nil
	}

	// Create inbound message (extracted once)
	sessionID := fmt.Sprintf("telegram:%d", msg.Chat.ID)
	metadata := map[string]any{
		"command":    command,
		"message_id": msg.MessageID,
		"chat_id":    msg.Chat.ID,
		"chat_type":  msg.Chat.Type,
		"username":   msg.From.Username,
	}

	inboundMsg := bus.NewInboundMessage(
		bus.ChannelTypeTelegram, userID, sessionID, msg.Text, metadata,
	)

	if err := h.bus.PublishInbound(*inboundMsg); err != nil {
		return fmt.Errorf("failed to publish command message: %w", err)
	}

	h.logger.DebugCtx(ctx, command+" command published",
		logger.Field{Key: "user_id", Value: userID},
		logger.Field{Key: "session_id", Value: sessionID})

	return nil
}
