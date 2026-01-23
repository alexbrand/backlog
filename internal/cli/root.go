package cli

import (
	"github.com/alexbrand/backlog/internal/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	// Global flags
	cfgFile   string
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
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return initConfig()
	},
	// Silence Cobra's default error/usage printing - we handle it ourselves
	SilenceErrors: true,
	SilenceUsage:  true,
}

// Execute runs the CLI application.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Config file flag
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.config/backlog/config.yaml)")

	// Global flags available to all commands
	rootCmd.PersistentFlags().StringVarP(&workspace, "workspace", "w", "", "Target workspace (default: workspace with default: true)")
	rootCmd.PersistentFlags().StringVarP(&format, "format", "f", "", "Output format: table, json, plain, id-only")
	rootCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "Suppress non-essential output")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Show debug information")

	// Bind flags to viper
	viper.BindPFlag("workspace", rootCmd.PersistentFlags().Lookup("workspace"))
	viper.BindPFlag("format", rootCmd.PersistentFlags().Lookup("format"))
	viper.BindPFlag("quiet", rootCmd.PersistentFlags().Lookup("quiet"))
	viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))
}

// initConfig reads in config file and ENV variables if set.
func initConfig() error {
	if err := config.Init(cfgFile); err != nil {
		// If a config file exists but has errors (e.g., invalid YAML), fail with exit code 4
		// The config.Init function already handles "file not found" gracefully
		return ConfigError(err.Error())
	}

	// Apply config defaults to flags if not set via CLI
	cfg := config.Get()
	if cfg != nil {
		if format == "" && cfg.Defaults.Format != "" {
			format = cfg.Defaults.Format
		}
	}

	// Set default format if still empty
	if format == "" {
		format = "table"
	}

	return nil
}

// GetWorkspace returns the workspace flag value.
func GetWorkspace() string {
	return workspace
}

// GetFormat returns the format flag value.
func GetFormat() string {
	return format
}

// IsQuiet returns true if quiet mode is enabled.
func IsQuiet() bool {
	return quiet
}

// IsVerbose returns true if verbose mode is enabled.
func IsVerbose() bool {
	return verbose
}
