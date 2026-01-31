package cli

import (
	"fmt"
	"os"

	"github.com/alexbrand/backlog/internal/backend"
	"github.com/alexbrand/backlog/internal/output"
	"github.com/spf13/cobra"
)

var (
	unlinkBlocks    string
	unlinkBlockedBy string
)

var unlinkCmd = &cobra.Command{
	Use:   "unlink <source-id>",
	Short: "Remove a dependency between two tasks",
	Long: `Remove a dependency relationship between two tasks.

Exactly one of --blocks or --blocked-by must be specified.

Examples:
  backlog unlink 001 --blocks 002       # remove 001 blocks 002
  backlog unlink 001 --blocked-by 002   # remove 001 blocked by 002`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runUnlink(args[0])
	},
}

func init() {
	rootCmd.AddCommand(unlinkCmd)

	unlinkCmd.Flags().StringVar(&unlinkBlocks, "blocks", "", "Target task ID that source blocks")
	unlinkCmd.Flags().StringVar(&unlinkBlockedBy, "blocked-by", "", "Target task ID that blocks source")
}

func runUnlink(sourceID string) error {
	// Validate exactly one flag is set
	if unlinkBlocks == "" && unlinkBlockedBy == "" {
		return InvalidInputError("one of --blocks or --blocked-by is required")
	}
	if unlinkBlocks != "" && unlinkBlockedBy != "" {
		return InvalidInputError("only one of --blocks or --blocked-by can be specified")
	}

	// Get backend and connect
	b, _, cleanup, err := connectBackend()
	if err != nil {
		return err
	}
	defer cleanup()

	// Check if backend supports relations
	relater, ok := b.(backend.Relater)
	if !ok {
		return fmt.Errorf("backend %q does not support task dependencies", b.Name())
	}

	var relationType backend.RelationType
	var targetID string
	if unlinkBlocks != "" {
		relationType = backend.RelationBlocks
		targetID = unlinkBlocks
	} else {
		relationType = backend.RelationBlockedBy
		targetID = unlinkBlockedBy
	}

	if err := relater.Unlink(sourceID, targetID, relationType); err != nil {
		return err
	}

	formatter := output.New(output.Format(GetFormat()))
	return formatter.FormatUnlinked(os.Stdout, sourceID, targetID)
}
