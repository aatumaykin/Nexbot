# Архитектура Nexbot

Полная документация архитектуры системы Nexbot, компонентов и потока данных.

## Обзор

Nexbot использует модульную многослойную архитектуру с message bus в основе. Система спроектирована как:
- **Декомпозированная**: Компоненты общаются через message bus
- **Расширяемая**: Легко добавлять новые каналы, инструменты и навыки
- **Async-friendly**: Фоновые задачи, cron задания и subagents
- **Высокопроизводительная**: Эффективные goroutines и очереди

## Диаграмма архитектуры

```
┌─────────────────────────────────────────────────────────────┐
│                        MESSAGE BUS                          │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │ Inbound Queue│  │ Outbound Queue│  │ Task Queue   │      │
│  └──────────────┘  └──────────────┘  └──────────────┘      │
└─────────────────────────────────────────────────────────────┘
                              │
          ┌───────────────────┼───────────────────┐
          │                   │                   │
  ┌───────▼───────┐   ┌───────▼──────────┐  ┌────▼─────────────┐
  │   CHANNELS    │   │    AGENT CORE    │  │   TASK SYSTEM    │
  └───────────────┘   └──────────────────┘  └─────────────────┘
          │                   │                   │
  ┌───────▼───────┐   ┌───────▼──────────┐  ┌────▼─────────────┐
  │   TELEGRAM    │   │   LLM ENGINE     │  │    WORKER POOL   │
  │   (input)     │   │   (provider)     │  │   (async tasks)  │
  │   (v0.2)      │   └──────────────────┘  │    (NEW v0.2)   │
  └───────┬───────┘                          └──────┬───────────┘
          │                                         │
          │                                         │
          ▼                                         ▼
┌─────────────────────────────────────────────┐   ┌─────────────────┐
│         Tool Registry                      │   │   Cron          │
│  ├─ File Tools                            │   │   Scheduler     │
│  ├─ Shell Tools                           │   │   (NEW v0.2)   │
│  ├─ Cron Tool (NEW v0.2)                  │   └────────┬────────┘
│  └─ Spawn Tool (NEW v0.2)                  │            │
└────────────┬────────────────────────────────┘            │
             │                                             │
             ▼                                             ▼
      ┌──────────────┐                               ┌─────────────────┐
      │   Subagent   │                               │   HEARTBEAT     │
      │   Manager    │                               │   System        │
      │  (NEW v0.2)  │                               │   (NEW v0.2)   │
      └──────────────┘                               └─────────────────┘
             │
      ┌────────┴────────┐
      │                 │
┌─────▼──────┐   ┌─────▼──────┐
│   SKILLS   │   │  WORKSPACE │
│   SYSTEM   │   │  (context) │
└────────────┘   └────────────┘
      │                 │
┌─────▼─────────────────▼───────┐
│   AGENT CONTEXT              │
│   (IDENTITY, AGENTS, etc.)   │
└───────────────────────────────┘
```

## Основные компоненты

### Message Bus

Message bus — основа архитектуры Nexbot, позволяющая декомпозированное общение между компонентами.

**Компоненты:**
- **Inbound Queue**: Хранит входящие сообщения (Telegram, запланированные задачи)
- **Outbound Queue**: Хранит исходящие сообщения (ответы, уведомления)
- **Task Queue**: Хранит фоновые задачи (worker pool, cron задания)

**Поток данных:**
```
Telegram → Inbound Queue → Agent → Outbound Queue → Telegram
                                    │
                              Task Queue
                                    │
                              Worker Pool / Subagents
```

**Реализация:** Использует Go каналы с буферизированными очередями для высокой пропускной способности.

### Agent Core

Agent loop — основной блок обработки, который координирует обработку сообщений, взаимодействие с LLM и tool calling.

**Процесс:**
1. Получить сообщение из inbound queue
2. Загрузить контекст из workspace (IDENTITY, AGENTS, SOUL, USER)
3. Отправить сообщение LLM провайдеру
4. Парсить ответ LLM
5. Выполнить tool calls, если есть
6. Обработать spawning subagent, если есть
7. Отправить результат в outbound queue
8. Повторить

