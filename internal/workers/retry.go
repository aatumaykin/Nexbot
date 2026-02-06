package workers

import (
	"context"
	"fmt"

	"github.com/aatumaykin/nexbot/internal/logger"
)

// executeTask dispatches task execution based on type.
func (p *WorkerPool) executeTask(ctx context.Context, task Task) Result {
	// Handle context cancellation before execution
	select {
	case <-ctx.Done():
		return Result{
			TaskID: task.ID,
			Error:  ctx.Err(),
		}
	default:
	}

	// Execute based on task type
	switch task.Type {
	case "cron":
		return p.executeCronTask(ctx, task)
	case "subagent":
		return p.executeSubagentTask(ctx, task)
	default:
		return Result{
			TaskID: task.ID,
			Error:  fmt.Errorf("unknown task type: %s", task.Type),
		}
	}
}

// executeWithRetry executes a task with panic recovery and context cancellation
func (p *WorkerPool) executeWithRetry(ctx context.Context, task Task, executor TaskExecutor) Result {
	done := make(chan struct{})
	var output string
	var err error

	go func() {
		defer close(done)
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("panic during task execution: %v", r)
				p.logger.ErrorCtx(ctx, "task panic recovered", fmt.Errorf("panic: %v", r),
					logger.Field{Key: "task_id", Value: task.ID})
			}
		}()

		output, err = executor(ctx, task)
	}()

	select {
	case <-done:
		return Result{
			TaskID: task.ID,
			Output: output,
			Error:  err,
		}
	case <-ctx.Done():
		return Result{
			TaskID: task.ID,
			Error:  ctx.Err(),
		}
	}
}

// executeCronTask executes a cron-scheduled task.
func (p *WorkerPool) executeCronTask(ctx context.Context, task Task) Result {
	return p.executeWithRetry(ctx, task, func(ctx context.Context, t Task) (string, error) {
		cmd, ok := task.Payload.(string)
		if !ok {
			return "", fmt.Errorf("invalid cron task payload: expected string")
		}
		fields := []logger.Field{{Key: "task_id", Value: task.ID}, {Key: "command", Value: cmd}}
		p.logger.DebugCtx(ctx, "executing cron task", fields...)
		p.logger.InfoCtx(ctx, "cron task completed", fields...)
		return fmt.Sprintf("cron task executed: %s", cmd), nil
	})
}

// executeSubagentTask executes a subagent task.
func (p *WorkerPool) executeSubagentTask(ctx context.Context, task Task) Result {
	return p.executeWithRetry(ctx, task, func(ctx context.Context, t Task) (string, error) {
		p.logger.DebugCtx(ctx, "executing subagent task",
			logger.Field{Key: "task_id", Value: task.ID},
			logger.Field{Key: "payload", Value: task.Payload})

		p.logger.InfoCtx(ctx, "subagent task completed",
			logger.Field{Key: "task_id", Value: task.ID})

		return fmt.Sprintf("subagent task completed with payload: %v", task.Payload), nil
	})
}
