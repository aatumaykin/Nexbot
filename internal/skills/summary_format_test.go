package skills

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSummaryBuilder_GroupByCategory(t *testing.T) {
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
		Format: "medium",
	}

	summary, err := builder.Build(opts)
	if err != nil {
		t.Fatalf("Failed to build summary: %v", err)
	}

	// Check that categories are present
	gitSection := strings.Index(summary, "### git")
	dockerSection := strings.Index(summary, "### docker")

	if gitSection == -1 {
		t.Error("Expected summary to contain git category section")
	}

	if dockerSection == -1 {
		t.Error("Expected summary to contain docker category section")
	}

	// Check order (should be alphabetical: docker < git alphabetically)
	if gitSection < dockerSection {
		t.Error("Expected categories to be sorted alphabetically (docker before git)")
	}
}

func TestSummaryBuilder_PreviewContent(t *testing.T) {
	tmpDir := t.TempDir()

	builtinDir := filepath.Join(tmpDir, "builtin", "skills")
	if err := os.MkdirAll(builtinDir, 0755); err != nil {
		t.Fatalf("Failed to create builtin directory: %v", err)
	}

	skillContent := `---
name: test-skill
description: Test skill
---

This is paragraph 1.

This is paragraph 2.

This is paragraph 3.

This is paragraph 4.

`

	if err := os.WriteFile(filepath.Join(builtinDir, "SKILL.md"), []byte(skillContent), 0644); err != nil {
		t.Fatalf("Failed to write skill: %v", err)
	}

	cfg := LoaderConfig{
		BuiltinDir:   builtinDir,
		CacheEnabled: false,
	}

	loader := NewLoader(cfg)
	builder := NewSummaryBuilder(loader)

	opts := SummaryOptions{
		Format: "long",
	}

	summary, err := builder.Build(opts)
	if err != nil {
		t.Fatalf("Failed to build summary: %v", err)
	}

	// Check that preview includes only first 3 paragraphs
	if !strings.Contains(summary, "paragraph 3") {
		t.Error("Expected content preview to include 3rd paragraph")
	}

	if strings.Contains(summary, "paragraph 4") {
		t.Error("Expected content preview to not include 4th paragraph")
	}
}
