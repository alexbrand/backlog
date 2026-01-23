// Package local implements a filesystem-based backend for the backlog CLI.
// Tasks are stored as markdown files with YAML frontmatter in a directory structure
// organized by status.
package local

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/alexbrand/backlog/internal/backend"
)

const (
	// Version is the current version of the local backend.
	Version = "0.1.0"

	// Name is the name of the local backend.
	Name = "local"
)

// LockMode represents the locking strategy for multi-agent coordination.
type LockMode string

const (
	// LockModeFile uses file-based locking (default).
	LockModeFile LockMode = "file"
	// LockModeGit uses git-based locking (requires git_sync: true).
	LockModeGit LockMode = "git"
)

// WorkspaceConfig holds local backend-specific workspace configuration.
type WorkspaceConfig struct {
	// Path is the path to the .backlog directory.
	Path string
	// LockMode specifies the locking strategy: "file" (default) or "git".
	LockMode LockMode
	// GitSync enables automatic git commits after mutations.
	GitSync bool
}

// Local implements the Backend interface using the local filesystem.
type Local struct {
	path             string
	agentID          string
	agentLabelPrefix string
	lockMode         LockMode
	gitSync          bool
	connected        bool
}

// New creates a new Local backend instance.
func New() *Local {
	return &Local{}
}

// Name returns the name of the backend.
func (l *Local) Name() string {
	return Name
}

// Version returns the version of the backend.
func (l *Local) Version() string {
	return Version
}

// Connect initializes the backend with the given configuration.
func (l *Local) Connect(cfg backend.Config) error {
	wsCfg, ok := cfg.Workspace.(*WorkspaceConfig)
	if !ok {
		return errors.New("invalid workspace configuration for local backend")
	}

	// Resolve to absolute path for consistent git operations
	absPath, err := filepath.Abs(wsCfg.Path)
	if err != nil {
		return fmt.Errorf("failed to resolve absolute path: %w", err)
	}
	l.path = absPath
	l.agentID = cfg.AgentID
	l.agentLabelPrefix = cfg.AgentLabelPrefix
	if l.agentLabelPrefix == "" {
		l.agentLabelPrefix = "agent"
	}

	// Set lock mode, defaulting to file-based locking
	l.lockMode = wsCfg.LockMode
	if l.lockMode == "" {
		l.lockMode = LockModeFile
	}

	// Set git sync
	l.gitSync = wsCfg.GitSync

	// Verify the .backlog directory exists
	if _, err := os.Stat(l.path); os.IsNotExist(err) {
		return fmt.Errorf("backlog directory does not exist: %s", l.path)
	}

	l.connected = true
	return nil
}

// Disconnect closes the backend connection.
func (l *Local) Disconnect() error {
	l.connected = false
	return nil
}

// HealthCheck verifies the backend is accessible.
func (l *Local) HealthCheck() (backend.HealthStatus, error) {
	start := time.Now()

	if !l.connected {
		return backend.HealthStatus{
			OK:      false,
			Message: "not connected",
			Latency: time.Since(start),
		}, nil
	}

	// Check if the directory is accessible
	if _, err := os.Stat(l.path); err != nil {
		return backend.HealthStatus{
			OK:      false,
			Message: fmt.Sprintf("cannot access directory: %v", err),
			Latency: time.Since(start),
		}, nil
	}

	return backend.HealthStatus{
		OK:      true,
		Message: "ok",
		Latency: time.Since(start),
	}, nil
}

