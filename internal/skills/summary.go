package skills

import (
	"fmt"
	"slices"
	"strings"
)

// SummaryBuilder generates formatted summaries of skills for inclusion in system prompts.
// It creates concise, well-formatted descriptions of all available skills.
type SummaryBuilder struct {
	loader *Loader
}

// NewSummaryBuilder creates a new SummaryBuilder instance.
func NewSummaryBuilder(loader *Loader) *SummaryBuilder {
	return &SummaryBuilder{
		loader: loader,
	}
}

// SummaryOptions represents options for generating skill summaries.
type SummaryOptions struct {
	Categories []string // Specific categories to include (empty = all)
	Format     string   // Format: "short", "medium", "long"
	MaxSkills  int      // Maximum number of skills to include (0 = all)
}

// DefaultSummaryOptions returns default summary options.
func DefaultSummaryOptions() SummaryOptions {
	return SummaryOptions{
		Categories: nil,
		Format:     "medium",
		MaxSkills:  0,
	}
}

// Build generates a summary of all loaded skills.
// The summary is formatted for inclusion in system prompts.
func (b *SummaryBuilder) Build(opts SummaryOptions) (string, error) {
	skills, err := b.loader.Load()
	if err != nil {
		return "", fmt.Errorf("failed to load skills: %w", err)
	}

	// Filter skills
	filtered := b.filterSkills(skills, opts)

	// Limit number of skills
	if opts.MaxSkills > 0 && len(filtered) > opts.MaxSkills {
		// Sort skills by name for deterministic ordering
		slices.SortFunc(filtered, func(a, b *Skill) int {
			return strings.Compare(a.Metadata.Name, b.Metadata.Name)
		})
		filtered = filtered[:opts.MaxSkills]
	}

	// Format summary based on format option
	switch opts.Format {
	case "short":
		return b.buildShortSummary(filtered), nil
	case "medium":
		return b.buildMediumSummary(filtered), nil
	case "long":
		return b.buildLongSummary(filtered), nil
	default:
		return b.buildMediumSummary(filtered), nil
	}
}

// BuildDefault generates a summary with default options.
func (b *SummaryBuilder) BuildDefault() (string, error) {
	return b.Build(DefaultSummaryOptions())
}

// filterSkills filters skills based on options.
func (b *SummaryBuilder) filterSkills(skills map[string]*Skill, opts SummaryOptions) []*Skill {
	var filtered []*Skill

	for _, skill := range skills {
		// Filter by category if specified
		if len(opts.Categories) > 0 {
			found := slices.Contains(opts.Categories, skill.Metadata.Category)
			if !found {
				continue
			}
		}

		filtered = append(filtered, skill)
	}

	return filtered
}

// buildShortSummary builds a short summary (name + description only).
func (b *SummaryBuilder) buildShortSummary(skills []*Skill) string {
	var builder strings.Builder

	builder.WriteString("## Available Skills\n\n")

	if len(skills) == 0 {
		builder.WriteString("No skills available.\n")
		return builder.String()
	}

	for _, skill := range skills {
		builder.WriteString(fmt.Sprintf("- **%s**: %s\n",
			skill.Metadata.Name, skill.Metadata.Description))
	}

	return builder.String()
}

// buildMediumSummary builds a medium summary (name, description, parameters, examples).
func (b *SummaryBuilder) buildMediumSummary(skills []*Skill) string {
	var builder strings.Builder

	builder.WriteString("## Available Skills\n\n")

	if len(skills) == 0 {
		builder.WriteString("No skills available.\n")
		return builder.String()
	}

	// Group by category
	categories := b.groupByCategory(skills)

	for _, category := range b.sortedCategories(categories) {
		builder.WriteString(fmt.Sprintf("### %s\n\n", category))

		for _, skill := range categories[category] {
			builder.WriteString(fmt.Sprintf("#### %s\n\n", skill.Metadata.Name))
			builder.WriteString(fmt.Sprintf("%s\n\n", skill.Metadata.Description))

			// Add version if available
			if skill.Metadata.Version != "" {
				builder.WriteString(fmt.Sprintf("**Version**: %s\n\n", skill.Metadata.Version))
			}

			// Add author if available
			if skill.Metadata.Author != "" {
				builder.WriteString(fmt.Sprintf("**Author**: %s\n\n", skill.Metadata.Author))
			}

			// Add parameters if any
			if len(skill.Metadata.Parameters) > 0 {
				builder.WriteString("**Parameters**:\n")
				for _, param := range skill.Metadata.Parameters {
					required := ""
					if param.Required {
						required = " (required)"
					}
					builder.WriteString(fmt.Sprintf("- `%s` (%s)%s: %s\n",
						param.Name, param.Type, required, param.Description))
				}
				builder.WriteString("\n")
			}

			// Add examples if any
			if len(skill.Metadata.Examples) > 0 {
				builder.WriteString("**Examples**:\n")
				for _, example := range skill.Metadata.Examples {
					builder.WriteString(fmt.Sprintf("- **%s**\n", example.Name))
					if example.Description != "" {
						builder.WriteString(fmt.Sprintf("  %s\n", example.Description))
					}
					builder.WriteString(fmt.Sprintf("  Input: `%s`\n", example.Input))
				}
				builder.WriteString("\n")
			}

			builder.WriteString("---\n\n")
		}
	}

	return builder.String()
}

