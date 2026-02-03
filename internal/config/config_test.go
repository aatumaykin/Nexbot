package config

import (
	"os"
	"path/filepath"
	"strings"
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
					ZAI:      ZAIConfig{APIKey: "zai-test-key-valid"},
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

func TestWorkspacePathExpansion(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("failed to get home directory: %v", err)
	}

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "tilde expansion",
			input: "~/.nexbot",
			want:  filepath.Join(home, ".nexbot"),
		},
		{
			name:  "tilde with nested path",
			input: "~/projects/nexbot",
			want:  filepath.Join(home, "projects", "nexbot"),
		},
		{
			name:  "absolute path",
			input: "/tmp/nexbot",
			want:  "/tmp/nexbot",
		},
		{
			name:  "relative path",
			input: "./nexbot",
			want:  "./nexbot",
		},
		{
			name:  "plain path without tilde",
			input: "nexbot",
			want:  "nexbot",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := expandHome(tt.input)
			if got != tt.want {
				t.Errorf("expandHome(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestWorkspacePathEnvExpansion(t *testing.T) {
	// Set test environment variable
	testEnv := "/test/workspace"
	if err := os.Setenv("TEST_WORKSPACE", testEnv); err != nil {
		t.Fatalf("failed to set env var: %v", err)
	}
	defer os.Unsetenv("TEST_WORKSPACE")

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "env variable without default",
			input: "${TEST_WORKSPACE}",
			want:  testEnv,
		},
		{
			name:  "env variable with default",
			input: "${NONEXISTENT_VAR:~/default}",
			want:  "~/default", // Not expanded yet (expandHome will handle it)
		},
		{
			name:  "env variable with default when set",
			input: "${TEST_WORKSPACE:~/default}",
			want:  testEnv,
		},
		{
			name:  "plain path without env vars",
			input: "~/.nexbot",
			want:  "~/.nexbot", // Not expanded yet (expandHome will handle it)
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

func TestWorkspacePathEdgeCases(t *testing.T) {
	home, _ := os.UserHomeDir()

	tests := []struct {
		name       string
		setup      func()
		teardown   func()
		input      string
		wantPrefix string
		wantErr    bool
	}{
		{
			name:  "empty path",
			input: "",
		},
		{
			name:       "env var then tilde",
			input:      "${NONEXISTENT_VAR:~/.nexbot}",
			wantPrefix: home,
		},
		{
			name:  "just tilde",
			input: "~",
		},
		{
			name:  "path starting with tilde but not followed by slash",
			input: "~test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}
			if tt.teardown != nil {
				defer tt.teardown()
			}

			// First expand env vars
			result := expandEnv(tt.input)
			// Then expand home
			result = expandHome(result)

			if tt.wantPrefix != "" {
				if !strings.HasPrefix(result, tt.wantPrefix) {
					t.Errorf("result = %q, want prefix %q", result, tt.wantPrefix)
				}
			}
		})
	}
}

func TestConfigToWorkspaceIntegration(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()

	// Create a test config file
	configContent := `[workspace]
path = "` + tmpDir + `"
bootstrap_max_chars = 5000

[llm]
provider = "zai"

[llm.zai]
api_key = "test-key"

[logging]
level = "info"
format = "json"
output = "stdout"
`

	configFile := filepath.Join(tmpDir, "config.toml")
	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to create config file: %v", err)
	}

	// Load config
	cfg, err := Load(configFile)
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	// Verify workspace path is expanded and absolute
	if cfg.Workspace.Path == "" {
		t.Error("workspace path is empty")
	}

	absPath, err := filepath.Abs(tmpDir)
	if err != nil {
		t.Fatalf("failed to get absolute path: %v", err)
	}

	if cfg.Workspace.Path != absPath {
		t.Errorf("Workspace.Path = %q, want %q", cfg.Workspace.Path, absPath)
	}
}

func TestConfigToWorkspaceIntegrationWithTilde(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("failed to get home directory: %v", err)
	}

	// Create a test config file with tilde path
	tmpDir := t.TempDir()
	configContent := `[workspace]
path = "~/.nexbot"
bootstrap_max_chars = 5000

[llm]
provider = "zai"

[llm.zai]
api_key = "test-key"

[logging]
level = "info"
format = "json"
output = "stdout"
`

	configFile := filepath.Join(tmpDir, "config.toml")
	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to create config file: %v", err)
	}

	// Load config
	cfg, err := Load(configFile)
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	expectedPath := filepath.Join(home, ".nexbot")
	if cfg.Workspace.Path != expectedPath {
		t.Errorf("Workspace.Path = %q, want %q", cfg.Workspace.Path, expectedPath)
	}
}

