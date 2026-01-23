package output

import (
	"fmt"
	"io"

	"github.com/alexbrand/backlog/internal/backend"
)

// PlainFormatter outputs data in plain text format, suitable for scripting.
type PlainFormatter struct{}

// FormatTask outputs a single task in plain format.
// Includes all task fields for detailed view (used by show command).
func (f *PlainFormatter) FormatTask(w io.Writer, task *backend.Task) error {
	fmt.Fprintf(w, "%s\t%s\t%s\t%s", task.ID, task.Status, task.Priority, task.Title)
	if task.Assignee != "" {
		fmt.Fprintf(w, "\t%s", task.Assignee)
	}
	if len(task.Labels) > 0 {
		fmt.Fprintf(w, "\t%s", joinLabels(task.Labels))
	}
	fmt.Fprintln(w)
	if task.Description != "" {
		fmt.Fprintln(w, task.Description)
	}
	return nil
}

// joinLabels joins labels with commas for plain output.
func joinLabels(labels []string) string {
	result := ""
	for i, l := range labels {
		if i > 0 {
			result += ","
		}
		result += l
	}
	return result
}

// formatTaskSummary outputs a single task in summary format (one line).
func (f *PlainFormatter) formatTaskSummary(w io.Writer, task *backend.Task) error {
	fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", task.ID, task.Status, task.Priority, task.Title)
	return nil
}

// FormatTaskList outputs a list of tasks in plain format.
// Uses summary format (one line per task) without descriptions.
func (f *PlainFormatter) FormatTaskList(w io.Writer, list *backend.TaskList) error {
	for _, task := range list.Tasks {
		if err := f.formatTaskSummary(w, &task); err != nil {
			return err
		}
	}
	return nil
}

// FormatComment outputs a single comment in plain format.
func (f *PlainFormatter) FormatComment(w io.Writer, comment *backend.Comment) error {
	fmt.Fprintf(w, "%s\t%s\t%s\n", comment.ID, comment.Author, comment.Body)
	return nil
}

// FormatComments outputs a list of comments in plain format.
func (f *PlainFormatter) FormatComments(w io.Writer, comments []backend.Comment) error {
	for _, comment := range comments {
		if err := f.FormatComment(w, &comment); err != nil {
			return err
		}
	}
	return nil
}

// FormatCreated outputs the result of creating a task in plain format.
func (f *PlainFormatter) FormatCreated(w io.Writer, task *backend.Task) error {
	fmt.Fprintln(w, task.ID)
	return nil
}

// FormatMoved outputs the result of moving a task in plain format.
func (f *PlainFormatter) FormatMoved(w io.Writer, task *backend.Task, oldStatus, newStatus backend.Status) error {
	fmt.Fprintf(w, "%s\t%s\t%s\n", task.ID, oldStatus, newStatus)
	return nil
}

// FormatUpdated outputs the result of updating a task in plain format.
func (f *PlainFormatter) FormatUpdated(w io.Writer, task *backend.Task) error {
	fmt.Fprintln(w, task.ID)
	return nil
}

// FormatClaimed outputs the result of claiming a task in plain format.
func (f *PlainFormatter) FormatClaimed(w io.Writer, task *backend.Task, agentID string, alreadyOwned bool) error {
	fmt.Fprintf(w, "%s\t%s\t%s\n", task.ID, task.Status, agentID)
	return nil
}

// FormatReleased outputs the result of releasing a task in plain format.
func (f *PlainFormatter) FormatReleased(w io.Writer, task *backend.Task) error {
	fmt.Fprintf(w, "%s\t%s\n", task.ID, task.Status)
	return nil
}

// FormatSynced outputs the result of a sync operation in plain format.
func (f *PlainFormatter) FormatSynced(w io.Writer, result *backend.SyncResult) error {
	fmt.Fprintf(w, "%d\t%d\t%d\t%d\t%d\n",
		result.Created, result.Updated, result.Deleted, result.Pushed, result.Conflicts)
	return nil
}

// FormatError outputs an error in plain format.
func (f *PlainFormatter) FormatError(w io.Writer, code string, message string, details map[string]any) error {
	fmt.Fprintf(w, "error: %s\n", message)
	return nil
}
