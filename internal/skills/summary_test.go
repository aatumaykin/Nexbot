package skills

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSummaryBuilder_Build_Short(t *testing.T) {
	tmpDir := t.TempDir()

	builtinDir := filepath.Join(tmpDir, "builtin", "skills")
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
		Format:            "short",
		IncludeDeprecated: false,
	}

	summary, err := builder.Build(opts)
	if err != nil {
		t.Fatalf("Failed to build summary: %v", err)
	}

	if !strings.Contains(summary, "git-commit") {
		t.Error("Expected summary to contain skill name")
	}

	if !strings.Contains(summary, "Commit changes to git") {
		t.Error("Expected summary to contain skill description")
	}
}

func TestSummaryBuilder_Build_Medium(t *testing.T) {
	tmpDir := t.TempDir()

	builtinDir := filepath.Join(tmpDir, "builtin", "skills")
	if err := os.MkdirAll(builtinDir, 0755); err != nil {
		t.Fatalf("Failed to create builtin directory: %v", err)
	}

	skillContent := `---
name: git-commit
description: Commit changes to git
version: 1.0.0
category: git
author: Test Author
parameters:
  - name: message
    type: string
    description: Commit message
    required: true
examples:
  - name: Basic commit
    input: "test message"
    description: Basic commit example
---

Git commit functionality.
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
		Format:            "medium",
		IncludeDeprecated: false,
	}

	summary, err := builder.Build(opts)
	if err != nil {
		t.Fatalf("Failed to build summary: %v", err)
	}

	// Check skill details
	if !strings.Contains(summary, "git-commit") {
		t.Error("Expected summary to contain skill name")
	}

	if !strings.Contains(summary, "1.0.0") {
		t.Error("Expected summary to contain version")
	}

	if !strings.Contains(summary, "Test Author") {
		t.Error("Expected summary to contain author")
	}

	if !strings.Contains(summary, "Parameters") {
		t.Error("Expected summary to contain parameters section")
	}

	if !strings.Contains(summary, "Examples") {
		t.Error("Expected summary to contain examples section")
	}
}

func TestSummaryBuilder_Build_Long(t *testing.T) {
	tmpDir := t.TempDir()

	builtinDir := filepath.Join(tmpDir, "builtin", "skills")
	if err := os.MkdirAll(builtinDir, 0755); err != nil {
		t.Fatalf("Failed to create builtin directory: %v", err)
	}

	skillContent := `---
name: git-commit
description: Commit changes to git
version: 1.0.0
category: git
author: Test Author
tags:
  - git
  - version-control
parameters:
  - name: message
    type: string
    description: Commit message
    required: true
    default: "Initial commit"
examples:
  - name: Basic commit
    input: "test message"
    description: Basic commit example
---

Git commit functionality.

This skill helps you create commits.

With proper formatting.
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
		Format:            "long",
		IncludeDeprecated: false,
	}

	summary, err := builder.Build(opts)
	if err != nil {
		t.Fatalf("Failed to build summary: %v", err)
	}

	// Check all sections
	if !strings.Contains(summary, "git-commit") {
		t.Error("Expected summary to contain skill name")
	}

	if !strings.Contains(summary, "1.0.0") {
		t.Error("Expected summary to contain version")
	}

	if !strings.Contains(summary, "git") {
		t.Error("Expected summary to contain category")
	}

	if !strings.Contains(summary, "version-control") {
		t.Error("Expected summary to contain tags")
	}

	if !strings.Contains(summary, "Content Preview") {
		t.Error("Expected summary to contain content preview")
	}

	if !strings.Contains(summary, "Initial commit") {
		t.Error("Expected summary to contain default parameter value")
	}
}

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
		Format:            "short",
		Categories:        []string{"git"},
		IncludeDeprecated: false,
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

func TestSummaryBuilder_ExcludeDeprecated(t *testing.T) {
	tmpDir := t.TempDir()

	builtinDir := filepath.Join(tmpDir, "builtin", "skills")
	if err := os.MkdirAll(builtinDir, 0755); err != nil {
		t.Fatalf("Failed to create builtin directory: %v", err)
	}

	activeContent := `---
name: active-skill
description: An active skill
---
`

	deprecatedContent := `---
name: deprecated-skill
description: A deprecated skill
deprecated: true
---
`

	if err := os.WriteFile(filepath.Join(builtinDir, "active.md"), []byte(activeContent), 0644); err != nil {
		t.Fatalf("Failed to write active skill: %v", err)
	}
	if err := os.WriteFile(filepath.Join(builtinDir, "deprecated.md"), []byte(deprecatedContent), 0644); err != nil {
		t.Fatalf("Failed to write deprecated skill: %v", err)
	}

	cfg := LoaderConfig{
		BuiltinDir:   builtinDir,
		CacheEnabled: false,
	}

	loader := NewLoader(cfg)
	builder := NewSummaryBuilder(loader)

	opts := SummaryOptions{
		Format:            "short",
		IncludeDeprecated: false,
	}

	summary, err := builder.Build(opts)
	if err != nil {
		t.Fatalf("Failed to build summary: %v", err)
	}

	// Should not contain deprecated skills
	if strings.Contains(summary, "deprecated-skill") {
		t.Error("Expected summary to not contain deprecated skill")
	}
}

