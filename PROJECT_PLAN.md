# Nexbot - Project Plan

## Vision

**Nexbot** — ultra-lightweight self-hosted ИИ-агент на Go (~8-10K строк кода) с message bus архитектурой, расширяемыми каналами и навыками. Вдохновлён nanobot, но с чистой архитектурой и фокусом на простоту.

---

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│              Telegram Connector (telego)                  │
│         Long polling / Webhook (future)                  │
└─────────────────────────────────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────┐
│                    Message Bus                           │
│   InboundQueue (chan) ──► OutboundQueue (chan)         │
└─────────────────────────────────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────┐
│                   Simple Agent Loop                       │
│  ┌──────────────┐  ┌──────────────┐  ┌─────────┐  │
│  │ Context      │  │ Tool         │  │  LLM    │  │
│  │ Builder      │  │ Registry     │  │Provider  │  │
│  └──────────────┘  └──────────────┘  └─────────┘  │
│  ┌──────────────────────────────────────────────┐         │
│  │            Skills Loader                │         │
│  └──────────────────────────────────────────────┘         │
│  ┌──────────────────────────────────────────────┐         │
│  │         Subagent Manager (v0.2)          │         │
│  └──────────────────────────────────────────────┘         │
└─────────────────────────────────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────┐
│                   Storage Layer                          │
│      Workspace (Markdown) │ SQLite (future v0.5)       │
└─────────────────────────────────────────────────────────────┘
```

---

## Project Structure

```
nexbot/
├── cmd/
│   └── nexbot/
│       └── main.go                 # Entry point
├── internal/
│   ├── agent/
│   │   ├── loop.go                 # Simple agent loop (core)
│   │   ├── context.go              # System prompt builder
│   │   ├── memory.go               # Memory store (markdown)
│   │   ├── session.go              # Session manager
│   │   └── tools.go                # Tool registry
│   ├── bus/
│   │   ├── events.go               # Event types (Inbound/Outbound)
│   │   └── queue.go                # Async message queue
│   ├── channels/
│   │   ├── connector.go             # Connector interface
│   │   └── telegram/
│   │       └── connector.go        # Telegram implementation (telego)
│   ├── llm/
│   │   ├── provider.go             # LLM provider interface
│   │   ├── zai.go                  # Z.ai (GLM) implementation
│   │   └── openai.go               # OpenAI-compatible (future)
│   ├── skills/
│   │   ├── loader.go               # Skills loader
│   │   ├── parser.go               # SKILL.md parser (OpenClaw format)
│   │   └── metadata.go             # Skill metadata
│   ├── tools/
│   │   ├── registry.go             # Tool registry
│   │   ├── file.go                 # File operations
│   │   └── shell.go                # Shell execution
│   ├── workspace/
│   │   ├── workspace.go            # Workspace manager
│   │   └── bootstrap.go            # Bootstrap files loader
│   ├── config/
│   │   ├── config.go                # TOML config parsing
│   │   └── schema.go               # Config structs
│   └── logger/
│       └── logger.go                # slog wrapper
├── pkg/
│   └── messagebus/                 # Public message bus (для расширений)
├── workspace/                       # Default workspace
│   ├── AGENTS.md
│   ├── SOUL.md
│   ├── USER.md
│   ├── TOOLS.md
│   ├── IDENTITY.md
│   ├── HEARTBEAT.md              # Heartbeat tasks (v0.2)
│   └── memory/
│       └── MEMORY.md
├── skills/
│   └── examples/
│       └── example-skill/
│           └── SKILL.md
├── config.example.toml
├── .env.example
├── go.mod
├── Makefile
└── README.md
```

---

## Core Interfaces

```go
// Connector interface for chat channels
type Connector interface {
    Start(ctx context.Context, inboundCh chan<- InboundMessage) error
    Stop() error
    SendMessage(ctx context.Context, msg OutboundMessage) error
}

// LLM provider interface
type Provider interface {
    Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error)
    SupportsToolCalling() bool
    GetDefaultModel() string
}

