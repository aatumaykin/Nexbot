# Nexbot Project Rules

**Project:** Self-hosted AI agent on Go for task management via Telegram with Z.ai LLM provider and skills system.

## Stack
- Go 1.25.5+
- LLM Provider: Z.ai (GLM-4.7 Flash)
- Telegram: telego library
- Config: TOML
- Architecture: Simple loop + message bus

## Directory Structure
```
cmd/nexbot/          — Entry point
internal/
  agent/             — Core agent logic (loop, context, session, memory, tools)
  bus/               — Message bus (queue, events)
  channels/          — Connectors (telegram)
  config/            — Configuration
  llm/               — LLM providers (zai, mock)
  logger/            — Logging
  skills/            — Skills system
  tools/             — Tools (file, shell)
  workspace/         — Workspace management
pkg/                  — Exported packages
skills/               — External skills (OpenClaw compatible)
workspace/            — Bootstrap files (~/.nexbot/)
```

## Lazy Loading Rules

**READ FIRST (MANDATORY):**
- `docs/rules/security.md` — CRITICAL. Mandatory for ALL changes
- `docs/rules/projectrules.md` — General project rules

**READ BY TASK:**
- `docs/rules/architecture.md` — Architecture tasks (layers, dependencies)
- `docs/rules/codequality.md` — Code style and quality
- `docs/rules/apidesign.md` — API/Web tasks
- `docs/rules/testing.md` — Testing rules

## Key Concepts

**Workspace:**
- Location: `~/.nexbot/`
- Bootstrap files: IDENTITY.md, AGENTS.md, SOUL.md, USER.md, TOOLS.md, MEMORY.md
- Context builder order: IDENTITY → AGENTS → SOUL → USER → TOOLS → memory

**Tool Calling:**
- Tools registered via `tools.Registry`
- Tool schemas converted to `llm.ToolDefinition`
- Agent processes tool calls recursively (max 10 iterations)

**Skills:**
- Location: `skills/` directory
- Format: Directory with `SKILL.md`
- Structure: YAML frontmatter + markdown body
- OpenClaw compatible

## Инструменты агента

**cron** — планировщик задач с tool/payload
- Пример: `{"tool": "send_message", "payload": {"session_id": "abc123", "message": "Reminder"}}`

**send_message** — отправка сообщения через каналы (Telegram)
- Пример: `{"session_id": "telegram:123456789", "message": "Hello"}`

**shell_exec** — выполнение shell команд с ограничениями безопасности
- Описание: Execute shell commands with security restrictions (whitelist, timeout, logging)

**read_file** — чтение файлов из workspace
- Описание: Read file contents from workspace. Returns content with line numbers

**write_file** — запись файлов в workspace
- Описание: Write content to a file in workspace. Supports create, append, overwrite modes

**list_dir** — листинг директорий workspace
- Описание: List directory contents in workspace. Supports recursive listing

**delete_file** — удаление файлов и директорий
- Описание: Delete file or directory from workspace. Supports recursive deletion

**system_time** — получение текущего времени
- Возвращает текущее системное время и дату в фиксированном формате
- Форматы: RFC3339 (2026-02-07T10:30:00+03:00) + человекочитаемый

## Commands

```bash
make build            # Build project
make test             # Run tests
make lint             # Run linter
make fmt              # Format code
make ci               # Run all CI checks
```

## Workflow for Ending Session

**MANDATORY STEPS before stopping:**

1. Create issues for remaining work
2. Run quality gates (if code changed): `make ci`
3. Update task statuses (close completed, update in-progress)
4. **PUSH TO REMOTE (CRITICAL):**
   ```bash
   git pull --rebase
   git push
   git status  # MUST show "up to date with origin"
   ```
5. Cleanup: Remove stashes, clean remote branches
6. Verify: All changes committed AND pushed
7. Provide context for next session

**CRITICAL RULES:**
- Work NOT completed until successful `git push`
- NEVER stop before push — leaves work stranded locally
- NEVER say "ready to push when you are" — YOU must push
- If push fails, resolve and retry until successful

## Priority

1. Security (`docs/rules/security.md`) — CRITICAL
2. Architecture (`docs/rules/architecture.md`) — Follow layers and dependencies
3. Code quality (`docs/rules/codequality.md`) — Follow conventions
4. Testing (`docs/rules/testing.md`) — Write tests

## Language Rules

- Answer ONLY in Russian
- Technical terms stay in English (API, endpoint, commit)
- No report/notes files (REFACTORING_NOTES.md, etc.) unless explicitly requested
- Delete files only with explicit permission
- Specify confidence (0-100%) after answers
- At 50% context fill: provide brief summary
- Minimize tokens, be concise (<4 lines unless details requested)
