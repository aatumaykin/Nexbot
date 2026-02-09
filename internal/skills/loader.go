package skills

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/aatumaykin/nexbot/internal/workspace"
)

// Loader manages loading and caching of skills from multiple sources.
// It supports loading from workspace skills and builtin skills with workspace priority.
type Loader struct {
	workspace          *workspace.Workspace
	builtinDir         string
	parser             *Parser
	searcher           SkillSearcher
	cache              map[string]*Skill
	cacheMutex         sync.RWMutex
	cacheEnabled       bool
	loaded             bool
	workspaceSkillsDir string
}

// LoaderConfig represents configuration for the Loader.
type LoaderConfig struct {
	Workspace    *workspace.Workspace
	BuiltinDir   string
	CacheEnabled bool
}

// NewLoader creates a new Loader instance.
func NewLoader(cfg LoaderConfig) *Loader {
	loader := &Loader{
		workspace:    cfg.Workspace,
		builtinDir:   cfg.BuiltinDir,
		parser:       NewParser(),
		cache:        make(map[string]*Skill),
		cacheEnabled: cfg.CacheEnabled,
		loaded:       false,
	}
	loader.searcher = newSkillSearcher(loader)
	return loader
}

// Load loads all skills from configured directories.
// Skills are loaded from both builtin and workspace directories,
// with workspace skills taking priority over builtin skills.
// Returns a map of skill name to Skill, or an error.
func (l *Loader) Load() (map[string]*Skill, error) {
	// If already loaded and cache is enabled, return cached skills
	if l.loaded && l.cacheEnabled {
		l.cacheMutex.RLock()
		defer l.cacheMutex.RUnlock()
		return l.copyCache(), nil
	}

	// Determine workspace skills directory
	if l.workspace != nil {
		l.workspaceSkillsDir = l.workspace.Subpath(workspace.SubdirSkills)
	}

	// Load builtin skills
	builtinSkills, err := l.loadDirectory(l.builtinDir, "builtin")
	if err != nil {
		return nil, fmt.Errorf("failed to load builtin skills: %w", err)
	}

	// Load workspace skills
	var workspaceSkills map[string]*Skill
	if l.workspaceSkillsDir != "" {
		workspaceSkills, err = l.loadDirectory(l.workspaceSkillsDir, "workspace")
		if err != nil {
			return nil, fmt.Errorf("failed to load workspace skills: %w", err)
		}
	} else {
		workspaceSkills = make(map[string]*Skill)
	}

	// Merge skills with workspace priority
	mergedSkills := l.mergeSkills(builtinSkills, workspaceSkills)

	// Always update cache and loaded flag
	l.cacheMutex.Lock()
	l.cache = mergedSkills
	l.loaded = true
	l.cacheMutex.Unlock()

	return mergedSkills, nil
}

// LoadFile loads a single skill file.
// This bypasses caching and loads the skill directly.
func (l *Loader) LoadFile(filePath string) (*Skill, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read skill file: %w", err)
	}

	skill, err := l.parser.Parse(string(content))
	if err != nil {
		return nil, fmt.Errorf("failed to parse skill file: %w", err)
	}

	skill.FilePath = filePath

	return skill, nil
}

// Reload reloads all skills from configured directories.
// This clears the cache and reloads all skills.
func (l *Loader) Reload() (map[string]*Skill, error) {
	l.cacheMutex.Lock()
	l.cache = make(map[string]*Skill)
	l.loaded = false
	l.cacheMutex.Unlock()

	return l.Load()
}

// Get retrieves a skill by name.
// Returns nil if the skill is not found.
func (l *Loader) Get(name string) (*Skill, error) {
	// Ensure skills are loaded
	if !l.loaded {
		_, err := l.Load()
		if err != nil {
			return nil, err
		}
	}

	l.cacheMutex.RLock()
	defer l.cacheMutex.RUnlock()

	return l.cache[name], nil
}