func TestSummaryBuilder_IncludeDeprecated(t *testing.T) {
	tmpDir := t.TempDir()

	builtinDir := filepath.Join(tmpDir, "builtin", "skills")
	if err := os.MkdirAll(builtinDir, 0755); err != nil {
		t.Fatalf("Failed to create builtin directory: %v", err)
	}

	deprecatedContent := `---
name: deprecated-skill
description: A deprecated skill
deprecated: true
---
`

	if err := os.WriteFile(filepath.Join(builtinDir, "SKILL.md"), []byte(deprecatedContent), 0644); err != nil {
		t.Fatalf("Failed to write deprecated skill: %v", err)
	}

	cfg := LoaderConfig{
		BuiltinDir:   builtinDir,
		CacheEnabled: false,
	}

	loader := NewLoader(cfg)
	builder := NewSummaryBuilder(loader)

	opts := SummaryOptions{
		Format:            "short",
		IncludeDeprecated: true,
	}

	summary, err := builder.Build(opts)
	if err != nil {
		t.Fatalf("Failed to build summary: %v", err)
	}

	// Should contain deprecated skills
	if !strings.Contains(summary, "deprecated-skill") {
		t.Error("Expected summary to contain deprecated skill when IncludeDeprecated is true")
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
		Format:            "short",
		IncludeDeprecated: false,
		MaxSkills:         3,
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

func TestSummaryBuilder_BuildDefault(t *testing.T) {
	tmpDir := t.TempDir()

	builtinDir := filepath.Join(tmpDir, "builtin", "skills")
	if err := os.MkdirAll(builtinDir, 0755); err != nil {
		t.Fatalf("Failed to create builtin directory: %v", err)
	}

	skillContent := `---
name: test-skill
description: A test skill
---
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

	summary, err := builder.BuildDefault()
	if err != nil {
		t.Fatalf("Failed to build default summary: %v", err)
	}

	if !strings.Contains(summary, "test-skill") {
		t.Error("Expected default summary to contain skill")
	}
}

func TestSummaryBuilder_BuildForPrompt(t *testing.T) {
	tmpDir := t.TempDir()

	builtinDir := filepath.Join(tmpDir, "builtin", "skills")
	if err := os.MkdirAll(builtinDir, 0755); err != nil {
		t.Fatalf("Failed to create builtin directory: %v", err)
	}

	skillContent := `---
name: git-commit
description: Commit changes to git
parameters:
  - name: message
    type: string
    description: Commit message
    required: true
  - name: amend
    type: boolean
    description: Amend last commit
    required: false
examples:
  - name: Basic commit
    input: "test message"
    description: Basic commit example
---

Git commit functionality.
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
		IncludeDeprecated: false,
	}

	summary, err := builder.BuildForPrompt(opts)
	if err != nil {
		t.Fatalf("Failed to build summary for prompt: %v", err)
	}

	// Check prompt format
	if !strings.Contains(summary, "You have access to the following skills") {
		t.Error("Expected prompt summary to start with introduction")
	}

	if !strings.Contains(summary, "git-commit") {
		t.Error("Expected prompt summary to contain skill name")
	}

	if !strings.Contains(summary, "params:") {
		t.Error("Expected prompt summary to contain parameters")
	}

	if !strings.Contains(summary, "message*") {
		t.Error("Expected prompt summary to mark required parameters with *")
	}

	if !strings.Contains(summary, "example:") {
		t.Error("Expected prompt summary to contain example")
	}
}

func TestSummaryBuilder_NoSkills(t *testing.T) {
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
	builder := NewSummaryBuilder(loader)

	opts := SummaryOptions{
		Format:            "short",
		IncludeDeprecated: false,
	}

	summary, err := builder.Build(opts)
	if err != nil {
		t.Fatalf("Failed to build summary: %v", err)
	}

	if !strings.Contains(summary, "No skills available") {
		t.Error("Expected summary to mention no skills available")
	}
}

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
		Format:            "medium",
		IncludeDeprecated: false,
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
		Format:            "long",
		IncludeDeprecated: false,
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

func BenchmarkSortedCategories(b *testing.B) {
	categories := make(map[string][]*Skill)

	// Create 100+ categories for meaningful benchmark
	for i := 0; i < 100; i++ {
		category := fmt.Sprintf("cat_%d", rand.Int())
		categories[category] = []*Skill{}
	}

	builder := &SummaryBuilder{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		builder.sortedCategories(categories)
	}
}

func TestSortedCategories(t *testing.T) {
	categories := make(map[string][]*Skill)
	categories["zebra"] = []*Skill{}
	categories["apple"] = []*Skill{}
	categories["banana"] = []*Skill{}
	categories["orange"] = []*Skill{}

	builder := &SummaryBuilder{}
	result := builder.sortedCategories(categories)

	// Verify result is alphabetically sorted
	expected := []string{"apple", "banana", "orange", "zebra"}
	if len(result) != len(expected) {
		t.Fatalf("Expected %d categories, got %d", len(expected), len(result))
	}

	for i, cat := range expected {
		if result[i] != cat {
			t.Errorf("Expected category at index %d to be %s, got %s", i, cat, result[i])
		}
	}
}
