# Agent Memory

## Назначение

Agent Memory управляет хранением истории сообщений с поддержкой нескольких форматов. Использует JSONL для эффективного хранения и параллельного чтения.

## Основные компоненты

### Store
Менеджер памяти с функциями:
- Запись сообщений
- Чтение истории сессий
- Добавление множества сообщений
- Получение последних N сообщений
- Очистка сессии
- Проверка существования
- Получение списка всех сессий

### StorageFormat
Интерфейс для разных форматов хранения:
- `JSONLFormat` — JSONL формат (по умолчанию)
- `MarkdownFormat` — Markdown формат

## Использование

### Создание store

```go
import (
    "github.com/aatumaykin/nexbot/internal/agent/memory"
)

func main() {
    // Создание store (JSONL по умолчанию)
    store, err := memory.NewStore(memory.Config{
        BaseDir: "/path/to/memory",
        Format:  "jsonl",
    })
    if err != nil {
        log.Fatal(err)
    }
}
```

### Запись и чтение

```go
// Запись сообщения
msg := llm.Message{
    Role:    llm.RoleUser,
    Content: "Привет!",
}

err := store.Write(sessionID, msg)

// Чтение всех сообщений
messages, err := store.Read(sessionID)

// Добавление множества сообщений
messages := []llm.Message{
    {Role: llm.RoleUser, Content: "Привет!"},
    {Role: llm.RoleAssistant, Content: "Привет!"},
}
err = store.Append(sessionID, messages)

// Получение последних N сообщений
lastMessages, err := store.GetLastN(sessionID, 5)

// Проверка существования
exists := store.Exists(sessionID)

// Получение всех сессий
sessions, err := store.GetSessions()

// Очистка сессии
err = store.Clear(sessionID)
```

### Markdown формат

```go
// Создание store с Markdown форматом
store, err := memory.NewStore(memory.Config{
    BaseDir: "/path/to/memory",
    Format:  "markdown",
})

// Запись в Markdown формате
// Формат: ### Role [ID]\nContent
```

## Конфигурация

### Параметры Config

- `BaseDir` — базовая директория для файлов памяти (обязательно)
- `Format` — формат хранения (jsonl или markdown, по умолчанию: jsonl)

## Зависимости

- `github.com/aatumaykin/nexbot/internal/llm` — типы сообщений
- `sync` — конкурентное чтение/запись

## Примечания

- JSONL формат обеспечивает эффективное параллельное чтение
- Markdown формат поддерживает многострочный контент
- Файлы сессий: `<session_id>.jsonl` или `<session_id>.md`
- Concurrent safe через RWMutex
- Пустые строки в JSONL пропускаются при чтении

## См. также

- `internal/agent/session` — альтернативное хранение сессий
- `internal/agent/loop` — использование памяти в loop
