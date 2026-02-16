package docker

import (
	"context"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"time"

	"github.com/moby/moby/api/types/container"
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
	lastInspectResult *container.InspectResponse
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
	TaskTimeout         time.Duration
	SkillsMountPath     string
	BinaryPath          string // Path to binary executable (e.g., os.Executable())
	SubagentPromptsPath string // Path to subagent prompts directory
	ConfigPath          string // Path to config.toml file (optional, defaults to ~/.config/nexbot/config.toml)

	MemoryLimit string
	CPULimit    float64
	PidsLimit   int64

	LLMAPIKeyEnv string

	PullPolicy string

	MaxTasksPerMinute       int
	CircuitBreakerThreshold int
	CircuitBreakerTimeout   time.Duration
	SecretsTTL              time.Duration
	HealthCheckInterval     time.Duration
	MaxPendingPerContainer  int64
	InspectTTL              time.Duration

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
