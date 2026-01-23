package cli

import (
	"github.com/spf13/cobra"
)

var (
	// Global flags
	workspace string
	format    string
	quiet     bool
	verbose   bool
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "backlog",
	Short: "A CLI tool for managing tasks across multiple issue tracking backends",
	Long: `backlog is a command-line tool for managing tasks across multiple issue
tracking backends. It provides a unified, agent-friendly interface that
abstracts away provider-specific APIs, enabling both humans and AI agents
to manage backlogs through simple, composable commands.`,
}

// Execute runs the CLI application.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Global flags available to all commands
	rootCmd.PersistentFlags().StringVarP(&workspace, "workspace", "w", "", "Target workspace (default: workspace with default: true)")
	rootCmd.PersistentFlags().StringVarP(&format, "format", "f", "table", "Output format: table, json, plain, id-only")
	rootCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "Suppress non-essential output")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Show debug information")
}
