package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

// Load загружает конфигурацию из TOML файла
func Load(path string) (*Config, error) {
	// Чтение файла
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Парсинг TOML
	var cfg Config
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Применяем значения по умолчанию
	applyDefaults(&cfg)

	// Расширяем переменные окружения
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
	}

	// Проверка LLM конфигурации
	if c.LLM.Provider == "" {
		errors = append(errors, fmt.Errorf("llm.provider is required"))
	} else {
		switch c.LLM.Provider {
		case "zai":
			if c.LLM.ZAI.APIKey == "" {
				errors = append(errors, fmt.Errorf("llm.zai.api_key is required when provider is 'zai'"))
			}
		case "openai":
			if c.LLM.OpenAI.APIKey == "" {
				errors = append(errors, fmt.Errorf("llm.openai.api_key is required when provider is 'openai'"))
			}
		default:
			errors = append(errors, fmt.Errorf("invalid llm.provider: %s (expected: zai, openai)", c.LLM.Provider))
		}
	}

	// Проверка Telegram канала
	if c.Channels.Telegram.Enabled {
		if c.Channels.Telegram.Token == "" {
			errors = append(errors, fmt.Errorf("channels.telegram.token is required when telegram is enabled"))
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

	return errors
}

// applyDefaults применяет значения по умолчанию
func applyDefaults(c *Config) {
	if c.Workspace.Path == "" {
		c.Workspace.Path = "~/.nexbot"
	}
	if c.Workspace.BootstrapMaxChars == 0 {
		c.Workspace.BootstrapMaxChars = 20000
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

	if c.LLM.Provider == "" {
		c.LLM.Provider = "zai"
	}
	if c.LLM.ZAI.BaseURL == "" {
		c.LLM.ZAI.BaseURL = "https://api.z.ai/api/coding/paas/v4"
	}
	if c.LLM.ZAI.Model == "" {
		c.LLM.ZAI.Model = "glm-4.7-flash"
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

	// Workspace path - support both environment variables and ~ expansion
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

	// Находим закрывающую скобку
	end := strings.Index(s, "}")
	if end == -1 {
		return s
	}

	content := s[2:end]
	// Проверяем наличие значения по умолчанию
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
