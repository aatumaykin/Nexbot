package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/aatumaykin/nexbot/internal/cron"
)

var cronCmd = &cobra.Command{
	Use:   "cron",
	Short: "Manage scheduled tasks",
}

var cronAddCmd = &cobra.Command{
	Use:   "add <schedule> <command>",
	Short: "Add a scheduled task",
	Args:  cobra.ExactArgs(2),
	Run:   runCronAdd,
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

func runCronAdd(cmd *cobra.Command, args []string) {
	schedule := args[0]
	command := args[1]

	// Load existing jobs
	jobs, err := loadJobs()
	if err != nil && !os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Error loading jobs: %v\n", err)
		os.Exit(1)
	}

	// Create new job
	job := cron.Job{
		ID:       generateJobID(),
		Schedule: schedule,
		Command:  command,
		UserID:   "cli", // Jobs from CLI are system jobs
	}

	// Validate cron expression
	// Note: We don't use cron.New here as it would require full scheduler initialization
	// Instead, we'll rely on the scheduler to validate when the job is added
	// For CLI, we can do basic validation of the expression format

	// Add to jobs map
	if jobs == nil {
		jobs = make(map[string]cron.Job)
	}
	jobs[job.ID] = job

	// Save jobs
	if err := saveJobs(jobs); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving job: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ Job added successfully\n")
	fmt.Printf("   ID:       %s\n", job.ID)
	fmt.Printf("   Schedule: %s\n", schedule)
	fmt.Printf("   Command:  %s\n", command)
	fmt.Printf("\nNote: Start 'nexbot serve' to activate this job\n")
}

func runCronList(cmd *cobra.Command, args []string) {
	// Load jobs
	jobs, err := loadJobs()
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("No scheduled tasks found.")
			return
		}
		fmt.Fprintf(os.Stderr, "Error loading jobs: %v\n", err)
		os.Exit(1)
	}

	if len(jobs) == 0 {
		fmt.Println("No scheduled tasks found.")
		return
	}

	// Print jobs
	fmt.Println("Scheduled Tasks:")
	fmt.Println("-----------------")
	for _, job := range jobs {
		fmt.Printf("ID:       %s\n", job.ID)
		fmt.Printf("Schedule: %s\n", job.Schedule)
		fmt.Printf("Command:  %s\n", job.Command)
		if len(job.Metadata) > 0 {
			fmt.Print("Metadata: ")
			for k, v := range job.Metadata {
				fmt.Printf("%s=%s ", k, v)
			}
			fmt.Println()
		}
		fmt.Println("-----------------")
	}
	fmt.Printf("Total: %d job(s)\n", len(jobs))
}

func runCronRemove(cmd *cobra.Command, args []string) {
	jobID := args[0]

	// Load jobs
	jobs, err := loadJobs()
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "Error: No jobs found\n")
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "Error loading jobs: %v\n", err)
		os.Exit(1)
	}

	// Check if job exists
	if _, exists := jobs[jobID]; !exists {
		fmt.Fprintf(os.Stderr, "Error: Job '%s' not found\n", jobID)
		fmt.Printf("Use 'nexbot cron list' to see all jobs\n")
		os.Exit(1)
	}

	// Remove job
	delete(jobs, jobID)

	// Save jobs
	if err := saveJobs(jobs); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving jobs: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ Job '%s' removed successfully\n", jobID)
}

// loadJobs loads jobs from workspace/jobs.json
func loadJobs() (map[string]cron.Job, error) {
	configPath, err := getConfigPath()
	if err != nil {
		return nil, fmt.Errorf("failed to get config path: %w", err)
	}

	jobsPath := configPath + "/jobs.json"
	data, err := os.ReadFile(jobsPath)
	if err != nil {
		return nil, err
	}

	var jobs map[string]cron.Job
	if err := json.Unmarshal(data, &jobs); err != nil {
		return nil, fmt.Errorf("failed to parse jobs file: %w", err)
	}

	return jobs, nil
}

// saveJobs saves jobs to workspace/jobs.json
func saveJobs(jobs map[string]cron.Job) error {
	configPath, err := getConfigPath()
	if err != nil {
		return fmt.Errorf("failed to get config path: %w", err)
	}

	jobsPath := configPath + "/jobs.json"
	data, err := json.MarshalIndent(jobs, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal jobs: %w", err)
	}

	if err := os.WriteFile(jobsPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write jobs file: %w", err)
	}

	return nil
}

// getConfigPath returns the config directory path (parent of config file)
func getConfigPath() (string, error) {
	// For now, use current directory
	// TODO: Make this configurable via config file or flag
	return ".", nil
}

// generateJobID generates a unique job ID
func generateJobID() string {
	return fmt.Sprintf("job_%d", os.Getpid())
}

func init() {
	rootCmd.AddCommand(cronCmd)
	cronCmd.AddCommand(cronAddCmd)
	cronCmd.AddCommand(cronListCmd)
	cronCmd.AddCommand(cronRemoveCmd)
}
