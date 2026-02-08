// Package telegram provides Telegram Bot integration using the Telego library.
// It handles message routing between Telegram and the internal message bus,
// supporting basic chat functionality without tools.
//
// Features:
//   - Long polling for receiving updates
//   - Whitelist-based user authorization
//   - Graceful shutdown handling
//   - Integration with internal message bus
package telegram

import (
	"context"
	"errors"
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/aatumaykin/nexbot/internal/bus"
	"github.com/aatumaykin/nexbot/internal/channels"
	"github.com/aatumaykin/nexbot/internal/config"
	"github.com/aatumaykin/nexbot/internal/constants"
	"github.com/aatumaykin/nexbot/internal/logger"
	"github.com/mymmrac/telego"
	telegoapi "github.com/mymmrac/telego/telegoapi"
)

// Connector represents the Telegram bot connector
type Connector struct {
	cfg             config.TelegramConfig
	logger          *logger.Logger
	bus             *bus.MessageBus
	bot             BotInterface
	ctx             context.Context
	cancel          context.CancelFunc
	outboundCh      <-chan bus.OutboundMessage
	eventCh         <-chan bus.Event
	commandHandler  *CommandHandler
	typingManager   *TypingManager
	longPollManager *LongPollManager
	updateHandler   *UpdateHandler
}

// New creates a new Telegram connector
func New(cfg config.TelegramConfig, log *logger.Logger, msgBus *bus.MessageBus) *Connector {
	conn := &Connector{
		cfg:             cfg,
		logger:          log,
		bus:             msgBus,
		commandHandler:  NewCommandHandler(log, msgBus),
		typingManager:   NewTypingManager(nil, log),
		longPollManager: NewLongPollManager(nil, nil, log),
		updateHandler:   NewUpdateHandler(nil, log, msgBus),
	}
	conn.longPollManager.connector = conn
	conn.updateHandler.connector = conn
	return conn
}

// Start initializes the Telegram bot and starts listening for updates
func (c *Connector) Start(ctx context.Context) error {
	c.logger.Info("starting telegram connector",
		logger.Field{Key: "enabled", Value: c.cfg.Enabled})

	if !c.cfg.Enabled {
		c.logger.Info("telegram connector disabled in config")
		return nil
	}

	// Validate configuration
	if err := c.validateConfig(); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	// Initialize Telegram bot
	bot, err := telego.NewBot(c.cfg.Token)
	if err != nil {
		return fmt.Errorf("failed to initialize telegram bot: %w", err)
	}

	c.bot = NewBotAdapter(bot)
	c.ctx, c.cancel = context.WithCancel(ctx)

	// Update typing manager with bot
	c.typingManager.SetContext(c.ctx)
	c.typingManager.bot = c.bot

	// Update long poll manager with bot and context
	c.longPollManager.SetContext(c.ctx)
	c.longPollManager.bot = c.bot

	// Get bot info
	botUser, err := c.bot.GetMe(c.ctx)
	if err != nil {
		return fmt.Errorf("failed to get bot info: %w", err)
	}

	c.logger.Info("telegram bot initialized",
		logger.Field{Key: "bot_id", Value: botUser.ID},
		logger.Field{Key: "username", Value: botUser.Username})

	if err := c.registerCommands(); err != nil {
		c.logger.ErrorCtx(c.ctx, "failed to register bot commands", err)
	}

	if err := c.sendStartupMessage(); err != nil {
		c.logger.ErrorCtx(c.ctx, "failed to send startup message", err)
	}

	// Subscribe to outbound messages
	c.outboundCh = c.bus.SubscribeOutbound(c.ctx)
	go c.handleOutbound()

	// Subscribe to events for typing indicator
	c.eventCh = c.bus.SubscribeEvent(c.ctx)
	go c.handleEvents()

	// Start long polling for updates
	go c.longPollManager.Start()

	return nil
}

// Stop gracefully stops the Telegram connector
func (c *Connector) Stop() error {
	c.logger.Info("stopping telegram connector")

	// Stop all typing indicators
	c.typingManager.StopAll()

	// Cancel context to stop all goroutines (long polling, outbound handler)
	if c.cancel != nil {
		c.cancel()
	}

	// Clear bot reference
	c.bot = nil

	// Clear channel reference
	c.outboundCh = nil

	c.logger.Info("telegram connector stopped gracefully")

	return nil
}

