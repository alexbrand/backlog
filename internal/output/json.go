package output

import (
	"encoding/json"
	"io"

	"github.com/alexbrand/backlog/internal/backend"
)

// JSONFormatter outputs data in JSON format.
type JSONFormatter struct{}

// FormatTask outputs a single task as JSON.
func (f *JSONFormatter) FormatTask(w io.Writer, task *backend.Task) error {
	return f.writeJSON(w, task)
}

// FormatTaskList outputs a list of tasks as JSON.
func (f *JSONFormatter) FormatTaskList(w io.Writer, list *backend.TaskList) error {
	return f.writeJSON(w, list)
}

// FormatComment outputs a single comment as JSON.
func (f *JSONFormatter) FormatComment(w io.Writer, comment *backend.Comment) error {
	return f.writeJSON(w, comment)
}

// FormatComments outputs a list of comments as JSON.
func (f *JSONFormatter) FormatComments(w io.Writer, comments []backend.Comment) error {
	return f.writeJSON(w, map[string]any{
		"comments": comments,
		"count":    len(comments),
	})
}

// FormatCreated outputs the result of creating a task as JSON.
func (f *JSONFormatter) FormatCreated(w io.Writer, task *backend.Task) error {
	return f.writeJSON(w, map[string]any{
		"id":    task.ID,
		"title": task.Title,
		"url":   task.URL,
	})
}

// FormatMoved outputs the result of moving a task as JSON.
func (f *JSONFormatter) FormatMoved(w io.Writer, task *backend.Task, oldStatus, newStatus backend.Status) error {
	return f.writeJSON(w, map[string]any{
		"id":     task.ID,
		"title":  task.Title,
		"status": newStatus,
	})
}

// FormatUpdated outputs the result of updating a task as JSON.
func (f *JSONFormatter) FormatUpdated(w io.Writer, task *backend.Task) error {
	return f.writeJSON(w, map[string]any{
		"id":    task.ID,
		"title": task.Title,
		"url":   task.URL,
	})
}

// FormatClaimed outputs the result of claiming a task as JSON.
func (f *JSONFormatter) FormatClaimed(w io.Writer, task *backend.Task, agentID string, alreadyOwned bool) error {
	return f.writeJSON(w, map[string]any{
		"id":           task.ID,
		"title":        task.Title,
		"status":       task.Status,
		"agent":        agentID,
		"alreadyOwned": alreadyOwned,
		"url":          task.URL,
	})
}

// FormatError outputs an error as JSON.
func (f *JSONFormatter) FormatError(w io.Writer, code string, message string, details map[string]any) error {
	errObj := map[string]any{
		"error": map[string]any{
			"code":    code,
			"message": message,
		},
	}
	if details != nil {
		errObj["error"].(map[string]any)["details"] = details
	} else {
		errObj["error"].(map[string]any)["details"] = map[string]any{}
	}
	return f.writeJSON(w, errObj)
}

// writeJSON encodes the value as indented JSON and writes it to w.
func (f *JSONFormatter) writeJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}
