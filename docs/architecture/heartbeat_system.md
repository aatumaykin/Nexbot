# Heartbeat System Architecture

## Overview

Heartbeat System — это компонент для периодической проверки состояния системы через HEARTBEAT.md файл. Он загружает задачи, парсит их, и через Checker периодически выполняет проверки через Agent интерфейс. Система интегрируется с Cron Scheduler для планирования задач.

## Components

### 1. Loader (`internal/heartbeat/loader.go`)

**Responsibilities:**
- Загрузка HEARTBEAT.md файла из workspace
- Валидация загруженных задач
- Форматирование контекста для включения в system context

**Key Fields:**
```go
type Loader struct {
    parser    *Parser         // Parser for HEARTBEAT.md
    workspace string          // Path to workspace directory
    logger    *logger.Logger  // Logger instance
    tasks     []HeartbeatTask // Loaded heartbeat tasks
    isLoaded  bool            // Load status flag
}
```

**Key Methods:**
- `Load()` — загрузка и парсинг HEARTBEAT.md
- `GetTasks()` — получение загруженных задач
- `GetContext()` — форматирование контекста для LLM
- `SetTasks(tasks)` — установка задач (для тестов)

### 2. Parser (`internal/heartbeat/parser.go`)

**Responsibilities:**
- Парсинг HEARTBEAT.md контента
- Извлечение задач в формате markdown
- Валидация cron выражений
- Форматирование контекста для вывода

**Key Fields:**
```go
type Parser struct{}
```

**Key Methods:**
- `Parse(content)` — парсинг контента в список задач
- `Validate(task)` — валидация задачи
- `splitSections(content)` — разделение на секции по `##`
- `splitTaskSections(section)` — разделение на подзадачи по `###`
- `parseTaskSection(section)` — парсинг одной задачи
- `extractQuotedValue(line)` — извлечение значения в кавычках

**HeartbeatTask Structure:**
```go
type HeartbeatTask struct {
    Name     string `json:"name"`     // Task name (e.g., "Daily Standup")
    Schedule string `json:"schedule"` // Cron expression (e.g., "0 9 * * *")
    Task     string `json:"task"`     // Task description/command
}
```

### 3. Checker (`internal/heartbeat/checker.go`)

**Responsibilities:**
- Периодическая проверка heartbeat статуса
- Вызов Agent.ProcessHeartbeatCheck() по расписанию
- Обработка ответов от агента
- Graceful shutdown

**Key Fields:**
```go
type Checker struct {
    interval time.Duration       // Check interval
    agent    Agent              // Agent interface for heartbeat checks
    logger   *logger.Logger     // Logger instance
    ctx      context.Context    // Checker context
    cancel   context.CancelFunc // Cancel function
    started  bool               // Start status flag
    mu       sync.RWMutex       // Mutex for thread-safety
}
```

**Key Methods:**
- `Start()` — запуск checker loop
- `Stop()` — остановка checker
- `run()` — основной цикл проверок
- `check()` — выполнение одной проверки
- `processResponse(response)` — обработка ответа

**Agent Interface:**
```go
type Agent interface {
    ProcessHeartbeatCheck(ctx context.Context) (string, error)
}
```

## Flow Diagrams

### HEARTBEAT.md Loading Flow

```
┌─────────────┐
│  Load()     │
└──────┬──────┘
       │
       ▼
┌─────────────────────┐
│ Get Heartbeat Path  │
│ (~/.nexbot/         │
│  HEARTBEAT.md)      │
└──────┬──────────────┘
       │
       ▼
┌─────────────────────┐
│ Check File Exists?  │
└──────┬──────────────┘
       │
       ▼
    ┌──┴──┐
    │ No  │ Yes
    ▼      ▼
┌─────────────┐  ┌─────────────────────┐
│ Return nil  │  │ Read File Content   │
│ (no tasks) │  └──────┬──────────────┘
└─────────────┘         │
                        ▼
                ┌─────────────────────┐
                │ Parse Content       │
                │ (parser.Parse)      │
                └──────┬──────────────┘
                       │
                       ▼
                ┌─────────────────────┐
                │ Validate Tasks      │
                │ (Validate func)     │
                └──────┬──────────────┘
                       │
                       ▼
                ┌─────────────────────┐
                │ Filter Valid Tasks   │
                │ (skip invalid)       │
                └──────┬──────────────┘
                       │
                       ▼
                ┌─────────────────────┐
                │ Store in Loader     │
                │ (tasks, isLoaded)   │
                └──────┬──────────────┘
                       │
                       ▼
                ┌─────────────────────┐
                │ Return Valid Tasks  │
                └─────────────────────┘
```

### HEARTBEAT.md Parsing Flow

```
┌─────────────────────┐
│ Parse(content)      │
└──────┬──────────────┘
       │
       ▼
┌─────────────────────┐
│ Split by `##`       │
│ (sections)          │
└──────┬──────────────┘
       │
       ▼
┌─────────────────────┐
│ For Each Section:   │
│ Split by `###`      │
│ (taskSections)      │
└──────┬──────────────┘
       │
       ▼
