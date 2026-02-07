package skills

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoader_ValidateAll(t *testing.T) {
	tmpDir := t.TempDir()

	builtinDir := filepath.Join(tmpDir, "builtin", "skills")
	if err := os.MkdirAll(builtinDir, 0755); err != nil {
		t.Fatalf("Failed to create builtin directory: %v", err)
	}

	validContent := `---
name: valid-skill
description: A valid skill
---

Valid content.
`

	invalidContent := `---
description: Invalid skill (missing name)
---

Invalid content.
`

	if err := os.WriteFile(filepath.Join(builtinDir, "valid.md"), []byte(validContent), 0644); err != nil {
		t.Fatalf("Failed to write valid skill: %v", err)
	}
	if err := os.WriteFile(filepath.Join(builtinDir, "invalid.md"), []byte(invalidContent), 0644); err != nil {
		t.Fatalf("Failed to write invalid skill: %v", err)
	}

	cfg := LoaderConfig{
		BuiltinDir:   builtinDir,
		CacheEnabled: false,
	}

	loader := NewLoader(cfg)
	errors, err := loader.ValidateAll()

	if err != nil {
		t.Fatalf("Failed to validate skills: %v", err)
	}

	// No errors because files are not SKILL.md
	if len(errors) != 0 {
		t.Errorf("Expected 0 validation errors (files are not SKILL.md), got %d", len(errors))
	}
}

func TestLoader_Stats(t *testing.T) {
	tmpDir := t.TempDir()

	builtinDir := filepath.Join(tmpDir, "builtin", "skills")
	gitDir := filepath.Join(builtinDir, "git")
	dockerDir := filepath.Join(builtinDir, "docker")

	if err := os.MkdirAll(gitDir, 0755); err != nil {
		t.Fatalf("Failed to create git directory: %v", err)
	}
	if err := os.MkdirAll(dockerDir, 0755); err != nil {
		t.Fatalf("Failed to create docker directory: %v", err)
	}

	gitContent := `---
name: git-commit
description: Git commit
category: git
parameters:
  - name: message
    type: string
    description: Commit message
    required: true
examples:
  - name: Example 1
    input: "test"
    description: Example description
---
`

	dockerContent := `---
name: docker-build
description: Docker build
category: docker
parameters:
  - name: context
    type: string
    description: Build context
    required: true
  - name: tag
    type: string
    description: Image tag
    required: false
examples:
  - name: Example 1
    input: "test"
    description: Example description
  - name: Example 2
    input: "test2"
    description: Example description 2
---
`

	if err := os.WriteFile(filepath.Join(gitDir, "SKILL.md"), []byte(gitContent), 0644); err != nil {
		t.Fatalf("Failed to write git skill: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dockerDir, "SKILL.md"), []byte(dockerContent), 0644); err != nil {
		t.Fatalf("Failed to write docker skill: %v", err)
	}

	cfg := LoaderConfig{
		BuiltinDir:   builtinDir,
		CacheEnabled: false,
	}

	loader := NewLoader(cfg)
	stats, err := loader.Stats()

	if err != nil {
		t.Fatalf("Failed to get stats: %v", err)
	}

	if stats.Total != 2 {
		t.Errorf("Expected 2 total skills, got %d", stats.Total)
	}

	if stats.ParameterCount != 3 {
		t.Errorf("Expected 3 parameters total, got %d", stats.ParameterCount)
	}

	if stats.ExampleCount != 3 {
		t.Errorf("Expected 3 examples total, got %d", stats.ExampleCount)
	}

	if stats.Categories["git"] != 1 {
		t.Errorf("Expected 1 git skill, got %d", stats.Categories["git"])
	}

	if stats.Categories["docker"] != 1 {
		t.Errorf("Expected 1 docker skill, got %d", stats.Categories["docker"])
	}
}

func TestLoader_DuplicateSkillName(t *testing.T) {
	tmpDir := t.TempDir()

	builtinDir := filepath.Join(tmpDir, "builtin", "skills")
	gitDir := filepath.Join(builtinDir, "git")
	anotherDir := filepath.Join(builtinDir, "another")

	if err := os.MkdirAll(gitDir, 0755); err != nil {
		t.Fatalf("Failed to create git directory: %v", err)
	}
	if err := os.MkdirAll(anotherDir, 0755); err != nil {
		t.Fatalf("Failed to create another directory: %v", err)
	}

	gitContent := `---
name: test-skill
description: First skill
---
`

	anotherContent := `---
name: test-skill
description: Second skill
---
`

	if err := os.WriteFile(filepath.Join(gitDir, "SKILL.md"), []byte(gitContent), 0644); err != nil {
		t.Fatalf("Failed to write first skill: %v", err)
	}
	if err := os.WriteFile(filepath.Join(anotherDir, "SKILL.md"), []byte(anotherContent), 0644); err != nil {
		t.Fatalf("Failed to write second skill: %v", err)
	}

	cfg := LoaderConfig{
		BuiltinDir:   builtinDir,
		CacheEnabled: false,
	}

	loader := NewLoader(cfg)
	_, err := loader.Load()

	if err == nil {
		t.Error("Expected error for duplicate skill names")
	}

	if !strings.Contains(err.Error(), "duplicate") {
		t.Errorf("Expected error to mention 'duplicate', got: %v", err)
	}
}

func TestLoader_ClearCache(t *testing.T) {
	tmpDir := t.TempDir()

	builtinDir := filepath.Join(tmpDir, "builtin", "skills")
	if err := os.MkdirAll(builtinDir, 0755); err != nil {
		t.Fatalf("Failed to create builtin directory: %v", err)
	}

	skillContent := `---
name: test-skill
description: Test skill
---
`

	if err := os.WriteFile(filepath.Join(builtinDir, "SKILL.md"), []byte(skillContent), 0644); err != nil {
		t.Fatalf("Failed to write skill: %v", err)
	}

	cfg := LoaderConfig{
		BuiltinDir:   builtinDir,
		CacheEnabled: true,
	}

	loader := NewLoader(cfg)
	skills, err := loader.Load()

	if err != nil {
		t.Fatalf("Failed to load skills: %v", err)
	}

	if len(skills) != 1 {
		t.Errorf("Expected 1 skill, got %d", len(skills))
	}

	// Clear cache
	loader.ClearCache()

	// Load again - should work fine
	skills, err = loader.Load()

	if err != nil {
		t.Fatalf("Failed to load skills after clearing cache: %v", err)
	}

	if len(skills) != 1 {
		t.Errorf("Expected 1 skill after reload, got %d", len(skills))
	}
}
