# Cron Scheduler

## Назначение

Cron Scheduler обеспечивает планирование периодических задач с использованием библиотеки robfig/cron/v3. Задачи могут быть recurring или oneshot.

## Основные компоненты

### Scheduler
Планировщик cron с функциями:
- `AddJob` — добавление задачи
- `RemoveJob` — удаление задачи
- `ListJobs` — список всех задач
- `GetJob` — получение задачи по ID
- `IsStarted` — проверка статуса запуска

### Job
Единица работы:
- `ID` — уникальный ID
- `Type` — тип (recurring, oneshot)
- `Schedule` — cron выражение
- `ExecuteAt` — время выполнения (для oneshot)
- `Command` — команда для выполнения
- `UserID` — пользователь
- `Metadata` — метаданные

### JobType
Типы задач:
- `JobTypeRecurring` — периодические задачи
- `JobTypeOneshot` — разовые задачи

### Storage
Постоянное хранение задач:
- `UpsertJob` — создание или обновление
- `Remove` — удаление
- `GetJobs` — получение всех задач

## Использование

### Создание scheduler

```go
import (
    "github.com/aatumaykin/nexbot/internal/cron"
    "github.com/aatumaykin/nexbot/internal/bus"
    "github.com/aatumaykin/nexbot/internal/logger"
    "github.com/aatumaykin/nexbot/internal/workers"
)

func main() {
    log, _ := logger.New(logger.Config{
        Level:  "info",
        Format: "json",
        Output: "stdout",
    })

    // Создание worker pool
    pool := workers.NewPool(5, 100, log)

    // Создание scheduler
    scheduler := cron.NewScheduler(log, messageBus, pool, storage)
}
```

### Запуск scheduler

```go
// Запуск планировщика
err = scheduler.Start(ctx)
if err != nil {
    log.Fatal(err)
}
```

### Добавление recurring задачи

```go
// Ежедневная задача в 10:00
job := cron.Job{
    ID:      "daily-backup",
    Type:    cron.JobTypeRecurring,
    Schedule: "0 10 * * * *",
    Command:  "backup database",
    UserID:  "admin",
}

id, err := scheduler.AddJob(job)
```

### Добавление oneshot задачи

```go
// Выполнение через 5 минут
executeAt := time.Now().Add(5 * time.Minute)
job := cron.Job{
    ID:       "report-1",
    Type:     cron.JobTypeOneshot,
    ExecuteAt: &executeAt,
    Command:  "send weekly report",
}

id, err := scheduler.AddJob(job)
```

### Удаление задачи

```go
// Удаление задачи
err = scheduler.RemoveJob("daily-backup")
```

### Список задач

```go
// Список всех задач
jobs := scheduler.List()

// Получение конкретной задачи
job, err := scheduler.GetJob("daily-backup")
```

## Конфигурация

### Scheduler

- `logger` — логгер (обязательно)
- `messageBus` — message bus (обязательно)
- `workerPool` — worker pool для выполнения (обязательно)
- `storage` — хранилище задач (обязательно)

## Зависимости

- `github.com/robfig/cron/v3` — планировщик cron
- `github.com/aatumaykin/nexbot/internal/bus` — message bus
- `github.com/aatumaykin/nexbot/internal/logger` — логирование
- `github.com/aatumaykin/nexbot/internal/workers` — worker pool

## Примечания

- Recurring задачи добавляются в cron scheduler
- Oneshot задачи валидируются, но не добавляются в scheduler
- Oneshot задачи выполняются сразу при добавлении (если время прошло)
- Тasks отправляются в message bus как inbound
- Cron формат: `second minute hour dom month dow`

## См. также

- `internal/workers` — execution worker pool
- `internal/bus` — message bus
- `internal/heartbeat` — periodic heartbeat check
