package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/aatumaykin/nexbot/internal/docker"
	"github.com/google/uuid"
)

// SpawnFunc is a function type for spawning subagents.
// This avoids circular import with the subagent package.
// Deprecated: Use NewSpawnToolWithDocker for Docker-based isolation.
type SpawnFunc func(ctx context.Context, parentSession string, task string) (string, error)

// SpawnArgs represents the arguments for the spawn tool.
type SpawnArgs struct {
	Task            string   `json:"task"`
	TimeoutSeconds  int      `json:"timeout_seconds"`
	RequiredSecrets []string `json:"required_secrets"`
}

// DockerUnavailableError is returned when Docker is not available
// and the task cannot be safely executed locally.
type DockerUnavailableError struct {
	Task string
	Hint string
}

func (e *DockerUnavailableError) Error() string {
	return fmt.Sprintf("cannot delegate task '%s': %s", e.Task, e.Hint)
}

// UserMessage returns a user-friendly message explaining the error.
func (e *DockerUnavailableError) UserMessage() string {
	return fmt.Sprintf(
		"⚠️ Не могу выполнить задачу с внешними данными.\n\n"+
			"Задача: %s\n\n"+
			"Причина: Docker-изоляция недоступна.\n\n"+
			"Решение:\n"+
			"1. Убедитесь, что Docker установлен и запущен: docker ps\n"+
			"2. Проверьте образ: docker images | grep alpine",
		e.Task,
	)
}

// SpawnTool implements the Tool and ContextualTool interfaces for spawning subagents.
// It supports both Docker-based isolation and legacy SpawnFunc-based execution.
type SpawnTool struct {
	// Docker-based execution
	dockerPool    *docker.ContainerPool
	secretsFilter *docker.SecretsFilter
	llmAPIKey     string

	// Legacy SpawnFunc-based execution
	spawnFunc SpawnFunc
}

// NewSpawnTool creates a new SpawnTool instance using legacy SpawnFunc.
// Deprecated: Use NewSpawnToolWithDocker for Docker-based isolation.
func NewSpawnTool(spawnFunc SpawnFunc) *SpawnTool {
	return &SpawnTool{spawnFunc: spawnFunc}
}

// NewSpawnToolWithDocker creates a new SpawnTool instance with Docker-based isolation.
func NewSpawnToolWithDocker(dockerPool *docker.ContainerPool, secretsFilter *docker.SecretsFilter, llmAPIKeyEnv string) *SpawnTool {
	apiKey := os.Getenv(llmAPIKeyEnv)
	if apiKey == "" {
		apiKey = os.Getenv("ZAI_API_KEY")
	}
	return &SpawnTool{
		dockerPool:    dockerPool,
		secretsFilter: secretsFilter,
		llmAPIKey:     apiKey,
	}
}

// Name returns the tool name.
func (t *SpawnTool) Name() string {
	return "spawn"
}

// Description returns a description of what the tool does.
func (t *SpawnTool) Description() string {
	return "Delegate task to isolated subagent for external data fetching (web, APIs)"
}

// Parameters returns the JSON Schema for the tool's parameters.
func (t *SpawnTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"task": map[string]interface{}{
				"type":        "string",
				"description": "Task description for subagent",
			},
			"timeout_seconds": map[string]interface{}{
				"type":        "number",
				"description": "Task timeout in seconds (default: 60)",
				"default":     60,
			},
			"required_secrets": map[string]interface{}{
				"type":        "array",
				"items":       map[string]string{"type": "string"},
				"description": "List of secret names required for this task",
			},
		},
		"required": []string{"task"},
	}
}

// Execute runs the tool with the provided arguments.
func (t *SpawnTool) Execute(args string) (string, error) {
	return t.ExecuteWithContext(context.Background(), args)
}

// ExecuteWithContext runs the tool with the provided arguments and execution context.
func (t *SpawnTool) ExecuteWithContext(ctx context.Context, args string) (string, error) {
	var spawnArgs SpawnArgs
	if err := json.Unmarshal([]byte(args), &spawnArgs); err != nil {
		return "", fmt.Errorf("failed to parse arguments: %w", err)
	}

	// Validate required arguments
	if spawnArgs.Task == "" {
		return "", fmt.Errorf("task is required")
	}

	// Validate timeout
	if spawnArgs.TimeoutSeconds < 0 {
		return "", fmt.Errorf("timeout_seconds must be positive")
	}

	// Use Docker-based execution if available
	if t.dockerPool != nil {
		return t.executeWithDocker(ctx, spawnArgs)
	}

	// Fallback to legacy SpawnFunc-based execution
	return t.executeWithSpawnFunc(ctx, spawnArgs)
}

