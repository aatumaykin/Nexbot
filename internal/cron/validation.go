// Package cron provides cron expression validation logic.
package cron

import (
	"fmt"
	"time"

	"github.com/robfig/cron/v3"
)

// validateCronExpression validates a cron expression using the cron parser
func validateCronExpression(expression string, parser cron.Parser) error {
	_, err := parser.Parse(expression)
	if err != nil {
		return fmt.Errorf("invalid cron expression: %w", err)
	}
	return nil
}

// validateOneshotJobExecution validates if an oneshot job should be executed now
func validateOneshotJobExecution(executeAt *time.Time, now time.Time) (shouldExecute bool) {
	if executeAt == nil {
		return false
	}
	return executeAt.Before(now) || executeAt.Equal(now)
}

// validateJobFields validates job fields based on job type and tool type
func validateJobFields(job Job, parser cron.Parser) error {
	// Oneshot jobs should not have schedule field
	if job.Type == JobTypeOneshot && job.Schedule != "" {
		return fmt.Errorf("oneshot jobs cannot have schedule field")
	}

	// Recurring jobs (or empty type) must have schedule field
	if (job.Type == JobTypeRecurring || job.Type == "") && job.Schedule == "" {
		return fmt.Errorf("invalid cron expression: empty schedule")
	}

	return nil
}
