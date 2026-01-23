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
	editTitle       string
	editPriority    string
	editDescription string
	editAddLabels   []string
	editRemoveLabel []string
)

var editCmd = &cobra.Command{
	Use:   "edit <id>",
	Short: "Modify task fields",
	Long: `Edit an existing task's fields.

You can update the title, priority, description, and labels using the
available flags. Only the fields you specify will be changed.

Examples:
  backlog edit 001 --title="New title"
  backlog edit 001 --priority=urgent
  backlog edit 001 --add-label=blocked --remove-label=ready
  backlog edit 001 --description="Updated description"`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runEdit(args[0])
	},
}

func init() {
	rootCmd.AddCommand(editCmd)

	editCmd.Flags().StringVarP(&editTitle, "title", "t", "", "New title for the task")
	editCmd.Flags().StringVarP(&editPriority, "priority", "p", "", "New priority: urgent, high, medium, low, none")
	editCmd.Flags().StringVarP(&editDescription, "description", "d", "", "New description for the task")
	editCmd.Flags().StringSliceVar(&editAddLabels, "add-label", nil, "Labels to add (can be specified multiple times)")
	editCmd.Flags().StringSliceVar(&editRemoveLabel, "remove-label", nil, "Labels to remove (can be specified multiple times)")
}

func runEdit(id string) error {
	// Check if any changes were specified
	if editTitle == "" && editPriority == "" && editDescription == "" &&
		len(editAddLabels) == 0 && len(editRemoveLabel) == 0 {
		return fmt.Errorf("no changes specified")
	}

	// Validate priority if specified
	var priority *backend.Priority
	if editPriority != "" {
		p := backend.Priority(editPriority)
		if !p.IsValid() {
			return InvalidInputError(fmt.Sprintf("invalid priority %q (valid: urgent, high, medium, low, none)", editPriority))
		}
		priority = &p
	}

	// Get backend and connect
	b, _, cleanup, err := connectBackend()
	if err != nil {
		return err
	}
	defer cleanup()

	// Build the changes struct
	changes := backend.TaskChanges{
		Priority:     priority,
		AddLabels:    editAddLabels,
		RemoveLabels: editRemoveLabel,
	}

	if editTitle != "" {
		changes.Title = &editTitle
	}

	if editDescription != "" {
		changes.Description = &editDescription
	}

	// Update the task
	task, err := b.Update(id, changes)
	if err != nil {
		// Check if this is a "not found" error (case-insensitive check for 404/Not Found)
		errLower := strings.ToLower(err.Error())
		if strings.Contains(errLower, "not found") || strings.Contains(errLower, "404") {
			return NotFoundError(err.Error())
		}
		return err
	}

	// Output the result
	formatter := output.New(output.Format(GetFormat()))
	return formatter.FormatUpdated(os.Stdout, task)
}
