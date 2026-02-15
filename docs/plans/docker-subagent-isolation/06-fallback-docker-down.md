# Этап 6: Fallback при Docker down

## Цель

Обработка недоступности Docker с понятными сообщениями для пользователя.

## Файлы

### `internal/tools/spawn.go`

```go
package tools

import (
    "context"
    "encoding/json"
    "fmt"
    "net/url"  // v14: для декодирования URL encoding
    "os"
    "strings"

    "github.com/aatumaykin/nexbot/internal/docker"
    "github.com/google/uuid"
)

type SpawnArgs struct {
    Task            string   `json:"task"`
    TimeoutSeconds  int      `json:"timeout_seconds"`
    RequiredSecrets []string `json:"required_secrets"`
}

type DockerUnavailableError struct {
    Task string
    Hint string
}

func (e *DockerUnavailableError) Error() string {
    return fmt.Sprintf("cannot delegate task '%s': %s", e.Task, e.Hint)
}

func (e *DockerUnavailableError) UserMessage() string {
    return fmt.Sprintf(
        "⚠️ Не могу выполнить задачу с внешними данными.\n\n"+
        "Задача: %s\n\n"+
        "Причина: Docker-изоляция недоступна.\n\n"+
        "Решение:\n"+
        "1. Убедитесь, что Docker установлен и запущен: docker ps\n"+
        "2. Проверьте образ: docker images | grep nexbot/subagent",
        e.Task,
    )
}

type SpawnTool struct {
    dockerPool    *docker.ContainerPool
    secretsFilter *docker.SecretsFilter
    llmAPIKey     string
}

func NewSpawnTool(dockerPool *docker.ContainerPool, secretsFilter *docker.SecretsFilter, llmAPIKeyEnv string) *SpawnTool {
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

func (t *SpawnTool) Name() string {
    return "spawn"
}

func (t *SpawnTool) Description() string {
    return "Delegate task to isolated subagent for external data fetching (web, APIs)"
}

func (t *SpawnTool) Schema() Schema {
    return Schema{
        Name:        "spawn",
        Description: "Delegate task to isolated subagent for external data fetching",
        Parameters: map[string]interface{}{
            "type": "object",
            "properties": map[string]interface{}{
                "task": map[string]interface{}{
                    "type":        "string",
                    "description": "Task description for subagent",
                },
                "timeout_seconds": map[string]interface{}{
                    "type":        "integer",
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
        },
    }
}

func (t *SpawnTool) ExecuteWithContext(ctx context.Context, args string) (string, error) {
    var spawnArgs SpawnArgs
    if err := json.Unmarshal([]byte(args), &spawnArgs); err != nil {
        return "", fmt.Errorf("failed to parse arguments: %w", err)
    }
    
    if t.dockerPool == nil {
        // Graceful degradation (v13) — локальное выполнение для safe tasks
        if t.isLocalExecutionSafe(spawnArgs) {
            return t.executeLocally(ctx, spawnArgs)
        }
        return "", &DockerUnavailableError{
            Task: spawnArgs.Task,
            Hint: "Docker pool is not initialized",
        }
    }
    
    if !t.dockerPool.IsHealthy() {
        // Graceful degradation (v13) — локальное выполнение для safe tasks
        if t.isLocalExecutionSafe(spawnArgs) {
            return t.executeLocally(ctx, spawnArgs)
        }
        return "", &DockerUnavailableError{
            Task: spawnArgs.Task,
            Hint: "Docker pool is unhealthy or circuit breaker is open",
        }
    }
    
    secrets, err := t.secretsFilter.FilterForTask(spawnArgs.RequiredSecrets)
    if err != nil {
        return "", fmt.Errorf("failed to filter secrets: %w", err)
    }
    
    timeout := spawnArgs.TimeoutSeconds
    if timeout == 0 {
        timeout = 60
    }
    
    req := docker.SubagentRequest{
        Version:       docker.ProtocolVersion,
        Type:          "execute",
        ID:            uuid.New().String(),
        CorrelationID: generateCorrelationID(),
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

func generateCorrelationID() string {
    return uuid.New().String()[:16]
}

// === Graceful Degradation (v14) ===

// isLocalExecutionSafe проверяет, можно ли выполнить задачу локально без Docker
func (t *SpawnTool) isLocalExecutionSafe(args SpawnArgs) bool {
    task := args.Task
    
    // v14: Декодируем URL encoding для обнаружения скрытых URL
    if decoded, err := url.QueryUnescape(task); err == nil {
        task = decoded
    }
    taskLower := strings.ToLower(task)
    
    // Небезопасно выполнять локально если есть внешние URL
    // v14: Проверяем также URL-encoded версии
    if strings.Contains(taskLower, "http://") || 
       strings.Contains(taskLower, "https://") ||
       strings.Contains(taskLower, "http%3a%2f%2f") ||
       strings.Contains(taskLower, "https%3a%2f%2f") {
        return false
    }
    
    // Небезопасно если требуются секреты
    if len(args.RequiredSecrets) > 0 {
        return false
    }
    
    // Небезопасно если есть признаки prompt injection в задаче
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

// executeLocally выполняет простую задачу без Docker (fallback)
func (t *SpawnTool) executeLocally(ctx context.Context, args SpawnArgs) (string, error) {
    // Для MVP — возвращаем информативное сообщение
    // В будущем можно добавить локальную обработку простых задач
    return "", &DockerUnavailableError{
        Task: args.Task,
        Hint: "Docker unavailable and task cannot be executed safely without isolation",
    }
}
```

## Обработка ошибок в агенте

### В `internal/agent/loop.go`

```go
func (l *Loop) executeToolCall(ctx context.Context, call ToolCall) (string, error) {
    result, err := l.registry.ExecuteWithContext(ctx, call.Name, call.Arguments)
    if err != nil {
        // Специальная обработка DockerUnavailableError
        if dockerErr, ok := err.(*tools.DockerUnavailableError); ok {
            return dockerErr.UserMessage(), nil
        }
        return "", err
    }
    return result, nil
}
```

## Сценарии ошибок

### 1. Docker не установлен/не запущен

```
⚠️ Не могу выполнить задачу с внешними данными.

Задача: Fetch https://example.com/data

Причина: Docker-изоляция недоступна.

Решение:
1. Убедитесь, что Docker установлен и запущен: docker ps
2. Проверьте образ: docker images | grep nexbot/subagent
```

### 2. Circuit breaker открыт

```
⚠️ Не могу выполнить задачу с внешними данными.

Задача: Fetch https://example.com/data

Причина: circuit breaker open, retry after 30s

Решение:
1. Подождите 30 секунд
2. Если проблема повторяется, проверьте логи контейнеров
```

### 3. Rate limit

```
Too many requests, retry after 1s
```

### 4. Queue full

```
Task queue full, retry after 500ms
```

## Ключевые решения

1. **DockerUnavailableError** — специальный тип ошибки с UserMessage()
2. **Проверка IsHealthy()** — проверка перед выполнением
3. **Graceful degradation** — понятное сообщение вместо технической ошибки
4. **Type switch для ошибок** — разная обработка разных типов
