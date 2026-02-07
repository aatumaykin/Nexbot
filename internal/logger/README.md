# Logger

## Назначение

Logger предоставляет структурированное логирование вокруг Go's slog. Поддерживает JSON и text форматы, multiple log levels и гибкие destinations.

## Основные компоненты

### Logger
Обёртка вокруг slog.Logger:
- `Debug` — debug логирование
- `Info` — info логирование
- `Warn` — warning логирование
- `Error` — error логирование
- `DebugCtx` — debug с контекстом
- `InfoCtx` — info с контекстом
- `WarnCtx` — warning с контекстом
- `ErrorCtx` — error с контекстом
- `With` — создание logger с дополнительными полями

### Field
Поле для структурированного логирования:
- `Key` — ключ поля
- `Value` — значение

### Config
Конфигурация логгера:
- `Level` — уровень логирования (debug, info, warn, error)
- `Format` — формат (json, text)
- `Output` — вывод (stdout, stderr, файл)

## Использование

### Создание logger

```go
import (
    "github.com/aatumaykin/nexbot/internal/logger"
)

func main() {
    log, err := logger.New(logger.Config{
        Level:  "info",
        Format: "json",
        Output: "stdout",
    })
    if err != nil {
        log.Fatal(err)
    }
}
```

### Базовое логирование

```go
// Debug
log.Debug("Debug message",
    logger.Field{Key: "key", Value: "value"})

// Info
log.Info("Application started",
    logger.Field{Key: "version", Value: "1.0.0"})

// Warn
log.Warn("Potential issue detected",
    logger.Field{Key: "count", Value: 5})

// Error
log.Error("Operation failed", err,
    logger.Field{Key: "operation", Value: "backup"})
```

### Логирование с контекстом

```go
ctx := context.Background()

log.DebugCtx(ctx, "Processing message",
    logger.Field{Key: "session_id", Value: "123"},
    logger.Field{Key: "user_id", Value: "user1"})

log.InfoCtx(ctx, "Task completed successfully",
    logger.Field{Key: "duration", Value: "1.2s"})

log.ErrorCtx(ctx, "Task failed", err,
    logger.Field{Key: "task_id", Value: "task-1"})
```

### Логирование с полями

```go
// Создание logger с дополнительными полями
userLogger := log.With(
    logger.Field{Key: "user_id", Value: "123"},
    logger.Field{Key: "session_id", Value: "session-456"},
)

userLogger.Info("User action",
    logger.Field{Key: "action", Value: "login"})
```

### Файл логирование

```go
log, err := logger.New(logger.Config{
    Level:  "info",
    Format: "json",
    Output: "/var/log/nexbot.log",
})
```

## Конфигурация

### Config

- `Level` — уровень логирования (debug, info, warn, error)
- `Format` — формат (json, text)
- `Output` — stdout, stderr или путь к файлу

## Зависимости

- `log/slog` — Go structured logging
- `sync` — конкурентное логирование

## Примечания

- JSON формат удобен для parsing и aggregation
- Text формат удобен для чтения
- Все логи автоматически включают timestamp и level
- Error поля автоматически добавляются в ErrorCtx

## См. также

- `github.com/aatumaykin/nexbot/internal/logger` — реализация
