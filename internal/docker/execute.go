package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/aatumaykin/nexbot/internal/subagent/sanitizer"
)

const MaxResponseSize = 1 * 1024 * 1024

func (p *ContainerPool) ExecuteTask(ctx context.Context, req SubagentRequest, secrets map[string]string, llmAPIKey string) (*SubagentResponse, error) {
	allowed, _ := p.circuitBreaker.Allow()
	if !allowed {
		return nil, &CircuitOpenError{RetryAfter: p.cfg.CircuitBreakerTimeout}
	}

	rateOk, retryAfter := p.rateLimiter.Allow()
	if !rateOk {
		return nil, &RateLimitError{RetryAfter: retryAfter}
	}

	if p.draining.Load() {
		return nil, &SubagentError{Code: ErrCodeDraining, Message: "pool is draining"}
	}

	p.log.Info("execute task",
		"task_id", req.ID,
		"task", sanitizer.RedactForLog(req.Task, secrets),
		"secret_count", len(secrets))

	container, err := p.acquire()
	if err != nil {
		return nil, err
	}
	defer p.Release(container.ID)

	if !container.tryIncrementPending() {
		p.metrics.QueueFullHits.Add(1)
		return nil, &SubagentError{
			Code:    ErrCodeQueueFull,
			Message: "pending queue full",
			Retry:   false,
		}
	}

	isRunning, err := container.IsRunning(ctx, p.client)
	if err != nil || !isRunning {
		container.pendingCount.Add(-1)
		p.markContainerDead(container.ID)
		p.circuitBreaker.RecordFailure()
		return nil, &SubagentError{
			Code:       ErrCodeContainerDead,
			Message:    fmt.Sprintf("container %s not running", container.ID),
			Retry:      true,
			RetryAfter: 1 * time.Second,
		}
	}

	req.Version = ProtocolVersion
	if req.CorrelationID == "" {
		req.CorrelationID = generateCorrelationID()
	}
	req.Secrets = secrets
	req.LLMAPIKey = llmAPIKey

	respChan := make(chan SubagentResponse, 1)
	entry := &pendingEntry{
		ch:      respChan,
		created: time.Now(),
		done:    false,
	}
	container.pendingMu.Lock()
	container.pending[req.ID] = entry
	container.pendingMu.Unlock()

	cleanup := func() {
		container.pendingMu.Lock()
		if pe, ok := container.pending[req.ID]; ok {
			pe.mu.Lock()
			pe.done = true
			pe.mu.Unlock()
			delete(container.pending, req.ID)
		}
		container.pendingMu.Unlock()
		container.pendingCount.Add(-1)
		select {
		case <-respChan:
		default:
		}
	}
	defer cleanup()

	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	writeCtx, writeCancel := context.WithTimeout(ctx, 5*time.Second)
	defer writeCancel()

	writeDone := make(chan error, 1)
	go func() {
		select {
		case <-writeCtx.Done():
			writeDone <- writeCtx.Err()
			return
		default:
		}

		if _, err := container.StdinPipe.Write(data); err != nil {
			writeDone <- err
			return
		}

		select {
		case <-writeCtx.Done():
			writeDone <- writeCtx.Err()
			return
		default:
		}

		_, err := container.StdinPipe.Write([]byte("\n"))
		writeDone <- err
	}()

	select {
	case writeErr := <-writeDone:
		if writeErr != nil {
			p.circuitBreaker.RecordFailure()
			return nil, fmt.Errorf("failed to write request: %w", writeErr)
		}
	case <-writeCtx.Done():
		p.circuitBreaker.RecordFailure()
		select {
		case <-writeDone:
		case <-time.After(100 * time.Millisecond):
		}
		return nil, fmt.Errorf("write timeout after 5s")
	}

	timeout := p.cfg.TaskTimeout
	if req.Timeout > 0 {
		timeout = time.Duration(req.Timeout) * time.Second
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	select {
	case resp := <-respChan:
		if resp.Status == "error" {
			p.metrics.TasksFailed.Add(1)
			p.circuitBreaker.RecordFailure()
			return nil, fmt.Errorf("subagent error: %s", resp.Error)
		}
		if len(resp.Result) > MaxResponseSize {
			resp.Result = resp.Result[:MaxResponseSize] + "\n[TRUNCATED]"
		}
		resp.Result = p.validator.SanitizeToolOutput(resp.Result)
		p.metrics.TasksCompleted.Add(1)
		p.circuitBreaker.RecordSuccess()
		return &resp, nil
	case <-ctx.Done():
		p.metrics.TasksTimedOut.Add(1)
		p.circuitBreaker.RecordFailure()
		return nil, &SubagentError{
			Code:       ErrCodeTimeout,
			Message:    fmt.Sprintf("task timeout after %v", timeout),
			Retry:      true,
			RetryAfter: 1 * time.Second,
		}
	}
}

func (c *Container) IsRunning(ctx context.Context, client DockerClientInterface) (bool, error) {
	if time.Since(c.lastInspect) < c.inspectTTL && c.lastInspectResult != nil {
		return c.lastInspectResult.State.Running, nil
	}

	inspect, err := client.InspectContainer(ctx, c.ID)
	if err != nil {
		return false, err
	}

	c.lastInspect = time.Now()
	c.lastInspectResult = &inspect
	return inspect.State.Running, nil
}

func (p *ContainerPool) acquire() (*Container, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, c := range p.containers {
		if c.GetStatus() != StatusIdle {
			continue
		}

		isRunning, err := c.IsRunning(p.ctx, p.client)
		if err != nil || !isRunning {
			c.SetStatus(StatusError)
			continue
		}

		c.SetStatus(StatusBusy)
		c.LastUsed = time.Now()
		return c, nil
	}

	p.mu.Unlock()
	container, err := p.CreateContainer(context.Background())
	if err != nil {
		return nil, err
	}

	p.mu.Lock()
	p.containers[container.ID] = container
	container.SetStatus(StatusBusy)
	container.LastUsed = time.Now()
	return container, nil
}

func (p *ContainerPool) Release(containerID string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if c, ok := p.containers[containerID]; ok {
		c.SetStatus(StatusIdle)

		timeout := 5
		if err := p.client.StopContainer(context.Background(), containerID, &timeout); err != nil {
			p.log.Warn("failed to stop container", "container_id", containerID, "error", err)
		}

		c.Close()
		delete(p.containers, containerID)
		p.log.Info("container stopped and removed", "container_id", containerID)
	}
}

func (p *ContainerPool) markContainerDead(containerID string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if c, ok := p.containers[containerID]; ok {
		c.SetStatus(StatusError)
		p.log.Warn("container marked as dead", "container_id", containerID)
	}
}
