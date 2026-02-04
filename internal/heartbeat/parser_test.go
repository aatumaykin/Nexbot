package heartbeat

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse_ValidContent(t *testing.T) {
	parser := NewParser()

	content := `# Heartbeat Tasks

## Periodic Reviews

### Daily Standup
- Schedule: "0 0 9 * * *"
- Task: "Review daily progress, check for blocked tasks, update priorities"

### Weekly Summary
- Schedule: "0 0 17 * * 5"
- Task: "Generate weekly summary of completed tasks and planned work"

### Health Check
- Schedule: "0 0 */6 * * *"
- Task: "Check system health, monitor logs, alert on errors"
`

	tasks, err := parser.Parse(content)

	require.NoError(t, err)
	assert.Len(t, tasks, 3)

	assert.Equal(t, "Daily Standup", tasks[0].Name)
	assert.Equal(t, "0 0 9 * * *", tasks[0].Schedule)
	assert.Equal(t, "Review daily progress, check for blocked tasks, update priorities", tasks[0].Task)

	assert.Equal(t, "Weekly Summary", tasks[1].Name)
	assert.Equal(t, "0 0 17 * * 5", tasks[1].Schedule)
	assert.Equal(t, "Generate weekly summary of completed tasks and planned work", tasks[1].Task)

	assert.Equal(t, "Health Check", tasks[2].Name)
	assert.Equal(t, "0 0 */6 * * *", tasks[2].Schedule)
	assert.Equal(t, "Check system health, monitor logs, alert on errors", tasks[2].Task)
}

func TestParse_InvalidSchedule(t *testing.T) {
	parser := NewParser()

	content := `# Heartbeat Tasks

## Periodic Reviews

### Daily Standup
- Schedule: "invalid-cron"
- Task: "Review daily progress"
`

	tasks, err := parser.Parse(content)

	require.NoError(t, err)
	assert.Len(t, tasks, 1)
	assert.Equal(t, "Daily Standup", tasks[0].Name)
	assert.Equal(t, "invalid-cron", tasks[0].Schedule)
}

func TestParse_EmptyContent(t *testing.T) {
	parser := NewParser()

	content := ""

	tasks, err := parser.Parse(content)

	require.NoError(t, err)
	assert.Len(t, tasks, 0)
}

func TestParse_OnlyHeader(t *testing.T) {
	parser := NewParser()

	content := `# Heartbeat Tasks
`

	tasks, err := parser.Parse(content)

	require.NoError(t, err)
	assert.Len(t, tasks, 0)
}

func TestParse_OnlySectionHeader(t *testing.T) {
	parser := NewParser()

	content := `# Heartbeat Tasks

## Periodic Reviews
`

	tasks, err := parser.Parse(content)

	require.NoError(t, err)
	assert.Len(t, tasks, 0)
}

func TestParse_MultipleSections(t *testing.T) {
	parser := NewParser()

	content := `# Heartbeat Tasks

## Periodic Reviews

### Daily Standup
- Schedule: "0 9 * * *"
- Task: "Review daily progress"

## System Maintenance

### Backup
- Schedule: "0 2 * * *"
- Task: "Run system backup"
`

	tasks, err := parser.Parse(content)

	require.NoError(t, err)
	assert.Len(t, tasks, 2)

	assert.Equal(t, "Daily Standup", tasks[0].Name)
	assert.Equal(t, "Backup", tasks[1].Name)
}

func TestParse_TaskWithoutQuotes(t *testing.T) {
	parser := NewParser()

	content := `# Heartbeat Tasks

## Periodic Reviews

### Daily Standup
- Schedule: 0 9 * * *
- Task: Review daily progress
`

	tasks, err := parser.Parse(content)

	require.NoError(t, err)
	assert.Len(t, tasks, 1)
	assert.Equal(t, "0 9 * * *", tasks[0].Schedule)
	assert.Equal(t, "Review daily progress", tasks[0].Task)
}

