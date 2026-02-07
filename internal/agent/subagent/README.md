# Agent Subagent

## Назначение

Agent Subagent управляет созданием и управлением concurrent экземпляров агента с изолированными сессиями. Subagents работают параллельно, каждый со своим контекстом и памятью.

## Основные компоненты

### Subagent
Экземпляр spawned агента с функциями:
- Обработка задач
- Изолированный контекст
- Уникальный session ID

### Manager
Менеджер subagents с функциями:
- Спавн новых subagents
- Остановка subagents
- Список всех активных subagents
- Получение конкретного subagent
- Остановка всех subagents
- Подсчет количества subagents

## Использование

### Создание manager

```go
import (
    "github.com/aatumaykin/nexbot/internal/agent/subagent"
)

func main() {
    // Создание manager
    mgr, err := subagent.NewManager(subagent.Config{
        SessionDir: "/path/to/sessions",
        Logger:     log,
        LoopConfig: loop.Config{
            Workspace:   "/path/to/workspace",
            SessionDir:  "/path/to/sessions",
            LLMProvider: provider,
            Logger:      log,
            Model:       "glm-4.7-flash",
        },
    })
    if err != nil {
        log.Fatal(err)
    }
}
```

### Спавн subagent

```go
// Спавн нового subagent
sub, err := mgr.Spawn(ctx, parentSession, "Выполнить задачу")
if err != nil {
    log.Error("Failed to spawn subagent", err)
}

// Обработка задачи
response, err := sub.Process(ctx, "Детали задачи...")
if err != nil {
    log.Error("Subagent processing failed", err)
}

fmt.Println(response)
```

### Управление subagents

```go
// Список всех subagents
subagents := mgr.List()

// Получение конкретного subagent
sub, err := mgr.Get(subagentID)

// Остановка конкретного subagent
err = mgr.Stop(subagentID)

// Остановка всех subagents
mgr.StopAll()

// Подсчет subagents
count := mgr.Count()
```

## Конфигурация

### Manager Config

- `SessionDir` — директория для хранения сессий subagents (обязательно)
- `Logger` — логгер manager (обязательно)
- `LoopConfig` — конфигурация для создания new loop (обязательно)

## Зависимости

- `github.com/aatumaykin/nexbot/internal/agent/loop` — создание экземпляров loop
- `github.com/aatumaykin/nexbot/internal/agent/session` — изолированные сессии
- `github.com/google/uuid` — уникальные ID

## Примечания

- Subagents имеют префикс `subagent-` в session ID
- Каждый subagent имеет уникальный UUID ID
- Context автоматически имеет timeout 5 минут
- Context изолирован от parent (отменяется при Stop)
- Sessions хранятся в `<sessionDir>/subagents/`

## См. также

- `internal/agent/loop` — основной loop
- `internal/agent/session` — управление сессиями
