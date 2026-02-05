# Heartbeat + Cron Features - Implementation Plan

**Status:** Pending  
**Version:** 0.2.5  
**Created:** 2026-02-05  
**Dependencies:** v0.2.0 (Cron + Spawn) COMPLETE

---

## Overview

Реализует heartbeart check service и расширенную cron систему с поддержкой:
- HEARTBEAT.md — файл с задачами на человеческом языке
- cron.jsonl — JSONL persistence для cron задач (recurring + oneshot)
- Send Message Tool — tool для отправки сообщений через LLM
- Heartbeat Checker — сервис проверок HEARTBEAT.md каждые 10 минут

---

## Architecture

```
┌──────────────────────────────────────────────┐
│ Workspace (~/.nexbot/)                   │
│ ├── HEARTBEAT.md  ← LLM создает задачи  │
│ └── cron.jsonl     ← Runtime задачи     │
└──────────────────────────────────────────────┘
                    │
                    │ 10 минут
                    ▼
           ┌──────────────────┐
           │ Heartbeat Check │  ← Отправляет запрос к Agent LLM
           └────────┬─────────┘
                    │ "проверь HEARTBEAT.md, если что-то пора выполнить - сделай это"
                    ▼
            ┌──────────────┐
            │ Agent LLM     │
            └──────┬───────┘
                   │
                   ├──────────────┬──────────────────┐
                   │              │                  │
                   ▼              ▼                  ▼
          ┌─────────────┐  ┌─────────────┐  ┌─────────────┐
          │HEARTBEAT.md│  │cron.jsonl  │  │cron.jsonl  │
          │(heartbeat) │  │(recurring) │  │(oneshot)   │
          └──────┬─────┘  └──────┬─────┘  └──────┬─────┘
                 │                │                │
                 └────────────────┴────────────────┘
                                  │
                                  ▼
                        ┌─────────────────────┐
                        │ Cron Scheduler      │
                        │ - recurring (cron) │
                        │ - oneshot (time)  │
                        │ - cleanup (24h)    │
                        └────────┬────────────┘
                                 │
                                 ▼
                           Worker Pool
```

---

## Features

### 1. HEARTBEAT.md
- Файл с задачами на человеческом языке
- Создается в workspace при старте если отсутствует
- LLM управляет через file tools (read_file, write_file)
- Содержит задачи и конфигурацию отправки (User ID, Channel)

### 2. cron.jsonl Persistence
- JSONL формат (одна задача на строку)
- Поддержка recurring и oneshot задач
- Atomic write (через temp файл)
- Cleanup executed oneshot задач каждые 24 часа

### 3. Cron Scheduler
- Recurring задачи: robfig/cron/v3
- Oneshot задачи: time.Ticker проверка каждую минуту
- Executed oneshot: пометить как выполненный
- Cleanup: удалить executed oneshot из памяти и файла каждые 24 часа

### 4. Send Message Tool
- Tool для LLM: send_message
- Параметры: user_id (default: "user"), channel_type (default: "telegram"), session_id (default: "heartbeat-check"), message (required)
- Отправляет через MessageBus.Outbound → Telegram

### 5. Heartbeat Checker
- Запускается каждые 10 минут
- Отправляет prompt к Agent.ProcessHeartbeatCheck()
- LLM читает HEARTBEAT.md
- Если пора выполнить: использует send_message tool
- Если ничего: возвращает "HEARTBEAT_OK"
- Checker проверяет: если HEARTBEAT_OK → логирует, иначе → ничего (LLM сам отправил через tool)

---

## Implementation Phases

### Phase 1: Update Configuration

**Files:**
- `config.example.toml`
- `internal/config/schema.go`

**Changes:**

`config.example.toml`:
```toml
[cron]
enabled = true
timezone = "UTC"

[heartbeat]
enabled = true
check_interval_minutes = 10
```

`schema.go`:
```go
type CronConfig struct {
    Enabled  bool   `toml:"enabled"`
    Timezone string `toml:"timezone"`
}

type HeartbeatConfig struct {
    Enabled           bool `toml:"enabled"`
    CheckIntervalMinutes int  `toml:"check_interval_minutes"`
}
```

---

### Phase 2: JSONL Storage Layer

**Files:**
- `internal/cron/storage.go` (NEW)
- `internal/cron/storage_test.go` (NEW)

**API:**

