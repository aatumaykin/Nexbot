# Cron Scheduler Architecture

## Overview

Cron Scheduler — это компонент для планирования и выполнения периодических задач с использованием cron выражений. Он интегрируется с Worker Pool для асинхронного выполнения задач и обеспечивает персистентное хранение задач в формате JSONL.

## Components

### 1. Scheduler (`internal/cron/scheduler.go`)

**Responsibilities:**
- Управление жизненным циклом cron job'ов
- Планирование задач по расписанию (recurring) или по времени (oneshot)
- Интеграция с Worker Pool для асинхронного выполнения
- Обработка ошибок и логирование

**Key Fields:**
```go
type Scheduler struct {
    cron          *cron.Cron           // robfig/cron scheduler
    logger        *logger.Logger       // Logger instance
    bus           *bus.MessageBus      // Message bus for fallback
    workerPool    WorkerPool          // Worker pool for task execution
    storage       *Storage            // Persistent storage
    jobs          map[string]Job      // Job registry
    jobIDs        map[cron.EntryID]string   // cron.EntryID -> Job.ID mapping
    jobEntryIDs   map[string]cron.EntryID   // Job.ID -> cron.EntryID mapping
}
```

**Key Methods:**
- `Start()` — запуск планировщика
- `AddJob(job Job)` — добавление новой задачи
- `RemoveJob(jobID)` — удаление задачи
- `ListJobs()` — список всех задач
- `GetJob(jobID)` — получение задачи по ID
- `executeJob(job)` — выполнение задачи через Worker Pool

### 2. Storage (`internal/cron/storage.go`)

**Responsibilities:**
- Персистентное хранение cron job'ов в формате JSONL
- Атомарные операции записи
- Управление жизненным циклом oneshot задач

**Key Methods:**
- `Load()` — загрузка всех задач
- `Append(job)` — добавление новой задачи
- `Remove(jobID)` — удаление задачи
- `Save(jobs)` — сохранение списка задач (atomic write)
- `RemoveExecutedOneshots()` — очистка выполненных oneshot задач

**Storage Format:**
```json
{"id":"job_123","type":"recurring","schedule":"0 9 * * *","command":"check_status","user_id":"user1","metadata":{},"executed":false,"executed_at":null}
```

### 3. Worker Pool Integration

**Flow:**
1. Scheduler создает Task из Job
2. Task отправляется в Worker Pool
3. Worker Pool возвращает Result через канал результатов

**Task Structure:**
```go
type Task struct {
    ID      string      // Unique task identifier
    Type    string      // "cron"
    Payload interface{} // CronTaskPayload
    Context context.Context
}
```

**Payload:**
```go
type CronTaskPayload struct {
    Command  string            // Command to execute
    UserID   string            // User ID
    Metadata map[string]string // Job metadata
}
```

## Flow Diagrams

### Job Addition Flow

```
┌─────────────┐
│   AddJob()  │
└──────┬──────┘
       │
       ▼
┌─────────────────┐
│ Generate Job ID │
└──────┬──────────┘
       │
       ▼
┌─────────────────────┐
│ Validate Schedule   │
│ (robfig/cron)       │
└──────┬──────────────┘
       │
       ▼
┌─────────────────┐
│ Add to cron     │
│ (cron.AddFunc)  │
└──────┬──────────┘
       │
       ▼
┌─────────────────────┐
│ Store in Registry   │
│ (jobs, jobIDs,      │
│  jobEntryIDs)       │
└──────┬──────────────┘
       │
       ▼
┌─────────────────────┐
│ Append to Storage   │
│ (JSONL)             │
└─────────────────────┘
```

### Job Execution Flow

```
┌─────────────────────┐
│ Cron Triggered      │
│ (schedule matched)  │
└──────┬──────────────┘
       │
       ▼
┌─────────────────────┐
│ executeJob(job)     │
└──────┬──────────────┘
       │
       ▼
┌─────────────────────┐
│ Create Task Payload │
│ (Command, UserID,   │
│  Metadata)          │
└──────┬──────────────┘
       │
       ▼
┌─────────────────────┐
│ Create Task ID      │
│ (cron_<job>_<ts>)   │
└──────┬──────────────┘
       │
       ▼
┌─────────────────────┐
│ Submit to Worker    │
│ Pool                │
└──────┬──────────────┘
       │
       ▼
┌─────────────────────┐
│ Worker Executes     │
│ Task                │
└──────┬──────────────┘
       │
       ▼
┌─────────────────────┐
│ Result sent to      │
│ result channel      │
└─────────────────────┘
```

