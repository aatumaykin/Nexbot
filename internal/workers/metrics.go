package workers

import (
	"time"
)

// Metrics returns the current pool metrics.
func (p *WorkerPool) Metrics() PoolMetrics {
	p.wg.RLock()
	defer p.wg.RUnlock()
	return *p.metrics
}

// incrementSubmitted increments the submitted task counter.
func (p *WorkerPool) incrementSubmitted() {
	p.wg.Lock()
	defer p.wg.Unlock()
	p.metrics.TasksSubmitted++
}

// incrementCompleted increments the completed task counter.
func (p *WorkerPool) incrementCompleted() {
	p.wg.Lock()
	defer p.wg.Unlock()
	p.metrics.TasksCompleted++
}

// incrementFailed increments the failed task counter.
func (p *WorkerPool) incrementFailed() {
	p.wg.Lock()
	defer p.wg.Unlock()
	p.metrics.TasksFailed++
}

// recordDuration records task execution duration.
func (p *WorkerPool) recordDuration(d time.Duration) {
	p.wg.Lock()
	defer p.wg.Unlock()
	p.metrics.TotalDuration += d
}
