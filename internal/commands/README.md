# Commands

## Назначение

Commands обеспечивает обработку команд Telegram бота. Поддерживает команды: `new`, `status`, `restart`.

## Основные компоненты

### Handler
Обработчик команд с функциями:
- `HandleCommand` — обработка команды
- `handleNewSession` — новая сессия
- `handleStatus` — статус сессии
- `handleRestart` — перезапуск

### Интерфейсы

#### AgentLoopInterface
Интерфейс для операций с agent loop:
- `ClearSession`
- `GetSessionStatus`

#### MessageBusInterface
Интерфейс для операций с message bus:
- `PublishOutbound`

## Использование

### Создание handler

```go
import (
    "github.com/aatumaykin/nexbot/internal/commands"
    "github.com/aatumaykin/nexbot/internal/logger"
)

func main() {
    log, _ := logger.New(logger.Config{
        Level:  "info",
        Format: "json",
        Output: "stdout",
    })

    // Создание handler
    handler := commands.NewHandler(
        agentLoop,
        messageBus,
        log,
        onRestart,
    )
}
```

### Обработка команды

```go
// Обработка команды "new"
err = handler.HandleCommand(ctx, "new", inboundMsg)

// Обработка команды "status"
err = handler.HandleCommand(ctx, "status", inboundMsg)

// Обработка команды "restart"
err = handler.HandleCommand(ctx, "restart", inboundMsg)
```

### Callback на restart

```go
// Функция перезапуска
onRestart := func() error {
    log.Info("Restarting application...")
    // Логика перезапуска
    return nil
}

// Создание handler с callback
handler := commands.NewHandler(
    agentLoop,
    messageBus,
    log,
    onRestart,
)
```

## Конфигурация

### Параметры Handler

- `agentLoop` — агент loop (обязательно)
- `messageBus` — message bus (обязательно)
- `logger` — логгер (обязательно)
- `onRestart` — callback на команду restart (необязательно)

## Зависимости

- `github.com/aatumaykin/nexbot/internal/bus` — message bus
- `github.com/aatumaykin/nexbot/internal/logger` — логирование
- `github.com/aatumaykin/nexbot/internal/messages` — форматирование сообщений

## Примечания

- Команды регистрируются в Telegram через SetMyCommands
- Новая сессия очищает историю сообщений
- Статус включает информацию о сессии и LLM конфигурации
- Сообщения отправляются через message bus

## См. также

- `internal/channels/telegram` — обработка команд в telegram
- `internal/messages` — форматирование сообщений
