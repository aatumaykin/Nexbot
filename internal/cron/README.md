# Cron Scheduler

## Назначение

Cron Scheduler обеспечивает планирование периодических задач с использованием библиотеки robfig/cron/v3. Задачи могут быть recurring или oneshot.

## Основные компоненты

### Scheduler

Планировщик cron с функциями:
- `AddJob` — добавление задачи
- `RemoveJob` — удаление задачи
- `ListJobs` — список всех задач
- `GetJob` — получение задачи по ID
- `IsStarted` — проверка статуса запуска

### Job

Единица работы:
- `ID` — уникальный ID
- `Type` — тип (recurring, oneshot)
- `Schedule` — cron выражение
- `ExecuteAt` — время выполнения (для oneshot)
- `Tool` — внутренний инструмент для выполнения (send_message, agent)
- `Payload` — параметры для tool (JSON объект)
- `SessionID` — ID сессии для отправки сообщения (формат "telegram:chat_id")
- `UserID` — пользователь
- `Metadata` — метаданные

### JobType

Типы задач:
- `JobTypeRecurring` — периодические задачи
- `JobTypeOneshot` — разовые задачи

### Storage

Постоянное хранение задач:
- `UpsertJob` — создание или обновление
- `Remove` — удаление
- `GetJobs` — получение всех
- `RemoveExecutedOneshots` — очистка выполненных oneshot

## Создание напоминаний через Cron Tool

### Обязательный формат для напоминаний

Когда пользователь просит напоминание ("напомни через X минут" или "поставь напоминание"):

```json
{
  "action": "add_oneshot",
  "execute_at": "2026-02-08T01:00:00Z",
  "tool": "send_message",
  "payload": "{\"message\": \"YOUR_TEXT_HERE\"}",
  "session_id": "telegram:CHAT_ID_HERE"
}
```

### Обязательные параметры

**`tool`** — что выполнить (ОБЯЗАТЕЛЬНО):
- `"send_message"` — отправляет сообщение напрямую в Telegram чат
- `"agent"` — обрабатывает команду через агент используя `payload`

**`payload`** — параметры для tool (ОБЯЗАТЕЛЬНО):
- JSON строка содержащая параметры для tool
- Для `send_message` или `agent`: `{"message": "текст"}`
- REQUIRED когда указан `tool`

**`session_id`** — куда отправить (ОБЯЗАТЕЛЬНО для send_message/agent):
- ID сессии из раздела "Session Information" в начале промпта агента
- Формат: `"telegram:CHAT_ID"` (например, "telegram:35052705")
- REQUIRED для `tool="send_message"` или `tool="agent"`

**`execute_at`** — когда выполнить:
- ISO8601 формат даты и времени (например, "2026-02-08T01:00:00Z")
- Рассчитывается как: текущее время + X минут
- REQUIRED для `action="add_oneshot"`

### Как работает

1. Пользователь просит напоминание → агент создает cron job с параметрами `tool`, `payload`, `session_id`
2. Cron job выполняется в назначенное время → вызывает tool с указанными параметрами
3. Сообщение отправляется в правильный Telegram чат ✅

### Примеры использования

#### Создание напоминания на 1 минуту:
```json
{
  "action": "add_oneshot",
  "execute_at": "2026-02-08T01:01:00Z",
  "tool": "send_message",
  "payload": "{\"message\": \"Напоминание: прошла 1 минута\"}",
  "session_id": "telegram:35052705"
}
```

#### Создание повторяющегося напоминания:
```json
{
  "action": "add_recurring",
  "schedule": "0 * * * *",
  "tool": "send_message",
  "payload": "{\"message\": \"Ежедневное напоминание\"}",
  "session_id": "telegram:35052705"
}
```

### Ошибки при выполнении

**Будут логироваться ошибки если:**
- `tool` не указан (теперь обязателен)
- `payload` отсутствует когда указан `tool`
- `session_id` отсутствует для `tool="send_message"` или `tool="agent"`
- Неверный формат даты в `execute_at`

### Устарело

**НЕ ИСПОЛЬЗУЙТЕ:**
- ~~Параметр `command`~~ для создания задач через CLI или config
- Используйте `tool` + `payload` + `session_id` для создания напоминаний через cron tool
