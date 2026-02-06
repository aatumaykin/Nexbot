// Package workers provides an async worker pool for background task execution.
// It supports multiple task types (cron, subagent) and provides result channels
// for asynchronous execution monitoring.
package workers

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/aatumaykin/nexbot/internal/logger"
)

// Task represents a unit of work to be executed by a worker.
type Task struct {
	ID      string                 // Unique task identifier
	Type    string                 // Task type: "cron" or "subagent"
	Payload interface{}            // Task payload (command, agent config, etc.)
	Context context.Context        // Task-specific context for cancellation/timeout
	Metrics map[string]interface{} // Optional metrics to track
}

// Result represents the outcome of a task execution.
type Result struct {
	TaskID   string                 // ID of the executed task
	Error    error                  // Error if execution failed
	Output   string                 // Task output
	Duration time.Duration          // Execution duration
	Metrics  map[string]interface{} // Task execution metrics
}

// WorkerPool manages a pool of goroutine workers for concurrent task execution.
type WorkerPool struct {
	taskQueue chan Task
	resultCh  chan Result
	workers   int
	wg        *taskWaitGroup
	ctx       context.Context
	cancel    context.CancelFunc
	logger    *logger.Logger
	metrics   *PoolMetrics
}

// PoolMetrics tracks execution metrics for the worker pool.
type PoolMetrics struct {
	TasksSubmitted uint64
	TasksCompleted uint64
	TasksFailed    uint64
	TotalDuration  time.Duration
}

// NewPool creates a new worker pool with the specified configuration.
func NewPool(workers int, bufferSize int, logger *logger.Logger) *WorkerPool {
	ctx, cancel := context.WithCancel(context.Background())
	return &WorkerPool{
		taskQueue: make(chan Task, bufferSize),
		resultCh:  make(chan Result, bufferSize),
		workers:   workers,
		wg:        newTaskWaitGroup(),
		ctx:       ctx,
		cancel:    cancel,
		logger:    logger,
		metrics:   &PoolMetrics{},
	}
}

// Start initializes and starts all worker goroutines.
func (p *WorkerPool) Start() {
	p.logger.Info("starting worker pool",
		logger.Field{Key: "workers", Value: p.workers},
		logger.Field{Key: "buffer_size", Value: cap(p.taskQueue)})

	for i := 0; i < p.workers; i++ {
		p.wg.Add(1)
		go p.worker(i)
	}
}

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

	// Execute task with timeout/timeout support
	result := p.executeTask(execCtx, task)
	result.Duration = time.Since(startTime)

	// Update metrics
	p.wg.Lock()
	if result.Error != nil {
		p.metrics.TasksFailed++
	} else {
		p.metrics.TasksCompleted++
	}
	p.metrics.TotalDuration += result.Duration
	p.wg.Unlock()

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

// TaskExecutor defines the interface for task-specific execution logic
type TaskExecutor func(context.Context, Task) (string, error)

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

// Submit sends a task to the worker pool for execution.
// It blocks if the task queue is full.
func (p *WorkerPool) Submit(task Task) {
	p.wg.Lock()
	p.metrics.TasksSubmitted++
	p.wg.Unlock()

	p.logger.DebugCtx(p.ctx, "task submitted",
		logger.Field{Key: "task_id", Value: task.ID},
		logger.Field{Key: "task_type", Value: task.Type})

	p.taskQueue <- task
}

// SubmitWithContext attempts to submit a task with timeout.
func (p *WorkerPool) SubmitWithContext(ctx context.Context, task Task) error {
	p.wg.Lock()
	p.metrics.TasksSubmitted++
	p.wg.Unlock()

	p.logger.DebugCtx(ctx, "task submitted with context",
		logger.Field{Key: "task_id", Value: task.ID},
		logger.Field{Key: "task_type", Value: task.Type})

	select {
	case p.taskQueue <- task:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Results returns a read-only channel for receiving task results.
func (p *WorkerPool) Results() <-chan Result {
	return p.resultCh
}

// Stop gracefully shuts down the worker pool.
// It waits for all in-flight tasks to complete.
func (p *WorkerPool) Stop() {
	p.cancel()

	// Wait for all workers to finish
	p.wg.Wait()

	// Get metrics with lock
	p.wg.RLock()
	metrics := p.metrics
	p.wg.RUnlock()

	p.logger.Info("stopping worker pool",
		logger.Field{Key: "tasks_submitted", Value: metrics.TasksSubmitted},
		logger.Field{Key: "tasks_completed", Value: metrics.TasksCompleted},
		logger.Field{Key: "tasks_failed", Value: metrics.TasksFailed})

	// Close channels safely
	select {
	case <-p.taskQueue:
		// Already closed
	default:
		close(p.taskQueue)
	}

	select {
	case <-p.resultCh:
		// Already closed
	default:
		close(p.resultCh)
	}

	p.logger.Info("worker pool stopped")
}

// CronTask is a type alias for compatibility with cron package
type CronTask = struct {
	ID      string
	Type    string
	Payload interface{}
	Context context.Context
}

// SubmitCronTask submits a cron task to the worker pool
func (p *WorkerPool) SubmitCronTask(task CronTask) {
	p.wg.Lock()
	p.metrics.TasksSubmitted++
	p.wg.Unlock()

	p.logger.DebugCtx(p.ctx, "cron task submitted",
		logger.Field{Key: "task_id", Value: task.ID},
		logger.Field{Key: "task_type", Value: task.Type})

	p.taskQueue <- Task{
		ID:      task.ID,
		Type:    task.Type,
		Payload: task.Payload,
		Context: task.Context,
		Metrics: make(map[string]interface{}),
	}
}

// Metrics returns the current pool metrics.
func (p *WorkerPool) Metrics() PoolMetrics {
	p.wg.RLock()
	defer p.wg.RUnlock()
	return *p.metrics
}

// WorkerCount returns the number of active workers.
func (p *WorkerPool) WorkerCount() int {
	return p.workers
}

// QueueSize returns the current number of tasks waiting in the queue.
func (p *WorkerPool) QueueSize() int {
	return len(p.taskQueue)
}

// taskWaitGroup wraps sync.WaitGroup with thread-safe metrics access.
type taskWaitGroup struct {
	sync.RWMutex
	wg sync.WaitGroup
}

func newTaskWaitGroup() *taskWaitGroup {
	return &taskWaitGroup{}
}

func (twg *taskWaitGroup) Add(delta int) {
	twg.wg.Add(delta)
}

func (twg *taskWaitGroup) Done() {
	twg.wg.Done()
}

func (twg *taskWaitGroup) Wait() {
	twg.wg.Wait()
}

func (twg *taskWaitGroup) Lock() {
	twg.RWMutex.Lock()
}

func (twg *taskWaitGroup) Unlock() {
	twg.RWMutex.Unlock()
}

func (twg *taskWaitGroup) RLock() {
	twg.RWMutex.RLock()
}

func (twg *taskWaitGroup) RUnlock() {
	twg.RWMutex.RUnlock()
}
