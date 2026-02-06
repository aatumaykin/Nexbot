package cron

// CronSchedulerAdapter реализует agent.CronManager
type CronSchedulerAdapter struct {
	scheduler *Scheduler
	storage   *Storage
}

// NewCronSchedulerAdapter создает новый адаптер для cron scheduler
func NewCronSchedulerAdapter(scheduler *Scheduler, storage *Storage) *CronSchedulerAdapter {
	return &CronSchedulerAdapter{
		scheduler: scheduler,
		storage:   storage,
	}
}

func (a *CronSchedulerAdapter) AddJob(job Job) (string, error) {
	return a.scheduler.AddJob(job)
}

func (a *CronSchedulerAdapter) RemoveJob(jobID string) error {
	return a.scheduler.RemoveJob(jobID)
}

func (a *CronSchedulerAdapter) RemoveFromStorage(jobID string) error {
	return a.storage.Remove(jobID)
}

func (a *CronSchedulerAdapter) ListJobs() []StorageJob {
	jobs, err := a.storage.Load()
	if err != nil {
		return []StorageJob{}
	}
	return jobs
}

func (a *CronSchedulerAdapter) AppendJob(job StorageJob) error {
	return a.storage.Append(job)
}