func TestConfigToWorkspaceIntegrationWithEnvVar(t *testing.T) {
	// Create a temporary directory
	tmpDir := t.TempDir()

	// Set environment variable
	if err := os.Setenv("TEST_NEXBOT_PATH", tmpDir); err != nil {
		t.Fatalf("failed to set env var: %v", err)
	}
	defer os.Unsetenv("TEST_NEXBOT_PATH")

	// Create a test config file with env var
	configContent := `[workspace]
path = "${TEST_NEXBOT_PATH}"
bootstrap_max_chars = 5000

[llm]
provider = "zai"

[llm.zai]
api_key = "test-key"

[logging]
level = "info"
format = "json"
output = "stdout"
`

	configFile := filepath.Join(tmpDir, "config.toml")
	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to create config file: %v", err)
	}

	// Load config
	cfg, err := Load(configFile)
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	absPath, err := filepath.Abs(tmpDir)
	if err != nil {
		t.Fatalf("failed to get absolute path: %v", err)
	}

	if cfg.Workspace.Path != absPath {
		t.Errorf("Workspace.Path = %q, want %q", cfg.Workspace.Path, absPath)
	}
}

// Tests for validateAPIKey
func TestValidateAPIKey(t *testing.T) {
	tests := []struct {
		name      string
		key       string
		fieldName string
		wantErr   bool
		errMsg    string
	}{
		{
			name:      "valid zai api key with zai- prefix",
			key:       "zai-test-key-valid",
			fieldName: "llm.zai.api_key",
			wantErr:   false,
		},
		{
			name:      "valid zai api key with sk- prefix",
			key:       "sk-test-key-valid",
			fieldName: "llm.zai.api_key",
			wantErr:   false,
		},
		{
			name:      "valid openai api key with sk- prefix",
			key:       "sk-test-key-valid",
			fieldName: "llm.openai.api_key",
			wantErr:   false,
		},
		{
			name:      "valid openai api key with org- prefix",
			key:       "org-test-key-valid",
			fieldName: "llm.openai.api_key",
			wantErr:   false,
		},
		{
			name:      "empty api key",
			key:       "",
			fieldName: "llm.zai.api_key",
			wantErr:   true,
			errMsg:    "cannot be empty",
		},
		{
			name:      "api key too short (9 chars)",
			key:       "zai-short",
			fieldName: "llm.zai.api_key",
			wantErr:   true,
			errMsg:    "too short",
		},
		{
			name:      "api key exactly 10 chars",
			key:       "zai-123456",
			fieldName: "llm.zai.api_key",
			wantErr:   false,
		},
		{
			name:      "zai api key with invalid prefix",
			key:       "invalid-test-key",
			fieldName: "llm.zai.api_key",
			wantErr:   true,
			errMsg:    "invalid format",
		},
		{
			name:      "openai api key with invalid prefix",
			key:       "invalid-test-key",
			fieldName: "llm.openai.api_key",
			wantErr:   true,
			errMsg:    "invalid format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateAPIKey(tt.key, tt.fieldName)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateAPIKey() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errMsg != "" && err != nil {
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("validateAPIKey() error = %v, want error message to contain %q", err, tt.errMsg)
				}
			}
		})
	}
}

