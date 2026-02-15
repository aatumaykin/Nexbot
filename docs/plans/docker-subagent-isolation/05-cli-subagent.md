# Этап 5: CLI сабагент (MVP)

## Цель

Реализация CLI приложения для сабагента, работающего внутри Docker-контейнера.

## Файлы

### 5.1 `cmd/subagent/main.go`

```go
package main

import (
    "bufio"
    "context"
    "encoding/json"
    "fmt"
    "io"
    "os"
    "os/signal"
    "strings"
    "syscall"
    "time"

    "github.com/aatumaykin/nexbot/internal/agent"
    "github.com/aatumaykin/nexbot/internal/security"
    "github.com/aatumaykin/nexbot/internal/subagent/prompts"
    "github.com/aatumaykin/nexbot/internal/subagent/sanitizer"
    "github.com/aatumaykin/nexbot/internal/tools"
    "github.com/aatumaykin/nexbot/internal/llm/zai"
)

const (
    ProtocolVersion   = "1.0"
    DefaultSkillsPath = "/workspace/skills"
    DefaultTimezone   = "UTC"
    MaxRequestSize    = 1 * 1024 * 1024
)

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

type Subagent struct {
    secretsStore  *security.SecretsStore
    llmClient     *zai.Client
    loop          *agent.Loop
    registry      *tools.Registry
    systemPrompt  string
    validator     *sanitizer.Validator
    promptLoader  *prompts.PromptLoader
}

func NewSubagent() *Subagent {
    registry := tools.NewRegistry()
    registerSubagentTools(registry)
    
    return &Subagent{
        secretsStore: security.NewSecretsStore(5 * time.Minute),
        registry:     registry,
        validator:    sanitizer.NewValidator(sanitizer.SanitizerConfig{}),
        promptLoader: prompts.NewPromptLoader(""),
    }
}

func (s *Subagent) InitLLM(apiKey string) {
    s.llmClient = zai.NewClient(zai.Config{
        APIKey: apiKey,
        Model:  "glm-4-flash",
    })
    
    skillsPath := getSkillsPath()
    promptBuilder := prompts.NewSubagentPromptBuilder(DefaultTimezone, s.registry, skillsPath)
    s.systemPrompt = promptBuilder.Build()
    
    s.loop = agent.NewLoop(agent.LoopConfig{
        SystemPrompt:  s.systemPrompt,
        LLM:           s.llmClient,
        Tools:         s.registry,
        MaxIterations: 10,
    })
}

func main() {
    subagent := NewSubagent()
    
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)
    
    go func() {
        <-sigChan
        logInfo("", "shutdown", "received shutdown signal", nil)
        cancel()
    }()
    
    limitedReader := io.LimitReader(os.Stdin, MaxRequestSize)
    scanner := bufio.NewScanner(limitedReader)
    scanner.Buffer(make([]byte, 64*1024), MaxRequestSize)
    
    for scanner.Scan() {
        select {
        case <-ctx.Done():
            logInfo("", "shutdown", "shutdown complete", nil)
            return
        default:
        }
        
        line := scanner.Bytes()
        if len(line) == 0 {
            continue
        }
        
        var req SubagentRequest
        if err := json.Unmarshal(line, &req); err != nil {
            sendError("", "", fmt.Errorf("invalid JSON: %w", err))
            continue
        }
        
        compatible, deprecated := isCompatibleVersion(req.Version)
        if !compatible {
            sendError(req.ID, req.CorrelationID, 
                fmt.Errorf("unsupported protocol version: %s", req.Version))
            continue
        }
        if deprecated {
            logInfo(req.ID, "protocol", "deprecated version", map[string]interface{}{
                "version": req.Version,
            })
        }
        
        if req.Type == "ping" {
            sendPong(req.ID, req.CorrelationID)
            continue
        }
        
        if req.LLMAPIKey != "" && subagent.llmClient == nil {
            subagent.InitLLM(req.LLMAPIKey)
            logInfo(req.ID, "init", "LLM client initialized", nil)
        }
        
        if subagent.loop == nil {
            sendError(req.ID, req.CorrelationID, 
                fmt.Errorf("LLM not initialized: send llm_api_key"))
            continue
        }
        
        if req.Secrets != nil {
            if err := subagent.secretsStore.SetAll(req.Secrets); err != nil {
                sendError(req.ID, req.CorrelationID, fmt.Errorf("secrets validation: %w", err))
                continue
            }
        }
        
        logInfo(req.ID, "task", "processing", map[string]interface{}{
            "correlation_id": req.CorrelationID,
            "secret_count":   len(req.Secrets),
        })
        
        preparedTask := sanitizer.PrepareTask(req.Task)
        
        taskCtx := ctx
        var taskCancel context.CancelFunc
        if req.Deadline > 0 {
            deadline := time.Unix(req.Deadline, 0)
            if time.Now().After(deadline) {
                sendError(req.ID, req.CorrelationID, fmt.Errorf("request expired"))
                continue
            }
            taskCtx, taskCancel = context.WithDeadline(ctx, deadline)
        }
        
        result, err := subagent.loop.Process(taskCtx, preparedTask)
        
        if taskCancel != nil {
            taskCancel()
        }
        
        result = subagent.validator.SanitizeToolOutput(result)
        
        subagent.secretsStore.Clear()
        
        if err != nil {
            logError(req.ID, "task", "failed", err.Error())
        } else {
            logInfo(req.ID, "task", "completed", nil)
        }
        
        sendResponse(req.ID, req.CorrelationID, result, err)
    }
}

func isCompatibleVersion(v string) (compatible bool, deprecated bool) {
    switch v {
    case "", "1.0":
        return true, false
    case "0.9":
        return true, true
    default:
        return false, false
    }
}

func getSkillsPath() string {
    if path := os.Getenv("SKILLS_PATH"); path != "" {
        return path
    }
    return DefaultSkillsPath
}

func sendResponse(id, correlationID, result string, err error) {
    resp := map[string]interface{}{
        "id":             id,
        "correlation_id": correlationID,
        "version":        ProtocolVersion,
        "status":         "success",
        "result":         result,
    }
    if err != nil {
        resp["status"] = "error"
        resp["error"] = err.Error()
    }
    data, _ := json.Marshal(resp)
    os.Stdout.Write(data)
    os.Stdout.Write([]byte("\n"))
}

func sendError(id, correlationID string, err error) {
    resp := map[string]interface{}{
        "id":             id,
        "correlation_id": correlationID,
        "version":        ProtocolVersion,
        "status":         "error",
        "error":          err.Error(),
    }
    data, _ := json.Marshal(resp)
    os.Stdout.Write(data)
    os.Stdout.Write([]byte("\n"))
}

func sendPong(id, correlationID string) {
    resp := map[string]interface{}{
        "id":             id,
        "correlation_id": correlationID,
        "version":        ProtocolVersion,
        "status":         "pong",
    }
    data, _ := json.Marshal(resp)
    os.Stdout.Write(data)
    os.Stdout.Write([]byte("\n"))
}

type LogEntry struct {
    Time          string                 `json:"time"`
    Level         string                 `json:"level"`
    Message       string                 `json:"message"`
    TaskID        string                 `json:"task_id,omitempty"`
    CorrelationID string                 `json:"correlation_id,omitempty"`
    Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

func logInfo(taskID, event, msg string, metadata map[string]interface{}) {
    if metadata == nil {
        metadata = make(map[string]interface{})
    }
    metadata["event"] = event
    entry := LogEntry{
        Time:     time.Now().Format(time.RFC3339),
        Level:    "info",
        Message:  msg,
        TaskID:   taskID,
        Metadata: metadata,
    }
    data, _ := json.Marshal(entry)
    os.Stderr.Write(data)
    os.Stderr.Write([]byte("\n"))
}

func logError(taskID, event, msg, errMsg string) {
    entry := LogEntry{
        Time:    time.Now().Format(time.RFC3339),
        Level:   "error",
        Message: msg,
        TaskID:  taskID,
        Metadata: map[string]interface{}{
            "event": event,
            "error": errMsg,
        },
    }
    data, _ := json.Marshal(entry)
    os.Stderr.Write(data)
    os.Stderr.Write([]byte("\n"))
}
```