// List returns tasks matching the given filters.
func (l *Local) List(filters backend.TaskFilters) (*backend.TaskList, error) {
	if !l.connected {
		return nil, errors.New("not connected")
	}

	var tasks []backend.Task

	// Determine which status directories to scan
	statusDirs := []backend.Status{
		backend.StatusBacklog,
		backend.StatusTodo,
		backend.StatusInProgress,
		backend.StatusReview,
	}
	if filters.IncludeDone {
		statusDirs = append(statusDirs, backend.StatusDone)
	}

	// Filter by status if specified
	if len(filters.Status) > 0 {
		statusDirs = filters.Status
	}

	// Scan each status directory
	for _, status := range statusDirs {
		dirPath := filepath.Join(l.path, string(status))
		entries, err := os.ReadDir(dirPath)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read directory %s: %w", dirPath, err)
		}

		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
				continue
			}

			filePath := filepath.Join(dirPath, entry.Name())
			task, err := l.readTaskFile(filePath, status)
			if err != nil {
				// Skip files that can't be parsed
				continue
			}

			// Apply filters
			if !l.matchesFilters(task, filters) {
				continue
			}

			tasks = append(tasks, *task)
		}
	}

	// Sort by priority (urgent first) then by created (oldest first)
	sort.Slice(tasks, func(i, j int) bool {
		pi := priorityOrder(tasks[i].Priority)
		pj := priorityOrder(tasks[j].Priority)
		if pi != pj {
			return pi < pj
		}
		return tasks[i].Created.Before(tasks[j].Created)
	})

	// Apply limit
	hasMore := false
	if filters.Limit > 0 && len(tasks) > filters.Limit {
		tasks = tasks[:filters.Limit]
		hasMore = true
	}

	return &backend.TaskList{
		Tasks:   tasks,
		Count:   len(tasks),
		HasMore: hasMore,
	}, nil
}

// Get returns a single task by ID.
func (l *Local) Get(id string) (*backend.Task, error) {
	if !l.connected {
		return nil, errors.New("not connected")
	}

	return l.findTask(id)
}

// Create creates a new task and returns it.
func (l *Local) Create(input backend.TaskInput) (*backend.Task, error) {
	if !l.connected {
		return nil, errors.New("not connected")
	}

	// Generate a new ID
	id, err := l.generateID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate ID: %w", err)
	}

	// Set defaults
	status := input.Status
	if status == "" {
		status = backend.StatusBacklog
	}
	priority := input.Priority
	if priority == "" {
		priority = backend.PriorityNone
	}

	now := time.Now().UTC()
	task := &backend.Task{
		ID:          id,
		Title:       input.Title,
		Description: input.Description,
		Status:      status,
		Priority:    priority,
		Assignee:    input.Assignee,
		Labels:      input.Labels,
		Created:     now,
		Updated:     now,
	}

	// Write the task file
	if err := l.writeTask(task); err != nil {
		return nil, fmt.Errorf("failed to write task: %w", err)
	}

	// Git commit if enabled
	if err := l.gitCommit("add", id); err != nil {
		return nil, fmt.Errorf("failed to commit: %w", err)
	}

	return task, nil
}

// Update modifies an existing task and returns the updated task.
// This is the public method that commits changes to git if enabled.
func (l *Local) Update(id string, changes backend.TaskChanges) (*backend.Task, error) {
	task, err := l.updateInternal(id, changes)
	if err != nil {
		return nil, err
	}

	// Git commit if enabled
	if err := l.gitCommit("edit", id); err != nil {
		return nil, fmt.Errorf("failed to commit: %w", err)
	}

	return task, nil
}

// updateInternal modifies an existing task without git commit.
// Used internally by Claim, Release, etc. that handle their own commits.
func (l *Local) updateInternal(id string, changes backend.TaskChanges) (*backend.Task, error) {
	if !l.connected {
		return nil, errors.New("not connected")
	}

	// Find the old file path before applying changes
	oldFilePath, err := l.findTaskFile(id)
	if err != nil {
		return nil, err
	}

	task, err := l.readTaskFile(oldFilePath, l.statusFromPath(oldFilePath))
	if err != nil {
		return nil, err
	}

	// Apply changes
	if changes.Title != nil {
		task.Title = *changes.Title
	}
	if changes.Description != nil {
		task.Description = *changes.Description
	}
	if changes.Priority != nil {
		task.Priority = *changes.Priority
	}
	if changes.Assignee != nil {
		task.Assignee = *changes.Assignee
	}

	// Handle label changes
	if len(changes.AddLabels) > 0 {
		labelSet := make(map[string]bool)
		for _, l := range task.Labels {
			labelSet[l] = true
		}
		for _, l := range changes.AddLabels {
			labelSet[l] = true
		}
		task.Labels = make([]string, 0, len(labelSet))
		for l := range labelSet {
			task.Labels = append(task.Labels, l)
		}
		sort.Strings(task.Labels)
	}
	if len(changes.RemoveLabels) > 0 {
		removeSet := make(map[string]bool)
		for _, l := range changes.RemoveLabels {
			removeSet[l] = true
		}
		newLabels := make([]string, 0, len(task.Labels))
		for _, l := range task.Labels {
			if !removeSet[l] {
				newLabels = append(newLabels, l)
			}
		}
		task.Labels = newLabels
	}

	task.Updated = time.Now().UTC()

	// Write the updated task
	if err := l.writeTask(task); err != nil {
		return nil, fmt.Errorf("failed to write task: %w", err)
	}

	// Remove old file if the filename changed (due to title change)
	newFilename := generateFilename(task.ID, task.Title)
	newFilePath := filepath.Join(l.path, string(task.Status), newFilename)
	if oldFilePath != newFilePath {
		os.Remove(oldFilePath)
	}

	return task, nil
}

