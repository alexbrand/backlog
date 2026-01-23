package local

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/alexbrand/backlog/internal/backend"
)

func TestNew(t *testing.T) {
	l := New()
	if l == nil {
		t.Fatal("New() returned nil")
	}
	if l.Name() != "local" {
		t.Errorf("Name() = %q, want %q", l.Name(), "local")
	}
	if l.Version() != "0.1.0" {
		t.Errorf("Version() = %q, want %q", l.Version(), "0.1.0")
	}
}

func TestConnect(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()
	backlogDir := filepath.Join(tmpDir, ".backlog")
	if err := os.MkdirAll(backlogDir, 0755); err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	l := New()
	cfg := backend.Config{
		Workspace: &WorkspaceConfig{Path: backlogDir},
		AgentID:   "test-agent",
	}

	if err := l.Connect(cfg); err != nil {
		t.Fatalf("Connect() error = %v", err)
	}

	if !l.connected {
		t.Error("Connect() did not set connected = true")
	}
}

func TestConnectInvalidConfig(t *testing.T) {
	l := New()
	cfg := backend.Config{
		Workspace: "invalid", // Not a *WorkspaceConfig
	}

	err := l.Connect(cfg)
	if err == nil {
		t.Fatal("Connect() with invalid config should return error")
	}
}

func TestConnectNonExistentDir(t *testing.T) {
	l := New()
	cfg := backend.Config{
		Workspace: &WorkspaceConfig{Path: "/nonexistent/path"},
	}

	err := l.Connect(cfg)
	if err == nil {
		t.Fatal("Connect() with non-existent dir should return error")
	}
}

func TestHealthCheck(t *testing.T) {
	tmpDir := t.TempDir()
	backlogDir := filepath.Join(tmpDir, ".backlog")
	if err := os.MkdirAll(backlogDir, 0755); err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	l := New()
	cfg := backend.Config{
		Workspace: &WorkspaceConfig{Path: backlogDir},
	}

	// Not connected
	status, _ := l.HealthCheck()
	if status.OK {
		t.Error("HealthCheck() should return OK=false when not connected")
	}

	// Connected
	if err := l.Connect(cfg); err != nil {
		t.Fatalf("Connect() error = %v", err)
	}

	status, _ = l.HealthCheck()
	if !status.OK {
		t.Errorf("HealthCheck() returned OK=false, message=%q", status.Message)
	}
}

func setupBacklog(t *testing.T) (*Local, string) {
	tmpDir := t.TempDir()
	backlogDir := filepath.Join(tmpDir, ".backlog")

	// Create all status directories
	for _, status := range []string{"backlog", "todo", "in-progress", "review", "done"} {
		if err := os.MkdirAll(filepath.Join(backlogDir, status), 0755); err != nil {
			t.Fatalf("failed to create status dir: %v", err)
		}
	}

	l := New()
	cfg := backend.Config{
		Workspace: &WorkspaceConfig{Path: backlogDir},
		AgentID:   "test-agent",
	}

	if err := l.Connect(cfg); err != nil {
		t.Fatalf("Connect() error = %v", err)
	}

	return l, backlogDir
}

func TestCreate(t *testing.T) {
	l, _ := setupBacklog(t)

	input := backend.TaskInput{
		Title:       "Test task",
		Description: "This is a test task",
		Priority:    backend.PriorityHigh,
		Labels:      []string{"test", "unit"},
	}

	task, err := l.Create(input)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if task.ID == "" {
		t.Error("Create() returned task with empty ID")
	}
	if task.Title != input.Title {
		t.Errorf("task.Title = %q, want %q", task.Title, input.Title)
	}
	if task.Status != backend.StatusBacklog {
		t.Errorf("task.Status = %q, want %q", task.Status, backend.StatusBacklog)
	}
	if task.Priority != input.Priority {
		t.Errorf("task.Priority = %q, want %q", task.Priority, input.Priority)
	}
}

