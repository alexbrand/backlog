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

var claimCmd = &cobra.Command{
	Use:   "claim <id>",
	Short: "Claim a task for the current agent",
	Long: `Claim a task for the current agent. This is an atomic operation for multi-agent coordination.

The claim operation:
1. Acquires a lock on the task
2. Adds an agent label (e.g., agent:claude-1)
3. Moves the task to in-progress status

If the task is already claimed by the same agent, this is a no-op and returns success.
If the task is already claimed by a different agent, returns exit code 2 (conflict).

Examples:
  backlog claim 001
  backlog claim 001 --agent-id=claude-2
  backlog claim 001 -f json`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runClaim(args[0])
	},
}

func init() {
	rootCmd.AddCommand(claimCmd)
}

func runClaim(id string) error {
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
				GitSync:  ws.GitSync,
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

	// Check if backend supports claiming
	claimer, ok := b.(backend.Claimer)
	if !ok {
		return fmt.Errorf("backend %q does not support task claiming", b.Name())
	}

	// Resolve agent ID
	resolvedAgentID := ResolveAgentID(ws)

	// Attempt to claim the task
	result, err := claimer.Claim(id, resolvedAgentID)
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

	// Output the result
	formatter := output.New(output.Format(GetFormat()))
	return formatter.FormatClaimed(os.Stdout, result.Task, resolvedAgentID, result.AlreadyOwned)
}