**Ключевые возможности:**
- Context builder с упорядоченным приоритетом (IDENTITY → AGENTS → SOUL → USER → TOOLS → MEMORY)
- Tool calling с максимумом итераций (по умолчанию 20)
- Координация subagents
- Обработка timeout (по умолчанию 30 секунд)

### Channels

Channels — входные/выходные коннекторы, которые получают сообщения и отправляют ответы.

**Текущие каналы:**
- **Telegram Channel**: Подключается к Telegram ботам через библиотеку telego

**Расширяемая архитектура:**
Новые каналы можно добавить, реализовав интерфейс `Channel`:
- `Connect() error` - Установить соединение
- `HandleMessage(msg) error` - Обработать входящее сообщение
- `SendMessage(msg) error` - Отправить ответ

### LLM Engine

Управляет коммуникацией с разными LLM провайдерами.

**Поддерживаемые провайдеры:**
- **Z.ai**: GLM-4.7 Flash (по умолчанию)
- **OpenAI**: Модели GPT

**Возможности:**
- Абстракционный слой провайдера
- Обработка ошибок и повторы
- Подсчёт токенов
- Управление timeout

### Tools System

Встроенные инструменты, расширяющие возможности агента.

**Встроенные инструменты:**
- `read_file` - Прочитать содержимое файла
- `write_file` - Записать содержимое файла
- `list_dir` - Список содержимого директории
- `shell_exec` - Выполнить shell команды
- `spawn` - Создать subagents (v0.2)

**Интерфейс Tool:**
```go
type Tool struct {
    Name        string
    Description string
    Execute     func(ctx context.Context, args map[string]interface{}) (interface{}, error)
    Schema      llm.ToolDefinition
}
```

### Skills System

Skills — markdown файлы, которые обучают агента использовать инструменты или выполнять задачи.

**Структура:**
```
skills/
├── weather/
│   └── SKILL.md
├── github/
│   └── SKILL.md
└── custom/
    └── SKILL.md
```

**Формат файла Skill:**
```markdown
---
name: weather
description: Provides weather information
tools: [read_file, shell_exec]
---
```

### Workspace

Директория workspace (`~/.nexbot/`) хранит контекст агента и данные.

**Bootstrap файлы:**
1. **IDENTITY.md** - Основная идентичность бота
2. **AGENTS.md** - Инструкции агента и поведение
3. **SOUL.md** - Личность и тон бота
4. **USER.md** - Профиль пользователя и предпочтения
5. **TOOLS.md** - Справка по инструментам
6. **MEMORY.md** - Долгосрочная память

**Порядок сборки контекста:**
```
IDENTITY → AGENTS → SOUL → USER → TOOLS → MEMORY
```

## Новые компоненты v0.2

### Cron Scheduler

Фоновый планировщик для выполнения задач по расписанию с поддержкой recurring и oneshot задач.

**Подробная документация:** [docs/architecture/cron_scheduler.md](architecture/cron_scheduler.md)

**Компоненты:**
- **Scheduler**: Основной планировщик на robfig/cron/v3
- **Storage**: Персистентное хранение в JSONL формате
- **Worker Pool Integration**: Асинхронное выполнение задач через Worker Pool
- **Job Registry**: Отслеживание активных задач

**Архитектура:**
```
Cron Engine (robfig/cron)
    │
    ├─ Recurring Jobs (cron expression)
    │   └─ → Trigger → Worker Pool → Agent
    │
    └─ Oneshot Jobs (ticker 1 min)
        └─ → Check ExecuteAt → Worker Pool → Agent
            └─ Cleanup (ticker 24h)
```

**Типы задач:**
- **Recurring**: Повторяющиеся задачи по cron выражению
- **Oneshot**: Однократные задачи в указанное время

**Хранилище:** `~/.nexbot/cron/jobs.jsonl`

**Ключевые возможности:**
- Поддержка cron выражений с секундами (`* * * * * *`)
- Worker Pool для асинхронного выполнения
- Fallback на Message Bus если Worker Pool недоступен
- Graceful shutdown с cleanup выполненных oneshot задач
- Thread-safe операции через mutex

