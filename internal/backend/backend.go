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
	// Note: Not using omitempty so empty string is explicitly shown as "" in JSON
	Assignee string `json:"assignee" yaml:"assignee,omitempty"`

	// Labels are tags/labels associated with the task.
	Labels []string `json:"labels,omitempty" yaml:"labels,omitempty"`

	// Created is the creation timestamp.
	Created time.Time `json:"created" yaml:"created"`

	// Updated is the last modified timestamp.
	Updated time.Time `json:"updated" yaml:"updated"`

	// URL is the web URL to view the task in a browser.
	URL string `json:"url,omitempty" yaml:"url,omitempty"`

	// SortOrder is the explicit sort position of the task (lower = higher in list).
	// Zero value means no explicit ordering has been set.
	SortOrder float64 `json:"sort_order,omitempty" yaml:"sort_order,omitempty"`

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

// TaskFilters specifies filtering options for listing tasks.
type TaskFilters struct {
	// Status filters by task status.
	Status []Status

	// Priority filters by priority level.
	Priority []Priority

	// Assignee filters by assignee (use "@me" for current user, "unassigned" for no assignee).
	Assignee string

	// Labels filters by labels (task must have all specified labels).
	Labels []string

	// Limit is the maximum number of tasks to return.
	Limit int

	// IncludeDone includes tasks with done status (excluded by default).
	IncludeDone bool
}

// TaskInput specifies fields for creating a new task.
type TaskInput struct {
	// Title is the task title (required).
	Title string

	// Description is the task description (optional).
	Description string

	// Status is the initial status (defaults to backlog).
	Status Status

	// Priority is the priority level (defaults to none).
	Priority Priority

	// Labels are initial labels for the task.
	Labels []string

	// Assignee is the initial assignee (optional).
	Assignee string
}

// TaskChanges specifies fields to update on an existing task.
type TaskChanges struct {
	// Title is the new title (nil means no change).
	Title *string

	// Description is the new description (nil means no change).
	Description *string

	// Priority is the new priority (nil means no change).
	Priority *Priority

	// Assignee is the new assignee (nil means no change, empty string means unassign).
	Assignee *string

	// AddLabels are labels to add.
	AddLabels []string

	// RemoveLabels are labels to remove.
	RemoveLabels []string
}

// HealthStatus represents the health of a backend connection.
type HealthStatus struct {
	// OK indicates whether the backend is healthy.
	OK bool

	// Message provides additional details about the health status.
	Message string

	// Latency is the response time of the health check.
	Latency time.Duration
}

// ClaimResult represents the result of a claim operation.
type ClaimResult struct {
	// Task is the claimed task.
	Task *Task

	// AlreadyOwned indicates if the task was already claimed by this agent.
	AlreadyOwned bool
}

// SyncResult represents the result of a sync operation.
type SyncResult struct {
	// Created is the number of tasks created locally.
	Created int

	// Updated is the number of tasks updated locally.
	Updated int

	// Deleted is the number of tasks deleted locally.
	Deleted int

	// Pushed is the number of local changes pushed to remote.
	Pushed int

	// Conflicts is the number of conflicts encountered.
	Conflicts int
}

// Config holds backend-specific configuration.
type Config struct {
	// Workspace is the workspace configuration.
	Workspace any

	// AgentID is the identifier for this agent.
	AgentID string

	// AgentLabelPrefix is the prefix for agent labels (e.g., "agent").
	AgentLabelPrefix string
}

// Backend defines the interface that all backlog backends must implement.
type Backend interface {
	// Name returns the name of the backend (e.g., "github", "linear", "local").
	Name() string

	// Version returns the version of the backend implementation.
	Version() string

	// Connect initializes the backend with the given configuration.
	Connect(cfg Config) error

	// Disconnect closes the backend connection and cleans up resources.
	Disconnect() error

	// HealthCheck verifies the backend is accessible and working.
	HealthCheck() (HealthStatus, error)

	// List returns tasks matching the given filters.
	List(filters TaskFilters) (*TaskList, error)

	// Get returns a single task by ID.
	Get(id string) (*Task, error)

	// Create creates a new task and returns it.
	Create(input TaskInput) (*Task, error)

	// Update modifies an existing task and returns the updated task.
	Update(id string, changes TaskChanges) (*Task, error)

	// Delete removes a task by ID.
	Delete(id string) error

	// Move transitions a task to a new status.
	Move(id string, status Status) (*Task, error)

	// Assign assigns a task to a user.
	Assign(id string, assignee string) (*Task, error)

	// Unassign removes the assignee from a task.
	Unassign(id string) (*Task, error)

	// ListComments returns all comments for a task.
	ListComments(id string) ([]Comment, error)

	// AddComment adds a comment to a task.
	AddComment(id string, body string) (*Comment, error)
}

// Claimer is an optional interface for backends that support agent claim/release.
type Claimer interface {
	// Claim claims a task for an agent. Returns ClaimResult with the task.
	// Returns an error with exit code 2 if the task is already claimed by another agent.
	Claim(id string, agentID string) (*ClaimResult, error)

	// Release releases a claimed task back to todo status.
	Release(id string) error
}

// Syncer is an optional interface for backends that support sync operations.
type Syncer interface {
	// Sync synchronizes local state with remote.
	Sync(force bool) (*SyncResult, error)
}

// ReorderPosition specifies where to place a task in the sort order.
// Exactly one field should be set.
type ReorderPosition struct {
	// BeforeID places the task immediately before the task with this ID.
	BeforeID string

	// AfterID places the task immediately after the task with this ID.
	AfterID string

	// First moves the task to the top of its group.
	First bool

	// Last moves the task to the bottom of its group.
	Last bool
}

// Reorderer is an optional interface for backends that support explicit task reordering.
type Reorderer interface {
	// Reorder changes the sort position of a task.
	// Returns the updated task with its new SortOrder.
	Reorder(id string, position ReorderPosition) (*Task, error)
}
