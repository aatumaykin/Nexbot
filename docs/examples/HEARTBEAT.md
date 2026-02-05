# HEARTBEAT - Proactive Tasks

Этот файл определяет proactive задачи, которые будут выполняться автоматически по расписанию.

## Формат
Каждая задача определяется в следующем формате:

```markdown
- Task: "Описание задачи"
  Schedule: "cron-выражение"
  Description: "Подробное описание задачи"
```

## Примеры

### Daily Tasks

- Task: "Morning health check"
  Schedule: "0 9 * * *"
  Description: "Проверить состояние системы при запуске"

- Task: "Daily report"
  Schedule: "0 18 * * *"
  Description: "Создать и отправить ежедневный отчёт"

### Weekly Tasks

- Task: "Weekly cleanup"
  Schedule: "0 3 * * 0"
  Description: "Очистить старые лог-файлы"

- Task: "Weekly backup"
  Schedule: "0 4 * * 0"
  Description: "Создать резервную копию данных"
