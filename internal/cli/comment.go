package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/alexbrand/backlog/internal/output"
	"github.com/spf13/cobra"
)

var commentBodyFile string

var commentCmd = &cobra.Command{
	Use:   "comment <id> <message>",
	Short: "Add a comment to a task",
	Long: `Add a comment to a task.

The comment is attributed to the current agent (resolved via --agent-id, BACKLOG_AGENT_ID,
workspace config, or hostname fallback).

Examples:
  backlog comment 001 "Found the bug, working on fix"
  backlog comment 001 "Starting work on implementation" -f json
  backlog comment 001 --body-file=./analysis.md`,
	Args: func(cmd *cobra.Command, args []string) error {
		// With --body-file, we only need the ID
		if commentBodyFile != "" {
			if len(args) != 1 {
				return fmt.Errorf("requires exactly 1 argument (task ID) when using --body-file")
			}
			return nil
		}
		// Without --body-file, we need both ID and message
		if len(args) != 2 {
			return fmt.Errorf("requires exactly 2 arguments: <id> <message>")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]
		var message string

		if commentBodyFile != "" {
			// Read message from file
			content, err := os.ReadFile(commentBodyFile)
			if err != nil {
				return fmt.Errorf("failed to read body file: %w", err)
			}
			message = string(content)
		} else {
			message = args[1]
		}

		return runComment(id, message)
	},
}

func init() {
	commentCmd.Flags().StringVar(&commentBodyFile, "body-file", "", "Read comment body from file")
	rootCmd.AddCommand(commentCmd)
}

func runComment(id string, message string) error {
	// Get backend and connect
	b, _, cleanup, err := connectBackend()
	if err != nil {
		return err
	}
	defer cleanup()

	// Add the comment
	comment, err := b.AddComment(id, message)
	if err != nil {
		// Check for not found error
		if strings.Contains(err.Error(), "not found") {
			return NotFoundError(err.Error())
		}
		return err
	}

	// Output the result
	formatter := output.New(output.Format(GetFormat()))
	return formatter.FormatComment(os.Stdout, comment)
}
