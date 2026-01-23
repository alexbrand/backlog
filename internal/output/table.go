package output

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/alexbrand/backlog/internal/backend"
	"github.com/alexbrand/backlog/internal/config"
)

// TableFormatter outputs data in a human-readable table format.
type TableFormatter struct{}

// FormatTask outputs a single task in detailed format.
func (f *TableFormatter) FormatTask(w io.Writer, task *backend.Task) error {
	// Header with ID and title
	fmt.Fprintf(w, "%s: %s\n", task.ID, task.Title)
	fmt.Fprintln(w, strings.Repeat("━", 40))
	fmt.Fprintln(w)

	// Fields
	fmt.Fprintf(w, "Status:    %s\n", task.Status)
	fmt.Fprintf(w, "Priority:  %s\n", task.Priority)

	if task.Assignee != "" {
		fmt.Fprintf(w, "Assignee:  @%s\n", task.Assignee)
	} else {
		fmt.Fprintf(w, "Assignee:  —\n")
	}

	if len(task.Labels) > 0 {
		fmt.Fprintf(w, "Labels:    %s\n", strings.Join(task.Labels, ", "))
	}

	fmt.Fprintf(w, "Created:   %s\n", task.Created.Format("2006-01-02 15:04"))
	fmt.Fprintf(w, "Updated:   %s\n", task.Updated.Format("2006-01-02 15:04"))

	if task.URL != "" {
		fmt.Fprintf(w, "URL:       %s\n", task.URL)
	}

	// Description
	if task.Description != "" {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "## Description")
		fmt.Fprintln(w)
		fmt.Fprintln(w, task.Description)
	}

	return nil
}

// FormatTaskList outputs a list of tasks in table format.
func (f *TableFormatter) FormatTaskList(w io.Writer, list *backend.TaskList) error {
	if len(list.Tasks) == 0 {
		fmt.Fprintln(w, "No tasks found.")
		return nil
	}

	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)

	// Header
	fmt.Fprintln(tw, "ID\tSTATUS\tPRIORITY\tTITLE\tASSIGNEE")

	// Rows
	for _, task := range list.Tasks {
		assignee := "—"
		if task.Assignee != "" {
			assignee = "@" + task.Assignee
		}

		// Truncate title if too long
		title := task.Title
		if len(title) > 40 {
			title = title[:37] + "..."
		}

		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n",
			task.ID,
			task.Status,
			task.Priority,
			title,
			assignee,
		)
	}

	return tw.Flush()
}

// FormatComment outputs a single comment.
func (f *TableFormatter) FormatComment(w io.Writer, comment *backend.Comment) error {
	fmt.Fprintf(w, "### %s @%s\n", comment.Created.Format("2006-01-02"), comment.Author)
	fmt.Fprintln(w, comment.Body)
	return nil
}

// FormatComments outputs a list of comments.
func (f *TableFormatter) FormatComments(w io.Writer, comments []backend.Comment) error {
	if len(comments) == 0 {
		fmt.Fprintln(w, "No comments.")
		return nil
	}

	fmt.Fprintln(w, "## Comments")
	fmt.Fprintln(w)

	for i, comment := range comments {
		if err := f.FormatComment(w, &comment); err != nil {
			return err
		}
		if i < len(comments)-1 {
			fmt.Fprintln(w)
		}
	}

	return nil
}

// FormatCreated outputs the result of creating a task.
func (f *TableFormatter) FormatCreated(w io.Writer, task *backend.Task) error {
	fmt.Fprintf(w, "Created %s: %s\n", task.ID, task.Title)
	return nil
}

// FormatMoved outputs the result of moving a task to a new status.
func (f *TableFormatter) FormatMoved(w io.Writer, task *backend.Task, oldStatus, newStatus backend.Status) error {
	fmt.Fprintf(w, "Moved %s: %s → %s\n", task.ID, oldStatus, newStatus)
	return nil
}

// FormatUpdated outputs the result of updating a task.
func (f *TableFormatter) FormatUpdated(w io.Writer, task *backend.Task) error {
	fmt.Fprintf(w, "Updated %s: %s\n", task.ID, task.Title)
	return nil
}

// FormatClaimed outputs the result of claiming a task.
func (f *TableFormatter) FormatClaimed(w io.Writer, task *backend.Task, agentID string, alreadyOwned bool) error {
	if alreadyOwned {
		fmt.Fprintf(w, "Already claimed %s: %s (agent: %s)\n", task.ID, task.Title, agentID)
	} else {
		fmt.Fprintf(w, "Claimed %s: %s (agent: %s)\n", task.ID, task.Title, agentID)
	}
	return nil
}

// FormatReleased outputs the result of releasing a task.
func (f *TableFormatter) FormatReleased(w io.Writer, task *backend.Task) error {
	fmt.Fprintf(w, "Released %s: %s\n", task.ID, task.Title)
	return nil
}

// FormatSynced outputs the result of a sync operation.
func (f *TableFormatter) FormatSynced(w io.Writer, result *backend.SyncResult) error {
	if result.Updated == 0 && result.Pushed == 0 && result.Created == 0 && result.Deleted == 0 {
		fmt.Fprintln(w, "Already up to date.")
		return nil
	}

	fmt.Fprintln(w, "Sync complete:")
	if result.Created > 0 {
		fmt.Fprintf(w, "  Created: %d\n", result.Created)
	}
	if result.Updated > 0 {
		fmt.Fprintf(w, "  Updated: %d\n", result.Updated)
	}
	if result.Deleted > 0 {
		fmt.Fprintf(w, "  Deleted: %d\n", result.Deleted)
	}
	if result.Pushed > 0 {
		fmt.Fprintf(w, "  Pushed:  %d\n", result.Pushed)
	}
	if result.Conflicts > 0 {
		fmt.Fprintf(w, "  Conflicts: %d\n", result.Conflicts)
	}
	return nil
}

// FormatError outputs an error message.
func (f *TableFormatter) FormatError(w io.Writer, code string, message string, details map[string]any) error {
	fmt.Fprintf(w, "error: %s\n", message)
	return nil
}

// FormatConfig outputs configuration.
func (f *TableFormatter) FormatConfig(w io.Writer, cfg *config.Config) error {
	fmt.Fprintf(w, "Configuration:\n")
	fmt.Fprintf(w, "  Version: %d\n", cfg.Version)
	fmt.Fprintf(w, "  Default Workspace: %s\n", cfg.Defaults.Workspace)
	return nil
}

// FormatHealthCheck outputs health check results.
func (f *TableFormatter) FormatHealthCheck(w io.Writer, backendName string, ws *config.Workspace, status *backend.HealthStatus) error {
	if status.OK {
		fmt.Fprintf(w, "%s: healthy (%v)\n", backendName, status.Latency)
	} else {
		fmt.Fprintf(w, "%s: unhealthy - %s\n", backendName, status.Message)
	}
	if ws != nil && ws.Project > 0 {
		fmt.Fprintf(w, "project: %d\n", ws.Project)
	}
	return nil
}
