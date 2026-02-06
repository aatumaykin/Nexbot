package workers

import (
	"fmt"
	"time"

	"github.com/aatumaykin/nexbot/internal/logger"
)

// worker is the main worker goroutine that processes tasks from the queue.
func (p *WorkerPool) worker(id int) {
	defer p.wg.Done()
	defer func() {
		if r := recover(); r != nil {
			p.logger.Error("worker panic recovered",
				fmt.Errorf("panic: %v", r),
				logger.Field{Key: "worker_id", Value: id})
		}
	}()

	p.logger.DebugCtx(p.ctx, "worker started",
		logger.Field{Key: "worker_id", Value: id})

	for {
		select {
		case task := <-p.taskQueue:
			p.processTask(id, task)

		case <-p.ctx.Done():
			p.logger.DebugCtx(p.ctx, "worker stopping",
				logger.Field{Key: "worker_id", Value: id})
			return
		}
	}
}

// processTask handles a single task execution with metrics and error handling.
func (p *WorkerPool) processTask(workerID int, task Task) {
	startTime := time.Now()
	taskID := task.ID

	p.logger.DebugCtx(p.ctx, "processing task",
		logger.Field{Key: "worker_id", Value: workerID},
		logger.Field{Key: "task_id", Value: taskID},
		logger.Field{Key: "task_type", Value: task.Type})

	// Use task context if provided, otherwise use pool context
	execCtx := p.ctx
	if task.Context != nil {
		execCtx = task.Context
	}

	// Execute task
	result := p.executeTask(execCtx, task)
	result.Duration = time.Since(startTime)

	// Update metrics
	if result.Error != nil {
		p.incrementFailed()
	} else {
		p.incrementCompleted()
	}
	p.recordDuration(result.Duration)

	// Send result to result channel
	select {
	case p.resultCh <- result:
		// Result sent successfully
	case <-p.ctx.Done():
		p.logger.WarnCtx(p.ctx, "failed to send result, pool shutting down",
			logger.Field{Key: "task_id", Value: taskID})
	}

	p.logger.DebugCtx(p.ctx, "task processed",
		logger.Field{Key: "worker_id", Value: workerID},
		logger.Field{Key: "task_id", Value: taskID},
		logger.Field{Key: "duration_ms", Value: result.Duration.Milliseconds()},
		logger.Field{Key: "error", Value: result.Error})
}
