package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadEnv(t *testing.T) {
	// Создаем временный каталог для тестов
	tmpDir := t.TempDir()

	tests := []struct {
		name     string
		content  string
		setupEnv map[string]string
		wantEnv  map[string]string
		wantErr  bool
	}{
		{
			name: "valid .env file",
			content: `
# Comment line
KEY1=value1
KEY2=value2

KEY3=value with spaces
`,
			wantEnv: map[string]string{
				"KEY1": "value1",
				"KEY2": "value2",
				"KEY3": "value with spaces",
			},
			wantErr: false,
		},
		{
			name:    "empty file",
			content: "",
			wantEnv: map[string]string{},
			wantErr: false,
		},
		{
			name: "only comments",
			content: `
# This is a comment
# Another comment
`,
			wantEnv: map[string]string{},
			wantErr: false,
		},
		{
			name: "file with empty lines",
			content: `
KEY1=value1


KEY2=value2

`,
			wantEnv: map[string]string{
				"KEY1": "value1",
				"KEY2": "value2",
			},
			wantErr: false,
		},
		{
			name: "values with special characters",
			content: `
API_KEY=sk-1234567890abcdef
DATABASE_URL=postgres://user:pass@localhost:5432/db
PORT=8080
DEBUG=true
`,
			wantEnv: map[string]string{
				"API_KEY":      "sk-1234567890abcdef",
				"DATABASE_URL": "postgres://user:pass@localhost:5432/db",
				"PORT":         "8080",
				"DEBUG":        "true",
			},
			wantErr: false,
		},
		{
			name:    "overwrites existing env vars",
			content: `KEY1=newvalue`,
			setupEnv: map[string]string{
				"KEY1": "oldvalue",
			},
			wantEnv: map[string]string{
				"KEY1": "newvalue",
			},
			wantErr: false,
		},
		{
			name:    "file does not exist",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Удаляем переменные окружения, которые могут влиять на тест
			cleanupEnv := func() {
				keys := []string{"KEY1", "KEY2", "KEY3", "API_KEY", "DATABASE_URL", "PORT", "DEBUG"}
				for _, key := range keys {
					os.Unsetenv(key)
				}
			}

			cleanupEnv()
			defer cleanupEnv()

			// Устанавливаем начальные значения переменных окружения
			if tt.setupEnv != nil {
				for key, value := range tt.setupEnv {
					os.Setenv(key, value)
				}
			}

			var envPath string
			if !tt.wantErr {
				envPath = filepath.Join(tmpDir, ".env")
				if err := os.WriteFile(envPath, []byte(tt.content), 0600); err != nil {
					t.Fatalf("failed to write .env file: %v", err)
				}
			} else {
				envPath = filepath.Join(tmpDir, "nonexistent.env")
			}

			err := LoadEnv(envPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadEnv() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			// Проверяем переменные окружения
			for key, wantValue := range tt.wantEnv {
				gotValue := os.Getenv(key)
				if gotValue != wantValue {
					t.Errorf("os.Getenv(%q) = %q, want %q", key, gotValue, wantValue)
				}
			}
		})
	}
}

func TestLoadEnvOptional(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name    string
		content string
		wantEnv map[string]string
		wantErr bool
	}{
		{
			name: "file exists",
			content: `KEY1=value1
KEY2=value2`,
			wantEnv: map[string]string{
				"KEY1": "value1",
				"KEY2": "value2",
			},
			wantErr: false,
		},
		{
			name:    "file does not exist",
			wantEnv: map[string]string{},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Очистка
			cleanupEnv := func() {
				os.Unsetenv("KEY1")
				os.Unsetenv("KEY2")
			}

			cleanupEnv()
			defer cleanupEnv()

			var envPath string
			if tt.content != "" {
				envPath = filepath.Join(tmpDir, ".env")
				if err := os.WriteFile(envPath, []byte(tt.content), 0600); err != nil {
					t.Fatalf("failed to write .env file: %v", err)
				}
			} else {
				envPath = filepath.Join(tmpDir, "nonexistent.env")
			}

			err := LoadEnvOptional(envPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadEnvOptional() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Проверяем переменные окружения
			for key, wantValue := range tt.wantEnv {
				gotValue := os.Getenv(key)
				if gotValue != wantValue {
					t.Errorf("os.Getenv(%q) = %q, want %q", key, gotValue, wantValue)
				}
			}
		})
	}
}
