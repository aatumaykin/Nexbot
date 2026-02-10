// Package telegram provides Telegram Bot integration using the Telego library.
package telegram

import (
	"context"
	"fmt"
	"strings"

	"github.com/aatumaykin/nexbot/internal/bus"
	"github.com/aatumaykin/nexbot/internal/logger"
	"github.com/aatumaykin/nexbot/internal/secrets"
	"github.com/mymmrac/telego"
)

// CommandHandler handles Telegram bot commands
type CommandHandler struct {
	logger    *logger.Logger
	bus       *bus.MessageBus
	connector *Connector
	secrets   *secrets.Store
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

// SetSecretsStore sets the secrets store (called after secrets initialization)
func (h *CommandHandler) SetSecretsStore(secretsStore *secrets.Store) {
	h.secrets = secretsStore
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
	case "secret":
		return h.handleSecretCommand(ctx, msg)
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
/secret - Управление секретами (пароли, токены)
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

// handleSecretCommand handles /secret commands
func (h *CommandHandler) handleSecretCommand(ctx context.Context, msg *telego.Message) error {
	if h.connector == nil || h.connector.bot == nil {
		return fmt.Errorf("connector or bot not initialized")
	}

	sessionID := fmt.Sprintf("telegram:%d", msg.Chat.ID)

	// Parse command arguments
	parts := strings.Fields(msg.Text[len("/secret"):])
	if len(parts) == 0 {
		return h.sendSecretHelp(ctx, msg.Chat.ID)
	}

	action := parts[0]

	switch action {
	case "list":
		return h.listSecrets(ctx, msg.Chat.ID, sessionID)
	case "clear":
		return h.clearSecrets(ctx, msg.Chat.ID, sessionID)
	case "delete":
		if len(parts) < 2 {
			return h.sendSecretHelp(ctx, msg.Chat.ID)
		}
		return h.deleteSecret(ctx, msg.Chat.ID, sessionID, parts[1])
	default:
		// Treat as: /secret <name> <value>
		if len(parts) >= 2 {
			secretName := parts[0]
			secretValue := strings.Join(parts[1:], " ")
			return h.setSecret(ctx, msg.Chat.ID, sessionID, secretName, secretValue)
		} else {
			return h.sendSecretHelp(ctx, msg.Chat.ID)
		}
	}
}

// sendSecretHelp sends help for /secret command
func (h *CommandHandler) sendSecretHelp(ctx context.Context, chatID int64) error {
	helpText := `🔐 *Управление секретами*

Использование:
/secret <name> <value> - Создать или обновить секрет
/secret delete <name> - Удалить секрет
/secret list - Показать список секретов
/secret clear - Удалить все секреты сессии

Пример:
/secret API_KEY sk-1234567890
/secret delete API_KEY
/secret list
/secret clear

Примечание: Секреты хранятся в зашифрованном виде и изолированы по сессии.`

	params := &telego.SendMessageParams{
		ChatID:    telego.ChatID{ID: chatID},
		Text:      helpText,
		ParseMode: telego.ModeMarkdown,
	}

	_, err := h.connector.bot.SendMessage(ctx, params)
	if err != nil {
		return fmt.Errorf("failed to send secret help message: %w", err)
	}

	return nil
}

// setSecret creates or updates a secret
func (h *CommandHandler) setSecret(ctx context.Context, chatID int64, sessionID, name, value string) error {
	if h.secrets == nil {
		return h.sendMessage(ctx, chatID, "❌ Хранилище секретов не инициализировано")
	}

	if err := h.secrets.Put(sessionID, name, value); err != nil {
		h.logger.ErrorCtx(ctx, "failed to save secret", err,
			logger.Field{Key: "session_id", Value: sessionID},
			logger.Field{Key: "secret_name", Value: name})
		return h.sendMessage(ctx, chatID, fmt.Sprintf("❌ Ошибка сохранения секрета '%s': %v", name, err))
	}

	return h.sendMessage(ctx, chatID, fmt.Sprintf("✅ Секрет '%s' сохранен", name))
}

// deleteSecret deletes a secret
func (h *CommandHandler) deleteSecret(ctx context.Context, chatID int64, sessionID, name string) error {
	if h.secrets == nil {
		return h.sendMessage(ctx, chatID, "❌ Хранилище секретов не инициализировано")
	}

	if err := h.secrets.Delete(sessionID, name); err != nil {
		if err == secrets.ErrSecretNotFound {
			return h.sendMessage(ctx, chatID, fmt.Sprintf("❌ Секрет '%s' не найден", name))
		}
		h.logger.ErrorCtx(ctx, "failed to delete secret", err,
			logger.Field{Key: "session_id", Value: sessionID},
			logger.Field{Key: "secret_name", Value: name})
		return h.sendMessage(ctx, chatID, fmt.Sprintf("❌ Ошибка удаления секрета '%s': %v", name, err))
	}

	return h.sendMessage(ctx, chatID, fmt.Sprintf("✅ Секрет '%s' удален", name))
}

// listSecrets lists all secrets for the session
func (h *CommandHandler) listSecrets(ctx context.Context, chatID int64, sessionID string) error {
	if h.secrets == nil {
		return h.sendMessage(ctx, chatID, "❌ Хранилище секретов не инициализировано")
	}

	names, err := h.secrets.List(sessionID)
	if err != nil {
		h.logger.ErrorCtx(ctx, "failed to list secrets", err,
			logger.Field{Key: "session_id", Value: sessionID})
		return h.sendMessage(ctx, chatID, fmt.Sprintf("❌ Ошибка получения списка секретов: %v", err))
	}

	if len(names) == 0 {
		return h.sendMessage(ctx, chatID, "📭 Секреты не найдены")
	}

	secretList := "📋 *Список секретов:*\n\n"
	for i, name := range names {
		secretList += fmt.Sprintf("%d. `%s`\n", i+1, name)
	}
	secretList += "\nИспользуйте: `$SECRET_NAME` в командах"

	params := &telego.SendMessageParams{
		ChatID:    telego.ChatID{ID: chatID},
		Text:      secretList,
		ParseMode: telego.ModeMarkdown,
	}

	_, err = h.connector.bot.SendMessage(ctx, params)
	if err != nil {
		return fmt.Errorf("failed to send secrets list: %w", err)
	}

	return nil
}

// clearSecrets clears all secrets for the session
func (h *CommandHandler) clearSecrets(ctx context.Context, chatID int64, sessionID string) error {
	if h.secrets == nil {
		return h.sendMessage(ctx, chatID, "❌ Хранилище секретов не инициализировано")
	}

	if err := h.secrets.Clear(sessionID); err != nil {
		h.logger.ErrorCtx(ctx, "failed to clear secrets", err,
			logger.Field{Key: "session_id", Value: sessionID})
		return h.sendMessage(ctx, chatID, fmt.Sprintf("❌ Ошибка удаления секретов: %v", err))
	}

	return h.sendMessage(ctx, chatID, "✅ Все секреты удалены")
}

// sendMessage sends a simple text message
func (h *CommandHandler) sendMessage(ctx context.Context, chatID int64, text string) error {
	if h.connector == nil || h.connector.bot == nil {
		return fmt.Errorf("connector or bot not initialized")
	}

	params := &telego.SendMessageParams{
		ChatID: telego.ChatID{ID: chatID},
		Text:   text,
	}

	_, err := h.connector.bot.SendMessage(ctx, params)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	return nil
}
