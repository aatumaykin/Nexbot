package config

import (
	"fmt"
	"github.com/BurntSushi/toml"
	"os"
	"path/filepath"
	"strings"
)

// Load загружает конфигурацию из TOML файла
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	applyDefaults(&cfg)

	if err := expandEnvVars(&cfg); err != nil {
		return nil, fmt.Errorf("failed to expand environment variables: %w", err)
	}

	return &cfg, nil
}

// Validate проверяет валидность конфигурации
func (c *Config) Validate() []error {
	var errors []error

	// Проверка workspace
	if c.Workspace.Path == "" {
		errors = append(errors, fmt.Errorf("workspace.path is required"))
	} else if err := validatePath(c.Workspace.Path, "workspace.path"); err != nil {
		errors = append(errors, err)
	}

	// Проверка Agent конфигурации
	if c.Agent.Provider == "" {
		errors = append(errors, fmt.Errorf("agent.provider is required"))
	} else {
		switch c.Agent.Provider {
		case "zai":
			if c.LLM.ZAI.APIKey == "" {
				errors = append(errors, fmt.Errorf("llm.zai.api_key is required when provider is 'zai'"))
			} else if err := validateAPIKey(c.LLM.ZAI.APIKey, "llm.zai.api_key"); err != nil {
				errors = append(errors, err)
			}
		case "openai":
			if c.LLM.OpenAI.APIKey == "" {
				errors = append(errors, fmt.Errorf("llm.openai.api_key is required when provider is 'openai'"))
			} else if err := validateAPIKey(c.LLM.OpenAI.APIKey, "llm.openai.api_key"); err != nil {
				errors = append(errors, err)
			}
		default:
			errors = append(errors, fmt.Errorf("invalid agent.provider: %s (expected: zai, openai)", c.Agent.Provider))
		}
	}

	// Проверка Telegram канала
	if c.Channels.Telegram.Enabled {
		if c.Channels.Telegram.Token == "" {
			errors = append(errors, fmt.Errorf("channels.telegram.token is required when telegram is enabled"))
		} else if err := validateTelegramToken(c.Channels.Telegram.Token); err != nil {
			errors = append(errors, err)
		}
	}

	// Проверка logging config
	if c.Logging.Level == "" {
		errors = append(errors, fmt.Errorf("logging.level is required"))
	} else {
		validLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
		if !validLevels[strings.ToLower(c.Logging.Level)] {
			errors = append(errors, fmt.Errorf("invalid logging.level: %s (expected: debug, info, warn, error)", c.Logging.Level))
		}
	}

	if c.Logging.Format == "" {
		errors = append(errors, fmt.Errorf("logging.format is required"))
	} else {
		validFormats := map[string]bool{"json": true, "text": true}
		if !validFormats[strings.ToLower(c.Logging.Format)] {
			errors = append(errors, fmt.Errorf("invalid logging.format: %s (expected: json, text)", c.Logging.Format))
		}
	}

	if c.Logging.Output == "" {
		errors = append(errors, fmt.Errorf("logging.output is required"))
	}

	// Проверка shell tool whitelist
	if c.Tools.Shell.Enabled {
		if len(c.Tools.Shell.AllowedCommands) == 0 {
			errors = append(errors, fmt.Errorf("tools.shell.allowed_commands cannot be empty when shell tool is enabled"))
		} else {
			for _, cmd := range c.Tools.Shell.AllowedCommands {
				if cmd == "" {
					errors = append(errors, fmt.Errorf("tools.shell.allowed_commands contains empty command"))
				}
			}
		}

		// Проверка working directory
		if c.Tools.Shell.WorkingDir != "" {
			if err := validatePath(c.Tools.Shell.WorkingDir, "tools.shell.working_dir"); err != nil {
				errors = append(errors, err)
			}
		}
	}

	return errors
}

// Helper validation functions
func validateAPIKey(key, fieldName string) error {
	if key == "" {
		return fmt.Errorf("%s cannot be empty", fieldName)
	}

	if len(key) < 10 {
		return fmt.Errorf("%s is too short (minimum 10 characters, got %d)", fieldName, len(key))
	}

	return nil
}

