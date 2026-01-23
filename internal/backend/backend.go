// Package backend defines the core types and interfaces for backlog backends.
package backend

import "time"

// Status represents the canonical status of a task.
type Status string

const (
	StatusBacklog    Status = "backlog"
	StatusTodo       Status = "todo"
	StatusInProgress Status = "in-progress"
	StatusReview     Status = "review"
	StatusDone       Status = "done"
)

// ValidStatuses returns all valid status values.
func ValidStatuses() []Status {
	return []Status{StatusBacklog, StatusTodo, StatusInProgress, StatusReview, StatusDone}
}

// IsValid checks if the status is a valid canonical status.
func (s Status) IsValid() bool {
	switch s {
	case StatusBacklog, StatusTodo, StatusInProgress, StatusReview, StatusDone:
		return true
	default:
		return false
	}
}

// Priority represents the priority level of a task.
type Priority string

const (
	PriorityUrgent Priority = "urgent"
	PriorityHigh   Priority = "high"
	PriorityMedium Priority = "medium"
	PriorityLow    Priority = "low"
	PriorityNone   Priority = "none"
)

// ValidPriorities returns all valid priority values.
func ValidPriorities() []Priority {
	return []Priority{PriorityUrgent, PriorityHigh, PriorityMedium, PriorityLow, PriorityNone}
}

// IsValid checks if the priority is a valid priority level.
func (p Priority) IsValid() bool {
	switch p {
	case PriorityUrgent, PriorityHigh, PriorityMedium, PriorityLow, PriorityNone:
		return true
	default:
		return false
	}
}

// Task represents a work item in the backlog.
type Task struct {
	// ID is the unique identifier for the task (backend-specific format).
	ID string `json:"id" yaml:"id"`

	// Title is a short summary of the task.
	Title string `json:"title" yaml:"title"`

	// Description is the full description of the task (markdown).
	Description string `json:"description,omitempty" yaml:"description,omitempty"`

	// Status is the current status of the task.
	Status Status `json:"status" yaml:"status"`

	// Priority is the priority level of the task.
	Priority Priority `json:"priority" yaml:"priority"`

	// Assignee is the username or agent ID assigned to the task.
	Assignee string `json:"assignee,omitempty" yaml:"assignee,omitempty"`

	// Labels are tags/labels associated with the task.
	Labels []string `json:"labels,omitempty" yaml:"labels,omitempty"`

	// Created is the creation timestamp.
	Created time.Time `json:"created" yaml:"created"`

	// Updated is the last modified timestamp.
	Updated time.Time `json:"updated" yaml:"updated"`

	// URL is the web URL to view the task in a browser.
	URL string `json:"url,omitempty" yaml:"url,omitempty"`

	// Meta contains backend-specific fields.
	Meta map[string]any `json:"meta,omitempty" yaml:"meta,omitempty"`
}

// Comment represents a comment on a task.
type Comment struct {
	// ID is the unique identifier for the comment.
	ID string `json:"id" yaml:"id"`

	// Author is the username of the comment author.
	Author string `json:"author" yaml:"author"`

	// Body is the content of the comment (markdown).
	Body string `json:"body" yaml:"body"`

	// Created is the creation timestamp.
	Created time.Time `json:"created" yaml:"created"`
}

// TaskList represents a paginated list of tasks.
type TaskList struct {
	// Tasks is the list of tasks.
	Tasks []Task `json:"tasks"`

	// Count is the number of tasks in this response.
	Count int `json:"count"`

	// HasMore indicates if there are more tasks available.
	HasMore bool `json:"hasMore"`
}