┌─────────────────────┐
│ For Each Task:      │
│ parseTaskSection()  │
└──────┬──────────────┘
       │
       ▼
┌─────────────────────┐
│ Extract Name from   │
│ `### Header`        │
└──────┬──────────────┘
       │
       ▼
┌─────────────────────┐
│ Extract Schedule    │
│ from `- Schedule:`  │
└──────┬──────────────┘
       │
       ▼
┌─────────────────────┐
│ Extract Task        │
│ from `- Task:`      │
└──────┬──────────────┘
       │
       ▼
┌─────────────────────┐
│ Return Task List    │
└─────────────────────┘
```

### Task Validation Flow

```
┌─────────────────────┐
│ Validate(task)      │
└──────┬──────────────┘
       │
       ▼
┌─────────────────────┐
│ Check Name != ""    │
└──────┬──────────────┘
       │
       ▼
    ┌──┴──┐
    │ No  │ Yes
    ▼      ▼
┌─────────────┐  ┌─────────────────────┐
│ Return Error │  │ Check Schedule!="" │
└─────────────┘  └──────┬──────────────┘
                         │
                         ▼
                      ┌──┴──┐
                      │ No  │ Yes
                      ▼      ▼
              ┌─────────────┐  ┌─────────────────────┐
              │ Return Error │  │ Validate Cron Expr │
              └─────────────┘  └──────┬──────────────┘
                                       │
                                       ▼
                                    ┌──┴──┐
                                    │ No  │ Yes
                                    ▼      ▼
                            ┌─────────────┐  ┌─────────────────────┐
                            │ Return Error │  │ Check Task != ""    │
                            └─────────────┘  └──────┬──────────────┘
                                                     │
                                                     ▼
                                                  ┌──┴──┐
                                                  │ No  │ Yes
                                                  ▼      ▼
                                          ┌─────────────┐  ┌─────────────────────┐
                                          │ Return Error │  │ Return nil (valid)  │
                                          └─────────────┘  └─────────────────────┘
```

### Checker Start Flow

```
┌─────────────┐
│  Start()    │
└──────┬──────┘
       │
       ▼
┌─────────────────────┐
│ Lock Mutex          │
└──────┬──────────────┘
       │
       ▼
┌─────────────────────┐
│ Check Started?       │
└──────┬──────────────┘
       │
       ▼
    ┌──┴──┐
    │ Yes │ No
    ▼      ▼
┌─────────────┐  ┌─────────────────────┐
│ Return nil  │  │ Create Context      │
│ (already)   │  │ (Background)        │
└─────────────┘  └──────┬──────────────┘
                         │
                         ▼
                  ┌─────────────────────┐
                  │ Set Started = true  │
                  └──────┬──────────────┘
                         │
                         ▼
                  ┌─────────────────────┐
                  │ Log Started         │
                  └──────┬──────────────┘
                         │
                         ▼
                  ┌─────────────────────┐
                  │ go run()            │
                  └──────┬──────────────┘
                         │
                         ▼
                  ┌─────────────────────┐
                  │ Unlock Mutex        │
                  └──────┬──────────────┘
                         │
                         ▼
                  ┌─────────────────────┐
                  │ Return nil          │
                  └─────────────────────┘
```

### Heartbeat Check Flow

```
┌─────────────────────┐
│ check()              │
└──────┬──────────────┘
       │
       ▼
┌─────────────────────┐
│ Log "Performing     │
│ heartbeat check"    │
└──────┬──────────────┘
       │
       ▼
┌─────────────────────┐
│ Agent.Process       │
│ HeartbeatCheck(ctx) │
└──────┬──────────────┘
       │
       ▼
    ┌──┴──┐
    │ OK  │ Error
    ▼      ▼
┌─────────────────┐  ┌─────────────────────┐
│ processResponse │  │ Log Error           │
│ (response)      │  └─────────────────────┘
└──────┬──────────┘
       │
       ▼
┌─────────────────────┐
│ Return              │
└─────────────────────┘
```

### Response Processing Flow

```
┌─────────────────────┐
│ processResponse(     │
│ response)            │
└──────┬──────────────┘
       │
       ▼
┌─────────────────────┐
│ Check Empty?         │
└──────┬──────────────┘
       │
       ▼
    ┌──┴──┐
    │ Yes │ No
    ▼      ▼
┌─────────────┐  ┌─────────────────────┐
│ Log Warn    │  │ Check Contains       │
│ (empty)     │  │ HEARTBEAT_OK?        │
└─────────────┘  └──────┬──────────────┘
                         │
                         ▼
                      ┌──┴──┐
                      │ Yes │ No
                      ▼      ▼
              ┌─────────────────┐  ┌─────────────────────┐
              │ Log Info        │  │ Log Info            │
              │ "all good"      │  │ "action taken"     │
              └─────────────────┘  └─────────────────────┘
```

### Heartbeat Task Scheduling Flow

```
┌─────────────────────┐
│ HEARTBEAT.md        │
│ Loaded & Parsed     │
└──────┬──────────────┘
       │
       ▼
