package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/aatumaykin/nexbot/internal/constants"
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
	jobs, err := cron.LoadJobs(constants.DefaultWorkDir)
	if err != nil && !os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, constants.MsgErrorLoadingJobs, err)
		os.Exit(1)
	}

	// Create new job
	job := cron.Job{
		ID:       cron.GenerateJobID(),
		Schedule: schedule,
		Command:  command,
		UserID:   constants.CronDefaultUserID,
	}

	// Add to jobs map
	if jobs == nil {
		jobs = make(map[string]cron.Job)
	}
	jobs[job.ID] = job

	// Save jobs
	if err := cron.SaveJobs(constants.DefaultWorkDir, jobs); err != nil {
		fmt.Fprintf(os.Stderr, constants.MsgErrorSavingJobs, err)
		os.Exit(1)
	}

	fmt.Printf(constants.MsgJobAdded)
	fmt.Printf(constants.MsgJobID, job.ID)
	fmt.Printf(constants.MsgJobSchedule, schedule)
	fmt.Printf(constants.MsgJobCommand, command)
	fmt.Printf(constants.MsgJobRemoveNote)
}

func runCronList(cmd *cobra.Command, args []string) {
	// Load jobs
	jobs, err := cron.LoadJobs(constants.DefaultWorkDir)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Print(constants.MsgJobsNotFound)
			return
		}
		fmt.Fprintf(os.Stderr, constants.MsgErrorLoadingJobs, err)
		os.Exit(1)
	}

	if len(jobs) == 0 {
		fmt.Print(constants.MsgJobsNotFound)
		return
	}

	// Print jobs
	fmt.Print(constants.MsgJobsListHeader)
	for _, job := range jobs {
		fmt.Printf(constants.MsgJobID, job.ID)
		fmt.Printf(constants.MsgJobSchedule, job.Schedule)
		fmt.Printf(constants.MsgJobCommand, job.Command)
		if len(job.Metadata) > 0 {
			fmt.Print(constants.MsgJobsMetadata)
			for k, v := range job.Metadata {
				fmt.Printf("%s=%s ", k, v)
			}
			fmt.Println()
		}
		fmt.Printf(constants.MsgJobsListSep)
	}
	fmt.Printf(constants.MsgJobsTotal, len(jobs))
}

func runCronRemove(cmd *cobra.Command, args []string) {
	jobID := args[0]

	// Load jobs
	jobs, err := cron.LoadJobs(constants.DefaultWorkDir)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, constants.MsgErrorNoJobsFound)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, constants.MsgErrorLoadingJobs, err)
		os.Exit(1)
	}

	// Check if job exists
	if _, exists := jobs[jobID]; !exists {
		fmt.Fprintf(os.Stderr, constants.MsgErrorJobNotFound, jobID)
		fmt.Printf(constants.MsgJobNotFoundHint)
		os.Exit(1)
	}

	// Remove job
	delete(jobs, jobID)

	// Save jobs
	if err := cron.SaveJobs(constants.DefaultWorkDir, jobs); err != nil {
		fmt.Fprintf(os.Stderr, constants.MsgErrorSavingJobs, err)
		os.Exit(1)
	}

	fmt.Printf(constants.MsgJobRemoved, jobID)
}

func init() {
	rootCmd.AddCommand(cronCmd)
	cronCmd.AddCommand(cronAddCmd)
	cronCmd.AddCommand(cronListCmd)
	cronCmd.AddCommand(cronRemoveCmd)
}
