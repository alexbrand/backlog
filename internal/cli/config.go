package cli

import (
	"fmt"
	"os"

	"github.com/alexbrand/backlog/internal/config"
	"github.com/alexbrand/backlog/internal/output"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage configuration",
	Long:  `Manage backlog configuration settings.`,
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Display current configuration",
	Long:  `Display the current configuration in YAML format.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runConfigShow()
	},
}

var configHealthCmd = &cobra.Command{
	Use:   "health",
	Short: "Check backend health status",
	Long:  `Check the health status of the configured backend.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runConfigHealth()
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configHealthCmd)
}

func runConfigShow() error {
	cfg := config.Get()
	if cfg == nil {
		return ConfigError("no configuration loaded")
	}

	format := GetFormat()
	if format == "json" {
		// Output as JSON when requested
		formatter := output.New(output.FormatJSON)
		return formatter.FormatConfig(os.Stdout, cfg)
	}

	// Marshal config to YAML for display
	outputYAML, err := yaml.Marshal(cfg)
	if err != nil {
		return WrapExitCodeError(ExitError, "failed to format configuration", err)
	}

	fmt.Print(string(outputYAML))
	return nil
}

func runConfigHealth() error {
	// Get backend and connect
	b, ws, cleanup, err := connectBackend()
	if err != nil {
		return err
	}
	defer cleanup()

	// Perform health check
	status, err := b.HealthCheck()
	if err != nil {
		return WrapError("health check failed", err)
	}

	format := GetFormat()
	if format == "json" {
		formatter := output.New(output.FormatJSON)
		return formatter.FormatHealthCheck(os.Stdout, b.Name(), ws, &status)
	}

	// Output health status
	if status.OK {
		fmt.Printf("%s: healthy (%v)\n", b.Name(), status.Latency)
		if ws.Project > 0 {
			fmt.Printf("project: %d\n", ws.Project)
		}
	} else {
		fmt.Printf("%s: unhealthy - %s\n", b.Name(), status.Message)
		return WrapExitCodeError(ExitError, status.Message, nil)
	}

	return nil
}
