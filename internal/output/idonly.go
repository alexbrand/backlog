package output

import (
	"fmt"
	"io"

	"github.com/alexbrand/backlog/internal/backend"
)

// IDOnlyFormatter outputs only task IDs, one per line.
type IDOnlyFormatter struct{}

// FormatTask outputs only the task ID.
func (f *IDOnlyFormatter) FormatTask(w io.Writer, task *backend.Task) error {
	fmt.Fprintln(w, task.ID)
	return nil
}

// FormatTaskList outputs only task IDs, one per line.
func (f *IDOnlyFormatter) FormatTaskList(w io.Writer, list *backend.TaskList) error {
	for _, task := range list.Tasks {
		fmt.Fprintln(w, task.ID)
	}
	return nil
}

// FormatComment outputs only the comment ID.
func (f *IDOnlyFormatter) FormatComment(w io.Writer, comment *backend.Comment) error {
	fmt.Fprintln(w, comment.ID)
	return nil
}

// FormatComments outputs only comment IDs, one per line.
func (f *IDOnlyFormatter) FormatComments(w io.Writer, comments []backend.Comment) error {
	for _, comment := range comments {
		fmt.Fprintln(w, comment.ID)
	}
	return nil
}

// FormatCreated outputs only the created task ID.
func (f *IDOnlyFormatter) FormatCreated(w io.Writer, task *backend.Task) error {
	fmt.Fprintln(w, task.ID)
	return nil
}

// FormatMoved outputs only the moved task ID.
func (f *IDOnlyFormatter) FormatMoved(w io.Writer, task *backend.Task, _, _ backend.Status) error {
	fmt.Fprintln(w, task.ID)
	return nil
}

// FormatError outputs an error message (errors are always shown).
func (f *IDOnlyFormatter) FormatError(w io.Writer, code string, message string, details map[string]any) error {
	fmt.Fprintf(w, "error: %s\n", message)
	return nil
}
