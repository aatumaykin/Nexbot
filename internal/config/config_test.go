package config

import (
	"testing"
)

func TestConfigDefaults(t *testing.T) {
	cfg := &Config{}
	applyDefaults(cfg)

	tests := []struct {
		name  string
		field string
		want  string
		got   string
	}{
		{"workspace path", "workspace.path", "~/.nexbot", cfg.Workspace.Path},
		{"agent model", "agent.model", "glm-4.7-flash", cfg.Agent.Model},
		{"llm provider", "llm.provider", "zai", cfg.LLM.Provider},
		{"zai model", "llm.zai.model", "glm-4.7-flash", cfg.LLM.ZAI.Model},
		{"logging level", "logging.level", "info", cfg.Logging.Level},
		{"logging format", "logging.format", "json", cfg.Logging.Format},
		{"logging output", "logging.output", "stdout", cfg.Logging.Output},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("Expected %s = %s, got %s", tt.field, tt.want, tt.got)
			}
		})
	}
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *Config
		wantErr bool
	}{
		{
			name: "valid config with minimal fields",
			cfg: &Config{
				Workspace: WorkspaceConfig{Path: "~/.nexbot"},
				LLM: LLMConfig{
					Provider: "zai",
					ZAI:      ZAIConfig{APIKey: "test-key"},
				},
				Logging: LoggingConfig{
					Level:  "info",
					Format: "json",
					Output: "stdout",
				},
			},
			wantErr: false,
		},
		{
			name: "missing workspace path",
			cfg: &Config{
				LLM: LLMConfig{
					Provider: "zai",
					ZAI:      ZAIConfig{APIKey: "test-key"},
				},
				Logging: LoggingConfig{
					Level:  "info",
					Format: "json",
					Output: "stdout",
				},
			},
			wantErr: true,
		},
		{
			name: "missing llm provider",
			cfg: &Config{
				Workspace: WorkspaceConfig{Path: "~/.nexbot"},
				Logging: LoggingConfig{
					Level:  "info",
					Format: "json",
					Output: "stdout",
				},
			},
			wantErr: true,
		},
		{
			name: "invalid llm provider",
			cfg: &Config{
				Workspace: WorkspaceConfig{Path: "~/.nexbot"},
				LLM: LLMConfig{
					Provider: "invalid",
					ZAI:      ZAIConfig{APIKey: "test-key"},
				},
				Logging: LoggingConfig{
					Level:  "info",
					Format: "json",
					Output: "stdout",
				},
			},
			wantErr: true,
		},
		{
			name: "missing zai api key",
			cfg: &Config{
				Workspace: WorkspaceConfig{Path: "~/.nexbot"},
				LLM: LLMConfig{
					Provider: "zai",
					ZAI:      ZAIConfig{APIKey: ""},
				},
				Logging: LoggingConfig{
					Level:  "info",
					Format: "json",
					Output: "stdout",
				},
			},
			wantErr: true,
		},
		{
			name: "invalid logging level",
			cfg: &Config{
				Workspace: WorkspaceConfig{Path: "~/.nexbot"},
				LLM: LLMConfig{
					Provider: "zai",
					ZAI:      ZAIConfig{APIKey: "test-key"},
				},
				Logging: LoggingConfig{
					Level:  "invalid",
					Format: "json",
					Output: "stdout",
				},
			},
			wantErr: true,
		},
		{
			name: "invalid logging format",
			cfg: &Config{
				Workspace: WorkspaceConfig{Path: "~/.nexbot"},
				LLM: LLMConfig{
					Provider: "zai",
					ZAI:      ZAIConfig{APIKey: "test-key"},
				},
				Logging: LoggingConfig{
					Level:  "info",
					Format: "invalid",
					Output: "stdout",
				},
			},
			wantErr: true,
		},
		{
			name: "telegram enabled but missing token",
			cfg: &Config{
				Workspace: WorkspaceConfig{Path: "~/.nexbot"},
				LLM: LLMConfig{
					Provider: "zai",
					ZAI:      ZAIConfig{APIKey: "test-key"},
				},
				Channels: ChannelsConfig{
					Telegram: TelegramConfig{
						Enabled: true,
						Token:   "",
					},
				},
				Logging: LoggingConfig{
					Level:  "info",
					Format: "json",
					Output: "stdout",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := tt.cfg.Validate()
			hasErrors := len(errors) > 0
			if hasErrors != tt.wantErr {
				t.Errorf("Validate() hasErrors = %v, wantErr %v", hasErrors, tt.wantErr)
			}
		})
	}
}

func TestExpandEnv(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "simple variable",
			input: "${TEST_VAR}",
			want:  "", // Should be empty if not set
		},
		{
			name:  "variable with default",
			input: "${TEST_VAR:default}",
			want:  "default",
		},
		{
			name:  "no expansion",
			input: "plain text",
			want:  "plain text",
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := expandEnv(tt.input)
			if got != tt.want {
				t.Errorf("expandEnv(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
