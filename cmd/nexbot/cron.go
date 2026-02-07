package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/aatumaykin/nexbot/internal/constants"
	"github.com/aatumaykin/nexbot/internal/cron"
	"github.com/aatumaykin/nexbot/internal/logger"
)

var cronCmd = &cobra.Command{
	Use:   "cron",
	Short: "Manage scheduled tasks",
}

var cronListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all scheduled tasks",
	Run:   runCronList,
}

var cronRemoveCmd = &cobra.Command{
	Use:   "remove <job-id>",
	Short: "Remove a scheduled task",
	Args:  cobra.ExactArgs(1),
	Run:   runCronRemove,
}

func runCronList(cmd *cobra.Command, args []string) {
	// Initialize a minimal logger for this command
	log, err := logger.New(logger.Config{
		Level:  "info",
		Format: "text",
		Output: "stdout",
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}

	// Load jobs
	jobs, err := cron.LoadJobs(constants.DefaultWorkDir)
	if err != nil {
		if os.IsNotExist(err) {
			log.Info("No jobs found")
			return
		}
		log.Error("Failed to load jobs", err)
		os.Exit(1)
	}

	if len(jobs) == 0 {
		log.Info("No jobs found")
		return
	}

	// Print jobs
	log.Info("Scheduled jobs:")
	for _, job := range jobs {
		log.Info("Job",
			logger.Field{Key: "id", Value: job.ID},
			logger.Field{Key: "schedule", Value: job.Schedule},
			logger.Field{Key: "tool", Value: job.Tool})
		if len(job.Metadata) > 0 {
			log.Info("Metadata")
			for k, v := range job.Metadata {
				log.Info("Metadata", logger.Field{Key: k, Value: v})
			}
		}
		log.Info("---")
	}
	log.Info("Total jobs", logger.Field{Key: "count", Value: len(jobs)})
}

func runCronRemove(cmd *cobra.Command, args []string) {
	jobID := args[0]

	// Initialize a minimal logger for this command
	log, err := logger.New(logger.Config{
		Level:  "info",
		Format: "text",
		Output: "stdout",
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}

	// Load jobs
	jobs, err := cron.LoadJobs(constants.DefaultWorkDir)
	if err != nil {
		if os.IsNotExist(err) {
			log.Error("No jobs found", nil)
			os.Exit(1)
		}
		log.Error("Failed to load jobs", err)
		os.Exit(1)
	}

	// Check if job exists
	if _, exists := jobs[jobID]; !exists {
		log.Error("Job not found", fmt.Errorf("job ID: %s", jobID))
		log.Info("List jobs with: nexbot cron list")
		os.Exit(1)
	}

	// Remove job
	delete(jobs, jobID)

	// Save jobs
	if err := cron.SaveJobs(constants.DefaultWorkDir, jobs); err != nil {
		log.Error("Failed to save jobs", err)
		os.Exit(1)
	}

	log.Info("Job removed", logger.Field{Key: "job_id", Value: jobID})
}

func init() {
	rootCmd.AddCommand(cronCmd)
	cronCmd.AddCommand(cronListCmd)
	cronCmd.AddCommand(cronRemoveCmd)
}
