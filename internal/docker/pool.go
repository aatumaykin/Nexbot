package docker

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/aatumaykin/nexbot/internal/subagent/sanitizer"
)

type Logger interface {
	Info(msg string, args ...interface{})
	Warn(msg string, args ...interface{})
	Error(msg string, args ...interface{})
}

var bufferPool = sync.Pool{
	New: func() interface{} {
		buf := make([]byte, 64*1024)
		return &buf
	},
}

type ContainerPool struct {
	cfg            PoolConfig
	client         DockerClientInterface
	rateLimiter    *RateLimiter
	circuitBreaker *CircuitBreaker
	containers     map[string]*Container
	mu             sync.RWMutex
	log            Logger
	metrics        PoolMetrics
	draining       atomic.Bool
	ctx            context.Context
	cancel         context.CancelFunc
	prometheus     *PrometheusMetrics
	validator      *sanitizer.Validator
}

func NewContainerPool(cfg PoolConfig, log Logger) (*ContainerPool, error) {
	client, err := NewDockerClient()
	if err != nil {
		return nil, err
	}

	return NewContainerPoolWithClient(cfg, log, client)
}

func NewContainerPoolWithClient(cfg PoolConfig, log Logger, client DockerClientInterface) (*ContainerPool, error) {
	ctx, cancel := context.WithCancel(context.Background())

	pool := &ContainerPool{
		cfg:            cfg,
		client:         client,
		rateLimiter:    NewRateLimiter(cfg.MaxTasksPerMinute),
		circuitBreaker: NewCircuitBreaker(cfg.CircuitBreakerThreshold, cfg.CircuitBreakerTimeout, nil),
		containers:     make(map[string]*Container),
		log:            log,
		ctx:            ctx,
		cancel:         cancel,
		validator:      sanitizer.NewValidator(sanitizer.SanitizerConfig{}),
	}

	pool.circuitBreaker = NewCircuitBreaker(cfg.CircuitBreakerThreshold, cfg.CircuitBreakerTimeout, &pool.metrics)

	return pool, nil
}

func (p *ContainerPool) Start(ctx context.Context) error {
	if err := p.client.PullImage(ctx, p.cfg); err != nil {
		return err
	}

	containerCount := p.cfg.ContainerCount
	if containerCount == 0 {
		containerCount = 1
	}

	var createdContainers []string

	for i := 0; i < containerCount; i++ {
		id, err := p.createAndStartContainer(ctx)
		if err != nil {
			for _, createdID := range createdContainers {
				p.client.StopContainer(ctx, createdID, intPtr(5))
				p.client.RemoveContainer(ctx, createdID)
			}
			return fmt.Errorf("failed to create container %d: %w", i, err)
		}
		createdContainers = append(createdContainers, id)
	}

	go p.startCleanupLoop()

	p.log.Info("docker pool started", "containers", containerCount)
	return nil
}

func (p *ContainerPool) createAndStartContainer(ctx context.Context) (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	id, err := p.client.CreateContainer(ctx, p.cfg)
	if err != nil {
		return "", err
	}

	if err := p.client.StartContainer(ctx, id); err != nil {
		p.client.RemoveContainer(ctx, id)
		return "", err
	}

	hijack, err := p.client.AttachContainer(ctx, id)
	if err != nil {
		p.client.StopContainer(ctx, id, intPtr(5))
		p.client.RemoveContainer(ctx, id)
		return "", err
	}

	containerCtx, containerCancel := context.WithCancel(p.ctx)

	maxPending := p.cfg.MaxPendingPerContainer
	if maxPending == 0 {
		maxPending = 100
	}

	inspectTTL := p.cfg.InspectTTL
	if inspectTTL == 0 {
		inspectTTL = 5 * time.Second
	}

	container := &Container{
		ID:         id,
		Status:     StatusIdle,
		StdinPipe:  hijack.Conn,
		StdoutPipe: hijack.Reader,
		hijackConn: hijack.Conn,
		LastUsed:   time.Now(),
		ctx:        containerCtx,
		cancelFunc: containerCancel,
		pending:    make(map[string]*pendingEntry),
		maxPending: maxPending,
		inspectTTL: inspectTTL,
	}

	p.containers[id] = container
	go p.readResponses(container)

	return id, nil
}

func (p *ContainerPool) readResponses(c *Container) {
	scanner := bufio.NewScanner(c.StdoutPipe)
	scanner.Buffer(make([]byte, 64*1024), 1*1024*1024)

	scanChan := make(chan struct {
		ok   bool
		data []byte
	}, 1)

	go func() {
		defer close(scanChan)
		for scanner.Scan() {
			data := make([]byte, len(scanner.Bytes()))
			copy(data, scanner.Bytes())
			select {
			case scanChan <- struct {
				ok   bool
				data []byte
			}{ok: true, data: data}:
			case <-c.ctx.Done():
				return
			}
		}
		select {
		case scanChan <- struct {
			ok   bool
			data []byte
		}{ok: false, data: nil}:
		case <-c.ctx.Done():
		}
	}()

	for {
		select {
		case result, ok := <-scanChan:
			if !ok || !result.ok {
				return
			}
			var resp SubagentResponse
			if err := json.Unmarshal(result.data, &resp); err != nil {
				continue
			}
			c.pendingMu.RLock()
			pe, ok := c.pending[resp.ID]
			if ok {
				pe.mu.Lock()
				c.pendingMu.RUnlock()
				if !pe.done {
					select {
					case pe.ch <- resp:
					default:
					}
				}
				pe.mu.Unlock()
			} else {
				c.pendingMu.RUnlock()
			}
		case <-time.After(30 * time.Second):
			p.log.Warn("scanner timeout", "container_id", c.ID)
			return
		case <-c.ctx.Done():
			return
		}
	}
}

func (p *ContainerPool) Stop(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.cancel()

	var lastErr error
	for id, c := range p.containers {
		c.Close()

		timeout := 30
		if err := p.client.StopContainer(ctx, id, &timeout); err != nil {
			lastErr = err
		}
		p.client.RemoveContainer(ctx, id)
	}

	p.containers = make(map[string]*Container)
	p.client.Close()

	return lastErr
}

func (p *ContainerPool) IsHealthy() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.circuitBreaker.State() == CircuitOpen {
		return false
	}

	for _, c := range p.containers {
		if c.Status == StatusIdle {
			return true
		}
	}

	return len(p.containers) == 0
}

func (p *ContainerPool) startCleanupLoop() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			p.cleanupStalePending()
		case <-p.ctx.Done():
			return
		}
	}
}

func (p *ContainerPool) cleanupStalePending() {
	now := time.Now()
	staleThreshold := 5 * time.Minute

	p.mu.RLock()
	defer p.mu.RUnlock()

	for _, c := range p.containers {
		c.pendingMu.Lock()
		for key, entry := range c.pending {
			if now.Sub(entry.created) > staleThreshold {
				entry.mu.Lock()
				if !entry.done {
					entry.done = true
					c.pendingCount.Add(-1)
				}
				entry.mu.Unlock()
				delete(c.pending, key)
			}
		}
		c.pendingMu.Unlock()
	}
}

func intPtr(i int) *int {
	return &i
}

func generateCorrelationID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}
