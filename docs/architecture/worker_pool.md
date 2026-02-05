# Worker Pool Architecture

## Overview

Worker Pool — это асинхронный пул воркеров для фоновой обработки задач. Он поддерживает несколько типов задач (cron, subagent) и предоставляет каналы результатов для мониторинга выполнения задач. Пул обеспечивает конкурентное выполнение с контролируемым количеством воркеров.

## Components

### 1. WorkerPool (`internal/workers/pool.go`)

**Responsibilities:**
- Управление пулом goroutine воркеров
- Очередь задач с буферизацией
- Обработка задач разных типов
- Сбор метрик выполнения
- Graceful shutdown

**Key Fields:**
```go
type WorkerPool struct {
    taskQueue chan Task           // Bufferred queue for tasks
    resultCh  chan Result         // Channel for task results
    workers   int                 // Number of workers
    wg        *taskWaitGroup      // WaitGroup for shutdown
    ctx       context.Context     // Pool context
    cancel    context.CancelFunc  // Cancel function
    logger    *logger.Logger      // Logger instance
    metrics   *PoolMetrics        // Execution metrics
}
```

**Key Methods:**
- `Start()` — запуск всех воркеров
- `Submit(task)` — отправка задачи в очередь
- `SubmitWithContext(ctx, task)` — отправка с таймаутом
- `Results()` — канал для получения результатов
- `Stop()` — graceful shutdown
- `Metrics()` — текущие метрики

### 2. Task (`internal/workers/pool.go`)

**Responsibilities:**
- Представляет единицу работы
- Содержит контекст для отмены

**Structure:**
```go
type Task struct {
    ID      string                 // Unique task identifier
    Type    string                 // Task type: "cron" or "subagent"
    Payload interface{}            // Task payload (command, agent config, etc.)
    Context context.Context        // Task-specific context for cancellation/timeout
    Metrics map[string]interface{} // Optional metrics to track
}
```

### 3. Result (`internal/workers/pool.go`)

**Responsibilities:**
- Представляет результат выполнения задачи
- Содержит информацию о времени выполнения и ошибках

**Structure:**
```go
type Result struct {
    TaskID   string                 // ID of the executed task
    Error    error                  // Error if execution failed
    Output   string                 // Task output
    Duration time.Duration          // Execution duration
    Metrics  map[string]interface{} // Task execution metrics
}
```

### 4. PoolMetrics (`internal/workers/pool.go`)

**Responsibilities:**
- Отслеживание метрик выполнения пула

**Structure:**
```go
type PoolMetrics struct {
    TasksSubmitted uint64         // Total tasks submitted
    TasksCompleted uint64         // Successfully completed tasks
    TasksFailed    uint64         // Failed tasks
    TotalDuration  time.Duration  // Total execution time
}
```

### 5. taskWaitGroup (`internal/workers/pool.go`)

**Responsibilities:**
- Обертка вокруг `sync.WaitGroup` с thread-safe доступом к метрикам

**Methods:**
- `Add(delta)`, `Done()`, `Wait()` — делегирует sync.WaitGroup
- `Lock()`, `Unlock()`, `RLock()`, `RUnlock()` — для безопасного доступа к метрикам

## Flow Diagrams

### Worker Pool Start Flow

```
┌─────────────┐
│  Start()    │
└──────┬──────┘
       │
       ▼
┌─────────────────────┐
│ Create Context      │
│ (Background)        │
└──────┬──────────────┘
       │
       ▼
┌─────────────────────┐
│ Log Start Info      │
│ (workers, buffer)   │
└──────┬──────────────┘
       │
       ▼
┌─────────────────────┐
│ For each worker     │
│ (i < workers):      │
│   wg.Add(1)         │
│   go worker(i)      │
└──────┬──────────────┘
       │
       ▼
┌─────────────────────┐
│ Workers Running     │
└─────────────────────┘
```

### Task Submission Flow

```
┌─────────────┐
│  Submit()   │
└──────┬──────┘
       │
       ▼
┌─────────────────────┐
│ Lock WaitGroup      │
└──────┬──────────────┘
       │
       ▼
┌─────────────────────┐
│ Increment           │
│ TasksSubmitted      │
└──────┬──────────────┘
       │
       ▼
┌─────────────────────┐
│ Unlock WaitGroup    │
└──────┬──────────────┘
       │
       ▼
┌─────────────────────┐
│ Log Submit Debug    │
└──────┬──────────────┘
       │
       ▼
┌─────────────────────┐
│ Send to taskQueue    │
│ (blocks if full)     │
└─────────────────────┘
```

### Task Execution Flow

```
┌─────────────────────┐
│ Worker Goroutine    │
│ (waiting for task)  │
└──────┬──────────────┘
       │
       ▼
┌─────────────────────┐
│ Receive from        │
│ taskQueue           │
└──────┬──────────────┘
       │
       ▼
┌─────────────────────┐
│ processTask(task)   │
└──────┬──────────────┘
       │
       ▼
┌─────────────────────┐
│ Record Start Time   │
└──────┬──────────────┘
       │
       ▼
┌─────────────────────┐
│ Determine Context   │
│ (task or pool)      │
└──────┬──────────────┘
       │
       ▼
┌─────────────────────┐
│ executeTask(ctx,    │
│ task)               │
└──────┬──────────────┘
       │
       ▼
┌─────────────────────┐
│ Calculate Duration  │
└──────┬──────────────┘
       │
       ▼
┌─────────────────────┐
│ Update Metrics      │
│ (Completed/Failed)  │
└──────┬──────────────┘
       │
       ▼
┌─────────────────────┐
│ Send to resultCh    │
└──────┬──────────────┘
       │
       ▼
┌─────────────────────┐
│ Loop back to wait   │
│ for next task       │
└─────────────────────┘
```

