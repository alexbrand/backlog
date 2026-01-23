package cli

import (
	"fmt"
	"os"

	"github.com/alexbrand/backlog/internal/backend"
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
	// Get backend and connect
	b, _, cleanup, err := connectBackend()
	if err != nil {
		return err
	}
	defer cleanup()

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
