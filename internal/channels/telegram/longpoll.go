package telegram

import (
	"context"

	"github.com/aatumaykin/nexbot/internal/logger"
	"github.com/mymmrac/telego"
)

// LongPollManager handles long polling for Telegram updates.
type LongPollManager struct {
	connector *Connector
	bot       BotInterface
	logger    *logger.Logger
	ctx       context.Context
}

// NewLongPollManager creates a new long poll manager.
func NewLongPollManager(connector *Connector, bot BotInterface, logger *logger.Logger) *LongPollManager {
	return &LongPollManager{
		connector: connector,
		bot:       bot,
		logger:    logger,
	}
}

// SetContext sets the context for the long poll manager.
func (lpm *LongPollManager) SetContext(ctx context.Context) {
	lpm.ctx = ctx
}

// SetBot sets the bot for the long poll manager.
func (lpm *LongPollManager) SetBot(bot BotInterface) {
	lpm.bot = bot
}

// Start starts long polling for Telegram updates.
func (lpm *LongPollManager) Start() {
	lpm.logger.Info("starting long polling for telegram updates")

	updates, err := lpm.bot.UpdatesViaLongPolling(lpm.ctx, &telego.GetUpdatesParams{
		Timeout: 30,
	})
	if err != nil {
		lpm.logger.ErrorCtx(lpm.ctx, "failed to start long polling", err)
		return
	}

	for {
		select {
		case <-lpm.ctx.Done():
			lpm.logger.Info("long polling stopped")
			return
		case update, ok := <-updates:
			if !ok {
				lpm.logger.Info("updates channel closed")
				return
			}

			if err := lpm.connector.updateHandler.Handle(update); err != nil {
				lpm.logger.ErrorCtx(lpm.ctx, "failed to handle update", err)
			}
		}
	}
}
