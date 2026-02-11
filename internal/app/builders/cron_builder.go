package builders

import (
	"context"
	"fmt"

	"github.com/aatumaykin/nexbot/internal/bus"
	"github.com/aatumaykin/nexbot/internal/config"
	"github.com/aatumaykin/nexbot/internal/cron"
	"github.com/aatumaykin/nexbot/internal/logger"
	"github.com/aatumaykin/nexbot/internal/tools"
	"github.com/aatumaykin/nexbot/internal/workers"
)

type CronBuilder struct {
	config      *config.Config
	logger      *logger.Logger
	workerPool  *workers.WorkerPool
	cronStorage *cron.Storage
}

func NewCronBuilder(cfg *config.Config, log *logger.Logger, wp *workers.WorkerPool, cs *cron.Storage) *CronBuilder {
	return &CronBuilder{
		config:      cfg,
		logger:      log,
		workerPool:  wp,
		cronStorage: cs,
	}
}

func (b *CronBuilder) BuildAndStart(ctx context.Context, messageBus *bus.MessageBus, cronJobs []cron.StorageJob) (*cron.Scheduler, error) {
	if !b.config.Cron.Enabled {
		return nil, nil
	}

	workerPoolAdapter := newWorkerPoolAdapter(b.workerPool)
	scheduler := cron.NewScheduler(b.logger, messageBus, workerPoolAdapter, b.cronStorage)

	if err := scheduler.Start(ctx); err != nil {
		return nil, fmt.Errorf("failed to start cron scheduler: %w", err)
	}
	b.logger.Info("cron scheduler started")

	for _, storageJob := range cronJobs {
		job := cron.Job{
			ID:         storageJob.ID,
			Type:       cron.JobType(storageJob.Type),
			Schedule:   storageJob.Schedule,
			ExecuteAt:  storageJob.ExecuteAt,
			UserID:     storageJob.UserID,
			Tool:       storageJob.Tool,
			Payload:    storageJob.Payload,
			SessionID:  storageJob.SessionID,
			Metadata:   storageJob.Metadata,
			Executed:   storageJob.Executed,
			ExecutedAt: storageJob.ExecutedAt,
		}

		if job.Type == cron.JobTypeOneshot && job.Executed {
			continue
		}

		if _, err := scheduler.AddJob(job); err != nil {
			b.logger.Error("failed to add cron job to scheduler", err,
				logger.Field{Key: "job_id", Value: job.ID},
				logger.Field{Key: "schedule", Value: job.Schedule})
			continue
		}
	}

	return scheduler, nil
}

func (b *CronBuilder) CreateCronTool(scheduler *cron.Scheduler) *tools.CronTool {
	if scheduler == nil {
		return nil
	}
	cronAdapter := cron.NewCronSchedulerAdapter(scheduler, b.cronStorage)
	return tools.NewCronTool(cronAdapter, b.logger)
}

type workerPoolAdapter struct {
	pool *workers.WorkerPool
}

func newWorkerPoolAdapter(pool *workers.WorkerPool) cron.WorkerPool {
	return &workerPoolAdapter{pool: pool}
}

func (a *workerPoolAdapter) Submit(task cron.Task) {
	workersTask := workers.Task{
		ID:      task.ID,
		Type:    task.Type,
		Payload: task.Payload,
		Context: task.Context,
		Metrics: make(map[string]any),
	}
	a.pool.Submit(workersTask)
}
