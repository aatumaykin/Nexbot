package skills

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aatumaykin/nexbot/internal/config"
	"github.com/aatumaykin/nexbot/internal/workspace"
)

func TestLoader_Load_Empty(t *testing.T) {
	cfg := LoaderConfig{
		BuiltinDir:   "",
		CacheEnabled: false,
	}

	loader := NewLoader(cfg)
	skills, err := loader.Load()

	if err != nil {
		t.Fatalf("Failed to load skills: %v", err)
	}

	if len(skills) != 0 {
		t.Errorf("Expected 0 skills, got %d", len(skills))
	}
}

func TestLoader_Load_BuiltinOnly(t *testing.T) {
	tmpDir := t.TempDir()

	// Create builtin skill
	builtinDir := filepath.Join(tmpDir, "builtin", "skills", "git")
	if err := os.MkdirAll(builtinDir, 0755); err != nil {
		t.Fatalf("Failed to create builtin directory: %v", err)
	}

	skillContent := `---
name: git-commit
description: Commit changes to git
version: 1.0.0
category: git
---

Git commit functionality.
`

	skillPath := filepath.Join(builtinDir, "SKILL.md")
	if err := os.WriteFile(skillPath, []byte(skillContent), 0644); err != nil {
		t.Fatalf("Failed to write skill file: %v", err)
	}

	cfg := LoaderConfig{
		BuiltinDir:   filepath.Join(tmpDir, "builtin", "skills"),
		CacheEnabled: false,
	}

	loader := NewLoader(cfg)
	skills, err := loader.Load()

	if err != nil {
		t.Fatalf("Failed to load skills: %v", err)
	}

	if len(skills) != 1 {
		t.Errorf("Expected 1 skill, got %d", len(skills))
	}

	if _, exists := skills["git-commit"]; !exists {
		t.Error("Expected 'git-commit' skill to be loaded")
	}

	if skills["git-commit"].Metadata.Category != "git" {
		t.Errorf("Expected category 'git', got '%s'", skills["git-commit"].Metadata.Category)
	}
}

func TestLoader_Load_WorkspacePriority(t *testing.T) {
	tmpDir := t.TempDir()

	// Create builtin skill
	builtinDir := filepath.Join(tmpDir, "builtin", "skills")
	if err := os.MkdirAll(builtinDir, 0755); err != nil {
		t.Fatalf("Failed to create builtin directory: %v", err)
	}

	builtinContent := `---
name: test-skill
description: Builtin version
version: 1.0.0
---

Builtin skill content.
`

	builtinPath := filepath.Join(builtinDir, "SKILL.md")
	if err := os.WriteFile(builtinPath, []byte(builtinContent), 0644); err != nil {
		t.Fatalf("Failed to write builtin skill: %v", err)
	}

	// Create workspace skill
	workspaceDir := filepath.Join(tmpDir, "workspace", "skills")
	if err := os.MkdirAll(workspaceDir, 0755); err != nil {
		t.Fatalf("Failed to create workspace directory: %v", err)
	}

	workspaceContent := `---
name: test-skill
description: Workspace version
version: 2.0.0
---

Workspace skill content.
`

	workspacePath := filepath.Join(workspaceDir, "SKILL.md")
	if err := os.WriteFile(workspacePath, []byte(workspaceContent), 0644); err != nil {
		t.Fatalf("Failed to write workspace skill: %v", err)
	}

	cfg := LoaderConfig{
		BuiltinDir:   builtinDir,
		Workspace:    workspace.New(config.WorkspaceConfig{Path: tmpDir + "/workspace"}),
		CacheEnabled: false,
	}

	loader := NewLoader(cfg)
	skills, err := loader.Load()

	if err != nil {
		t.Fatalf("Failed to load skills: %v", err)
	}

	if len(skills) != 1 {
		t.Errorf("Expected 1 skill, got %d", len(skills))
	}

	// Workspace skill should take priority
	if skills["test-skill"].Metadata.Description != "Workspace version" {
		t.Errorf("Expected workspace version to take priority, got '%s'", skills["test-skill"].Metadata.Description)
	}

	if skills["test-skill"].Metadata.Version != "2.0.0" {
		t.Errorf("Expected version '2.0.0', got '%s'", skills["test-skill"].Metadata.Version)
	}
}

func TestLoader_Load_MultipleSkills(t *testing.T) {
	tmpDir := t.TempDir()

	// Create multiple skills
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
description: Commit changes to git
category: git
---

Git commit skill.
`

	dockerContent := `---
name: docker-build
description: Build Docker images
category: docker
---

Docker build skill.
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
	skills, err := loader.Load()

	if err != nil {
		t.Fatalf("Failed to load skills: %v", err)
	}

	if len(skills) != 2 {
		t.Errorf("Expected 2 skills, got %d", len(skills))
	}

	if _, exists := skills["git-commit"]; !exists {
		t.Error("Expected 'git-commit' skill to be loaded")
	}

	if _, exists := skills["docker-build"]; !exists {
		t.Error("Expected 'docker-build' skill to be loaded")
	}
}

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
deprecated: true
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

	if stats.Deprecated != 1 {
		t.Errorf("Expected 1 deprecated skill, got %d", stats.Deprecated)
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
