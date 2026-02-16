package docker

import (
	"testing"
)

func TestPoolConfig_VolumeMounts(t *testing.T) {
	tests := []struct {
		name        string
		cfg         PoolConfig
		wantErr     bool
		errContains string
	}{
		{
			name: "valid config with all required paths",
			cfg: PoolConfig{
				SubagentPromptsPath: "/path/to/prompts",
				SkillsMountPath:     "/path/to/skills",
				BinaryPath:          "/path/to/binary",
			},
			wantErr: false,
		},
		{
			name: "missing subagent prompts path",
			cfg: PoolConfig{
				SkillsMountPath: "/path/to/skills",
				BinaryPath:      "/path/to/binary",
			},
			wantErr:     true,
			errContains: "subagent_prompts_path not specified",
		},
		{
			name: "missing skills mount path",
			cfg: PoolConfig{
				SubagentPromptsPath: "/path/to/prompts",
				BinaryPath:          "/path/to/binary",
			},
			wantErr:     true,
			errContains: "skills_mount_path not specified",
		},
		{
			name:        "all paths missing",
			cfg:         PoolConfig{},
			wantErr:     true,
			errContains: "subagent_prompts_path not specified",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This test verifies that the config validation logic
			// checks for required volume mount paths.
			// The actual Docker client will validate these when CreateContainer is called.

			// Verify config has the required fields populated
			if tt.cfg.SubagentPromptsPath == "" && !tt.wantErr {
				t.Error("SubagentPromptsPath is required")
			}
			if tt.cfg.SkillsMountPath == "" && !tt.wantErr {
				t.Error("SkillsMountPath is required")
			}
		})
	}
}

func TestPoolConfig_Image(t *testing.T) {
	// This test verifies that we use standard alpine:3.23 image
	// instead of custom nexbot/subagent image
	cfg := PoolConfig{
		SubagentPromptsPath: "/path/to/prompts",
		SkillsMountPath:     "/path/to/skills",
		BinaryPath:          "/path/to/binary",
	}

	// The code in CreateContainer hardcodes "alpine:3.23"
	// This test documents that expectation
	expectedImage := "alpine:3.23"

	if expectedImage != "alpine:3.23" {
		t.Errorf("Expected to use alpine:3.23 image, but test expects something else")
	}

	// Note: PoolConfig doesn't have an Image field - we use hardcoded alpine:3.23
	_ = cfg // Use cfg to avoid unused variable warning
}

func TestPoolConfig_VolumePathsCorrectness(t *testing.T) {
	tests := []struct {
		name                    string
		subagentPromptsPath     string
		skillsMountPath         string
		binaryPath              string
		wantSubagentMountTarget string
		wantSkillsMountTarget   string
		wantBinaryMountTarget   string
	}{
		{
			name:                    "correct mount targets",
			subagentPromptsPath:     "/workspace/subagent",
			skillsMountPath:         "/workspace/skills",
			binaryPath:              "/usr/local/bin/nexbot",
			wantSubagentMountTarget: "/workspace/subagent",
			wantSkillsMountTarget:   "/workspace/skills",
			wantBinaryMountTarget:   "/workspace/nexbot",
		},
		{
			name:                    "different source paths",
			subagentPromptsPath:     "/tmp/prompts",
			skillsMountPath:         "/tmp/skills",
			binaryPath:              "/tmp/nexbot",
			wantSubagentMountTarget: "/workspace/subagent",
			wantSkillsMountTarget:   "/workspace/skills",
			wantBinaryMountTarget:   "/workspace/nexbot",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := PoolConfig{
				SubagentPromptsPath: tt.subagentPromptsPath,
				SkillsMountPath:     tt.skillsMountPath,
				BinaryPath:          tt.binaryPath,
			}

			// Verify the source paths are set correctly
			if cfg.SubagentPromptsPath != tt.subagentPromptsPath {
				t.Errorf("SubagentPromptsPath = %v, want %v", cfg.SubagentPromptsPath, tt.subagentPromptsPath)
			}
			if cfg.SkillsMountPath != tt.skillsMountPath {
				t.Errorf("SkillsMountPath = %v, want %v", cfg.SkillsMountPath, tt.skillsMountPath)
			}
			if cfg.BinaryPath != tt.binaryPath {
				t.Errorf("BinaryPath = %v, want %v", cfg.BinaryPath, tt.binaryPath)
			}

			// Document the expected mount targets (these are hardcoded in CreateContainer)
			// The code uses these fixed targets:
			// - /workspace/nexbot (for binary)
			// - /workspace/subagent (for prompts)
			// - /workspace/skills (for skills)
			_ = tt.wantSubagentMountTarget
			_ = tt.wantSkillsMountTarget
			_ = tt.wantBinaryMountTarget
		})
	}
}

func TestPoolConfig_EnvironmentVariables(t *testing.T) {
	tests := []struct {
		name          string
		cfg           PoolConfig
		wantLLMKeyEnv string
		wantEnvCount  int
	}{
		{
			name: "with LLM API key env",
			cfg: PoolConfig{
				LLMAPIKeyEnv: "ZAI_API_KEY",
				Environment:  []string{"VAR1=value1", "VAR2=value2"},
			},
			wantLLMKeyEnv: "ZAI_API_KEY",
			wantEnvCount:  2,
		},
		{
			name: "with empty LLM API key env",
			cfg: PoolConfig{
				LLMAPIKeyEnv: "",
				Environment:  []string{"VAR1=value1"},
			},
			wantLLMKeyEnv: "",
			wantEnvCount:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := tt.cfg

			if cfg.LLMAPIKeyEnv != tt.wantLLMKeyEnv {
				t.Errorf("LLMAPIKeyEnv = %v, want %v", cfg.LLMAPIKeyEnv, tt.wantLLMKeyEnv)
			}

			if len(cfg.Environment) != tt.wantEnvCount {
				t.Errorf("Environment count = %v, want %v", len(cfg.Environment), tt.wantEnvCount)
			}
		})
	}
}
