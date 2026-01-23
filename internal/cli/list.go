package cli

import (
	"fmt"
	"os"

	"github.com/alexbrand/backlog/internal/backend"
	"github.com/alexbrand/backlog/internal/config"
	"github.com/alexbrand/backlog/internal/local"
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
	// Validate and parse statuses
	var statusFilters []backend.Status
	for _, s := range listStatus {
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
		IncludeDone: listIncludeDone,
	}

	// Get backend and configuration
	var b backend.Backend
	var backendCfg backend.Config

	// Try to get workspace from config
	ws, _, err := config.GetWorkspace(GetWorkspace())
	if err == nil {
		// Have config - use it
		b, err = backend.Get(ws.Backend)
		if err != nil {
			return err
		}

		cfg := config.Get()
		backendCfg = backend.Config{
			AgentID:          cfg.Defaults.AgentID,
			AgentLabelPrefix: ws.AgentLabelPrefix,
		}

		switch ws.Backend {
		case "local":
			path := ws.Path
			if path == "" {
				path = ".backlog"
			}
			backendCfg.Workspace = &local.WorkspaceConfig{
				Path:     path,
				LockMode: local.LockMode(ws.LockMode),
			}
		default:
			return fmt.Errorf("unsupported backend: %s", ws.Backend)
		}
	} else {
		// No config - check for local .backlog directory
		if _, statErr := os.Stat(".backlog"); statErr == nil {
			// Local .backlog directory exists - use local backend
			b, err = backend.Get("local")
			if err != nil {
				return err
			}
			backendCfg = backend.Config{
				Workspace: &local.WorkspaceConfig{Path: ".backlog"},
			}
		} else {
			// No config and no local .backlog directory
			return err
		}
	}

	if err := b.Connect(backendCfg); err != nil {
		return fmt.Errorf("failed to connect to backend: %w", err)
	}
	defer b.Disconnect()

	// List tasks
	taskList, err := b.List(filters)
	if err != nil {
		return fmt.Errorf("failed to list tasks: %w", err)
	}

	// Output the result
	formatter := output.New(output.Format(GetFormat()))
	return formatter.FormatTaskList(os.Stdout, taskList)
}
