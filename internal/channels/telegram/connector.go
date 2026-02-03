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
	"fmt"
	"slices"

	"github.com/aatumaykin/nexbot/internal/bus"
	"github.com/aatumaykin/nexbot/internal/config"
	"github.com/aatumaykin/nexbot/internal/logger"
	"github.com/mymmrac/telego"
)

// Connector represents the Telegram bot connector
type Connector struct {
	cfg        config.TelegramConfig
	logger     *logger.Logger
	bus        *bus.MessageBus
	bot        *telego.Bot
	ctx        context.Context
	cancel     context.CancelFunc
	outboundCh <-chan bus.OutboundMessage
}

// New creates a new Telegram connector
func New(cfg config.TelegramConfig, log *logger.Logger, msgBus *bus.MessageBus) *Connector {
	return &Connector{
		cfg:    cfg,
		logger: log,
		bus:    msgBus,
	}
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

	c.bot = bot
	c.ctx, c.cancel = context.WithCancel(ctx)

	// Get bot info
	botUser, err := c.bot.GetMe(c.ctx)
	if err != nil {
		return fmt.Errorf("failed to get bot info: %w", err)
	}

	c.logger.Info("telegram bot initialized",
		logger.Field{Key: "bot_id", Value: botUser.ID},
		logger.Field{Key: "username", Value: botUser.Username})

	// Subscribe to outbound messages
	c.outboundCh = c.bus.SubscribeOutbound(c.ctx)
	go c.handleOutbound()

	// Start long polling for updates
	go c.startLongPolling()

	return nil
}

// Stop gracefully stops the Telegram connector
func (c *Connector) Stop() error {
	c.logger.Info("stopping telegram connector")

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

// isAllowedUser checks if the user is allowed based on the whitelist configuration
func (c *Connector) isAllowedUser(userID string) bool {
	// If no whitelist is configured, allow all users
	if len(c.cfg.AllowedUsers) == 0 {
		return true
	}

	// Check if user ID is in the whitelist
	return slices.Contains(c.cfg.AllowedUsers, userID)
}

// handleUpdate processes a Telegram update and publishes it to the message bus
func (c *Connector) handleUpdate(update telego.Update) error {
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

	// Check whitelist - block unauthorized users
	if !c.isAllowedUser(userID) {
		c.logger.WarnCtx(c.ctx, "message blocked - user not in whitelist",
			logger.Field{Key: "user_id", Value: userID},
			logger.Field{Key: "username", Value: msg.From.Username})

		// Optionally send a message back informing the user
		if msg.Chat.ID != 0 && c.bot != nil {
			notifyParams := telego.SendMessageParams{
				ChatID: telego.ChatID{ID: msg.Chat.ID},
				Text:   "Sorry, you are not authorized to use this bot.",
			}
			_, _ = c.bot.SendMessage(c.ctx, &notifyParams)
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
	if err := c.bus.PublishInbound(*inboundMsg); err != nil {
		return fmt.Errorf("failed to publish inbound message: %w", err)
	}

	c.logger.DebugCtx(c.ctx, "inbound message published",
		logger.Field{Key: "user_id", Value: userID},
		logger.Field{Key: "session_id", Value: sessionID},
		logger.Field{Key: "content", Value: msg.Text})

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
			var chatID int64
			_, err := fmt.Sscanf(msg.SessionID, "%d", &chatID)
			if err != nil {
				c.logger.ErrorCtx(c.ctx, "invalid session ID", err,
					logger.Field{Key: "session_id", Value: msg.SessionID})
				continue
			}

			// Send message to Telegram
			if c.bot == nil {
				c.logger.WarnCtx(c.ctx, "bot is nil, skipping message send")
				continue
			}

			params := telego.SendMessageParams{
				ChatID:    telego.ChatID{ID: chatID},
				Text:      msg.Content,
				ParseMode: "Markdown", // Use Markdown for formatting
			}

			_, err = c.bot.SendMessage(c.ctx, &params)
			if err != nil {
				c.logger.ErrorCtx(c.ctx, "failed to send message to Telegram", err,
					logger.Field{Key: "chat_id", Value: chatID})
				continue
			}

			c.logger.DebugCtx(c.ctx, "outbound message sent to Telegram",
				logger.Field{Key: "chat_id", Value: chatID},
				logger.Field{Key: "user_id", Value: msg.UserID},
				logger.Field{Key: "content", Value: msg.Content})
		}
	}
}

// startLongPolling sets up and runs long polling for Telegram updates
func (c *Connector) startLongPolling() {
	c.logger.Info("starting long polling for telegram updates")

	// Set up long polling with updates
	updates, err := c.bot.UpdatesViaLongPolling(c.ctx, &telego.GetUpdatesParams{
		Timeout: 30, // Poll for 30 seconds
	})
	if err != nil {
		c.logger.ErrorCtx(c.ctx, "failed to start long polling", err)
		return
	}

	for {
		select {
		case <-c.ctx.Done():
			c.logger.Info("long polling stopped")
			return
		case update, ok := <-updates:
			if !ok {
				c.logger.Info("updates channel closed")
				return
			}

			// Process the update
			if err := c.handleUpdate(update); err != nil {
				c.logger.ErrorCtx(c.ctx, "failed to handle update", err)
			}
		}
	}
}
