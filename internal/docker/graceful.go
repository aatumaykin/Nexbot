package docker

import (
	"context"
	"sync"
	"time"
)

type ShutdownConfig struct {
	Timeout      time.Duration
	DrainTimeout time.Duration
	ForceAfter   time.Duration
}

func (p *ContainerPool) GracefulShutdown(ctx context.Context, cfg ShutdownConfig) error {
	p.draining.Store(true)

	p.log.Info("starting graceful shutdown", "drain_timeout", cfg.DrainTimeout)

	drainCtx, cancel := context.WithTimeout(ctx, cfg.DrainTimeout)
	defer cancel()

drainLoop:
	for {
		busyCount := 0
		p.mu.RLock()
		for _, c := range p.containers {
			if c.GetStatus() == StatusBusy {
				busyCount++
			}
		}
		p.mu.RUnlock()

		if busyCount == 0 {
			break drainLoop
		}

		select {
		case <-drainCtx.Done():
			p.log.Warn("drain timeout, forcing shutdown", "busy_containers", busyCount)
			break drainLoop
		case <-time.After(100 * time.Millisecond):
		}
	}

	p.cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(ctx, cfg.Timeout)
	defer shutdownCancel()

	var wg sync.WaitGroup
	var containerIDs []string

	p.mu.RLock()
	for id := range p.containers {
		containerIDs = append(containerIDs, id)
	}
	p.mu.RUnlock()

	for _, id := range containerIDs {
		wg.Add(1)
		go func(containerID string) {
			defer wg.Done()
			timeout := int(cfg.ForceAfter.Seconds())
			if err := p.client.StopContainer(shutdownCtx, containerID, &timeout); err != nil {
				p.log.Error("failed to stop container", "container_id", containerID, "error", err)
			}
			if err := p.client.RemoveContainer(shutdownCtx, containerID); err != nil {
				p.log.Error("failed to remove container", "container_id", containerID, "error", err)
			}
		}(id)
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		p.log.Info("graceful shutdown complete")
	case <-shutdownCtx.Done():
		p.log.Warn("shutdown timeout exceeded")
	}

	p.mu.Lock()
	p.containers = make(map[string]*Container)
	p.mu.Unlock()

	return p.client.Close()
}