### Subagent Manager

Управляет созданием и выполнением изолированных subagents с собственными сессиями и памятью.

**Подробная документация:** [docs/architecture/subagent_manager.md](architecture/subagent_manager.md)

**Компоненты:**
- **Manager**: Создание и управление жизненным циклом subagents
- **Subagent**: Изолированный экземпляр агента с собственным Loop
- **Storage**: Изолированное хранение сессий в отдельных директориях

**Архитектура:**
```
Main Agent
    │
    ├─ Spawn Tool Call
    │
    └─ Subagent Manager
            │
            ├─ Generate UUID for subagent
            ├─ Create isolated session
            ├─ Create Loop (loopFactory)
            ├─ Start agent loop
            ├─ Process task independently
            └─ Return result to main agent
```

**Хранилище:** `~/.nexbot/sessions/subagents/<subagent_id>/session.jsonl`

**Ключевые возможности:**
- Параллельное выполнение задач
- Изолированный контекст и память для каждого subagent
- Thread-safe операции через mutex
- Parent-child сессии для отслеживания происхождения
- Graceful shutdown через context cancellation

### Worker Pool

Асинхронный пул воркеров для фоновой обработки задач разных типов.

**Подробная документация:** [docs/architecture/worker_pool.md](architecture/worker_pool.md)

**Компоненты:**
- **WorkerPool**: Управление пулом goroutine воркеров
- **Task**: Единица работы с контекстом и payload
- **Result**: Результат выполнения с метриками
- **PoolMetrics**: Отслеживание метрик выполнения
- **taskWaitGroup**: Обертка sync.WaitGroup с thread-safe доступом

**Архитектура:**
```
Task Queue (buffered)
         │
         ▼
    ┌────────┐
    │ Worker │───┐
    │ Pool   │   │
    └────────┘   │
         │       │
    ┌────┴────┬──┴────┐
    │         │         │
 Worker 1  Worker 2  Worker N
    │         │         │
    ▼         ▼         ▼
Task Types:
  ├─ cron (Cron Scheduler)
  └─ subagent (Subagent tasks)
    │         │         │
    └─────────┴─────────┘
              │
         Result Channel
```

**Типы задач:**
- **cron**: Периодические задачи от Cron Scheduler
- **subagent**: Задачи для выполнения в subagent

**Метрики:**
- TasksSubmitted
- TasksCompleted
- TasksFailed
- TotalDuration

**Ключевые возможности:**
- Фиксированное количество воркеров для предсказуемого использования ресурсов
- Буферизированная очередь задач
- Результаты через канал для мониторинга
- Panic recovery для каждого воркера
- Context cancellation для graceful shutdown

### HEARTBEAT System

Система периодических проверок состояния через HEARTBEAT.md файл.

**Подробная документация:** [docs/architecture/heartbeat_system.md](architecture/heartbeat_system.md)

**Компоненты:**
- **Loader**: Загрузка и валидация HEARTBEAT.md
- **Parser**: Парсинг markdown контента в список задач
- **Checker**: Периодическая проверка через Agent interface
- **Agent Interface**: Обработка heartbeat проверок

**Архитектура:**
```
HEARTBEAT.md (workspace)
         │
         ▼
     Loader → Parser → Validate Tasks
         │
         ▼
  Checker (periodic interval)
         │
         ▼
  Agent.ProcessHeartbeatCheck()
         │
         ├─ Read HEARTBEAT.md
         ├─ Follow tasks
         ├─ Use tools
         └─ Return response
              │
              ▼
         Process Response
              │
              ├─ HEARTBEAT_OK (all good)
              └─ Actions taken
```

**Хранилище:** `~/.nexbot/HEARTBEAT.md`

**Формат HEARTBEAT.md:**
```markdown
# Heartbeat Tasks

## Periodic Reviews

### Daily Standup
- Schedule: "0 9 * * *"
- Task: "Review daily progress, check for blocked tasks, update priorities"

### Weekly Summary
- Schedule: "0 17 * * 5"
- Task: "Review weekly achievements, plan next week"
```

