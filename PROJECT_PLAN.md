# Nexbot - Project Plan (Updated with Z.ai Coding API)

## Vision

**Nexbot** — ultra-lightweight self-hosted ИИ-агент на Go (~8–10K строк кода) с message bus архитектурой, расширяемыми каналами и навыками. Вдохновлён nanobot, но с чистой архитектурой и фокусом на простоту.

---

## Architecture

```

┌─────────────────────────────────────────────────────────────┐
│              Telegram Connector (telego)                    │
│         Long polling / Webhook (future)                     │
└─────────────────────────────────────────────────────────────┘
│
▼
┌─────────────────────────────────────────────────────────────┐
│                        Message Bus                          │
│   InboundQueue (chan) ──► OutboundQueue (chan)              │
└─────────────────────────────────────────────────────────────┘
│
▼
┌─────────────────────────────────────────────────────────────┐
│                     Simple Agent Loop                       │
│  ┌──────────────┐  ┌──────────────┐  ┌─────────┐          │
│  │ Context      │  │ Tool         │  │  LLM    │          │
│  │ Builder      │  │ Registry     │  │ Provider│          │
│  └──────────────┘  └──────────────┘  └─────────┘          │
│  ┌──────────────────────────────────────────────┐          │
│  │               Skills Loader                  │          │
│  └──────────────────────────────────────────────┘          │
│  ┌────────────────────────────────────────────────────────┐│
│  │         Subagent Manager (v0.2)                        ││
│  └────────────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────────────┘
│
▼
┌─────────────────────────────────────────────────────────────┐
│                       Storage Layer                         │
│      Workspace (Markdown) │ SQLite (future v0.5)            │
└─────────────────────────────────────────────────────────────┘

```

---

## Project Structure

```text
nexbot/
├── cmd/
│   └── nexbot/
│       └── main.go                 # Entry point
├── internal/
│   ├── agent/
│   │   ├── loop.go                 # Simple agent loop (core)
│   │   ├── context.go              # System prompt builder
│   │   ├── memory.go               # Memory store (markdown/JSONL)
│   │   ├── session.go              # Session manager
│   │   └── tools.go                # Tool registry
│   ├── bus/
│   │   ├── events.go               # Event types (Inbound/Outbound)
│   │   └── queue.go                # Async message queue
│   ├── channels/
│   │   ├── connector.go            # Connector interface
│   │   └── telegram/
│   │       └── connector.go        # Telegram implementation (telego)
│   ├── llm/
│   │   ├── provider.go             # LLM provider interface
│   │   ├── zai.go                  # Z.ai Coding API implementation
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
│   │   ├── config.go               # TOML config parsing
│   │   └── schema.go               # Config structs
│   └── logger/
│       └── logger.go               # slog wrapper
├── pkg/
│   └── messagebus/                 # Public message bus (для расширений)
├── workspace/                      # Default workspace
│   ├── AGENTS.md
│   ├── SOUL.md
│   ├── USER.md
│   ├── TOOLS.md
│   ├── IDENTITY.md
│   ├── HEARTBEAT.md                # Heartbeat tasks (v0.2)
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

## Z.ai Coding API Assumptions

- Основной endpoint для Nexbot:
`https://api.z.ai/api/coding/paas/v4`
(специальный Coding API, а не общий `.../api/paas/v4`).
- Конфиг должен позволять при желании переопределить `api_base`, но по умолчанию использовать Coding API.
- Аутентификация: `Authorization: Bearer <ZAI_API_KEY>`.
- Модель по умолчанию: `glm-4.7` (плюс возможность указать альтернативную/дешёвую модель).

---

## MVP Roadmap (Week 1–2, Updated v0.1.0)

### Цель v0.1.0

Один бинарь `nexbot`, который поднимает Telegram‑бота, общается с Z.ai Coding API (`https://api.z.ai/api/coding/paas/v4`), использует OpenAI‑совместимое tool calling и умеет через workspace/skills и встроенные tools выполнять базовые операции (файлы + whitelisted shell).

### Week 1: Core Foundation + Early Z.ai POC

#### Day 1–2: Config, Logger, Message Bus, CLI

 - [x] Initialize Go module (go mod init)
 - [x] Setup directory structure
 - [x] Config: TOML parser (config.example.toml)
 - [x] Logger: slog wrapper (internal/logger/logger.go)
 - [x] Message Bus:
     - [x] Events: InboundMessage, OutboundMessage structs (internal/bus/events.go)
     - [x] Queue: async channels with context support (internal/bus/queue.go)
     - [x] Publish/consume methods
 - [x] CLI: basic `nexbot serve` (cmd/nexbot/main.go)
 - [x] Makefile: build targets
 - [x] .gitignore: Go specific

Конфиг (идея):

```toml
[llm.zai]
api_base = "https://api.z.ai/api/coding/paas/v4"
api_key  = "ZAI_API_KEY"
model    = "glm-4.7"
timeout_seconds = 30
```


