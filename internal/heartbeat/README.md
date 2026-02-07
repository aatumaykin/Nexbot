# Heartbeat

## Назначение

Heartbeat обеспечивает периодическую проверку статуса системы через агента. Проверяет HEARTBEAT.md и определяет, требуют ли задачи внимания.

## Основные компоненты

### Checker
Проверщик сердцебиения с функциями:
- `Start` — запуск цикла проверки
- `Stop` — остановка проверки
- `run` — основной цикл проверки
- `check` — выполнение одиночной проверки

### Agent
Интерфейс агента:
- `ProcessHeartbeatCheck(ctx)` — обработка проверки

## Использование

### Создание checker

```go
import (
    "github.com/aatumaykin/nexbot/internal/heartbeat"
    "github.com/aatumaykin/nexbot/internal/logger"
)

func main() {
    log, _ := logger.New(logger.Config{
        Level:  "info",
        Format: "json",
        Output: "stdout",
    })

    // Создание checker (проверка каждую 5 минут)
    checker := heartbeat.NewChecker(5, agent, log)
}
```

### Запуск проверки

```go
// Запуск
err = checker.Start()
if err != nil {
    log.Fatal(err)
}
```

### Остановка проверки

```go
// Остановка
err = checker.Stop()
if err != nil {
    log.Error("Failed to stop heartbeat", err)
}
```

## Интерфейс Agent

### Реализация ProcessHeartbeatCheck

```go
// Реализация интерфейса Agent
type MyAgent struct {
    // ...
}

func (a *MyAgent) ProcessHeartbeatCheck(ctx context.Context) (string, error) {
    // Чтение HEARTBEAT.md
    // Обработка и возвращение результата

    // HEARTBEAT_OK — всё хорошо
    return "HEARTBEAT_OK", nil

    // Или конкретная задача
    return "Проверьте storage, он заполнен на 90%", nil
}
```

## Конфигурация

### Checker

- `intervalMinutes` — интервал проверки в минутах (обязательно)
- `agent` — агент для проверки (обязательно)
- `logger` — логгер (обязательно)

## Зависимости

- `github.com/aatumaykin/nexbot/internal/logger` — логирование
- `context` — управление контекстом
- `sync` — конкурентное выполнение
- `time` — таймеры

## Примечания

- Проверка выполняется в отдельной goroutine
- Интервал 0 используется для тестирования (1 миллисекунда)
- При получении `HEARTBEAT_OK` — всё хорошо
- Иначе — LLM уже отправил уведомления через инструменты
- LLM может использовать инструменты (send_message) для действий

## См. также

- `internal/agent/loop` — реализация ProcessHeartbeatCheck
- `internal/workspace` — HEARTBEAT.md файл
