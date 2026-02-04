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
	"strings"

	"github.com/aatumaykin/nexbot/internal/bus"
	"github.com/aatumaykin/nexbot/internal/config"
	"github.com/aatumaykin/nexbot/internal/llm"
	"github.com/aatumaykin/nexbot/internal/logger"
	"github.com/mymmrac/telego"
)

// Connector represents the Telegram bot connector
type Connector struct {
	cfg        config.TelegramConfig
	logger     *logger.Logger
	bus        *bus.MessageBus
	provider   llm.Provider
	bot        *telego.Bot
	ctx        context.Context
	cancel     context.CancelFunc
	outboundCh <-chan bus.OutboundMessage
	eventCh    <-chan bus.Event
}

// New creates a new Telegram connector
func New(cfg config.TelegramConfig, log *logger.Logger, msgBus *bus.MessageBus, provider llm.Provider) *Connector {
	return &Connector{
		cfg:      cfg,
		logger:   log,
		bus:      msgBus,
		provider: provider,
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

	// Subscribe to events for typing indicator
	c.eventCh = c.bus.SubscribeEvent(c.ctx)
	go c.handleEvents()

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

// handleModelsCommand handles the /models command by fetching available models
// from the LLM provider and sending them directly to Telegram (not to the session).
func (c *Connector) handleModelsCommand(chatID int64) {
	c.logger.DebugCtx(c.ctx, "handling /models command",
		logger.Field{Key: "chat_id", Value: chatID})

	// Get list of models from provider
	models, err := c.provider.ListModels(c.ctx)
	if err != nil {
		c.logger.ErrorCtx(c.ctx, "failed to list models", err,
			logger.Field{Key: "chat_id", Value: chatID})

		// Send error message
		if c.bot != nil {
			errorMsg := "❌ Failed to get available models. Please try again later."
			params := telego.SendMessageParams{
				ChatID: telego.ChatID{ID: chatID},
				Text:   errorMsg,
			}
			_, _ = c.bot.SendMessage(c.ctx, &params)
		}
		return
	}

	// Build formatted message
	var builder strings.Builder
	builder.WriteString("🤖 **Available LLM Models:**\n\n")

	for _, model := range models {
		// Add current indicator if this is the active model
		if model.Current {
			builder.WriteString(fmt.Sprintf("• *%s* ✅\n", model.Name))
		} else {
			builder.WriteString(fmt.Sprintf("• %s\n", model.Name))
		}
		builder.WriteString(fmt.Sprintf("  ID: `%s`\n", model.ID))
		if model.Description != "" {
			builder.WriteString(fmt.Sprintf("  %s\n", model.Description))
		}
		builder.WriteString("\n")
	}

	// Send message to Telegram
	if c.bot != nil {
		params := telego.SendMessageParams{
			ChatID:    telego.ChatID{ID: chatID},
			Text:      builder.String(),
			ParseMode: "Markdown",
		}
		_, err := c.bot.SendMessage(c.ctx, &params)
		if err != nil {
			c.logger.ErrorCtx(c.ctx, "failed to send models list", err,
				logger.Field{Key: "chat_id", Value: chatID})
			return
		}

		c.logger.DebugCtx(c.ctx, "models list sent successfully",
			logger.Field{Key: "chat_id", Value: chatID},
			logger.Field{Key: "models_count", Value: len(models)})
	}
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

	// Check for /new command before whitelist check (allow clearing session for authorized users)
	if msg.Text == "/new" {
		if !c.isAllowedUser(userID) {
			c.logger.WarnCtx(c.ctx, "command blocked - user not in whitelist",
				logger.Field{Key: "user_id", Value: userID},
				logger.Field{Key: "command", Value: "/new"})
			return nil
		}

		// Use chat ID as session ID
		sessionID := fmt.Sprintf("%d", msg.Chat.ID)

		// Create command message with metadata
		inboundMsg := bus.NewInboundMessage(
			bus.ChannelTypeTelegram,
			userID,
			sessionID,
			msg.Text,
			map[string]any{
				"command":    "new_session",
				"message_id": msg.MessageID,
				"chat_id":    msg.Chat.ID,
				"chat_type":  msg.Chat.Type,
				"username":   msg.From.Username,
			},
		)

		// Publish to message bus
		if err := c.bus.PublishInbound(*inboundMsg); err != nil {
			return fmt.Errorf("failed to publish command message: %w", err)
		}

		c.logger.DebugCtx(c.ctx, "new session command published",
			logger.Field{Key: "user_id", Value: userID},
			logger.Field{Key: "session_id", Value: sessionID})

		return nil
	}

	// Check for /models command - lists available LLM models (doesn't go to session)
	if msg.Text == "/models" {
		if !c.isAllowedUser(userID) {
			c.logger.WarnCtx(c.ctx, "command blocked - user not in whitelist",
				logger.Field{Key: "user_id", Value: userID},
				logger.Field{Key: "command", Value: "/models"})
			return nil
		}

		// Handle /models command directly - don't publish to message bus
		c.handleModelsCommand(msg.Chat.ID)
		return nil
	}

	// Check for /status command - shows session and bot status (doesn't go to session)
	if msg.Text == "/status" {
		if !c.isAllowedUser(userID) {
			c.logger.WarnCtx(c.ctx, "command blocked - user not in whitelist",
				logger.Field{Key: "user_id", Value: userID},
				logger.Field{Key: "command", Value: "/status"})
			return nil
		}

		// Use chat ID as session ID
		sessionID := fmt.Sprintf("%d", msg.Chat.ID)

		// Create command message with metadata
		inboundMsg := bus.NewInboundMessage(
			bus.ChannelTypeTelegram,
			userID,
			sessionID,
			msg.Text,
			map[string]any{
				"command":    "status",
				"message_id": msg.MessageID,
				"chat_id":    msg.Chat.ID,
				"chat_type":  msg.Chat.Type,
				"username":   msg.From.Username,
			},
		)

		// Publish to message bus
		if err := c.bus.PublishInbound(*inboundMsg); err != nil {
			return fmt.Errorf("failed to publish command message: %w", err)
		}

		c.logger.DebugCtx(c.ctx, "status command published",
			logger.Field{Key: "user_id", Value: userID},
			logger.Field{Key: "session_id", Value: sessionID})

		return nil
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
				// Send typing indicator when processing starts
				c.sendTypingIndicator(event)
			case bus.EventTypeProcessingEnd:
				// Typing indicator automatically stops when we send a message
				// No action needed here
			}
		}
	}
}

// sendTypingIndicator sends a typing indicator to the specified chat
func (c *Connector) sendTypingIndicator(event bus.Event) {
	// Extract chat ID from session ID
	var chatID int64
	_, err := fmt.Sscanf(event.SessionID, "%d", &chatID)
	if err != nil {
		c.logger.ErrorCtx(c.ctx, "invalid session ID for typing indicator", err,
			logger.Field{Key: "session_id", Value: event.SessionID})
		return
	}

	// Send typing indicator
	if c.bot == nil {
		c.logger.WarnCtx(c.ctx, "bot is nil, skipping typing indicator")
		return
	}

	params := &telego.SendChatActionParams{
		ChatID: telego.ChatID{ID: chatID},
		Action: telego.ChatActionTyping,
	}

	err = c.bot.SendChatAction(c.ctx, params)
	if err != nil {
		c.logger.ErrorCtx(c.ctx, "failed to send typing indicator", err,
			logger.Field{Key: "chat_id", Value: chatID})
		return
	}

	c.logger.DebugCtx(c.ctx, "typing indicator sent",
		logger.Field{Key: "chat_id", Value: chatID},
		logger.Field{Key: "user_id", Value: event.UserID})
}