// Tests for validateTelegramToken
func TestValidateTelegramToken(t *testing.T) {
	tests := []struct {
		name    string
		token   string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid telegram token",
			token:   "1234567890:ABCdefGHIjklMNOpqrsTUVwxyz",
			wantErr: false,
		},
		{
			name:    "valid telegram token with minimum bot ID (3 digits)",
			token:   "123:ABCDEFGHIJKLMNO",
			wantErr: false,
		},
		{
			name:    "valid telegram token with minimum token length (10 chars)",
			token:   "1234567890:ABCDEFGHIJ",
			wantErr: false,
		},
		{
			name:    "valid telegram token with maximum bot ID (15 digits)",
			token:   "123456789012345:ABCDEFGHIJKLMNO",
			wantErr: false,
		},
		{
			name:    "valid telegram token with maximum token length (50 chars)",
			token:   "1234567890:ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrst",
			wantErr: false,
		},
		{
			name:    "empty telegram token",
			token:   "",
			wantErr: true,
			errMsg:  "cannot be empty",
		},
		{
			name:    "missing colon separator",
			token:   "1234567890-ABCdefGHIjklMNOpqrsTUVwxyz",
			wantErr: true,
			errMsg:  "invalid format",
		},
		{
			name:    "bot ID too short (2 digits)",
			token:   "12:ABCdefGHIjklMNOpqrsTUVwxyz",
			wantErr: true,
			errMsg:  "bot ID",
		},
		{
			name:    "bot ID too long (16 digits)",
			token:   "1234567890123456:ABCdefGHIjklMNOpqrsTUVwxyz",
			wantErr: true,
			errMsg:  "bot ID",
		},
		{
			name:    "token too short (9 chars)",
			token:   "1234567890:ABCDEFGHI",
			wantErr: true,
			errMsg:  "token",
		},
		{
			name:    "token too long (51 chars)",
			token:   "1234567890:ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxy", // Token part has 51 chars
			wantErr: true,
			errMsg:  "token",
		},
		{
			name:    "bot ID contains non-digits",
			token:   "abc1234567:ABCdefGHIjklMNOpqrsTUVwxyz",
			wantErr: true,
			errMsg:  "bot ID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateTelegramToken(tt.token)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateTelegramToken() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errMsg != "" && err != nil {
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("validateTelegramToken() error = %v, want error message to contain %q", err, tt.errMsg)
				}
			}
		})
	}
}

// Tests for validatePath
func TestValidatePath(t *testing.T) {
	tests := []struct {
		name      string
		path      string
		fieldName string
		wantErr   bool
		errMsg    string
	}{
		{
			name:      "absolute path",
			path:      "/tmp/nexbot",
			fieldName: "workspace.path",
			wantErr:   false,
		},
		{
			name:      "relative path",
			path:      "./nexbot",
			fieldName: "workspace.path",
			wantErr:   false,
		},
		{
			name:      "path with tilde",
			path:      "~/.nexbot",
			fieldName: "workspace.path",
			wantErr:   false,
		},
		{
			name:      "path with tilde and subdirectory",
			path:      "~/projects/nexbot",
			fieldName: "workspace.path",
			wantErr:   false,
		},
		{
			name:      "simple path",
			path:      "nexbot",
			fieldName: "workspace.path",
			wantErr:   false,
		},
		{
			name:      "empty path",
			path:      "",
			fieldName: "workspace.path",
			wantErr:   true,
			errMsg:    "cannot be empty",
		},
		{
			name:      "path with double dot (path traversal)",
			path:      "/tmp/../etc",
			fieldName: "workspace.path",
			wantErr:   true,
			errMsg:    "path traversal",
		},
		{
			name:      "relative path with double dot",
			path:      "../etc",
			fieldName: "workspace.path",
			wantErr:   true,
			errMsg:    "path traversal",
		},
		{
			name:      "path with triple dot (contains ..)",
			path:      "/tmp/.../file",
			fieldName: "workspace.path",
			wantErr:   true,
			errMsg:    "path traversal",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePath(tt.path, tt.fieldName)
			if (err != nil) != tt.wantErr {
				t.Errorf("validatePath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errMsg != "" && err != nil {
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("validatePath() error = %v, want error message to contain %q", err, tt.errMsg)
				}
			}
		})
	}
}

