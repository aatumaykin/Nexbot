package skills

import (
	"strings"
)

// SkillSearcher provides search functionality for skills.
type SkillSearcher interface {
	// SearchSkills searches for skills by name or description.
	// The search is case-insensitive.
	SearchSkills(query string) ([]*Skill, error)

	// GetSkillsByCategory returns all skills in a given category.
	GetSkillsByCategory(category string) ([]*Skill, error)

	// GetSkillsByTags returns all skills that have any of the given tags.
	GetSkillsByTags(tags []string) ([]*Skill, error)
}

// skillSearcher implements SkillSearcher interface.
type skillSearcher struct {
	loader *Loader
}

// newSkillSearcher creates a new skillSearcher instance.
func newSkillSearcher(loader *Loader) *skillSearcher {
	return &skillSearcher{loader: loader}
}

// SearchSkills searches for skills by name or description.
// The search is case-insensitive.
func (s *skillSearcher) SearchSkills(query string) ([]*Skill, error) {
	skillsMap, err := s.loader.Load()
	if err != nil {
		return nil, err
	}

	var results []*Skill
	queryLower := strings.ToLower(query)

	for _, skill := range skillsMap {
		// Search in name
		if strings.Contains(strings.ToLower(skill.Metadata.Name), queryLower) {
			results = append(results, skill)
			continue
		}

		// Search in description
		if strings.Contains(strings.ToLower(skill.Metadata.Description), queryLower) {
			results = append(results, skill)
			continue
		}

		// Search in tags
		for _, tag := range skill.Metadata.Tags {
			if strings.Contains(strings.ToLower(tag), queryLower) {
				results = append(results, skill)
				break
			}
		}
	}

	return results, nil
}

// GetSkillsByCategory returns all skills in a given category.
func (s *skillSearcher) GetSkillsByCategory(category string) ([]*Skill, error) {
	skillsMap, err := s.loader.Load()
	if err != nil {
		return nil, err
	}

	var skills []*Skill
	for _, skill := range skillsMap {
		if skill.Metadata.Category == category {
			skills = append(skills, skill)
		}
	}

	return skills, nil
}

// GetSkillsByTags returns all skills that have any of the given tags.
func (s *skillSearcher) GetSkillsByTags(tags []string) ([]*Skill, error) {
	skillsMap, err := s.loader.Load()
	if err != nil {
		return nil, err
	}

	var skills []*Skill
	for _, skill := range skillsMap {
		for _, tag := range tags {
			if s.hasTag(skill, tag) {
				skills = append(skills, skill)
				break
			}
		}
	}

	return skills, nil
}

// hasTag checks if a skill has a specific tag.
func (s *skillSearcher) hasTag(skill *Skill, tag string) bool {
	for _, skillTag := range skill.Metadata.Tags {
		if strings.EqualFold(skillTag, tag) {
			return true
		}
	}
	return false
}
