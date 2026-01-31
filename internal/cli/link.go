package cli

import (
	"fmt"
	"os"

	"github.com/alexbrand/backlog/internal/backend"
	"github.com/alexbrand/backlog/internal/output"
	"github.com/spf13/cobra"
)

var (
	linkBlocks    string
	linkBlockedBy string
)

var linkCmd = &cobra.Command{
	Use:   "link <source-id>",
	Short: "Create a dependency between two tasks",
	Long: `Create a dependency relationship between two tasks.

Exactly one of --blocks or --blocked-by must be specified.

Examples:
  backlog link 001 --blocks 002       # 001 blocks 002
  backlog link 001 --blocked-by 002   # 001 is blocked by 002`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runLink(args[0])
	},
}

func init() {
	rootCmd.AddCommand(linkCmd)

	linkCmd.Flags().StringVar(&linkBlocks, "blocks", "", "Target task ID that source blocks")
	linkCmd.Flags().StringVar(&linkBlockedBy, "blocked-by", "", "Target task ID that blocks source")
}

func runLink(sourceID string) error {
	// Validate exactly one flag is set
	if linkBlocks == "" && linkBlockedBy == "" {
		return InvalidInputError("one of --blocks or --blocked-by is required")
	}
	if linkBlocks != "" && linkBlockedBy != "" {
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
	if linkBlocks != "" {
		relationType = backend.RelationBlocks
		targetID = linkBlocks
	} else {
		relationType = backend.RelationBlockedBy
		targetID = linkBlockedBy
	}

	relation, err := relater.Link(sourceID, targetID, relationType)
	if err != nil {
		return err
	}

	formatter := output.New(output.Format(GetFormat()))
	return formatter.FormatLinked(os.Stdout, relation, sourceID)
}