```go
type Storage struct {
    filePath string
    logger   *logger.Logger
}

func NewStorage(workspacePath string, logger *logger.Logger) *Storage
func (s *Storage) Load() ([]Job, error)
func (s *Storage) Append(job Job) error
func (s *Storage) Remove(jobID string) error
func (s *Storage) Save(jobs []Job) error
func (s *Storage) RemoveExecutedOneshots() error
```

**Format cron.jsonl:**

```jsonl
{"id":"job_xxx","type":"recurring","schedule":"0 0 9 * * *","command":"standup","user_id":"llm"}
{"id":"job_yyy","type":"oneshot","execute_at":"2026-02-06T10:00:00Z","command":"buy milk","user_id":"llm","executed":false}
{"id":"job_zzz","type":"oneshot","execute_at":"2026-02-05T09:00:00Z","command":"old reminder","user_id":"llm","executed":true,"executed_at":"2026-02-05T09:00:01Z"}
```

**Tests:**
- TestStorageLoadEmpty
- TestStorageLoadJobs
- TestStorageAppendJob
- TestStorageRemoveJob
- TestStorageSaveJobs
- TestStorageRemoveExecutedOneshots

---

### Phase 3: Extend Cron Scheduler

**Files:**
- `internal/cron/scheduler.go`
- `internal/cron/scheduler_test.go`

**Changes:**

```go
// Job types
type JobType string
const (
    JobTypeRecurring JobType = "recurring"
    JobTypeOneshot   JobType = "oneshot"
)

// Extended Job struct
type Job struct {
    ID        string     `json:"id"`
    Type      JobType    `json:"type"`
    Schedule  string     `json:"schedule,omitempty"`
    ExecuteAt *time.Time `json:"execute_at,omitempty"`
    Command   string     `json:"command"`
    UserID    string     `json:"user_id,omitempty"`
    Metadata  map[string]string `json:"metadata,omitempty"`
    Executed  bool       `json:"executed,omitempty"`
    ExecutedAt *time.Time `json:"executed_at,omitempty"`
}

// Storage field added
type Scheduler struct {
    cron       *cron.Cron
    logger     *logger.Logger
    bus        *bus.MessageBus
    workerPool WorkerPool
    storage    *Storage
    ctx        context.Context
    cancel     context.CancelFunc
    started    bool
    mu         sync.RWMutex
    
    jobs        map[string]Job
    jobIDs      map[cron.EntryID]string
    jobEntryIDs map[string]cron.EntryID
}

// Oneshot ticker (every 1 minute)
func (s *Scheduler) oneshotTicker()
func (s *Scheduler) checkAndExecuteOneshots(now time.Time)
func (s *Scheduler) executedCleanup()  // every 24 hours
func (s *Scheduler) CleanupExecutedOneshots()
```

**Tests:**
- TestSchedulerOneshotExecution
- TestSchedulerOneshotAlreadyExecuted
- TestSchedulerCleanupExecuted
- TestSchedulerStorageIntegration

---

### Phase 4: Cron Tool

**Files:**
- `internal/tools/cron.go` (NEW)
- `internal/tools/cron_test.go` (NEW)

**API:**

```go
type CronTool struct {
    scheduler *cron.Scheduler
    storage   *cron.Storage
    logger    *logger.Logger
}

func NewCronTool(scheduler *cron.Scheduler, storage *cron.Storage, logger *logger.Logger) *CronTool
func (t *CronTool) Name() string { return "cron" }
func (t *CronTool) Description() string { return "Manage scheduled tasks (recurring and one-time reminders)" }
func (t *CronTool) Parameters() map[string]interface{}
func (t *CronTool) Execute(ctx context.Context, params map[string]interface{}) (string, error)

// Actions:
func (t *CronTool) addRecurring(ctx context.Context, params map[string]interface{}) (string, error)
func (t *CronTool) addOneshot(ctx context.Context, params map[string]interface{}) (string, error)
func (t *CronTool) removeJob(ctx context.Context, params map[string]interface{}) (string, error)
func (t *CronTool) listJobs(ctx context.Context) (string, error)
```

**Usage by LLM:**

```
User: "Напомни завтра в 10:00 купить молоко"
↓
LLM: add_oneshot("2026-02-06T10:00:00Z", "Напомнить: купить молоко")
↓
CronTool: добавляет в cron.jsonl, scheduler.AddJob()
```

**Tests:**
- TestCronToolAddRecurring
- TestCronToolAddOneshot
- TestCronToolRemoveJob
- TestCronToolListJobs
- TestCronToolInvalidCron

