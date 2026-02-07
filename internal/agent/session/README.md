# Agent Session

## Назначение

Agent Session управляет хранением истории сессий в формате JSONL. Каждая сессия хранится в отдельном файле с метаданными (timestamp, metadata).

## Основные компоненты

### Session
Представляет сессию с функциями:
- Добавление сообщений в историю
- Чтение всех сообщений
- Очистка сессии
- Удаление сессии
- Проверка существования
- Подсчет количества сообщений

### Manager
Менеджер сессий с функциями:
- Создание или получение сессии
- Проверка существования сессии
- Получение всех сессий

## Использование

### Создание manager

```go
import (
    "github.com/aatumaykin/nexbot/internal/agent/session"
)

func main() {
    // Создание manager
    mgr, err := session.NewManager("/path/to/sessions")
    if err != nil {
        log.Fatal(err)
    }
}
```

### Управление сессией

```go
// Получение или создание сессии
session, isNew, err := mgr.GetOrCreate(sessionID)
if err != nil {
    log.Error("Failed to get or create session", err)
}

// Добавление сообщения
err = session.Append(llm.Message{
    Role:    llm.RoleUser,
    Content: "Привет!",
})

// Чтение всех сообщений
messages, err := session.Read()

// Очистка сессии
err = session.Clear()

// Удаление сессии
err = session.Delete()

// Проверка существования
exists := session.Exists()

// Подсчет сообщений
count, err := session.MessageCount()

// Проверка через manager
exists, err := mgr.Exists(sessionID)
```

## Конфигурация

### Manager

- `baseDir` — базовая директория для файлов сессий (обязательно)

## Зависимости

- `encoding/json` — сериализация сообщений
- `sync` — конкурентное чтение/запись
- `time` — метаданные времени

## Примечания

- Каждый файл сессии имеет расширение `.jsonl`
- Структура файла:
  ```json
  {"message": {...}, "timestamp": "2026-02-07T12:00:00Z", "metadata": null}
  ```
- Парсинг корректно обрабатывает `\n` и `\r\n`
- Сообщения возвращаются в хронологическом порядке
- Concurrent safe через Mutex

## См. также

- `internal/agent/memory` — альтернативное хранение памяти
- `internal/agent/loop` — использование сессий в loop