#### Day 3: Early Z.ai POC (plain chat, без tools)

- [ ] Provider interface definition (internal/llm/provider.go)
- [ ] Минимальный Z.ai client к `.../coding/paas/v4/chat/completions`:
    - один метод Chat без tool_calls,
    - маппинг config → HTTP headers/URL.
- [ ] Простейший прототип: команда/тест, который шлёт запрос и печатает ответ.
- [ ] Зафиксировать формат сообщений, latency, типичные ошибки.


#### Day 4: Workspace / Bootstrap (3a)

- [ ] Workspace manager (path, creation) (internal/workspace/workspace.go)
- [ ] Bootstrap files loader (AGENTS.md, SOUL.md, USER.md, TOOLS.md, IDENTITY.md) (internal/workspace/bootstrap.go)
- [x] Template files for bootstrap (workspace/*.md)
- [ ] Tests: базовые happy‑path тесты workspace/bootstrap.


#### Day 5: Memory + Context (3b)

- [ ] Memory store (markdown/JSONL files) (internal/agent/memory.go)
- [ ] Context builder (system prompt assembly, порядок: IDENTITY → AGENTS → SOUL → USER → TOOLS → memory) (internal/agent/context.go)
- [ ] Session manager (JSONL, простые операции append/read) (internal/agent/session.go)
- [ ] Tests: базовые тесты memory/context.


### Week 2: Agent Loop, Telegram, Tools, Skills

#### Day 6: Agent Loop (без tools)

- [ ] Simple loop: inbound → build context → LLM → outbound (internal/agent/loop.go)
- [ ] Интеграция с session manager (get_or_create, add_message, save).
- [ ] Mock Provider для graceful degradation:
    - эхо‑ответ, фиксированный ответ или fixtures.
- [ ] Tests: интеграционный тест loop’а с mock‑provider.


#### Day 7: Telegram Connector (plain chat, без tools)

- [ ] Telego integration (`github.com/mymmrac/telego`).
- [ ] Bot initialization with token from config (internal/channels/telegram/connector.go).
- [ ] Inbound: parse Telegram update → InboundMessage → bus.PublishInbound.
- [ ] Outbound: bus.SubscribeOutbound → Telegram `sendMessage`.
- [ ] Middleware: whitelist users (allowed_users from config).
- [ ] Long polling setup (`updatesViaLongPolling`).
- [ ] Graceful shutdown handling.
- [ ] Tests: mock Telegram bot, message parsing.

Результат: рабочий Telegram‑бот, использующий Z.ai Coding API без tools.

#### Day 8: Tool Calling Infrastructure + First Tool

- [ ] Расширение Provider под tool calling (SupportsToolCalling, schema форматы).
- [ ] Поддержка OpenAI-compatible tool calling (минимальный subset `tools` + `tool_calls`).
- [ ] Tool interface \& registry (internal/tools/registry.go).
- [ ] Первый built-in tool:
    - `read_file` — read file content (internal/tools/file.go).
- [ ] Tool schema generation для `read_file`.
- [ ] Интеграция в agent loop: распознавать tool_calls и вызывать `read_file`.
- [ ] Tests: tool schema + выполнение `read_file` в loop’е (можно с mock‑provider).


#### Day 9: Остальные Tools + Simplified Skills

- Tools:
    - [ ] `write_file` — write file content (internal/tools/file.go),
    - [ ] `list_dir` — list directory contents (internal/tools/file.go),
    - [ ] `shell_exec` — execute shell command с whitelist (internal/tools/shell.go).
- [ ] Tool execution с timeout и простым working_dir (из конфига или константа).
- Skills (простая версия):
    - [ ] SKILL.md parser (YAML frontmatter + markdown body) (internal/skills/parser.go).
    - [ ] Skills loader (workspace/skills/ + builtin/skills/) (internal/skills/loader.go).
    - [ ] Модель: все skills → один summary‑блок в system prompt.
    - [ ] Example skills (weather basics, file operations guide).
- [ ] Tests: parser, loader, basic tool tests.


#### Day 10: Integration \& Polish (E2E с tools)

- [ ] End-to-end integration:
    - Start Telegram bot.
    - User sends message.
    - Bus routes to agent.
    - Agent builds context (bootstrap + history).
    - Call Z.ai Coding API (`/coding/paas/v4/chat/completions`).
    - Execute tool calls (`read_file`, `write_file`, `list_dir`, `shell_exec`) при необходимости.
    - Send response back via Telegram.
- [ ] Config validation (check required fields, API keys).
- [ ] Error messages (user-friendly, masked secrets).
- [ ] Documentation:
    - README.md (overview, features, installation),
    - QUICKSTART.md (5-minute setup guide),
    - CONFIG.md (configuration reference).
- [ ] Makefile targets: build-all (Linux/macOS/Windows), release (checksums).
- [ ] v0.1.0 release (GitHub release с бинарями).

---

## Dependencies (Go)

| Компонент | Библиотека |
| :-- | :-- |
| Config | github.com/BurntSushi/toml |
| Logger | log/slog (Go 1.21+) |
| HTTP Client | net/http + retryablehttp |
| Telegram Bot | github.com/mymmrac/telego |
| YAML Parser | gopkg.in/yaml.v3 |
| Testing | github.com/stretchr/testify |
| CLI Flags | github.com/spf13/cobra |


---

## Success Criteria (v0.1.0, Updated)

- Telegram bot подключается к Z.ai Coding API (`https://api.z.ai/api/coding/paas/v4`) и ведёт диалог без tools.
- Tool calling в OpenAI-compatible формате включён и поддерживает:
    - минимум `read_file`,
    - затем базовый набор (`write_file`, `list_dir`, `shell_exec`).
- Workspace система с bootstrap файлами (IDENTITY.md, AGENTS.md, SOUL.md, USER.md, TOOLS.md) и простым memory (JSONL/markdown).
- Skills loader в упрощённом варианте:
    - все skills попадают в один summary‑блок в system prompt.
- Tool registry с базовыми операциями (file operations, shell execution с whitelist).
- Session manager хранит историю диалогов в JSONL (user/assistant/tool_calls).
- Message bus декомпозирует каналы от agent loop (inbound/outbound очереди).
- Single binary как минимум для Linux/amd64 и macOS/amd64/arm64 (остальные платформы — по возможности).
- Время деплоя ≤ 10 минут (git clone → build → configure → run).
- Время ответа ≤ 5 секунд для простых запросов (без тяжёлых tools).
- Базовые тесты на ключевые модули (~60% coverage).
- README.md с quickstart guide, базовой архитектурой и примером конфига.
- Makefile с целями: `build-all`, `test`, `release`.

---

## Post-MVP Roadmap (без Web UI)

### v0.2.0 - Cron + Spawn (2–3 недели)

- Cron scheduler (robfig/cron/v3).
- Cron commands: `nexbot cron add`, `nexbot cron list`, `nexbot cron remove`.
- Subagent manager (spawn tool implementation).
- Background task execution (async workers).
- Spawn tool registration.
- HEARTBEAT.md support (proactive tasks).
- Tests: cron scheduling, subagent coordination.


### v0.3.0 - Web Search + More Tools (2–3 недели)

- Brave Search API integration (новый skill).
- Web fetch tool (http client).
- URL summarization skill.
- More built-in tools (git, process info).
- Tests: web search, HTTP fetching.


### v0.4.0 - More Channels (2–3 недели)

- Discord connector (disgo или discordgo).
- Channel manager (register multiple channels).
- Multi-channel routing (user identity across channels).
- Tests: connector integration, routing.


### v0.5.0 - SQLite Migration (2–3 недели)

- SQLite integration (modernc.org/sqlite).
- Migration from markdown (workspace → SQLite).
- Query builder для сессий и памяти.
- Backup/restore commands.
- Tests: DB operations, migration.


### v1.0.0 - Production Ready (3–4 недели)

- Enhanced observability (structured logs, metrics).
- Health checks (LLM, channels, storage).
- Configuration validation (schema validation).
- Error handling improvements (retries, backoff).
- Performance optimizations (caching, connection pooling).
- Security improvements (input validation, secret masking).
- Documentation completeness (API docs, contribution guide).
- Tests: 80%+ coverage, integration tests.
- CI/CD (GitHub Actions).
- Release notes, changelog.


### v1.1.0 - Full MCP Support (3–4 недели)

- MCP client implementation.
- MCP server management (connect, disconnect, list).
- MCP servers support: filesystem, github, search, database.
- MCP tool wrapping (automatic registration).
- MCP resource management (resources, prompts).
- Tests: MCP protocol compliance.
- Documentation: MCP integration guide.

---

## Key Design Decisions (Recap)

1. **Simple Loop Architecture**: минималистичный agent loop с чистым разделением ответственности.
2. **Message Bus Decoupling**: каналы → bus → agent → bus → каналы.
3. **Simplified Skills Loading для v0.1.0**: все skills → summary‑блок; более сложная логика позже.
4. **Bootstrap Files Priority**: IDENTITY.md → AGENTS.md → SOUL.md → USER.md → TOOLS.md → memory.
5. **Z.ai Coding API as Primary Provider**: использовать `https://api.z.ai/api/coding/paas/v4` и модель `glm-4.7`.
6. **Markdown Storage First**: простой формат для MVP, SQLite позже.
7. **Single Binary Distribution**: Go компиляция для основных платформ.
8. **Extensible Channels**: Connector interface для будущих интеграций (Discord и др.).
9. **Incremental Delivery**: сначала plain chat без tools, затем tool calling и skills.