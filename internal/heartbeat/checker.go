package heartbeat

import (
	"context"
	"sync"
	"time"

	"github.com/aatumaykin/nexbot/internal/logger"
)

// heartbeatPrompt is the prompt used for heartbeat checks.
const heartbeatPrompt = "Read HEARTBEAT.md from workspace. Follow it strictly. Do not infer or repeat old tasks from prior chats. If nothing needs attention, reply HEARTBEAT_OK."

// heartbeatOKToken is the token that indicates heartbeat check passed successfully.
const heartbeatOKToken = "HEARTBEAT_OK"

// Agent interface represents an agent that can process heartbeat checks.
type Agent interface {
	ProcessHeartbeatCheck(ctx context.Context) (string, error)
}

// Checker periodically checks the heartbeat status by calling the agent's ProcessHeartbeatCheck method.
// It runs on a configurable interval and can be started and stopped.
type Checker struct {
	interval time.Duration
	agent    Agent
	logger   *logger.Logger
	ctx      context.Context
	cancel   context.CancelFunc
	started  bool
	mu       sync.RWMutex
}

// NewChecker creates a new heartbeat checker.
// intervalMinutes specifies the check interval in minutes.
// agent is the agent that will process heartbeat checks.
// logger is used for logging.
func NewChecker(intervalMinutes int, agent Agent, logger *logger.Logger) *Checker {
	return &Checker{
		interval: time.Duration(intervalMinutes) * time.Minute,
		agent:    agent,
		logger:   logger,
		started:  false,
	}
}

// Start begins the heartbeat checker loop.
// It runs ProcessHeartbeatCheck at the configured interval.
// If the checker is already started, it returns an error.
func (c *Checker) Start() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.started {
		return nil // Already started, don't error
	}

	c.ctx, c.cancel = context.WithCancel(context.Background())
	c.started = true

	c.logger.Info("Heartbeat checker started", logger.Field{Key: "interval", Value: c.interval})

	// Start the heartbeat loop in a goroutine
	go c.run()

	return nil
}

// Stop halts the heartbeat checker loop.
// It waits for the current heartbeat check to complete before returning.
func (c *Checker) Stop() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.started {
		return nil // Already stopped, don't error
	}

	c.logger.Info("Heartbeat checker stopping")

	c.cancel()
	c.started = false

	return nil
}

// run is the main loop that performs heartbeat checks at regular intervals.
func (c *Checker) run() {
	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			c.logger.Info("Heartbeat checker stopped")
			return
		case <-ticker.C:
			c.check()
		}
	}
}

// check performs a single heartbeat check.
func (c *Checker) check() {
	c.logger.Info("Performing heartbeat check")

	// Call agent to process heartbeat check
	response, err := c.agent.ProcessHeartbeatCheck(c.ctx)
	if err != nil {
		c.logger.Error("Heartbeat check failed", err,
			logger.Field{Key: "response", Value: response})
		return
	}

	// Process the response
	c.processResponse(response)
}

// processResponse processes the response from a heartbeat check.
// If the response contains HEARTBEAT_OK, it logs that everything is good.
// Otherwise, it assumes the LLM has already handled any necessary actions via tools.
func (c *Checker) processResponse(response string) {
	if response == "" {
		c.logger.Warn("Heartbeat check returned empty response")
		return
	}

	// Check if response contains the OK token
	if containsToken(response, heartbeatOKToken) {
		c.logger.Info("Heartbeat check: all good", logger.Field{Key: "response", Value: response})
	} else {
		// LLM has already sent notifications/actions via tools (e.g., send_message)
		c.logger.Info("Heartbeat check: action taken by LLM", logger.Field{Key: "response", Value: response})
	}
}

// containsToken checks if the response contains the specified token.
func containsToken(response, token string) bool {
	return response == token || response == "\n"+token || response == token+"\n"
}
