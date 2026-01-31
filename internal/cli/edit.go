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
	editBlocks      []string
	editBlockedBy   []string
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
	editCmd.Flags().StringSliceVar(&editBlocks, "blocks", nil, "Task IDs that this task blocks")
	editCmd.Flags().StringSliceVar(&editBlockedBy, "blocked-by", nil, "Task IDs that block this task")
}

func runEdit(id string) error {
	// Check if any changes were specified
	if editTitle == "" && editPriority == "" && editDescription == "" &&
		len(editAddLabels) == 0 && len(editRemoveLabel) == 0 &&
		len(editBlocks) == 0 && len(editBlockedBy) == 0 {
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

	// Only call Update if there are non-relation changes
	hasFieldChanges := editTitle != "" || editPriority != "" || editDescription != "" ||
		len(editAddLabels) > 0 || len(editRemoveLabel) > 0

	var task *backend.Task
	if hasFieldChanges {
		task, err = b.Update(id, changes)
		if err != nil {
			errLower := strings.ToLower(err.Error())
			if strings.Contains(errLower, "not found") || strings.Contains(errLower, "404") {
				return NotFoundError(err.Error())
			}
			return err
		}
	} else {
		// Still need to get the task for output
		task, err = b.Get(id)
		if err != nil {
			errLower := strings.ToLower(err.Error())
			if strings.Contains(errLower, "not found") || strings.Contains(errLower, "404") {
				return NotFoundError(err.Error())
			}
			return err
		}
	}

	// Create dependency links if specified
	if len(editBlocks) > 0 || len(editBlockedBy) > 0 {
		relater, ok := b.(backend.Relater)
		if !ok {
			return fmt.Errorf("backend %q does not support task dependencies", b.Name())
		}
		for _, targetID := range editBlocks {
			if _, err := relater.Link(id, targetID, backend.RelationBlocks); err != nil {
				return fmt.Errorf("failed to link %s --blocks %s: %w", id, targetID, err)
			}
		}
		for _, targetID := range editBlockedBy {
			if _, err := relater.Link(id, targetID, backend.RelationBlockedBy); err != nil {
				return fmt.Errorf("failed to link %s --blocked-by %s: %w", id, targetID, err)
			}
		}
	}

	// Output the result
	formatter := output.New(output.Format(GetFormat()))
	return formatter.FormatUpdated(os.Stdout, task)
}