### Oneshot Job Flow

```
┌─────────────────────┐
│ Ticker (1 min)      │
│ Triggered           │
└──────┬──────────────┘
       │
       ▼
┌─────────────────────┐
│ Load Jobs from      │
│ Storage             │
└──────┬──────────────┘
       │
       ▼
┌─────────────────────┐
│ Filter Oneshot Jobs │
└──────┬──────────────┘
       │
       ▼
┌─────────────────────┐
│ Check ExecuteAt     │
│ <= Now & !Executed  │
└──────┬──────────────┘
       │
       ▼
┌─────────────────────┐
│ Mark Executed       │
│ (true, ExecutedAt)  │
└──────┬──────────────┘
       │
       ▼
┌─────────────────────┐
│ Execute Job         │
│ (via Worker Pool)   │
└──────┬──────────────┘
       │
       ▼
┌─────────────────────┐
│ Save Updated Jobs   │
│ (Storage.Save)      │
└─────────────────────┘
```

### Job Cleanup Flow

```
┌─────────────────────┐
│ Ticker (24h)        │
│ Triggered           │
└──────┬──────────────┘
       │
       ▼
┌─────────────────────┐
│ RemoveExecuted     │
│ Oneshots()          │
└──────┬──────────────┘
       │
       ▼
┌─────────────────────┐
│ Load All Jobs       │
└──────┬──────────────┘
       │
       ▼
┌─────────────────────┐
│ Filter: Keep        │
│ - Recurring         │
│ - Oneshot !Executed │
└──────┬──────────────┘
       │
       ▼
┌─────────────────────┐
│ Save Filtered Jobs  │
│ (Atomic write)      │
└──────┬──────────────┘
       │
       ▼
┌─────────────────────┐
│ Reload In-Memory    │
│ Jobs Map            │
└─────────────────────┘
```

## Job Lifecycle

### Recurring Job

```
[Created]
    │
    │ AddJob(job)
    ▼
[Scheduled] ──(cron trigger)──> [Executing] ──(worker completes)──> [Completed]
    │
    │ RemoveJob(jobID)
    ▼
[Removed]
```

### Oneshot Job

```
[Created]
    │
    │ AddJob(job)
    ▼
[Scheduled] ──(ticker check)──> [Overdue?] ──yes──> [Executing] ──(worker completes)──> [Executed]
    │                                           │
    │                                          no│
    │                                           │
    │                         (cleanup ticker 24h)│
    ▼                                           ▼
[Removed]                                  [Cleaned Up]
```

## Configuration

### Schedule Format

Supports standard cron expressions with seconds:
```
* * * * * *  (sec min hour dom month dow)

Examples:
- "0 9 * * *"      — Every day at 9:00 AM
- "0 */6 * * *"    — Every 6 hours
- "0 0 1 * *"      — First day of every month
```

### Job Types

1. **Recurring** — повторяющиеся задачи по расписанию
2. **Oneshot** — однократные задачи в указанное время

### Storage Path

```
~/.nexbot/cron/jobs.jsonl
```

## Error Handling

- **Invalid cron expression** — возвращается ошибка при AddJob
- **Storage errors** — логируются, но не останавливают планировщик
- **Worker pool errors** — логируются в Result.Error
- **Panic recovery** — executeJob обернут в recover()

## Integration Points

1. **Message Bus** — fallback если Worker Pool недоступен
2. **Worker Pool** — основной канал выполнения задач
3. **Storage** — персистентное хранение в JSONL формате
4. **Logger** — структурированное логирование всех операций

## Metrics

Текущая версия не собирает метрики выполнения, но Worker Pool предоставляет:
- TasksSubmitted
- TasksCompleted
- TasksFailed
- TotalDuration
