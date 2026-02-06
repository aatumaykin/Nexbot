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
