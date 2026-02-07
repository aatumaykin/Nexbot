# Message Bus

## Назначение

Message Bus предоставляет централизованный механизм передачи сообщений между компонентами системы. Реализует паттерн Publish-Subscribe для децентрализованной коммуникации.

## Основные компоненты

### Bus
Центральный шина сообщений с очередями и подписчиками.

### Event
Представляет событие системы (обработка началась/завершилась).

### InboundMessage
Входящее сообщение от внешнего канала (Telegram, CLI и т.д.).

### OutboundMessage
Исходящее сообщение для отправки во внешний канал.

## Использование

### Создание шины

```go
import (
    "github.com/aatumaykin/nexbot/internal/bus"
    "github.com/aatumaykin/nexbot/internal/logger"
)

func main() {
    log, _ := logger.New(logger.Config{
        Level:  "info",
        Format: "json",
        Output: "stdout",
    })

    // Создание шины
    busInstance := bus.New(1000, log)
    busInstance.Start(ctx)
}
```

### Публикация входящего сообщения

```go
// Создание inbound сообщения
msg := bus.NewInboundMessage(
    bus.ChannelTypeTelegram,
    "user123",
    "session123",
    "Привет!",
)

// Публикация в шину
bus.PublishInbound(msg)
```

### Подписка на входящие сообщения

```go
// Подписка на входящие сообщения
ch := bus.SubscribeInbound(ctx)
for msg := range ch {
    // Обработка сообщения
    log.Info("Received message",
        logger.Field{Key: "channel", Value: msg.ChannelType},
        logger.Field{Key: "user_id", Value: msg.UserID},
        logger.Field{Key: "content", Value: msg.Content})
}
```

### Публикация исходящего сообщения

```go
// Создание outbound сообщения
msg := bus.NewOutboundMessage(
    bus.ChannelTypeTelegram,
    "user123",
    "session123",
    "Ответ на сообщение",
    nil,
)

// Публикация в шину
bus.PublishOutbound(msg)
```

### Подписка на события

```go
// Подписка на события
eventCh := bus.SubscribeEvent(ctx)
for event := range eventCh {
    log.Info("Received event",
        logger.Field{Key: "type", Value: event.Type},
        logger.Field{Key: "channel", Value: event.ChannelType})
}
```

### Отправка события

```go
// Отправка события
bus.PublishEvent(bus.Event{
    Type:        bus.EventTypeProcessingStart,
    ChannelType: bus.ChannelTypeTelegram,
    UserID:      "user123",
    SessionID:   "session123",
    Timestamp:   time.Now(),
})
```

## Конфигурация

### Параметры Bus

- `capacity` — максимальный размер очереди (по умолчанию: 1000)
- `logger` — логгер

## Зависимости

- `github.com/aatumaykin/nexbot/internal/logger` — логирование

## Примечания

- Bus автоматически закрывает каналы при Stop()
- Подписчики получают события только после SubscribeEvent()
- Подписчики получают сообщения только после SubscribeInbound()
- Отправка блокируется при заполнении очереди

## См. также

- `internal/agent/loop` — обработчик сообщений
- `internal/channels/telegram` — отправитель сообщений
