package skills

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoader_Get(t *testing.T) {
	tmpDir := t.TempDir()

	builtinDir := filepath.Join(tmpDir, "builtin", "skills")
	if err := os.MkdirAll(builtinDir, 0755); err != nil {
		t.Fatalf("Failed to create builtin directory: %v", err)
	}

	skillContent := `---
name: test-skill
description: A test skill
---

Test content.
`

	skillPath := filepath.Join(builtinDir, "SKILL.md")
	if err := os.WriteFile(skillPath, []byte(skillContent), 0644); err != nil {
		t.Fatalf("Failed to write skill: %v", err)
	}

	cfg := LoaderConfig{
		BuiltinDir:   builtinDir,
		CacheEnabled: false,
	}

	loader := NewLoader(cfg)
	skill, err := loader.Get("test-skill")

	if err != nil {
		t.Fatalf("Failed to get skill: %v", err)
	}

	if skill.Metadata.Name != "test-skill" {
		t.Errorf("Expected skill name 'test-skill', got '%s'", skill.Metadata.Name)
	}
}

func TestLoader_Get_NotFound(t *testing.T) {
	tmpDir := t.TempDir()

	builtinDir := filepath.Join(tmpDir, "builtin", "skills")
	if err := os.MkdirAll(builtinDir, 0755); err != nil {
		t.Fatalf("Failed to create builtin directory: %v", err)
	}

	cfg := LoaderConfig{
		BuiltinDir:   builtinDir,
		CacheEnabled: false,
	}

	loader := NewLoader(cfg)
	skill, err := loader.Get("nonexistent-skill")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if skill != nil {
		t.Error("Expected nil skill for non-existent skill")
	}
}

func TestLoader_List(t *testing.T) {
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
	names, err := loader.List()

	if err != nil {
		t.Fatalf("Failed to list skills: %v", err)
	}

	if len(names) != 2 {
		t.Errorf("Expected 2 skill names, got %d", len(names))
	}

	// Check that both names are present
	hasGit := false
	hasDocker := false
	for _, name := range names {
		if name == "git-commit" {
			hasGit = true
		}
		if name == "docker-build" {
			hasDocker = true
		}
	}

	if !hasGit {
		t.Error("Expected 'git-commit' in list")
	}
	if !hasDocker {
		t.Error("Expected 'docker-build' in list")
	}
}

func TestLoader_Reload(t *testing.T) {
	tmpDir := t.TempDir()

	builtinDir := filepath.Join(tmpDir, "builtin", "skills")
	if err := os.MkdirAll(builtinDir, 0755); err != nil {
		t.Fatalf("Failed to create builtin directory: %v", err)
	}

	skillContent := `---
name: test-skill
description: Initial version
---

Initial content.
`

	skillPath := filepath.Join(builtinDir, "SKILL.md")
	if err := os.WriteFile(skillPath, []byte(skillContent), 0644); err != nil {
		t.Fatalf("Failed to write skill: %v", err)
	}

	cfg := LoaderConfig{
		BuiltinDir:   builtinDir,
		CacheEnabled: false,
	}

	loader := NewLoader(cfg)
	skills, err := loader.Load()

	if err != nil {
		t.Fatalf("Failed to load skills: %v", err)
	}

	if skills["test-skill"].Metadata.Description != "Initial version" {
		t.Errorf("Expected 'Initial version', got '%s'", skills["test-skill"].Metadata.Description)
	}

	// Modify skill file
	newContent := `---
name: test-skill
description: Updated version
---

Updated content.
`

	if err := os.WriteFile(skillPath, []byte(newContent), 0644); err != nil {
		t.Fatalf("Failed to update skill: %v", err)
	}

	// Reload skills
	skills, err = loader.Reload()

	if err != nil {
		t.Fatalf("Failed to reload skills: %v", err)
	}

	if skills["test-skill"].Metadata.Description != "Updated version" {
		t.Errorf("Expected 'Updated version', got '%s'", skills["test-skill"].Metadata.Description)
	}
}