// Tests for masking functions
func TestMaskSecret(t *testing.T) {
	tests := []struct {
		name     string
		secret   string
		expected string
	}{
		{
			name:     "empty secret",
			secret:   "",
			expected: "",
		},
		{
			name:     "short secret (less than 8 chars)",
			secret:   "abc",
			expected: "***",
		},
		{
			name:     "secret with 8 chars",
			secret:   "abcdefgh",
			expected: "abcdefgh", // 8 chars - no middle to mask
		},
		{
			name:     "long secret",
			secret:   "zai-test-api-key-12345678",
			expected: "zai-*****************5678", // 28 chars - 20 stars
		},
		{
			name:     "very long secret",
			secret:   "sk-abcdefghijklmnopqrstuvwxyz1234567890",
			expected: "sk-a*******************************7890", // 47 chars - 39 stars
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := maskSecret(tt.secret)
			if result != tt.expected {
				t.Errorf("maskSecret(%q) = %q, want %q", tt.secret, result, tt.expected)
			}
		})
	}
}

func TestMaskAPIKey(t *testing.T) {
	tests := []struct {
		name     string
		apiKey   string
		expected string
	}{
		{
			name:     "empty API key",
			apiKey:   "",
			expected: "",
		},
		{
			name:     "zai API key",
			apiKey:   "zai-test-api-key",
			expected: "zai-********-key", // 16 chars - 8 stars
		},
		{
			name:     "sk- prefixed API key",
			apiKey:   "sk-proj-abc123def456",
			expected: "sk-p************f456", // 20 chars - 12 stars
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := maskAPIKey(tt.apiKey)
			if result != tt.expected {
				t.Errorf("maskAPIKey(%q) = %q, want %q", tt.apiKey, result, tt.expected)
			}
		})
	}
}

func TestMaskTelegramToken(t *testing.T) {
	tests := []struct {
		name     string
		token    string
		expected string
	}{
		{
			name:     "empty token",
			token:    "",
			expected: "",
		},
		{
			name:     "valid token",
			token:    "1234567890:ABCdefGHIjklMNOpqrsTUVwxyz",
			expected: "1234567890:ABCd******************wxyz", // 10 + 24 chars - 16 stars
		},
		{
			name:     "short token",
			token:    "123:ABCDEFGHIJ",
			expected: "123:ABCD**GHIJ", // 3 + 8 chars - 2 stars
		},
		{
			name:     "invalid format (no colon)",
			token:    "invalid-token",
			expected: "inva*****oken", // 12 chars - 4 stars
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := maskTelegramToken(tt.token)
			if result != tt.expected {
				t.Errorf("maskTelegramToken(%q) = %q, want %q", tt.token, result, tt.expected)
			}
		})
	}
}

func TestFormatValidationError(t *testing.T) {
	tests := []struct {
		name           string
		field          string
		message        string
		secret         string
		wantErr        bool
		expectedSubstr string
	}{
		{
			name:           "error without secret",
			field:          "workspace.path",
			message:        "is required",
			wantErr:        true,
			expectedSubstr: "workspace.path: is required",
		},
		{
			name:           "error with secret",
			field:          "llm.zai.api_key",
			message:        "is too short",
			secret:         "zai-short",
			wantErr:        true,
			expectedSubstr: "value: zai-*hort", // 9 chars - 1 star in middle
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := formatValidationError(tt.field, tt.message, tt.secret)
			if (err != nil) != tt.wantErr {
				t.Errorf("formatValidationError() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.expectedSubstr != "" && err != nil {
				if !strings.Contains(err.Error(), tt.expectedSubstr) {
					t.Errorf("formatValidationError() error = %v, want to contain %q", err, tt.expectedSubstr)
				}
			}
		})
	}
}

func TestValidationError(t *testing.T) {
	err := &ValidationError{Field: "test.field", Message: "test.field: is invalid"}

	expected := "test.field: is invalid"
	if err.Error() != expected {
		t.Errorf("ValidationError.Error() = %q, want %q", err.Error(), expected)
	}
}
