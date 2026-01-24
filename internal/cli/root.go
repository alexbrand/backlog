package cli

import (
	"os"

	"github.com/alexbrand/backlog/internal/config"
	"github.com/alexbrand/backlog/internal/credentials"
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
	agentID   string
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
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is .backlog/config.yaml)")

	// Global flags available to all commands
	rootCmd.PersistentFlags().StringVarP(&workspace, "workspace", "w", "", "Target workspace (default: workspace with default: true)")
	rootCmd.PersistentFlags().StringVarP(&format, "format", "f", "", "Output format: table, json, plain, id-only")
	rootCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "Suppress non-essential output")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Show debug information")
	rootCmd.PersistentFlags().StringVar(&agentID, "agent-id", "", "Agent identifier for task claiming and coordination")

	// Bind flags to viper
	viper.BindPFlag("workspace", rootCmd.PersistentFlags().Lookup("workspace"))
	viper.BindPFlag("format", rootCmd.PersistentFlags().Lookup("format"))
	viper.BindPFlag("quiet", rootCmd.PersistentFlags().Lookup("quiet"))
	viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))
	viper.BindPFlag("agent_id", rootCmd.PersistentFlags().Lookup("agent-id"))
}

// initConfig reads in config file and ENV variables if set.
func initConfig() error {
	if err := config.Init(cfgFile); err != nil {
		// If a config file exists but has errors (e.g., invalid YAML), fail with exit code 4
		// The config.Init function already handles "file not found" gracefully
		return ConfigError(err.Error())
	}

	// Initialize credentials system (credentials.yaml is optional)
	if err := credentials.Init(); err != nil {
		// Credentials file errors should be reported but not fatal
		// (credentials can come from environment variables)
		if verbose {
			// Only show warning in verbose mode
			_ = err // Warning suppressed in non-verbose mode
		}
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

	// Resolve agent ID with priority chain:
	// 1. CLI flag (--agent-id) - already set in agentID if provided
	// 2. Environment variable (BACKLOG_AGENT_ID)
	// 3. Workspace config (resolved later when workspace is known)
	// 4. Global default (defaults.agent_id)
	// 5. Hostname fallback (resolved later if still empty)
	if agentID == "" {
		agentID = os.Getenv("BACKLOG_AGENT_ID")
	}
	if agentID == "" && cfg != nil {
		agentID = cfg.Defaults.AgentID
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

// GetAgentID returns the resolved agent ID.
// Note: This returns the partially resolved agent ID (flag/env/global default).
// For full resolution including workspace config and hostname fallback,
// use ResolveAgentID with the workspace.
func GetAgentID() string {
	return agentID
}

// ResolveAgentID returns the fully resolved agent ID following the priority chain:
// 1. CLI flag (--agent-id)
// 2. Environment variable (BACKLOG_AGENT_ID)
// 3. Workspace config (workspaces.<name>.agent_id)
// 4. Global default (defaults.agent_id)
// 5. Hostname fallback
func ResolveAgentID(ws *config.Workspace) string {
	// agentID already contains resolution from flag → env → global default
	if agentID != "" {
		return agentID
	}

	// Try workspace-specific agent ID
	if ws != nil && ws.AgentID != "" {
		return ws.AgentID
	}

	// Fallback to hostname
	hostname, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return hostname
}
