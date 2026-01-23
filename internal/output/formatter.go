// Package output provides formatters for displaying backlog data.
package output

import (
	"io"

	"github.com/alexbrand/backlog/internal/backend"
	"github.com/alexbrand/backlog/internal/config"
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

	// FormatTaskWithComments outputs a single task with its comments.
	FormatTaskWithComments(w io.Writer, task *backend.Task, comments []backend.Comment) error

	// FormatComment outputs a single comment.
	FormatComment(w io.Writer, comment *backend.Comment) error

	// FormatComments outputs a list of comments.
	FormatComments(w io.Writer, comments []backend.Comment) error

	// FormatCreated outputs the result of creating a task.
	FormatCreated(w io.Writer, task *backend.Task) error

	// FormatMoved outputs the result of moving a task to a new status.
	FormatMoved(w io.Writer, task *backend.Task, oldStatus, newStatus backend.Status) error

	// FormatUpdated outputs the result of updating a task.
	FormatUpdated(w io.Writer, task *backend.Task) error

	// FormatClaimed outputs the result of claiming a task.
	FormatClaimed(w io.Writer, task *backend.Task, agentID string, alreadyOwned bool) error

	// FormatReleased outputs the result of releasing a task.
	FormatReleased(w io.Writer, task *backend.Task) error

	// FormatSynced outputs the result of a sync operation.
	FormatSynced(w io.Writer, result *backend.SyncResult) error

	// FormatError outputs an error.
	FormatError(w io.Writer, code string, message string, details map[string]any) error

	// FormatConfig outputs configuration.
	FormatConfig(w io.Writer, cfg *config.Config) error

	// FormatHealthCheck outputs health check results.
	FormatHealthCheck(w io.Writer, backendName string, ws *config.Workspace, status *backend.HealthStatus) error
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
