# Subagent Manager Architecture

## Overview

Subagent Manager — это компонент для создания и управления изолированными агентами (subagents), каждый с собственной сессией и памятью. Это позволяет выполнять задачи параллельно, сохраняя разделение ответственности и контекста между различными агентами.

## Components

### 1. Manager (`internal/agent/subagent/manager.go`)

**Responsibilities:**
- Создание новых subagents с изолированными сессиями
- Управление жизненным циклом subagents
- Thread-safe операции с помощью mutex
- Интеграция с Session Manager и Loop

**Key Fields:**
```go
type Manager struct {
    subagents   map[string]*Subagent // Registry of active subagents
    mu          sync.RWMutex         // Mutex for thread-safety
    loopFactory func() *loop.Loop    // Factory for creating new loops
    sessionMgr  *session.Manager     // Session manager
    logger      *logger.Logger       // Logger instance
}
```

**Key Methods:**
- `Spawn(ctx, parentSession, task)` — создание нового subagent
- `Stop(id)` — остановка subagent
- `List()` — список всех активных subagents
- `Get(id)` — получение subagent по ID
- `StopAll()` — остановка всех subagents
- `Count()` — количество активных subagents

### 2. Subagent (`internal/agent/subagent/manager.go`)

**Responsibilities:**
- Представляет изолированный экземпляр агента
- Обрабатывает задачи через свой Loop
- Управляет собственным контекстом и сессией

**Key Fields:**
```go
type Subagent struct {
    ID      string             // Unique subagent ID (UUID)
    Session string             // Session ID for this subagent
    Loop    *loop.Loop         // Agent loop for processing
    Context context.Context    // Context for lifecycle management
    Cancel  context.CancelFunc // Cancel function for graceful shutdown
    Logger  *logger.Logger     // Logger for this subagent
}
```

**Key Methods:**
- `Process(ctx, task)` — обработка задачи через Loop

### 3. Storage (`internal/agent/subagent/session.go`)

**Responsibilities:**
- Изолированное хранение сессий subagents
- Управление директориями для каждого subagent
- JSONL формат для хранения записей сессий

**Key Methods:**
- `Save(subagentID, entry)` — сохранение записи сессии
- `Load(subagentID)` — загрузка записей сессии
- `Delete(subagentID)` — удаление всех данных subagent
- `List()` — список всех subagent IDs

**Storage Structure:**
```
~/.nexbot/sessions/subagents/
  ├── <subagent_id_1>/
  │   └── session.jsonl
  ├── <subagent_id_2>/
  │   └── session.jsonl
  └── ...
```

## Flow Diagrams

### Subagent Spawn Flow

```
┌─────────────┐
│  Spawn()    │
└──────┬──────┘
       │
       ▼
┌─────────────────┐
│ Lock Mutex      │
└──────┬──────────┘
       │
       ▼
┌─────────────────────┐
│ Generate IDs        │
│ - UUID (subagent)   │
│ - subagent-<ts>     │
│   (session)         │
└──────┬──────────────┘
       │
       ▼
┌─────────────────────┐
│ Create Context      │
│ (WithCancel)        │
└──────┬──────────────┘
       │
       ▼
┌─────────────────────┐
│ Create Loop         │
│ (loopFactory)       │
└──────┬──────────────┘
       │
       ▼
┌─────────────────────┐
│ Create Subagent     │
│ struct              │
└──────┬──────────────┘
       │
       ▼
┌─────────────────────┐
│ Store in Registry   │
│ (subagents map)     │
└──────┬──────────────┘
       │
       ▼
┌─────────────────────┐
│ Unlock Mutex        │
└──────┬──────────────┘
       │
       ▼
┌─────────────────────┐
│ Return Subagent     │
└─────────────────────┘
```

### Subagent Task Processing Flow

```
┌─────────────────────┐
│ Process(task)       │
└──────┬──────────────┘
       │
       ▼
┌─────────────────────┐
│ Check Context       │
│ Deadline?           │
└──────┬──────────────┘
       │
       ▼
    ┌──┴──┐
    │ No  │ Yes
    ▼      ▼
┌─────────────────┐  ┌─────────────────────┐
│ WithTimeout     │  │ Use Existing       │
│ (5 min default)│  │ Context            │
└────┬────────────┘  └─────────────────────┘
     │
     ▼
┌─────────────────────┐
│ Loop.Process()      │
│ (Session, Task)     │
└──────┬──────────────┘
       │
       ▼
┌─────────────────────┐
│ Process in Loop     │
│ (Context, Session,  │
│  Task)              │
└──────┬──────────────┘
       │
       ▼
┌─────────────────────┐
│ Return Response     │
│ or Error            │
└─────────────────────┘
```

