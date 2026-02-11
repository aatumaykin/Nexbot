package builders

import (
	"fmt"
	"os"

	"github.com/aatumaykin/nexbot/internal/bus"
	"github.com/aatumaykin/nexbot/internal/config"
	"github.com/aatumaykin/nexbot/internal/cron"
	"github.com/aatumaykin/nexbot/internal/logger"
	"github.com/aatumaykin/nexbot/internal/workers"
	"github.com/aatumaykin/nexbot/internal/workspace"
)

type WorkspaceBuilder struct {
	config     *config.Config
	logger     *logger.Logger
	messageBus *bus.MessageBus
}

func NewWorkspaceBuilder(cfg *config.Config, log *logger.Logger, mb *bus.MessageBus) *WorkspaceBuilder {
	return &WorkspaceBuilder{
		config:     cfg,
		logger:     log,
		messageBus: mb,
	}
}

func (b *WorkspaceBuilder) Build() (*workspace.Workspace, error) {
	ws := workspace.New(b.config.Workspace)
	if err := ws.EnsureDir(); err != nil {
		return nil, fmt.Errorf("failed to create workspace directory: %w", err)
	}
	if err := ws.EnsureSubpath("sessions"); err != nil {
		return nil, fmt.Errorf("failed to create sessions subdirectory: %w", err)
	}
	return ws, nil
}

func (b *WorkspaceBuilder) InitializeSecrets(secretsDir string) error {
	if err := os.MkdirAll(secretsDir, 0700); err != nil {
		return fmt.Errorf("failed to create secrets directory: %w", err)
	}
	b.logger.Info("Secrets directory initialized", logger.Field{Key: "path", Value: secretsDir})
	return nil
}

func (b *WorkspaceBuilder) BuildWorkerPool() *workers.WorkerPool {
	workerPool := workers.NewPool(b.config.Workers.PoolSize, b.config.Workers.QueueSize, b.logger, b.messageBus)
	workerPool.Start()
	return workerPool
}

func (b *WorkspaceBuilder) BuildCronStorage(ws *workspace.Workspace) (*cron.Storage, []cron.StorageJob, error) {
	cronStorage := cron.NewStorage(ws.Path(), b.logger)
	cronJobs, err := cronStorage.Load()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load cron jobs from storage: %w", err)
	}
	b.logger.Info("loaded cron jobs from storage", logger.Field{Key: "count", Value: len(cronJobs)})
	return cronStorage, cronJobs, nil
}