**Ключевые возможности:**
- Парсинг задач из markdown формата
- Валидация cron выражений
- Интеграция с Cron Scheduler для планирования
- Периодические проверки через Checker
- Agent-driven выполнение с инструментами

## Поток данных

### Основной поток сообщений

```
1. Telegram Message
   ↓
2. Telegram Channel получает сообщение
   ↓
3. Сообщение добавлено в Inbound Queue
   ↓
4. Agent Loop извлекает сообщение из очереди
   ↓
5. Загрузка контекста из файлов workspace (IDENTITY → AGENTS → SOUL → USER → TOOLS → MEMORY)
   ↓
6. Отправка LLM провайдеру (Z.ai)
   ↓
7. Парсинг ответа LLM
   ↓
8. Выполнение tool calls (если есть):
   ├─ read_file/write_file/list_dir
   ├─ shell_exec
   ├─ cron (Cron Scheduler)
   └─ spawn (Subagent Manager)
   ↓
9. Spawn subagents (если запрошено)
   ↓
10. Отправка ответа в Outbound Queue
    ↓
11. Telegram Channel отправляет ответ
```

### Поток Cron задач

```
1. Cron Engine запускается по расписанию (recurring) или ticker (oneshot)
   ↓
2. Поиск соответствующей cron задачи
   ↓
3. Триггер выполнения задачи
   ↓
4. Создание Task (cron_<job>_<ts>)
   ↓
5. Submit Task в Worker Pool
   ↓
6. Worker Pool забирает задачу
   ↓
7. Worker выполняет задачу через Agent/Message Bus
   ↓
8. Результат отправляется в result channel
   ↓
9. Логирование результатов выполнения
```

### Поток Subagent

```
1. Главный агент вызывает spawn tool
   ↓
2. Subagent Manager получает запрос (Spawn ctx, parentSession, task)
   ↓
3. Lock mutex
   ↓
4. Generate UUID for subagent
   ↓
5. Create isolated session ID (subagent-<ts>)
   ↓
6. Create Context (WithCancel)
   ↓
7. Create Loop (loopFactory)
   ↓
8. Create Subagent struct
   ↓
9. Store in registry (subagents map)
   ↓
10. Unlock mutex
   ↓
11. Return Subagent
    ↓
12. Subagent.Process(task) → Loop.Process()
    ↓
13. Return response to main agent
    ↓
14. Stop(id) → Cancel Context → Delete from registry
```

### Поток Worker Pool

```
1. Задача отправлена в очередь (Submit(task))
   ↓
2. Задача добавлена в буферизированный канал (taskQueue)
   ↓
3. Worker Pool increment TasksSubmitted (mutex lock)
   ↓
4. Worker goroutine получает задачу из канала
   ↓
5. processTask(task) → Record start time
   ↓
6. Determine Context (task or pool)
   ↓
7. executeTask(ctx, task):
   ├─ Switch task.Type
   ├─ "cron" → executeCronTask()
   ├─ "subagent" → executeSubagentTask()
   └─ default → Error: Unknown Type
   ↓
8. Calculate Duration
   ↓
9. Update Metrics (TasksCompleted/TasksFailed)
   ↓
10. Send Result to resultCh
    ↓
11. Worker loop back to wait for next task
```

## Архитектура конфигурации

### Структура файла конфигурации

```
config.toml
├── [workspace] - Настройки workspace
├── [agent] - Поведение агента
├── [llm] - Настройки LLM провайдера
├── [llm.zai] - Специфичные настройки Z.ai
├── [llm.openai] - Специфичные настройки OpenAI
├── [logging] - Конфигурация логирования
├── [channels] - Коннекторы каналов
│   ├── [channels.telegram] - Конфигурация Telegram
│   └── [channels.discord] - Конфигурация Discord
├── [tools] - Конфигурация инструментов
│   ├── [tools.file] - File инструменты
│   └── [tools.shell] - Shell инструменты
├── [cron] - Cron планировщик (v0.2)
├── [workers] - Worker pool (v0.2)
├── [subagent] - Subagent manager (v0.2)
└── [message_bus] - Настройки message bus
```

