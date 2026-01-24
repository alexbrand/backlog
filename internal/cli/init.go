package cli

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new backlog in the current directory",
	Long: `Initialize a new backlog in the current directory.

This command creates the .backlog/ directory structure and guides you through
configuring the backend (GitHub Issues or local filesystem).

Created structure:
  .backlog/           - Root backlog directory
  .backlog/backlog/   - Tasks in backlog status
  .backlog/todo/      - Tasks ready to work on
  .backlog/in-progress/ - Tasks currently being worked on
  .backlog/review/    - Tasks in review
  .backlog/done/      - Completed tasks
  .backlog/.locks/    - Lock files for agent coordination
  .backlog/config.yaml - Configuration file`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runInit()
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}

func runInit() error {
	backlogDir := ".backlog"

	// Check if .backlog already exists
	if _, err := os.Stat(backlogDir); err == nil {
		return fmt.Errorf("%s already exists", backlogDir)
	}

	reader := bufio.NewReader(os.Stdin)

	fmt.Println("Initializing backlog...")
	fmt.Println()

	// Choose backend
	fmt.Println("Backend:")
	fmt.Println("  1. GitHub Issues (sync with GitHub)")
	fmt.Println("  2. Local filesystem (standalone)")
	fmt.Print("Choose [1]: ")
	backendChoice, _ := reader.ReadString('\n')
	backendChoice = strings.TrimSpace(backendChoice)

	var backendType string
	var workspaceConfig map[string]any

	switch backendChoice {
	case "", "1", "github":
		backendType = "github"
		workspaceConfig = configureGitHubBackendInit(reader)
	case "2", "local":
		backendType = "local"
		workspaceConfig = configureLocalBackendInit(reader)
	default:
		return fmt.Errorf("unknown backend choice: %s", backendChoice)
	}

	workspaceConfig["backend"] = backendType
	workspaceConfig["default"] = true

	// Build the config structure
	cfg := map[string]any{
		"version": 1,
		"defaults": map[string]any{
			"format": "table",
		},
		"workspaces": map[string]any{
			"default": workspaceConfig,
		},
	}

	// Create directory structure
	statusDirs := []string{
		"backlog",
		"todo",
		"in-progress",
		"review",
		"done",
	}

	for _, dir := range statusDirs {
		path := filepath.Join(backlogDir, dir)
		if err := os.MkdirAll(path, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", path, err)
		}
	}

	// Create .locks directory
	locksDir := filepath.Join(backlogDir, ".locks")
	if err := os.MkdirAll(locksDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", locksDir, err)
	}

	// Write config.yaml
	configPath := filepath.Join(backlogDir, "config.yaml")
	output, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to format configuration: %w", err)
	}

	if err := os.WriteFile(configPath, output, 0644); err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}

	fmt.Println()
	fmt.Println("Created .backlog/")
	fmt.Println("  - backlog/")
	fmt.Println("  - todo/")
	fmt.Println("  - in-progress/")
	fmt.Println("  - review/")
	fmt.Println("  - done/")
	fmt.Println("  - config.yaml")
	fmt.Println()
	fmt.Println("Ready! Try: backlog add \"My first task\"")

	return nil
}

func configureGitHubBackendInit(reader *bufio.Reader) map[string]any {
	config := make(map[string]any)

	fmt.Println()
	fmt.Println("GitHub Backend Setup:")

	// Get repository
	fmt.Print("  Repository (owner/repo): ")
	repo, _ := reader.ReadString('\n')
	repo = strings.TrimSpace(repo)
	if repo != "" {
		config["repo"] = repo
	}

	// Check for GitHub token
	if os.Getenv("GITHUB_TOKEN") == "" {
		fmt.Println()
		fmt.Println("  Note: Set GITHUB_TOKEN environment variable or add token to")
		fmt.Println("  ~/.config/backlog/credentials.yaml")
	}

	// Optional: Agent ID
	fmt.Print("  Agent ID (optional, press Enter to skip): ")
	agentID, _ := reader.ReadString('\n')
	agentID = strings.TrimSpace(agentID)
	if agentID != "" {
		config["agent_id"] = agentID
	}

	return config
}

func configureLocalBackendInit(reader *bufio.Reader) map[string]any {
	config := make(map[string]any)

	fmt.Println()
	fmt.Println("Local Backend Setup:")

	// Path defaults to .backlog
	config["path"] = "./.backlog"

	// Lock mode
	fmt.Println("  Lock mode:")
	fmt.Println("    1. File-based (single machine)")
	fmt.Println("    2. Git-based (distributed agents)")
	fmt.Print("  Choose [1]: ")
	lockChoice, _ := reader.ReadString('\n')
	lockChoice = strings.TrimSpace(lockChoice)

	switch lockChoice {
	case "", "1":
		config["lock_mode"] = "file"
	case "2":
		config["lock_mode"] = "git"
	}

	// Optional: Agent ID
	fmt.Print("  Agent ID (optional, press Enter to skip): ")
	agentID, _ := reader.ReadString('\n')
	agentID = strings.TrimSpace(agentID)
	if agentID != "" {
		config["agent_id"] = agentID
	}

	return config
}