// validateConfig validates the Telegram configuration
func (c *Connector) validateConfig() error {
	if c.cfg.Token == "" {
		return fmt.Errorf("telegram token is required")
	}

	return nil
}

// registerCommands registers bot commands with Telegram
func (c *Connector) registerCommands() error {
	if c.bot == nil {
		return fmt.Errorf("bot is not initialized")
	}

	commands := &telego.SetMyCommandsParams{
		Commands: []telego.BotCommand{
			{Command: "new", Description: "Start a new session (clear history)"},
			{Command: "status", Description: "Show session and bot status"},
			{Command: "restart", Description: "Restart bot"},
		},
	}

	err := c.bot.SetMyCommands(c.ctx, commands)
	if err != nil {
		return fmt.Errorf("failed to register commands: %w", err)
	}

	c.logger.Info("bot commands registered successfully")

	return nil
}

// isAllowedUser checks if the user is allowed based on the whitelist configuration
func (c *Connector) isAllowedUser(userID string) bool {
	// If no whitelist is configured, allow all users
	if len(c.cfg.AllowedUsers) == 0 {
		return true
	}

	// Check if user ID is in the whitelist
	return slices.Contains(c.cfg.AllowedUsers, userID)
}

// sendStartupMessage sends a startup message to all allowed users
func (c *Connector) sendStartupMessage() error {
	if len(c.cfg.AllowedUsers) == 0 {
		c.logger.Info("no allowed users configured, skipping startup message")
		return nil
	}

	message := constants.MsgTelegramStartup

	for _, userID := range c.cfg.AllowedUsers {
		var chatID int64
		_, err := fmt.Sscanf(userID, "%d", &chatID)
		if err != nil {
			c.logger.WarnCtx(c.ctx, "invalid user ID in allowed_users",
				logger.Field{Key: "user_id", Value: userID})
			continue
		}

		params := telego.SendMessageParams{
			ChatID: telego.ChatID{ID: chatID},
			Text:   message,
		}

		_, err = c.bot.SendMessage(c.ctx, &params)
		if err != nil {
			c.logger.ErrorCtx(c.ctx, "failed to send startup message", err,
				logger.Field{Key: "user_id", Value: userID})
			continue
		}

		c.logger.InfoCtx(c.ctx, "startup message sent",
			logger.Field{Key: "user_id", Value: userID})
	}

	return nil
}

// handleOutbound processes outbound messages from the message bus and sends them to Telegram
func (c *Connector) handleOutbound() {
	c.logger.Info("outbound message handler started")

	for {
		select {
		case <-c.ctx.Done():
			c.logger.Info("outbound message handler stopped")
			return
		case msg, ok := <-c.outboundCh:
			if !ok {
				c.logger.Info("outbound channel closed")
				return
			}

			// Only process Telegram messages
			if msg.ChannelType != bus.ChannelTypeTelegram {
				continue
			}

			// Extract chat ID from session ID
			chatID, err := c.extractChatID(msg.SessionID)
			if err != nil {
				c.logger.ErrorCtx(c.ctx, "failed to extract chat ID", err,
					logger.Field{Key: "session_id", Value: msg.SessionID},
					logger.Field{Key: "correlation_id", Value: msg.CorrelationID})
				continue
			}

			// Send message to Telegram
			if c.bot == nil {
				c.logger.WarnCtx(c.ctx, "bot is nil, skipping message send")
				continue
			}

			// Route message based on type
			switch msg.Type {
			case bus.MessageTypeText:
				c.sendTextMessage(msg, chatID)
			case bus.MessageTypeEdit:
				if !c.cfg.EnableInlineUpdates {
					c.logger.WarnCtx(c.ctx, "inline updates disabled in config",
						logger.Field{Key: "message_type", Value: msg.Type},
						logger.Field{Key: "correlation_id", Value: msg.CorrelationID})
					c.publishResult(msg, chatID, false, fmt.Errorf("inline updates disabled"))
					continue
				}
				c.editMessage(msg, chatID)
			case bus.MessageTypeDelete:
				if !c.cfg.EnableInlineUpdates {
					c.logger.WarnCtx(c.ctx, "inline updates disabled in config",
						logger.Field{Key: "message_type", Value: msg.Type},
						logger.Field{Key: "correlation_id", Value: msg.CorrelationID})
					c.publishResult(msg, chatID, false, fmt.Errorf("inline updates disabled"))
					continue
				}
				c.deleteMessage(msg, chatID)
			case bus.MessageTypePhoto:
				c.sendPhoto(msg, chatID)
			case bus.MessageTypeDocument:
				c.sendDocument(msg, chatID)
			default:
				c.logger.WarnCtx(c.ctx, "unknown message type",
					logger.Field{Key: "message_type", Value: msg.Type},
					logger.Field{Key: "correlation_id", Value: msg.CorrelationID})
				c.publishResult(msg, chatID, false, fmt.Errorf("unknown message type: %s", msg.Type))
			}
		}
	}
}

