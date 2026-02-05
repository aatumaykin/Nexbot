# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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
