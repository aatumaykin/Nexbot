package heartbeat

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/robfig/cron/v3"
)

// HeartbeatTask represents a periodic task defined in HEARTBEAT.md
type HeartbeatTask struct {
	Name     string `json:"name"`     // Task name (e.g., "Daily Standup")
	Schedule string `json:"schedule"` // Cron expression (e.g., "0 9 * * *")
	Task     string `json:"task"`     // Task description/command
}

// Parser handles parsing of HEARTBEAT.md files
type Parser struct{}

// NewParser creates a new Parser instance
func NewParser() *Parser {
	return &Parser{}
}

// Parse parses HEARTBEAT.md content and extracts heartbeat tasks
// Expected format:
//
//	# Heartbeat Tasks
//
//	## Periodic Reviews
//
//	### Daily Standup
//	- Schedule: "0 9 * * *"
//	- Task: "Review daily progress, check for blocked tasks, update priorities"
func (p *Parser) Parse(content string) ([]HeartbeatTask, error) {
	tasks := []HeartbeatTask{}

	// Split content into sections by level 2 headers (##)
	sections := p.splitSections(content)

	for _, section := range sections {
		// Each section contains level 3 headers (###) for individual tasks
		taskSections := p.splitTaskSections(section)

		for _, taskSection := range taskSections {
			task, err := p.parseTaskSection(taskSection)
			if err != nil {
				// Skip invalid tasks but continue parsing others
				continue
			}

			if task != nil {
				tasks = append(tasks, *task)
			}
		}
	}

	return tasks, nil
}

// Validate validates a HeartbeatTask
func Validate(task HeartbeatTask) error {
	if task.Name == "" {
		return fmt.Errorf("task name cannot be empty")
	}

	if task.Schedule == "" {
		return fmt.Errorf("task schedule cannot be empty")
	}

	// Validate cron expression
	if err := validateCronExpression(task.Schedule); err != nil {
		return fmt.Errorf("invalid cron expression: %w", err)
	}

	if task.Task == "" {
		return fmt.Errorf("task description cannot be empty")
	}

	return nil
}

// splitSections splits content into sections by level 2 headers (##)
func (p *Parser) splitSections(content string) []string {
	var sections []string
	lines := strings.Split(content, "\n")
	var currentSection []string

	for _, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "## ") {
			// New section found
			if len(currentSection) > 0 {
				sections = append(sections, strings.Join(currentSection, "\n"))
			}
			currentSection = []string{line}
		} else {
			currentSection = append(currentSection, line)
		}
	}

	// Add the last section
	if len(currentSection) > 0 {
		sections = append(sections, strings.Join(currentSection, "\n"))
	}

	return sections
}

// splitTaskSections splits a section into task subsections by level 3 headers (###)
func (p *Parser) splitTaskSections(section string) []string {
	var taskSections []string
	lines := strings.Split(section, "\n")
	var currentTask []string

	for _, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "### ") {
			// New task found
			if len(currentTask) > 0 {
				taskSections = append(taskSections, strings.Join(currentTask, "\n"))
			}
			currentTask = []string{line}
		} else {
			currentTask = append(currentTask, line)
		}
	}

	// Add the last task
	if len(currentTask) > 0 {
		taskSections = append(taskSections, strings.Join(currentTask, "\n"))
	}

	return taskSections
}

// parseTaskSection parses a single task section
func (p *Parser) parseTaskSection(section string) (*HeartbeatTask, error) {
	lines := strings.Split(section, "\n")

	task := &HeartbeatTask{}
	var taskLines []string

	// Extract task name from level 3 header
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "### ") {
			task.Name = strings.TrimPrefix(trimmed, "### ")
			task.Name = strings.TrimSpace(task.Name)
			taskLines = lines[i+1:]
			break
		}
	}

	if task.Name == "" {
		return nil, fmt.Errorf("task name not found")
	}

	// Extract Schedule and Task fields from task lines
	for _, line := range taskLines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "- Schedule:") {
			task.Schedule = p.extractQuotedValue(trimmed)
		} else if strings.HasPrefix(trimmed, "- Task:") {
			task.Task = p.extractQuotedValue(trimmed)
		}
	}

	// Return task even if some fields are missing - validation will catch it
	return task, nil
}

// extractQuotedValue extracts a quoted value from a line
// Example: "- Schedule: \"0 9 * * *\"" -> "0 9 * * *"
func (p *Parser) extractQuotedValue(line string) string {
	// Find quoted string
	re := regexp.MustCompile(`"([^"]*)"`)
	matches := re.FindStringSubmatch(line)
	if len(matches) > 1 {
		return matches[1]
	}

	// If no quotes found, try to extract after colon
	parts := strings.SplitN(line, ":", 2)
	if len(parts) > 1 {
		return strings.TrimSpace(parts[1])
	}

	return ""
}

// validateCronExpression validates a cron expression
func validateCronExpression(expr string) error {
	parser := cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	_, err := parser.Parse(expr)
	return err
}

// GetTaskNames returns a slice of task names
func GetTaskNames(tasks []HeartbeatTask) []string {
	names := make([]string, len(tasks))
	for i, task := range tasks {
		names[i] = task.Name
	}
	return names
}

// FormatContext formats heartbeat tasks for inclusion in system context
// Returns a string like: "Active heartbeat tasks: N (daily standup, weekly summary)"
func FormatContext(tasks []HeartbeatTask) string {
	if len(tasks) == 0 {
		return "No active heartbeat tasks"
	}

	names := GetTaskNames(tasks)
	nameList := strings.Join(names, ", ")

	return fmt.Sprintf("Active heartbeat tasks: %d (%s)", len(tasks), nameList)
}
