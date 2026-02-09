package telegram

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/aatumaykin/nexbot/internal/bus"
	"github.com/aatumaykin/nexbot/internal/logger"
	"github.com/mymmrac/telego"
)

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

	params, err := prepareMediaParams[telego.SendPhotoParams](c, msg, chatID, func(p *telego.SendPhotoParams, f telego.InputFile) {
		p.Photo = f
	})
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
	_, err = c.bot.SendPhoto(sendCtx, params)
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

	params, err := prepareMediaParams[telego.SendDocumentParams](c, msg, chatID, func(p *telego.SendDocumentParams, f telego.InputFile) {
		p.Document = f
	})
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
	_, err = c.bot.SendDocument(sendCtx, params)
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

// getSendTimeout возвращает контекст с таймаутом для отправки
func (c *Connector) getSendTimeout() (context.Context, context.CancelFunc) {
	timeout := time.Duration(c.cfg.SendTimeoutSeconds) * time.Second
	return context.WithTimeout(c.ctx, timeout)
}