---

### Phase 5: Send Message Tool

**Files:**
- `internal/tools/message.go` (NEW)
- `internal/tools/message_test.go` (NEW)

**API:**

```go
type SendMessageTool struct {
    messageBus *bus.MessageBus
    logger     *logger.Logger
}

func NewSendMessageTool(messageBus *bus.MessageBus, logger *logger.Logger) *SendMessageTool
func (t *SendMessageTool) Name() string { return "send_message" }
func (t *SendMessageTool) Description() string { return "Send a message to a user channel" }

func (t *SendMessageTool) Parameters() map[string]interface{} {
    return map[string]interface{}{
        "user_id": map[string]interface{}{
            "type":        "string",
            "description": "User ID to send message to",
            "default":     "user",
        },
        "channel_type": map[string]interface{}{
            "type":        "string",
            "description": "Channel type (e.g., 'telegram')",
            "default":     "telegram",
        },
        "session_id": map[string]interface{}{
            "type":        "string",
            "description": "Session ID for message tracking",
            "default":     "heartbeat-check",
        },
        "message": map[string]interface{}{
            "type":        "string",
            "description": "Message content to send",
            "required":    true,
        },
    }
}

func (t *SendMessageTool) Execute(ctx context.Context, params map[string]interface{}) (string, error)
```

**Tests:**
- TestSendMessageToolDefaults
- TestSendMessageToolCustomUser
- TestSendMessageToolCustomChannel
- TestSendMessageToolPublishError

---

### Phase 6: Agent Integration

**Files:**
- `internal/agent/loop/loop.go`

**Changes:**

```go
// Add ProcessHeartbeatCheck method
func (l *Loop) ProcessHeartbeatCheck(ctx context.Context) (string, error) {
    heartbeatPrompt := "Read HEARTBEAT.md from workspace. Follow it strictly. Do not infer or repeat old tasks from prior chats. If nothing needs attention, reply HEARTBEAT_OK."
    
    response, err := l.llmProvider.Complete(ctx, heartbeatPrompt, nil)
    if err != nil {
        return "", fmt.Errorf("heartbeat check failed: %w", err)
    }
    
    l.logger.DebugCtx(ctx, "heartbeat check response",
        logger.Field{Key: "response_length", Value: len(response)})
    
    return response, nil
}
```

---

### Phase 7: Heartbeat Checker

**Files:**
- `internal/heartbeat/checker.go` (NEW)
- `internal/heartbeat/checker_test.go` (NEW)

**API:**

```go
package heartbeat

const (
    heartbeatPrompt = "Read HEARTBEAT.md from workspace. Follow it strictly. Do not infer or repeat old tasks from prior chats. If nothing needs attention, reply HEARTBEAT_OK."
    heartbeatOKToken = "HEARTBEAT_OK"
)

// Agent interface for heartbeat checks
type Agent interface {
    ProcessHeartbeatCheck(ctx context.Context) (string, error)
}

// Checker sends periodic heartbeat check requests to agent
type Checker struct {
    interval time.Duration
    agent    Agent
    logger   *logger.Logger
    ctx      context.Context
    cancel   context.CancelFunc
    started  bool
}

func NewChecker(intervalMinutes int, agent Agent, logger *logger.Logger) *Checker
func (c *Checker) Start(ctx context.Context) error
func (c *Checker) Stop() error
func (c *Checker) processResponse(response string)
```

**Behavior:**

1. Starts ticker (10 minutes)
2. Sends prompt to Agent.ProcessHeartbeatCheck()
3. Agent reads HEARTBEAT.md
4. Agent uses tools to send messages or reply "HEARTBEAT_OK"
5. Checker checks: if "HEARTBEAT_OK" → log "all good", else → nothing (LLM used send_message tool)

**Tests:**
- TestCheckerStartStop
- TestCheckerProcessResponseOK
- TestCheckerProcessResponseAlert
- TestCheckerHeartbeatOKToken

---

### Phase 8: Serve Integration

**Files:**
- `cmd/nexbot/serve.go`

**Changes:**

