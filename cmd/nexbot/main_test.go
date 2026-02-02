package main

import (
	"testing"
)

func TestRunCmdFlags(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantConfig string
		wantDebug  bool
	}{
		{
			name:       "with config flag",
			args:       []string{"--config", "test.toml"},
			wantConfig: "test.toml",
			wantDebug:  false,
		},
		{
			name:       "with debug flag",
			args:       []string{"--debug"},
			wantConfig: "",
			wantDebug:  true,
		},
		{
			name:       "with both flags",
			args:       []string{"--config", "test.toml", "--debug"},
			wantConfig: "test.toml",
			wantDebug:  true,
		},
		{
			name:       "short flags",
			args:       []string{"-c", "test.toml", "-d"},
			wantConfig: "test.toml",
			wantDebug:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset flags
			runConfigPath = ""
			runDebug = false

			// Parse flags
			runCmd.SetArgs(tt.args)
			_ = runCmd.ParseFlags(tt.args)

			if runConfigPath != tt.wantConfig {
				t.Errorf("runConfigPath = %v, want %v", runConfigPath, tt.wantConfig)
			}
			if runDebug != tt.wantDebug {
				t.Errorf("runDebug = %v, want %v", runDebug, tt.wantDebug)
			}
		})
	}
}

func TestCommandStructure(t *testing.T) {
	// Test that all commands are properly registered
	if rootCmd == nil {
		t.Error("rootCmd should not be nil")
	}

	// Check that subcommands are added
	subcommands := rootCmd.Commands()
	expectedCommands := []string{"version", "config", "run"}
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
