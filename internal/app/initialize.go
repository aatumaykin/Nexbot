package app

import (
	"context"
	"fmt"
	"os"

	"github.com/aatumaykin/nexbot/internal/agent/loop"
	"github.com/aatumaykin/nexbot/internal/bus"
	"github.com/aatumaykin/nexbot/internal/channels/telegram"
	"github.com/aatumaykin/nexbot/internal/commands"
	"github.com/aatumaykin/nexbot/internal/cron"
	"github.com/aatumaykin/nexbot/internal/heartbeat"
	"github.com/aatumaykin/nexbot/internal/ipc"
	"github.com/aatumaykin/nexbot/internal/llm"
	"github.com/aatumaykin/nexbot/internal/logger"
	"github.com/aatumaykin/nexbot/internal/tools"
	"github.com/aatumaykin/nexbot/internal/workers"
	"github.com/aatumaykin/nexbot/internal/workspace"
)

// workerPoolAdapter adapts workers.WorkerPool to cron.WorkerPool interface.
// It converts cron.Task to workers.Task before submitting.
type workerPoolAdapter struct {
	pool *workers.WorkerPool
}

// newWorkerPoolAdapter creates a new adapter for workers pool.
func newWorkerPoolAdapter(pool *workers.WorkerPool) cron.WorkerPool {
	return &workerPoolAdapter{pool: pool}
}

// Submit adapts the cron.Task to workers.Task and submits it to the pool.
func (a *workerPoolAdapter) Submit(task cron.Task) {
	// Convert cron.Task to workers.Task
	workersTask := workers.Task{
		ID:      task.ID,
		Type:    task.Type,
		Payload: task.Payload,
		Context: task.Context,
		Metrics: make(map[string]interface{}),
	}

	// Submit to workers pool
	a.pool.Submit(workersTask)
}

