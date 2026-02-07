package skills

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// SkillMetadata represents the metadata extracted from a SKILL.md file.
// It contains the YAML frontmatter information about a skill.
type SkillMetadata struct {
	Name        string   `yaml:"name"`               // Unique name/identifier of the skill
	Description string   `yaml:"description"`        // Human-readable description of what the skill does
	Version     string   `yaml:"version,omitempty"`  // Version of the skill (e.g., "1.0.0")
	Category    string   `yaml:"category,omitempty"` // Category/group for the skill (e.g., "git", "docker")
	Tags        []string `yaml:"tags,omitempty"`     // Tags for the skill
	Parameters  []struct {
		Name        string `yaml:"name"`        // Parameter name
		Type        string `yaml:"type"`        // Parameter type (string, number, boolean, array, object)
		Description string `yaml:"description"` // Parameter description
		Required    bool   `yaml:"required"`    // Whether the parameter is required
		Default     any    `yaml:"default"`     // Default value (optional)
	} `yaml:"parameters,omitempty"` // Input parameters for the skill
	Examples []struct {
		Name        string `yaml:"name"`        // Example name/description
		Input       string `yaml:"input"`       // Example input
		Description string `yaml:"description"` // What the example demonstrates
	} `yaml:"examples,omitempty"` // Usage examples
	Author string `yaml:"author,omitempty"` // Author of the skill
}

// Skill represents a fully parsed skill with metadata and content.
type Skill struct {
	Metadata SkillMetadata `yaml:"metadata"`  // Parsed metadata from YAML frontmatter
	Content  string        `yaml:"content"`   // Markdown body content
	FilePath string        `yaml:"file_path"` // Path to the skill file
}

// Parser handles parsing of SKILL.md files.
type Parser struct{}

// NewParser creates a new Parser instance.
func NewParser() *Parser {
	return &Parser{}
}

// Parse parses a SKILL.md file content and extracts metadata and content.
// The file should have the following format:
//
//	---
//	name: skill-name
//	description: Skill description
//	version: 1.0.0
//	...
//	---
//
//	Markdown content here...
//
// Returns an error if the file format is invalid.
func (p *Parser) Parse(content string) (*Skill, error) {
	// Split content into frontmatter and body
	frontmatter, body, err := splitFrontmatter(content)
	if err != nil {
		return nil, fmt.Errorf("failed to split frontmatter: %w", err)
	}

	// Parse YAML frontmatter
	var metadata SkillMetadata
	if err := yaml.Unmarshal([]byte(frontmatter), &metadata); err != nil {
		return nil, fmt.Errorf("failed to parse YAML frontmatter: %w", err)
	}

	// Validate required fields
	if metadata.Name == "" {
		return nil, fmt.Errorf("skill metadata must have a 'name' field")
	}
	if metadata.Description == "" {
		return nil, fmt.Errorf("skill metadata must have a 'description' field")
	}

	// Create skill
	skill := &Skill{
		Metadata: metadata,
		Content:  strings.TrimSpace(body),
	}

	return skill, nil
}

// ParseFile parses a SKILL.md file given its file path.
// This is a convenience method that reads the file and calls Parse.
func (p *Parser) ParseFile(filePath string) (*Skill, error) {
	// This will be implemented when file reading functionality is added
	// For now, return an error indicating this is not yet implemented
	return nil, fmt.Errorf("ParseFile not yet implemented - use Parse with content instead")
}

// splitFrontmatter splits content into YAML frontmatter and markdown body.
// The frontmatter is enclosed in "---" delimiters.
func splitFrontmatter(content string) (frontmatter string, body string, err error) {
	// Normalize line endings
	content = strings.ReplaceAll(content, "\r\n", "\n")

	// Split into lines
	lines := strings.Split(content, "\n")

	// Check if content starts with "---"
	if len(lines) < 2 || !strings.HasPrefix(strings.TrimSpace(lines[0]), "---") {
		return "", "", fmt.Errorf("content must start with YAML frontmatter delimited by '---'")
	}

	// Find the closing "---"
	var frontmatterLines []string
	var bodyLines []string
	inFrontmatter := false
	foundEnd := false

	for i, line := range lines {
		if i == 0 {
			// First line should be "---"
			inFrontmatter = true
			continue
		}

		trimmed := strings.TrimSpace(line)

		// Check for end delimiter
		if inFrontmatter && trimmed == "---" {
			inFrontmatter = false
			foundEnd = true
			continue
		}

		if inFrontmatter {
			frontmatterLines = append(frontmatterLines, line)
		} else if foundEnd || i == 1 {
			// We've found the end delimiter or there's no frontmatter
			bodyLines = append(bodyLines, line)
		}
	}

	if !foundEnd {
		return "", "", fmt.Errorf("YAML frontmatter must be closed with '---'")
	}

	if len(frontmatterLines) == 0 {
		return "", "", fmt.Errorf("YAML frontmatter is empty")
	}

	return strings.Join(frontmatterLines, "\n"), strings.Join(bodyLines, "\n"), nil
}

// Validate validates a parsed skill for correctness.
// It checks that all required fields are present and valid.
func (s *Skill) Validate() error {
	// Validate metadata
	if s.Metadata.Name == "" {
		return fmt.Errorf("skill name is required")
	}

	if s.Metadata.Description == "" {
		return fmt.Errorf("skill description is required")
	}

	// Validate parameters
	for i, param := range s.Metadata.Parameters {
		if param.Name == "" {
			return fmt.Errorf("parameter %d: name is required", i)
		}
		if param.Type == "" {
			return fmt.Errorf("parameter %d (%s): type is required", i, param.Name)
		}
	}

	return nil
}

// String returns a string representation of the skill.
func (s *Skill) String() string {
	return fmt.Sprintf("Skill[%s v%s - %s]", s.Metadata.Name, s.Metadata.Version, s.Metadata.Description)
}

// ToMarkdown converts the skill back to a SKILL.md format.
// This is useful for testing or regenerating skill files.
func (s *Skill) ToMarkdown() (string, error) {
	// Marshal metadata to YAML
	yamlBytes, err := yaml.Marshal(s.Metadata)
	if err != nil {
		return "", fmt.Errorf("failed to marshal metadata: %w", err)
	}

	// Build output
	var output strings.Builder

	// Add frontmatter delimiters
	output.WriteString("---\n")
	output.WriteString(string(yamlBytes))
	output.WriteString("---\n")

	// Add content if present
	if s.Content != "" {
		output.WriteString("\n")
		output.WriteString(s.Content)
	}

	return output.String(), nil
}
