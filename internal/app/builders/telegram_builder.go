package builders

import (
	"context"
	"fmt"

	"github.com/aatumaykin/nexbot/internal/bus"
	"github.com/aatumaykin/nexbot/internal/channels/telegram"
	"github.com/aatumaykin/nexbot/internal/config"
	"github.com/aatumaykin/nexbot/internal/logger"
	"github.com/aatumaykin/nexbot/internal/secrets"
)

type TelegramBuilder struct {
	config     *config.Config
	logger     *logger.Logger
	messageBus *bus.MessageBus
}

func NewTelegramBuilder(cfg *config.Config, log *logger.Logger, mb *bus.MessageBus) *TelegramBuilder {
	return &TelegramBuilder{
		config:     cfg,
		logger:     log,
		messageBus: mb,
	}
}

func (b *TelegramBuilder) Build(ctx context.Context, secretsStore *secrets.Store) (*telegram.Connector, error) {
	if !b.config.Channels.Telegram.Enabled {
		return nil, nil
	}

	tg := telegram.New(
		b.config.Channels.Telegram,
		b.logger,
		b.messageBus,
	)
	if err := tg.Start(ctx); err != nil {
		return nil, fmt.Errorf("failed to start telegram connector: %w", err)
	}

	// Set secrets store on telegram command handler
	if cmdHandler := tg.GetCommandHandler(); cmdHandler != nil && secretsStore != nil {
		cmdHandler.SetSecretsStore(secretsStore)
		b.logger.Info("Secrets store configured for telegram commands")
	}

	return tg, nil
}