// extractChatID extracts chat ID from session ID
// Format: "telegram:chat_id"
func (c *Connector) extractChatID(sessionID string) (int64, error) {
	parts := strings.Split(sessionID, ":")
	if len(parts) != 2 {
		return 0, fmt.Errorf("invalid session ID format: expected 'channel:chat_id', got: %s", sessionID)
	}

	channel := parts[0]
	chatIDStr := parts[1]

	// Verify channel matches telegram
	if channel != string(bus.ChannelTypeTelegram) {
		return 0, fmt.Errorf("session ID channel mismatch: expected %s, got %s",
			bus.ChannelTypeTelegram, channel)
	}

	var chatID int64
	_, err := fmt.Sscanf(chatIDStr, "%d", &chatID)
	if err != nil {
		return 0, fmt.Errorf("invalid chat ID in session ID: %w", err)
	}

	return chatID, nil
}

// getSendTimeout возвращает контекст с таймаутом для отправки
func (c *Connector) getSendTimeout() (context.Context, context.CancelFunc) {
	timeout := time.Duration(c.cfg.SendTimeoutSeconds) * time.Second
	return context.WithTimeout(c.ctx, timeout)
}

// sendTextMessage sends a text message to Telegram
func (c *Connector) sendTextMessage(msg bus.OutboundMessage, chatID int64) {
	// Prepare message with smart content detection
	params, err := c.prepareMessage(msg.Content, chatID)
	if err != nil {
		c.logger.ErrorCtx(c.ctx, "failed to prepare text message", err,
			logger.Field{Key: "chat_id", Value: chatID},
			logger.Field{Key: "correlation_id", Value: msg.CorrelationID})
		c.publishResult(msg, chatID, false, err)
		return
	}

	// Attach inline keyboard if enabled and present
	if msg.InlineKeyboard != nil && c.cfg.EnableInlineKeyboard {
		params.ReplyMarkup = c.buildInlineKeyboard(msg.InlineKeyboard)
	}

	// Try to send with detected formatting and timeout
	sendCtx, cancel := c.getSendTimeout()
	defer cancel()
	_, err = c.bot.SendMessage(sendCtx, &params)
	if err != nil {
		// Smart fallback for markdown errors
		c.handleSendError(err, msg, chatID, params)
		return
	}

	// Successful send - publish result immediately
	c.publishResult(msg, chatID, true, nil)
}

// editMessage edits an existing message in Telegram
func (c *Connector) editMessage(msg bus.OutboundMessage, chatID int64) {
	if msg.MessageID == "" {
		c.logger.ErrorCtx(c.ctx, "message ID is required for edit", nil,
			logger.Field{Key: "correlation_id", Value: msg.CorrelationID})
		c.publishResult(msg, chatID, false, fmt.Errorf("message ID is required for edit"))
		return
	}

	// Prepare message with smart content detection
	params := c.prepareEditMessageParams(msg.Content, chatID, msg.MessageID)

	// Attach inline keyboard if enabled and present
	if msg.InlineKeyboard != nil && c.cfg.EnableInlineKeyboard {
		params.ReplyMarkup = c.buildInlineKeyboard(msg.InlineKeyboard)
	}

	// Try to send with detected formatting and timeout
	sendCtx, cancel := c.getSendTimeout()
	defer cancel()
	_, err := c.bot.EditMessageText(sendCtx, &params)
	if err != nil {
		c.handleSendError(err, msg, chatID, telego.SendMessageParams{}) // params not needed for edit fallback
		return
	}

	// Successful send - publish result immediately
	c.publishResult(msg, chatID, true, nil)
}

