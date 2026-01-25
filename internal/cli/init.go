package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/alexbrand/backlog/internal/credentials"
	"github.com/alexbrand/backlog/internal/github"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new backlog in the current directory",
	Long: `Initialize a new backlog in the current directory.

This command creates the .backlog/ directory structure and guides you through
configuring the backend (GitHub Issues or local filesystem).

For GitHub backend, it will attempt to auto-detect:
  - Repository from git remote
  - Authentication from gh CLI or GITHUB_TOKEN

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

	// Detect if we're in a git repo with a GitHub remote
	detectedRepo := detectGitHubRepo()
	defaultBackend := "1"
	if detectedRepo != "" {
		fmt.Printf("Detected GitHub repository: %s\n", detectedRepo)
		fmt.Println()
	}

	// Choose backend
	fmt.Println("Backend:")
	fmt.Println("  1. GitHub Issues (sync with GitHub)")
	fmt.Println("  2. Local filesystem (standalone)")
	fmt.Print("Choose [1]: ")
	backendChoice, _ := reader.ReadString('\n')
	backendChoice = strings.TrimSpace(backendChoice)
	if backendChoice == "" {
		backendChoice = defaultBackend
	}

	var backendType string
	var workspaceConfig map[string]any

	switch backendChoice {
	case "1", "github":
		backendType = "github"
		workspaceConfig = configureGitHubBackendInit(reader, detectedRepo)
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

// detectGitHubRepo attempts to detect the GitHub repository from git remote.
// Returns "owner/repo" format or empty string if not detected.
func detectGitHubRepo() string {
	// Try to get the origin remote URL
	cmd := exec.Command("git", "config", "--get", "remote.origin.url")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	url := strings.TrimSpace(string(output))
	return parseGitHubRepoFromURL(url)
}

// parseGitHubRepoFromURL extracts owner/repo from various GitHub URL formats.
func parseGitHubRepoFromURL(url string) string {
	// Handle SSH format: git@github.com:owner/repo.git
	sshPattern := regexp.MustCompile(`git@github\.com:([^/]+)/([^/]+?)(?:\.git)?$`)
	if matches := sshPattern.FindStringSubmatch(url); len(matches) == 3 {
		return matches[1] + "/" + matches[2]
	}

	// Handle HTTPS format: https://github.com/owner/repo.git
	httpsPattern := regexp.MustCompile(`https://github\.com/([^/]+)/([^/]+?)(?:\.git)?$`)
	if matches := httpsPattern.FindStringSubmatch(url); len(matches) == 3 {
		return matches[1] + "/" + matches[2]
	}

	return ""
}

// detectGitHubToken checks for GitHub token from various sources.
// Returns the token and a description of where it was found.
func detectGitHubToken() (token string, source string) {
	// 1. Check GITHUB_TOKEN environment variable
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		return token, "GITHUB_TOKEN environment variable"
	}

	// 2. Check credentials file
	if err := credentials.Init(); err == nil {
		if token, err := credentials.GetGitHubToken(); err == nil {
			return token, "credentials file"
		}
	}

	// 3. Try gh CLI
	if token := getGhCliToken(); token != "" {
		return token, "gh CLI"
	}

	return "", ""
}

