# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.3.0] - 2026-02-08

### Added
- **Spawn tool integration** — LLM может создавать subagents для выполнения задач
- **Synchronous task execution** — subagents выполняют задачи и возвращают результаты
- **Automatic cleanup** — subagents удаляются после выполнения задачи
- **Session isolation** — каждый subagent имеет изолированную сессию
- **Nested spawning** — subagents могут создавать другие subagents
- **ExecuteTask method** — новый метод в Subagent Manager для синхронного выполнения
- **DeleteSession method** — новый метод в Session Manager для очистки сессий

### Changed
- Improved agent capabilities with parallel task execution
- Enhanced documentation with spawn tool examples
- Updated architecture docs with spawn integration details
- SpawnTool now returns task result instead of subagent ID

### Fixed
- Subagent sessions are now properly cleaned up after task completion
- No memory leaks from subagent registry

### Testing
- Added tests for spawn tool execution
- Added tests for subagent lifecycle (spawn, process, stop, delete)
- Added integration tests for nested subagents

### Documentation
- Updated README.md with spawn tool usage
- Updated QUICKSTART.md with subagent spawning guide
- Updated EXAMPLES.md with subagent examples
- Created docs/examples/subagent-usage.md with detailed examples
- Updated docs/workspace/TOOLS.md with spawn tool reference
- Updated docs/workspace/AGENTS.md with subagent usage guide
- Updated docs/workspace/IDENTITY.md with spawn core truths
- Updated docs/architecture/subagent_manager.md with spawn integration details

## [Unreleased]

## [0.2.0] - 2026-02-05

### Added
- **Cron Scheduler**: Added cron-based task scheduling using `robfig/cron/v3`
  - Scheduled jobs are stored in `~/.nexbot/cron.json`
  - Automatic job execution at specified intervals
- **Cron CLI Commands**: Added command-line interface for cron management
  - `nexbot cron add <schedule> <command>` - Add new scheduled task
  - `nexbot cron list` - List all scheduled tasks
  - `nexbot cron remove <job_id>` - Remove scheduled task
- **Subagent Manager**: Added ability to spawn and manage subagents
  - Create isolated subagent sessions with unique IDs
  - Track and coordinate multiple concurrent subagents
  - Graceful shutdown of all subagents
- **Background Task Execution**: Added worker pool for async task processing
  - Configurable number of workers (default: 5)
  - Support for both cron and subagent task types
  - Task queue with configurable buffer size (default: 20)
- **Spawn Tool**: Registered `spawn` tool for LLM to create subagents
  - Automatically generate unique subagent IDs
  - Return task results to parent agent
- **HEARTBEAT.md Support**: Added proactive task execution from workspace
  - Tasks defined in `~/.nexbot/HEARTBEAT.md` are processed automatically
  - Supports scheduling with cron expressions
- **Command Handler Infrastructure**: Added `internal/commands/handler.go` for CLI command management
- **Constants Packages**: Added `internal/constants/` for centralized constants
  - `commands.go` - Command definitions
  - `messages.go` - Message templates
  - `paths.go` - Path constants
  - `defaults.go` - Default values
  - `cron.go` - Cron-specific constants
  - `test.go` - Test-related constants
- **Messages Package**: Added `internal/messages/` for message handling
  - `status.go` - Status messages
  - `errors.go` - Error messages
- **Environment Variables**: Added `internal/config/env.go` for environment variable handling

### Changed
- Enhanced message bus to support event streaming
- Improved error handling throughout the codebase
- Updated agent loop to support subagent coordination

### Fixed
- Fixed syntax errors in test files (duplicate `if err :=` statements)
- Fixed message bus subscription issues in E2E tests
- Removed redundant `bus.Stop()` calls in test suite

### Testing
- Added comprehensive tests for cron scheduling (96.1% coverage)
- Added tests for subagent management (88.3% coverage)
- Added tests for worker pool (94.4% coverage)
- Added tests for heartbeat functionality (97.4% coverage)
- All packages now have >80% test coverage

### Documentation
- Updated PROJECT_PLAN.md with v0.2.0 completion status
- Added detailed documentation for cron features
- Added documentation for subagent management
- Added documentation for heartbeat tasks
- Added `docs/architecture/` directory with component architecture documentation:
  - `cron_scheduler.md` - Complete cron scheduler architecture with flow diagrams
  - `subagent_manager.md` - Subagent manager architecture and lifecycle
  - `worker_pool.md` - Worker pool architecture and task execution flow
  - `heartbeat_system.md` - HEARTBEAT system architecture and integration
- Updated `docs/ARCHITECTURE.md` with v0.2.0 components and diagrams

## [0.1.0] - 2026-02-03

### Added
- Initial release of Nexbot
- Telegram bot integration with `telego` library
- Z.ai Coding API integration
- Tool calling infrastructure (read_file, write_file, list_dir, shell_exec)
- Workspace management with bootstrap files
- Session management with JSONL storage
- Memory store with markdown/JSONL support
- Skills loader for OpenClaw-compatible skills
- Message bus architecture with inbound/outbound queues
- Configuration system with TOML support
- Logging with slog wrapper
- Makefile with build targets for multiple platforms

### Testing
- 200+ unit and integration tests
- >80% code coverage across all modules

### Documentation
- README.md with quickstart guide
- QUICKSTART.md with 5-minute setup
- CONFIG.md with configuration reference
- Comprehensive inline documentation