```go
// 1. Initialize storage (after worker pool)
cronStorage := cron.NewStorage(ws.Path(), log)

// 2. Load jobs from cron.jsonl
cronJobs, err := cronStorage.Load()
if err != nil {
    log.Error("Failed to load cron jobs", err)
} else {
    log.InfoCtx(ctx, "Loaded cron jobs",
        logger.Field{Key: "count", Value: len(cronJobs)})
}

// 3. Initialize cron scheduler
var cronScheduler *cron.Scheduler
if cfg.Cron.Enabled {
    log.Info("⏰ Initializing cron scheduler")
    
    workerPoolAdapter := &cronWorkerPoolAdapter{pool: workerPool}
    cronScheduler = cron.NewScheduler(log, messageBus, workerPoolAdapter, cronStorage)
    if err := cronScheduler.Start(ctx); err != nil {
        log.Error("Failed to start cron scheduler", err)
        os.Exit(1)
    }
    
    // 4. Load jobs into scheduler
    for _, job := range cronJobs {
        if _, err := cronScheduler.AddJob(job); err != nil {
            log.WarnCtx(ctx, "Failed to add cron job",
                logger.Field{Key: "error", Value: err},
                logger.Field{Key: "job_id", Value: job.ID})
        }
    }
    
    log.Info("✅ Cron scheduler started")
}

// 5. Register CronTool
if cronScheduler != nil {
    cronTool := tools.NewCronTool(cronScheduler, cronStorage, log)
    agentLoop.RegisterTool(cronTool)
    log.Info("✅ Cron tool registered")
}

// 6. Register SendMessageTool
sendMessageTool := tools.NewSendMessageTool(messageBus, log)
agentLoop.RegisterTool(sendMessageTool)
log.Info("✅ Send message tool registered")

// 7. Initialize heartbeat checker
var heartbeatChecker *heartbeat.Checker
if cfg.Heartbeat.Enabled && cronScheduler != nil {
    log.Info("💓 Initializing heartbeat checker",
        logger.Field{Key: "interval_minutes", Value: cfg.Heartbeat.CheckIntervalMinutes})
    
    heartbeatChecker = heartbeat.NewChecker(
        cfg.Heartbeat.CheckIntervalMinutes,
        agentLoop,
        log,
    )
    go heartbeatChecker.Start(ctx)
    log.Info("✅ Heartbeat checker started")
}

// 8. Create HEARTBEAT.md if not exists
heartbeatPath := filepath.Join(ws.Path(), "HEARTBEAT.md")
if _, err := os.Stat(heartbeatPath); os.IsNotExist(err) {
    log.Info("Creating HEARTBEAT.md bootstrap")
    
    defaultHeartbeatContent := `# HEARTBEAT - Задачи и отправка

Этот файл читается каждые 10 минут. 

## Как использовать

### Для LLM

1. Читай секцию "Задачи"
2. Проверяй время выполнения
3. Если пора выполнить:
   - Выполни задачу (используй доступные tools: read_file, write_file, send_message)
   - Если нужно отправить сообщение — используй send_message tool
   - Если нужно обновить HEARTBEAT.md — используй write_file tool
4. Если ничего — верни "HEARTBEAT_OK"

## Задачи

---

Добавляй задачи сюда.
`
    
    if err := os.WriteFile(heartbeatPath, []byte(defaultHeartbeatContent), 0644); err != nil {
        log.Warn("Failed to create HEARTBEAT.md", err)
    } else {
        log.Info("✅ HEARTBEAT.md created")
    }
```

---

### Phase 9: Documentation

**Files:**
- `README.md`
- `docs/CONFIGURATION.md`

**Changes:**

`README.md` — add section:

```md
## Heartbeat

Heartbeat запускает проверку HEARTBEAT.md каждые 10 минут. 
Если есть задачи для выполнения — агент их выполнит через send_message tool.
Если ничего — ответит HEARTBEAT_OK (не отправляется пользователю).

Файл HEARTBEAT.md находится в workspace и управляется агентом через file tools (read_file, write_file).

### Конфигурация

```toml
[heartbeat]
enabled = true
check_interval_minutes = 10
```

### Доступные tools для Heartbeat

- `send_message` — отправить сообщение в канал
- `read_file` — прочитать файл
- `write_file` — записать файл
- `cron` — управление cron задачами
```

`docs/CONFIGURATION.md`:

```md
### Heartbeat Configuration

| Параметр               | Тип  | Default | Описание                    |
| ---------------------- | ---- | ------- | --------------------------- |
| enabled                | bool | true    | Включить heartbeat проверки |
| check_interval_minutes | int  | 10      | Интервал проверки в минутах |

### Cron Configuration

| Параметр | Тип    | Default | Описание                |
| --------- | ------ | ------- | ----------------------- |
| enabled  | bool   | true    | Включить cron scheduler |
| timezone | string | UTC     | Часовой пояс            |
```

