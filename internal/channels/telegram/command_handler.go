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
	logger    *logger.Logger
	bus       *bus.MessageBus
	connector *Connector
}

// NewCommandHandler creates a new command handler
func NewCommandHandler(logger *logger.Logger, msgBus *bus.MessageBus) *CommandHandler {
	return &CommandHandler{
		logger: logger,
		bus:    msgBus,
	}
}

// SetConnector sets the connector reference (called after connector initialization)
func (h *CommandHandler) SetConnector(conn *Connector) {
	h.connector = conn
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

	// Handle built-in commands directly
	switch command {
	case "help":
		return h.sendHelpResponse(ctx, msg.Chat.ID)
	case "settings":
		return h.sendSettingsResponse(ctx, msg.Chat.ID)
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

// sendHelpResponse sends help message to the specified chat
func (h *CommandHandler) sendHelpResponse(ctx context.Context, chatID int64) error {
	if h.connector == nil || h.connector.bot == nil {
		return fmt.Errorf("connector or bot not initialized")
	}

	helpText := `🤖 *Nexbot - AI Assistant*

Доступные команды:
/new - Начать новую сессию (очистить историю)
/status - Показать статус сессии и бота
/settings - Настройки бота
/restart - Перезапустить бота
/help - Показать справку`

	params := &telego.SendMessageParams{
		ChatID:    telego.ChatID{ID: chatID},
		Text:      helpText,
		ParseMode: telego.ModeMarkdown,
	}

	_, err := h.connector.bot.SendMessage(ctx, params)
	if err != nil {
		return fmt.Errorf("failed to send help message: %w", err)
	}

	h.logger.DebugCtx(ctx, "help message sent", logger.Field{Key: "chat_id", Value: chatID})

	return nil
}

// sendSettingsResponse sends settings message to the specified chat
func (h *CommandHandler) sendSettingsResponse(ctx context.Context, chatID int64) error {
	if h.connector == nil || h.connector.bot == nil {
		return fmt.Errorf("connector or bot not initialized")
	}

	settingsText := `⚙️ *Настройки бота*

Текущие настройки:

*Статус:* Работает
*Канал:* Telegram
*Провайдер:* Z.ai (GLM-4.7)

Примечание: Настройки управляются через конфигурационный файл TOML.`

	params := &telego.SendMessageParams{
		ChatID:    telego.ChatID{ID: chatID},
		Text:      settingsText,
		ParseMode: telego.ModeMarkdown,
	}

	_, err := h.connector.bot.SendMessage(ctx, params)
	if err != nil {
		return fmt.Errorf("failed to send settings message: %w", err)
	}

	h.logger.DebugCtx(ctx, "settings message sent", logger.Field{Key: "chat_id", Value: chatID})

	return nil
}