┌─────────────────────┐
│ Tasks Validated     │
└──────┬──────────────┘
       │
       ▼
┌─────────────────────┐
│ For Each Task:      │
│ Create Cron Job     │
└──────┬──────────────┘
       │
       ▼
┌─────────────────────┐
│ AddJob() to         │
│ Cron Scheduler      │
└──────┬──────────────┘
       │
       ▼
┌─────────────────────┐
│ Job Executes on     │
│ Schedule            │
└──────┬──────────────┘
       │
       ▼
┌─────────────────────┐
│ Execute Task via    │
│ Worker Pool         │
└──────┬──────────────┘
       │
       ▼
┌─────────────────────┐
│ Agent Processes     │
│ Heartbeat Check     │
└──────┬──────────────┘
       │
       ▼
┌─────────────────────┐
│ Response Processed  │
└─────────────────────┘
```

## Lifecycle

### Loader Lifecycle

```
[Created]
    │
    │ NewLoader(workspace, logger)
    ▼
[Idle] ──Load()──> [Loading] ──success──> [Loaded]
    │                         │
    │                         └──error──> [Error]
    │
    │ GetTasks()/GetContext()
    ▼
[Tasks Returned]
```

### Checker Lifecycle

```
[Created]
    │
    │ NewChecker(interval, agent, logger)
    ▼
[Idle] ──Start()──> [Running] ──stop──> [Stopped]
    │              │
    │              └──ticker──> [Checking] ──complete──┐
    │                                                   │
    └───────────────────────────────────────────────────┘
```

### Task Lifecycle

```
[Defined in HEARTBEAT.md]
    │
    │ Load() → Parse()
    ▼
[Parsed] ──Validate()──> [Valid] ──AddJob()──> [Scheduled]
    │                         │
    │                         └──invalid──> [Skipped]
    │
    │ Schedule Triggered
    ▼
[Executing] ──complete──> [Completed]
    │
    └──error──> [Failed]
```

## Configuration

### HEARTBEAT.md Format

```markdown
# Heartbeat Tasks

## Periodic Reviews

### Daily Standup
- Schedule: "0 9 * * *"
- Task: "Review daily progress, check for blocked tasks, update priorities"

### Weekly Summary
- Schedule: "0 17 * * 5"
- Task: "Review weekly achievements, plan next week"
```

### Checker Configuration

```go
NewChecker(intervalMinutes int, agent Agent, logger *logger.Logger)
```

**Parameters:**
- `intervalMinutes` — интервал проверки в минутах
- `agent` — агент, реализующий Agent interface
- `logger` — экземпляр логгера

### Workspace Path

```
~/.nexbot/HEARTBEAT.md
```

## Integration Points

### With Cron Scheduler

1. Loader загружает задачи из HEARTBEAT.md
2. Каждая задача добавляется в Cron Scheduler как Job
3. Scheduler планирует выполнение по расписанию
4. При срабатывании schedule задача отправляется в Worker Pool

### With Agent

Checker вызывает `Agent.ProcessHeartbeatCheck(ctx)`:
- Agent должен прочитать HEARTBEAT.md
- Agent следует задачам из HEARTBEAT.md
- Agent использует инструменты (tools) для выполнения действий
- Agent возвращает ответ (HEARTBEAT_OK или описание действий)

### With Worker Pool

Heartbeat задачи выполняются через Worker Pool:
- Задача типа "cron" с payload из HEARTBEAT
- Результат выполнения через result channel
- Метрики выполнения в PoolMetrics

## Error Handling

### Loader Errors

- **File not found** — возвращает nil, tasks = nil (не ошибка)
- **Parse error** — логируется, задача пропускается
- **Validation error** — задача пропускается с предупреждением

### Checker Errors

- **Agent error** — логируется в `Error()` с response
- **Empty response** — логируется предупреждение
- **Context cancellation** — checker останавливается gracefully

## Best Practices

1. **HEARTBEAT.md location** — храните в корне workspace
2. **Task names** — используйте описательные имена (Daily Standup, Weekly Summary)
3. **Schedules** — используйте валидные cron выражения
4. **Task descriptions** — четкие и конкретные инструкции
5. **Checker interval** — настройте оптимальный интервал (обычно 30-60 мин)

## Constants

```go
const heartbeatPrompt = "Read HEARTBEAT.md from workspace. Follow it strictly. Do not infer or repeat old tasks from prior chats. If nothing needs attention, reply HEARTBEAT_OK."

const heartbeatOKToken = "HEARTBEAT_OK"
```

## Context Formatting

`GetContext()` возвращает строку в формате:
```
Active heartbeat tasks: N (daily standup, weekly summary)
```

Если задач нет:
```
No active heartbeat tasks
```

## Future Enhancements

1. **Priority levels** — поддержка приоритетов для задач
2. **Task history** — история выполнения задач
3. **Task templates** — шаблоны для повторяющихся задач
4. **Notifications** — уведомления при важных событиях
5. **Task dependencies** — зависимости между задачами
