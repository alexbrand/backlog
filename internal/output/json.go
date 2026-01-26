package output

import (
	"encoding/json"
	"io"

	"github.com/alexbrand/backlog/internal/backend"
	"github.com/alexbrand/backlog/internal/config"
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

// FormatTaskWithComments outputs a single task with its comments as JSON.
func (f *JSONFormatter) FormatTaskWithComments(w io.Writer, task *backend.Task, comments []backend.Comment) error {
	// Create a combined structure that embeds the task and adds comments
	result := map[string]any{
		"id":          task.ID,
		"title":       task.Title,
		"description": task.Description,
		"status":      task.Status,
		"priority":    task.Priority,
		"assignee":    task.Assignee,
		"created":     task.Created,
		"updated":     task.Updated,
		"url":         task.URL,
		"labels":      task.Labels,
		"meta":        task.Meta,
		"comments":    comments,
	}
	return f.writeJSON(w, result)
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
		"id":       task.ID,
		"title":    task.Title,
		"url":      task.URL,
		"status":   task.Status,
		"labels":   task.Labels,
		"priority": task.Priority,
	})
}

// FormatMoved outputs the result of moving a task as JSON.
func (f *JSONFormatter) FormatMoved(w io.Writer, task *backend.Task, oldStatus, newStatus backend.Status) error {
	return f.writeJSON(w, map[string]any{
		"id":       task.ID,
		"title":    task.Title,
		"status":   newStatus,
		"labels":   task.Labels,
		"priority": task.Priority,
	})
}

// FormatUpdated outputs the result of updating a task as JSON.
func (f *JSONFormatter) FormatUpdated(w io.Writer, task *backend.Task) error {
	return f.writeJSON(w, map[string]any{
		"id":       task.ID,
		"title":    task.Title,
		"url":      task.URL,
		"labels":   task.Labels,
		"priority": task.Priority,
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
		"labels":       task.Labels,
		"assignee":     task.Assignee,
	})
}

// FormatReleased outputs the result of releasing a task as JSON.
func (f *JSONFormatter) FormatReleased(w io.Writer, task *backend.Task) error {
	return f.writeJSON(w, map[string]any{
		"id":       task.ID,
		"title":    task.Title,
		"status":   task.Status,
		"url":      task.URL,
		"assignee": task.Assignee,
		"labels":   task.Labels,
	})
}

// FormatSynced outputs the result of a sync operation as JSON.
func (f *JSONFormatter) FormatSynced(w io.Writer, result *backend.SyncResult) error {
	return f.writeJSON(w, map[string]any{
		"created":   result.Created,
		"updated":   result.Updated,
		"deleted":   result.Deleted,
		"pushed":    result.Pushed,
		"conflicts": result.Conflicts,
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

// FormatConfig outputs configuration as JSON.
func (f *JSONFormatter) FormatConfig(w io.Writer, cfg *config.Config) error {
	return f.writeJSON(w, cfg)
}

// FormatHealthCheck outputs health check results as JSON.
func (f *JSONFormatter) FormatHealthCheck(w io.Writer, backendName string, ws *config.Workspace, status *backend.HealthStatus) error {
	result := map[string]any{
		"backend": backendName,
		"healthy": status.OK,
		"message": status.Message,
		"latency": status.Latency.String(),
	}
	if ws != nil {
		wsInfo := map[string]any{}
		if ws.Project > 0 {
			wsInfo["project"] = ws.Project
		}
		if len(wsInfo) > 0 {
			result["workspace"] = wsInfo
		}
	}
	return f.writeJSON(w, result)
}

// FormatDeleted outputs the result of deleting a task as JSON.
func (f *JSONFormatter) FormatDeleted(w io.Writer, id string) error {
	return f.writeJSON(w, map[string]any{
		"id":      id,
		"deleted": true,
	})
}

// FormatReordered outputs the result of reordering a task as JSON.
func (f *JSONFormatter) FormatReordered(w io.Writer, task *backend.Task) error {
	return f.writeJSON(w, map[string]any{
		"id":         task.ID,
		"title":      task.Title,
		"status":     task.Status,
		"priority":   task.Priority,
		"sort_order": task.SortOrder,
	})
}

// writeJSON encodes the value as indented JSON and writes it to w.
func (f *JSONFormatter) writeJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}