### Task Dispatch Flow

```
┌─────────────────────┐
│ executeTask()       │
└──────┬──────────────┘
       │
       ▼
┌─────────────────────┐
│ Check Context       │
│ Cancelled?          │
└──────┬──────────────┘
       │
       ▼
    ┌──┴──┐
    │ No  │ Yes
    ▼      ▼
┌─────────────┐  ┌─────────────────────┐
│ Switch Type │  │ Return Error:       │
│             │  │ Context Cancelled   │
└──┬──────┬───┘  └─────────────────────┘
   │      │
   │      ├─── "cron" ────► executeCronTask()
   │      │
   │      ├─── "subagent" ─► executeSubagentTask()
   │      │
   │      └─── default ───► Return Error: Unknown Type
   │
   ▼
┌─────────────────────┐
│ Return Result       │
└─────────────────────┘
```

### Worker Pool Shutdown Flow

```
┌─────────────┐
│  Stop()     │
└──────┬──────┘
       │
       ▼
┌─────────────────────┐
│ Cancel Context      │
└──────┬──────────────┘
       │
       ▼
┌─────────────────────┐
│ Wait for Workers    │
│ (wg.Wait())         │
└──────┬──────────────┘
       │
       ▼
┌─────────────────────┐
│ Read Metrics        │
│ (with lock)         │
└──────┬──────────────┘
       │
       ▼
┌─────────────────────┐
│ Log Stop Info       │
│ (submitted, etc.)   │
└──────┬──────────────┘
       │
       ▼
┌─────────────────────┐
│ Close taskQueue     │
│ (if not closed)     │
└──────┬──────────────┘
       │
       ▼
┌─────────────────────┐
│ Close resultCh      │
│ (if not closed)     │
└──────┬──────────────┘
       │
       ▼
┌─────────────────────┐
│ Log Stopped         │
└─────────────────────┘
```

## Task Execution Lifecycle

```
[Submitted]
    │
    │ Submit(task)
    ▼
[Queued] ──(worker picks up)──> [Executing] ──┐
    │                                        │
    │                                        ├─── success ──► [Completed]
    │                                        │
    │                                        └─── failure ──► [Failed]
    │
    │ Stop()
    ▼
[Pool Stopping] ──(workers finish)──> [Stopped]
```

## Configuration

### Pool Configuration

```go
func NewPool(workers int, bufferSize int, logger *logger.Logger) *WorkerPool
```

**Parameters:**
- `workers` — количество воркеров (goroutine)
- `bufferSize` — размер буфера очереди задач
- `logger` — экземпляр логгера

### Task Types

1. **cron** — периодические задачи от Cron Scheduler
2. **subagent** — задачи для выполнения в subagent

### Context Handling

- Task context имеет приоритет над pool context
- Если task context не указан, используется pool context
- Context cancellation прерывает выполнение задачи

## Error Handling

### Worker Panic Recovery

Каждый воркер обернут в `recover()`:
```go
defer func() {
    if r := recover(); r != nil {
        p.logger.Error("worker panic recovered", ...)
    }
}()
```

### Task Execution Errors

- Ошибки возвращаются в `Result.Error`
- Метрики обновляются (TasksFailed)
- Задача не блокирует пул

### Channel Errors

- Если resultCh закрыт, результат не отправляется
- Если taskQueue закрыт, Submit блокируется навсегда

## Metrics

### Available Metrics

```go
type PoolMetrics struct {
    TasksSubmitted uint64        // Общее количество отправленных задач
    TasksCompleted uint64        // Успешно выполненные задачи
    TasksFailed    uint64        // Задачи с ошибками
    TotalDuration  time.Duration // Общее время выполнения
}
```

### Additional Info

- `WorkerCount()` — количество активных воркеров
- `QueueSize()` — текущий размер очереди

## Concurrency Model

### Worker Goroutines

- Каждый воркер — отдельная goroutine
- Все воркеры работают параллельно
- Число воркеров фиксировано при создании

### Task Queue

- Буферизированный канал (`make(chan Task, bufferSize)`)
- Блокировка при полной очереди
- FIFO порядок обработки

### Result Channel

- Буферизированный канал (`make(chan Result, bufferSize)`)
- Non-blocking отправка (с проверкой ctx.Done())
- Потребитель должен читать из канала

## Integration Points

1. **Cron Scheduler** — отправляет cron задачи через `Submit(task)`
2. **Subagent Manager** — может отправлять subagent задачи (TODO: интеграция)
3. **Logger** — структурированное логирование всех операций
4. **Context** — управление жизненным циклом и отменой

## Best Practices

1. **Выбор размера пула:** основывайте на CPU cores и типе задач
2. **Размер буфера:** баланс между памятью и блокировкой
3. **Обработка результатов:** всегда читайте из `Results()` канала
4. **Graceful shutdown:** вызывайте `Stop()` для завершения
5. **Context cancellation:** используйте task context для таймаутов

## Limitations

1. Нет приоритизации задач
2. Нет динамического масштабирования
3. Фиксированное количество воркеров
4. ExecuteCronTask и ExecuteSubagentTask — mock реализации
5. Нет rate limiting или throttling

## Future Enhancements

1. **Priority Queue** — поддержка приоритетов задач
2. **Dynamic Scaling** — автоматическое изменение количества воркеров
3. **Task Retry** — повторное выполнение при ошибках
4. **Task Timeout** — глобальные таймауты для задач
5. **Metrics Export** — экспорт метрик в Prometheus/StatsD
