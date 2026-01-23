package output

import (
	"fmt"
	"io"

	"github.com/alexbrand/backlog/internal/backend"
)

// PlainFormatter outputs data in plain text format, suitable for scripting.
type PlainFormatter struct{}

// FormatTask outputs a single task in plain format.
func (f *PlainFormatter) FormatTask(w io.Writer, task *backend.Task) error {
	fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", task.ID, task.Status, task.Priority, task.Title)
	return nil
}

// FormatTaskList outputs a list of tasks in plain format.
func (f *PlainFormatter) FormatTaskList(w io.Writer, list *backend.TaskList) error {
	for _, task := range list.Tasks {
		if err := f.FormatTask(w, &task); err != nil {
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

// FormatError outputs an error in plain format.
func (f *PlainFormatter) FormatError(w io.Writer, code string, message string, details map[string]any) error {
	fmt.Fprintf(w, "error: %s\n", message)
	return nil
}
