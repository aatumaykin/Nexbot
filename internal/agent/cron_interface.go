package agent

import "github.com/aatumaykin/nexbot/internal/cron"

// CronManager интерфейс для управления cron jobs из tools
type CronManager interface {
	AddJob(job cron.Job) (string, error)
	RemoveJob(jobID string) error
	RemoveFromStorage(jobID string) error
	ListJobs() []cron.StorageJob
	AppendJob(job cron.StorageJob) error
}
