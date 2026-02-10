// Package config provides configuration loading and validation for Nexbot.
// It supports TOML configuration files with environment variable expansion,
// default values, and comprehensive validation.
//
// Configuration structure:
//   - [workspace]: Workspace directory and bootstrap settings
//   - [agent]: Agent model and behavior configuration
//   - [llm]: LLM provider configuration (Z.ai, OpenAI)
//   - [logging]: Logging level, format, and output
//   - [channels]: Channel configurations (Telegram, Discord)
//   - [tools]: Tool configurations (file, shell)
//   - [cron]: Cron job configuration
//   - [message_bus]: Message bus capacity settings
//
// Environment variables:
// Environment variables can be referenced using ${VAR} or ${VAR:default} syntax.
// For example: api_key = "${ZAI_API_KEY:default_key}"
package config

import "path/filepath"

// Config represents the main application configuration.
type Config struct {
	Workspace  WorkspaceConfig  `toml:"workspace"`
	Agent      AgentConfig      `toml:"agent"`
	LLM        LLMConfig        `toml:"llm"`
	Logging    LoggingConfig    `toml:"logging"`
	Channels   ChannelsConfig   `toml:"channels"`
	Tools      ToolsConfig      `toml:"tools"`
	Cron       CronConfig       `toml:"cron"`
	Heartbeat  HeartbeatConfig  `toml:"heartbeat"`
	Workers    WorkersConfig    `toml:"workers"`
	Subagent   SubagentConfig   `toml:"subagent"`
	MessageBus MessageBusConfig `toml:"message_bus"`
	Cleanup    CleanupConfig    `toml:"cleanup"`
}

// WorkspaceConfig представляет конфигурацию workspace
type WorkspaceConfig struct {
	Path              string `toml:"path"`
	BootstrapMaxChars int    `toml:"bootstrap_max_chars"`
}

// AgentConfig представляет конфигурацию agent
type AgentConfig struct {
	Provider       string  `toml:"provider"`
	Model          string  `toml:"model"`
	MaxTokens      int     `toml:"max_tokens"`
	MaxIterations  int     `toml:"max_iterations"`
	Temperature    float64 `toml:"temperature"`
	TimeoutSeconds int     `toml:"timeout_seconds"`
}

// LLMConfig представляет конфигурацию LLM провайдера
type LLMConfig struct {
	ZAI    ZAIConfig `toml:"zai"`
	OpenAI struct {
		APIKey  string `toml:"api_key"`
		BaseURL string `toml:"base_url"`
	} `toml:"openai"`
}

// ZAIConfig представляет конфигурацию Z.ai провайдера
type ZAIConfig struct {
	APIKey         string `toml:"api_key"`
	BaseURL        string `toml:"base_url"`
	TimeoutSeconds int    `toml:"timeout_seconds"`
}

// LoggingConfig представляет конфигурацию логирования
type LoggingConfig struct {
	Level  string `toml:"level"`
	Format string `toml:"format"`
	Output string `toml:"output"`
}

// ChannelsConfig представляет конфигурацию каналов
type ChannelsConfig struct {
	Telegram TelegramConfig `toml:"telegram"`
	Discord  struct {
		Enabled       bool     `toml:"enabled"`
		Token         string   `toml:"token"`
		AllowedUsers  []string `toml:"allowed_users"`
		AllowedGuilds []string `toml:"allowed_guilds"`
	} `toml:"discord"`
}

// TelegramConfig представляет конфигурацию Telegram канала
type TelegramConfig struct {
	Enabled               bool     `toml:"enabled"`
	Token                 string   `toml:"token"`
	AllowedUsers          []string `toml:"allowed_users"`
	AllowedChats          []string `toml:"allowed_chats"`
	SendTimeoutSeconds    int      `toml:"send_timeout_seconds"`
	EnableInlineUpdates   bool     `toml:"enable_inline_updates"`
	DefaultParseMode      string   `toml:"default_parse_mode"`
	EnableInlineKeyboard  bool     `toml:"enable_inline_keyboard"`
	QuietMode             bool     `toml:"quiet_mode"`
	AnswerCallbackTimeout int      `toml:"answer_callback_timeout"`
}

