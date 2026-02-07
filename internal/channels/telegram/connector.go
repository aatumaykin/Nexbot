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
	"slices"
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
			// Support both formats: "chat_id" (legacy) and "channel:chat_id" (new)
			var chatID int64
			var err error
			if strings.Contains(msg.SessionID, ":") {
				// New format: "channel:chat_id"
				parts := strings.Split(msg.SessionID, ":")
				if len(parts) != 2 {
					c.logger.ErrorCtx(c.ctx, "invalid session ID format: expected 'channel:chat_id'",
						nil,
						logger.Field{Key: "session_id", Value: msg.SessionID})
					continue
				}
				channel := parts[0]
				chatIDStr := parts[1]

				// Verify channel matches telegram
				if channel != string(bus.ChannelTypeTelegram) {
					c.logger.ErrorCtx(c.ctx, "session ID channel mismatch",
						nil,
						logger.Field{Key: "expected", Value: bus.ChannelTypeTelegram},
						logger.Field{Key: "got", Value: channel},
						logger.Field{Key: "session_id", Value: msg.SessionID})
					continue
				}

				_, err = fmt.Sscanf(chatIDStr, "%d", &chatID)
				if err != nil {
					c.logger.ErrorCtx(c.ctx, "invalid chat ID in session ID", err,
						logger.Field{Key: "session_id", Value: msg.SessionID})
					continue
				}
			} else {
				// Legacy format: "chat_id"
				_, err = fmt.Sscanf(msg.SessionID, "%d", &chatID)
				if err != nil {
					c.logger.ErrorCtx(c.ctx, "invalid session ID", err,
						logger.Field{Key: "session_id", Value: msg.SessionID})
					continue
				}
			}

			// Send message to Telegram
			if c.bot == nil {
				c.logger.WarnCtx(c.ctx, "bot is nil, skipping message send")
				continue
			}

			params := telego.SendMessageParams{
				ChatID:    telego.ChatID{ID: chatID},
				Text:      msg.Content,
				ParseMode: telego.ModeMarkdown,
			}

			_, err = c.bot.SendMessage(c.ctx, &params)
			if err != nil {
				// Парсим ошибку telego для получения деталей
				var telErr *telegoapi.Error
				isMarkdownError := false

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

					// Проверяем, если это ошибка парсинга markdown (400 Bad Request)
					if telErr.ErrorCode == 400 {
						desc := telErr.Description
						isMarkdownError = strings.Contains(desc, "can't parse entities") ||
							strings.Contains(desc, "Can't find end of the entity") ||
							strings.Contains(desc, "wrong number of entities") ||
							strings.Contains(desc, "specified new message entity")
					}

					// Fallback: отправка без форматирования при ошибке markdown
					if isMarkdownError {
						c.logger.WarnCtx(c.ctx, "markdown parse error, retrying without formatting",
							logger.Field{Key: "chat_id", Value: chatID},
							logger.Field{Key: "correlation_id", Value: msg.CorrelationID},
							logger.Field{Key: "error", Value: telErr.Description})

						params.ParseMode = ""
						_, fallbackErr := c.bot.SendMessage(c.ctx, &params)
						if fallbackErr == nil {
							c.logger.InfoCtx(c.ctx, "message sent with fallback (no formatting)",
								logger.Field{Key: "chat_id", Value: chatID},
								logger.Field{Key: "correlation_id", Value: msg.CorrelationID})

							result := bus.MessageSendResult{
								CorrelationID: msg.CorrelationID,
								ChannelType:   bus.ChannelTypeTelegram,
								Success:       true,
								Timestamp:     time.Now(),
							}

							if pubErr := c.bus.PublishSendResult(result); pubErr != nil {
								c.logger.ErrorCtx(c.ctx, "failed to publish send result", pubErr,
									logger.Field{Key: "correlation_id", Value: msg.CorrelationID})
							}
							continue
						}

						c.logger.ErrorCtx(c.ctx, "fallback send also failed", fallbackErr,
							logger.Field{Key: "chat_id", Value: chatID},
							logger.Field{Key: "correlation_id", Value: msg.CorrelationID})
					}

					result := bus.MessageSendResult{
						CorrelationID: msg.CorrelationID,
						ChannelType:   bus.ChannelTypeTelegram,
						Success:       false,
						Error:         details,
						Timestamp:     time.Now(),
					}

					if pubErr := c.bus.PublishSendResult(result); pubErr != nil {
						c.logger.ErrorCtx(c.ctx, "failed to publish send result", pubErr,
							logger.Field{Key: "correlation_id", Value: msg.CorrelationID})
					}
				}

				c.logger.ErrorCtx(c.ctx, "failed to send message to Telegram", err,
					logger.Field{Key: "chat_id", Value: chatID},
					logger.Field{Key: "correlation_id", Value: msg.CorrelationID})
				continue
			}

			// Успешная отправка
			result := bus.MessageSendResult{
				CorrelationID: msg.CorrelationID,
				ChannelType:   bus.ChannelTypeTelegram,
				Success:       true,
				Timestamp:     time.Now(),
			}

			if pubErr := c.bus.PublishSendResult(result); pubErr != nil {
				c.logger.ErrorCtx(c.ctx, "failed to publish send result", pubErr,
					logger.Field{Key: "correlation_id", Value: msg.CorrelationID})
			}

			c.logger.DebugCtx(c.ctx, "outbound message sent to Telegram",
				logger.Field{Key: "chat_id", Value: chatID},
				logger.Field{Key: "user_id", Value: msg.UserID},
				logger.Field{Key: "correlation_id", Value: msg.CorrelationID},
				logger.Field{Key: "content", Value: msg.Content})
		}
	}
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
