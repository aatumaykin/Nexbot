# Architecture Rules

## Overview

Nexbot uses **simple loop + message bus** architecture:

```
Telegram ──► Inbound Queue ──► Agent Loop ──► Outbound Queue ──► Telegram
                  (bus)            (LLM + Tools)         (bus)
```

## Layers

### Channels Layer (internal/channels/)
**Purpose:** Connect to external platforms (Telegram, Discord, etc.)

**Components:**
- telegram/connector.go — Telegram connector
- Interfaces for extensibility

**Rules:**
- Channels must work asynchronously via message bus
- No dependency on business logic
- Convert messages to universal format

### Bus Layer (internal/bus/)
**Purpose:** Asynchronous message queue between components

**Components:**
- queue.go — In-memory queue
- events.go — Event types

**Rules:**
- Decouple components via events
- In-memory implementation
- Event handlers must be idempotent

### Agent Layer (internal/agent/)
**Purpose:** Core agent logic with LLM and tool calling

**Components:**
- loop/loop.go — Agent execution loop
- context/context.go — System prompt builder
- session/session.go — Session management
- memory/memory.go — Memory management
- tools/tools.go — Tool integration

**Rules:**
- Loop processes messages recursively with tool calling
- Maximum 10 iterations of tool calling to prevent infinite loops
- Session history persisted to disk

### LLM Layer (internal/llm/)
**Purpose:** Abstraction over LLM providers

**Components:**
- provider.go — Provider interface
- zai.go — Z.ai implementation
- mock.go — Mock provider for testing

**Rules:**
- Provider interface must be minimal and extensible
- Support tool calling via Provider.SupportsToolCalling()
- Implementation must be easily replaceable

### Tools Layer (internal/tools/)
**Purpose:** Tools for executing tasks (files, shell, etc.)

**Components:**
- registry.go — Tool registry
- file.go — File operations
- shell.go — Shell commands

**Rules:**
- All tools registered via Registry
- Tool schema must be valid JSON Schema
- Tools must handle errors and return ToolResult

### Skills Layer (internal/skills/)
**Purpose:** Extensible skill system (OpenClaw compatible)

**Components:**
- loader.go — Skills loader
- parser.go — YAML frontmatter parser

**Rules:**
- Skills stored in skills/ directory
- Each skill is a directory with SKILL.md
- YAML frontmatter + markdown body
- Skills loaded at startup

### Config Layer (internal/config/)
**Purpose:** Application configuration

**Components:**
- schema.go — TOML config schema
- masking.go — Secret masking
- Validation

**Rules:**
- Configuration in TOML format
- Secrets via environment variables
- Validate on load

### Logger Layer (internal/logger/)
**Purpose:** Structured logging

**Components:**
- logger.go — Logger implementation

**Rules:**
- Structured logs with fields
- Support contexts
- Mask secrets

### Workspace Layer (internal/workspace/)
**Purpose:** Workspace and bootstrap file management

**Components:**
- workspace.go — Workspace operations
- bootstrap.go — Bootstrap file management

**Rules:**
- Workspace in ~/.nexbot/
- Bootstrap files: IDENTITY.md, AGENTS.md, SOUL.md, USER.md, TOOLS.md, MEMORY.md
- Context builder reads: IDENTITY → AGENTS → SOUL → USER → TOOLS → memory

## Dependency Direction

```
Channels ──► Bus ──► Agent ──► LLM
                           └─► Tools
                           └─► Skills
                           └─► Config
                           └─► Logger
                           └─► Workspace
```

**Rules:**
- Upper layers do not depend on lower layers
- All layers depend on Config and Logger
- Agent coordinates all components

## Extensibility

### Add New Channel
1. Create internal/channels/<name>/connector.go
2. Implement channel interface
3. Register in main

### Add New LLM Provider
1. Create internal/llm/<provider>.go
2. Implement Provider interface
3. Add to config.toml

### Add New Tool
1. Create file in internal/tools/
2. Implement Tool interface
3. Register in Registry

### Add New Skill
1. Create skills/<name>/SKILL.md
2. Add YAML frontmatter
3. Restart bot

## Anti-patterns

❌ Direct dependencies between channels
❌ Business logic in channels
❌ Synchronous message processing in channels
❌ Hardcoded configuration
❌ Uncontrolled tool call iterations
