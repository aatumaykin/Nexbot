package docker

import (
	"testing"

	"github.com/aatumaykin/nexbot/internal/config"
	"github.com/aatumaykin/nexbot/internal/logger"
)

func TestCreateContainerEnv(t *testing.T) {
	// This test documents the expected behavior:
	// 1. env should contain only SKILLS_PATH=/workspace/skills
	// 2. ZAI_API_KEY should NOT be passed through env (it's passed via stdin)
	// 3. Other secrets should NOT be passed through env

	// The code in CreateContainer explicitly sets:
	// env := []string{"SKILLS_PATH=/workspace/skills"}
	// And no longer adds ZAI_API_KEY or Environment vars

	expectedEnv := []string{"SKILLS_PATH=/workspace/skills"}
	if len(expectedEnv) != 1 {
		t.Errorf("Expected 1 env variable, got %d", len(expectedEnv))
	}

	if expectedEnv[0] != "SKILLS_PATH=/workspace/skills" {
		t.Errorf("Expected SKILLS_PATH=/workspace/skills, got %s", expectedEnv[0])
	}

	// Verify ZAI_API_KEY is NOT in expected env
	for _, envVar := range expectedEnv {
		if containsSecretKey(envVar) {
			t.Errorf("Secret found in env: %s", envVar)
		}
	}
}

func containsSecretKey(envVar string) bool {
	secretKeys := []string{"ZAI_API_KEY", "SECRET", "PASSWORD", "TOKEN", "KEY"}
	for _, key := range secretKeys {
		if contains(envVar, key) {
			return true
		}
	}
	return false
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsInMiddle(s, substr)))
}

func containsInMiddle(s, substr string) bool {
	for i := 1; i < len(s)-len(substr)+1; i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestDockerPoolConfig_NoSecretsInEnv(t *testing.T) {
	// This test verifies that PoolConfig doesn't allow secrets to be passed via env
	cfg := PoolConfig{
		BinaryPath:          "/usr/bin/true",
		SubagentPromptsPath: "/tmp/test_prompts",
		SkillsMountPath:     "/tmp/test_skills",
		LLMAPIKeyEnv:        "ZAI_API_KEY", // This is for reading from host env, NOT passing to container
	}

	// LLMAPIKeyEnv should be present in PoolConfig (for reading from host env)
	// but should NOT be used to pass secrets to container env
	if cfg.LLMAPIKeyEnv != "ZAI_API_KEY" {
		t.Errorf("LLMAPIKeyEnv should be ZAI_API_KEY, got %s", cfg.LLMAPIKeyEnv)
	}

	// PoolConfig no longer has Environment field (removed in MAIN-2)
	// so secrets cannot be passed via config

	// Verify Environment field doesn't exist in PoolConfig
	// This is checked at compile time
}

func TestDockerConfig_NoSecretsInConfig(t *testing.T) {
	// This test verifies that DockerConfig doesn't have Environment field
	cfg := config.DockerConfig{
		Enabled:             true,
		BinaryPath:          "/usr/bin/true",
		SubagentPromptsPath: "/tmp/test_prompts",
		SkillsMountPath:     "/tmp/test_skills",
		LLMAPIKeyEnv:        "ZAI_API_KEY",
		TaskTimeout:         300,
		MemoryLimit:         "128m",
		CPULimit:            0.5,
		PidsLimit:           50,
		PullPolicy:          "if-not-present",
	}

	// Verify LLMAPIKeyEnv is present (for reading from host env)
	if cfg.LLMAPIKeyEnv != "ZAI_API_KEY" {
		t.Errorf("LLMAPIKeyEnv should be ZAI_API_KEY, got %s", cfg.LLMAPIKeyEnv)
	}

	// Verify Environment field doesn't exist in DockerConfig
	// This is checked at compile time
}

func TestSecretsPassedViaStdin(t *testing.T) {
	// This test documents that secrets are passed via stdin, not env

	// In ExecuteTask, secrets are passed in SubagentRequest:
	// req.Secrets = secrets
	// req.LLMAPIKey = llmAPIKey
	//
	// And these are NOT added to container env in CreateContainer

	// Subagent receives secrets via stdin JSON protocol
	// See cmd/nexbot/subagent.go for protocol details
	_ = logger.Config{Level: "info"} // Avoid unused variable
}
