package constants

// Cron constants for scheduler configuration and job management.

// CronDefaultUserID is the default user ID for CLI-triggered jobs.
const CronDefaultUserID = "cli"

// CronJobIDFormat is the format string used to generate unique job IDs.
// Uses printf-style formatting with an integer counter.
const CronJobIDFormat = "job_%d"

// CronJobsFile is the filename used to persist job definitions.
const CronJobsFile = "jobs.json"