// Delete removes a task by ID.
func (l *Local) Delete(id string) error {
	if !l.connected {
		return errors.New("not connected")
	}

	filePath, err := l.findTaskFile(id)
	if err != nil {
		return err
	}

	if err := os.Remove(filePath); err != nil {
		return fmt.Errorf("failed to delete task: %w", err)
	}

	return nil
}

// Move transitions a task to a new status.
// This is the public method that commits changes to git if enabled.
func (l *Local) Move(id string, status backend.Status) (*backend.Task, error) {
	task, err := l.moveInternal(id, status)
	if err != nil {
		return nil, err
	}

	// Git commit if enabled
	if err := l.gitCommit("move", id); err != nil {
		return nil, fmt.Errorf("failed to commit: %w", err)
	}

	return task, nil
}

// moveInternal transitions a task to a new status without git commit.
// Used internally by Claim, Release, etc. that handle their own commits.
func (l *Local) moveInternal(id string, status backend.Status) (*backend.Task, error) {
	if !l.connected {
		return nil, errors.New("not connected")
	}

	if !status.IsValid() {
		return nil, fmt.Errorf("invalid status: %s", status)
	}

	task, err := l.findTask(id)
	if err != nil {
		return nil, err
	}

	oldStatus := task.Status
	task.Status = status
	task.Updated = time.Now().UTC()

	// If status changed, we need to move the file
	if oldStatus != status {
		// Remove old file
		oldPath, err := l.findTaskFile(id)
		if err != nil {
			return nil, err
		}
		if err := os.Remove(oldPath); err != nil {
			return nil, fmt.Errorf("failed to remove old task file: %w", err)
		}
	}

	// Write to new location
	if err := l.writeTask(task); err != nil {
		return nil, fmt.Errorf("failed to write task: %w", err)
	}

	return task, nil
}

// Assign assigns a task to a user.
// Uses updateInternal to avoid duplicate git commits when used in compound operations.
func (l *Local) Assign(id string, assignee string) (*backend.Task, error) {
	return l.updateInternal(id, backend.TaskChanges{Assignee: &assignee})
}

// Unassign removes the assignee from a task.
// Uses updateInternal to avoid duplicate git commits when used in compound operations.
func (l *Local) Unassign(id string) (*backend.Task, error) {
	empty := ""
	return l.updateInternal(id, backend.TaskChanges{Assignee: &empty})
}

// ListComments returns all comments for a task.
func (l *Local) ListComments(id string) ([]backend.Comment, error) {
	if !l.connected {
		return nil, errors.New("not connected")
	}

	task, err := l.findTask(id)
	if err != nil {
		return nil, err
	}

	// Comments are stored in the task's meta field
	comments, ok := task.Meta["comments"].([]backend.Comment)
	if !ok {
		return []backend.Comment{}, nil
	}

	return comments, nil
}

