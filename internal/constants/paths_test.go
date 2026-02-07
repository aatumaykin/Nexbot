package constants

import (
	"testing"
)

func TestPathConstants(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{
			name:  "DefaultEnvPath",
			value: DefaultEnvPath,
		},
		{
			name:  "DefaultConfigPath",
			value: DefaultConfigPath,
		},
		{
			name:  "DefaultWorkDir",
			value: DefaultWorkDir,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value == "" {
				t.Errorf("%s should not be empty", tt.name)
			}
		})
	}
}

func TestDefaultEnvPath(t *testing.T) {
	if DefaultEnvPath != "~/.config/nexbot/.env" {
		t.Errorf("DefaultEnvPath = %s, want '~/.config/nexbot/.env'", DefaultEnvPath)
	}

	// Check that path starts with ~/ (home directory)
	if len(DefaultEnvPath) < 2 || DefaultEnvPath[0:2] != "~/" {
		t.Errorf("DefaultEnvPath should start with '~/', got: %s", DefaultEnvPath)
	}

	// Check that it has .env extension
	if len(DefaultEnvPath) < 5 || DefaultEnvPath[len(DefaultEnvPath)-4:] != ".env" {
		t.Errorf("DefaultEnvPath should have .env extension, got: %s", DefaultEnvPath)
	}
}

func TestDefaultConfigPath(t *testing.T) {
	if DefaultConfigPath != "~/.config/nexbot/config.toml" {
		t.Errorf("DefaultConfigPath = %s, want '~/.config/nexbot/config.toml'", DefaultConfigPath)
	}

	// Check that path starts with ~/ (home directory)
	if len(DefaultConfigPath) < 2 || DefaultConfigPath[0:2] != "~/" {
		t.Errorf("DefaultConfigPath should start with '~/', got: %s", DefaultConfigPath)
	}

	// Check that it has .toml extension
	if len(DefaultConfigPath) < 6 || DefaultConfigPath[len(DefaultConfigPath)-5:] != ".toml" {
		t.Errorf("DefaultConfigPath should have .toml extension, got: %s", DefaultConfigPath)
	}
}

func TestDefaultWorkDir(t *testing.T) {
	if DefaultWorkDir != "." {
		t.Errorf("DefaultWorkDir = %s, want '.'", DefaultWorkDir)
	}

	// Check that it's a valid path reference
	if DefaultWorkDir == "" {
		t.Errorf("DefaultWorkDir should not be empty")
	}
}

func TestPathExtensions(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		extension string
	}{
		{
			name:      "DefaultEnvPath extension",
			value:     DefaultEnvPath,
			extension: ".env",
		},
		{
			name:      "DefaultConfigPath extension",
			value:     DefaultConfigPath,
			extension: ".toml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if len(tt.value) < len(tt.extension) || tt.value[len(tt.value)-len(tt.extension):] != tt.extension {
				t.Errorf("Expected extension %s, got path: %s", tt.extension, tt.value)
			}
		})
	}
}

func TestPathConsistency(t *testing.T) {
	// Test that all paths use consistent format
	paths := []string{DefaultEnvPath, DefaultConfigPath}

	for i, path := range paths {
		// All paths should start with ~/ (home directory)
		if len(path) >= 2 && path[0:2] != "~/" {
			t.Errorf("Path at index %d should start with '~/', got: %s", i, path)
		}
	}
}