### Иерархия конфигурации

```
Command Line Flag (наивысший приоритет)
    ↓
config.toml в текущей директории
    ↓
~/.nexbot/config.toml (по умолчанию)
    ↓
Переменные окружения (используются для секретов)
```

## Производительность

### Управление ресурсами

- **Message Bus**: Использует буферизированные каналы с настраиваемой ёмкостью
- **Worker Pool**: Фиксированный размер для предсказуемого использования памяти
- **Subagent Manager**: Лимит одновременных процессов предотвращает истощение ресурсов
- **Cron Engine**: Минимальные накладные расходы, эффективное отслеживание времени

### Конкурентность

- **Goroutines**: Эффективны для I/O-bound операций
- **Channels**: Low-latency передача сообщений
- **Mutex/Channel**: Безопасные паттерны конкурентного доступа
- **Context**: Правильная отмена долгих задач

### Масштабируемость

- **Горизонтально**: Можно масштабировать добавляя больше worker процессов
- **Вертикально**: Можно масштабировать увеличивая размер пула/ёмкость очереди
- **Декомпозировано**: Каждый компонент можно оптимизировать независимо

## Архитектура безопасности

### Валидация ввода

- **Конфигурация**: Schema валидация при запуске
- **Аргументы инструментов**: Whitelist паттерны
- **Path Traversal**: Всегда блокируется (функция безопасности)
- **Переменные окружения**: Только валидация ссылок

### Контроль доступа

- **Telegram**: User/Chat whitelisting
- **Операции с файлами**: Whitelist директорий
- **Shell команды**: Whitelist команд
- **Subagents**: Лимиты конкурентности

### Управление секретами

- **Переменные окружения**: Рекомендуется для API ключей
- **Права на файлы**: Файлы конфигурации должны быть доступны только пользователю
- **No Hardcoding**: Никогда не коммитить API ключи в репозиторий

## Стратегия тестирования

### Unit тесты

- Каждый компонент тестируется изолированно
- Mock интерфейсы для внешних зависимостей
- Тесты валидации конфигурации

### Интеграционные тесты

- Поток message bus
- Выполнение инструментов
- Создание subagent
- Выполнение cron задач

### End-to-End тесты

- Полный workflow агента
- Несколько инструментов вместе
- Взаимодействие subagent + главный агент

## Будущая архитектура

### Планируемые улучшения

1. **Слой базы данных** (v0.5)
   - SQLite хранилище для персистентности
   - История сессий
   - Функциональность backup/restore

2. **Поддержка MCP** (v1.1)
   - MCP клиент для внешних инструментов
   - Управление MCP серверами
   - Обёртывание MCP инструментов

3. **Web UI** (v0.4)
   - REST API для внешнего доступа
   - Dashboard для мониторинга
   - Конфигурационный UI

4. **Больше каналов** (v0.4)
   - Discord коннектор
   - Slack коннектор
   - Email уведомления

## Ссылки

### Документация компонентов v0.2

- [Cron Scheduler Architecture](architecture/cron_scheduler.md) — Полная документация планировщика задач
- [Subagent Manager Architecture](architecture/subagent_manager.md) — Архитектура управления subagents
- [Worker Pool Architecture](architecture/worker_pool.md) — Архитектура пула воркеров
- [HEARTBEAT System Architecture](architecture/heartbeat_system.md) — Архитектура системы проверок

### Внешние ресурсы

- [Message Bus Design](https://martinfowler.com/articles/patterns-of-distributed-systems/MessageBus.html)
- [Go Concurrency](https://go.dev/doc/effective_go#concurrency)
- [robfig/cron](https://pkg.go.dev/github.com/robfig/cron/v3)
- [Context Package](https://pkg.go.dev/context)

---

**Последнее обновление:** v0.2.0 Архитектура (обновлено для компонентов Cron Scheduler, Subagent Manager, Worker Pool, HEARTBEAT System)
