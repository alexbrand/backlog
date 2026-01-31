package cli

import (
	"fmt"
	"os"

	"github.com/alexbrand/backlog/internal/backend"
	"github.com/alexbrand/backlog/internal/output"
	"github.com/spf13/cobra"
)

var (
	addPriority    string
	addLabels      []string
	addDescription string
	addBodyFile    string
	addStatus      string
	addBlocks      []string
	addBlockedBy   []string
)

var addCmd = &cobra.Command{
	Use:   "add <title>",
	Short: "Create a new task",
	Long: `Create a new task in the backlog.

The title is required and provided as the first argument. Additional fields
can be set using flags.

Examples:
  backlog add "Implement rate limiting"
  backlog add "Fix login bug" --priority=urgent --label=bug
  backlog add "Refactor API" --description="Split into modules" --status=todo
  backlog add "Research caching" --body-file=./task-details.md`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runAdd(args[0])
	},
}

func init() {
	rootCmd.AddCommand(addCmd)

	addCmd.Flags().StringVarP(&addPriority, "priority", "p", "", "Priority: urgent, high, medium, low, none (default: none)")
	addCmd.Flags().StringSliceVarP(&addLabels, "label", "l", nil, "Add labels (can be specified multiple times)")
	addCmd.Flags().StringVarP(&addDescription, "description", "d", "", "Task description")
	addCmd.Flags().StringVar(&addBodyFile, "body-file", "", "Read description from file")
	addCmd.Flags().StringVarP(&addStatus, "status", "s", "", "Initial status: backlog, todo, in-progress, review, done (default: backlog)")
	addCmd.Flags().StringSliceVar(&addBlocks, "blocks", nil, "Task IDs that this task blocks")
	addCmd.Flags().StringSliceVar(&addBlockedBy, "blocked-by", nil, "Task IDs that block this task")
}

func runAdd(title string) error {
	// Validate title
	if title == "" {
		return fmt.Errorf("title is required")
	}

	// Handle description from file
	description := addDescription
	if addBodyFile != "" {
		content, err := os.ReadFile(addBodyFile)
		if err != nil {
			return fmt.Errorf("failed to read body file: %w", err)
		}
		description = string(content)
	}

	// Validate and parse priority
	var priority backend.Priority
	if addPriority != "" {
		priority = backend.Priority(addPriority)
		if !priority.IsValid() {
			return InvalidInputError(fmt.Sprintf("invalid priority %q (valid: urgent, high, medium, low, none)", addPriority))
		}
	}

	// Validate and parse status
	var status backend.Status
	if addStatus != "" {
		status = backend.Status(addStatus)
		if !status.IsValid() {
			return InvalidInputError(fmt.Sprintf("invalid status %q (valid: backlog, todo, in-progress, review, done)", addStatus))
		}
	}

	// Get backend and connect
	b, _, cleanup, err := connectBackend()
	if err != nil {
		return err
	}
	defer cleanup()

	// Create the task
	input := backend.TaskInput{
		Title:       title,
		Description: description,
		Status:      status,
		Priority:    priority,
		Labels:      addLabels,
	}

	task, err := b.Create(input)
	if err != nil {
		return fmt.Errorf("failed to create task: %w", err)
	}

	// Create dependency links if specified
	if len(addBlocks) > 0 || len(addBlockedBy) > 0 {
		relater, ok := b.(backend.Relater)
		if !ok {
			return fmt.Errorf("backend %q does not support task dependencies", b.Name())
		}
		for _, targetID := range addBlocks {
			if _, err := relater.Link(task.ID, targetID, backend.RelationBlocks); err != nil {
				return fmt.Errorf("failed to link %s --blocks %s: %w", task.ID, targetID, err)
			}
		}
		for _, targetID := range addBlockedBy {
			if _, err := relater.Link(task.ID, targetID, backend.RelationBlockedBy); err != nil {
				return fmt.Errorf("failed to link %s --blocked-by %s: %w", task.ID, targetID, err)
			}
		}
	}

	// Output the result (unless quiet mode is enabled)
	if IsQuiet() {
		return nil
	}
	formatter := output.New(output.Format(GetFormat()))
	return formatter.FormatCreated(os.Stdout, task)
}
