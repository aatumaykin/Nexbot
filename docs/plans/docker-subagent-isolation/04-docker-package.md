# Этап 4: Docker пакет (MVP)

## Цель

Реализация пула Docker-контейнеров с pool-level Circuit Breaker, простым Rate Limiting и graceful shutdown.

## Файлы

### 4.1 `internal/docker/types.go`

```go
package docker

import (
    "context"
    "fmt"
    "io"
    "sync"
    "sync/atomic"
    "time"
    
    "github.com/docker/docker/api/types"
)

const ProtocolVersion = "1.0"

type ContainerStatus string

const (
    StatusIdle  ContainerStatus = "idle"
    StatusBusy  ContainerStatus = "busy"
    StatusError ContainerStatus = "error"
)

type pendingEntry struct {
    ch      chan SubagentResponse
    created time.Time
    done    bool
    mu      sync.Mutex
}

type Container struct {
    ID                    string
    Status                ContainerStatus
    StdinPipe             io.Writer
    StdoutPipe            io.Reader
    hijackConn            io.ReadWriteCloser
    LastUsed              time.Time
    LastHealthCheck       time.Time
    healthCheckInProgress atomic.Bool
    pending               map[string]*pendingEntry
    pendingMu             sync.RWMutex
    pendingCount          atomic.Int64
    maxPending            int64
    cancelFunc            context.CancelFunc
    ctx                   context.Context
    
    lastInspect       time.Time
    lastInspectResult *types.ContainerJSON
    inspectTTL        time.Duration
}

func (c *Container) Close() error {
    if c.cancelFunc != nil {
        c.cancelFunc()
    }
    if c.hijackConn != nil {
        return c.hijackConn.Close()
    }
    return nil
}

func (c *Container) tryIncrementPending() bool {
    for {
        current := c.pendingCount.Load()
        if current >= c.maxPending {
            return false
        }
        if c.pendingCount.CompareAndSwap(current, current+1) {
            return true
        }
    }
}

type PoolConfig struct {
    ContainerCount int
    ImageName      string
    TaskTimeout    time.Duration
    WorkspacePath  string
    SkillsPath     string
    
    MemoryLimit string
    CPULimit    float64
    PidsLimit   int64
    
    LLMAPIKeyEnv string
    
    ImageTag    string
    ImageDigest string
    PullPolicy  string
    
    MaxTasksPerMinute        int
    CircuitBreakerThreshold  int
    CircuitBreakerTimeout    time.Duration
    SecretsTTL               time.Duration
    HealthCheckInterval      time.Duration
    MaxPendingPerContainer   int64
    InspectTTL               time.Duration
    
    SecurityOpt    []string
    ReadonlyRootfs *bool
}

type SubagentRequest struct {
    Version       string            `json:"version"`
    ID            string            `json:"id"`
    CorrelationID string            `json:"correlation_id,omitempty"`
    Type          string            `json:"type"`
    Task          string            `json:"task"`
    Timeout       int               `json:"timeout"`
    Deadline      int64             `json:"deadline,omitempty"`
    Secrets       map[string]string `json:"secrets,omitempty"`
    LLMAPIKey     string            `json:"llm_api_key,omitempty"`
}

type SubagentResponse struct {
    ID            string `json:"id"`
    CorrelationID string `json:"correlation_id,omitempty"`
    Status        string `json:"status"`
    Result        string `json:"result"`
    Error         string `json:"error,omitempty"`
    Version       string `json:"version,omitempty"`
}

type ErrorCode string

const (
    ErrCodeDraining      ErrorCode = "DRAINING"
    ErrCodeQueueFull     ErrorCode = "QUEUE_FULL"
    ErrCodeContainerDead ErrorCode = "CONTAINER_DEAD"
    ErrCodeTimeout       ErrorCode = "TIMEOUT"
)

type SubagentError struct {
    Code       ErrorCode
    Message    string
    Retry      bool
    RetryAfter time.Duration
}

func (e *SubagentError) Error() string {
    return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

type DockerError struct {
    Op      string
    Err     error
    Message string
}

func (e *DockerError) Error() string {
    return fmt.Sprintf("docker %s: %s: %v", e.Op, e.Message, e.Err)
}

func (e *DockerError) Unwrap() error {
    return e.Err
}

type RateLimitError struct {
    RetryAfter time.Duration
}

func (e *RateLimitError) Error() string {
    return fmt.Sprintf("rate limit exceeded, retry after %v", e.RetryAfter)
}

type CircuitOpenError struct {
    RetryAfter time.Duration
}

func (e *CircuitOpenError) Error() string {
    return fmt.Sprintf("circuit breaker open, retry after %v", e.RetryAfter)
}

type PoolMetrics struct {
    TotalContainers atomic.Int64
    IdleContainers  atomic.Int64
    BusyContainers  atomic.Int64
    TasksCompleted  atomic.Int64
    TasksFailed     atomic.Int64
    TasksTimedOut   atomic.Int64
    OOMKills        atomic.Int64
    Recreations     atomic.Int64
    CircuitTrips    atomic.Int64
    QueueFullHits   atomic.Int64
}
```