// List returns all loaded skill names.
func (l *Loader) List() ([]string, error) {
	// Ensure skills are loaded
	if !l.loaded {
		_, err := l.Load()
		if err != nil {
			return nil, err
		}
	}

	l.cacheMutex.RLock()
	defer l.cacheMutex.RUnlock()

	names := make([]string, 0, len(l.cache))
	for name := range l.cache {
		names = append(names, name)
	}

	return names, nil
}

// ClearCache clears the skills cache.
func (l *Loader) ClearCache() {
	l.cacheMutex.Lock()
	defer l.cacheMutex.Unlock()
	l.cache = make(map[string]*Skill)
	l.loaded = false
}

// loadDirectory loads all SKILL.md files from a directory.
func (l *Loader) loadDirectory(dirPath, source string) (map[string]*Skill, error) {
	skills := make(map[string]*Skill)

	// Check if directory exists
	if dirPath == "" {
		return skills, nil
	}

	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		// Directory doesn't exist, return empty skills
		return skills, nil
	}

	// Walk directory for SKILL.md files
	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Check if file is SKILL.md
		if info.Name() == "SKILL.md" {
			// Load skill
			skill, err := l.LoadFile(path)
			if err != nil {
				return fmt.Errorf("failed to load skill from %s: %w", path, err)
			}

			// Check for duplicate skill names within the same source
			if _, exists := skills[skill.Metadata.Name]; exists {
				return fmt.Errorf("duplicate skill name '%s' in %s (already defined in %s)",
					skill.Metadata.Name, path, skills[skill.Metadata.Name].FilePath)
			}

			skills[skill.Metadata.Name] = skill
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return skills, nil
}

// mergeSkills merges skills from builtin and workspace sources.
// Workspace skills take priority over builtin skills.
func (l *Loader) mergeSkills(builtin, workspace map[string]*Skill) map[string]*Skill {
	merged := make(map[string]*Skill)

	// First add all builtin skills
	for name, skill := range builtin {
		merged[name] = skill
	}

	// Then override with workspace skills
	for name, skill := range workspace {
		merged[name] = skill
	}

	return merged
}

// copyCache creates a copy of the skills cache.
func (l *Loader) copyCache() map[string]*Skill {
	copy := make(map[string]*Skill, len(l.cache))
	for k, v := range l.cache {
		copy[k] = v
	}
	return copy
}

// GetSkillsByCategory returns all skills in a given category.
func (l *Loader) GetSkillsByCategory(category string) ([]*Skill, error) {
	return l.searcher.GetSkillsByCategory(category)
}

// GetSkillsByTags returns all skills that have any of the given tags.
func (l *Loader) GetSkillsByTags(tags []string) ([]*Skill, error) {
	return l.searcher.GetSkillsByTags(tags)
}

// SearchSkills searches for skills by name or description.
// The search is case-insensitive.
func (l *Loader) SearchSkills(query string) ([]*Skill, error) {
	return l.searcher.SearchSkills(query)
}

// ValidateAll validates all loaded skills.
// Returns a map of skill name to validation error (empty if all valid).
func (l *Loader) ValidateAll() (map[string]error, error) {
	skillsMap, err := l.Load()
	if err != nil {
		return nil, err
	}

	errors := make(map[string]error)
	for name, skill := range skillsMap {
		if err := skill.Validate(); err != nil {
			errors[name] = err
		}
	}

	return errors, nil
}

// Stats returns statistics about loaded skills.
func (l *Loader) Stats() (*Stats, error) {
	skillsMap, err := l.Load()
	if err != nil {
		return nil, err
	}

	stats := &Stats{
		Total:      len(skillsMap),
		Categories: make(map[string]int),
	}

	for _, skill := range skillsMap {
		if skill.Metadata.Category != "" {
			stats.Categories[skill.Metadata.Category]++
		}

		stats.ParameterCount += len(skill.Metadata.Parameters)
		stats.ExampleCount += len(skill.Metadata.Examples)
	}

	return stats, nil
}

// Stats represents statistics about loaded skills.
type Stats struct {
	Total          int            `json:"total"`
	Categories     map[string]int `json:"categories"`
	ParameterCount int            `json:"parameter_count"`
	ExampleCount   int            `json:"example_count"`
}
