package skills

import (
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
		Format: "short",
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
		Format: "short",
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
		Format: "medium",
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

	opts := SummaryOptions{}

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
		Format: "short",
	}

	summary, err := builder.Build(opts)
	if err != nil {
		t.Fatalf("Failed to build summary: %v", err)
	}

	if !strings.Contains(summary, "No skills available") {
		t.Error("Expected summary to mention no skills available")
	}
}
