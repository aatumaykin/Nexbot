# Project Rules

## Development Rules

- Use Go 1.26+
- Use modern Go features (see Go 1.24+ Features section below)
- Code and commits in English
- Keep project lightweight (~8-10K lines of code)
- Follow principles: DRY, SOLID, KISS, YAGNI

## Testing

- Write unit tests for each package (file_test.go)
- Write integration tests in tests/
- Target coverage: >70%

## Logging

- Use structured logging via internal/logger
- All logs must be contextual (with fields)
- Levels: Debug, Info, Warn, Error
- Mask secrets in all logs

## Configuration

- Configuration in TOML (config.toml)
- Secrets via environment variables (.env)
- Validate config via internal/config/schema.go
- Mask secrets in logs via internal/config/masking.go

## Workspace

- Workspace in ~/.nexbot/
- Bootstrap files: IDENTITY.md, AGENTS.md, SOUL.md, USER.md, TOOLS.md, MEMORY.md
- Context builder reads in order: IDENTITY → AGENTS → SOUL → USER → TOOLS → memory

## Skills

- Skills stored in skills/ directory
- Each skill is a directory with SKILL.md
- Format: YAML frontmatter + markdown body
- OpenClaw compatible
- Skill files must be created via write_file tool with path validation:
  - Must be in skills/ directory
  - Must be named SKILL.md
  - Optional YAML frontmatter validation (configurable via tools.file.validate_skill_content)
  - Required frontmatter fields: name, description

## Tool Calling

- Tools registered via tools.Registry
- Each tool must implement tools.Tool interface
- Tool schemas converted to llm.ToolDefinition for LLM

## Repository Work

- Main branch: main
- Feature branch: feature/<feature-name>
- Commit message: imperative mood (e.g., "Add amazing feature")
- Run `make ci` before committing

## Go 1.24+ Features

При рефакторинге и добавлении нового функционала используйте современные возможности Go:

### Go 1.24 (February 2025)
- **`go tool dist` improvements** — улучшения в toolchain
- **Weak dependencies (`go quorum`)** — опциональные зависимости в go.mod
- **`testing.B.Loop`** — более точные бенчмарки
- **Improved map iteration** — детерминированный порядок итерации в тестах

### Go 1.25 (August 2025)
- **Range over integers** — `for i := range 10` (0..9)
- **Range over functions** — custom iterators via `yield`
- **Improved type inference** — более умный вывод типов в generics
- **`slices.Values`, `slices.Keys`, `maps.Keys`** — iterator helpers
- **Enhanced `go test`** — улучшенные возможности тестирования

### Go 1.26 (February 2026)
- **Improved iterators** — более стабильная поддержка range over func
- **Performance improvements** — оптимизации runtime и GC
- **Toolchain updates** — улучшения в go tool

### Рекомендации по применению
1. **Range over integers**: используйте `for i := range n` вместо `for i := 0; i < n; i++`
2. **Custom iterators**: создавайте итераторы для коллекций через `yield`
3. **slices/maps packages**: предпочитайте `slices.*` и `maps.*` вместо ручных циклов
4. **Type inference**: упрощайте generic-код, полагаясь на вывод типов
5. **Modern patterns**: применяйте `modernizers` для автоматического обновления кода
