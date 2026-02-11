package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/aatumaykin/nexbot/internal/agent"
	"github.com/aatumaykin/nexbot/internal/logger"
)

// CronTool implements the Tool interface for cron job management.
// It allows scheduling, listing, and managing recurring tasks.
type CronTool struct {
	cronManager agent.CronManager
	logger      *logger.Logger
}

// CronArgs represents the arguments for the cron tool.
type CronArgs struct {
	Action    string `json:"action"`     // Action: "add_recurring", "add_oneshot", "remove", "list"
	Schedule  string `json:"schedule"`   // Cron expression for recurring jobs
	ExecuteAt string `json:"execute_at"` // ISO8601 datetime for oneshot jobs
	Tool      string `json:"tool"`       // Internal tool: "send_message" | "agent"
	Payload   string `json:"payload"`    // Parameters for the tool (JSON string)
	SessionID string `json:"session_id"` // Context session ID
	JobID     string `json:"job_id"`     // Job ID for removal
}

// NewCronTool creates a new CronTool instance.
func NewCronTool(cronManager agent.CronManager, logger *logger.Logger) *CronTool {
	return &CronTool{
		cronManager: cronManager,
		logger:      logger,
	}
}

// Name returns the tool name.
func (t *CronTool) Name() string {
	return "cron"
}

// Description returns a description of what the tool does.
func (t *CronTool) Description() string {
	return "Manages cron jobs for scheduling recurring and one-time tasks. Supports adding recurring jobs, one-time jobs, listing jobs, and removing jobs."
}

// Parameters returns the JSON Schema for the tool's parameters.
func (t *CronTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"action": map[string]any{
				"type":        "string",
				"description": "Action to perform: 'add_recurring' to create a recurring job, 'add_oneshot' to create a one-time job, 'list' to show all jobs, 'remove' to delete a job.",
				"enum":        []string{"add_recurring", "add_oneshot", "remove", "list"},
			},
			"schedule": map[string]any{
				"type":        "string",
				"description": "Cron expression defining the schedule (e.g., '0 * * * *' for hourly). Required for 'add_recurring' action.",
			},
			"execute_at": map[string]any{
				"type":        "string",
				"description": "ISO8601 datetime for one-time job execution (e.g., '2026-02-05T18:00:00Z'). Required for 'add_oneshot' action.",
			},
			"tool": map[string]any{
				"type":        "string",
				"description": "Internal tool to use: 'send_message' (sends message directly to chat) OR 'agent' (processes command via agent).",
				"enum":        []string{"send_message", "agent"},
			},
			"payload": map[string]any{
				"type":        "string",
				"description": "JSON string with parameters for the tool. For 'send_message' or 'agent', this should be {\"message\": \"your text\"}. Required when tool is not empty.",
			},
			"session_id": map[string]any{
				"type":        "string",
				"description": "Session ID for sending message. Format: 'channel:chat_id' (e.g., 'telegram:35052705'). REQUIRED for send_message or agent tools to know where to send.",
			},
			"job_id": map[string]any{
				"type":        "string",
				"description": "Job ID to remove. Required for 'remove' action.",
			},
		},
		"required": []string{"action"},
	}
}

