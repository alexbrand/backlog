package cli

import (
	"fmt"

	"github.com/alexbrand/backlog/internal/config"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage configuration",
	Long:  `Manage backlog configuration settings and workspaces.`,
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Display current configuration",
	Long:  `Display the current configuration in YAML format.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runConfigShow()
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configShowCmd)
}

func runConfigShow() error {
	cfg := config.Get()
	if cfg == nil {
		return ConfigError("no configuration loaded")
	}

	// Marshal config to YAML for display
	output, err := yaml.Marshal(cfg)
	if err != nil {
		return WrapExitCodeError(ExitError, "failed to format configuration", err)
	}

	fmt.Print(string(output))
	return nil
}