func validateTelegramToken(token string) error {
	if token == "" {
		return fmt.Errorf("telegram token cannot be empty")
	}

	parts := strings.Split(token, ":")
	if len(parts) != 2 {
		return fmt.Errorf("telegram token has invalid format (expected format: <bot_id>:<token>, got: %s)", maskSecret(token))
	}

	botID := parts[0]
	botToken := parts[1]

	if len(botID) < 3 || len(botID) > 15 {
		return fmt.Errorf("telegram token has invalid bot ID length (expected 3-15 digits, got %d digits)", len(botID))
	}

	// Check that bot ID contains only digits
	for _, r := range botID {
		if r < '0' || r > '9' {
			return fmt.Errorf("telegram token has invalid bot ID (expected digits only, got: %s)", botID)
		}
	}

	if len(botToken) < 10 || len(botToken) > 50 {
		return fmt.Errorf("telegram token has invalid token length (expected 10-50 characters, got %d)", len(botToken))
	}

	return nil
}

func validatePath(path, fieldName string) error {
	if path == "" {
		return fmt.Errorf("%s cannot be empty", fieldName)
	}

	if strings.HasPrefix(path, "~") {
		return nil
	}

	if strings.Contains(path, "..") {
		return fmt.Errorf("%s contains potentially dangerous path traversal sequence", fieldName)
	}

	return nil
}

// applyDefaults применяет значения по умолчанию
func applyDefaults(c *Config) {
	if c.Workspace.Path == "" {
		c.Workspace.Path = "~/.nexbot"
	}
	if c.Workspace.BootstrapMaxChars == 0 {
		c.Workspace.BootstrapMaxChars = 20000
	}

	if c.Agent.Provider == "" {
		c.Agent.Provider = "zai"
	}
	if c.Agent.Model == "" {
		c.Agent.Model = "glm-4.7-flash"
	}
	if c.Agent.MaxTokens == 0 {
		c.Agent.MaxTokens = 8192
	}
	if c.Agent.MaxIterations == 0 {
		c.Agent.MaxIterations = 20
	}
	if c.Agent.Temperature == 0 {
		c.Agent.Temperature = 0.7
	}
	if c.Agent.TimeoutSeconds == 0 {
		c.Agent.TimeoutSeconds = 30
	}

	if c.LLM.ZAI.BaseURL == "" {
		c.LLM.ZAI.BaseURL = "https://api.z.ai/api/coding/paas/v4"
	}

	if c.Logging.Level == "" {
		c.Logging.Level = "info"
	}
	if c.Logging.Format == "" {
		c.Logging.Format = "json"
	}
	if c.Logging.Output == "" {
		c.Logging.Output = "stdout"
	}

	if c.Tools.Shell.TimeoutSeconds == 0 {
		c.Tools.Shell.TimeoutSeconds = 30
	}

	if c.MessageBus.Capacity == 0 {
		c.MessageBus.Capacity = 1000
	}
}

// expandEnvVars расширяет переменные окружения в конфигурации
func expandEnvVars(c *Config) error {
	// ZAI API Key
	if strings.HasPrefix(c.LLM.ZAI.APIKey, "${") {
		c.LLM.ZAI.APIKey = expandEnv(c.LLM.ZAI.APIKey)
	}

	// Telegram Token
	if strings.HasPrefix(c.Channels.Telegram.Token, "${") {
		c.Channels.Telegram.Token = expandEnv(c.Channels.Telegram.Token)
	}

	// Workspace path
	if strings.HasPrefix(c.Workspace.Path, "${") {
		c.Workspace.Path = expandEnv(c.Workspace.Path)
	}
	c.Workspace.Path = expandHome(c.Workspace.Path)

	// Shell working dir
	if strings.HasPrefix(c.Tools.Shell.WorkingDir, "${") {
		c.Tools.Shell.WorkingDir = expandEnv(c.Tools.Shell.WorkingDir)
	}
	c.Tools.Shell.WorkingDir = expandHome(c.Tools.Shell.WorkingDir)

	// File tool directories
	for i, dir := range c.Tools.File.WhitelistDirs {
		c.Tools.File.WhitelistDirs[i] = expandHome(dir)
	}
	for i, dir := range c.Tools.File.ReadOnlyDirs {
		c.Tools.File.ReadOnlyDirs[i] = expandHome(dir)
	}

	return nil
}

// expandEnv расширяет переменную окружения формата ${VAR:default}
func expandEnv(s string) string {
	if !strings.HasPrefix(s, "${") {
		return s
	}

	end := strings.Index(s, "}")
	if end == -1 {
		return s
	}

	content := s[2:end]
	if parts := strings.SplitN(content, ":", 2); len(parts) == 2 {
		key := parts[0]
		defaultVal := parts[1]
		if val := os.Getenv(key); val != "" {
			return val
		}
		return defaultVal
	}

	// Без значения по умолчанию
	return os.Getenv(s[2:end])
}

// expandHome расширяет ~ в пути
func expandHome(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(home, path[2:])
	}
	return path
}
