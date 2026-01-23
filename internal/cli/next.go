package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/alexbrand/backlog/internal/backend"
	"github.com/alexbrand/backlog/internal/config"
	"github.com/alexbrand/backlog/internal/local"
	"github.com/alexbrand/backlog/internal/output"
	"github.com/spf13/cobra"
)

var (
	nextClaim  bool
	nextLabels []string
)

var nextCmd = &cobra.Command{
	Use:   "next",
	Short: "Get the next recommended task to work on",
	Long: `Get the next recommended task to work on.

Returns the highest priority unclaimed task from the backlog. Useful for agents
that need to pick up the next available work item.

By default, considers tasks with status 'todo' or 'backlog' that have no assignee.
Tasks are sorted by priority (urgent > high > medium > low > none).

Use --claim to atomically claim the task, preventing other agents from working on it.

Examples:
  backlog next                    # get highest priority unassigned task
  backlog next --label=backend    # filter by label
  backlog next --claim            # get and claim the task
  backlog next --claim -f json    # claim and output as JSON`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runNext()
	},
}

func init() {
	rootCmd.AddCommand(nextCmd)

	nextCmd.Flags().BoolVar(&nextClaim, "claim", false, "Atomically claim the task after finding it")
	nextCmd.Flags().StringSliceVarP(&nextLabels, "label", "l", nil, "Filter by labels (task must have all specified labels)")
}

// priorityOrder maps priorities to numeric order for sorting (lower = higher priority)
var priorityOrder = map[backend.Priority]int{
	backend.PriorityUrgent: 0,
	backend.PriorityHigh:   1,
	backend.PriorityMedium: 2,
	backend.PriorityLow:    3,
	backend.PriorityNone:   4,
}

func runNext() error {
	// Build filters to find unclaimed tasks
	filters := backend.TaskFilters{
		Status:      []backend.Status{backend.StatusTodo, backend.StatusBacklog},
		Assignee:    "unassigned",
		Labels:      nextLabels,
		IncludeDone: false,
	}

	// Get backend and configuration
	var b backend.Backend
	var backendCfg backend.Config
	var ws *config.Workspace

	// Try to get workspace from config
	workspace, _, err := config.GetWorkspace(GetWorkspace())
	if err == nil {
		ws = workspace
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

	// If no tasks found, return success with no output
	// This allows agents to check for available work without error handling
	if taskList.Count == 0 {
		return nil
	}

	// Find the highest priority task
	nextTask := findHighestPriorityTask(taskList.Tasks)
	if nextTask == nil {
		return nil
	}

	formatter := output.New(output.Format(GetFormat()))

	// If --claim flag is set, claim the task
	if nextClaim {
		// Check if backend supports claiming
		claimer, ok := b.(backend.Claimer)
		if !ok {
			return fmt.Errorf("backend %q does not support task claiming", b.Name())
		}

		// Resolve agent ID
		resolvedAgentID := ResolveAgentID(ws)

		// Attempt to claim the task
		result, err := claimer.Claim(nextTask.ID, resolvedAgentID)
		if err != nil {
			// Check for conflict error (task already claimed by another agent)
			if _, isConflict := err.(*local.ClaimConflictError); isConflict {
				return ConflictError(err.Error())
			}
			// Check for not found error
			if strings.Contains(err.Error(), "not found") {
				return NotFoundError(err.Error())
			}
			return err
		}

		return formatter.FormatClaimed(os.Stdout, result.Task, resolvedAgentID, result.AlreadyOwned)
	}

	// Output the task without claiming
	return formatter.FormatTask(os.Stdout, nextTask)
}

// findHighestPriorityTask returns the task with the highest priority from the list.
// Among tasks with the same priority, the first one encountered is returned.
func findHighestPriorityTask(tasks []backend.Task) *backend.Task {
	if len(tasks) == 0 {
		return nil
	}

	highest := &tasks[0]
	highestOrder := priorityOrder[highest.Priority]

	for i := 1; i < len(tasks); i++ {
		order := priorityOrder[tasks[i].Priority]
		if order < highestOrder {
			highest = &tasks[i]
			highestOrder = order
		}
	}

	return highest
}