// Execute executes the cron tool.
// args is a JSON-encoded string containing the tool's input parameters.
func (t *CronTool) Execute(args string) (string, error) {
	// Parse arguments
	var params CronArgs
	if err := json.Unmarshal([]byte(args), &params); err != nil {
		return "", fmt.Errorf("failed to parse cron arguments: %w", err)
	}

	// Execute based on action
	switch params.Action {
	case "add_recurring":
		if params.Tool == "" {
			return "", fmt.Errorf("tool parameter is required for add_recurring action. Use 'send_message' or 'agent'")
		}
		return t.addRecurring(context.Background(), map[string]any{
			"schedule":   params.Schedule,
			"tool":       params.Tool,
			"payload":    params.Payload,
			"session_id": params.SessionID,
		})
	case "add_oneshot":
		if params.Tool == "" {
			return "", fmt.Errorf("tool parameter is required for add_oneshot action. Use 'send_message' or 'agent'")
		}
		return t.addOneshot(context.Background(), map[string]any{
			"execute_at": params.ExecuteAt,
			"tool":       params.Tool,
			"payload":    params.Payload,
			"session_id": params.SessionID,
		})
	case "remove":
		return t.removeJob(context.Background(), map[string]any{
			"job_id": params.JobID,
		})
	case "list":
		return t.listJobs(context.Background(), map[string]any{})
	default:
		return "", fmt.Errorf("invalid action: %s. Valid actions: add_recurring, add_oneshot, remove, list", params.Action)
	}
}

// addRecurring creates a recurring cron job.
func (t *CronTool) addRecurring(ctx context.Context, params map[string]any) (string, error) {
	// Extract parameters
	schedule, ok := params["schedule"].(string)
	if !ok || schedule == "" {
		return "", fmt.Errorf("schedule parameter is required for add_recurring action")
	}

	tool, _ := params["tool"].(string)
	payloadStr, _ := params["payload"].(string)
	sessionID, _ := params["session_id"].(string)

	// Payload is required
	if payloadStr == "" {
		return "", fmt.Errorf("payload parameter is required")
	}

	// Parse payload
	var payload map[string]any
	if err := json.Unmarshal([]byte(payloadStr), &payload); err != nil {
		return "", fmt.Errorf("failed to parse payload JSON: %w", err)
	}

	// For send_message or agent tools, session_id is required
	if (tool == "send_message" || tool == "agent") && sessionID == "" {
		return "", fmt.Errorf("session_id is required for send_message and agent tools")
	}

	// Create job using domain model
	job := agent.Job{
		Type:      "recurring",
		Schedule:  schedule,
		Tool:      tool,
		Payload:   payload,
		SessionID: sessionID,
		Metadata: map[string]string{
			"created_by": "cron_tool",
			"created_at": time.Now().Format(time.RFC3339),
		},
	}

	// Add job to scheduler (also saves to storage via UpsertJob)
	jobID, err := t.cronManager.AddJob(job)
	if err != nil {
		return "", fmt.Errorf("failed to add recurring job: %w", err)
	}

	t.logger.InfoCtx(ctx, "recurring job added", logger.Field{Key: "job_id", Value: jobID}, logger.Field{Key: "schedule", Value: schedule})

	// Extract message for display
	message := "N/A"
	if msg, ok := payload["message"].(string); ok {
		message = msg
	}

	return fmt.Sprintf("✅ Recurring job added successfully\n   Job ID: %s\n   Schedule: %s\n   Tool: %s\n   Message: %s", jobID, schedule, tool, message), nil
}

