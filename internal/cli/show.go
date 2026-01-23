package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/alexbrand/backlog/internal/output"
	"github.com/spf13/cobra"
)

var (
	showComments bool
)

var showCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Display full task details",
	Long: `Display the full details of a task including its description.

Use the --comments flag to include the comment thread.

Examples:
  backlog show 001
  backlog show 001 -f json
  backlog show 001 --comments`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runShow(args[0])
	},
}

func init() {
	rootCmd.AddCommand(showCmd)

	showCmd.Flags().BoolVar(&showComments, "comments", false, "Include comment thread")
}

func runShow(id string) error {
	// Get backend and connect
	b, _, cleanup, err := connectBackend()
	if err != nil {
		return err
	}
	defer cleanup()

	// Get the task
	task, err := b.Get(id)
	if err != nil {
		// Check if this is a "not found" error (case-insensitive check for 404/Not Found)
		errLower := strings.ToLower(err.Error())
		if strings.Contains(errLower, "not found") || strings.Contains(errLower, "404") {
			return NotFoundError(err.Error())
		}
		return err
	}

	// Output the task
	formatter := output.New(output.Format(GetFormat()))
	if err := formatter.FormatTask(os.Stdout, task); err != nil {
		return err
	}

	// If comments requested, fetch and display them
	if showComments {
		comments, err := b.ListComments(id)
		if err != nil {
			return fmt.Errorf("failed to list comments: %w", err)
		}

		fmt.Fprintln(os.Stdout)
		if err := formatter.FormatComments(os.Stdout, comments); err != nil {
			return err
		}
	}

	return nil
}
