# Конфигурация Nexbot

Полная справка по всем опциям конфигурации Nexbot.

## Расположение файла конфигурации

Nexbot ищет конфигурацию в следующих местах (в порядке приоритета):

1. Флаг `--config` (наивысший приоритет)
2. `./config.toml` в текущей директории
3. `~/.nexbot/config.toml`

## Переменные окружения

Переменные окружения можно использовать в конфигурации:

```toml
# Простая ссылка
api_key = "${ZAI_API_KEY}"

# Со значением по умолчанию
api_key = "${ZAI_API_KEY:default-key}"
```

## Расширение путей

Пути в конфигурации поддерживают следующее:

- `~` раскрывается в домашнюю директорию пользователя
- Переменные окружения раскрываются через `${VAR}` синтаксис
- Примеры:
  - `path = "~/.nexbot"` → `/home/user/.nexbot`
  - `path = "${HOME}/.nexbot"` → `/home/user/.nexbot`

---

## Секции конфигурации

### `[workspace]` — Настройки workspace

Конфигурация директории workspace, где Nexbot хранит данные.

| Параметр | Тип | По умолчанию | Описание |
|----------|-----|--------------|----------|
| `path` | string | `~/.nexbot` | Путь к директории workspace. Поддерживает `~` расширение. |
| `bootstrap_max_chars` | int | `20000` | Максимум символов для чтения из bootstrap файлов (IDENTITY.md, AGENTS.md и т.д.) |

**Пример:**

```toml
[workspace]
path = "~/.nexbot"
bootstrap_max_chars = 20000
```

**Валидация:**
- `path` не должен быть пустым
- `path` не должен содержать `..` (path traversal prevention)
- `bootstrap_max_chars` должен быть положительным

---

### `[agent]` — Настройки агента

Конфигурация поведения агента и параметров модели.

| Параметр | Тип | По умолчанию | Описание |
|----------|-----|--------------|----------|
| `model` | string | `glm-4.7-flash` | Модель по умолчанию для запросов к LLM |
| `max_tokens` | int | `8192` | Максимум токенов в ответе LLM |
| `max_iterations` | int | `20` | Максимум итераций tool calling на запрос |
| `temperature` | float64 | `0.7` | Temperature для сэмплинга LLM (0.0 - 1.0) |
| `timeout_seconds` | int | `30` | Таймаут обработки запроса агента (включая tool calls) |

**Пример:**

```toml
[agent]
model = "glm-4.7-flash"
max_tokens = 8192
max_iterations = 20
temperature = 0.7
timeout_seconds = 60
```

**Валидация:**
- `max_tokens` должен быть положительным
- `max_iterations` должен быть положительным
- `temperature` должен быть между 0.0 и 1.0
- `timeout_seconds` должен быть положительным

---

### `[llm]` — Конфигурация LLM провайдера

Основная конфигурация LLM провайдера.

| Параметр | Тип | По умолчанию | Описание |
|----------|-----|--------------|----------|
| `provider` | string | `zai` | LLM провайдер: `zai` или `openai` |

**Пример:**

```toml
[llm]
provider = "zai"
```

**Валидация:**
- `provider` должен быть одним из: `zai`, `openai`

---

#### `[llm.zai]` — Конфигурация Z.ai

Конфигурация LLM провайдера Z.ai.

| Параметр | Тип | По умолчанию | Описание |
|----------|-----|--------------|----------|
| `api_key` | string | (требуется) | API ключ Z.ai (формат: `zai-*` или `sk-*`, минимум 10 символов) |
| `base_url` | string | `https://api.z.ai/api/coding/paas/v4` | Base URL Z.ai API |
| `model` | string | `glm-4.7-flash` | Модель Z.ai по умолчанию |
| `timeout_seconds` | int | `30` | Таймаут HTTP запросов к Z.ai API |

**Пример:**

```toml
[llm.zai]
api_key = "${ZAI_API_KEY}"
base_url = "https://api.z.ai/api/coding/paas/v4"
timeout_seconds = 60
model = "glm-4.7-flash"
```

**Валидация:**
- `api_key` обязателен, когда `provider = "zai"`
- `api_key` должен быть минимум 10 символов
- `api_key` должен начинаться с `zai-` или `sk-`