// deleteMessage deletes an existing message from Telegram
func (c *Connector) deleteMessage(msg bus.OutboundMessage, chatID int64) {
	if msg.MessageID == "" {
		c.logger.ErrorCtx(c.ctx, "message ID is required for delete", nil,
			logger.Field{Key: "correlation_id", Value: msg.CorrelationID})
		c.publishResult(msg, chatID, false, fmt.Errorf("message ID is required for delete"))
		return
	}

	messageID, err := strconv.Atoi(msg.MessageID)
	if err != nil {
		c.logger.ErrorCtx(c.ctx, "invalid message ID format", err,
			logger.Field{Key: "message_id", Value: msg.MessageID},
			logger.Field{Key: "correlation_id", Value: msg.CorrelationID})
		c.publishResult(msg, chatID, false, fmt.Errorf("invalid message ID format: %w", err))
		return
	}

	params := telego.DeleteMessageParams{
		ChatID:    telego.ChatID{ID: chatID},
		MessageID: messageID,
	}

	err = c.bot.DeleteMessage(c.ctx, &params)
	if err != nil {
		c.logger.ErrorCtx(c.ctx, "failed to delete message", err,
			logger.Field{Key: "chat_id", Value: chatID},
			logger.Field{Key: "message_id", Value: msg.MessageID},
			logger.Field{Key: "correlation_id", Value: msg.CorrelationID})
		c.publishResult(msg, chatID, false, err)
		return
	}

	// Successful delete - publish result immediately
	c.publishResult(msg, chatID, true, nil)
}

// sendPhoto sends a photo message to Telegram
func (c *Connector) sendPhoto(msg bus.OutboundMessage, chatID int64) {
	if msg.Media == nil {
		c.logger.ErrorCtx(c.ctx, "media data is required for photo message", nil,
			logger.Field{Key: "correlation_id", Value: msg.CorrelationID})
		c.publishResult(msg, chatID, false, fmt.Errorf("media data is required for photo message"))
		return
	}

	params, err := c.preparePhotoParams(msg, chatID)
	if err != nil {
		c.logger.ErrorCtx(c.ctx, "failed to prepare photo message", err,
			logger.Field{Key: "chat_id", Value: chatID},
			logger.Field{Key: "correlation_id", Value: msg.CorrelationID})
		c.publishResult(msg, chatID, false, err)
		return
	}

	// Attach inline keyboard if enabled and present
	if msg.InlineKeyboard != nil && c.cfg.EnableInlineKeyboard {
		params.ReplyMarkup = c.buildInlineKeyboard(msg.InlineKeyboard)
	}

	// Send with timeout
	sendCtx, cancel := c.getSendTimeout()
	defer cancel()
	_, err = c.bot.SendPhoto(sendCtx, &params)
	if err != nil {
		c.logger.ErrorCtx(c.ctx, "failed to send photo", err,
			logger.Field{Key: "chat_id", Value: chatID},
			logger.Field{Key: "correlation_id", Value: msg.CorrelationID})
		c.publishResult(msg, chatID, false, err)
		return
	}

	// Successful send - publish result immediately
	c.publishResult(msg, chatID, true, nil)
}

// sendDocument sends a document message to Telegram
func (c *Connector) sendDocument(msg bus.OutboundMessage, chatID int64) {
	if msg.Media == nil {
		c.logger.ErrorCtx(c.ctx, "media data is required for document message", nil,
			logger.Field{Key: "correlation_id", Value: msg.CorrelationID})
		c.publishResult(msg, chatID, false, fmt.Errorf("media data is required for document message"))
		return
	}

	params, err := c.prepareDocumentParams(msg, chatID)
	if err != nil {
		c.logger.ErrorCtx(c.ctx, "failed to prepare document message", err,
			logger.Field{Key: "chat_id", Value: chatID},
			logger.Field{Key: "correlation_id", Value: msg.CorrelationID})
		c.publishResult(msg, chatID, false, err)
		return
	}

	// Attach inline keyboard if enabled and present
	if msg.InlineKeyboard != nil && c.cfg.EnableInlineKeyboard {
		params.ReplyMarkup = c.buildInlineKeyboard(msg.InlineKeyboard)
	}

	// Send with timeout
	sendCtx, cancel := c.getSendTimeout()
	defer cancel()
	_, err = c.bot.SendDocument(sendCtx, &params)
	if err != nil {
		c.logger.ErrorCtx(c.ctx, "failed to send document", err,
			logger.Field{Key: "chat_id", Value: chatID},
			logger.Field{Key: "correlation_id", Value: msg.CorrelationID})
		c.publishResult(msg, chatID, false, err)
		return
	}

	// Successful send - publish result immediately
	c.publishResult(msg, chatID, true, nil)
}