// ToolsConfig представляет конфигурацию tools
type ToolsConfig struct {
	File  FileToolConfig  `toml:"file"`
	Shell ShellToolConfig `toml:"shell"`
	Fetch FetchToolConfig `toml:"fetch"`
}

// FileToolConfig представляет конфигурацию file tool
type FileToolConfig struct {
	Enabled              bool     `toml:"enabled"`
	WhitelistDirs        []string `toml:"whitelist_dirs"`
	ReadOnlyDirs         []string `toml:"read_only_dirs"`
	ValidateSkillContent bool     `toml:"validate_skill_content"`
}

// ShellToolConfig представляет конфигурацию shell tool
type ShellToolConfig struct {
	Enabled         bool     `toml:"enabled"`
	AllowedCommands []string `toml:"allowed_commands"`
	DenyCommands    []string `toml:"deny_commands"`
	AskCommands     []string `toml:"ask_commands"`
	TimeoutSeconds  int      `toml:"timeout_seconds"`
}

// FetchToolConfig представляет конфигурацию fetch tool
type FetchToolConfig struct {
	Enabled         bool   `toml:"enabled"`
	TimeoutSeconds  int    `toml:"timeout_seconds"`
	MaxResponseSize int64  `toml:"max_response_size"`
	UserAgent       string `toml:"user_agent"`
}

const (
	// CronSubdirectory is the subdirectory name for cron jobs within workspace
	CronSubdirectory = "cron"
)

// CronConfig представляет конфигурацию cron (v0.2)
type CronConfig struct {
	Enabled  bool   `toml:"enabled"`
	Timezone string `toml:"timezone"`
}

// JobsDir возвращает путь к директории для хранения cron jobs
func (c *CronConfig) JobsDir(workspacePath string) string {
	return filepath.Join(workspacePath, CronSubdirectory)
}

// WorkersConfig представляет конфигурацию worker pool (v0.2)
type WorkersConfig struct {
	PoolSize  int `toml:"pool_size"`
	QueueSize int `toml:"queue_size"`
}

// SubagentConfig представляет конфигурацию subagent manager (v0.2)
type SubagentConfig struct {
	Enabled        bool   `toml:"enabled"`
	MaxConcurrent  int    `toml:"max_concurrent"`
	TimeoutSeconds int    `toml:"timeout_seconds"`
	SessionPrefix  string `toml:"session_prefix"`
}

// MessageBusConfig представляет конфигурацию message bus
type MessageBusConfig struct {
	Capacity                  int  `toml:"capacity"`
	ResultChannelCapacity     int  `toml:"result_channel_capacity"`
	EnableHighPriorityResults bool `toml:"enable_high_priority_results"`
}

// HeartbeatConfig представляет конфигурацию HEARTBEAT loader (v0.2)
type HeartbeatConfig struct {
	Enabled              bool `toml:"enabled"`
	CheckIntervalMinutes int  `toml:"check_interval_minutes"`
}

// CleanupConfig представляет конфигурацию cleanup механизма для памяти и сессий
type CleanupConfig struct {
	Enabled          bool  `toml:"enabled"`
	IntervalMinutes  int   `toml:"interval_minutes"`
	MessageTTLDays   int   `toml:"message_ttl_days"`
	SessionTTLDays   int   `toml:"session_ttl_days"`
	MaxSessionSizeMB int64 `toml:"max_session_size_mb"`
	KeepActiveDays   int   `toml:"keep_active_days"`
}

// SecretsDir возвращает путь к директории для хранения секретов
func (c *Config) SecretsDir() string {
	return filepath.Join(c.Workspace.Path, "secrets")
}
