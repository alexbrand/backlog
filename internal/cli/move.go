package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/alexbrand/backlog/internal/backend"
	"github.com/alexbrand/backlog/internal/output"
	"github.com/spf13/cobra"
)

var moveComment string

var moveCmd = &cobra.Command{
	Use:   "move <id> <status>",
	Short: "Transition a task to a new status",
	Long: `Move a task to a new status.

Valid statuses: backlog, todo, in-progress, review, done

Examples:
  backlog move 001 in-progress
  backlog move 001 done
  backlog move 001 review --comment="Ready for review"
  backlog move 001 review -f json`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runMove(args[0], args[1], moveComment)
	},
}

func init() {
	moveCmd.Flags().StringVar(&moveComment, "comment", "", "Add a comment when moving the task")
	rootCmd.AddCommand(moveCmd)
}

func runMove(id, statusStr, comment string) error {
	// Validate status
	status := backend.Status(statusStr)
	if !status.IsValid() {
		return InvalidInputError(fmt.Sprintf("invalid status %q (valid: backlog, todo, in-progress, review, done)", statusStr))
	}

	// Get backend and connect
	b, _, cleanup, err := connectBackend()
	if err != nil {
		return err
	}
	defer cleanup()

	// Get the current task first to capture old status
	currentTask, err := b.Get(id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return NotFoundError(err.Error())
		}
		return err
	}

	oldStatus := currentTask.Status

	// Move the task
	task, err := b.Move(id, status)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return NotFoundError(err.Error())
		}
		return err
	}

	// Add comment if provided
	if comment != "" {
		if _, err := b.AddComment(id, comment); err != nil {
			return fmt.Errorf("task moved but failed to add comment: %w", err)
		}
	}

	// Output the result
	formatter := output.New(output.Format(GetFormat()))
	return formatter.FormatMoved(os.Stdout, task, oldStatus, status)
}
