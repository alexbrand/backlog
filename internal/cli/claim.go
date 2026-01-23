package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/alexbrand/backlog/internal/backend"
	"github.com/alexbrand/backlog/internal/github"
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
	// Get backend and connect
	b, ws, cleanup, err := connectBackend()
	if err != nil {
		return err
	}
	defer cleanup()

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
		if _, isLocalConflict := err.(*local.ClaimConflictError); isLocalConflict {
			return ConflictError(err.Error())
		}
		if _, isGitHubConflict := err.(*github.ClaimConflictError); isGitHubConflict {
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
