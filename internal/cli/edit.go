package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/alexbrand/backlog/internal/backend"
	"github.com/alexbrand/backlog/internal/config"
	"github.com/alexbrand/backlog/internal/local"
	"github.com/alexbrand/backlog/internal/output"
	"github.com/spf13/cobra"
)

var (
	editTitle       string
	editPriority    string
	editDescription string
	editAddLabels   []string
	editRemoveLabel []string
)

var editCmd = &cobra.Command{
	Use:   "edit <id>",
	Short: "Modify task fields",
	Long: `Edit an existing task's fields.

You can update the title, priority, description, and labels using the
available flags. Only the fields you specify will be changed.

Examples:
  backlog edit 001 --title="New title"
  backlog edit 001 --priority=urgent
  backlog edit 001 --add-label=blocked --remove-label=ready
  backlog edit 001 --description="Updated description"`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runEdit(args[0])
	},
}

func init() {
	rootCmd.AddCommand(editCmd)

	editCmd.Flags().StringVarP(&editTitle, "title", "t", "", "New title for the task")
	editCmd.Flags().StringVarP(&editPriority, "priority", "p", "", "New priority: urgent, high, medium, low, none")
	editCmd.Flags().StringVarP(&editDescription, "description", "d", "", "New description for the task")
	editCmd.Flags().StringSliceVar(&editAddLabels, "add-label", nil, "Labels to add (can be specified multiple times)")
	editCmd.Flags().StringSliceVar(&editRemoveLabel, "remove-label", nil, "Labels to remove (can be specified multiple times)")
}

func runEdit(id string) error {
	// Check if any changes were specified
	if editTitle == "" && editPriority == "" && editDescription == "" &&
		len(editAddLabels) == 0 && len(editRemoveLabel) == 0 {
		fmt.Fprintf(os.Stderr, "error: no changes specified\n")
		return fmt.Errorf("no changes specified")
	}

	// Validate priority if specified
	var priority *backend.Priority
	if editPriority != "" {
		p := backend.Priority(editPriority)
		if !p.IsValid() {
			fmt.Fprintf(os.Stderr, "error: invalid priority %q (valid: urgent, high, medium, low, none)\n", editPriority)
			return fmt.Errorf("invalid priority: %s", editPriority)
		}
		priority = &p
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

	// Build the changes struct
	changes := backend.TaskChanges{
		Priority:     priority,
		AddLabels:    editAddLabels,
		RemoveLabels: editRemoveLabel,
	}

	if editTitle != "" {
		changes.Title = &editTitle
	}

	if editDescription != "" {
		changes.Description = &editDescription
	}

	// Update the task
	task, err := b.Update(id, changes)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		if strings.Contains(err.Error(), "not found") {
			return NotFoundError(err.Error())
		}
		return err
	}

	// Output the result
	formatter := output.New(output.Format(GetFormat()))
	return formatter.FormatUpdated(os.Stdout, task)
}
