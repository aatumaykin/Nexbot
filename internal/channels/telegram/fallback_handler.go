package telegram

import (
	"errors"
	"strings"
	"time"

	"github.com/aatumaykin/nexbot/internal/bus"
	"github.com/aatumaykin/nexbot/internal/channels"
	"github.com/aatumaykin/nexbot/internal/logger"
	"github.com/mymmrac/telego"
	telegoapi "github.com/mymmrac/telego/telegoapi"
)

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
			return
		}

		// Publish result for non-markdown Telegram API errors
		c.publishResult(msg, chatID, false, err)
		return
	}

	// Publish result for non-Telegram errors
	c.publishResult(msg, chatID, false, err)
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