// Tool interface
type Tool interface {
    Name() string
    Description() string
    Parameters() map[string]any
    Execute(ctx context.Context, args map[string]any) (string, error)
    ToSchema() map[string]any  // OpenAI function schema
}
```

---

## MVP Roadmap (Week 1-2)

### Week 1: Core Foundation

#### Day 1: Project Setup
- [x] Initialize Go module (go mod init)
- [x] Setup directory structure
- [x] Config: TOML parser (config.example.toml)
- [ ] Logger: slog wrapper (internal/logger/logger.go)
- [ ] CLI: basic commands (cmd/nexbot/main.go)
- [x] Makefile: build targets
- [x] .gitignore: Go specific

#### Day 2: Message Bus
- [ ] Events: InboundMessage, OutboundMessage structs (internal/bus/events.go)
- [ ] Queue: async channels with context support (internal/bus/queue.go)
- [ ] Bus: publish/consume methods
- [ ] Tests: message flow unit tests

#### Day 3: Workspace System
- [ ] Workspace manager (path, creation) (internal/workspace/workspace.go)
- [ ] Bootstrap files loader (AGENTS.md, SOUL.md, USER.md, TOOLS.md, IDENTITY.md) (internal/workspace/bootstrap.go)
- [ ] Memory store (markdown files) (internal/agent/memory.go)
- [ ] Context builder (system prompt assembly) (internal/agent/context.go)
- [x] Template files for bootstrap (workspace/*.md)
- [ ] Tests: workspace creation, bootstrap loading (60% coverage)

#### Day 4: Z.ai LLM Provider
- [ ] Provider interface definition (internal/llm/provider.go)
- [ ] Z.ai client (HTTP client to https://api.z.ai/api/coding/paas/v4) (internal/llm/zai.go)
- [ ] OpenAI-compatible tool calling format (позаимствовать из nanobot)
- [ ] Response parsing (content, tool_calls, usage)
- [ ] Provider factory (select Z.ai by config)
- [ ] Tests: mock Z.ai responses, tool calling (60% coverage)

#### Day 5: Agent Loop
- [ ] Simple loop: inbound → build context → LLM → execute tools → outbound (internal/agent/loop.go)
- [ ] Tool calling support (max_iterations from config)
- [ ] Session integration (get_or_create, add_message, save) (internal/agent/session.go)
- [ ] Error handling (LLM errors, tool errors)
- [ ] Retry logic for transient errors
- [ ] Tests: full loop integration (60% coverage)

### Week 2: Telegram + Tools + Skills

#### Day 6-7: Telegram Connector
- [ ] Telego integration (github.com/mymmrac/telego)
- [ ] Bot initialization with token from config (internal/channels/telegram/connector.go)
- [ ] Inbound: parse Telegram update → InboundMessage → bus.publish_inbound
- [ ] Outbound: bus.subscribe_outbound → Telegram sendMessage
- [ ] Middleware: whitelist users (allowed_users from config)
- [ ] Long polling setup (updatesViaLongPolling)
- [ ] Graceful shutdown handling
- [ ] Tests: mock Telegram bot, message parsing (60% coverage)

#### Day 8: Tool Registry
- [ ] Tool interface & registry (internal/tools/registry.go)
- [ ] Built-in tools:
  - `read_file` - read file content (internal/tools/file.go)
  - `write_file` - write file content (internal/tools/file.go)
  - `list_dir` - list directory contents (internal/tools/file.go)
  - `shell_exec` - execute shell command with whitelist (internal/tools/shell.go)
- [ ] Tool registration
- [ ] Tool execution with context (timeout, working_dir)
- [ ] Tool schema generation (OpenAI function format)
- [ ] Tests: each tool individually (60% coverage)

#### Day 9: Skills System
- [ ] SKILL.md parser (YAML frontmatter + markdown body) (internal/skills/parser.go)
- [ ] Skills loader (workspace/skills/ + builtin/skills/) (internal/skills/loader.go)
- [ ] Progressive loading:
  - `always=true` skills → load full content into system prompt
  - Available skills → summary XML (like nanobot)
- [ ] Skills integration with context builder
- [ ] Example skills (weather basics, file operations guide)
- [ ] Tests: parser, loader, progressive loading (60% coverage)

#### Day 10: Integration & Polish
- [ ] End-to-end integration:
  - Start Telegram bot
  - User sends message
  - Bus routes to agent
  - Agent builds context (bootstrap + history)
  - Call Z.ai LLM
  - Execute tool calls if any
  - Send response back via Telegram
- [ ] Config validation (check required fields, API keys)
- [ ] Error messages (user-friendly, masked secrets)
- [ ] Documentation:
  - README.md (overview, features, installation)
  - QUICKSTART.md (5-minute setup guide)
  - CONFIG.md (configuration reference)
- [ ] Makefile targets: build-all (Linux/macOS/Windows), release (checksums)
- [ ] v0.1.0 release (GitHub release with binaries)

---

## Dependencies (Go)

| Компонент    | Библиотека                     |
| ------------ | ------------------------------ |
| **Config**       | github.com/BurntSushi/toml     |
| **Logger**       | log/slog (Go 1.21+)            |
| **HTTP Client**  | net/http + retryablehttp       |
| **Telegram Bot** | github.com/mymmrac/telego      |
| **YAML Parser**  | gopkg.in/yaml.v3               |
| **Testing**      | github.com/stretchr/testify    |
| **CLI Flags**    | github.com/spf13/cobra         |

---

## Success Criteria (v0.1.0)

- [ ] Telegram bot подключается к Z.ai LLM (coding endpoint)
- [ ] Tool calling работает (OpenAI-compatible формат)
- [ ] Workspace система с bootstrap файлами (IDENTITY.md, AGENTS.md, SOUL.md, USER.md, TOOLS.md)
- [ ] Skills loader (OpenClaw совместимый, progressive loading)
- [ ] Tool registry (file operations, shell execution)
- [ ] Session manager (история диалогов в JSONL формате)
- [ ] Message bus декомпозирует каналы от agent loop
- [ ] Single binary для Linux/amd64, Linux/arm64, macOS/amd64, macOS/arm64, Windows/amd64
- [ ] Время деплоя ≤ 10 минут (git clone → build → configure → run)
- [ ] Время ответа ≤ 5 секунд для простых запросов
- [ ] Базовые тесты (~60% coverage)
- [ ] README.md с quickstart guide
- [ ] Makefile с build-all, test, release

---

## Post-MVP Roadmap

### v0.2.0 - Cron + Spawn (2-3 недели)
- [ ] Cron scheduler (robfig/cronv3)
- [ ] Cron commands: `nexbot cron add`, `nexbot cron list`, `nexbot cron remove`
- [ ] Subagent manager (spawn tool implementation)
- [ ] Background task execution (async workers)
- [ ] Spawn tool registration
- [ ] HEARTBEAT.md support (proactive tasks)
- [ ] Tests: cron scheduling, subagent coordination

### v0.3.0 - Web Search + More Tools (2-3 недели)
- [ ] Brave Search API integration (новый skill)
- [ ] Web fetch tool (http client)
- [ ] URL summarization skill
- [ ] More built-in tools (git, process info)
- [ ] Tests: web search, HTTP fetching

### v0.4.0 - More Channels (2-3 недели)
- [ ] Discord connector (disgo или discordgo)
- [ ] Channel manager (register multiple channels)
- [ ] Multi-channel routing (user identity across channels)
- [ ] Web UI (basic SPA для отладки)
- [ ] Tests: connector integration, routing

### v0.5.0 - SQLite Migration (2-3 недели)
- [ ] SQLite integration (modernc.org/sqlite)
- [ ] Migration from markdown (workspace → SQLite)
- [ ] Query builder для сессий и памяти
- [ ] Backup/restore commands
- [ ] Tests: DB operations, migration

### v1.0.0 - Production Ready (3-4 недели)
- [ ] Enhanced observability (structured logs, metrics)
- [ ] Health checks (LLM, channels, storage)
- [ ] Configuration validation (schema validation)
- [ ] Error handling improvements (retries, backoff)
- [ ] Performance optimizations (caching, connection pooling)
- [ ] Security improvements (input validation, secret masking)
- [ ] Documentation completeness (API docs, contribution guide)
- [ ] Tests: 80%+ coverage, integration tests
- [ ] CI/CD (GitHub Actions)
- [ ] Release notes, changelog

### v1.1.0 - Full MCP Support (3-4 недели)
- [ ] MCP client implementation (https://modelcontextprotocol.io)
- [ ] MCP server management (connect, disconnect, list)
- [ ] MCP servers support: filesystem, github, search, database
- [ ] MCP tool wrapping (automatic registration)
- [ ] MCP resource management (resources, prompts)
- [ ] Tests: MCP protocol compliance
- [ ] Documentation: MCP integration guide

---

## Effort Estimation

| Версия   | Фичи                             | Дни  | Сложность | Риски                                 |
| -------- | -------------------------------- | ---- | --------- | ------------------------------------- |
| **v0.1.0**   | Telegram + Z.ai + Skills + Tools | 10   | Средний   | Z.ai API stability, Telegram webhooks |
| **v0.2.0**   | Cron + Spawn + Heartbeat         | 15   | Средний   | Subagent coordination, Cron accuracy  |
| **v0.3.0**   | Web Search + More Tools          | 15   | Средний   | Brave API rate limits                 |
| **v0.4.0**   | More Channels + Web UI           | 15   | Средний   | Discord API, SPA complexity           |
| **v0.5.0**   | SQLite Migration                 | 10   | Средний   | Data loss during migration            |
| **v1.0.0**   | Production Ready                 | 25   | Высокий   | Performance bottlenecks, bugs         |
| **v1.1.0**   | Full MCP                         | 20   | Высокий   | MCP protocol changes, complexity      |
| **Итого**    |                                  | ~110 |           |                                       |

---

## Key Design Decisions

1. **Simple Loop Architecture**: Минималистичный agent loop как в nanobot, но с чистым разделением ответственности
2. **Message Bus Decoupling**: Каналы → bus → agent → bus → каналы
3. **Progressive Skills Loading**: Always-loaded skills в system prompt, available skills только summary
4. **Bootstrap Files Priority**: IDENTITY.md → AGENTS.md → SOUL.md → USER.md → TOOLS.md → memory
5. **Z.ai as Primary Provider**: GLM-4.7 Flash для скорости и экономии, fallback на GLM-4.6
6. **Markdown Storage First**: Простой формат для MVP, SQLite позже
7. **Single Binary Distribution**: Go компиляция для всех платформ
8. **Extensible Channels**: Connector interface для будущих интеграций (Discord, Web UI)

---

## Clarifications (from user)

1. ✅ **Z.ai API**: Использовать coding endpoint `https://api.z.ai/api/coding/paas/v4`
2. ✅ **Tool calling**: OpenAI-compatible формат (позаимствовать у nanobot)
3. ✅ **Brave Search**: Вынести в v0.3.0 (не в MVP)
4. ✅ **MCP**: v1.1.0 (не в MVP)
5. ✅ **Testing**: Базовый (~60% coverage)
6. ✅ **Documentation**: Только Markdown