func (t *SpawnTool) executeWithDocker(ctx context.Context, spawnArgs SpawnArgs) (string, error) {
	if !t.dockerPool.IsHealthy() {
		if t.isLocalExecutionSafe(spawnArgs) {
			return t.executeLocally(ctx, spawnArgs)
		}
		return "", &DockerUnavailableError{
			Task: spawnArgs.Task,
			Hint: "Docker pool is unhealthy or circuit breaker is open",
		}
	}

	var secrets map[string]string
	var err error
	if t.secretsFilter != nil {
		secrets, err = t.secretsFilter.FilterForTask(spawnArgs.RequiredSecrets)
		if err != nil {
			return "", fmt.Errorf("failed to filter secrets: %w", err)
		}
	}

	timeout := spawnArgs.TimeoutSeconds
	if timeout == 0 {
		timeout = 60
	}

	req := docker.SubagentRequest{
		Version:       docker.ProtocolVersion,
		Type:          "execute",
		ID:            uuid.New().String(),
		CorrelationID: generateSpawnCorrelationID(),
		Task:          spawnArgs.Task,
		Timeout:       timeout,
	}

	resp, err := t.dockerPool.ExecuteTask(ctx, req, secrets, t.llmAPIKey)
	if err != nil {
		switch e := err.(type) {
		case *docker.RateLimitError:
			return "", fmt.Errorf("too many requests, retry after %v", e.RetryAfter)
		case *docker.CircuitOpenError:
			return "", &DockerUnavailableError{
				Task: spawnArgs.Task,
				Hint: fmt.Sprintf("circuit breaker open, retry after %v", e.RetryAfter),
			}
		case *docker.SubagentError:
			if e.Code == docker.ErrCodeQueueFull {
				return "", fmt.Errorf("task queue full, retry after %v", e.RetryAfter)
			}
			return "", fmt.Errorf("subagent error [%s]: %s", e.Code, e.Message)
		}
		return "", fmt.Errorf("subagent execution failed: %w", err)
	}

	return resp.Result, nil
}

func (t *SpawnTool) executeWithSpawnFunc(ctx context.Context, spawnArgs SpawnArgs) (string, error) {
	if t.spawnFunc == nil {
		return "", fmt.Errorf("no spawn function configured")
	}

	timeoutSeconds := spawnArgs.TimeoutSeconds
	if timeoutSeconds == 0 {
		timeoutSeconds = 300
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSeconds)*time.Second)
	defer cancel()

	result, err := t.spawnFunc(timeoutCtx, "parent", spawnArgs.Task)
	if err != nil {
		return "", fmt.Errorf("failed to execute task via subagent: %w", err)
	}

	return result, nil
}

func generateSpawnCorrelationID() string {
	return uuid.New().String()[:16]
}

// isLocalExecutionSafe checks if the task can be safely executed without Docker isolation.
func (t *SpawnTool) isLocalExecutionSafe(args SpawnArgs) bool {
	task := args.Task

	// Try URL decoding to detect hidden URLs
	if decoded, err := url.QueryUnescape(task); err == nil {
		task = decoded
	}
	taskLower := strings.ToLower(task)

	// Check for URLs (including URL-encoded)
	if strings.Contains(taskLower, "http://") ||
		strings.Contains(taskLower, "https://") ||
		strings.Contains(taskLower, "http%3a%2f%2f") ||
		strings.Contains(taskLower, "https%3a%2f%2f") {
		return false
	}

	// Check for required secrets
	if len(args.RequiredSecrets) > 0 {
		return false
	}

	// Check for suspicious patterns
	suspiciousPatterns := []string{
		"ignore", "forget", "override", "system:",
		"assistant:", "exec(", "eval(",
	}
	for _, pattern := range suspiciousPatterns {
		if strings.Contains(taskLower, pattern) {
			return false
		}
	}

	return true
}

// executeLocally attempts to execute the task locally when Docker is unavailable.
func (t *SpawnTool) executeLocally(ctx context.Context, args SpawnArgs) (string, error) {
	return "", &DockerUnavailableError{
		Task: args.Task,
		Hint: "Docker unavailable and task cannot be executed safely without isolation",
	}
}

// Ensure SpawnTool implements Tool interface
var _ Tool = (*SpawnTool)(nil)

// Ensure SpawnTool implements ContextualTool interface
var _ ContextualTool = (*SpawnTool)(nil)
