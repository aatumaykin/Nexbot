package cron

import (
	"github.com/aatumaykin/nexbot/internal/agent"
)

// CronSchedulerAdapter implements agent.CronManager by adapting cron.Scheduler
type CronSchedulerAdapter struct {
	scheduler *Scheduler
	storage   *Storage
}

// NewCronSchedulerAdapter creates a new adapter for cron scheduler
func NewCronSchedulerAdapter(scheduler *Scheduler, storage *Storage) *CronSchedulerAdapter {
	return &CronSchedulerAdapter{
		scheduler: scheduler,
		storage:   storage,
	}
}

// AddJob implements agent.CronManager
func (a *CronSchedulerAdapter) AddJob(job agent.Job) (string, error) {
	// Convert domain model to internal model
	cronJob := Job{
		ID:         job.ID,
		Type:       JobType(job.Type),
		Schedule:   job.Schedule,
		ExecuteAt:  job.ExecuteAt,
		UserID:     job.UserID,
		Tool:       job.Tool,
		Payload:    job.Payload,
		SessionID:  job.SessionID,
		Metadata:   job.Metadata,
		Executed:   job.Executed,
		ExecutedAt: job.ExecutedAt,
	}

	// Normalize job data
	if cronJob.Type == JobTypeOneshot {
		cronJob.Schedule = ""
	}

	return a.scheduler.AddJob(cronJob)
}

// RemoveJob implements agent.CronManager
func (a *CronSchedulerAdapter) RemoveJob(jobID string) error {
	return a.scheduler.RemoveJob(jobID)
}

// RemoveFromStorage implements agent.CronManager
func (a *CronSchedulerAdapter) RemoveFromStorage(jobID string) error {
	return a.storage.Remove(jobID)
}

// ListJobs implements agent.CronManager
func (a *CronSchedulerAdapter) ListJobs() []agent.Job {
	// Load jobs from storage
	storageJobs, err := a.storage.Load()
	if err != nil {
		return []agent.Job{}
	}

	// Convert to domain model
	jobs := make([]agent.Job, len(storageJobs))
	for i, sj := range storageJobs {
		// Normalize loaded jobs
		if sj.Type == string(JobTypeOneshot) {
			sj.Schedule = ""
		}

		jobs[i] = agent.Job{
			ID:         sj.ID,
			Type:       sj.Type,
			Schedule:   sj.Schedule,
			ExecuteAt:  sj.ExecuteAt,
			UserID:     sj.UserID,
			Tool:       sj.Tool,
			Payload:    sj.Payload,
			SessionID:  sj.SessionID,
			Metadata:   sj.Metadata,
			Executed:   sj.Executed,
			ExecutedAt: sj.ExecutedAt,
		}
	}
	return jobs
}

// AppendJob implements agent.CronManager
func (a *CronSchedulerAdapter) AppendJob(job agent.Job) error {
	// Convert domain model to storage model
	storageJob := StorageJob{
		ID:         job.ID,
		Type:       job.Type,
		Schedule:   job.Schedule,
		ExecuteAt:  job.ExecuteAt,
		UserID:     job.UserID,
		Tool:       job.Tool,
		Payload:    job.Payload,
		SessionID:  job.SessionID,
		Metadata:   job.Metadata,
		Executed:   job.Executed,
		ExecutedAt: job.ExecutedAt,
	}

	// Normalize job data
	if storageJob.Type == string(JobTypeOneshot) {
		storageJob.Schedule = ""
	}

	return a.storage.Append(storageJob)
}
