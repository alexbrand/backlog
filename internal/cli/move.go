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

var moveCmd = &cobra.Command{
	Use:   "move <id> <status>",
	Short: "Transition a task to a new status",
	Long: `Move a task to a new status.

Valid statuses: backlog, todo, in-progress, review, done

Examples:
  backlog move 001 in-progress
  backlog move 001 done
  backlog move 001 review -f json`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runMove(args[0], args[1])
	},
}

func init() {
	rootCmd.AddCommand(moveCmd)
}

func runMove(id, statusStr string) error {
	// Validate status
	status := backend.Status(statusStr)
	if !status.IsValid() {
		return InvalidInputError(fmt.Sprintf("invalid status %q (valid: backlog, todo, in-progress, review, done)", statusStr))
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
			backendCfg.Workspace = &local.WorkspaceConfig{Path: path}
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

	// Get the current task first to capture old status
	currentTask, err := b.Get(id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return NotFoundError(err.Error())
		}
		return err
	}

	oldStatus := currentTask.Status

	// Move the task
	task, err := b.Move(id, status)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return NotFoundError(err.Error())
		}
		return err
	}

	// Output the result
	formatter := output.New(output.Format(GetFormat()))
	return formatter.FormatMoved(os.Stdout, task, oldStatus, status)
}
