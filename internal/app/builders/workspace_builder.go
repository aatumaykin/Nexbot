package builders

import (
	"fmt"
	"os"

	"github.com/aatumaykin/nexbot/internal/bus"
	"github.com/aatumaykin/nexbot/internal/config"
	"github.com/aatumaykin/nexbot/internal/cron"
	"github.com/aatumaykin/nexbot/internal/heartbeat"
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

func (b *WorkspaceBuilder) InitializeHeartbeat(ws *workspace.Workspace) error {
	heartbeatPath := ws.Subpath("HEARTBEAT.md")
	if _, err := os.Stat(heartbeatPath); os.IsNotExist(err) {
		b.logger.Info("Creating HEARTBEAT.md bootstrap", logger.Field{Key: "path", Value: heartbeatPath})

		heartbeatContent := `# HEARTBEAT - Задачи и отправка

Этот файл читается каждые 10 минут.

## Как использовать

### Для LLM

1. Читай секцию "Задачи"
2. Проверяй время выполнения
3. Если пора выполнить:
   - Выполни задачу (используй доступные tools: read_file, write_file, send_message)
   - Если нужно отправить сообщение — используй send_message tool
   - Если нужно обновить HEARTBEAT.md — используй write_file tool
4. Если ничего — верни "HEARTBEAT_OK"

## Задачи

---

Добавляй задачи сюда.
`
		if err := os.WriteFile(heartbeatPath, []byte(heartbeatContent), 0644); err != nil {
			return fmt.Errorf("failed to create HEARTBEAT.md: %w", err)
		}
		b.logger.Info("HEARTBEAT.md created")
	}
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

func (b *WorkspaceBuilder) BuildHeartbeatChecker(agentLoop heartbeat.Agent) (*heartbeat.Checker, error) {
	checker := heartbeat.NewChecker(
		b.config.Heartbeat.CheckIntervalMinutes,
		agentLoop,
		b.logger,
	)
	if err := checker.Start(); err != nil {
		return nil, fmt.Errorf("failed to start heartbeat checker: %w", err)
	}
	return checker, nil
}
