package heartbeat

import (
	"fmt"
	"os"

	"github.com/aatumaykin/nexbot/internal/logger"
)

// Loader loads heartbeat tasks from HEARTBEAT.md files
type Loader struct {
	parser    *Parser
	workspace string
	logger    *logger.Logger
	tasks     []HeartbeatTask
	isLoaded  bool
}

// NewLoader creates a new Loader instance
func NewLoader(workspace string, logger *logger.Logger) *Loader {
	return &Loader{
		parser:    NewParser(),
		workspace: workspace,
		logger:    logger,
	}
}

// Load loads and parses the HEARTBEAT.md file from the workspace
// If the file doesn't exist, it returns nil tasks without error
func (l *Loader) Load() ([]HeartbeatTask, error) {
	filePath := l.getHeartbeatPath()

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		l.logger.Debug("HEARTBEAT.md not found, skipping")
		l.tasks = nil
		l.isLoaded = true
		return nil, nil
	}

	// Read file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read HEARTBEAT.md: %w", err)
	}

	// Parse content
	tasks, err := l.parser.Parse(string(content))
	if err != nil {
		return nil, fmt.Errorf("failed to parse HEARTBEAT.md: %w", err)
	}

	// Validate tasks
	validTasks := []HeartbeatTask{}
	for _, task := range tasks {
		if err := Validate(task); err != nil {
			l.logger.Warn("skipping invalid heartbeat task",
				logger.Field{Key: "task_name", Value: task.Name},
				logger.Field{Key: "error", Value: err})
			continue
		}
		validTasks = append(validTasks, task)
	}

	l.tasks = validTasks
	l.isLoaded = true

	l.logger.Info("heartbeat tasks loaded",
		logger.Field{Key: "total", Value: len(validTasks)},
		logger.Field{Key: "workspace", Value: l.workspace})

	return validTasks, nil
}

// GetTasks returns the loaded heartbeat tasks
// Returns nil if Load() has not been called or if no tasks were loaded
func (l *Loader) GetTasks() []HeartbeatTask {
	if !l.isLoaded {
		return nil
	}
	return l.tasks
}

// GetContext returns the formatted context string for heartbeat tasks
func (l *Loader) GetContext() string {
	return FormatContext(l.tasks)
}

// getHeartbeatPath returns the path to HEARTBEAT.md
func (l *Loader) getHeartbeatPath() string {
	return fmt.Sprintf("%s/HEARTBEAT.md", l.workspace)
}

// SetTasks sets heartbeat tasks (useful for testing)
func (l *Loader) SetTasks(tasks []HeartbeatTask) {
	l.tasks = tasks
	l.isLoaded = true
}
