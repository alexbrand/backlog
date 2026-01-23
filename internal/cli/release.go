package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/alexbrand/backlog/internal/backend"
	"github.com/alexbrand/backlog/internal/linear"
	"github.com/alexbrand/backlog/internal/local"
	"github.com/alexbrand/backlog/internal/output"
	"github.com/spf13/cobra"
)

var releaseComment string

var releaseCmd = &cobra.Command{
	Use:   "release <id>",
	Short: "Release a claimed task back to todo",
	Long: `Release a claimed task back to todo status.

The release operation:
1. Removes the lock on the task
2. Removes the agent label (e.g., agent:claude-1)
3. Unassigns the task
4. Moves the task to todo status

Use this when an agent cannot complete work on a task and wants to make it
available for other agents.

Examples:
  backlog release 001
  backlog release 001 --comment="Blocked on external API"
  backlog release 001 -f json`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runRelease(args[0], releaseComment)
	},
}

func init() {
	releaseCmd.Flags().StringVar(&releaseComment, "comment", "", "Add a comment when releasing the task")
	rootCmd.AddCommand(releaseCmd)
}

func runRelease(id, comment string) error {
	// Get backend and connect
	b, _, cleanup, err := connectBackend()
	if err != nil {
		return err
	}
	defer cleanup()

	// Check if backend supports releasing
	claimer, ok := b.(backend.Claimer)
	if !ok {
		return fmt.Errorf("backend %q does not support task releasing", b.Name())
	}

	// Get the task first so we can display it in the output
	task, err := b.Get(id)
	if err != nil {
		// Check if this is a "not found" error (case-insensitive check for 404/Not Found)
		errLower := strings.ToLower(err.Error())
		if strings.Contains(errLower, "not found") || strings.Contains(errLower, "404") {
			return NotFoundError(err.Error())
		}
		return err
	}

	// Release the task
	if err := claimer.Release(id); err != nil {
		// Check if this is a "not found" error (case-insensitive check for 404/Not Found)
		errLower := strings.ToLower(err.Error())
		if strings.Contains(errLower, "not found") || strings.Contains(errLower, "404") {
			return NotFoundError(err.Error())
		}
		// Check for release conflict error (not claimed or claimed by different agent)
		if _, isReleaseConflict := err.(*local.ReleaseConflictError); isReleaseConflict {
			return ConflictError(err.Error())
		}
		if _, isLinearReleaseConflict := err.(*linear.ReleaseConflictError); isLinearReleaseConflict {
			return ConflictError(err.Error())
		}
		return err
	}

	// Add comment if provided
	if comment != "" {
		if _, err := b.AddComment(id, comment); err != nil {
			return fmt.Errorf("task released but failed to add comment: %w", err)
		}
	}

	// Get the updated task for output
	updatedTask, err := b.Get(id)
	if err != nil {
		// If we can't get the updated task, use the original with updated status
		task.Status = backend.StatusTodo
		updatedTask = task
	}

	// Output the result
	formatter := output.New(output.Format(GetFormat()))
	return formatter.FormatReleased(os.Stdout, updatedTask)
}
