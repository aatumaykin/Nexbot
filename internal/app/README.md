# Application

## Назначение

Application представляет собой главный контейнер приложения Nexbot. Координирует все компоненты: agent loop, message bus, channels, cron scheduler, heartbeat checker и command handlers.

## Основные компоненты

### App
Главное приложение с управлением жизненным циклом:
- `Initialize` — инициализация всех компонентов
- `Run` — запуск приложения
- `StartMessageProcessing` — запуск обработки сообщений
- `Shutdown` — graceful shutdown
- `Restart` — перезапуск приложения

### Структура компонентов

```
App
├── Config
├── Logger
├── MessageBus
├── AgentLoop
├── CommandHandler
├── TelegramConnector
├── CronScheduler
├── WorkerPool
├── HeartbeatChecker
└── IPCHandler
```

## Использование

### Создание приложения

```go
import (
    "github.com/aatumaykin/nexbot/internal/app"
    "github.com/aatumaykin/nexbot/internal/config"
    "github.com/aatumaykin/nexbot/internal/logger"
)

func main() {
    // Загрузка конфигурации
    cfg, err := config.Load("config.toml")
    if err != nil {
        log.Fatal(err)
    }

    // Создание логгера
    log, err := logger.New(logger.Config{
        Level:  cfg.Logging.Level,
        Format: cfg.Logging.Format,
        Output: cfg.Logging.Output,
    })
    if err != nil {
        log.Fatal(err)
    }

    // Создание приложения
    appInstance := app.New(cfg, log)
}
```

### Запуск приложения

```go
// Запуск и блокировка до отмены контекста
err := appInstance.Run(ctx)
if err != nil {
    log.Error("Application failed", err)
}
```

### Инициализация компонентов

```go
// Инициализация всех компонентов
err = appInstance.Initialize(ctx)
if err != nil {
    log.Fatal(err)
}
```

### Запуск обработки сообщений

```go
// Запуск обработки сообщений
err = appInstance.StartMessageProcessing(ctx)
if err != nil {
    log.Fatal(err)
}
```

### Graceful shutdown

```go
// Остановка всех компонентов
err = appInstance.Shutdown()
if err != nil {
    log.Error("Shutdown failed", err)
}
```

### Перезапуск

```go
// Перезапуск приложения
err = appInstance.Restart()
if err != nil {
    log.Error("Restart failed", err)
}
```

## Конфигурация

### Параметры App

- `config` — конфигурация приложения (обязательно)
- `logger` — логгер (обязательно)

## Зависимости

- `github.com/aatumaykin/nexbot/internal/config` — конфигурация
- `github.com/aatumaykin/nexbot/internal/logger` — логирование
- `github.com/aatumaykin/nexbot/internal/bus` — message bus
- `github.com/aatumaykin/nexbot/internal/agent/loop` — agent loop
- `github.com/aatumaykin/nexbot/internal/commands` — команды
- `github.com/aatumaykin/nexbot/internal/channels/telegram` — Telegram
- `github.com/aatumaykin/nexbot/internal/cron` — cron scheduler
- `github.com/aatumaykin/nexbot/internal/heartbeat` — heartbeat
- `github.com/aatumaykin/nexbot/internal/workers` — worker pool
- `github.com/aatumaykin/nexbot/internal/ipc` — IPC

## Примечания

- Инициализация происходит в Initialize()
- Запуск обработки сообщений в StartMessageProcessing()
- App управляет контекстом для всех компонентов
- Restart использует mutex для безопасности
- Все компоненты shutdown корректно в Shutdown()

## См. также

- `internal/agent/loop` — agent loop
- `internal/bus` — message bus
- `internal/channels/telegram` — Telegram
- `internal/cron` — cron scheduler