// buildLongSummary builds a long summary (all details including content preview).
func (b *SummaryBuilder) buildLongSummary(skills []*Skill) string {
	var builder strings.Builder

	builder.WriteString("## Available Skills\n\n")

	if len(skills) == 0 {
		builder.WriteString("No skills available.\n")
		return builder.String()
	}

	// Group by category
	categories := b.groupByCategory(skills)

	for _, category := range b.sortedCategories(categories) {
		builder.WriteString(fmt.Sprintf("### %s\n\n", category))

		for _, skill := range categories[category] {
			builder.WriteString(fmt.Sprintf("#### %s\n\n", skill.Metadata.Name))
			builder.WriteString(fmt.Sprintf("%s\n\n", skill.Metadata.Description))

			// Add metadata
			if skill.Metadata.Version != "" {
				builder.WriteString(fmt.Sprintf("**Version**: %s\n", skill.Metadata.Version))
			}
			if skill.Metadata.Category != "" {
				builder.WriteString(fmt.Sprintf("**Category**: %s\n", skill.Metadata.Category))
			}
			if skill.Metadata.Author != "" {
				builder.WriteString(fmt.Sprintf("**Author**: %s\n", skill.Metadata.Author))
			}
			if len(skill.Metadata.Tags) > 0 {
				builder.WriteString(fmt.Sprintf("**Tags**: %s\n", strings.Join(skill.Metadata.Tags, ", ")))
			}
			builder.WriteString("\n")

			// Add parameters
			if len(skill.Metadata.Parameters) > 0 {
				builder.WriteString("**Parameters**:\n")
				for _, param := range skill.Metadata.Parameters {
					required := ""
					if param.Required {
						required = " (required)"
					}
					builder.WriteString(fmt.Sprintf("- `%s` (%s)%s: %s\n",
						param.Name, param.Type, required, param.Description))
					if param.Default != nil {
						builder.WriteString(fmt.Sprintf("  Default: `%v`\n", param.Default))
					}
				}
				builder.WriteString("\n")
			}

			// Add examples
			if len(skill.Metadata.Examples) > 0 {
				builder.WriteString("**Examples**:\n")
				for _, example := range skill.Metadata.Examples {
					builder.WriteString(fmt.Sprintf("- **%s**\n", example.Name))
					if example.Description != "" {
						builder.WriteString(fmt.Sprintf("  %s\n", example.Description))
					}
					builder.WriteString(fmt.Sprintf("  Input: `%s`\n", example.Input))
				}
				builder.WriteString("\n")
			}

			// Add content preview (first 3 paragraphs)
			if skill.Content != "" {
				builder.WriteString("**Content Preview**:\n")
				preview := b.previewContent(skill.Content, 3)
				builder.WriteString(fmt.Sprintf("```\n%s\n```\n", preview))
				builder.WriteString("\n")
			}

			builder.WriteString("---\n\n")
		}
	}

	return builder.String()
}

// groupByCategory groups skills by category.
func (b *SummaryBuilder) groupByCategory(skills []*Skill) map[string][]*Skill {
	categories := make(map[string][]*Skill)

	for _, skill := range skills {
		category := skill.Metadata.Category
		if category == "" {
			category = "Other"
		}

		categories[category] = append(categories[category], skill)
	}

	return categories
}

// sortedCategories returns a sorted list of category names.
func (b *SummaryBuilder) sortedCategories(categories map[string][]*Skill) []string {
	names := make([]string, 0, len(categories))
	for name := range categories {
		names = append(names, name)
	}

	// Use efficient O(n log n) sort from Go 1.21+
	slices.Sort(names)

	return names
}

// previewContent generates a preview of skill content.
func (b *SummaryBuilder) previewContent(content string, maxParagraphs int) string {
	paragraphs := strings.Split(content, "\n\n")

	var preview []string
	for i, para := range paragraphs {
		if i >= maxParagraphs {
			break
		}

		trimmed := strings.TrimSpace(para)
		if trimmed != "" {
			preview = append(preview, trimmed)
		}
	}

	previewText := strings.Join(preview, "\n\n")

	// Truncate if too long
	maxLength := 500
	if len(previewText) > maxLength {
		previewText = previewText[:maxLength] + "..."
	}

	return previewText
}

// BuildForPrompt generates a summary specifically formatted for inclusion in system prompts.
// This uses a compact format suitable for LLM context.
func (b *SummaryBuilder) BuildForPrompt(opts SummaryOptions) (string, error) {
	skills, err := b.loader.Load()
	if err != nil {
		return "", fmt.Errorf("failed to load skills: %w", err)
	}

	// Filter skills
	filtered := b.filterSkills(skills, opts)

	// Limit number of skills
	if opts.MaxSkills > 0 && len(filtered) > opts.MaxSkills {
		// Sort skills by name for deterministic ordering
		slices.SortFunc(filtered, func(a, b *Skill) int {
			return strings.Compare(a.Metadata.Name, b.Metadata.Name)
		})
		filtered = filtered[:opts.MaxSkills]
	}

	var builder strings.Builder

	builder.WriteString("You have access to the following skills:\n\n")

	if len(filtered) == 0 {
		builder.WriteString("No skills available.\n")
		return builder.String(), nil
	}

	for _, skill := range filtered {
		builder.WriteString(fmt.Sprintf("- **%s**: %s", skill.Metadata.Name, skill.Metadata.Description))

		// Add parameters if any
		if len(skill.Metadata.Parameters) > 0 {
			params := make([]string, 0, len(skill.Metadata.Parameters))
			for _, param := range skill.Metadata.Parameters {
				if param.Required {
					params = append(params, param.Name+"*")
				} else {
					params = append(params, param.Name)
				}
			}
			builder.WriteString(fmt.Sprintf(" (params: %s)", strings.Join(params, ", ")))
		}

		// Add example if available
		if len(skill.Metadata.Examples) > 0 {
			builder.WriteString(fmt.Sprintf(" [example: %s]", skill.Metadata.Examples[0].Input))
		}

		builder.WriteString("\n")
	}

	return builder.String(), nil
}