### 4.2 `internal/docker/client.go`

```go
package docker

import (
    "context"
    "fmt"
    "io"
    "strconv"
    "strings"
    
    "github.com/docker/docker/api/types"
    "github.com/docker/docker/api/types/container"
    "github.com/docker/docker/api/types/mount"
    "github.com/docker/docker/client"
)

type DockerClientInterface interface {
    PullImage(ctx context.Context, cfg PoolConfig) error
    CreateContainer(ctx context.Context, cfg PoolConfig) (string, error)
    StartContainer(ctx context.Context, id string) error
    StopContainer(ctx context.Context, id string, timeout *int) error
    RemoveContainer(ctx context.Context, id string) error
    AttachContainer(ctx context.Context, id string) (types.HijackedResponse, error)
    InspectContainer(ctx context.Context, id string) (types.ContainerJSON, error)
    Close() error
}

type DockerClient struct {
    client *client.Client
}

func NewDockerClient() (*DockerClient, error) {
    cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
    if err != nil {
        return nil, &DockerError{Op: "connect", Err: err, Message: "failed to connect to Docker daemon"}
    }
    
    ctx := context.Background()
    if _, err := cli.Ping(ctx); err != nil {
        return nil, &DockerError{Op: "ping", Err: err, Message: "Docker daemon not available"}
    }
    
    return &DockerClient{client: cli}, nil
}

func (c *DockerClient) Close() error {
    return c.client.Close()
}

func (c *DockerClient) PullImage(ctx context.Context, cfg PoolConfig) error {
    if cfg.PullPolicy == "never" {
        return nil
    }
    
    imageRef := cfg.ImageName
    if cfg.ImageDigest != "" {
        imageRef = cfg.ImageName + "@" + cfg.ImageDigest
    } else if cfg.ImageTag != "" && !strings.Contains(cfg.ImageName, ":") {
        imageRef = cfg.ImageName + ":" + cfg.ImageTag
    }
    
    reader, err := c.client.ImagePull(ctx, imageRef, types.ImagePullOptions{})
    if err != nil {
        if cfg.PullPolicy == "if-not-present" {
            return nil
        }
        return &DockerError{Op: "pull", Err: err, Message: fmt.Sprintf("failed to pull image %s", imageRef)}
    }
    defer reader.Close()
    io.Copy(io.Discard, reader)
    
    return nil
}

func (c *DockerClient) CreateContainer(ctx context.Context, cfg PoolConfig) (string, error) {
    env := []string{}
    
    mounts := []mount.Mount{
        {
            Type:     mount.TypeBind,
            Source:   cfg.SkillsPath,
            Target:   "/workspace/skills",
            ReadOnly: true,
        },
    }
    
    memoryLimit := parseMemory(cfg.MemoryLimit)
    if memoryLimit == 0 {
        memoryLimit = 128 * 1024 * 1024
    }
    
    cpuLimit := cfg.CPULimit
    if cpuLimit == 0 {
        cpuLimit = 0.5
    }
    
    pidsLimit := cfg.PidsLimit
    if pidsLimit == 0 {
        pidsLimit = 50
    }
    
    securityOpt := cfg.SecurityOpt
    if len(securityOpt) == 0 {
        securityOpt = []string{"no-new-privileges"}
    }
    
    readonlyRootfs := cfg.ReadonlyRootfs != nil && *cfg.ReadonlyRootfs
    
    resp, err := c.client.ContainerCreate(ctx, &container.Config{
        Image: cfg.ImageName,
        Env:   env,
    }, &container.HostConfig{
        Resources: container.Resources{
            Memory:    memoryLimit,
            NanoCPUs:  int64(cpuLimit * 1e9),
            PidsLimit: &pidsLimit,
        },
        Mounts:         mounts,
        SecurityOpt:    securityOpt,
        ReadonlyRootfs: readonlyRootfs,
        Tmpfs:          map[string]string{"/tmp": "rw,size=50m"},
    }, nil, nil, "")
    if err != nil {
        return "", &DockerError{Op: "create", Err: err, Message: "failed to create container"}
    }
    
    return resp.ID, nil
}

func (c *DockerClient) StartContainer(ctx context.Context, id string) error {
    if err := c.client.ContainerStart(ctx, id, container.StartOptions{}); err != nil {
        return &DockerError{Op: "start", Err: err, Message: fmt.Sprintf("failed to start container %s", id)}
    }
    return nil
}

func (c *DockerClient) StopContainer(ctx context.Context, id string, timeout *int) error {
    if err := c.client.ContainerStop(ctx, id, container.StopOptions{Timeout: timeout}); err != nil {
        return &DockerError{Op: "stop", Err: err, Message: fmt.Sprintf("failed to stop container %s", id)}
    }
    return nil
}

func (c *DockerClient) RemoveContainer(ctx context.Context, id string) error {
    if err := c.client.ContainerRemove(ctx, id, container.RemoveOptions{Force: true}); err != nil {
        return &DockerError{Op: "remove", Err: err, Message: fmt.Sprintf("failed to remove container %s", id)}
    }
    return nil
}

func (c *DockerClient) AttachContainer(ctx context.Context, id string) (types.HijackedResponse, error) {
    resp, err := c.client.ContainerAttach(ctx, id, container.AttachOptions{
        Stream: true,
        Stdin:  true,
        Stdout: true,
        Stderr: true,
    })
    if err != nil {
        return types.HijackedResponse{}, &DockerError{Op: "attach", Err: err, Message: fmt.Sprintf("failed to attach to container %s", id)}
    }
    return resp, nil
}

func (c *DockerClient) InspectContainer(ctx context.Context, id string) (types.ContainerJSON, error) {
    return c.client.ContainerInspect(ctx, id)
}

func parseMemory(s string) int64 {
    if s == "" {
        return 0
    }
    
    s = strings.TrimSpace(strings.ToLower(s))
    
    multiplier := int64(1)
    if strings.HasSuffix(s, "g") {
        multiplier = 1024 * 1024 * 1024
        s = strings.TrimSuffix(s, "g")
    } else if strings.HasSuffix(s, "m") {
        multiplier = 1024 * 1024
        s = strings.TrimSuffix(s, "m")
    } else if strings.HasSuffix(s, "k") {
        multiplier = 1024
        s = strings.TrimSuffix(s, "k")
    }
    
    val, err := strconv.ParseInt(s, 10, 64)
    if err != nil {
        return 0
    }
    
    return val * multiplier
}
```