// prepareEditMessageParams prepares parameters for editing a message
func (c *Connector) prepareEditMessageParams(content string, chatID int64, messageID string) telego.EditMessageTextParams {
	messageIDInt, err := strconv.Atoi(messageID)
	if err != nil {
		// If conversion fails, we'll let the API call handle the error
		messageIDInt = 0
	}

	params := telego.EditMessageTextParams{
		ChatID:    telego.ChatID{ID: chatID},
		MessageID: messageIDInt,
		Text:      content,
	}

	// Detect content type
	contentType := DetectContentType(content)

	switch contentType {
	case ContentTypeCode:
		// Code content - use HTML for better code block support
		params.ParseMode = telego.ModeHTML
		params.Text = MarkdownToHTML(content)
	case ContentTypeMarkdown:
		// Markdown content - try HTML first (more robust)
		params.ParseMode = telego.ModeHTML
		params.Text = MarkdownToHTML(content)
	case ContentTypePlain:
		// Plain text - no formatting
		params.ParseMode = ""
	default:
		// Default: no formatting
		params.ParseMode = ""
	}

	return params
}

// preparePhotoParams prepares parameters for sending a photo
func (c *Connector) preparePhotoParams(msg bus.OutboundMessage, chatID int64) (telego.SendPhotoParams, error) {
	params := telego.SendPhotoParams{
		ChatID: telego.ChatID{ID: chatID},
	}

	// Set caption if provided
	if msg.Content != "" {
		params.Caption = msg.Content
	}

	media := msg.Media

	// Priority order: LocalPath > FileID > URL
	if media.LocalPath != "" {
		if !c.isValidFilePath(media.LocalPath) {
			return params, fmt.Errorf("invalid file path: %s", media.LocalPath)
		}

		// Open file for reading
		file, err := os.Open(media.LocalPath)
		if err != nil {
			return params, fmt.Errorf("failed to open file: %w", err)
		}
		defer file.Close()

		params.Photo = telego.InputFile{File: file}
	} else if media.FileID != "" {
		params.Photo = telego.InputFile{FileID: media.FileID}
	} else if media.URL != "" {
		params.Photo = telego.InputFile{URL: media.URL}
	} else {
		return params, fmt.Errorf("no valid media source provided (local_path, file_id, or url)")
	}

	return params, nil
}

// prepareDocumentParams prepares parameters for sending a document
func (c *Connector) prepareDocumentParams(msg bus.OutboundMessage, chatID int64) (telego.SendDocumentParams, error) {
	params := telego.SendDocumentParams{
		ChatID: telego.ChatID{ID: chatID},
	}

	// Set caption if provided
	if msg.Content != "" {
		params.Caption = msg.Content
	}

	media := msg.Media

	// Priority order: LocalPath > FileID > URL
	if media.LocalPath != "" {
		if !c.isValidFilePath(media.LocalPath) {
			return params, fmt.Errorf("invalid file path: %s", media.LocalPath)
		}

		// Open file for reading
		file, err := os.Open(media.LocalPath)
		if err != nil {
			return params, fmt.Errorf("failed to open file: %w", err)
		}
		defer file.Close()

		params.Document = telego.InputFile{File: file}
	} else if media.FileID != "" {
		params.Document = telego.InputFile{FileID: media.FileID}
	} else if media.URL != "" {
		params.Document = telego.InputFile{URL: media.URL}
	} else {
		return params, fmt.Errorf("no valid media source provided (local_path, file_id, or url)")
	}

	return params, nil
}

// isValidFilePath validates a file path
func (c *Connector) isValidFilePath(path string) bool {
	if path == "" {
		return false
	}

	// Check for absolute path
	if strings.HasPrefix(path, "/") {
		return true
	}

	// Check for relative path starting with . or ..
	if strings.HasPrefix(path, "./") || strings.HasPrefix(path, "../") {
		return true
	}

	// Path with just filename is also valid
	return true
}