// getGhCliToken attempts to get a GitHub token from the gh CLI.
func getGhCliToken() string {
	cmd := exec.Command("gh", "auth", "token")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

// isGhCliAuthenticated checks if gh CLI is installed and authenticated.
func isGhCliAuthenticated() bool {
	cmd := exec.Command("gh", "auth", "status")
	err := cmd.Run()
	return err == nil
}

func configureGitHubBackendInit(reader *bufio.Reader, detectedRepo string) map[string]any {
	config := make(map[string]any)

	fmt.Println()
	fmt.Println("GitHub Backend Setup:")

	// Repository - use detected value as default
	if detectedRepo != "" {
		fmt.Printf("  Repository [%s]: ", detectedRepo)
	} else {
		fmt.Print("  Repository (owner/repo): ")
	}
	repo, _ := reader.ReadString('\n')
	repo = strings.TrimSpace(repo)
	if repo == "" && detectedRepo != "" {
		repo = detectedRepo
	}
	if repo != "" {
		config["repo"] = repo
	}

	// Check for GitHub token
	token, tokenSource := detectGitHubToken()
	if token != "" {
		fmt.Printf("  Token: Found via %s\n", tokenSource)

		// If token came from gh CLI, offer to save it to credentials file
		if tokenSource == "gh CLI" {
			fmt.Print("  Save token to credentials file for faster access? [Y/n]: ")
			saveChoice, _ := reader.ReadString('\n')
			saveChoice = strings.TrimSpace(strings.ToLower(saveChoice))
			if saveChoice == "" || saveChoice == "y" || saveChoice == "yes" {
				if err := credentials.SaveGitHubToken(token); err != nil {
					fmt.Printf("  Warning: failed to save token: %v\n", err)
				} else {
					credPath, _ := credentials.DefaultCredentialsPath()
					fmt.Printf("  Token saved to %s\n", credPath)
				}
			}
		}
	} else {
		fmt.Println()
		fmt.Println("  No GitHub token found.")
		fmt.Println("  Options:")
		fmt.Println("    - Run 'gh auth login' to authenticate with GitHub CLI")
		fmt.Println("    - Set GITHUB_TOKEN environment variable")
		fmt.Println("    - Add token to ~/.config/backlog/credentials.yaml")
	}

	// GitHub Projects setup (if token and repo are available)
	if token != "" && repo != "" {
		projectNum, err := configureGitHubProject(reader, token, repo)
		if err != nil {
			fmt.Printf("  Warning: %v\n", err)
			fmt.Println("  Continuing with label-based status tracking...")
		} else if projectNum > 0 {
			config["project"] = projectNum
			config["status_field"] = "Status"
		}
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

// configureGitHubProject handles the GitHub Projects setup during init.
// Returns the project number to use, or 0 to skip project integration.
func configureGitHubProject(reader *bufio.Reader, token, repo string) (int, error) {
	// Parse owner/repo
	parts := strings.SplitN(repo, "/", 2)
	if len(parts) != 2 {
		return 0, fmt.Errorf("invalid repo format: %s", repo)
	}
	owner, repoName := parts[0], parts[1]

	ctx := context.Background()
	apiURL := os.Getenv("GITHUB_API_URL") // For testing

	fmt.Println()
	fmt.Println("  GitHub Projects:")

	// List existing projects
	projects, err := github.ListRepositoryProjects(ctx, token, owner, repoName, apiURL)
	if err != nil {
		return 0, fmt.Errorf("failed to list projects: %w", err)
	}

	// Display options based on existing projects
	if len(projects) > 0 {
		fmt.Printf("    Found %d project(s):\n", len(projects))
		for i, p := range projects {
			fmt.Printf("      %d. %s (#%d)\n", i+1, p.Title, p.Number)
		}
		fmt.Println()
		fmt.Println("    Options:")
		fmt.Printf("      [1-%d] Select existing project\n", len(projects))
		fmt.Println("      [c]   Create new project")
		fmt.Println("      [s]   Skip (use labels for status)")
		fmt.Println()

		defaultChoice := "1"
		if len(projects) == 1 {
			fmt.Printf("    Choose [%d]: ", projects[0].Number)
			defaultChoice = "1"
		} else {
			fmt.Print("    Choose: ")
			defaultChoice = ""
		}

		choice, _ := reader.ReadString('\n')
		choice = strings.TrimSpace(strings.ToLower(choice))
		if choice == "" {
			choice = defaultChoice
		}

		switch choice {
		case "s", "skip":
			fmt.Println("    Skipping GitHub Projects integration.")
			return 0, nil
		case "c", "create":
			return createNewProject(reader, ctx, token, owner, repoName, apiURL)
		default:
			// Try to parse as number (1-based index)
			if idx, err := strconv.Atoi(choice); err == nil && idx >= 1 && idx <= len(projects) {
				selected := projects[idx-1]
				fmt.Printf("    Selected: %s (#%d)\n", selected.Title, selected.Number)
				return selected.Number, nil
			}
			fmt.Println("    Invalid choice, skipping project setup.")
			return 0, nil
		}
	}

	// No existing projects
	fmt.Println("    No projects found for this repository.")
	fmt.Println()
	fmt.Println("    Options:")
	fmt.Println("      [c] Create new project (recommended)")
	fmt.Println("      [s] Skip (use labels for status)")
	fmt.Print("    Choose [c]: ")

	choice, _ := reader.ReadString('\n')
	choice = strings.TrimSpace(strings.ToLower(choice))
	if choice == "" {
		choice = "c"
	}

	switch choice {
	case "s", "skip":
		fmt.Println("    Skipping GitHub Projects integration.")
		return 0, nil
	case "c", "create", "":
		return createNewProject(reader, ctx, token, owner, repoName, apiURL)
	default:
		fmt.Println("    Invalid choice, skipping project setup.")
		return 0, nil
	}
}

// createNewProject creates a new GitHub Project with standard status columns.
func createNewProject(reader *bufio.Reader, ctx context.Context, token, owner, repo, apiURL string) (int, error) {
	// Prompt for project name
	defaultName := fmt.Sprintf("%s/%s Backlog", owner, repo)
	fmt.Printf("    Project name [%s]: ", defaultName)
	name, _ := reader.ReadString('\n')
	name = strings.TrimSpace(name)
	if name == "" {
		name = defaultName
	}

	// Get owner ID for project creation
	fmt.Print("    Creating project...")
	ownerID, err := github.GetOwnerID(ctx, token, owner, apiURL)
	if err != nil {
		fmt.Println(" failed")
		return 0, fmt.Errorf("failed to get owner ID: %w", err)
	}

	// Create the project
	result, err := github.CreateProject(ctx, token, ownerID, name, apiURL)
	if err != nil {
		fmt.Println(" failed")
		return 0, fmt.Errorf("failed to create project: %w", err)
	}

	fmt.Printf(" created (#%d)\n", result.Number)

	// Link project to repository
	fmt.Print("    Linking to repository...")
	repoID, err := github.GetRepositoryID(ctx, token, owner, repo, apiURL)
	if err != nil {
		fmt.Printf(" warning: %v\n", err)
	} else {
		if err := github.LinkProjectToRepository(ctx, token, result.ID, repoID, apiURL); err != nil {
			fmt.Printf(" warning: %v\n", err)
		} else {
			fmt.Println(" done")
		}
	}

	// Check status options
	fmt.Print("    Checking status columns...")
	if err := github.ConfigureProjectStatus(ctx, token, result.ID, apiURL); err != nil {
		fmt.Printf(" warning: %v\n", err)
		fmt.Println("    Note: Add missing columns manually in GitHub Projects settings")
	} else {
		fmt.Println(" done")
	}

	return result.Number, nil
}