---

## Event Bus Explanation

### What is Event Bus?

Event Bus (Message Bus) — это архитектурный паттерн для **декомпозиции компонентов** через асинхронную передачу сообщений.

### How it works

```
Telegram ──► Inbound Queue ──► Agent ──► Outbound Queue ──► Telegram
            (bus)                          (bus)
```

### Key Benefits

1. **Decoupling**: Компоненты не зависят друг от друга
2. **Async Processing**: Неблокирующая коммуникация между компонентами
3. **Scalability**: Легко добавлять новые каналы/агенты
4. **Error Isolation**: Ошибка в одном компоненте не валит систему
5. **Testing**: Легко mock'ать bus для тестов

### Implementation in Go

```go
type MessageBus struct {
    inbound     chan InboundMessage
    outbound    chan OutboundMessage
    subscribers map[string][]func(OutboundMessage)
    running     bool
}

// Publish (отправка)
func (b *MessageBus) PublishInbound(msg InboundMessage) {
    b.inbound <- msg
}

// Consume (забор)
func (b *MessageBus) ConsumeInbound(ctx context.Context) (InboundMessage, error) {
    select {
    case msg := <-b.inbound:
        return msg, nil
    case <-ctx.Done():
        return InboundMessage{}, ctx.Err()
    }
}
```

---

## Next Steps

План готов к реализации. Можно начать с Phase 1:

1. Создать `cmd/nexbot/main.go` (entry point)
2. Создать `internal/logger/logger.go` (slog wrapper)
3. Создать `internal/bus/events.go` (event types)
4. Создать `internal/bus/queue.go` (message queue)
5. Создать `internal/config/schema.go` (config structs)
6. Создать `internal/config/config.go` (config loader)
7. Продолжить по плану...

---

**Status**: ✅ Project initialized, ready for development
