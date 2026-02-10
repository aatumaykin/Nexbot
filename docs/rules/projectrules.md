# Project Rules

## Development Rules

- Use Go 1.25.5+
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
