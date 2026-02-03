# Nexbot - Ultra-Lightweight Personal AI Agent

**Nexbot** — self-hosted ИИ-агент на Go для управления цифровыми потоками задач через Telegram с LLM-провайдером Z.ai (GLM-4.7) и навыками (skills).

## Features

- 🤖 **Ultra-lightweight** (~8-10K строк кода)
- 🔌 **Telegram connector** — общение через Telegram бота
- 🧠 **Z.ai LLM** — GLM-4.7 Flash для быстрых ответов
- 🛠️ **Tool calling** — встроенные инструменты (файлы, shell)
- 📚 **Skills system** — расширяемые навыки (OpenClaw compatible)
- 💾 **Workspace** — AGENTS.md, SOUL.md, USER.md, TOOLS.md, IDENTITY.md
- 📝 **Session management** — история диалогов
- 🚌 **Message bus** — декомпозированная архитектура
- 🚀 **Single binary** — Linux/macOS/Windows

## Quick Start

### 1. Install

```bash
# Clone repository
git clone https://github.com/aatumaykin/nexbot.git
cd nexbot

# Or build from source
make build
```

### 2. Configure

```bash
# Copy example config
cp config.example.toml config.toml

# Copy example env
cp .env.example .env

# Edit .env and add your API keys
nano .env
```

**Required variables:**
- `ZAI_API_KEY` — API ключ от [Z.ai](https://z.ai)
- `TELEGRAM_BOT_TOKEN` — токен Telegram бота от [@BotFather](https://t.me/BotFather)

### 3. Run

```bash
# Start the bot
./nexbot

# Or from source
make run
```

## Configuration

See `config.example.toml` for all available configuration options.

```toml
[agent]
model = "glm-4.7-flash"
max_tokens = 8192

[llm.zai]
api_key = "${ZAI_API_KEY:}"

[channels.telegram]
token = "${TELEGRAM_BOT_TOKEN:}"
allowed_users = []
```

## Skills

Skills are markdown files that teach Nexbot how to use specific tools or perform certain tasks.

```
skills/
├── weather/
│   └── SKILL.md
└── github/
    └── SKILL.md
```

Skills use YAML frontmatter with markdown body for defining agent capabilities.

## Bootstrap Files

Nexbot uses bootstrap files in your workspace (`~/.nexbot/`):

- `IDENTITY.md` — Core identity of the bot
- `AGENTS.md` — Agent instructions
- `SOUL.md` — Bot personality
- `USER.md` — User profile
- `TOOLS.md` — Tools reference
- `MEMORY.md` — Long-term memory

## CLI Commands

```bash
nexbot serve              # Start Nexbot agent (main command)
nexbot run                # Start Nexbot agent
nexbot config validate    # Validate configuration file
nexbot test               # Test Nexbot components
nexbot version            # Print version information
nexbot --help             # Show help
nexbot --version          # Show version
```

## Building

```bash
# Build for current platform
make build

# Build for all platforms
make build-all

# Run tests
make test

# Install to /usr/local/bin
make install
```

## Roadmap

### v0.1.0 (Current) — MVP
- Telegram connector
- Z.ai LLM provider
- Tool calling
- Workspace system
- Skills loader
- Session manager

### v0.2.0 — Cron + Spawn
- Cron scheduler
- Subagent manager (spawn tool)
- Heartbeat tasks

### v0.3.0 — Web Search
- Brave Search API
- Web fetch tool
- URL summarization

### v0.4.0 — More Channels
- Discord connector
- Web UI
- Multi-channel routing

### v0.5.0 — SQLite Migration
- SQLite storage
- Migration from markdown
- Backup/restore

### v1.1.0 — Full MCP
- MCP client
- MCP server management
- MCP tools wrapping

## Architecture

Nexbot uses a simple loop + message bus architecture:

```
Telegram ──► Inbound Queue ──► Agent ──► Outbound Queue ──► Telegram
            (bus)                          (bus)
```

Key components:
- **Message Bus** - Async queue for inbound/outbound messages
- **Agent Loop** - Processes messages with LLM and tool calling
- **Channels** - Connectors for Telegram (extensible to other platforms)
- **Tools** - Built-in tools (read_file, write_file, list_dir, shell_exec)
- **Skills** - Extensible markdown-based skills system
- **Workspace** - Directory structure for agent context (IDENTITY.md, AGENTS.md, SOUL.md, etc.)

## Contributing

Contributions are welcome! Please follow these guidelines:

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

### Development

```bash
# Run tests
make test

# Run with coverage
make test-cover

# Format code
make fmt

# Run linter
make lint

# Run all CI checks
make ci
```

## License

MIT License — see [LICENSE](LICENSE) for details.

---

**Inspired by:** [nanobot](https://github.com/HKUDS/nanobot) and [Nexflow](https://github.com/aatumaykin/nexflow)
