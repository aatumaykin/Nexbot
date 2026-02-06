package telegram

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/aatumaykin/nexbot/internal/bus"
	"github.com/aatumaykin/nexbot/internal/logger"
	"github.com/mymmrac/telego"
)

// TypingManager handles typing indicator logic for Telegram connector.
type TypingManager struct {
	bot          *telego.Bot
	logger       *logger.Logger
	ctx          context.Context
	typingLock   sync.RWMutex
	typingCancel map[string]context.CancelFunc
}

// NewTypingManager creates a new typing manager.
func NewTypingManager(bot *telego.Bot, logger *logger.Logger) *TypingManager {
	return &TypingManager{
		bot:          bot,
		logger:       logger,
		typingCancel: make(map[string]context.CancelFunc),
	}
}

// SetContext sets the context for the typing manager.
func (tm *TypingManager) SetContext(ctx context.Context) {
	tm.ctx = ctx
}

// Start starts a periodic typing indicator for the specified chat.
func (tm *TypingManager) Start(event bus.Event) {
	// Extract chat ID from session ID
	var chatID int64
	_, err := fmt.Sscanf(event.SessionID, "%d", &chatID)
	if err != nil {
		tm.logger.ErrorCtx(tm.ctx, "invalid session ID for typing indicator", err,
			logger.Field{Key: "session_id", Value: event.SessionID})
		return
	}

	// Check if already typing for this session
	tm.typingLock.RLock()
	_, exists := tm.typingCancel[event.SessionID]
	tm.typingLock.RUnlock()

	if exists {
		return
	}

	// Use background context if not set
	ctx := tm.ctx
	if ctx == nil {
		ctx = context.Background()
	}

	// Create cancel context for this session
	typingCtx, cancel := context.WithCancel(ctx)

	// Store cancel function
	tm.typingLock.Lock()
	tm.typingCancel[event.SessionID] = cancel
	tm.typingLock.Unlock()

	// Start goroutine to send typing indicator periodically
	go func() {
		ticker := time.NewTicker(3 * time.Second)
		defer ticker.Stop()

		// Send first typing indicator immediately
		tm.Send(event)

		for {
			select {
			case <-typingCtx.Done():
				return
			case <-ticker.C:
				tm.Send(event)
			}
		}
	}()
}

// Stop stops the typing indicator for the specified chat.
func (tm *TypingManager) Stop(event bus.Event) {
	tm.typingLock.Lock()
	defer tm.typingLock.Unlock()

	if cancel, exists := tm.typingCancel[event.SessionID]; exists {
		cancel()
		delete(tm.typingCancel, event.SessionID)
	}
}

// StopAll stops all typing indicators.
func (tm *TypingManager) StopAll() {
	tm.typingLock.Lock()
	defer tm.typingLock.Unlock()

	for sessionID, cancel := range tm.typingCancel {
		cancel()
		delete(tm.typingCancel, sessionID)
	}
}

// Send sends a typing indicator to the specified chat.
func (tm *TypingManager) Send(event bus.Event) {
	// Extract chat ID from session ID
	var chatID int64
	_, err := fmt.Sscanf(event.SessionID, "%d", &chatID)
	if err != nil {
		tm.logger.ErrorCtx(tm.ctx, "invalid session ID for typing indicator", err,
			logger.Field{Key: "session_id", Value: event.SessionID})
		return
	}

	// Send typing indicator
	if tm.bot == nil {
		tm.logger.WarnCtx(tm.ctx, "bot is nil, skipping typing indicator")
		return
	}

	params := &telego.SendChatActionParams{
		ChatID: telego.ChatID{ID: chatID},
		Action: telego.ChatActionTyping,
	}

	ctx := tm.ctx
	if ctx == nil {
		ctx = context.Background()
	}

	err = tm.bot.SendChatAction(ctx, params)
	if err != nil {
		tm.logger.ErrorCtx(ctx, "failed to send typing indicator", err,
			logger.Field{Key: "chat_id", Value: chatID})
		return
	}

	tm.logger.DebugCtx(ctx, "typing indicator sent",
		logger.Field{Key: "chat_id", Value: chatID},
		logger.Field{Key: "user_id", Value: event.UserID})
}
