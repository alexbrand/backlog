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

var releaseCmd = &cobra.Command{
	Use:   "release <id>",
	Short: "Release a claimed task back to todo",
	Long: `Release a claimed task back to todo status.

The release operation:
1. Removes the lock on the task
2. Removes the agent label (e.g., agent:claude-1)
3. Unassigns the task
4. Moves the task to todo status

Use this when an agent cannot complete work on a task and wants to make it
available for other agents.

Examples:
  backlog release 001
  backlog release 001 -f json`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runRelease(args[0])
	},
}

func init() {
	rootCmd.AddCommand(releaseCmd)
}

func runRelease(id string) error {
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

	// Check if backend supports releasing
	claimer, ok := b.(backend.Claimer)
	if !ok {
		return fmt.Errorf("backend %q does not support task releasing", b.Name())
	}

	// Get the task first so we can display it in the output
	task, err := b.Get(id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return NotFoundError(err.Error())
		}
		return err
	}

	// Release the task
	if err := claimer.Release(id); err != nil {
		if strings.Contains(err.Error(), "not found") {
			return NotFoundError(err.Error())
		}
		return err
	}

	// Get the updated task for output
	updatedTask, err := b.Get(id)
	if err != nil {
		// If we can't get the updated task, use the original with updated status
		task.Status = backend.StatusTodo
		updatedTask = task
	}

	// Output the result
	formatter := output.New(output.Format(GetFormat()))
	return formatter.FormatReleased(os.Stdout, updatedTask)
}
