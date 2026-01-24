package cli

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/alexbrand/backlog/internal/config"
	"github.com/alexbrand/backlog/internal/credentials"
	"github.com/alexbrand/backlog/internal/output"
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

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Interactive configuration setup",
	Long: `Interactively set up backlog configuration.

This command guides you through configuring a workspace with prompts for:
- Backend type (github, local)
- Backend-specific settings (repo, path, etc.)
- Default workspace settings

The configuration is saved to ~/.config/backlog/config.yaml.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runConfigInit()
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
	configCmd.AddCommand(configInitCmd)
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

func runConfigInit() error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("Backlog Configuration Setup")
	fmt.Println("============================")
	fmt.Println()

	// Get workspace name
	fmt.Print("Workspace name [main]: ")
	workspaceName, _ := reader.ReadString('\n')
	workspaceName = strings.TrimSpace(workspaceName)
	if workspaceName == "" {
		workspaceName = "main"
	}

	// Choose backend
	fmt.Println()
	fmt.Println("Available backends:")
	fmt.Println("  1. github - GitHub Issues")
	fmt.Println("  2. local  - Local filesystem")
	fmt.Print("Choose backend [1]: ")
	backendChoice, _ := reader.ReadString('\n')
	backendChoice = strings.TrimSpace(backendChoice)

	var backendType string
	var workspaceConfig map[string]any

	switch backendChoice {
	case "", "1", "github":
		backendType = "github"
		workspaceConfig, _ = configureGitHubBackend(reader)
	case "2", "local":
		backendType = "local"
		workspaceConfig, _ = configureLocalBackend(reader)
	default:
		return ConfigError(fmt.Sprintf("unknown backend choice: %s", backendChoice))
	}

	workspaceConfig["backend"] = backendType
	workspaceConfig["default"] = true

	// Build the config structure
	cfg := map[string]any{
		"version": 1,
		"defaults": map[string]any{
			"format":    "table",
			"workspace": workspaceName,
		},
		"workspaces": map[string]any{
			workspaceName: workspaceConfig,
		},
	}

	// Determine config path - prefer project-local if it exists
	configPath := ".backlog/config.yaml"
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// No project-local config, use global config
		configPath, err = config.DefaultConfigPath()
		if err != nil {
			return WrapExitCodeError(ExitConfigError, "failed to determine config path", err)
		}
	}

	// Create config directory if it doesn't exist
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return WrapExitCodeError(ExitConfigError, "failed to create config directory", err)
	}

	// Check if config file already exists
	if _, err := os.Stat(configPath); err == nil {
		fmt.Printf("\nConfiguration file already exists at %s\n", configPath)
		fmt.Print("Overwrite? [y/N]: ")
		confirm, _ := reader.ReadString('\n')
		confirm = strings.TrimSpace(strings.ToLower(confirm))
		if confirm != "y" && confirm != "yes" {
			fmt.Println("Aborted.")
			return nil
		}
	}

	// Marshal and write config
	output, err := yaml.Marshal(cfg)
	if err != nil {
		return WrapExitCodeError(ExitError, "failed to format configuration", err)
	}

	if err := os.WriteFile(configPath, output, 0644); err != nil {
		return WrapExitCodeError(ExitConfigError, "failed to write configuration", err)
	}

	fmt.Printf("\nConfiguration saved to %s\n", configPath)
	return nil
}

func configureGitHubBackend(reader *bufio.Reader) (map[string]any, error) {
	config := make(map[string]any)

	fmt.Println()
	fmt.Println("GitHub Backend Configuration")
	fmt.Println("----------------------------")

	// Get repository
	fmt.Print("Repository (owner/repo): ")
	repo, _ := reader.ReadString('\n')
	repo = strings.TrimSpace(repo)
	if repo == "" {
		return nil, fmt.Errorf("repository is required")
	}
	config["repo"] = repo

	// Check for GitHub token (env var or credentials file)
	_, err := credentials.GetGitHubToken()
	if err != nil {
		fmt.Println()
		fmt.Println("GitHub token not found.")
		fmt.Println("Options:")
		fmt.Println("  1. Enter a token now (will be saved to credentials.yaml)")
		fmt.Println("  2. Set GITHUB_TOKEN environment variable later")
		fmt.Print("Enter GitHub token (or press Enter to skip): ")
		inputToken, _ := reader.ReadString('\n')
		inputToken = strings.TrimSpace(inputToken)
		if inputToken != "" {
			if err := credentials.SaveGitHubToken(inputToken); err != nil {
				fmt.Printf("Warning: failed to save token: %v\n", err)
			} else {
				credPath, _ := credentials.DefaultCredentialsPath()
				fmt.Printf("Token saved to %s\n", credPath)
			}
		} else {
			fmt.Println("You will need to set GITHUB_TOKEN before using the GitHub backend:")
			fmt.Println("  export GITHUB_TOKEN=your_token")
		}
		fmt.Println()
	} else {
		// Check if it came from env var or credentials file
		if os.Getenv("GITHUB_TOKEN") != "" {
			fmt.Println("GitHub token found in GITHUB_TOKEN environment variable")
		} else {
			fmt.Println("GitHub token found in credentials.yaml")
		}
	}

	// Optional: agent settings
	fmt.Println()
	fmt.Print("Agent ID (leave empty for hostname): ")
	agentID, _ := reader.ReadString('\n')
	agentID = strings.TrimSpace(agentID)
	if agentID != "" {
		config["agent_id"] = agentID
	}

	fmt.Print("Agent label prefix [agent]: ")
	agentLabelPrefix, _ := reader.ReadString('\n')
	agentLabelPrefix = strings.TrimSpace(agentLabelPrefix)
	if agentLabelPrefix != "" && agentLabelPrefix != "agent" {
		config["agent_label_prefix"] = agentLabelPrefix
	}

	return config, nil
}

func configureLocalBackend(reader *bufio.Reader) (map[string]any, error) {
	config := make(map[string]any)

	fmt.Println()
	fmt.Println("Local Backend Configuration")
	fmt.Println("---------------------------")

	// Get path
	fmt.Print("Backlog path [.backlog]: ")
	path, _ := reader.ReadString('\n')
	path = strings.TrimSpace(path)
	if path == "" {
		path = ".backlog"
	}
	config["path"] = path

	// Lock mode
	fmt.Println()
	fmt.Println("Lock mode:")
	fmt.Println("  1. file - File-based locking (default, single machine)")
	fmt.Println("  2. git  - Git-based locking (distributed agents)")
	fmt.Print("Choose lock mode [1]: ")
	lockChoice, _ := reader.ReadString('\n')
	lockChoice = strings.TrimSpace(lockChoice)

	switch lockChoice {
	case "", "1", "file":
		config["lock_mode"] = "file"
	case "2", "git":
		config["lock_mode"] = "git"
		config["git_sync"] = true
	}

	// Optional: agent settings
	fmt.Println()
	fmt.Print("Agent ID (leave empty for hostname): ")
	agentID, _ := reader.ReadString('\n')
	agentID = strings.TrimSpace(agentID)
	if agentID != "" {
		config["agent_id"] = agentID
	}

	return config, nil
}