// AddComment adds a comment to a task.
func (l *Local) AddComment(id string, body string) (*backend.Comment, error) {
	if !l.connected {
		return nil, errors.New("not connected")
	}

	task, err := l.findTask(id)
	if err != nil {
		return nil, err
	}

	// Initialize meta if needed
	if task.Meta == nil {
		task.Meta = make(map[string]any)
	}

	// Get existing comments
	var comments []backend.Comment
	if existing, ok := task.Meta["comments"].([]backend.Comment); ok {
		comments = existing
	}

	// Generate comment ID
	commentID := fmt.Sprintf("%s-c%d", id, len(comments)+1)
	comment := backend.Comment{
		ID:      commentID,
		Author:  l.agentID,
		Body:    body,
		Created: time.Now().UTC(),
	}

	comments = append(comments, comment)
	task.Meta["comments"] = comments
	task.Updated = time.Now().UTC()

	if err := l.writeTask(task); err != nil {
		return nil, fmt.Errorf("failed to write task: %w", err)
	}

	// Git commit if enabled
	if err := l.gitCommit("comment", id); err != nil {
		return nil, fmt.Errorf("failed to commit: %w", err)
	}

	return &comment, nil
}

// Helper functions

// findTask finds a task by ID across all status directories.
func (l *Local) findTask(id string) (*backend.Task, error) {
	filePath, err := l.findTaskFile(id)
	if err != nil {
		return nil, err
	}

	// Determine status from path
	status := l.statusFromPath(filePath)

	return l.readTaskFile(filePath, status)
}

// findTaskFile finds the file path for a task by ID.
func (l *Local) findTaskFile(id string) (string, error) {
	statuses := []backend.Status{
		backend.StatusBacklog,
		backend.StatusTodo,
		backend.StatusInProgress,
		backend.StatusReview,
		backend.StatusDone,
	}

	for _, status := range statuses {
		dirPath := filepath.Join(l.path, string(status))
		entries, err := os.ReadDir(dirPath)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			continue
		}

		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
				continue
			}

			// Check if filename starts with the ID
			baseName := strings.TrimSuffix(entry.Name(), ".md")
			if baseName == id || strings.HasPrefix(baseName, id+"-") {
				return filepath.Join(dirPath, entry.Name()), nil
			}
		}
	}

	return "", fmt.Errorf("task not found: %s", id)
}

// statusFromPath extracts the status from a file path.
func (l *Local) statusFromPath(filePath string) backend.Status {
	dir := filepath.Base(filepath.Dir(filePath))
	return backend.Status(dir)
}

// generateID generates a new unique task ID.
func (l *Local) generateID() (string, error) {
	maxID := 0

	statuses := []backend.Status{
		backend.StatusBacklog,
		backend.StatusTodo,
		backend.StatusInProgress,
		backend.StatusReview,
		backend.StatusDone,
	}

	for _, status := range statuses {
		dirPath := filepath.Join(l.path, string(status))
		entries, err := os.ReadDir(dirPath)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			continue
		}

		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
				continue
			}

			baseName := strings.TrimSuffix(entry.Name(), ".md")
			// Extract ID from filename (format: "001-title" or just "001")
			parts := strings.SplitN(baseName, "-", 2)
			if len(parts) > 0 {
				if num, err := strconv.Atoi(parts[0]); err == nil && num > maxID {
					maxID = num
				}
			}
		}
	}

	return fmt.Sprintf("%03d", maxID+1), nil
}

