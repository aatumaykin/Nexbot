# IPC Handler

## Назначение

IPC Handler обеспечивает взаимодействие с приложением через Unix socket (CLI). Поддерживает отправку сообщений и запросы к агенту.

## Основные компоненты

### Handler
Обработчик IPC с функциями:
- `Start` — запуск IPC сервера
- `Stop` — остановка сервера
- `handleConnection` — обработка подключений
- `handleSendMessage` — обработка отправки сообщений
- `handleAgent` — обработка запросов к агенту

### Request
Структура запроса:
- `Type` — тип запроса (send_message, agent)
- `Channel` — канал (telegram, discord, slack, web, api, cron)
- `SessionID` — ID сессии
- `UserID` — ID пользователя
- `Content` — содержимое

### Response
Структура ответа:
- `Success` — успешность
- `Error` — ошибка (если нет)

## Использование

### Создание handler

```go
import (
    "github.com/aatumaykin/nexbot/internal/ipc"
    "github.com/aatumaykin/nexbot/internal/bus"
    "github.com/aatumaykin/nexbot/internal/logger"
)

func main() {
    log, _ := logger.New(logger.Config{
        Level:  "info",
        Format: "json",
        Output: "stdout",
    })

    // Создание message bus
    busInstance := bus.New(1000, log)
    busInstance.Start(ctx)

    // Создание handler
    handler, err := ipc.NewHandler(log, "/path/to/sessions", busInstance)
    if err != nil {
        log.Fatal(err)
    }
}
```

### Запуск IPC сервера

```go
// Запуск
err = handler.Start(ctx, "/tmp/nexbot.sock")
if err != nil {
    log.Fatal(err)
}
```

### Отправка сообщения

```go
// Создание запроса
req := ipc.Request{
    Type:      "send_message",
    Channel:   "telegram",
    SessionID: "session123",
    UserID:    "user123",
    Content:   "Привет!",
}

// Отправка через socket
conn, _ := net.Dial("unix", "/tmp/nexbot.sock")
json.NewEncoder(conn).Encode(req)
// Чтение ответа
var resp ipc.Response
json.NewDecoder(conn).Decode(&resp)
```

### Запрос к агенту

```go
// Создание запроса
req := ipc.Request{
    Type:      "agent",
    Channel:   "telegram",
    SessionID: "session123",
    UserID:    "user123",
    Content:   "Выполнить задачу",
}

// Отправка через socket
conn, _ := net.Dial("unix", "/tmp/nexbot.sock")
json.NewEncoder(conn).Encode(req)
// Ответ придет в канал через message bus
```

## Конфигурация

### Handler

- `logger` — логгер (обязательно)
- `sessionDir` — директория сессий (обязательно)
- `messageBus` — message bus (обязательно)

## Зависимости

- `net` — Unix socket
- `encoding/json` — JSON сериализация
- `github.com/aatumaykin/nexbot/internal/bus` — message bus
- `github.com/aatumaykin/nexbot/internal/agent/session` — сессии

## Примечания

- Socket автоматически удаляется при запуске, если существует
- Используется Unix socket для локального взаимодействия
- Multiple подключений поддерживаются
- Отправки блокируются при обработке предыдущего запроса
- Валидация каналов: telegram, discord, slack, web, api, cron

## См. также

- `internal/bus` — message bus
- `internal/channels/telegram` — Telegram integration
- `internal/agent/loop` — обработка запросов
