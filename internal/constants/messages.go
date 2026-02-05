package constants

// Package messages contains all text message constants used throughout the Nexbot application.

// Command messages
const (
	// MsgSessionCleared is the confirmation message after clearing a session.
	MsgSessionCleared = "✅ Session cleared. Starting a fresh conversation!"

	// MsgStatusError is the error message when status information cannot be retrieved.
	MsgStatusError = "❌ Failed to get status information. Please try again later."

	// MsgRestarting is the notification message when a restart command is received.
	MsgRestarting = "🔄 Restarting..."

	// MsgErrorFormat is the prefix for formatting error messages.
	MsgErrorFormat = "Error: %v"
)

// Status messages
const (
	// MsgStatusHeader is the header for the status display.
	MsgStatusHeader = "📊 **Session Status**\n\n"

	// MsgStatusSessionID is the label for the session ID field.
	MsgStatusSessionID = "**Session ID:** `%s`\n"

	// MsgStatusMessages is the label for the message count field.
	MsgStatusMessages = "**Messages:** %d\n"

	// MsgStatusSessionSize is the label for the session size field.
	MsgStatusSessionSize = "**Session Size:** %s\n"

	// MsgStatusLLMConfig is the header for LLM configuration section.
	MsgStatusLLMConfig = "\n**LLM Configuration:**\n"

	// MsgStatusModel is the label for the model field.
	MsgStatusModel = "**Model:** %s\n"

	// MsgStatusTemp is the label for the temperature field.
	MsgStatusTemp = "**Temperature:** %.2f\n"

	// MsgStatusMaxTokens is the label for the max tokens field.
	MsgStatusMaxTokens = "**Max Tokens:** %d\n"
)

// Config messages
const (
	// MsgConfigLoadError is the error message when configuration loading fails.
	MsgConfigLoadError = "❌ Failed to load configuration: %v\n"

	// MsgConfigValidationError is the message when configuration validation fails.
	MsgConfigValidationError = "❌ Configuration validation failed:\n"

	// MsgConfigValid is the message when configuration is successfully loaded and validated.
	MsgConfigValid = "✅ Configuration loaded"

	// MsgConfigValidatePrefix is the prefix for configuration validation errors.
	MsgConfigValidatePrefix = "  - %v\n"
)

// Error messages
const (
	// MsgErrorLoadingJobs is the error message when jobs file cannot be loaded.
	MsgErrorLoadingJobs = "Error loading jobs: %v\n"

	// MsgErrorSavingJobs is the error message when jobs cannot be saved.
	MsgErrorSavingJobs = "Error saving job: %v\n"

	// MsgErrorJobNotFound is the error message when a specific job is not found.
	MsgErrorJobNotFound = "Error: Job '%s' not found\n"

	// MsgErrorNoJobsFound is the message when no jobs are found.
	MsgErrorNoJobsFound = "Error: No jobs found\n"

	// MsgErrorConfigLoad is the error message when config cannot be loaded.
	MsgErrorConfigLoad = "failed to get config path: %w"
)

// Job messages
const (
	// MsgJobAdded is the success message when a job is added.
	MsgJobAdded = "✅ Job added successfully\n"

	// MsgJobID is the label for the job ID field.
	MsgJobID = "   ID:       %s\n"

	// MsgJobSchedule is the label for the job schedule field.
	MsgJobSchedule = "   Schedule: %s\n"

	// MsgJobCommand is the label for the job command field.
	MsgJobCommand = "   Command:  %s\n"

	// MsgJobRemoveNote is the note about activating a job.
	MsgJobRemoveNote = "\nNote: Start 'nexbot serve' to activate this job\n"

	// MsgJobRemoved is the success message when a job is removed.
	MsgJobRemoved = "✅ Job '%s' removed successfully\n"

	// MsgJobNotFoundHint is the hint when a job is not found.
	MsgJobNotFoundHint = "Use 'nexbot cron list' to see all jobs\n"
)

// Jobs list messages
const (
	// MsgJobsListHeader is the header for the jobs list display.
	MsgJobsListHeader = "Scheduled Tasks:\n-----------------\n"

	// MsgJobsListSep is the separator between jobs in the list.
	MsgJobsListSep = "-----------------\n"

	// MsgJobsMetadata is the label for job metadata.
	MsgJobsMetadata = "Metadata: "

	// MsgJobsTotal is the message showing the total count of jobs.
	MsgJobsTotal = "Total: %d job(s)\n"

	// MsgJobsNotFound is the message when no jobs are found.
	MsgJobsNotFound = "No scheduled tasks found."
)

// Telegram messages
const (
	// MsgTelegramStartup is the startup message for Telegram connector.
	MsgTelegramStartup = "📱 Initializing Telegram connector"

	// TelegramMsgAuthError is the error message for Telegram authentication failure.
	TelegramMsgAuthError = "❌ Z.ai API key is not configured in [llm.zai.api_key]"
)