// matchesFilters checks if a task matches the given filters.
func (l *Local) matchesFilters(task *backend.Task, filters backend.TaskFilters) bool {
	// Priority filter
	if len(filters.Priority) > 0 {
		found := false
		for _, p := range filters.Priority {
			if task.Priority == p {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Assignee filter
	if filters.Assignee != "" {
		if filters.Assignee == "unassigned" {
			if task.Assignee != "" {
				return false
			}
		} else if filters.Assignee == "@me" {
			if task.Assignee != l.agentID {
				return false
			}
		} else if task.Assignee != filters.Assignee {
			return false
		}
	}

	// Labels filter (task must have all specified labels)
	if len(filters.Labels) > 0 {
		taskLabels := make(map[string]bool)
		for _, label := range task.Labels {
			taskLabels[label] = true
		}
		for _, required := range filters.Labels {
			if !taskLabels[required] {
				return false
			}
		}
	}

	return true
}

// priorityOrder returns a numeric order for priorities (lower = higher priority).
func priorityOrder(p backend.Priority) int {
	switch p {
	case backend.PriorityUrgent:
		return 0
	case backend.PriorityHigh:
		return 1
	case backend.PriorityMedium:
		return 2
	case backend.PriorityLow:
		return 3
	case backend.PriorityNone:
		return 4
	default:
		return 5
	}
}

// Claim claims a task for the current agent.
// Implements the backend.Claimer interface.
func (l *Local) Claim(id string, agentID string) (*backend.ClaimResult, error) {
	if !l.connected {
		return nil, errors.New("not connected")
	}

	// Use the provided agentID, or fall back to the configured one
	if agentID == "" {
		agentID = l.agentID
	} else {
		// Update l.agentID for use in gitCommit message
		l.agentID = agentID
	}

	// Git mode: pull → check → claim → commit → push
	if l.lockMode == LockModeGit {
		return l.claimWithGit(id, agentID)
	}

	// File mode: use file-based locking
	return l.claimWithFileLock(id, agentID)
}

// claimWithGit implements git-based claim coordination.
// Flow: pull latest → check agent labels → make changes → commit → push
// Push failures indicate another agent claimed the task first (exit code 2).
func (l *Local) claimWithGit(id string, agentID string) (*backend.ClaimResult, error) {
	// Pull latest changes from remote
	if err := l.gitPull(); err != nil {
		return nil, fmt.Errorf("failed to pull: %w", err)
	}

	// Find the task (re-read after pull to get latest state)
	task, err := l.findTask(id)
	if err != nil {
		return nil, err
	}

	// Check if task is already claimed by checking agent labels
	existingAgentLabels := l.findAgentLabels(task.Labels)
	if len(existingAgentLabels) > 0 {
		// Extract agent ID from label (format: "agent:agent-id")
		claimedByAgent := strings.TrimPrefix(existingAgentLabels[0], l.agentLabelPrefix+":")
		if claimedByAgent == agentID {
			return &backend.ClaimResult{
				Task:         task,
				AlreadyOwned: true,
			}, nil
		}
		// Claimed by another agent
		return nil, &ClaimConflictError{
			TaskID:       id,
			ClaimedBy:    claimedByAgent,
			CurrentAgent: agentID,
		}
	}

	// Clean up any stale file locks (git mode doesn't use them)
	l.removeLock(id)

	// Remove any existing agent labels, add the new one, and set assignee to agent ID
	agentLabel := fmt.Sprintf("%s:%s", l.agentLabelPrefix, agentID)
	changes := backend.TaskChanges{
		RemoveLabels: existingAgentLabels,
		AddLabels:    []string{agentLabel},
		Assignee:     &agentID,
	}

	// Apply label changes
	task, err = l.updateInternal(id, changes)
	if err != nil {
		return nil, fmt.Errorf("failed to update task: %w", err)
	}

	// Move to in-progress
	task, err = l.moveInternal(id, backend.StatusInProgress)
	if err != nil {
		return nil, fmt.Errorf("failed to move task: %w", err)
	}

	// Commit the changes
	if err := l.gitCommit("claim", id); err != nil {
		return nil, fmt.Errorf("failed to commit: %w", err)
	}

	// Push to remote - this is the coordination point
	// If push fails with non-fast-forward, another agent claimed first
	if err := l.gitPush(); err != nil {
		// Check if it's a push conflict (another agent beat us)
		if _, isConflict := err.(*GitPushConflictError); isConflict {
			return nil, &ClaimConflictError{
				TaskID:       id,
				ClaimedBy:    "another agent (push conflict)",
				CurrentAgent: agentID,
			}
		}
		return nil, fmt.Errorf("failed to push: %w", err)
	}

	return &backend.ClaimResult{
		Task:         task,
		AlreadyOwned: false,
	}, nil
}

// claimWithFileLock implements file-based claim coordination.
func (l *Local) claimWithFileLock(id string, agentID string) (*backend.ClaimResult, error) {
	// Find the task
	task, err := l.findTask(id)
	if err != nil {
		return nil, err
	}

	// Check for existing lock
	existingLock, err := l.readLock(id)
	if err != nil {
		return nil, fmt.Errorf("failed to read lock: %w", err)
	}

	// Check if task is already claimed
	if existingLock != nil && existingLock.isActive() {
		// Check if claimed by the same agent
		if existingLock.Agent == agentID {
			return &backend.ClaimResult{
				Task:         task,
				AlreadyOwned: true,
			}, nil
		}
		// Claimed by another agent
		return nil, &ClaimConflictError{
			TaskID:       id,
			ClaimedBy:    existingLock.Agent,
			CurrentAgent: agentID,
		}
	}

	// Create new lock
	now := time.Now().UTC()
	lock := &LockFile{
		Agent:     agentID,
		ClaimedAt: now,
		ExpiresAt: now.Add(DefaultLockTTL),
	}

	if err := l.writeLock(id, lock); err != nil {
		return nil, fmt.Errorf("failed to write lock: %w", err)
	}

	// Remove any existing agent labels, add the new one, and set assignee to agent ID
	agentLabel := fmt.Sprintf("%s:%s", l.agentLabelPrefix, agentID)
	changes := backend.TaskChanges{
		RemoveLabels: l.findAgentLabels(task.Labels),
		AddLabels:    []string{agentLabel},
		Assignee:     &agentID,
	}

	// Move to in-progress and apply label changes
	task, err = l.updateInternal(id, changes)
	if err != nil {
		// Try to remove the lock if we fail to update the task
		l.removeLock(id)
		return nil, fmt.Errorf("failed to update task: %w", err)
	}

	task, err = l.moveInternal(id, backend.StatusInProgress)
	if err != nil {
		// Try to remove the lock if we fail to move the task
		l.removeLock(id)
		return nil, fmt.Errorf("failed to move task: %w", err)
	}

	// Git commit if enabled
	if err := l.gitCommit("claim", id); err != nil {
		return nil, fmt.Errorf("failed to commit: %w", err)
	}

	return &backend.ClaimResult{
		Task:         task,
		AlreadyOwned: false,
	}, nil
}

// Release releases a claimed task back to todo status.
// Implements the backend.Claimer interface.
func (l *Local) Release(id string) error {
	if !l.connected {
		return errors.New("not connected")
	}

	// Git mode: pull → release → commit → push
	if l.lockMode == LockModeGit {
		return l.releaseWithGit(id)
	}

	// File mode: use file-based locking
	return l.releaseWithFileLock(id)
}

// releaseWithGit implements git-based release coordination.
// Flow: pull latest → make changes → commit → push
func (l *Local) releaseWithGit(id string) error {
	// Pull latest changes from remote
	if err := l.gitPull(); err != nil {
		return fmt.Errorf("failed to pull: %w", err)
	}

	// Find the task (re-read after pull to get latest state)
	task, err := l.findTask(id)
	if err != nil {
		return err
	}

	// Clean up any stale file locks (git mode doesn't use them)
	l.removeLock(id)

	// Remove agent labels
	agentLabels := l.findAgentLabels(task.Labels)
	if len(agentLabels) > 0 {
		_, err = l.updateInternal(id, backend.TaskChanges{
			RemoveLabels: agentLabels,
		})
		if err != nil {
			return fmt.Errorf("failed to remove agent labels: %w", err)
		}
	}

	// Unassign the task (uses updateInternal internally)
	_, err = l.Unassign(id)
	if err != nil {
		return fmt.Errorf("failed to unassign task: %w", err)
	}

	// Move to todo
	_, err = l.moveInternal(id, backend.StatusTodo)
	if err != nil {
		return fmt.Errorf("failed to move task to todo: %w", err)
	}

	// Commit the changes
	if err := l.gitCommit("release", id); err != nil {
		return fmt.Errorf("failed to commit: %w", err)
	}

	// Push to remote
	if err := l.gitPush(); err != nil {
		// For release, a push conflict is still an error but not the same as claim conflict
		if _, isConflict := err.(*GitPushConflictError); isConflict {
			return fmt.Errorf("failed to push release: remote has conflicting changes")
		}
		return fmt.Errorf("failed to push: %w", err)
	}

	return nil
}

// releaseWithFileLock implements file-based release coordination.
func (l *Local) releaseWithFileLock(id string) error {
	// Find the task
	task, err := l.findTask(id)
	if err != nil {
		return err
	}

	// Remove the lock file
	if err := l.removeLock(id); err != nil {
		return fmt.Errorf("failed to remove lock: %w", err)
	}

	// Remove agent labels
	agentLabels := l.findAgentLabels(task.Labels)
	if len(agentLabels) > 0 {
		_, err = l.updateInternal(id, backend.TaskChanges{
			RemoveLabels: agentLabels,
		})
		if err != nil {
			return fmt.Errorf("failed to remove agent labels: %w", err)
		}
	}

	// Unassign the task (uses updateInternal internally)
	_, err = l.Unassign(id)
	if err != nil {
		return fmt.Errorf("failed to unassign task: %w", err)
	}

	// Move to todo
	_, err = l.moveInternal(id, backend.StatusTodo)
	if err != nil {
		return fmt.Errorf("failed to move task to todo: %w", err)
	}

	// Git commit if enabled
	if err := l.gitCommit("release", id); err != nil {
		return fmt.Errorf("failed to commit: %w", err)
	}

	return nil
}

// findAgentLabels returns all labels that match the agent label pattern.
func (l *Local) findAgentLabels(labels []string) []string {
	var agentLabels []string
	prefix := l.agentLabelPrefix + ":"
	for _, label := range labels {
		if strings.HasPrefix(label, prefix) {
			agentLabels = append(agentLabels, label)
		}
	}
	return agentLabels
}

// ClaimConflictError represents an error when a task is already claimed by another agent.
type ClaimConflictError struct {
	TaskID       string
	ClaimedBy    string
	CurrentAgent string
}

func (e *ClaimConflictError) Error() string {
	return fmt.Sprintf("task %s is already claimed by agent %s", e.TaskID, e.ClaimedBy)
}

// gitCommit creates a git commit with the given message if git sync is enabled.
// The action parameter is one of: add, edit, move, claim, release, comment.
// The taskID is the ID of the task being modified.
// The agentID is included in the commit message for claim/release operations.
func (l *Local) gitCommit(action, taskID string) error {
	if !l.gitSync {
		return nil
	}

	// Build commit message
	var message string
	if action == "claim" || action == "release" {
		message = fmt.Sprintf("%s: %s [agent:%s]", action, taskID, l.agentID)
	} else {
		message = fmt.Sprintf("%s: %s", action, taskID)
	}

	// Get the parent directory of the .backlog folder to run git commands
	// l.path is absolute, so we get its parent for the git repo root
	gitDir := filepath.Dir(l.path)

	// Stage all changes in the .backlog directory
	addCmd := exec.Command("git", "add", l.path)
	addCmd.Dir = gitDir
	if output, err := addCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git add failed: %w\n%s", err, output)
	}

	// Commit the changes
	commitCmd := exec.Command("git", "commit", "-m", message)
	commitCmd.Dir = gitDir
	if output, err := commitCmd.CombinedOutput(); err != nil {
		// If nothing to commit, that's OK
		if strings.Contains(string(output), "nothing to commit") {
			return nil
		}
		return fmt.Errorf("git commit failed: %w\n%s", err, output)
	}

	return nil
}

// gitPull pulls changes from the remote repository.
// Returns an error if pull fails or has conflicts.
func (l *Local) gitPull() error {
	gitDir := filepath.Dir(l.path)

	pullCmd := exec.Command("git", "pull")
	pullCmd.Dir = gitDir
	pullOutput, err := pullCmd.CombinedOutput()
	if err != nil {
		outputStr := string(pullOutput)
		// Check for conflicts
		if strings.Contains(outputStr, "CONFLICT") || strings.Contains(outputStr, "conflict") {
			return &SyncConflictError{
				Operation: "pull",
				Message:   outputStr,
			}
		}
		// Check if it's just "already up to date"
		if !strings.Contains(outputStr, "Already up to date") &&
			!strings.Contains(outputStr, "Already up-to-date") {
			return fmt.Errorf("git pull failed: %w\n%s", err, outputStr)
		}
	}
	return nil
}

// gitPush pushes changes to the remote repository.
// Returns a ClaimConflictError if push is rejected (for use with git-based claims).
func (l *Local) gitPush() error {
	gitDir := filepath.Dir(l.path)

	pushCmd := exec.Command("git", "push")
	pushCmd.Dir = gitDir
	pushOutput, err := pushCmd.CombinedOutput()
	if err != nil {
		outputStr := string(pushOutput)
		// Check for rejection (conflict)
		if strings.Contains(outputStr, "rejected") ||
			strings.Contains(outputStr, "non-fast-forward") {
			return &GitPushConflictError{
				Message: "push rejected - remote has changes that conflict with local changes",
			}
		}
		// Check for remote connectivity issues
		if strings.Contains(outputStr, "Could not read from remote") ||
			strings.Contains(outputStr, "unable to access") ||
			strings.Contains(outputStr, "fatal: unable to") ||
			strings.Contains(outputStr, "Connection refused") {
			return fmt.Errorf("git push failed: remote unreachable\n%s", outputStr)
		}
		// Check if there's nothing to push (not an error)
		if !strings.Contains(outputStr, "Everything up-to-date") &&
			!strings.Contains(outputStr, "nothing to commit") {
			return fmt.Errorf("git push failed: %w\n%s", err, outputStr)
		}
	}
	return nil
}

// GitPushConflictError represents a conflict when pushing to remote.
// This is returned when a git push is rejected due to non-fast-forward updates,
// indicating another agent has pushed changes since we last pulled.
type GitPushConflictError struct {
	Message string
}

func (e *GitPushConflictError) Error() string {
	return fmt.Sprintf("git push conflict: %s", e.Message)
}

// Sync synchronizes the local backlog with a remote git repository.
// Implements the backend.Syncer interface.
func (l *Local) Sync(force bool) (*backend.SyncResult, error) {
	if !l.connected {
		return nil, errors.New("not connected")
	}

	// Get the parent directory of the .backlog folder to run git commands
	gitDir := filepath.Dir(l.path)

	result := &backend.SyncResult{}

	// First, pull changes from remote
	pullArgs := []string{"pull"}
	if force {
		pullArgs = append(pullArgs, "--rebase")
	}
	pullCmd := exec.Command("git", pullArgs...)
	pullCmd.Dir = gitDir
	pullOutput, err := pullCmd.CombinedOutput()
	if err != nil {
		// Check for conflicts
		outputStr := string(pullOutput)
		if strings.Contains(outputStr, "CONFLICT") || strings.Contains(outputStr, "conflict") {
			return nil, &SyncConflictError{
				Operation: "pull",
				Message:   outputStr,
			}
		}
		// Check if it's just "already up to date"
		if !strings.Contains(outputStr, "Already up to date") &&
			!strings.Contains(outputStr, "Already up-to-date") {
			return nil, fmt.Errorf("git pull failed: %w\n%s", err, outputStr)
		}
	}

	// Parse pull output to count changes
	pullOutputStr := string(pullOutput)
	if strings.Contains(pullOutputStr, "files changed") ||
		strings.Contains(pullOutputStr, "file changed") {
		// Some changes were pulled
		result.Updated = 1 // Simplified: we don't parse exact counts
	}

	// Then, push local changes to remote
	pushArgs := []string{"push"}
	if force {
		pushArgs = append(pushArgs, "--force")
	}
	pushCmd := exec.Command("git", pushArgs...)
	pushCmd.Dir = gitDir
	pushOutput, err := pushCmd.CombinedOutput()
	if err != nil {
		outputStr := string(pushOutput)
		// Check for conflicts or rejection
		if strings.Contains(outputStr, "rejected") ||
			strings.Contains(outputStr, "non-fast-forward") {
			return nil, &SyncConflictError{
				Operation: "push",
				Message:   "push rejected - remote has changes. Use --force to overwrite or pull first",
			}
		}
		// Check if there's nothing to push
		if !strings.Contains(outputStr, "Everything up-to-date") &&
			!strings.Contains(outputStr, "nothing to commit") {
			return nil, fmt.Errorf("git push failed: %w\n%s", err, outputStr)
		}
	}

	// Parse push output
	pushOutputStr := string(pushOutput)
	if !strings.Contains(pushOutputStr, "Everything up-to-date") &&
		!strings.Contains(pushOutputStr, "up-to-date") {
		result.Pushed = 1 // Simplified: we don't parse exact counts
	}

	return result, nil
}

// SyncConflictError represents a conflict during sync operation.
type SyncConflictError struct {
	Operation string
	Message   string
}

func (e *SyncConflictError) Error() string {
	return fmt.Sprintf("sync conflict during %s: %s", e.Operation, e.Message)
}

// Register registers the local backend with the registry.
func Register() {
	backend.Register(Name, func() backend.Backend {
		return New()
	})
}
