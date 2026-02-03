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

This is the skill content.

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

	if !strings.Contains(skill.Content, "This is the skill content") {
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

func TestParse_MissingName(t *testing.T) {
	content := `---
description: A skill without a name
version: 1.0.0
---

Content here.
`

	parser := NewParser()
	_, err := parser.Parse(content)

	if err == nil {
		t.Error("Expected error for skill without name")
	}

	if !strings.Contains(err.Error(), "name") {
		t.Errorf("Expected error to mention 'name', got: %v", err)
	}
}

func TestParse_MissingDescription(t *testing.T) {
	content := `---
name: test-skill
version: 1.0.0
---

Content here.
`

	parser := NewParser()
	_, err := parser.Parse(content)

	if err == nil {
		t.Error("Expected error for skill without description")
	}

	if !strings.Contains(err.Error(), "description") {
		t.Errorf("Expected error to mention 'description', got: %v", err)
	}
}

func TestParse_InvalidYAML(t *testing.T) {
	content := `---
name: test-skill
description: A skill
invalid: yaml: content: here:
---

Content here.
`

	parser := NewParser()
	_, err := parser.Parse(content)

	if err == nil {
		t.Error("Expected error for invalid YAML")
	}

	if !strings.Contains(err.Error(), "YAML") {
		t.Errorf("Expected error to mention 'YAML', got: %v", err)
	}
}

func TestParse_NoFrontmatter(t *testing.T) {
	content := `This is just markdown content without frontmatter.
`

	parser := NewParser()
	_, err := parser.Parse(content)

	if err == nil {
		t.Error("Expected error for content without frontmatter")
	}

	if !strings.Contains(err.Error(), "frontmatter") {
		t.Errorf("Expected error to mention 'frontmatter', got: %v", err)
	}
}

func TestParse_UnclosedFrontmatter(t *testing.T) {
	content := `---
name: test-skill
description: A skill with unclosed frontmatter

This is just markdown content without closing frontmatter.
`

	parser := NewParser()
	_, err := parser.Parse(content)

	if err == nil {
		t.Error("Expected error for unclosed frontmatter")
	}

	if !strings.Contains(err.Error(), "closed") {
		t.Errorf("Expected error to mention 'closed', got: %v", err)
	}
}

func TestParse_EmptyFrontmatter(t *testing.T) {
	content := `---
---

Content here.
`

	parser := NewParser()
	_, err := parser.Parse(content)

	if err == nil {
		t.Error("Expected error for empty frontmatter")
	}

	if !strings.Contains(err.Error(), "empty") {
		t.Errorf("Expected error to mention 'empty', got: %v", err)
	}
}

func TestParse_DeprecatedSkill(t *testing.T) {
	content := `---
name: deprecated-skill
description: A deprecated skill
deprecated: true
---

This skill is deprecated.
`

	parser := NewParser()
	skill, err := parser.Parse(content)

	if err != nil {
		t.Fatalf("Failed to parse skill: %v", err)
	}

	if !skill.Metadata.Deprecated {
		t.Error("Expected skill to be marked as deprecated")
	}
}

func TestParse_WithAuthor(t *testing.T) {
	content := `---
name: authored-skill
description: A skill with an author
author: John Doe <john@example.com>
---

Content here.
`

	parser := NewParser()
	skill, err := parser.Parse(content)

	if err != nil {
		t.Fatalf("Failed to parse skill: %v", err)
	}

	if skill.Metadata.Author != "John Doe <john@example.com>" {
		t.Errorf("Expected author 'John Doe <john@example.com>', got '%s'", skill.Metadata.Author)
	}
}

func TestParse_ContentWithFormatting(t *testing.T) {
	content := `---
name: formatted-skill
description: A skill with formatted content
---

# Heading

This is **bold** and *italic* text.

` + "```" + `
Code block
` + "```" + `

> Blockquote
`

	parser := NewParser()
	skill, err := parser.Parse(content)

	if err != nil {
		t.Fatalf("Failed to parse skill: %v", err)
	}

	if !strings.Contains(skill.Content, "# Heading") {
		t.Error("Expected content to contain heading")
	}

	if !strings.Contains(skill.Content, "**bold**") {
		t.Error("Expected content to contain bold text")
	}

	if !strings.Contains(skill.Content, "Code block") {
		t.Error("Expected content to contain code block")
	}
}

func TestValidate_ValidSkill(t *testing.T) {
	skill := &Skill{
		Metadata: SkillMetadata{
			Name:        "test-skill",
			Description: "A valid skill",
		},
	}

	if err := skill.Validate(); err != nil {
		t.Errorf("Expected validation to succeed, got: %v", err)
	}
}

func TestValidate_MissingName(t *testing.T) {
	skill := &Skill{
		Metadata: SkillMetadata{
			Description: "A skill without name",
		},
	}

	if err := skill.Validate(); err == nil {
		t.Error("Expected validation error for missing name")
	}
}

func TestValidate_MissingDescription(t *testing.T) {
	skill := &Skill{
		Metadata: SkillMetadata{
			Name: "test-skill",
		},
	}

	if err := skill.Validate(); err == nil {
		t.Error("Expected validation error for missing description")
	}
}

func TestValidate_ParameterWithoutName(t *testing.T) {
	skill := &Skill{
		Metadata: SkillMetadata{
			Name:        "test-skill",
			Description: "A skill",
			Parameters: []struct {
				Name        string `yaml:"name"`
				Type        string `yaml:"type"`
				Description string `yaml:"description"`
				Required    bool   `yaml:"required"`
				Default     any    `yaml:"default"`
			}{
				{
					Type:        "string",
					Description: "A parameter without name",
				},
			},
		},
	}

	if err := skill.Validate(); err == nil {
		t.Error("Expected validation error for parameter without name")
	}
}

func TestValidate_ParameterWithoutType(t *testing.T) {
	skill := &Skill{
		Metadata: SkillMetadata{
			Name:        "test-skill",
			Description: "A skill",
			Parameters: []struct {
				Name        string `yaml:"name"`
				Type        string `yaml:"type"`
				Description string `yaml:"description"`
				Required    bool   `yaml:"required"`
				Default     any    `yaml:"default"`
			}{
				{
					Name:        "path",
					Description: "A parameter without type",
				},
			},
		},
	}

	if err := skill.Validate(); err == nil {
		t.Error("Expected validation error for parameter without type")
	}
}

func TestToMarkdown_RoundTrip(t *testing.T) {
	originalContent := `---
name: test-skill
description: A test skill
version: 1.0.0
category: test
tags:
  - tag1
  - tag2
parameters:
  - name: path
    type: string
    description: The path
    required: true
examples:
  - name: Example 1
    input: "test input"
    description: An example
---

This is the skill content.
`

	parser := NewParser()
	skill, err := parser.Parse(originalContent)

	if err != nil {
		t.Fatalf("Failed to parse skill: %v", err)
	}

	markdown, err := skill.ToMarkdown()
	if err != nil {
		t.Fatalf("Failed to convert to markdown: %v", err)
	}

	// Re-parse the generated markdown
	skill2, err := parser.Parse(markdown)
	if err != nil {
		t.Fatalf("Failed to parse generated markdown: %v", err)
	}

	// Compare metadata
	if skill.Metadata.Name != skill2.Metadata.Name {
		t.Errorf("Name mismatch: '%s' != '%s'", skill.Metadata.Name, skill2.Metadata.Name)
	}

	if skill.Metadata.Description != skill2.Metadata.Description {
		t.Errorf("Description mismatch: '%s' != '%s'", skill.Metadata.Description, skill2.Metadata.Description)
	}

	if len(skill.Metadata.Parameters) != len(skill2.Metadata.Parameters) {
		t.Errorf("Parameter count mismatch: %d != %d", len(skill.Metadata.Parameters), len(skill2.Metadata.Parameters))
	}
}

func TestString(t *testing.T) {
	skill := &Skill{
		Metadata: SkillMetadata{
			Name:        "test-skill",
			Description: "A test skill",
			Version:     "1.0.0",
		},
	}

	str := skill.String()

	if !strings.Contains(str, "test-skill") {
		t.Errorf("Expected string representation to contain skill name")
	}

	if !strings.Contains(str, "1.0.0") {
		t.Errorf("Expected string representation to contain version")
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

	// Validate metadata
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

	// Validate parameters
	if len(skill.Metadata.Parameters) != 2 {
		t.Fatalf("Expected 2 parameters, got %d", len(skill.Metadata.Parameters))
	}

	if skill.Metadata.Parameters[0].Name != "message" {
		t.Errorf("Expected first parameter name 'message', got '%s'", skill.Metadata.Parameters[0].Name)
	}

	// Validate examples
	if len(skill.Metadata.Examples) != 2 {
		t.Fatalf("Expected 2 examples, got %d", len(skill.Metadata.Examples))
	}

	// Validate content
	if !strings.Contains(skill.Content, "This skill helps you create git commits") {
		t.Error("Expected content to contain introduction")
	}

	if !strings.Contains(skill.Content, "## Usage") {
		t.Error("Expected content to contain Usage section")
	}

	// Validate the skill
	if err := skill.Validate(); err != nil {
		t.Errorf("Expected validation to succeed, got: %v", err)
	}
}