### 4.3 `internal/docker/circuit_breaker.go`

```go
package docker

import (
    "sync"
    "sync/atomic"
    "time"
)

type CircuitState int32

const (
    CircuitClosed CircuitState = iota
    CircuitOpen
    CircuitHalfOpen
)

type CircuitBreaker struct {
    state            atomic.Int32
    failures         atomic.Int32
    lastFail         atomic.Int64
    halfOpenAttempts atomic.Int32
    threshold        int32
    timeout          time.Duration
    metrics          *PoolMetrics
}

func NewCircuitBreaker(threshold int, timeout time.Duration, metrics *PoolMetrics) *CircuitBreaker {
    if threshold == 0 {
        threshold = 5
    }
    if timeout == 0 {
        timeout = 30 * time.Second
    }
    return &CircuitBreaker{
        threshold: int32(threshold),
        timeout:   timeout,
        metrics:   metrics,
    }
}

func (cb *CircuitBreaker) Allow() (bool, int64) {
    token := time.Now().UnixNano()
    
    for {
        state := CircuitState(cb.state.Load())
        
        switch state {
        case CircuitClosed:
            return true, token
            
        case CircuitOpen:
            lastFailNano := cb.lastFail.Load()
            lastFail := time.Unix(0, lastFailNano)
            if time.Since(lastFail) <= cb.timeout {
                return false, 0
            }
            if !cb.state.CompareAndSwap(int32(CircuitOpen), int32(CircuitHalfOpen)) {
                continue
            }
            cb.halfOpenAttempts.Store(0)
            return true, token
            
        case CircuitHalfOpen:
            if cb.halfOpenAttempts.CompareAndSwap(0, 1) {
                return true, token
            }
            return false, 0
        }
    }
}

func (cb *CircuitBreaker) RecordSuccess() {
    cb.failures.Store(0)
    cb.halfOpenAttempts.Store(0)
    cb.state.Store(int32(CircuitClosed))
}

func (cb *CircuitBreaker) RecordFailure() {
    cb.failures.Add(1)
    cb.lastFail.Store(time.Now().UnixNano())
    
    state := CircuitState(cb.state.Load())
    if state == CircuitHalfOpen {
        cb.state.Store(int32(CircuitOpen))
    } else if cb.failures.Load() >= cb.threshold {
        if cb.state.CompareAndSwap(int32(CircuitClosed), int32(CircuitOpen)) {
            if cb.metrics != nil {
                cb.metrics.CircuitTrips.Add(1)
            }
        }
    }
}

func (cb *CircuitBreaker) State() CircuitState {
    return CircuitState(cb.state.Load())
}

func (cb *CircuitBreaker) Reset() {
    cb.failures.Store(0)
    cb.halfOpenAttempts.Store(0)
    cb.state.Store(int32(CircuitClosed))
}
```