// buildInlineKeyboard converts an InlineKeyboard to Telegram's InlineKeyboardMarkup format
func (c *Connector) buildInlineKeyboard(keyboard *bus.InlineKeyboard) *telego.InlineKeyboardMarkup {
	if keyboard == nil {
		return nil
	}

	markup := &telego.InlineKeyboardMarkup{
		InlineKeyboard: make([][]telego.InlineKeyboardButton, len(keyboard.Rows)),
	}

	for i, row := range keyboard.Rows {
		buttons := make([]telego.InlineKeyboardButton, len(row))
		for j, button := range row {
			buttons[j] = telego.InlineKeyboardButton{
				Text:         button.Text,
				CallbackData: button.Data,
			}
		}
		markup.InlineKeyboard[i] = buttons
	}

	return markup
}

// tryFallbacks attempts to send message with different fallback strategies
func (c *Connector) tryFallbacks(msg bus.OutboundMessage, chatID int64, originalErr error) bool {
	c.logger.InfoCtx(c.ctx, "trying HTML fallback")
	htmlContent := MarkdownToHTML(msg.Content)
	htmlParams, _ := c.prepareMessage(htmlContent, chatID)
	_, htmlErr := c.bot.SendMessage(c.ctx, &htmlParams)
	if htmlErr == nil {
		c.logger.InfoCtx(c.ctx, "message sent with HTML fallback")
		c.publishResult(msg, chatID, true, nil)
		return true
	}

	c.logger.WarnCtx(c.ctx, "HTML fallback failed, trying plain text")
	plainContent := StripFormatting(msg.Content)
	plainParams, _ := c.prepareMessage(plainContent, chatID)
	_, plainErr := c.bot.SendMessage(c.ctx, &plainParams)
	if plainErr == nil {
		c.logger.InfoCtx(c.ctx, "message sent with plain text fallback")
		c.publishResult(msg, chatID, true, nil)
		return true
	}

	c.logger.ErrorCtx(c.ctx, "all fallbacks failed", originalErr,
		logger.Field{Key: "chat_id", Value: chatID},
		logger.Field{Key: "correlation_id", Value: msg.CorrelationID})
	c.publishResult(msg, chatID, false, originalErr)
	return false
}

// publishResult публикует результат отправки сообщения
func (c *Connector) publishResult(msg bus.OutboundMessage, chatID int64, success bool, err error) {
	result := bus.MessageSendResult{
		CorrelationID: msg.CorrelationID,
		ChannelType:   bus.ChannelTypeTelegram,
		Success:       success,
		Timestamp:     time.Now(),
	}

	if !success && err != nil {
		var telErr *telegoapi.Error

		if errors.As(err, &telErr) {
			details := &channels.TelegramErrorDetails{
				ErrorCode:       telErr.ErrorCode,
				Description:     telErr.Description,
				RetryAfterSec:   0,
				OriginalMessage: msg.Content,
				ChatID:          chatID,
				Timestamp:       time.Now(),
			}

			if telErr.Parameters != nil {
				details.RetryAfterSec = telErr.Parameters.RetryAfter
			}

			result.Error = details
		}
	}

	if pubErr := c.bus.PublishSendResult(result); pubErr != nil {
		c.logger.ErrorCtx(c.ctx, "failed to publish send result", pubErr,
			logger.Field{Key: "correlation_id", Value: msg.CorrelationID})
	}
}