---

### Phase 10: Testing

**Unit tests:**
- `internal/cron/storage_test.go`
- `internal/cron/scheduler_test.go`
- `internal/tools/cron_test.go`
- `internal/tools/message_test.go`
- `internal/heartbeat/checker_test.go`

**Integration test:**
- `internal/cron/integration_test.go` — full workflow

**Manual testing checklist:**
- [ ] CronTool: add_recurring работает
- [ ] CronTool: add_oneshot работает
- [ ] CronTool: remove работает
- [ ] CronTool: list работает
- [ ] SendMessageTool отправляет сообщения
- [ ] Oneshot задачи выполняются вовремя
- [ ] Oneshot задачи удаляются через сутки
- [ ] Heartbeat check отправляет запросы каждые 10 минут
- [ ] Heartbeat HEARTBEAT_OK не отправляется в Telegram
- [ ] Heartbeat алерты отправляются в Telegram
- [ ] LLM использует send_message для heartbeat алертов
- [ ] LLM использует write_file для обновления HEARTBEAT.md
- [ ] HEARTBEAT.md создается если нет
- [ ] cron.jsonl создается и загружается

---

## Dependencies

```
Phase 1 (Config) → Phase 2 (Storage) → Phase 3 (Scheduler) → Phase 4 (CronTool)
                                                                 ↓
                                                        Phase 5 (Send Message)
                                                                 ↓
                                                        Phase 6 (Agent)
                                                                 ↓
                                                        Phase 7 (Heartbeat Checker)
                                                                 ↓
                                                        Phase 8 (Serve Integration)
                                                                 ↓
                                                        Phase 9 (Docs)
                                                                 ↓
                                                        Phase 10 (Testing)
```

---

## Files to Create/Modify

### New files:
- `internal/cron/storage.go`
- `internal/cron/storage_test.go`
- `internal/tools/cron.go`
- `internal/tools/cron_test.go`
- `internal/tools/message.go`
- `internal/tools/message_test.go`
- `internal/heartbeat/checker.go`
- `internal/heartbeat/checker_test.go`
- `internal/cron/integration_test.go`

### Modify files:
- `config.example.toml` — add check_interval_minutes, timezone to [heartbeat], [cron]
- `internal/config/schema.go` — update HeartbeatConfig, CronConfig
- `internal/cron/scheduler.go` — add JobType, oneshot support, storage, cleanup
- `internal/cron/scheduler_test.go` — add oneshot tests
- `internal/agent/loop/loop.go` — add ProcessHeartbeatCheck method
- `cmd/nexbot/serve.go` — integration
- `README.md` — add Heartbeat documentation
- `docs/CONFIGURATION.md` — add Heartbeat + Cron config
- `workspace/HEARTBEAT.md` — bootstrap file (created if missing)

---

## Success Criteria

- [ ] HEARTBEAT.md создается в workspace при старте
- [ ] Heartbeat checker запускается каждые 10 минут
- [ ] Heartbeat checker отправляет запросы к Agent.ProcessHeartbeatCheck()
- [ ] Agent читает HEARTBEAT.md и использует tools
- [ ] HEARTBEAT_OK обрабатывается корректно (не отправляется в Telegram)
- [ ] Cron scheduler поддерживает recurring (robfig/cron) задачи
- [ ] Cron scheduler поддерживает oneshot задачи
- [ ] Oneshot задачи проверяются каждую минуту
- [ ] Executed oneshot задачи удаляются каждые 24 часа
- [ ] cron.jsonl используется для persistence
- [ ] CronTool работает (add_recurring, add_oneshot, remove, list)
- [ ] SendMessageTool работает
- [ ] LLM использует send_message tool для heartbeat алертов
- [ ] Unit tests для всех компонентов
- [ ] Integration test проходит
- [ ] Документация обновлена
- [ ] `make ci` проходит

---

## Notes

- CLI commands отложены (будут в отдельной задаче)
- HEARTBEAT.md bootstrap создается в workspace при старте
- UserID для Send Message Tool: "user" (generic)
- Channel type для Send Message Tool: "telegram" (hardcode)
- Session ID для heartbeat: "heartbeat-check"
- Oneshot cleanup interval: 24 часа
- Timezone для cron: "UTC"
- Check interval для heartbeat: 10 минут