// addOneshot creates a one-time cron job.
func (t *CronTool) addOneshot(ctx context.Context, params map[string]any) (string, error) {
	// Extract parameters
	executeAtStr, ok := params["execute_at"].(string)
	if !ok || executeAtStr == "" {
		return "", fmt.Errorf("execute_at parameter is required for add_oneshot action")
	}

	tool, _ := params["tool"].(string)
	payloadStr, _ := params["payload"].(string)
	sessionID, _ := params["session_id"].(string)

	// Payload is required
	if payloadStr == "" {
		return "", fmt.Errorf("payload parameter is required")
	}

	// Parse payload
	var payload map[string]any
	if err := json.Unmarshal([]byte(payloadStr), &payload); err != nil {
		return "", fmt.Errorf("failed to parse payload JSON: %w", err)
	}

	// For send_message or agent tools, session_id is required
	if (tool == "send_message" || tool == "agent") && sessionID == "" {
		return "", fmt.Errorf("session_id is required for send_message and agent tools")
	}

	// Parse execute_at time
	executeAt, err := time.Parse(time.RFC3339, executeAtStr)
	if err != nil {
		return "", fmt.Errorf("invalid execute_at format (expected ISO8601): %w", err)
	}

	// For oneshot jobs, we use a schedule that matches the specific time
	// Format: second minute hour day month weekday
	schedule := fmt.Sprintf("0 %d %d %d %d *", executeAt.Minute(), executeAt.Hour(), executeAt.Day(), executeAt.Month())

	// Create job using domain model
	job := agent.Job{
		Type:      "oneshot",
		Schedule:  schedule,
		ExecuteAt: &executeAt,
		Tool:      tool,
		Payload:   payload,
		SessionID: sessionID,
		Metadata: map[string]string{
			"created_by": "cron_tool",
			"created_at": time.Now().Format(time.RFC3339),
		},
	}

	// Add job to scheduler (also saves to storage via UpsertJob)
	jobID, err := t.cronManager.AddJob(job)
	if err != nil {
		return "", fmt.Errorf("failed to add oneshot job: %w", err)
	}

	t.logger.InfoCtx(ctx, "oneshot job added", logger.Field{Key: "job_id", Value: jobID}, logger.Field{Key: "execute_at", Value: executeAt})

	// Extract message for display
	message := "N/A"
	if msg, ok := payload["message"].(string); ok {
		message = msg
	}

	return fmt.Sprintf("✅ One-time job added successfully\n   Job ID: %s\n   Execute at: %s\n   Tool: %s\n   Message: %s", jobID, executeAt.Format(time.RFC1123), tool, message), nil
}

// removeJob removes a cron job.
func (t *CronTool) removeJob(ctx context.Context, params map[string]any) (string, error) {
	// Extract parameters
	jobID, ok := params["job_id"].(string)
	if !ok || jobID == "" {
		return "", fmt.Errorf("job_id parameter is required for remove action")
	}

	// Remove job from scheduler
	if err := t.cronManager.RemoveJob(jobID); err != nil {
		return "", fmt.Errorf("failed to remove job: %w", err)
	}

	// Remove from storage
	if err := t.cronManager.RemoveFromStorage(jobID); err != nil {
		t.logger.WarnCtx(ctx, "failed to delete job from storage", logger.Field{Key: "job_id", Value: jobID}, logger.Field{Key: "error", Value: err})
	}

	t.logger.InfoCtx(ctx, "job removed", logger.Field{Key: "job_id", Value: jobID})

	return fmt.Sprintf("✅ Job removed successfully\n   Job ID: %s", jobID), nil
}

// listJobs lists all cron jobs.
func (t *CronTool) listJobs(ctx context.Context, params map[string]any) (string, error) {
	jobs := t.cronManager.ListJobs()

	if len(jobs) == 0 {
		return "No scheduled jobs found.", nil
	}

	var result strings.Builder
	result.WriteString("Scheduled Jobs:\n---------------\n")
	for _, job := range jobs {
		result.WriteString(fmt.Sprintf("Job ID: %s\n", job.ID))
		result.WriteString(fmt.Sprintf("Type: %s\n", job.Type))
		result.WriteString(fmt.Sprintf("Schedule: %s\n", job.Schedule))
		if job.ExecuteAt != nil {
			result.WriteString(fmt.Sprintf("Execute at: %s\n", job.ExecuteAt.Format(time.RFC1123)))
		}
		result.WriteString(fmt.Sprintf("Tool: %s\n", job.Tool))
		if job.Payload != nil {
			if msg, ok := job.Payload["message"].(string); ok {
				result.WriteString(fmt.Sprintf("Message: %s\n", msg))
			}
		}
		if job.SessionID != "" {
			result.WriteString(fmt.Sprintf("Session ID: %s\n", job.SessionID))
		}
		result.WriteString("---------------\n")
	}

	return result.String(), nil
}

// ToSchema returns the OpenAI-compatible schema for this tool.
func (t *CronTool) ToSchema() map[string]any {
	return t.Parameters()
}
