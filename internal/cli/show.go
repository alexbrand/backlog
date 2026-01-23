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
	showComments bool
)

var showCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Display full task details",
	Long: `Display the full details of a task including its description.

Use the --comments flag to include the comment thread.

Examples:
  backlog show 001
  backlog show 001 -f json
  backlog show 001 --comments`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runShow(args[0])
	},
}

func init() {
	rootCmd.AddCommand(showCmd)

	showCmd.Flags().BoolVar(&showComments, "comments", false, "Include comment thread")
}

func runShow(id string) error {
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

	// Get the task
	task, err := b.Get(id)
	if err != nil {
		// Check if this is a "not found" error
		if strings.Contains(err.Error(), "not found") {
			return NotFoundError(err.Error())
		}
		return err
	}

	// Output the task
	formatter := output.New(output.Format(GetFormat()))
	if err := formatter.FormatTask(os.Stdout, task); err != nil {
		return err
	}

	// If comments requested, fetch and display them
	if showComments {
		comments, err := b.ListComments(id)
		if err != nil {
			return fmt.Errorf("failed to list comments: %w", err)
		}

		fmt.Fprintln(os.Stdout)
		if err := formatter.FormatComments(os.Stdout, comments); err != nil {
			return err
		}
	}

	return nil
}
