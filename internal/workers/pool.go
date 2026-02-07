// Package workers provides an async worker pool for background task execution.
// It supports multiple task types (cron, subagent) and provides result channels
// for asynchronous execution monitoring.
package workers

import (
	"context"
	"sync"

	"github.com/aatumaykin/nexbot/internal/bus"
	"github.com/aatumaykin/nexbot/internal/logger"
)

// WorkerPool manages a pool of goroutine workers for concurrent task execution.
type WorkerPool struct {
	taskQueue  chan Task
	resultCh   chan Result
	workers    int
	wg         *taskWaitGroup
	ctx        context.Context
	cancel     context.CancelFunc
	logger     *logger.Logger
	metrics    *PoolMetrics
	messageBus *bus.MessageBus
}

// NewPool creates a new worker pool with the specified configuration.
func NewPool(workers int, bufferSize int, logger *logger.Logger, messageBus *bus.MessageBus) *WorkerPool {
	ctx, cancel := context.WithCancel(context.Background())
	return &WorkerPool{
		taskQueue:  make(chan Task, bufferSize),
		resultCh:   make(chan Result, bufferSize),
		workers:    workers,
		wg:         newTaskWaitGroup(),
		ctx:        ctx,
		cancel:     cancel,
		logger:     logger,
		metrics:    &PoolMetrics{},
		messageBus: messageBus,
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

// Submit sends a task to the worker pool for execution.
// It blocks if the task queue is full.
func (p *WorkerPool) Submit(task Task) {
	p.incrementSubmitted()

	p.logger.DebugCtx(p.ctx, "task submitted",
		logger.Field{Key: "task_id", Value: task.ID},
		logger.Field{Key: "task_type", Value: task.Type})

	p.taskQueue <- task
}

// SubmitWithContext attempts to submit a task with timeout.
func (p *WorkerPool) SubmitWithContext(ctx context.Context, task Task) error {
	p.incrementSubmitted()

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

// SubmitCronTask submits a cron task to the worker pool.
func (p *WorkerPool) SubmitCronTask(task Task) {
	p.incrementSubmitted()

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
	metrics := p.Metrics()

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
