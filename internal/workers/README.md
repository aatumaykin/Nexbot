# Worker Pool

## Назначение

Worker Pool управляет пулом goroutine workers для асинхронного выполнения задач. Поддерживает multiple task types (cron, subagent) и предоставляет каналы результатов для мониторинга.

## Основные компоненты

### WorkerPool
Пул workers с очередью задач:
- `Submit` — отправка задачи
- `SubmitCronTask` — отправка cron задачи
- `SubmitWithContext` — отправка с timeout
- `Results` — канал результатов
- `Stop` — graceful shutdown
- `Metrics` — метрики пула

### Task
Единица работы для выполнения.

### CronTask
Задача для cron планировщика.

### Result
Результат выполнения задачи.

### PoolMetrics
Метрики пула:
- `TasksSubmitted` — количество отправленных задач
- `TasksCompleted` — количество завершенных задач
- `TasksFailed` — количество_failed задач

## Использование

### Создание пула

```go
import (
    "github.com/aatumaykin/nexbot/internal/workers"
    "github.com/aatumaykin/nexbot/internal/logger"
)

func main() {
    log, _ := logger.New(logger.Config{
        Level:  "info",
        Format: "json",
        Output: "stdout",
    })

    // Создание пула (5 workers, буфер 100)
    pool := workers.NewPool(5, 100, log)
    pool.Start()
}
```

### Отправка задачи

```go
// Базовая задача
task := workers.Task{
    ID:      "task-1",
    Type:    "test",
    Payload: "test data",
    Context: ctx,
}

pool.Submit(task)

// Cron задача
cronTask := workers.CronTask{
    ID:      "cron-1",
    Type:    "cron",
    Payload: "execute command",
    Context: ctx,
}
pool.SubmitCronTask(cronTask)
```

### Получение результатов

```go
// Результаты задач
for result := range pool.Results() {
    if result.Error != nil {
        log.Error("Task failed", result.Error,
            logger.Field{Key: "task_id", Value: result.Task.ID})
    } else {
        log.Info("Task completed", result.Output,
            logger.Field{Key: "task_id", Value: result.Task.ID})
    }
}
```

### Метрики пула

```go
// Получение метрик
metrics := pool.Metrics()
log.Info("Pool metrics",
    logger.Field{Key: "submitted", Value: metrics.TasksSubmitted},
    logger.Field{Key: "completed", Value: metrics.TasksCompleted},
    logger.Field{Key: "failed", Value: metrics.TasksFailed})

// Размер очереди
queueSize := pool.QueueSize()

// Количество workers
workerCount := pool.WorkerCount()
```

### Graceful shutdown

```go
// Остановка пула
pool.Stop()

// Ожидание завершения задач
for len(pool.Results()) > 0 {
    time.Sleep(100 * time.Millisecond)
}
```

## Конфигурация

### Параметры NewPool

- `workers` — количество workers (рекомендуется: 5)
- `bufferSize` — размер очереди (рекомендуется: 100)
- `logger` — логгер

## Зависимости

- `github.com/aatumaykin/nexbot/internal/logger` — логирование
- `sync` — конкурентное выполнение

## Примечания

- Tasks автоматически завершаются при Stop()
- Workers запускаются пулом goroutines
- Queue размер — длина канала taskQueue
- Results канал закрывается после Stop()
- Таски выполняются в порядке поступления

## См. также

- `internal/cron` — использует worker pool для cron задач
- `internal/agent/subagent` — использует worker pool для subagent задач