#### `[llm.openai]` — Конфигурация OpenAI

Конфигурация LLM провайдера OpenAI.

| Параметр | Тип | По умолчанию | Описание |
|----------|-----|--------------|----------|
| `api_key` | string | (требуется) | API ключ OpenAI (формат: `sk-*` или `org-*`, минимум 10 символов) |
| `base_url` | string | `https://api.openai.com/v1` | Base URL OpenAI API |
| `model` | string | `gpt-4` | Модель OpenAI по умолчанию |

**Пример:**

```toml
[llm]
provider = "openai"

[llm.openai]
api_key = "${OPENAI_API_KEY}"
base_url = "https://api.openai.com/v1"
model = "gpt-4"
```

**Валидация:**
- `api_key` обязателен, когда `provider = "openai"`
- `api_key` должен быть минимум 10 символов
- `api_key` должен начинаться с `sk-` или `org-`

---

### `[logging]` — Настройки логирования

Конфигурация вывода логов.

| Параметр | Тип | По умолчанию | Описание |
|----------|-----|--------------|----------|
| `level` | string | `info` | Уровень логирования: `debug`, `info`, `warn`, `error` |
| `format` | string | `json` | Формат логов: `json` или `text` |
| `output` | string | `stdout` | Вывод логов: `stdout`, `stderr`, или путь к файлу |

**Пример:**

```toml
[logging]
level = "info"
format = "json"
output = "stdout"
```

Или для вывода в файл:

```toml
[logging]
level = "debug"
format = "text"
output = "~/.nexbot/nexbot.log"
```

**Валидация:**
- `level` должен быть одним из: `debug`, `info`, `warn`, `error`
- `format` должен быть одним из: `json`, `text`
- `output` не должен быть пустым

---

### `[channels]` — Настройки каналов

Конфигурация каналов коммуникации (Telegram, Discord и др.).

#### `[channels.telegram]` — Конфигурация Telegram

