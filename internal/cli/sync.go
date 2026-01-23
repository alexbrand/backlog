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

var syncForce bool

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync local backlog with remote git repository",
	Long: `Sync local backlog with a remote git repository.

The sync operation:
1. Pulls changes from the remote repository (git pull)
2. Pushes local changes to the remote repository (git push)

This command requires git_sync to be enabled in your workspace configuration.

Use --force to force push/pull even if there are conflicts.

Examples:
  backlog sync
  backlog sync --force
  backlog sync -f json`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runSync(syncForce)
	},
}

func init() {
	syncCmd.Flags().BoolVar(&syncForce, "force", false, "Force sync even if there are conflicts")
	rootCmd.AddCommand(syncCmd)
}

func runSync(force bool) error {
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

	// Check if backend supports syncing
	syncer, ok := b.(backend.Syncer)
	if !ok {
		return fmt.Errorf("backend %q does not support sync operations", b.Name())
	}

	// Perform the sync
	result, err := syncer.Sync(force)
	if err != nil {
		// Check if it's a conflict error (exit code 2)
		if _, ok := err.(*local.SyncConflictError); ok {
			return ConflictError(err.Error())
		}
		return err
	}

	// Output the result
	formatter := output.New(output.Format(GetFormat()))
	return formatter.FormatSynced(os.Stdout, result)
}
