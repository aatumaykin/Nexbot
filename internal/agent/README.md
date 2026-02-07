# Agent Loop

## Назначение

Agent Loop управляет циклом выполнения агента, координируя взаимодействие между провайдером LLM, менеджером сессий и инструментами. Обрабатывает сообщения пользователя, вызовы инструментов и управляет контекстом диалога.

## Основные компоненты

### Loop
Основной цикл обработки сообщений с функциями:
- Обработка входящих сообщений
- Интеграция с LLM провайдером
- Управление инструментами (tool calling)
- Поддержка сессий

### SessionOperations
Менеджер операций над сессиями:
- Добавление сообщений в историю
- Получение истории сессий
- Очистка и удаление сессий
- Получение статистики сессий

### ToolExecutor
Исполнитель инструментов:
- Подготовка вызовов инструментов
- Выполнение инструментов с контекстом
- Обработка результатов

## Использование

### Создание loop

```go
import (
    "context"
    "github.com/aatumaykin/nexbot/internal/agent/loop"
    "github.com/aatumaykin/nexbot/internal/llm"
    "github.com/aatumaykin/nexbot/internal/logger"
)

func main() {
    ctx := context.Background()

    // Создание провайдера LLM
    provider := zai.NewProvider(cfg.ZAI)

    // Создание логгера
    log, err := logger.New(logger.Config{
        Level:  "info",
        Format: "json",
        Output: "stdout",
    })
    if err != nil {
        log.Fatal(err)
    }

    // Создание loop
    cfg := loop.Config{
        Workspace:    "/path/to/workspace",
        SessionDir:   "/path/to/sessions",
        LLMProvider:  provider,
        Logger:       log,
        Model:        "glm-4.7-flash",
        MaxTokens:    4096,
        Temperature:  0.7,
        MaxToolIterations: 10,
    }

    l, err := loop.NewLoop(cfg)
    if err != nil {
        log.Fatal(err)
    }
}
```

### Обработка сообщения

```go
// Обработка пользовательского сообщения
response, err := l.Process(ctx, sessionID, "Привет! Расскажи о себе.")
if err != nil {
    log.Error("Failed to process message", err)
}

fmt.Println(response)
```

### Регистрация инструментов

```go
// Регистрация пользовательского инструмента
tool := &MyTool{}
if err := l.RegisterTool(tool); err != nil {
    log.Error("Failed to register tool", err)
}
```

### Управление сессией

```go
// Добавление сообщения в сессию
err := l.AddMessageToSession(ctx, sessionID, llm.Message{
    Role:    llm.RoleUser,
    Content: "Новая задача",
})

// Получение истории сессии
messages, err := l.GetSessionHistory(ctx, sessionID)

// Очистка сессии
err = l.ClearSession(ctx, sessionID)

// Удаление сессии
err = l.DeleteSession(ctx, sessionID)
```

### Проверка сердцебиения

```go
// Обработка проверки сердцебиения
response, err := l.ProcessHeartbeatCheck(ctx)
if err != nil {
    log.Error("Heartbeat check failed", err)
}

if response == "HEARTBEAT_OK" {
    log.Info("Everything is good")
}
```

## Конфигурация

### Параметры Config

- `Workspace` — путь к рабочей директории (обязательно)
- `SessionDir` — путь к директории сессий (обязательно)
- `LLMProvider` — провайдер LLM (обязательно)
- `Logger` — логгер (обязательно)
- `Model` — имя модели (по умолчанию: "glm-4.7-flash")
- `MaxTokens` — максимальное количество токенов (по умолчанию: 4096)
- `Temperature` — температура сэмплирования (по умолчанию: 0.7)
- `MaxToolIterations` — максимальное количество итераций tool calling (по умолчанию: 10)

## Зависимости

- `github.com/aatumaykin/nexbot/internal/agent/context` — построение контекста системы
- `github.com/aatumaykin/nexbot/internal/agent/session` — управление сессиями
- `github.com/aatumaykin/nexbot/internal/agent/tools` — реестр инструментов
- `github.com/aatumaykin/nexbot/internal/llm` — провайдер LLM
- `github.com/aatumaykin/nexbot/internal/logger` — логирование

## Примечания

- Максимальное количество итераций tool calling предотвращает бесконечные циклы
- При ошибках возвращается graceful error сообщение, а не паника
- Все сессии хранятся в JSONL формате
- Контекст системы собирается из bootstrap файлов (AGENTS.md, IDENTITY.md, USER.md, TOOLS.md, HEARTBEAT.md)

## См. также

- `internal/agent/context` — построение контекста
- `internal/agent/session` — управление сессиями
- `internal/agent/memory` — хранение памяти
- `internal/agent/subagent` — subagents
