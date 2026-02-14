package docker

import (
	"context"
	"fmt"
	"time"
)

const DefaultHealthCheckInterval = 30 * time.Second

type HealthStatus struct {
	ContainerID string
	Healthy     bool
	LastCheck   time.Time
	Error       string
	OOMKilled   bool
}

func (p *ContainerPool) HealthCheck(ctx context.Context) []HealthStatus {
	p.mu.RLock()
	defer p.mu.RUnlock()

	now := time.Now()
	interval := p.cfg.HealthCheckInterval
	if interval == 0 {
		interval = DefaultHealthCheckInterval
	}

	var results []HealthStatus

	for id, c := range p.containers {
		if now.Sub(c.LastHealthCheck) < interval {
			continue
		}

		if !c.healthCheckInProgress.CompareAndSwap(false, true) {
			continue
		}

		status := HealthStatus{
			ContainerID: id,
			LastCheck:   now,
		}

		inspect, err := p.client.InspectContainer(ctx, id)
		if err != nil {
			status.Healthy = false
			status.Error = err.Error()
			c.healthCheckInProgress.Store(false)
			results = append(results, status)
			continue
		}

		if !inspect.State.Running {
			status.Healthy = false
			status.Error = "container not running"
			c.healthCheckInProgress.Store(false)
			results = append(results, status)
			continue
		}

		if inspect.State.OOMKilled {
			status.Healthy = false
			status.Error = "OOM killed"
			status.OOMKilled = true
			p.metrics.OOMKills.Add(1)
			c.healthCheckInProgress.Store(false)
			results = append(results, status)
			continue
		}

		c.LastHealthCheck = now
		c.healthCheckInProgress.Store(false)
		status.Healthy = true
		results = append(results, status)
	}

	return results
}

func (p *ContainerPool) RecreateUnhealthy(ctx context.Context) error {
	statuses := p.HealthCheck(ctx)

	for _, status := range statuses {
		if !status.Healthy {
			p.log.Warn("recreating unhealthy container",
				"container_id", status.ContainerID,
				"error", status.Error)

			p.mu.Lock()
			if c, ok := p.containers[status.ContainerID]; ok {
				c.Close()
				delete(p.containers, status.ContainerID)
			}
			p.mu.Unlock()

			if err := p.client.StopContainer(ctx, status.ContainerID, intPtr(5)); err != nil {
				p.log.Error("failed to stop unhealthy container", "container_id", status.ContainerID, "error", err)
			}
			if err := p.client.RemoveContainer(ctx, status.ContainerID); err != nil {
				p.log.Error("failed to remove unhealthy container", "container_id", status.ContainerID, "error", err)
			}

			if _, err := p.createAndStartContainer(ctx); err != nil {
				return fmt.Errorf("failed to recreate container: %w", err)
			}

			p.metrics.Recreations.Add(1)
		}
	}

	return nil
}
