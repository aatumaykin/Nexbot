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

// Config represents the main application configuration.
type Config struct {
	Workspace  WorkspaceConfig  `toml:"workspace"`
	Agent      AgentConfig      `toml:"agent"`
	LLM        LLMConfig        `toml:"llm"`
	Logging    LoggingConfig    `toml:"logging"`
	Channels   ChannelsConfig   `toml:"channels"`
	Tools      ToolsConfig      `toml:"tools"`
	Cron       CronConfig       `toml:"cron"`
	MessageBus MessageBusConfig `toml:"message_bus"`
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
	APIKey  string `toml:"api_key"`
	BaseURL string `toml:"base_url"`
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
	Enabled      bool     `toml:"enabled"`
	Token        string   `toml:"token"`
	AllowedUsers []string `toml:"allowed_users"`
	AllowedChats []string `toml:"allowed_chats"`
}

// ToolsConfig представляет конфигурацию tools
type ToolsConfig struct {
	File  FileToolConfig  `toml:"file"`
	Shell ShellToolConfig `toml:"shell"`
}

// FileToolConfig представляет конфигурацию file tool
type FileToolConfig struct {
	Enabled       bool     `toml:"enabled"`
	WhitelistDirs []string `toml:"whitelist_dirs"`
	ReadOnlyDirs  []string `toml:"read_only_dirs"`
}

// ShellToolConfig представляет конфигурацию shell tool
type ShellToolConfig struct {
	Enabled         bool     `toml:"enabled"`
	AllowedCommands []string `toml:"allowed_commands"`
	WorkingDir      string   `toml:"working_dir"`
	TimeoutSeconds  int      `toml:"timeout_seconds"`
}

// CronConfig представляет конфигурацию cron (v0.2)
type CronConfig struct {
	Enabled bool   `toml:"enabled"`
	JobsDir string `toml:"jobs_dir"`
}

// MessageBusConfig представляет конфигурацию message bus
type MessageBusConfig struct {
	Capacity int `toml:"capacity"`
}
