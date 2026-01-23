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

var commentBodyFile string

var commentCmd = &cobra.Command{
	Use:   "comment <id> <message>",
	Short: "Add a comment to a task",
	Long: `Add a comment to a task.

The comment is attributed to the current agent (resolved via --agent-id, BACKLOG_AGENT_ID,
workspace config, or hostname fallback).

Examples:
  backlog comment 001 "Found the bug, working on fix"
  backlog comment 001 "Starting work on implementation" -f json
  backlog comment 001 --body-file=./analysis.md`,
	Args: func(cmd *cobra.Command, args []string) error {
		// With --body-file, we only need the ID
		if commentBodyFile != "" {
			if len(args) != 1 {
				return fmt.Errorf("requires exactly 1 argument (task ID) when using --body-file")
			}
			return nil
		}
		// Without --body-file, we need both ID and message
		if len(args) != 2 {
			return fmt.Errorf("requires exactly 2 arguments: <id> <message>")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]
		var message string

		if commentBodyFile != "" {
			// Read message from file
			content, err := os.ReadFile(commentBodyFile)
			if err != nil {
				return fmt.Errorf("failed to read body file: %w", err)
			}
			message = string(content)
		} else {
			message = args[1]
		}

		return runComment(id, message)
	},
}

func init() {
	commentCmd.Flags().StringVar(&commentBodyFile, "body-file", "", "Read comment body from file")
	rootCmd.AddCommand(commentCmd)
}

func runComment(id string, message string) error {
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

	// Set agent ID in config for comment attribution
	resolvedAgentID := ResolveAgentID(ws)
	backendCfg.AgentID = resolvedAgentID

	if err := b.Connect(backendCfg); err != nil {
		return fmt.Errorf("failed to connect to backend: %w", err)
	}
	defer b.Disconnect()

	// Add the comment
	comment, err := b.AddComment(id, message)
	if err != nil {
		// Check for not found error
		if strings.Contains(err.Error(), "not found") {
			return NotFoundError(err.Error())
		}
		return err
	}

	// Output the result
	formatter := output.New(output.Format(GetFormat()))
	return formatter.FormatComment(os.Stdout, comment)
}