### Subagent Stop Flow

```
┌─────────────┐
│  Stop(id)   │
└──────┬──────┘
       │
       ▼
┌─────────────────┐
│ Lock Mutex      │
└──────┬──────────┘
       │
       ▼
┌─────────────────────┐
│ Find Subagent       │
│ (in registry)      │
└──────┬──────────────┘
       │
       ▼
    ┌──┴──┐
    │Found│ Not Found
    ▼      ▼
┌─────────────┐  ┌─────────────────────┐
│ Cancel()    │  │ Return Error        │
│ (Context)   │  └─────────────────────┘
└────┬────────┘
     │
     ▼
┌─────────────────────┐
│ Delete from         │
│ Registry            │
└──────┬──────────────┘
       │
       ▼
┌─────────────────────┐
│ Unlock Mutex        │
└──────┬──────────────┘
       │
       ▼
┌─────────────────────┐
│ Return Success      │
└─────────────────────┘
```

### Subagent Storage Flow

```
┌─────────────────────┐
│ Save Entry          │
└──────┬──────────────┘
       │
       ▼
┌─────────────────────┐
│ Create Directory    │
│ (<base>/<id>/)      │
└──────┬──────────────┘
       │
       ▼
┌─────────────────────┐
│ Write to JSONL     │
│ (session.jsonl)     │
└──────┬──────────────┘
       │
       ▼
┌─────────────────────┘
```

## Subagent Lifecycle

```
[Spawning]
    │
    │ Spawn(ctx, parent, task)
    ▼
[Created] ──┐
    │       │ Process(task)
    │       ▼
    │   [Processing] ──(complete)──> [Idle]
    │       │
    │       └────────(error)─────────> [Error]
    │
    │ Stop(id)
    ▼
[Stopping] ──(context cancelled)──> [Stopped] ──(delete from registry)──> [Removed]
```

## Configuration

### Config Structure

```go
type Config struct {
    SessionDir string         // Directory for storing subagent sessions
    Logger     *logger.Logger // Logger for manager operations
    LoopConfig loop.Config    // Configuration for creating new loops
}
```

### Session ID Format

```
subagent-<timestamp_nanoseconds>
```

### Storage Path

```
~/.nexbot/sessions/subagents/<subagent_id>/session.jsonl
```

## Coordination

### Thread Safety

Все операции с Manager защищены `sync.RWMutex`:
- `Spawn()` — exclusive lock
- `Stop()` — exclusive lock
- `List()` — shared lock
- `Get()` — shared lock
- `Count()` — shared lock

### Parent-Child Relationship

Каждый subagent связан с родительской сессией:
```go
func Spawn(ctx context.Context, parentSession string, task string)
```

Это позволяет:
- Отслеживать происхождение задачи
- Управлять цепочкой выполнения
- Логировать родительскую сессию

## Integration Points

1. **Session Manager** — управление сессиями subagents
2. **Loop** — основной цикл обработки сообщений
3. **Storage** — персистентное хранение в изолированных директориях
4. **Context** — управление жизненным циклом

## Error Handling

- **Invalid configuration** — ошибка при создании Manager
- **Subagent not found** — ошибка при Stop/Get
- **Loop creation failure** — panic (should not happen with valid config)
- **Task processing error** — возвращается из Process()

## Concurrency Model

### Parallel Execution

Каждый subagent работает в отдельном контексте:
- Независимая обработка задач
- Изолированная память и сессия
- Отдельные goroutine для каждого Loop

### Resource Management

- Context cancellation для graceful shutdown
- Mutex для thread-safe доступа к registry
- Isolated storage для каждого subagent

## Use Cases

1. **Parallel task execution** — несколько задач одновременно
2. **Isolated workspaces** — разделение контекста между задачами
3. **Background processing** — фоновые задачи в отдельных агентах
4. **Hierarchical agent structure** — parent-child отношения

## Best Practices

1. Всегда вызывайте `Stop()` для завершения subagent
2. Используйте `StopAll()` для graceful shutdown
3. Логируйте parent session для отслеживания происхождения
4. Управляйте количеством активных subagents через `Count()`
5. Обрабатывайте ошибки из `Process()`

## Limitations

- Нет встроенной очереди задач для subagents
- Нет приоритизации задач
- Нет автоматического масштабирования
- Storage реализован частично (TODO)
