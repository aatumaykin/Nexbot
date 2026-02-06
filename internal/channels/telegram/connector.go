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
	"sync"
	"time"

	"github.com/aatumaykin/nexbot/internal/bus"
	"github.com/aatumaykin/nexbot/internal/config"
	"github.com/aatumaykin/nexbot/internal/constants"
	"github.com/aatumaykin/nexbot/internal/logger"
	"github.com/mymmrac/telego"
)

// Connector represents the Telegram bot connector
type Connector struct {
	cfg            config.TelegramConfig
	logger         *logger.Logger
	bus            *bus.MessageBus
	bot            *telego.Bot
	ctx            context.Context
	cancel         context.CancelFunc
	outboundCh     <-chan bus.OutboundMessage
	eventCh        <-chan bus.Event
	typingLock     sync.RWMutex
	typingCancel   map[string]context.CancelFunc
	commandHandler *CommandHandler
}

// New creates a new Telegram connector
func New(cfg config.TelegramConfig, log *logger.Logger, msgBus *bus.MessageBus) *Connector {
	return &Connector{
		cfg:            cfg,
		logger:         log,
		bus:            msgBus,
		typingCancel:   make(map[string]context.CancelFunc),
		commandHandler: NewCommandHandler(log, msgBus),
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
	go c.startLongPolling()

	return nil
}

// Stop gracefully stops the Telegram connector
func (c *Connector) Stop() error {
	c.logger.Info("stopping telegram connector")

	// Stop all typing indicators
	c.typingLock.Lock()
	for sessionID, cancel := range c.typingCancel {
		cancel()
		delete(c.typingCancel, sessionID)
	}
	c.typingLock.Unlock()

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
		return c.commandHandler.HandleCommand(c.ctx, c.isAllowedUser, msg, "new_session", userID)
	}

	// Check for /status command - shows session and bot status (doesn't go to session)
	if msg.Text == "/status" {
		return c.commandHandler.HandleCommand(c.ctx, c.isAllowedUser, msg, "status", userID)
	}

	// Check for /restart command - restarts the bot
	if msg.Text == "/restart" {
		return c.commandHandler.HandleCommand(c.ctx, c.isAllowedUser, msg, "restart", userID)
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
			_, err := c.bot.SendMessage(c.ctx, &notifyParams)
			if err != nil {
				c.logger.ErrorCtx(c.ctx, "failed to send notification", err)
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
				ParseMode: telego.ModeMarkdown,
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
				// Start periodic typing indicator
				c.startTypingIndicator(event)
			case bus.EventTypeProcessingEnd:
				// Stop typing indicator
				c.stopTypingIndicator(event)
			}
		}
	}
}

// startTypingIndicator starts a periodic typing indicator for the specified chat
func (c *Connector) startTypingIndicator(event bus.Event) {
	// Extract chat ID from session ID
	var chatID int64
	_, err := fmt.Sscanf(event.SessionID, "%d", &chatID)
	if err != nil {
		c.logger.ErrorCtx(c.ctx, "invalid session ID for typing indicator", err,
			logger.Field{Key: "session_id", Value: event.SessionID})
		return
	}

	// Check if already typing for this session
	c.typingLock.RLock()
	_, exists := c.typingCancel[event.SessionID]
	c.typingLock.RUnlock()

	if exists {
		return
	}

	// Create cancel context for this session
	typingCtx, cancel := context.WithCancel(c.ctx)

	// Store cancel function
	c.typingLock.Lock()
	c.typingCancel[event.SessionID] = cancel
	c.typingLock.Unlock()

	// Start goroutine to send typing indicator periodically
	go func() {
		ticker := time.NewTicker(3 * time.Second)
		defer ticker.Stop()

		// Send first typing indicator immediately
		c.sendTypingIndicator(event)

		for {
			select {
			case <-typingCtx.Done():
				return
			case <-ticker.C:
				c.sendTypingIndicator(event)
			}
		}
	}()
}

// stopTypingIndicator stops the typing indicator for the specified chat
func (c *Connector) stopTypingIndicator(event bus.Event) {
	c.typingLock.Lock()
	defer c.typingLock.Unlock()

	if cancel, exists := c.typingCancel[event.SessionID]; exists {
		cancel()
		delete(c.typingCancel, event.SessionID)
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