func TestParse_IncompleteTask(t *testing.T) {
	parser := NewParser()

	content := `# Heartbeat Tasks

## Periodic Reviews

### Daily Standup
- Schedule: "0 9 * * *"
`

	tasks, err := parser.Parse(content)

	require.NoError(t, err)
	assert.Len(t, tasks, 1)
	assert.Equal(t, "Daily Standup", tasks[0].Name)
	assert.Equal(t, "0 9 * * *", tasks[0].Schedule)
	assert.Equal(t, "", tasks[0].Task)
}

func TestValidate_ValidTask(t *testing.T) {
	task := HeartbeatTask{
		Name:     "Daily Standup",
		Schedule: "0 0 9 * * *",
		Task:     "Review daily progress",
	}

	err := Validate(task)

	assert.NoError(t, err)
}

func TestValidate_EmptyName(t *testing.T) {
	task := HeartbeatTask{
		Name:     "",
		Schedule: "0 9 * * *",
		Task:     "Review daily progress",
	}

	err := Validate(task)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "task name cannot be empty")
}

func TestValidate_EmptySchedule(t *testing.T) {
	task := HeartbeatTask{
		Name:     "Daily Standup",
		Schedule: "",
		Task:     "Review daily progress",
	}

	err := Validate(task)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "task schedule cannot be empty")
}

func TestValidate_InvalidCron(t *testing.T) {
	task := HeartbeatTask{
		Name:     "Daily Standup",
		Schedule: "invalid-cron",
		Task:     "Review daily progress",
	}

	err := Validate(task)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid cron expression")
}

func TestValidate_EmptyTask(t *testing.T) {
	task := HeartbeatTask{
		Name:     "Daily Standup",
		Schedule: "0 0 9 * * *",
		Task:     "",
	}

	err := Validate(task)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "task description cannot be empty")
}

func TestGetTaskNames(t *testing.T) {
	tasks := []HeartbeatTask{
		{Name: "Task 1"},
		{Name: "Task 2"},
		{Name: "Task 3"},
	}

	names := GetTaskNames(tasks)

	assert.Len(t, names, 3)
	assert.Equal(t, []string{"Task 1", "Task 2", "Task 3"}, names)
}

func TestGetTaskNames_Empty(t *testing.T) {
	tasks := []HeartbeatTask{}

	names := GetTaskNames(tasks)

	assert.Len(t, names, 0)
}

func TestFormatContext_WithTasks(t *testing.T) {
	tasks := []HeartbeatTask{
		{Name: "Daily Standup", Schedule: "0 9 * * *", Task: "Review"},
		{Name: "Weekly Summary", Schedule: "0 17 * * 5", Task: "Summarize"},
	}

	context := FormatContext(tasks)

	assert.Equal(t, "Active heartbeat tasks: 2 (Daily Standup, Weekly Summary)", context)
}

func TestFormatContext_NoTasks(t *testing.T) {
	tasks := []HeartbeatTask{}

	context := FormatContext(tasks)

	assert.Equal(t, "No active heartbeat tasks", context)
}

func TestExtractQuotedValue(t *testing.T) {
	parser := NewParser()

	tests := []struct {
		input    string
		expected string
	}{
		{`- Schedule: "0 9 * * *"`, "0 9 * * *"},
		{`- Task: "Review daily progress"`, "Review daily progress"},
		{`- Schedule: 0 9 * * *`, "0 9 * * *"},
		{`- Task: Review daily progress`, "Review daily progress"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parser.extractQuotedValue(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestValidateCronExpression_Valid(t *testing.T) {
	expressions := []string{
		"0 0 9 * * *",
		"0 0 */6 * * *",
		"0 0 17 * * 5",
		"0 0 0 * * *",
		"0 */5 * * * *",
	}

	for _, expr := range expressions {
		t.Run(expr, func(t *testing.T) {
			err := validateCronExpression(expr)
			assert.NoError(t, err)
		})
	}
}

func TestValidateCronExpression_Invalid(t *testing.T) {
	expressions := []string{
		"invalid",
		"0 0 25 * * *", // Invalid hour
		"0 0 * 32 * *", // Invalid day of month
		"0 0 * * 13 *", // Invalid month
		"0 0 * * * 8",  // Invalid day of week
	}

	for _, expr := range expressions {
		t.Run(expr, func(t *testing.T) {
			err := validateCronExpression(expr)
			assert.Error(t, err)
		})
	}
}