### 4.4 `internal/docker/rate_limiter.go`

```go
package docker

import (
    "sync"
    "time"
)

// RateLimiter — простой counter + window (MVP)
type RateLimiter struct {
    mu       sync.Mutex
    count    int
    limit    int
    window   time.Duration
    start    time.Time
}

func NewRateLimiter(maxPerMinute int) *RateLimiter {
    if maxPerMinute <= 0 {
        maxPerMinute = 60
    }
    return &RateLimiter{
        limit:  maxPerMinute,
        window: time.Minute,
        start:  time.Now(),
    }
}

func (r *RateLimiter) Allow() (bool, time.Duration) {
    r.mu.Lock()
    defer r.mu.Unlock()
    
    now := time.Now()
    
    if now.Sub(r.start) >= r.window {
        r.count = 0
        r.start = now
    }
    
    if r.count < r.limit {
        r.count++
        return true, 0
    }
    
    waitTime := r.window - now.Sub(r.start)
    return false, waitTime
}

func (r *RateLimiter) MaxPerMinute() int {
    return r.limit
}
```

### 4.5 `internal/docker/pool.go`

```go
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
        ID:           id,
        Status:       StatusIdle,
        StdinPipe:    hijack.Conn,
        StdoutPipe:   hijack.Reader,
        hijackConn:   hijack.Conn,
        LastUsed:     time.Now(),
        ctx:          containerCtx,
        cancelFunc:   containerCancel,
        pending:      make(map[string]*pendingEntry),
        maxPending:   maxPending,
        inspectTTL:   inspectTTL,
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
```

### 4.6 `internal/docker/execute.go`

```go
package docker

import (
    "context"
    "encoding/json"
    "fmt"
    "time"
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
            Code:       ErrCodeQueueFull,
            Message:    fmt.Sprintf("pending queue full"),
            Retry:      true,
            RetryAfter: 500 * time.Millisecond,
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
    
    for id, c := range p.containers {
        if c.Status != StatusIdle {
            continue
        }
        
        isRunning, err := c.IsRunning(p.ctx, p.client)
        if err != nil || !isRunning {
            c.Status = StatusError
            continue
        }
        
        c.Status = StatusBusy
        c.LastUsed = time.Now()
        return c, nil
    }
    
    return nil, fmt.Errorf("no idle containers available")
}

func (p *ContainerPool) Release(containerID string) {
    p.mu.Lock()
    defer p.mu.Unlock()
    
    if c, ok := p.containers[containerID]; ok {
        c.Status = StatusIdle
    }
}

func (p *ContainerPool) markContainerDead(containerID string) {
    if c, ok := p.containers[containerID]; ok {
        c.Status = StatusError
        p.log.Warn("container marked as dead", "container_id", containerID)
    }
}
```

### 4.7 `internal/docker/health.go`

