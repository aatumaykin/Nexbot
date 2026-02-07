package skills

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSummaryBuilder_FilterByCategory(t *testing.T) {
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
---

`

	dockerContent := `---
name: docker-build
description: Docker build
category: docker
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
	builder := NewSummaryBuilder(loader)

	opts := SummaryOptions{
		Format:     "short",
		Categories: []string{"git"},
	}

	summary, err := builder.Build(opts)
	if err != nil {
		t.Fatalf("Failed to build summary: %v", err)
	}

	// Should only contain git skills
	if !strings.Contains(summary, "git-commit") {
		t.Error("Expected summary to contain git skill")
	}

	if strings.Contains(summary, "docker-build") {
		t.Error("Expected summary to not contain docker skill")
	}
}

func TestSummaryBuilder_MaxSkills(t *testing.T) {
	tmpDir := t.TempDir()

	builtinDir := filepath.Join(tmpDir, "builtin", "skills")
	if err := os.MkdirAll(builtinDir, 0755); err != nil {
		t.Fatalf("Failed to create builtin directory: %v", err)
	}

	// Create skills in subdirectories to be loaded as SKILL.md files
	for i := 1; i <= 5; i++ {
		skillDir := filepath.Join(builtinDir, fmt.Sprintf("skill%d", i))
		if err := os.MkdirAll(skillDir, 0755); err != nil {
			t.Fatalf("Failed to create skill directory %d: %v", i, err)
		}

		content := fmt.Sprintf(`---
name: skill-%d
description: Skill number %d
---
`, i, i)

		if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write skill %d: %v", i, err)
		}
	}

	cfg := LoaderConfig{
		BuiltinDir:   builtinDir,
		CacheEnabled: false,
	}

	loader := NewLoader(cfg)
	builder := NewSummaryBuilder(loader)

	opts := SummaryOptions{
		Format:    "short",
		MaxSkills: 3,
	}

	summary, err := builder.Build(opts)
	if err != nil {
		t.Fatalf("Failed to build summary: %v", err)
	}

	// Should only contain first 3 skills
	if !strings.Contains(summary, "skill-1") {
		t.Error("Expected summary to contain skill-1")
	}

	if !strings.Contains(summary, "skill-3") {
		t.Error("Expected summary to contain skill-3")
	}

	if strings.Contains(summary, "skill-4") {
		t.Error("Expected summary to not contain skill-4 (exceeds MaxSkills)")
	}
}