// Initialize initializes all application components.
// It sets up the message bus, LLM provider, workspace, agent loop,
// command handler, tools, telegram connector, and cron scheduler.
func (a *App) Initialize(ctx context.Context) error {
	// 1. Create application context
	a.ctx, a.cancel = context.WithCancel(ctx)

	// 2. Initialize message bus
	a.messageBus = bus.New(a.config.MessageBus.Capacity, a.logger)
	if err := a.messageBus.Start(a.ctx); err != nil {
		return fmt.Errorf("failed to start message bus: %w", err)
	}

	// 3. Initialize LLM provider
	var provider llm.Provider
	switch a.config.Agent.Provider {
	case "zai":
		zaiConfig := llm.ZAIConfig{
			APIKey:         a.config.LLM.ZAI.APIKey,
			TimeoutSeconds: a.config.LLM.ZAI.TimeoutSeconds,
		}
		provider = llm.NewZAIProvider(zaiConfig, a.logger)
	default:
		return fmt.Errorf("unsupported LLM provider: %s", a.config.Agent.Provider)
	}

	// 4. Initialize workspace
	ws := workspace.New(a.config.Workspace)
	if err := ws.EnsureDir(); err != nil {
		return fmt.Errorf("failed to create workspace directory: %w", err)
	}
	if err := ws.EnsureSubpath("sessions"); err != nil {
		return fmt.Errorf("failed to create sessions subdirectory: %w", err)
	}

	// 4.1. Create HEARTBEAT.md bootstrap if it doesn't exist
	heartbeatPath := ws.Subpath("HEARTBEAT.md")
	if _, err := os.Stat(heartbeatPath); os.IsNotExist(err) {
		a.logger.Info("Creating HEARTBEAT.md bootstrap",
			logger.Field{Key: "path", Value: heartbeatPath})

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
		a.logger.Info("HEARTBEAT.md created")
	}

	// 4.1. Initialize worker pool
	workerPool := workers.NewPool(a.config.Workers.PoolSize, a.config.Workers.QueueSize, a.logger)
	workerPool.Start()
	a.workerPool = workerPool

	// 4.2. Initialize cron storage
	cronStorage := cron.NewStorage(ws.Path(), a.logger)
	cronJobs, err := cronStorage.Load()
	if err != nil {
		return fmt.Errorf("failed to load cron jobs from storage: %w", err)
	}
	a.logger.Info("loaded cron jobs from storage",
		logger.Field{Key: "count", Value: len(cronJobs)})

	// 5. Initialize agent loop
	agentLoop, err := loop.NewLoop(loop.Config{
		Workspace:         ws.Path(),
		SessionDir:        ws.Subpath("sessions"),
		LLMProvider:       provider,
		Logger:            a.logger,
		Model:             a.config.Agent.Model,
		MaxTokens:         a.config.Agent.MaxTokens,
		Temperature:       a.config.Agent.Temperature,
		MaxToolIterations: a.config.Agent.MaxIterations,
	})
	if err != nil {
		return fmt.Errorf("failed to create agent loop: %w", err)
	}
	a.agentLoop = agentLoop

	// 6. Create command handler
	a.commandHandler = commands.NewHandler(
		a.agentLoop,
		a.messageBus,
		a.logger,
		a.Restart,
	)

	// 7. Register tools
	// Register SendMessageTool
	sendMessageTool := tools.NewSendMessageTool(a.messageBus, a.logger)
	if err := a.agentLoop.RegisterTool(sendMessageTool); err != nil {
		return fmt.Errorf("failed to register send message tool: %w", err)
	}
	a.logger.Info("Send message tool registered")

	// Register shell tool if enabled
	if a.config.Tools.Shell.Enabled {
		shellTool := tools.NewShellExecTool(a.config, a.logger)
		if err := a.agentLoop.RegisterTool(shellTool); err != nil {
			return fmt.Errorf("failed to register shell tool: %w", err)
		}
	}

	// Register file tools if enabled
	if a.config.Tools.File.Enabled {
		readFileTool := tools.NewReadFileTool(ws, a.config)
		if err := a.agentLoop.RegisterTool(readFileTool); err != nil {
			return fmt.Errorf("failed to register read file tool: %w", err)
		}

		writeFileTool := tools.NewWriteFileTool(ws, a.config)
		if err := a.agentLoop.RegisterTool(writeFileTool); err != nil {
			return fmt.Errorf("failed to register write file tool: %w", err)
		}

		listDirTool := tools.NewListDirTool(ws, a.config)
		if err := a.agentLoop.RegisterTool(listDirTool); err != nil {
			return fmt.Errorf("failed to register list dir tool: %w", err)
		}
	}

	// 8. Initialize telegram connector if enabled
	if a.config.Channels.Telegram.Enabled {
		a.telegram = telegram.New(
			a.config.Channels.Telegram,
			a.logger,
			a.messageBus,
		)
		if err := a.telegram.Start(a.ctx); err != nil {
			return fmt.Errorf("failed to start telegram connector: %w", err)
		}
	}

	// 9. Initialize cron scheduler if enabled
	if a.config.Cron.Enabled {
		// Create worker pool adapter
		workerPoolAdapter := newWorkerPoolAdapter(workerPool)

		// Create cron scheduler
		a.cronScheduler = cron.NewScheduler(a.logger, a.messageBus, workerPoolAdapter, cronStorage)

		// Start cron scheduler
		if err := a.cronScheduler.Start(a.ctx); err != nil {
			return fmt.Errorf("failed to start cron scheduler: %w", err)
		}
		a.logger.Info("cron scheduler started")

		// Load jobs from storage and add to scheduler
		for _, storageJob := range cronJobs {
			job := cron.Job{
				ID:         storageJob.ID,
				Type:       cron.JobType(storageJob.Type),
				Schedule:   storageJob.Schedule,
				ExecuteAt:  storageJob.ExecuteAt,
				Command:    storageJob.Command,
				UserID:     storageJob.UserID,
				Metadata:   storageJob.Metadata,
				Executed:   storageJob.Executed,
				ExecutedAt: storageJob.ExecutedAt,
			}

			// Skip oneshot jobs that are already executed
			if job.Type == cron.JobTypeOneshot && job.Executed {
				continue
			}

			// Add job to scheduler
			if _, err := a.cronScheduler.AddJob(job); err != nil {
				a.logger.Error("failed to add cron job to scheduler", err,
					logger.Field{Key: "job_id", Value: job.ID},
					logger.Field{Key: "schedule", Value: job.Schedule})
				continue
			}
		}

		// Register CronTool
		cronTool := tools.NewCronTool(a.cronScheduler, cronStorage, a.logger)
		if err := a.agentLoop.RegisterTool(cronTool); err != nil {
			return fmt.Errorf("failed to register cron tool: %w", err)
		}
	}

	// 10. Initialize heartbeat checker if enabled
	if a.config.Heartbeat.Enabled {
		a.heartbeatChecker = heartbeat.NewChecker(
			a.config.Heartbeat.CheckIntervalMinutes,
			a.agentLoop,
			a.logger,
		)
		if err := a.heartbeatChecker.Start(); err != nil {
			return fmt.Errorf("failed to start heartbeat checker: %w", err)
		}
	}

	// 11. Initialize IPC handler
	a.ipcHandler, err = ipc.NewHandler(a.logger, ws.Subpath("sessions"), a.messageBus)
	if err != nil {
		return fmt.Errorf("failed to create IPC handler: %w", err)
	}

	// Write PID file
	if err := ipc.WritePID(ws.Path(), os.Getpid()); err != nil {
		return fmt.Errorf("failed to write PID file: %w", err)
	}

	// Start IPC server
	socketPath := ipc.GetSocketPath(ws.Path())
	if err := a.ipcHandler.Start(ctx, socketPath); err != nil {
		return fmt.Errorf("failed to start IPC server: %w", err)
	}

	// 12. Mark as started
	a.mu.Lock()
	a.started = true
	a.mu.Unlock()

	return nil
}
