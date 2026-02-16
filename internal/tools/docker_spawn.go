package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/aatumaykin/nexbot/internal/docker"
	"github.com/google/uuid"
)

// DockerSpawnArgs represents the arguments for the spawn tool with Docker support.
type DockerSpawnArgs struct {
	Task            string   `json:"task"`
	TimeoutSeconds  int      `json:"timeout_seconds"`
	RequiredSecrets []string `json:"required_secrets"`
}

// DockerSpawnTool implements the Tool interface for spawning subagents in Docker containers.
type DockerSpawnTool struct {
	dockerPool    *docker.ContainerPool
	secretsFilter *docker.SecretsFilter
	llmAPIKey     string
}

// NewDockerSpawnTool creates a new DockerSpawnTool instance.
func NewDockerSpawnTool(dockerPool *docker.ContainerPool, secretsFilter *docker.SecretsFilter, llmAPIKeyEnv string) *DockerSpawnTool {
	apiKey := os.Getenv(llmAPIKeyEnv)
	if apiKey == "" {
		apiKey = os.Getenv("ZAI_API_KEY")
	}
	return &DockerSpawnTool{
		dockerPool:    dockerPool,
		secretsFilter: secretsFilter,
		llmAPIKey:     apiKey,
	}
}

// Name returns the tool name.
func (t *DockerSpawnTool) Name() string {
	return "spawn"
}

// Description returns a description of what the tool does.
func (t *DockerSpawnTool) Description() string {
	return "Delegate task to isolated subagent for external data fetching (web, APIs)"
}

// Parameters returns the JSON Schema for the tool's parameters.
func (t *DockerSpawnTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"task": map[string]any{
				"type":        "string",
				"description": "Task description for subagent",
			},
			"timeout_seconds": map[string]any{
				"type":        "integer",
				"description": "Task timeout in seconds (default: 120)",
				"default":     120,
			},
			"required_secrets": map[string]any{
				"type":        "array",
				"items":       map[string]string{"type": "string"},
				"description": "List of secret names required for this task",
			},
		},
		"required": []string{"task"},
	}
}

// Execute runs the tool with the provided arguments.
func (t *DockerSpawnTool) Execute(args string) (string, error) {
	return t.ExecuteWithContext(context.Background(), args)
}

// ExecuteWithContext runs the tool with the provided arguments and execution context.
func (t *DockerSpawnTool) ExecuteWithContext(ctx context.Context, args string) (string, error) {
	var spawnArgs DockerSpawnArgs
	if err := json.Unmarshal([]byte(args), &spawnArgs); err != nil {
		return "", fmt.Errorf("failed to parse arguments: %w", err)
	}

	if t.dockerPool == nil {
		if t.isLocalExecutionSafe(spawnArgs) {
			return t.executeLocally(ctx, spawnArgs)
		}
		return "", &DockerUnavailableError{
			Task: spawnArgs.Task,
			Hint: "Docker pool is not initialized",
		}
	}

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
		timeout = 120
	}

	req := docker.SubagentRequest{
		Version:       docker.ProtocolVersion,
		Type:          "execute",
		ID:            uuid.New().String(),
		CorrelationID: generateDockerCorrelationID(),
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

func generateDockerCorrelationID() string {
	return uuid.New().String()[:16]
}

// isLocalExecutionSafe checks if the task can be safely executed without Docker isolation.
func (t *DockerSpawnTool) isLocalExecutionSafe(args DockerSpawnArgs) bool {
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
func (t *DockerSpawnTool) executeLocally(ctx context.Context, args DockerSpawnArgs) (string, error) {
	return "", &DockerUnavailableError{
		Task: args.Task,
		Hint: "Docker unavailable and task cannot be executed safely without isolation",
	}
}

// Ensure DockerSpawnTool implements Tool interface
var _ Tool = (*DockerSpawnTool)(nil)

// Ensure DockerSpawnTool implements ContextualTool interface
var _ ContextualTool = (*DockerSpawnTool)(nil)
