package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Version information (set at build time via ldflags)
var (
	Version   = "0.1.0-dev"
	GitCommit = "unknown"
	BuildDate = "unknown"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Long:  `Print the version, git commit, and build date of the backlog CLI.`,
	Run: func(cmd *cobra.Command, args []string) {
		if verbose {
			fmt.Printf("backlog version %s\n", Version)
			fmt.Printf("  git commit: %s\n", GitCommit)
			fmt.Printf("  build date: %s\n", BuildDate)
		} else {
			fmt.Printf("backlog version %s\n", Version)
		}
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