| Параметр | Тип | По умолчанию | Описание |
|----------|-----|--------------|----------|
| `enabled` | bool | `false` | Включить канал Telegram |
| `token` | string | (требуется) | Токен Telegram бота от [@BotFather](https://t.me/BotFather) |
| `allowed_users` | []string | `[]` | Список разрешённых Telegram user ID (пусто = разрешить всем) |
| `allowed_chats` | []string | `[]` | Список разрешённых Telegram chat ID (пусто = разрешить всем) |

**Пример:**

```toml
[channels.telegram]
enabled = true
token = "${TELEGRAM_BOT_TOKEN}"
allowed_users = ["123456789", "987654321"]
allowed_chats = []
```

**Валидация:**
- `token` обязателен, когда `enabled = true`
- `token` должен соответствовать формату: `<bot_id>:<token>`
  - `bot_id`: 3-15 цифр
  - `token`: 10-50 символов

**Заметки по безопасности:**
- Используйте `allowed_users` для ограничения доступа конкретным пользователям
- Оставьте `allowed_users` пустым для разрешения всем (не рекомендуется для продакшена)
- Вы можете найти свой Telegram user ID через ботов типа [@userinfobot](https://t.me/userinfobot)

#### `[channels.discord]` — Конфигурация Discord (будущее)

| Параметр | Тип | По умолчанию | Описание |
|----------|-----|--------------|----------|
| `enabled` | bool | `false` | Включить канал Discord (пока не реализован) |
| `token` | string | (требуется) | Токен Discord бота |
| `allowed_users` | []string | `[]` | Список разрешённых Discord user ID |
| `allowed_guilds` | []string | `[]` | Список разрешённых Discord server ID |

**Пример:**

```toml
[channels.discord]
enabled = false
token = "${DISCORD_BOT_TOKEN}"
allowed_users = []
allowed_guilds = []
```

---

### `[tools]` — Настройки инструментов

Конфигурация встроенных инструментов (file, shell).

#### `[tools.file]` — Инструменты работы с файлами

| Параметр | Тип | По умолчанию | Описание |
|----------|-----|--------------|----------|
| `enabled` | bool | `true` | Включить операции с файлами (read_file, write_file, list_dir) |
| `whitelist_dirs` | []string | `[]` | Список директорий, где разрешены операции с файлами |
| `read_only_dirs` | []string | `[]` | Список директорий только для чтения |

**Пример:**

```toml
[tools.file]
enabled = true
whitelist_dirs = ["~/.nexbot", "~/projects", "~/Documents"]
read_only_dirs = ["/etc", "/usr", "/bin"]
```

**Заметки по безопасности:**
- Операции с файлами ограничены `whitelist_dirs`
- Файлы в `read_only_dirs` можно только читать, не писать
- Path traversal всегда блокирован (функция безопасности)
- Пустой `whitelist_dirs` по умолчанию означает, что операции с файлами запрещены

#### `[tools.shell]` — Инструменты работы с shell

| Параметр | Тип | По умолчанию | Описание |
|----------|-----|--------------|----------|
| `enabled` | bool | `true` | Включить выполнение shell команд |
| `allowed_commands` | []string | `[]` | Список разрешённых shell команд |
| `deny_commands` | []string | `[]` | Список запрещённых shell команд (блокируют выполнение) |
| `ask_commands` | []string | `[]` | Список команд, требующих подтверждения пользователя |
| `working_dir` | string | `~/.nexbot` | Рабочая директория по умолчанию для shell команд |
| `timeout_seconds` | int | `30` | Таймаут выполнения shell команды |

**Порядок проверки команд:**

1. **deny_commands** — если команда совпадает → ошибка (запрещено)
2. **ask_commands** — если команда совпадает → запрос подтверждения
3. **allowed_commands** — если список не пустой и команда НЕ совпадает → ошибка

Если все три списка пустые → все команды разрешены (fail-open).

**Паттерны для команд:**

- **Точное совпадение:** `echo` → совпадает только `echo`
- **Базовая команда:** `git` → совпадает `git commit`, `git status`, `git log` и т.д.
- **Wildcard с `*`:** `git *` → совпадает все команды git (также как базовая команда)
- **Полный wildcard:** `*` → совпадает все команды

**Пример:**

```toml
[tools.shell]
enabled = true
allowed_commands = ["ls", "cat", "grep", "find", "cd", "pwd", "echo", "date", "git"]
deny_commands = ["rm", "rmdir", "dd", "mkfs", "fdisk", "shutdown"]
ask_commands = ["git *", "docker *"]
working_dir = "${NEXBOT_WORKSPACE:~/.nexbot}"
timeout_seconds = 30
```

**Пример с fail-open (все команды разрешены):**

```toml
[tools.shell]
enabled = true
allowed_commands = []
deny_commands = []
ask_commands = []
timeout_seconds = 30
```

**Валидация:**
- `allowed_commands` не может содержать пустые строки
- `deny_commands` не может содержать пустые строки
- `ask_commands` не может содержать пустые строки
- `working_dir` не должен содержать `..` (path traversal)

**Заметки по безопасности:**
- `deny_commands` имеет наивысший приоритет — если команда в этом списке, она всегда заблокирована
- `ask_commands` полезен для опасных команд (например, `docker *`, `git push`)
- `allowed_commands` используется для whitelist — только перечисленные команды разрешены
- Безопасные команды: `ls`, `cat`, `grep`, `find`, `pwd`, `echo`, `date`
- Опасные команды для deny: `rm`, `rmdir`, `dd`, `mkfs`, `fdisk`, `shutdown`

---

### `[cron]` — Настройки Cron (v0.2)

Конфигурация планирования задач.

| Параметр | Тип | По умолчанию | Описание |
|----------|-----|--------------|----------|
| `enabled` | bool | `false` | Включить cron планировщик |
| `timezone` | string | `UTC` | Часовой пояс для планирования задач |
| `jobs_file` | string | `jobs.json` | Файл с определениями cron задач |

**Пример:**

```toml
[cron]
enabled = true
timezone = "UTC"
jobs_file = "jobs.json"
```

**Формат файла cron задач:**

Cron задачи хранятся в JSON формате в файле `jobs_file`.

**Пример `jobs.json`:**

```json
{
  "jobs": [
    {
      "name": "daily-backup",
      "cron": "0 3 * * *",
      "description": "Запустить скрипт резервного копирования",
      "enabled": true
    },
    {
      "name": "weekly-report",
      "cron": "0 10 * * 1",
      "description": "Создать недельный отчёт",
      "enabled": true
    },
    {
      "name": "hourly-check",
      "cron": "15 * * * *",
      "description": "Запустить health check",
      "enabled": true
    }
  ]
}
```

**Формат cron выражений:**

Cron выражения следуют стандартному 5-полевому формату:

```
* * * * *
│ │ │ │ │
│ │ │ │ └─── День недели (0-7, 0 и 7 — воскресенье)
│ │ │ └───── Месяц (1-12)
│ │ └─────── День месяца (1-31)
│ └───────── Час (0-23)
└─────────── Минута (0-59)
```

**Общие примеры:**

| Выражение | Описание |
|-----------|----------|
| `* * * * *` | Каждую минуту |
| `*/5 * * * *` | Каждые 5 минут |
| `0 * * * *` | Каждый час |
| `0 0 * * *` | Ежедневно в полночь |
| `0 3 * * *` | Ежедневно в 3:00 |
| `0 9 * * 1` | Каждый понедельник в 9:00 |
| `0 */6 * * *` | Каждые 6 часов |
| `30 11 * * 1-5` | В будни в 11:30 |

**Переменные окружения в Cron:**

Cron задачи могут ссылаться на переменные окружения в описаниях/действиях:

```json
{
  "name": "daily-backup",
  "cron": "0 3 * * *",
  "description": "Запустить скрипт резервного копирования используя $HOME/.nexbot/backup.sh"
}
```

**Валидация:**
- `timezone` должен быть валидным часовым поясом (например, "UTC", "America/New_York")
- `jobs_file` должен быть валидным путём к JSON файлу
- Каждая задача должна иметь поля `name`, `cron`, и `description`
- `cron` выражение должно быть валидным (см. справку по cron)

**Cron Configuration (краткая справка):**

| Параметр | Тип    | Default | Описание                |
| --------- | ------ | ------- | ----------------------- |
| enabled   | bool   | true    | Включить cron scheduler |
| timezone  | string | UTC     | Часовой пояс            |

---

### `[workers]` — Настройки Worker Pool (v0.2)

Конфигурация пула async задач.

| Параметр | Тип | По умолчанию | Описание |
|----------|-----|--------------|----------|
| `pool_size` | int | `5` | Количество worker goroutines |
| `queue_size` | int | `100` | Ёмкость очереди для ожидающих задач |

**Пример:**

```toml
[workers]
pool_size = 5
queue_size = 100
```

**Валидация:**
- `pool_size` должен быть положительным (минимум 1)
- `queue_size` должен быть положительным (минимум 1)

---

### `[heartbeat]` — Настройки HEARTBEAT (v0.2)

Конфигурация системы proactive задач из HEARTBEAT.md.

| Параметр | Тип | По умолчанию | Описание |
|----------|-----|--------------|----------|
| `enabled` | bool | `true` | Включить выполнение HEARTBEAT задач |
| `heartbeat_file` | string | `HEARTBEAT.md` | Файл с задачами в workspace |
| `check_interval` | string | `"* * * * *"` | Как часто проверять HEARTBEAT.md на изменения |
| `workspace_path` | string | `~/.nexbot` | Путь к workspace для HEARTBEAT.md |

**Heartbeat Configuration (краткая справка):**

| Параметр | Тип  | Default | Описание                    |
| --------- | ---- | ------- | --------------------------- |
| enabled   | bool | true    | Включить heartbeat проверки |
| check_interval_minutes | int | 10      | Интервал проверки в минутах |

**Пример:**

```toml
[heartbeat]
enabled = true
heartbeat_file = "HEARTBEAT.md"
check_interval = "* * * * *"
workspace_path = "~/.nexbot"
```

**Формат HEARTBEAT.md:**

HEARTBEAT.md должен быть размещён в workspace (по умолчанию: `~/.nexbot/`).

Каждая задача определяется в следующем формате:

```markdown
- Task: "Описание задачи"
  Schedule: "cron-expression"
  Description: "Подробное описание"
```

**Пример HEARTBEAT.md:**

```markdown
# HEARTBEAT - Proactive Tasks

## Daily Tasks

- Task: "Daily health check"
  Schedule: "0 9 * * *"
  Description: "Проверить состояние системы"

- Task: "Daily report generation"
  Schedule: "0 18 * * *"
  Description: "Создать ежедневный отчёт"

## Weekly Tasks

- Task: "Weekly cleanup"
  Schedule: "0 3 * * 0"
  Description: "Очистить старые лог-файлы"

- Task: "Weekly backup"
  Schedule: "0 4 * * 0"
  Description: "Создать резервную копию данных"
```

**Cron expression format:**
- Минимум: `0-59`
- Час: `0-23`
- День месяца: `1-31`
- Месяц: `1-12`
- День недели: `0-6` (0=воскресенье, 6=суббота)

**Примеры cron expressions:**
- `* * * * *` — каждую минуту
- `0 * * * *` — каждый час
- `0 9 * * *` — ежедневно в 9:00
- `*/5 * * * *` — каждые 5 минут
- `0 0 * * 0` — каждую неделю в полночь (воскресенье)

---

### `[subagent]` — Настройки Subagent Manager (v0.2)

Конфигурация управления subagent'ами.

| Параметр | Тип | По умолчанию | Описание |
|----------|-----|--------------|----------|
| `enabled` | bool | `true` | Включить функциональность subagent |
| `max_concurrent` | int | `10` | Максимум одновременных subagents |
| `timeout_seconds` | int | `300` | Таймаут по умолчанию для задач subagent |
| `session_prefix` | string | `subagent-` | Префикс для session ID subagent |

**Пример:**

```toml
[subagent]
enabled = true
max_concurrent = 10
timeout_seconds = 300
session_prefix = "subagent-"
```

**Валидация:**
- `max_concurrent` должен быть минимум 1, когда включён
- `timeout_seconds` должен быть минимум 1, когда включён

---

### `[message_bus]` — Настройки Message Bus

Конфигурация системы очередей сообщений.

| Параметр | Тип | По умолчанию | Описание |
|----------|-----|--------------|----------|
| `capacity` | int | `1000` | Ёмкость очереди для входящих/исходящих сообщений |

**Пример:**

```toml
[message_bus]
capacity = 1000
```

**Валидация:**
- `capacity` должен быть положительным

**Заметки:**
- Большая ёмкость позволяет больше одновременных сообщений, но использует больше памяти
- По умолчанию (1000) подходит для большинства случаев использования
- Увеличьте ёмкость, если ожидаете пиковый трафик

---

## Полный пример конфигурации

```toml
# Конфигурация workspace
[workspace]
path = "~/.nexbot"
bootstrap_max_chars = 20000

# Настройки агента
[agent]
model = "glm-4.7-flash"
max_tokens = 8192
max_iterations = 20
temperature = 0.7
timeout_seconds = 60

# Конфигурация LLM провайдера
[llm]
provider = "zai"

[llm.zai]
api_key = "${ZAI_API_KEY}"
base_url = "https://api.z.ai/api/coding/paas/v4"
timeout_seconds = 60
model = "glm-4.7-flash"

# Конфигурация канала Telegram
[channels.telegram]
enabled = true
token = "${TELEGRAM_BOT_TOKEN}"
allowed_users = ["123456789"]  # Опционально: ограничить доступ конкретным пользователям
allowed_chats = []

# Конфигурация инструментов
[tools.file]
enabled = true
whitelist_dirs = ["~/.nexbot", "~/projects", "~/Documents"]
read_only_dirs = ["/etc", "/usr", "/bin"]

[tools.shell]
enabled = true
allowed_commands = ["ls", "cat", "grep", "find", "cd", "pwd", "echo", "date", "git"]
deny_commands = ["rm", "rmdir", "dd", "mkfs", "fdisk", "shutdown"]
ask_commands = ["git *", "docker *"]
working_dir = "~/.nexbot"
timeout_seconds = 30

# Конфигурация cron (v0.2.0)
[cron]
enabled = true
timezone = "UTC"
jobs_file = "jobs.json"

# Конфигурация worker pool (v0.2.0)
[workers]
pool_size = 5
queue_size = 100

# Конфигурация subagent manager (v0.2.0)
[subagent]
enabled = true
max_concurrent = 10
timeout_seconds = 300
session_prefix = "subagent-"

# Конфигурация HEARTBEAT (v0.2.0)
[heartbeat]
enabled = true
check_interval_minutes = 30

# Конфигурация логирования
[logging]
level = "info"
format = "json"
output = "stdout"

# Конфигурация message bus
[message_bus]
capacity = 1000
```

---

## Правила валидации

Nexbot проверяет конфигурацию при запуске. Если валидация не проходит, Nexbot покажет сообщения об ошибках и завершит работу с кодом 1.

### Валидация API ключей

- API ключи Z.ai должны:
  - Начинаться с `zai-` или `sk-`
  - Быть минимум 10 символов длиной
- API ключи OpenAI должны:
  - Начинаться с `sk-` или `org-`
  - Быть минимум 10 символов длиной

### Валидация токена Telegram

- Должен соответствовать формату: `<bot_id>:<token>`
  - `bot_id`: 3-15 цифр
  - `token`: 10-50 символов
- Пример: `1234567890:ABCdefGHIjklMNOpqrsTUVwxyz`

### Валидация путей

- Не должен быть пустым
- Не должен содержать `..` (path traversal prevention)
- Поддерживает `~` расширение для домашней директории
- Поддерживает расширение переменных окружения

### Валидация уровня логирования

- Должен быть одним из: `debug`, `info`, `warn`, `error`
- По умолчанию: `info`

### Валидация формата логирования

- Должен быть одним из: `json`, `text`
- По умолчанию: `json`

---

## Лучшие практики безопасности

1. **Используйте переменные окружения для секретов:**
   ```toml
   api_key = "${ZAI_API_KEY}"
   token = "${TELEGRAM_BOT_TOKEN}"
   ```

2. **Ограничьте доступ:**
   ```toml
   [channels.telegram]
   allowed_users = ["123456789"]  # Разрешить только конкретных пользователей
   ```

3. **Ограничьте shell команды:**
    ```toml
    [tools.shell]
    allowed_commands = ["ls", "cat", "grep"]  # Разрешить только безопасные команды
    deny_commands = ["rm", "rmdir", "dd"]     # Заблокировать опасные команды
    ask_commands = ["git *", "docker *"]       # Запрашивать подтверждение для этих команд
    ```

4. **Ограничьте доступ к файлам:**
   ```toml
   [tools.file]
   whitelist_dirs = ["~/.nexbot"]  # Доступ только к конкретным директориям
   ```

5. **Установите подходящий уровень логирования:**
   ```toml
   [logging]
   level = "info"  # Не используйте "debug" в продакшене
   ```

---

## Устранение неполадок

### Ошибки валидации конфигурации

**Ошибка:** "workspace.path is required"
- **Решение:** Добавьте секцию `[workspace]` со значением `path`

**Ошибка:** "llm.zai.api_key is required when provider is 'zai'"
- **Решение:** Добавьте `api_key` в секцию `[llm.zai]` или установите переменную окружения `ZAI_API_KEY`

**Ошибка:** "telegram token has invalid format"
- **Решение:** Убедитесь, что формат токена — `bot_id:token` (например, `1234567890:ABCdefGHIjklMNOpqrsTUVwxyz`)

**Ошибка:** "tools.shell.allowed_commands contains empty command"
- **Решение:** Удалите пустые строки из списка `allowed_commands`

**Ошибка:** "tools.shell.deny_commands contains empty command"
- **Решение:** Удалите пустые строки из списка `deny_commands`

**Ошибка:** "tools.shell.ask_commands contains empty command"
- **Решение:** Удалите пустые строки из списка `ask_commands`

### Ошибки выполнения

**Ошибка:** "Permission denied" при доступе к директориям
- **Решение:** Проверьте права доступа к директориям и добавьте их в `whitelist_dirs`

**Ошибка:** "Authentication error" от LLM провайдера
- **Решение:** Проверьте, что API ключ правильный и не истёк

---

## См. также

- [Вводный гайд](README.md) — обзор проекта и быстрый старт
- [Архитектура](ARCHITECTURE.md) — архитектура системы и внутренний дизайн
- [Пример конфига](../config.example.toml) — пример файла конфигурации
- [Cron справка](https://pkg.go.dev/github.com/robfig/cron/v3) — документация Go cron пакета