func TestLoader_GetSkillsByCategory(t *testing.T) {
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

	gitStatusContent := `---
name: git-status
description: Git status
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
	_ = os.WriteFile(filepath.Join(gitDir, "git-status.md"), []byte(gitStatusContent), 0644)
	if err := os.WriteFile(filepath.Join(dockerDir, "SKILL.md"), []byte(dockerContent), 0644); err != nil {
		t.Fatalf("Failed to write docker skill: %v", err)
	}

	cfg := LoaderConfig{
		BuiltinDir:   builtinDir,
		CacheEnabled: false,
	}

	loader := NewLoader(cfg)
	gitSkills, err := loader.GetSkillsByCategory("git")

	if err != nil {
		t.Fatalf("Failed to get skills by category: %v", err)
	}

	if len(gitSkills) != 1 {
		t.Errorf("Expected 1 git skill, got %d", len(gitSkills))
	}
}

func TestLoader_GetSkillsByTags(t *testing.T) {
	tmpDir := t.TempDir()

	builtinDir := filepath.Join(tmpDir, "builtin", "skills")
	if err := os.MkdirAll(builtinDir, 0755); err != nil {
		t.Fatalf("Failed to create builtin directory: %v", err)
	}

	skill1 := `---
name: git-skill
description: Git related skill
tags:
  - git
  - version-control
---
`

	skill2 := `---
name: docker-skill
description: Docker related skill
tags:
  - docker
  - container
---
`

	if err := os.WriteFile(filepath.Join(builtinDir, "git.md"), []byte(skill1), 0644); err != nil {
		t.Fatalf("Failed to write git skill: %v", err)
	}
	if err := os.WriteFile(filepath.Join(builtinDir, "docker.md"), []byte(skill2), 0644); err != nil {
		t.Fatalf("Failed to write docker skill: %v", err)
	}

	cfg := LoaderConfig{
		BuiltinDir:   builtinDir,
		CacheEnabled: false,
	}

	loader := NewLoader(cfg)
	skills, err := loader.GetSkillsByTags([]string{"git"})

	if err != nil {
		t.Fatalf("Failed to get skills by tags: %v", err)
	}

	if len(skills) != 0 {
		// git.md and docker.md are not SKILL.md files, so they won't be loaded
		t.Errorf("Expected 0 skills (files are not SKILL.md), got %d", len(skills))
	}

	// Now create proper SKILL.md files
	if err := os.WriteFile(filepath.Join(builtinDir, "SKILL.md"), []byte(skill1), 0644); err != nil {
		t.Fatalf("Failed to write SKILL.md: %v", err)
	}

	_, err = loader.Reload()
	if err != nil {
		t.Fatalf("Failed to reload skills: %v", err)
	}

	// Get skills again
	skills, err = loader.GetSkillsByTags([]string{"git"})
	if err != nil {
		t.Fatalf("Failed to get skills by tags: %v", err)
	}

	if len(skills) != 1 {
		t.Errorf("Expected 1 skill, got %d", len(skills))
	}
}

func TestLoader_SearchSkills(t *testing.T) {
	tmpDir := t.TempDir()

	builtinDir := filepath.Join(tmpDir, "builtin", "skills")
	if err := os.MkdirAll(builtinDir, 0755); err != nil {
		t.Fatalf("Failed to create builtin directory: %v", err)
	}

	skillContent := `---
name: git-commit
description: Commit changes to git repository
tags:
  - git
  - version-control
---

This skill commits changes.
`

	if err := os.WriteFile(filepath.Join(builtinDir, "SKILL.md"), []byte(skillContent), 0644); err != nil {
		t.Fatalf("Failed to write skill: %v", err)
	}

	cfg := LoaderConfig{
		BuiltinDir:   builtinDir,
		CacheEnabled: false,
	}

	loader := NewLoader(cfg)

	// Search by name
	results, err := loader.SearchSkills("git")
	if err != nil {
		t.Fatalf("Failed to search skills: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("Expected 1 result for 'git', got %d", len(results))
	}

	// Search by description
	results, err = loader.SearchSkills("commit")
	if err != nil {
		t.Fatalf("Failed to search skills: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("Expected 1 result for 'commit', got %d", len(results))
	}

	// Search by tag
	results, err = loader.SearchSkills("version")
	if err != nil {
		t.Fatalf("Failed to search skills: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("Expected 1 result for 'version', got %d", len(results))
	}

	// Case insensitive search
	results, err = loader.SearchSkills("GIT")
	if err != nil {
		t.Fatalf("Failed to search skills: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("Expected 1 result for 'GIT', got %d", len(results))
	}
}