```go
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
            
            p.client.StopContainer(ctx, status.ContainerID, intPtr(5))
            p.client.RemoveContainer(ctx, status.ContainerID)
            
            if _, err := p.createAndStartContainer(ctx); err != nil {
                return fmt.Errorf("failed to recreate container: %w", err)
            }
            
            p.metrics.Recreations.Add(1)
        }
    }
    
    return nil
}
```

### 4.8 `internal/docker/graceful.go`

```go
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
            if c.Status == StatusBusy {
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
            p.client.StopContainer(shutdownCtx, containerID, &timeout)
            p.client.RemoveContainer(shutdownCtx, containerID)
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
```

### 4.9 `internal/docker/metrics.go` (6 базовых)

```go
package docker

import (
    "time"
    
    "github.com/prometheus/client_golang/prometheus"
)

type PrometheusMetrics struct {
    registry        prometheus.Registerer
    containersTotal prometheus.Gauge
    containersActive prometheus.Gauge
    tasksTotal      *prometheus.CounterVec
    taskDuration    *prometheus.HistogramVec
    circuitState    prometheus.Gauge
    pendingRequests prometheus.Gauge
}

func InitPrometheusMetrics(namespace string, reg prometheus.Registerer) *PrometheusMetrics {
    if reg == nil {
        reg = prometheus.DefaultRegisterer
    }
    
    m := &PrometheusMetrics{
        registry: reg,
        containersTotal: prometheus.NewGauge(
            prometheus.GaugeOpts{
                Namespace: namespace,
                Name:      "subagent_containers_total",
                Help:      "Total number of containers",
            },
        ),
        containersActive: prometheus.NewGauge(
            prometheus.GaugeOpts{
                Namespace: namespace,
                Name:      "subagent_containers_active",
                Help:      "Number of active (busy) containers",
            },
        ),
        tasksTotal: prometheus.NewCounterVec(
            prometheus.CounterOpts{
                Namespace: namespace,
                Name:      "subagent_tasks_total",
                Help:      "Total number of subagent tasks",
            },
            []string{"status"},
        ),
        taskDuration: prometheus.NewHistogramVec(
            prometheus.HistogramOpts{
                Namespace: namespace,
                Name:      "subagent_task_duration_seconds",
                Help:      "Duration of subagent tasks",
                Buckets:   []float64{.1, .5, 1, 5, 10, 30, 60, 120, 300},
            },
            []string{"status"},
        ),
        circuitState: prometheus.NewGauge(
            prometheus.GaugeOpts{
                Namespace: namespace,
                Name:      "subagent_circuit_breaker_state",
                Help:      "Circuit breaker state: 0=closed, 1=open, 2=half-open",
            },
        ),
        pendingRequests: prometheus.NewGauge(
            prometheus.GaugeOpts{
                Namespace: namespace,
                Name:      "subagent_pending_requests",
                Help:      "Number of pending requests",
            },
        ),
    }
    
    reg.MustRegister(
        m.containersTotal,
        m.containersActive,
        m.tasksTotal,
        m.taskDuration,
        m.circuitState,
        m.pendingRequests,
    )
    
    return m
}

func (m *PrometheusMetrics) RecordTask(status string, duration time.Duration) {
    m.tasksTotal.WithLabelValues(status).Inc()
    m.taskDuration.WithLabelValues(status).Observe(duration.Seconds())
}

func (m *PrometheusMetrics) SetCircuitState(state CircuitState) {
    m.circuitState.Set(float64(state))
}

func (m *PrometheusMetrics) SetPendingCount(count int64) {
    m.pendingRequests.Set(float64(count))
}

func (m *PrometheusMetrics) SetContainerCounts(total, active int64) {
    m.containersTotal.Set(float64(total))
    m.containersActive.Set(float64(active))
}
```

## Ключевые решения (MVP)

1. **Pool-level Circuit Breaker** — один CB для всего пула (достаточно для 1-3 контейнеров)
2. **Counter + window rate limiter** — проще token bucket, достаточно для MVP
3. **6 базовых metrics** — containers, tasks, latency, circuit state, pending
4. **tmpfs mount для /tmp** — ReadonlyRootfs + tmpfs для временных файлов
5. **sync.Pool для буферов** — снижение GC pressure
6. **Write timeout 5s** — защита от зависания stdin
7. **Scanner timeout 30s** — защита от зависания stdout
