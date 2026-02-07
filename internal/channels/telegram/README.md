# Telegram Channel

## Назначение

Telegram Channel обеспечивает интеграцию с Telegram Bot через библиотеку Telego. Обрабатывает входящие сообщения, маршрутизирует их в message bus и отправляет исходящие сообщения.

## Основные компоненты

### Connector
Телефонный бот connector:
- `Start` — инициализация и запуск
- `Stop` — graceful shutdown
- `HandleUpdate` — обработка обновлений Telegram
- `sendTypingIndicator` — отправка индикатора печати

### UpdateHandler
Обработчик обновлений Telegram.
- Обработка входящих сообщений
- Валидация пользователей (whitelist)
- Создание inbound сообщений

### CommandHandler
Обработчик команд бота.
- `new` — новая сессия
- `status` — статус сессии
- `restart` — перезапуск бота

### TypingManager
Менеджер индикатора печати.
- Автоматическое включение/выключение при обработке
- Управление несколькими чатами

### LongPollManager
Менеджер long polling.
- Получение обновлений от Telegram
- Обработка ошибок сети

## Использование

### Создание connector

```go
import (
    "github.com/aatumaykin/nexbot/internal/channels/telegram"
    "github.com/aatumaykin/nexbot/internal/bus"
    "github.com/aatumaykin/nexbot/internal/config"
    "github.com/aatumaykin/nexbot/internal/logger"
)

func main() {
    cfg := config.TelegramConfig{
        Token:        "123456789:ABCdefGHIjklMNOpqrsTUVwxyz",
        Enabled:      true,
        AllowedUsers: []string{"user123", "user456"},
    }

    log, _ := logger.New(logger.Config{
        Level:  "info",
        Format: "json",
        Output: "stdout",
    })

    // Создание message bus
    busInstance := bus.New(1000, log)
    busInstance.Start(ctx)

    // Создание connector
    conn := telegram.New(cfg, log, busInstance)
}
```

### Запуск connector

```go
// Запуск Telegram bot
err := conn.Start(ctx)
if err != nil {
    log.Fatal(err)
}
```

### Обработка сообщений

```go
// Connector автоматически обрабатывает входящие сообщения
// Сообщения публикуются в message bus

// Подписка на inbound сообщения
ch := bus.SubscribeInbound(ctx)
for msg := range ch {
    if msg.ChannelType == bus.ChannelTypeTelegram {
        // Обработка сообщения
        log.Info("Telegram message received",
            logger.Field{Key: "user_id", Value: msg.UserID},
            logger.Field{Key: "content", Value: msg.Content})
    }
}
```

## Конфигурация

### TelegramConfig

- `Token` — токен бота (обязательно)
- `Enabled` — включен ли канал (по умолчанию: false)
- `AllowedUsers` — список разрешенных пользователей (whitelist)

## Зависимости

- `github.com/mymmrac/telego` — библиотека Telegram Bot
- `github.com/aatumaykin/nexbot/internal/bus` — message bus
- `github.com/aatumaykin/nexbot/internal/logger` — логирование

## Примечания

- Токен имеет формат: `<bot_id>:<token>`
- Whitelist: если пустой — все пользователи разрешены
- Сообщения отправляются в markdown формате
- Индикатор печати включается автоматически при обработке
- Long polling имеет timeout по умолчанию 30 секунд

## См. также

- `internal/bus` — message bus
- `internal/commands` — обработка команд
