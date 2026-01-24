package cli

import (
	"os"
	"strings"

	"github.com/alexbrand/backlog/internal/output"
	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a task",
	Long: `Remove a task from the backlog permanently.

This operation cannot be undone. The task file will be deleted from the
filesystem.

Examples:
  backlog delete 001
  backlog delete 001 -f json`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runDelete(args[0])
	},
}

func init() {
	rootCmd.AddCommand(deleteCmd)
}

func runDelete(id string) error {
	// Get backend and connect
	b, _, cleanup, err := connectBackend()
	if err != nil {
		return err
	}
	defer cleanup()

	// Delete the task
	if err := b.Delete(id); err != nil {
		// Check if this is a "not found" error (case-insensitive check for 404/Not Found)
		errLower := strings.ToLower(err.Error())
		if strings.Contains(errLower, "not found") || strings.Contains(errLower, "404") {
			return NotFoundError(err.Error())
		}
		return err
	}

	// Output the result
	formatter := output.New(output.Format(GetFormat()))
	return formatter.FormatDeleted(os.Stdout, id)
}
