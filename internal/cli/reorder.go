package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/alexbrand/backlog/internal/backend"
	"github.com/alexbrand/backlog/internal/output"
	"github.com/spf13/cobra"
)

var (
	reorderBefore string
	reorderAfter  string
	reorderFirst  bool
	reorderLast   bool
)

var reorderCmd = &cobra.Command{
	Use:   "reorder <id>",
	Short: "Change the position of a task in the list",
	Long: `Reorder a task within its status and priority group.

Specify where to place the task using one of: --before, --after, --first, --last.
The reference task (for --before/--after) must have the same status as the target task.

Examples:
  backlog reorder 001 --before 003
  backlog reorder 001 --after 002
  backlog reorder 001 --first
  backlog reorder 001 --last
  backlog reorder 001 --first -f json`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runReorder(args[0])
	},
}

func init() {
	rootCmd.AddCommand(reorderCmd)

	reorderCmd.Flags().StringVar(&reorderBefore, "before", "", "Place task before this task ID")
	reorderCmd.Flags().StringVar(&reorderAfter, "after", "", "Place task after this task ID")
	reorderCmd.Flags().BoolVar(&reorderFirst, "first", false, "Move task to the top of its group")
	reorderCmd.Flags().BoolVar(&reorderLast, "last", false, "Move task to the bottom of its group")
}

func runReorder(id string) error {
	// Validate: exactly one position flag must be set
	position, err := parseReorderPosition()
	if err != nil {
		return InvalidInputError(err.Error())
	}

	// Get backend and connect
	b, _, cleanup, err := connectBackend()
	if err != nil {
		return err
	}
	defer cleanup()

	// Check if backend supports reordering
	reorderer, ok := b.(backend.Reorderer)
	if !ok {
		return fmt.Errorf("backend %q does not support task reordering", b.Name())
	}

	// Perform the reorder
	task, err := reorderer.Reorder(id, position)
	if err != nil {
		errLower := strings.ToLower(err.Error())
		if strings.Contains(errLower, "not found") || strings.Contains(errLower, "404") {
			return NotFoundError(err.Error())
		}
		return err
	}

	// Output the result
	formatter := output.New(output.Format(GetFormat()))
	return formatter.FormatReordered(os.Stdout, task)
}

func parseReorderPosition() (backend.ReorderPosition, error) {
	count := 0
	var pos backend.ReorderPosition

	if reorderBefore != "" {
		count++
		pos.BeforeID = reorderBefore
	}
	if reorderAfter != "" {
		count++
		pos.AfterID = reorderAfter
	}
	if reorderFirst {
		count++
		pos.First = true
	}
	if reorderLast {
		count++
		pos.Last = true
	}

	if count == 0 {
		return pos, fmt.Errorf("one of --before, --after, --first, or --last is required")
	}
	if count > 1 {
		return pos, fmt.Errorf("only one of --before, --after, --first, or --last may be specified")
	}

	return pos, nil
}