// handleSendError обрабатывает ошибки отправки с smart fallback для markdown
func (c *Connector) handleSendError(err error, msg bus.OutboundMessage, chatID int64, params telego.SendMessageParams) {
	var telErr *telegoapi.Error

	if errors.As(err, &telErr) {
		details := &channels.TelegramErrorDetails{
			ErrorCode:       telErr.ErrorCode,
			Description:     telErr.Description,
			RetryAfterSec:   0,
			OriginalMessage: msg.Content,
			ChatID:          chatID,
			Timestamp:       time.Now(),
		}

		if telErr.Parameters != nil {
			details.RetryAfterSec = telErr.Parameters.RetryAfter
		}

		isMarkdownError := false

		if telErr.ErrorCode == 400 {
			desc := telErr.Description
			isMarkdownError = strings.Contains(desc, "can't parse entities") ||
				strings.Contains(desc, "Can't find end of the entity") ||
				strings.Contains(desc, "wrong number of entities") ||
				strings.Contains(desc, "specified new message entity")
		}

		// Smart fallback: try different parsing modes based on content type
		if isMarkdownError {
			c.logger.WarnCtx(c.ctx, "markdown parse error, trying fallback strategies",
				logger.Field{Key: "chat_id", Value: chatID},
				logger.Field{Key: "correlation_id", Value: msg.CorrelationID},
				logger.Field{Key: "error", Value: telErr.Description})

			// Fallback 1: Try HTML
			c.logger.InfoCtx(c.ctx, "trying HTML fallback")
			htmlContent := MarkdownToHTML(msg.Content)
			params.ParseMode = telego.ModeHTML
			params.Text = htmlContent
			_, htmlErr := c.bot.SendMessage(c.ctx, &params)
			if htmlErr == nil {
				c.logger.InfoCtx(c.ctx, "message sent with HTML fallback")
				c.publishResult(msg, chatID, true, nil)
				return
			}

			c.logger.WarnCtx(c.ctx, "HTML fallback failed, trying plain text")
			plainContent := StripFormatting(msg.Content)
			params.ParseMode = ""
			params.Text = plainContent
			_, plainErr := c.bot.SendMessage(c.ctx, &params)
			if plainErr == nil {
				c.logger.InfoCtx(c.ctx, "message sent with plain text fallback")
				c.publishResult(msg, chatID, true, nil)
				return
			}

			c.logger.ErrorCtx(c.ctx, "all markdown fallbacks failed", plainErr,
				logger.Field{Key: "chat_id", Value: chatID},
				logger.Field{Key: "correlation_id", Value: msg.CorrelationID})
			c.publishResult(msg, chatID, false, plainErr)
		}

		c.publishResult(msg, chatID, false, err)
	}
}

// prepareMessage подготавливает параметры сообщения с определением типа контента
func (c *Connector) prepareMessage(content string, chatID int64) (telego.SendMessageParams, error) {
	params := telego.SendMessageParams{
		ChatID: telego.ChatID{ID: chatID},
		Text:   content,
	}

	// Apply quiet mode - disable notifications
	if c.cfg.QuietMode {
		params.DisableNotification = true
	}

	// Detect content type
	contentType := DetectContentType(content)

	switch contentType {
	case ContentTypeCode:
		// Code content - use HTML for better code block support
		params.ParseMode = telego.ModeHTML
		params.Text = MarkdownToHTML(content)
	case ContentTypeMarkdown:
		// Markdown content - try HTML first (more robust)
		params.ParseMode = telego.ModeHTML
		params.Text = MarkdownToHTML(content)
	case ContentTypePlain:
		// Plain text - no formatting
		params.ParseMode = ""
	default:
		// Default: use parse mode from config
		switch c.cfg.DefaultParseMode {
		case "markdown":
			params.ParseMode = telego.ModeMarkdown
		case "html":
			params.ParseMode = telego.ModeHTML
		default:
			params.ParseMode = ""
		}
	}

	return params, nil
}

// handleEvents processes lifecycle events from the message bus
func (c *Connector) handleEvents() {
	c.logger.Info("event handler started")

	for {
		select {
		case <-c.ctx.Done():
			c.logger.Info("event handler stopped")
			return
		case event, ok := <-c.eventCh:
			if !ok {
				c.logger.Info("event channel closed")
				return
			}

			// Only process Telegram events
			if event.ChannelType != bus.ChannelTypeTelegram {
				continue
			}

			switch event.Type {
			case bus.EventTypeProcessingStart:
				// Start periodic typing indicator
				c.typingManager.Start(event)
			case bus.EventTypeProcessingEnd:
				// Stop typing indicator
				c.typingManager.Stop(event)
			}
		}
	}
}

// sendTypingIndicator sends a typing indicator to the specified chat.
// This is a public wrapper for testing purposes.
func (c *Connector) sendTypingIndicator(event bus.Event) {
	c.typingManager.Send(event)
}

// handleUpdate processes a Telegram update.
// This is a public wrapper for testing purposes.
func (c *Connector) handleUpdate(update telego.Update) error {
	return c.updateHandler.Handle(update)
}
