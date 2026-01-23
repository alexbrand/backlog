// Package output provides formatters for displaying backlog data.
package output

import (
	"io"

	"github.com/alexbrand/backlog/internal/backend"
)

// Format represents an output format type.
type Format string

const (
	FormatTable  Format = "table"
	FormatJSON   Format = "json"
	FormatPlain  Format = "plain"
	FormatIDOnly Format = "id-only"
)

// ValidFormats returns all valid format values.
func ValidFormats() []Format {
	return []Format{FormatTable, FormatJSON, FormatPlain, FormatIDOnly}
}

// IsValid checks if the format is a valid output format.
func (f Format) IsValid() bool {
	switch f {
	case FormatTable, FormatJSON, FormatPlain, FormatIDOnly:
		return true
	default:
		return false
	}
}

// Formatter defines the interface for outputting backlog data in various formats.
type Formatter interface {
	// FormatTask outputs a single task.
	FormatTask(w io.Writer, task *backend.Task) error

	// FormatTaskList outputs a list of tasks.
	FormatTaskList(w io.Writer, list *backend.TaskList) error

	// FormatComment outputs a single comment.
	FormatComment(w io.Writer, comment *backend.Comment) error

	// FormatComments outputs a list of comments.
	FormatComments(w io.Writer, comments []backend.Comment) error

	// FormatCreated outputs the result of creating a task.
	FormatCreated(w io.Writer, task *backend.Task) error

	// FormatMoved outputs the result of moving a task to a new status.
	FormatMoved(w io.Writer, task *backend.Task, oldStatus, newStatus backend.Status) error

	// FormatError outputs an error.
	FormatError(w io.Writer, code string, message string, details map[string]any) error
}

// New creates a formatter for the specified format.
func New(format Format) Formatter {
	switch format {
	case FormatJSON:
		return &JSONFormatter{}
	case FormatPlain:
		return &PlainFormatter{}
	case FormatIDOnly:
		return &IDOnlyFormatter{}
	case FormatTable:
		fallthrough
	default:
		return &TableFormatter{}
	}
}
