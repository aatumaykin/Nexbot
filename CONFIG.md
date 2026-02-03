# Configuration Reference

This document provides a complete reference for all Nexbot configuration options.

## Configuration File Location

Nexbot looks for configuration in the following locations (in order):

1. `--config` flag (highest priority)
2. `./config.toml` in the current directory
3. `~/.nexbot/config.toml`

## Environment Variables

Environment variables can be referenced in configuration using:

```toml
# Simple reference
api_key = "${ZAI_API_KEY}"

# With default value
api_key = "${ZAI_API_KEY:default-key}"
```

## Path Expansion

Paths in configuration support the following expansions:

- `~` expands to user's home directory
- Environment variables expand using `${VAR}` syntax
- Examples:
  - `path = "~/.nexbot"` → `/home/user/.nexbot`
  - `path = "${HOME}/.nexbot"` → `/home/user/.nexbot`

---

## Configuration Sections

### `[workspace]` - Workspace Configuration

Configuration for the workspace directory where Nexbot stores data.

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `path` | string | `~/.nexbot` | Path to workspace directory. Supports `~` expansion. |
| `bootstrap_max_chars` | int | `20000` | Maximum characters to read from bootstrap files (IDENTITY.md, AGENTS.md, etc.) |

**Example:**

```toml
[workspace]
path = "~/.nexbot"
bootstrap_max_chars = 20000
```

**Validation:**
- `path` must not be empty
- `path` must not contain `..` (path traversal)
- `bootstrap_max_chars` must be positive

---

### `[agent]` - Agent Settings

Configuration for agent behavior and model parameters.

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `model` | string | `glm-4.7-flash` | Default model for LLM requests |
| `max_tokens` | int | `8192` | Maximum tokens in LLM response |
| `max_iterations` | int | `20` | Maximum tool calling iterations per request |
| `temperature` | float64 | `0.7` | Temperature for LLM sampling (0.0 - 1.0) |
| `timeout_seconds` | int | `30` | Timeout for agent requests |

**Example:**

```toml
[agent]
model = "glm-4.7-flash"
max_tokens = 8192
max_iterations = 20
temperature = 0.7
timeout_seconds = 30
```

**Validation:**
- `max_tokens` must be positive
- `max_iterations` must be positive
- `temperature` must be between 0.0 and 1.0
- `timeout_seconds` must be positive

---

### `[llm]` - LLM Provider Configuration

Main configuration for the LLM provider.

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `provider` | string | `zai` | LLM provider to use: `zai` or `openai` |

**Example:**

```toml
[llm]
provider = "zai"
```

**Validation:**
- `provider` must be one of: `zai`, `openai`

---

#### `[llm.zai]` - Z.ai Configuration

Configuration for Z.ai LLM provider.

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `api_key` | string | (required) | Z.ai API key (format: `zai-*` or `sk-*`, min 10 chars) |
| `base_url` | string | `https://api.z.ai/api/coding/paas/v4` | Z.ai API base URL |
| `model` | string | `glm-4.7-flash` | Default Z.ai model to use |

**Example:**

```toml
[llm.zai]
api_key = "${ZAI_API_KEY}"
base_url = "https://api.z.ai/api/coding/paas/v4"
model = "glm-4.7-flash"
```

**Validation:**
- `api_key` is required when `provider = "zai"`
- `api_key` must be at least 10 characters
- `api_key` must start with `zai-` or `sk-`

#### `[llm.openai]` - OpenAI Configuration

Configuration for OpenAI LLM provider.

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `api_key` | string | (required) | OpenAI API key (format: `sk-*` or `org-*`, min 10 chars) |
| `base_url` | string | `https://api.openai.com/v1` | OpenAI API base URL |
| `model` | string | `gpt-4` | Default OpenAI model to use |

**Example:**

```toml
[llm]
provider = "openai"

[llm.openai]
api_key = "${OPENAI_API_KEY}"
base_url = "https://api.openai.com/v1"
model = "gpt-4"
```

**Validation:**
- `api_key` is required when `provider = "openai"`
- `api_key` must be at least 10 characters
- `api_key` must start with `sk-` or `org-`

---

### `[logging]` - Logging Configuration

Configuration for logging output.

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `level` | string | `info` | Log level: `debug`, `info`, `warn`, `error` |
| `format` | string | `json` | Log format: `json` or `text` |
| `output` | string | `stdout` | Log output: `stdout`, `stderr`, or file path |

**Example:**

```toml
[logging]
level = "info"
format = "json"
output = "stdout"
```

