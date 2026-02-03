package main

import (
	"testing"
)

func TestCommandStructure(t *testing.T) {
	// Test that all commands are properly registered
	if rootCmd == nil {
		t.Error("rootCmd should not be nil")
	}

	// Check that subcommands are added
	subcommands := rootCmd.Commands()
	expectedCommands := []string{"version", "config", "serve", "test"}
	foundCommands := make(map[string]bool)

	for _, cmd := range subcommands {
		foundCommands[cmd.Name()] = true
	}

	for _, expected := range expectedCommands {
		if !foundCommands[expected] {
			t.Errorf("Expected command '%s' not found in rootCmd", expected)
		}
	}
}

func TestConfigSubcommands(t *testing.T) {
	// Test that config subcommands are properly registered
	if configCmd == nil {
		t.Error("configCmd should not be nil")
	}

	// Check that validate subcommand is added
	subcommands := configCmd.Commands()
	foundValidate := false

	for _, cmd := range subcommands {
		if cmd.Name() == "validate" {
			foundValidate = true
			break
		}
	}

	if !foundValidate {
		t.Error("Expected 'validate' subcommand not found in configCmd")
	}
}

func TestServeCmdFlags(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		wantConfig   string
		wantLogLevel string
	}{
		{
			name:         "with config flag",
			args:         []string{"--config", "test.toml"},
			wantConfig:   "test.toml",
			wantLogLevel: "",
		},
		{
			name:         "with log-level flag",
			args:         []string{"--log-level", "debug"},
			wantConfig:   "",
			wantLogLevel: "debug",
		},
		{
			name:         "with both flags",
			args:         []string{"--config", "test.toml", "--log-level", "info"},
			wantConfig:   "test.toml",
			wantLogLevel: "info",
		},
		{
			name:         "short flags",
			args:         []string{"-c", "test.toml", "-l", "warn"},
			wantConfig:   "test.toml",
			wantLogLevel: "warn",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset flags
			serveConfigPath = ""
			serveLogLevel = ""

			// Parse flags
			serveCmd.SetArgs(tt.args)
			_ = serveCmd.ParseFlags(tt.args)

			if serveConfigPath != tt.wantConfig {
				t.Errorf("serveConfigPath = %v, want %v", serveConfigPath, tt.wantConfig)
			}
			if serveLogLevel != tt.wantLogLevel {
				t.Errorf("serveLogLevel = %v, want %v", serveLogLevel, tt.wantLogLevel)
			}
		})
	}
}
