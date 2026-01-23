package cli

import (
	"fmt"
	"os"

	"github.com/alexbrand/backlog/internal/backend"
	"github.com/alexbrand/backlog/internal/config"
	"github.com/alexbrand/backlog/internal/local"
	"github.com/alexbrand/backlog/internal/output"
	"github.com/spf13/cobra"
)

var (
	addPriority    string
	addLabels      []string
	addDescription string
	addBodyFile    string
	addStatus      string
)

var addCmd = &cobra.Command{
	Use:   "add <title>",
	Short: "Create a new task",
	Long: `Create a new task in the backlog.

The title is required and provided as the first argument. Additional fields
can be set using flags.

Examples:
  backlog add "Implement rate limiting"
  backlog add "Fix login bug" --priority=urgent --label=bug
  backlog add "Refactor API" --description="Split into modules" --status=todo
  backlog add "Research caching" --body-file=./task-details.md`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runAdd(args[0])
	},
}

func init() {
	rootCmd.AddCommand(addCmd)

	addCmd.Flags().StringVarP(&addPriority, "priority", "p", "", "Priority: urgent, high, medium, low, none (default: none)")
	addCmd.Flags().StringSliceVarP(&addLabels, "label", "l", nil, "Add labels (can be specified multiple times)")
	addCmd.Flags().StringVarP(&addDescription, "description", "d", "", "Task description")
	addCmd.Flags().StringVar(&addBodyFile, "body-file", "", "Read description from file")
	addCmd.Flags().StringVarP(&addStatus, "status", "s", "", "Initial status: backlog, todo, in-progress, review, done (default: backlog)")
}

func runAdd(title string) error {
	// Validate title
	if title == "" {
		fmt.Fprintf(os.Stderr, "error: title is required\n")
		return fmt.Errorf("title is required")
	}

	// Handle description from file
	description := addDescription
	if addBodyFile != "" {
		content, err := os.ReadFile(addBodyFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: failed to read body file: %v\n", err)
			return err
		}
		description = string(content)
	}

	// Validate and parse priority
	var priority backend.Priority
	if addPriority != "" {
		priority = backend.Priority(addPriority)
		if !priority.IsValid() {
			fmt.Fprintf(os.Stderr, "error: invalid priority %q (valid: urgent, high, medium, low, none)\n", addPriority)
			return fmt.Errorf("invalid priority: %s", addPriority)
		}
	}

	// Validate and parse status
	var status backend.Status
	if addStatus != "" {
		status = backend.Status(addStatus)
		if !status.IsValid() {
			fmt.Fprintf(os.Stderr, "error: invalid status %q (valid: backlog, todo, in-progress, review, done)\n", addStatus)
			return fmt.Errorf("invalid status: %s", addStatus)
		}
	}

	// Get backend and configuration
	var b backend.Backend
	var backendCfg backend.Config

	// Try to get workspace from config
	ws, _, err := config.GetWorkspace(GetWorkspace())
	if err == nil {
		// Have config - use it
		b, err = backend.Get(ws.Backend)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			return err
		}

		cfg := config.Get()
		backendCfg = backend.Config{
			AgentID:          cfg.Defaults.AgentID,
			AgentLabelPrefix: ws.AgentLabelPrefix,
		}

		switch ws.Backend {
		case "local":
			path := ws.Path
			if path == "" {
				path = ".backlog"
			}
			backendCfg.Workspace = &local.WorkspaceConfig{Path: path}
		default:
			fmt.Fprintf(os.Stderr, "error: unsupported backend: %s\n", ws.Backend)
			return fmt.Errorf("unsupported backend: %s", ws.Backend)
		}
	} else {
		// No config - check for local .backlog directory
		if _, statErr := os.Stat(".backlog"); statErr == nil {
			// Local .backlog directory exists - use local backend
			b, err = backend.Get("local")
			if err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
				return err
			}
			backendCfg = backend.Config{
				Workspace: &local.WorkspaceConfig{Path: ".backlog"},
			}
		} else {
			// No config and no local .backlog directory
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			return err
		}
	}

	if err := b.Connect(backendCfg); err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to connect to backend: %v\n", err)
		return err
	}
	defer b.Disconnect()

	// Create the task
	input := backend.TaskInput{
		Title:       title,
		Description: description,
		Status:      status,
		Priority:    priority,
		Labels:      addLabels,
	}

	task, err := b.Create(input)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to create task: %v\n", err)
		return err
	}

	// Output the result
	formatter := output.New(output.Format(GetFormat()))
	return formatter.FormatCreated(os.Stdout, task)
}