**Or for file output:**

```toml
[logging]
level = "debug"
format = "text"
output = "~/.nexbot/nexbot.log"
```

**Validation:**
- `level` must be one of: `debug`, `info`, `warn`, `error`
- `format` must be one of: `json`, `text`
- `output` must not be empty

---

### `[channels]` - Channel Configuration

Configuration for communication channels (Telegram, Discord, etc.).

#### `[channels.telegram]` - Telegram Configuration

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `enabled` | bool | `false` | Enable Telegram channel |
| `token` | string | (required) | Telegram bot token from [@BotFather](https://t.me/BotFather) |
| `allowed_users` | []string | `[]` | List of allowed Telegram user IDs (empty = allow all) |
| `allowed_chats` | []string | `[]` | List of allowed Telegram chat IDs (empty = allow all) |

**Example:**

```toml
[channels.telegram]
enabled = true
token = "${TELEGRAM_BOT_TOKEN}"
allowed_users = ["123456789", "987654321"]
allowed_chats = []
```

**Validation:**
- `token` is required when `enabled = true`
- `token` must follow format: `<bot_id>:<token>`
  - `bot_id`: 3-15 digits
  - `token`: 10-50 characters

**Security Notes:**
- Use `allowed_users` to restrict access to specific Telegram users
- Leave `allowed_users` empty to allow all users (not recommended for production)
- You can find your Telegram user ID using bots like [@userinfobot](https://t.me/userinfobot)

#### `[channels.discord]` - Discord Configuration (Future)

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `enabled` | bool | `false` | Enable Discord channel (not yet implemented) |
| `token` | string | (required) | Discord bot token |
| `allowed_users` | []string | `[]` | List of allowed Discord user IDs |
| `allowed_guilds` | []string | `[]` | List of allowed Discord server IDs |

**Example:**

```toml
[channels.discord]
enabled = false
token = "${DISCORD_BOT_TOKEN}"
allowed_users = []
allowed_guilds = []
```

---

### `[tools]` - Tool Configuration

Configuration for built-in tools (file, shell).

#### `[tools.file]` - File Tool Configuration

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `enabled` | bool | `true` | Enable file operations (read_file, write_file, list_dir) |
| `whitelist_dirs` | []string | `[]` | List of directories where file operations are allowed |
| `read_only_dirs` | []string | `[]` | List of directories that are read-only |

**Example:**

```toml
[tools.file]
enabled = true
whitelist_dirs = ["~/.nexbot", "~/projects", "~/Documents"]
read_only_dirs = ["/etc", "/usr", "/bin"]
```

**Security Notes:**
- File operations are restricted to `whitelist_dirs`
- Files in `read_only_dirs` can only be read, not written
- Path traversal is always blocked (security feature)
- Default `whitelist_dirs` empty means no file operations allowed

#### `[tools.shell]` - Shell Tool Configuration

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `enabled` | bool | `true` | Enable shell command execution |
| `allowed_commands` | []string | `[]` | List of allowed shell commands (empty = shell disabled) |
| `working_dir` | string | `~/.nexbot` | Default working directory for shell commands |
| `timeout_seconds` | int | `30` | Timeout for shell command execution |

**Example:**

```toml
[tools.shell]
enabled = true
allowed_commands = ["ls", "cat", "grep", "find", "cd", "pwd", "echo", "date", "git"]
working_dir = "${NEXBOT_WORKSPACE:~/.nexbot}"
timeout_seconds = 30
```

**Validation:**
- `allowed_commands` cannot be empty when `enabled = true`
- `allowed_commands` cannot contain empty strings
- `working_dir` must not contain `..` (path traversal)

**Security Notes:**
- Shell commands are restricted to `allowed_commands` list
- Each command is validated before execution
- Use `allowed_commands` to control what the bot can do
- Common safe commands: `ls`, `cat`, `grep`, `find`, `pwd`, `echo`, `date`

---

### `[cron]` - Cron Configuration (v0.2)

Configuration for scheduled tasks (coming in v0.2.0).

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `enabled` | bool | `false` | Enable cron scheduler |
| `jobs_dir` | string | `~/.nexbot/cron` | Directory containing cron job definitions |

**Example:**

```toml
[cron]
enabled = false
jobs_dir = "~/.nexbot/cron"
```

**Note:** This feature is planned for v0.2.0 and is not yet implemented.

---

### `[message_bus]` - Message Bus Configuration

Configuration for the message bus queue system.

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `capacity` | int | `1000` | Queue capacity for inbound/outbound messages |

**Example:**

```toml
[message_bus]
capacity = 1000
```

**Validation:**
- `capacity` must be positive

**Notes:**
- Higher capacity allows more concurrent messages but uses more memory
- Default (1000) is suitable for most use cases
- Increase capacity if you expect burst traffic

---

## Complete Example Configuration

```toml
# Workspace configuration
[workspace]
path = "~/.nexbot"
bootstrap_max_chars = 20000

# Agent settings
[agent]
model = "glm-4.7-flash"
max_tokens = 8192
max_iterations = 20
temperature = 0.7
timeout_seconds = 30

# LLM provider configuration
[llm]
provider = "zai"

[llm.zai]
api_key = "${ZAI_API_KEY}"
base_url = "https://api.z.ai/api/coding/paas/v4"
model = "glm-4.7-flash"

# Telegram channel configuration
[channels.telegram]
enabled = true
token = "${TELEGRAM_BOT_TOKEN}"
allowed_users = ["123456789"]  # Optional: restrict to specific users
allowed_chats = []

# Tool configuration
[tools.file]
enabled = true
whitelist_dirs = ["~/.nexbot", "~/projects", "~/Documents"]
read_only_dirs = ["/etc", "/usr", "/bin"]

[tools.shell]
enabled = true
allowed_commands = ["ls", "cat", "grep", "find", "cd", "pwd", "echo", "date", "git"]
working_dir = "~/.nexbot"
timeout_seconds = 30

# Logging configuration
[logging]
level = "info"
format = "json"
output = "stdout"

# Message bus configuration
[message_bus]
capacity = 1000
```

---

## Validation Rules

Nexbot validates configuration on startup. If validation fails, Nexbot will display error messages and exit with status code 1.

### API Key Validation

- Z.ai API keys must:
  - Start with `zai-` or `sk-`
  - Be at least 10 characters long
- OpenAI API keys must:
  - Start with `sk-` or `org-`
  - Be at least 10 characters long

### Telegram Token Validation

- Must follow format: `<bot_id>:<token>`
  - `bot_id`: 3-15 digits
  - `token`: 10-50 characters
- Example: `1234567890:ABCdefGHIjklMNOpqrsTUVwxyz`

### Path Validation

- Cannot be empty
- Cannot contain `..` (path traversal prevention)
- Supports `~` expansion for home directory
- Supports environment variable expansion

### Logging Level Validation

- Must be one of: `debug`, `info`, `warn`, `error`
- Default: `info`

### Logging Format Validation

- Must be one of: `json`, `text`
- Default: `json`

---

## Security Best Practices

1. **Use Environment Variables for Secrets:**
   ```toml
   api_key = "${ZAI_API_KEY}"
   token = "${TELEGRAM_BOT_TOKEN}"
   ```

2. **Restrict Access:**
   ```toml
   [channels.telegram]
   allowed_users = ["123456789"]  # Only allow specific users
   ```

3. **Limit Shell Commands:**
   ```toml
   [tools.shell]
   allowed_commands = ["ls", "cat", "grep"]  # Only allow safe commands
   ```

4. **Restrict File Access:**
   ```toml
   [tools.file]
   whitelist_dirs = ["~/.nexbot"]  # Only access specific directories
   ```

5. **Set Appropriate Log Level:**
   ```toml
   [logging]
   level = "info"  # Don't use "debug" in production
   ```

---

## Troubleshooting

### Configuration Validation Errors

**Error:** "workspace.path is required"
- **Solution:** Add `[workspace]` section with `path` value

**Error:** "llm.zai.api_key is required when provider is 'zai'"
- **Solution:** Add `api_key` to `[llm.zai]` section or set `ZAI_API_KEY` environment variable

**Error:** "telegram token has invalid format"
- **Solution:** Ensure token format is `bot_id:token` (e.g., `1234567890:ABCdefGHIjklMNOpqrsTUVwxyz`)

**Error:** "tools.shell.allowed_commands cannot be empty when shell tool is enabled"
- **Solution:** Add commands to `allowed_commands` list or disable shell tool

### Runtime Errors

**Error:** "Permission denied" accessing directories
- **Solution:** Check directory permissions and add to `whitelist_dirs`

**Error:** "Authentication error" from LLM provider
- **Solution:** Verify API key is correct and not expired

---

## See Also

- [README.md](README.md) - Project overview and quick start
- [QUICKSTART.md](QUICKSTART.md) - 5-minute setup guide
- [Example Config](config.example.toml) - Example configuration file
