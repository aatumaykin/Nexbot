# Agent Context

## Назначение

Agent Context отвечает за построение системного промпта из компонентов контекста. Комбинирует bootstrap файлы (AGENTS.md, IDENTITY.md, USER.md, TOOLS.md, HEARTBEAT.md) в приоритетном порядке для создания полного контекста системы.

## Основные компоненты

### Builder
Построитель контекста с функциями:
- Сборка системного промпта из всех компонентов
- Обработка шаблонов с динамическими данными
- Добавление памяти из директории памяти
- Создание промпта для конкретной сессии

### Context
Структура контекста с путями к файлам.

## Использование

### Создание builder

```go
import (
    "github.com/aatumaykin/nexbot/internal/agent/context"
)

func main() {
    // Создание builder
    b, err := context.NewBuilder(context.Config{
        Workspace: "/path/to/workspace",
    })
    if err != nil {
        log.Fatal(err)
    }
}
```

### Сборка контекста

```go
// Базовая сборка (все компоненты)
systemPrompt, err := b.Build()
if err != nil {
    log.Error("Failed to build context", err)
}

// Добавление памяти
messages := []llm.Message{
    {Role: llm.RoleUser, Content: "Привет!"},
    {Role: llm.RoleAssistant, Content: "Привет!"},
}
systemWithMemory, err := b.BuildWithMemory(messages)

// Создание промпта для сессии
sessionPrompt, err := b.BuildForSession(sessionID, messages)
```

### Чтение памяти

```go
// Чтение всех файлов памяти
memoryMessages, err := b.ReadMemory()
if err != nil {
    log.Error("Failed to read memory", err)
}

// Чтение конкретного компонента
identity, err := b.GetComponent("IDENTITY")
user, err := b.GetComponent("USER")
```

## Конфигурация

### Параметры Config

- `Workspace` — путь к рабочей директории (обязательно)

## Зависимости

- `github.com/aatumaykin/nexbot/internal/heartbeat` — парсинг HEARTBEAT.md
- `github.com/aatumaykin/nexbot/internal/llm` — типы сообщений
- `github.com/aatumaykin/nexbot/internal/workspace` — bootstrap файлы

## Примечания

- Порядок компонентов: AGENTS → IDENTITY → USER → TOOLS → HEARTBEAT → memory
- Шаблоны заменяются автоматически ({{CURRENT_TIME}}, {{CURRENT_DATE}}, {{WORKSPACE_PATH}})
- HEARTBEAT.md парсится и форматируется как контекст
- Memory файлы читаются из директории `workspace/memory/`

## См. также

- `internal/agent/loop` — использование контекста в loop
- `internal/agent/memory` — хранение памяти