### 5.2 `cmd/subagent/tools.go`

```go
package main

import (
    "github.com/aatumaykin/nexbot/internal/config"
    "github.com/aatumaykin/nexbot/internal/logger"
    "github.com/aatumaykin/nexbot/internal/tools"
    "github.com/aatumaykin/nexbot/internal/tools/fetch"
)

func registerSubagentTools(registry *tools.Registry) {
    log := logger.NewLogger(logger.Config{Level: "info"})
    
    fetchCfg := &config.Config{
        Tools: config.ToolsConfig{
            Fetch: config.FetchToolConfig{
                Enabled:         true,
                TimeoutSeconds:  30,
                MaxResponseSize: 5 * 1024 * 1024,
                UserAgent:       "Nexbot-Subagent/1.0",
            },
        },
    }
    registry.Register(fetch.NewFetchTool(fetchCfg, log))
}
```

## Протокол

### Типы запросов

1. **ping** — health check
   ```json
   {"version": "1.0", "type": "ping", "id": "check-123"}
   ```

2. **execute** — выполнение задачи
   ```json
   {
     "version": "1.0",
     "type": "execute",
     "id": "task-uuid",
     "task": "Fetch https://example.com/data",
     "timeout": 60,
     "secrets": {"API_KEY": "secret"},
     "llm_api_key": "zai-key"
   }
   ```

### Типы ответов

1. **pong** — ответ на ping
   ```json
   {"id": "check-123", "status": "pong", "version": "1.0"}
   ```

2. **success** — успешное выполнение
   ```json
   {"id": "task-uuid", "status": "success", "result": "...", "version": "1.0"}
   ```

3. **error** — ошибка
   ```json
   {"id": "task-uuid", "status": "error", "error": "...", "version": "1.0"}
   ```

## Ключевые решения (MVP)

1. **Единый SecretsStore** — использует `internal/security/secrets.go` (нет дублирования)
2. **MaxRequestSize 1MB** — защита от OOM
3. **Lazy LLM init** — инициализация при первом запросе
4. **Version negotiation** — поддержка совместимости
5. **JSON logging в stderr** — структурированные логи
6. **Secret cleanup после задачи** — автоматическая очистка
