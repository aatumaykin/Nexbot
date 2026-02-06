package skills

import (
	"strings"
	"testing"
)

func TestParse_SimpleSkill(t *testing.T) {
	content := `---
name: test-skill
description: A simple test skill
version: 1.0.0
category: test
---

This is skill content.

It can have multiple paragraphs.

- List item 1
- List item 2
`

	parser := NewParser()
	skill, err := parser.Parse(content)

	if err != nil {
		t.Fatalf("Failed to parse skill: %v", err)
	}

	if skill.Metadata.Name != "test-skill" {
		t.Errorf("Expected name 'test-skill', got '%s'", skill.Metadata.Name)
	}

	if skill.Metadata.Description != "A simple test skill" {
		t.Errorf("Expected description 'A simple test skill', got '%s'", skill.Metadata.Description)
	}

	if skill.Metadata.Version != "1.0.0" {
		t.Errorf("Expected version '1.0.0', got '%s'", skill.Metadata.Version)
	}

	if skill.Metadata.Category != "test" {
		t.Errorf("Expected category 'test', got '%s'", skill.Metadata.Category)
	}

	if !strings.Contains(skill.Content, "This is skill content") {
		t.Errorf("Expected content to contain skill description")
	}
}

func TestParse_WithParameters(t *testing.T) {
	content := `---
name: param-skill
description: A skill with parameters
parameters:
  - name: path
    type: string
    description: The file path
    required: true
  - name: recursive
    type: boolean
    description: Whether to search recursively
    required: false
    default: false
---

This skill accepts parameters.
`

	parser := NewParser()
	skill, err := parser.Parse(content)

	if err != nil {
		t.Fatalf("Failed to parse skill: %v", err)
	}

	if len(skill.Metadata.Parameters) != 2 {
		t.Fatalf("Expected 2 parameters, got %d", len(skill.Metadata.Parameters))
	}

	if skill.Metadata.Parameters[0].Name != "path" {
		t.Errorf("Expected first parameter name 'path', got '%s'", skill.Metadata.Parameters[0].Name)
	}

	if skill.Metadata.Parameters[0].Type != "string" {
		t.Errorf("Expected first parameter type 'string', got '%s'", skill.Metadata.Parameters[0].Type)
	}

	if !skill.Metadata.Parameters[0].Required {
		t.Errorf("Expected first parameter to be required")
	}

	if skill.Metadata.Parameters[1].Default != false {
		t.Errorf("Expected second parameter default false, got %v", skill.Metadata.Parameters[1].Default)
	}
}

func TestParse_WithExamples(t *testing.T) {
	content := `---
name: example-skill
description: A skill with examples
examples:
  - name: Basic usage
    input: "example input"
    description: This shows basic usage
  - name: Advanced usage
    input: "advanced input"
    description: This shows advanced features
---

Content here.
`

	parser := NewParser()
	skill, err := parser.Parse(content)

	if err != nil {
		t.Fatalf("Failed to parse skill: %v", err)
	}

	if len(skill.Metadata.Examples) != 2 {
		t.Fatalf("Expected 2 examples, got %d", len(skill.Metadata.Examples))
	}

	if skill.Metadata.Examples[0].Name != "Basic usage" {
		t.Errorf("Expected first example name 'Basic usage', got '%s'", skill.Metadata.Examples[0].Name)
	}

	if skill.Metadata.Examples[1].Description != "This shows advanced features" {
		t.Errorf("Expected second example description 'This shows advanced features', got '%s'", skill.Metadata.Examples[1].Description)
	}
}

func TestParse_WithTags(t *testing.T) {
	content := `---
name: tag-skill
description: A skill with tags
tags:
  - git
  - code-review
  - automation
---

Content here.
`

	parser := NewParser()
	skill, err := parser.Parse(content)

	if err != nil {
		t.Fatalf("Failed to parse skill: %v", err)
	}

	if len(skill.Metadata.Tags) != 3 {
		t.Fatalf("Expected 3 tags, got %d", len(skill.Metadata.Tags))
	}

	if skill.Metadata.Tags[0] != "git" {
		t.Errorf("Expected first tag 'git', got '%s'", skill.Metadata.Tags[0])
	}

	if skill.Metadata.Tags[2] != "automation" {
		t.Errorf("Expected third tag 'automation', got '%s'", skill.Metadata.Tags[2])
	}
}

func TestParse_CRLF(t *testing.T) {
	content := "---\r\nname: test-skill\r\ndescription: A skill\r\n---\r\n\r\nContent here.\r\n"

	parser := NewParser()
	skill, err := parser.Parse(content)

	if err != nil {
		t.Fatalf("Failed to parse skill with CRLF: %v", err)
	}

	if skill.Metadata.Name != "test-skill" {
		t.Errorf("Expected name 'test-skill', got '%s'", skill.Metadata.Name)
	}

	if skill.Content != "Content here." {
		t.Errorf("Expected content 'Content here.', got '%s'", skill.Content)
	}
}

func TestParse_CompleteSkill(t *testing.T) {
	content := `---
name: git-commit
description: Commit changes to git with a descriptive message
version: 1.0.0
category: git
tags:
  - git
  - version-control
author: Nexbot Team
parameters:
  - name: message
    type: string
    description: The commit message
    required: true
  - name: stage_all
    type: boolean
    description: Whether to stage all changes
    required: false
    default: false
examples:
  - name: Basic commit
    input: '{"message": "Fix bug in user authentication"}'
    description: Commits all staged changes with the given message
  - name: Commit all changes
    input: '{"message": "Add new feature", "stage_all": true}'
    description: Stages all changes and commits them
---

This skill helps you create git commits with properly formatted messages.

## Usage

Provide a clear, descriptive commit message following conventional commit format:
- feat: for new features
- fix: for bug fixes
- docs: for documentation changes
- refactor: for code refactoring
- test: for adding tests
- chore: for maintenance tasks

## Notes

- Always review your changes before committing
- Use meaningful commit messages
- Keep commits focused and atomic
`

	parser := NewParser()
	skill, err := parser.Parse(content)

	if err != nil {
		t.Fatalf("Failed to parse skill: %v", err)
	}

	if skill.Metadata.Name != "git-commit" {
		t.Errorf("Expected name 'git-commit', got '%s'", skill.Metadata.Name)
	}

	if skill.Metadata.Version != "1.0.0" {
		t.Errorf("Expected version '1.0.0', got '%s'", skill.Metadata.Version)
	}

	if skill.Metadata.Category != "git" {
		t.Errorf("Expected category 'git', got '%s'", skill.Metadata.Category)
	}

	if skill.Metadata.Author != "Nexbot Team" {
		t.Errorf("Expected author 'Nexbot Team', got '%s'", skill.Metadata.Author)
	}

	if len(skill.Metadata.Parameters) != 2 {
		t.Fatalf("Expected 2 parameters, got %d", len(skill.Metadata.Parameters))
	}

	if skill.Metadata.Parameters[0].Name != "message" {
		t.Errorf("Expected first parameter name 'message', got '%s'", skill.Metadata.Parameters[0].Name)
	}

	if len(skill.Metadata.Examples) != 2 {
		t.Fatalf("Expected 2 examples, got %d", len(skill.Metadata.Examples))
	}

	if !strings.Contains(skill.Content, "This skill helps you create git commits") {
		t.Error("Expected content to contain introduction")
	}

	if !strings.Contains(skill.Content, "## Usage") {
		t.Error("Expected content to contain Usage section")
	}

	if err := skill.Validate(); err != nil {
		t.Errorf("Expected validation to succeed, got: %v", err)
	}
}
