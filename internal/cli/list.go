package cli

import (
	"fmt"
	"os"

	"github.com/alexbrand/backlog/internal/backend"
	"github.com/alexbrand/backlog/internal/output"
	"github.com/spf13/cobra"
)

var (
	listStatus      []string
	listPriority    []string
	listAssignee    string
	listLabels      []string
	listLimit       int
	listIncludeDone bool
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List tasks in the backlog",
	Long: `List tasks from the backlog with optional filtering.

By default, lists all non-done tasks. Use flags to filter by status,
priority, assignee, or labels.

Examples:
  backlog list                          # all non-done tasks
  backlog list --status=todo            # filter by status
  backlog list --assignee=@me           # my tasks
  backlog list --assignee=unassigned    # unclaimed tasks
  backlog list --priority=high,urgent   # multiple values
  backlog list --label=bug              # by label
  backlog list --limit=10               # pagination
  backlog list -f json                  # JSON output for agents
  backlog list --include-done           # include completed tasks`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runList()
	},
}

func init() {
	rootCmd.AddCommand(listCmd)

	listCmd.Flags().StringSliceVarP(&listStatus, "status", "s", nil, "Filter by status (can be specified multiple times or comma-separated)")
	listCmd.Flags().StringSliceVarP(&listPriority, "priority", "p", nil, "Filter by priority (can be specified multiple times or comma-separated)")
	listCmd.Flags().StringVarP(&listAssignee, "assignee", "a", "", "Filter by assignee (use @me for current user, unassigned for no assignee)")
	listCmd.Flags().StringSliceVarP(&listLabels, "label", "l", nil, "Filter by labels (task must have all specified labels)")
	listCmd.Flags().IntVar(&listLimit, "limit", 0, "Maximum number of tasks to return (0 for no limit)")
	listCmd.Flags().BoolVar(&listIncludeDone, "include-done", false, "Include tasks with done status")
}

func runList() error {
	// Output debug information if verbose mode is enabled
	if IsVerbose() {
		fmt.Fprintln(os.Stderr, "debug: listing tasks with filters")
	}

	// Validate and parse statuses
	var statusFilters []backend.Status
	includeDone := listIncludeDone
	for _, s := range listStatus {
		// Special handling for "all" which means all statuses including done
		if s == "all" {
			statusFilters = backend.ValidStatuses()
			includeDone = true
			break
		}
		status := backend.Status(s)
		if !status.IsValid() {
			return InvalidInputError(fmt.Sprintf("invalid status %q (valid: backlog, todo, in-progress, review, done)", s))
		}
		statusFilters = append(statusFilters, status)
	}

	// Validate and parse priorities
	var priorityFilters []backend.Priority
	for _, p := range listPriority {
		priority := backend.Priority(p)
		if !priority.IsValid() {
			return InvalidInputError(fmt.Sprintf("invalid priority %q (valid: urgent, high, medium, low, none)", p))
		}
		priorityFilters = append(priorityFilters, priority)
	}

	// Build filters
	filters := backend.TaskFilters{
		Status:      statusFilters,
		Priority:    priorityFilters,
		Assignee:    listAssignee,
		Labels:      listLabels,
		Limit:       listLimit,
		IncludeDone: includeDone,
	}

	// Get backend and connect
	b, _, cleanup, err := connectBackend()
	if err != nil {
		return err
	}
	defer cleanup()

	// List tasks
	taskList, err := b.List(filters)
	if err != nil {
		return WrapError("failed to list tasks", err)
	}

	// Output the result
	formatter := output.New(output.Format(GetFormat()))
	return formatter.FormatTaskList(os.Stdout, taskList)
}