func TestGet(t *testing.T) {
	l, _ := setupBacklog(t)

	// Create a task first
	created, err := l.Create(backend.TaskInput{
		Title:       "Test task",
		Description: "Description here",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Get the task
	task, err := l.Get(created.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if task.ID != created.ID {
		t.Errorf("task.ID = %q, want %q", task.ID, created.ID)
	}
	if task.Title != created.Title {
		t.Errorf("task.Title = %q, want %q", task.Title, created.Title)
	}
}

func TestGetNotFound(t *testing.T) {
	l, _ := setupBacklog(t)

	_, err := l.Get("nonexistent")
	if err == nil {
		t.Fatal("Get() for nonexistent task should return error")
	}
}

func TestList(t *testing.T) {
	l, _ := setupBacklog(t)

	// Create some tasks
	_, _ = l.Create(backend.TaskInput{Title: "Task 1", Priority: backend.PriorityHigh})
	_, _ = l.Create(backend.TaskInput{Title: "Task 2", Priority: backend.PriorityLow})
	_, _ = l.Create(backend.TaskInput{Title: "Task 3", Priority: backend.PriorityUrgent})

	list, err := l.List(backend.TaskFilters{})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if list.Count != 3 {
		t.Errorf("list.Count = %d, want 3", list.Count)
	}

	// Should be sorted by priority (urgent first)
	if list.Tasks[0].Priority != backend.PriorityUrgent {
		t.Errorf("first task priority = %q, want %q", list.Tasks[0].Priority, backend.PriorityUrgent)
	}
}

func TestListWithFilters(t *testing.T) {
	l, _ := setupBacklog(t)

	// Create tasks with different priorities
	_, _ = l.Create(backend.TaskInput{Title: "High 1", Priority: backend.PriorityHigh})
	_, _ = l.Create(backend.TaskInput{Title: "Low 1", Priority: backend.PriorityLow})
	_, _ = l.Create(backend.TaskInput{Title: "High 2", Priority: backend.PriorityHigh})

	// Filter by priority
	list, err := l.List(backend.TaskFilters{
		Priority: []backend.Priority{backend.PriorityHigh},
	})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if list.Count != 2 {
		t.Errorf("filtered list.Count = %d, want 2", list.Count)
	}
}

func TestListWithLimit(t *testing.T) {
	l, _ := setupBacklog(t)

	// Create 5 tasks
	for i := 0; i < 5; i++ {
		_, _ = l.Create(backend.TaskInput{Title: "Task"})
	}

	list, err := l.List(backend.TaskFilters{Limit: 2})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if list.Count != 2 {
		t.Errorf("list.Count = %d, want 2", list.Count)
	}
	if !list.HasMore {
		t.Error("list.HasMore = false, want true")
	}
}

func TestUpdate(t *testing.T) {
	l, _ := setupBacklog(t)

	created, _ := l.Create(backend.TaskInput{Title: "Original"})

	newTitle := "Updated"
	task, err := l.Update(created.ID, backend.TaskChanges{
		Title: &newTitle,
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	if task.Title != newTitle {
		t.Errorf("task.Title = %q, want %q", task.Title, newTitle)
	}
}

func TestUpdateLabels(t *testing.T) {
	l, _ := setupBacklog(t)

	created, _ := l.Create(backend.TaskInput{
		Title:  "Task",
		Labels: []string{"existing"},
	})

	task, err := l.Update(created.ID, backend.TaskChanges{
		AddLabels:    []string{"new1", "new2"},
		RemoveLabels: []string{"existing"},
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	if len(task.Labels) != 2 {
		t.Errorf("len(task.Labels) = %d, want 2", len(task.Labels))
	}
}

func TestMove(t *testing.T) {
	l, backlogDir := setupBacklog(t)

	created, _ := l.Create(backend.TaskInput{Title: "Task"})

	// Move to in-progress
	task, err := l.Move(created.ID, backend.StatusInProgress)
	if err != nil {
		t.Fatalf("Move() error = %v", err)
	}

	if task.Status != backend.StatusInProgress {
		t.Errorf("task.Status = %q, want %q", task.Status, backend.StatusInProgress)
	}

	// Verify file was moved
	oldPath := filepath.Join(backlogDir, "backlog", created.ID+"*.md")
	matches, _ := filepath.Glob(oldPath)
	if len(matches) > 0 {
		t.Error("old file still exists after Move()")
	}
}

func TestDelete(t *testing.T) {
	l, _ := setupBacklog(t)

	created, _ := l.Create(backend.TaskInput{Title: "Task"})

	err := l.Delete(created.ID)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// Verify task is gone
	_, err = l.Get(created.ID)
	if err == nil {
		t.Error("Get() after Delete() should return error")
	}
}

func TestAssign(t *testing.T) {
	l, _ := setupBacklog(t)

	created, _ := l.Create(backend.TaskInput{Title: "Task"})

	task, err := l.Assign(created.ID, "user1")
	if err != nil {
		t.Fatalf("Assign() error = %v", err)
	}

	if task.Assignee != "user1" {
		t.Errorf("task.Assignee = %q, want %q", task.Assignee, "user1")
	}
}

func TestUnassign(t *testing.T) {
	l, _ := setupBacklog(t)

	created, _ := l.Create(backend.TaskInput{Title: "Task", Assignee: "user1"})

	task, err := l.Unassign(created.ID)
	if err != nil {
		t.Fatalf("Unassign() error = %v", err)
	}

	if task.Assignee != "" {
		t.Errorf("task.Assignee = %q, want empty", task.Assignee)
	}
}

func TestAddComment(t *testing.T) {
	l, _ := setupBacklog(t)

	created, _ := l.Create(backend.TaskInput{Title: "Task"})

	comment, err := l.AddComment(created.ID, "This is a comment")
	if err != nil {
		t.Fatalf("AddComment() error = %v", err)
	}

	if comment.Body != "This is a comment" {
		t.Errorf("comment.Body = %q, want %q", comment.Body, "This is a comment")
	}
	if comment.Author != "test-agent" {
		t.Errorf("comment.Author = %q, want %q", comment.Author, "test-agent")
	}
}

func TestListComments(t *testing.T) {
	l, _ := setupBacklog(t)

	created, _ := l.Create(backend.TaskInput{Title: "Task"})
	_, _ = l.AddComment(created.ID, "Comment 1")
	_, _ = l.AddComment(created.ID, "Comment 2")

	comments, err := l.ListComments(created.ID)
	if err != nil {
		t.Fatalf("ListComments() error = %v", err)
	}

	if len(comments) != 2 {
		t.Errorf("len(comments) = %d, want 2", len(comments))
	}
}

func TestGenerateID(t *testing.T) {
	l, _ := setupBacklog(t)

	// First ID should be 001
	id1, err := l.generateID()
	if err != nil {
		t.Fatalf("generateID() error = %v", err)
	}
	if id1 != "001" {
		t.Errorf("first ID = %q, want %q", id1, "001")
	}

	// Create a task
	_, _ = l.Create(backend.TaskInput{Title: "Task"})

	// Second ID should be 002
	id2, err := l.generateID()
	if err != nil {
		t.Fatalf("generateID() error = %v", err)
	}
	if id2 != "002" {
		t.Errorf("second ID = %q, want %q", id2, "002")
	}
}

func TestSlugify(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Hello World", "hello-world"},
		{"Test_Task", "test-task"},
		{"Already-Slugified", "already-slugified"},
		{"Special!@#$Characters", "specialcharacters"},
		{"Multiple   Spaces", "multiple-spaces"},
		{"  Trim  ", "trim"},
		{"123 Numbers", "123-numbers"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := slugify(tt.input)
			if got != tt.want {
				t.Errorf("slugify(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestRegister(t *testing.T) {
	// Clear registry first
	backend.UnregisterAll()

	Register()

	if !backend.IsRegistered("local") {
		t.Error("Register() did not register 'local' backend")
	}

	// Verify we can get an instance
	b, err := backend.Get("local")
	if err != nil {
		t.Fatalf("backend.Get('local') error = %v", err)
	}
	if b == nil {
		t.Error("backend.Get('local') returned nil")
	}
	if b.Name() != "local" {
		t.Errorf("backend.Name() = %q, want %q", b.Name(), "local")
	}
}

func TestParseFrontmatter(t *testing.T) {
	content := []byte(`---
id: "001"
title: Test Task
priority: high
---

## Description

This is the description.
`)

	fm, body, err := parseFrontmatter(content)
	if err != nil {
		t.Fatalf("parseFrontmatter() error = %v", err)
	}

	if len(fm) == 0 {
		t.Error("frontmatter is empty")
	}
	if len(body) == 0 {
		t.Error("body is empty")
	}
}

func TestParseFrontmatterNoDelimiter(t *testing.T) {
	content := []byte(`No frontmatter here`)

	_, _, err := parseFrontmatter(content)
	if err == nil {
		t.Error("parseFrontmatter() should fail without delimiter")
	}
}

func TestReadWriteRoundtrip(t *testing.T) {
	l, _ := setupBacklog(t)

	original := &backend.Task{
		ID:          "001",
		Title:       "Test Task",
		Description: "This is a test description.\n\nWith multiple paragraphs.",
		Status:      backend.StatusTodo,
		Priority:    backend.PriorityHigh,
		Assignee:    "user1",
		Labels:      []string{"feature", "urgent"},
		Created:     time.Date(2025, 1, 15, 9, 0, 0, 0, time.UTC),
		Updated:     time.Date(2025, 1, 18, 14, 30, 0, 0, time.UTC),
	}

	// Write
	if err := l.writeTask(original); err != nil {
		t.Fatalf("writeTask() error = %v", err)
	}

	// Read
	task, err := l.Get(original.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	// Compare
	if task.ID != original.ID {
		t.Errorf("ID = %q, want %q", task.ID, original.ID)
	}
	if task.Title != original.Title {
		t.Errorf("Title = %q, want %q", task.Title, original.Title)
	}
	if task.Description != original.Description {
		t.Errorf("Description = %q, want %q", task.Description, original.Description)
	}
	if task.Status != original.Status {
		t.Errorf("Status = %q, want %q", task.Status, original.Status)
	}
	if task.Priority != original.Priority {
		t.Errorf("Priority = %q, want %q", task.Priority, original.Priority)
	}
	if task.Assignee != original.Assignee {
		t.Errorf("Assignee = %q, want %q", task.Assignee, original.Assignee)
	}
}
