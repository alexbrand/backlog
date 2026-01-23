package support

import (
	"path/filepath"
	"testing"
)

func TestTaskFileReader_ReadTaskFile(t *testing.T) {
	env, err := NewTestEnv()
	if err != nil {
		t.Fatalf("failed to create test env: %v", err)
	}
	defer env.Cleanup()

	// Create a task file
	if err := env.CreateBacklogDir(); err != nil {
		t.Fatalf("failed to create backlog dir: %v", err)
	}

	taskContent := `---
id: "001"
title: Implement auth flow
priority: high
assignee: alex
labels: [feature, auth, agent:claude-1]
agent_id: claude-1
created: 2025-01-15T09:00:00Z
updated: 2025-01-18T14:30:00Z
---

## Description

OAuth2 implementation details here.

More content.
`
	if err := env.CreateFile(".backlog/todo/001-implement-auth-flow.md", taskContent); err != nil {
		t.Fatalf("failed to create task file: %v", err)
	}

	reader := NewTaskFileReader(env.BacklogDir)
	task := reader.ReadTaskFile(filepath.Join(env.BacklogDir, "todo", "001-implement-auth-flow.md"))

	if !task.Valid() {
		t.Fatalf("task should be valid, got error: %s", task.Error())
	}

	tests := []struct {
		name     string
		got      string
		expected string
	}{
		{"ID", task.ID, "001"},
		{"Title", task.Title, "Implement auth flow"},
		{"Priority", task.Priority, "high"},
		{"Assignee", task.Assignee, "alex"},
		{"AgentID", task.AgentID, "claude-1"},
		{"Status", task.Status, "todo"},
		{"Created", task.Created, "2025-01-15T09:00:00Z"},
		{"Updated", task.Updated, "2025-01-18T14:30:00Z"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.expected {
				t.Errorf("expected %s to be %q, got %q", tt.name, tt.expected, tt.got)
			}
		})
	}

	// Check labels
	if len(task.Labels) != 3 {
		t.Errorf("expected 3 labels, got %d", len(task.Labels))
	}
	if !task.HasLabel("feature") {
		t.Error("expected task to have 'feature' label")
	}
	if !task.HasLabel("auth") {
		t.Error("expected task to have 'auth' label")
	}

	// Check description
	expectedDesc := "OAuth2 implementation details here.\n\nMore content."
	if task.Description != expectedDesc {
		t.Errorf("expected description %q, got %q", expectedDesc, task.Description)
	}
}

func TestTaskFileReader_ReadTask(t *testing.T) {
	env, err := NewTestEnv()
	if err != nil {
		t.Fatalf("failed to create test env: %v", err)
	}
	defer env.Cleanup()

	if err := env.CreateBacklogDir(); err != nil {
		t.Fatalf("failed to create backlog dir: %v", err)
	}

	// Create tasks in different status directories
	tasks := []struct {
		status   string
		filename string
		content  string
	}{
		{"backlog", "001-task-one.md", `---
id: "001"
title: Task one
priority: low
assignee: null
labels: []
---
`},
		{"in-progress", "002-task-two.md", `---
id: "002"
title: Task two
priority: high
assignee: alex
labels: [feature]
---
`},
		{"done", "003-task-three.md", `---
id: "003"
title: Task three
priority: medium
assignee: bob
labels: [bug]
---
`},
	}

	for _, task := range tasks {
		path := filepath.Join(".backlog", task.status, task.filename)
		if err := env.CreateFile(path, task.content); err != nil {
			t.Fatalf("failed to create task: %v", err)
		}
	}

	reader := NewTaskFileReader(env.BacklogDir)

	// Test finding task by ID
	t.Run("FindTaskByID", func(t *testing.T) {
		task := reader.ReadTask("002")
		if !task.Valid() {
			t.Fatalf("task should be valid, got error: %s", task.Error())
		}
		if task.ID != "002" {
			t.Errorf("expected ID 002, got %s", task.ID)
		}
		if task.Status != "in-progress" {
			t.Errorf("expected status in-progress, got %s", task.Status)
		}
	})

	t.Run("TaskNotFound", func(t *testing.T) {
		task := reader.ReadTask("999")
		if task.Valid() {
			t.Error("expected task to be invalid for non-existent ID")
		}
	})
}

func TestTaskFileReader_ListTasks(t *testing.T) {
	env, err := NewTestEnv()
	if err != nil {
		t.Fatalf("failed to create test env: %v", err)
	}
	defer env.Cleanup()

	if err := env.CreateBacklogDir(); err != nil {
		t.Fatalf("failed to create backlog dir: %v", err)
	}

	// Create multiple tasks
	env.CreateFile(".backlog/todo/001-task-one.md", `---
id: "001"
title: Task one
assignee: null
labels: []
---
`)
	env.CreateFile(".backlog/todo/002-task-two.md", `---
id: "002"
title: Task two
assignee: null
labels: []
---
`)
	env.CreateFile(".backlog/done/003-task-three.md", `---
id: "003"
title: Task three
assignee: null
labels: []
---
`)

	reader := NewTaskFileReader(env.BacklogDir)

	t.Run("ListAllTasks", func(t *testing.T) {
		tasks := reader.ListTasks()
		if len(tasks) != 3 {
			t.Errorf("expected 3 tasks, got %d", len(tasks))
		}
	})

	t.Run("ListTasksByStatus", func(t *testing.T) {
		tasks := reader.ListTasksByStatus("todo")
		if len(tasks) != 2 {
			t.Errorf("expected 2 tasks in todo, got %d", len(tasks))
		}

		tasks = reader.ListTasksByStatus("done")
		if len(tasks) != 1 {
			t.Errorf("expected 1 task in done, got %d", len(tasks))
		}
	})
}

func TestTaskFileReader_TaskExists(t *testing.T) {
	env, err := NewTestEnv()
	if err != nil {
		t.Fatalf("failed to create test env: %v", err)
	}
	defer env.Cleanup()

	if err := env.CreateBacklogDir(); err != nil {
		t.Fatalf("failed to create backlog dir: %v", err)
	}

	env.CreateFile(".backlog/todo/001-task.md", `---
id: "001"
title: Test task
assignee: null
labels: []
---
`)

	reader := NewTaskFileReader(env.BacklogDir)

	if !reader.TaskExists("001") {
		t.Error("expected task 001 to exist")
	}

	if reader.TaskExists("999") {
		t.Error("expected task 999 to not exist")
	}
}

func TestTaskFile_HasLabel(t *testing.T) {
	task := &TaskFile{
		Labels: []string{"feature", "auth", "agent:claude-1"},
	}

	tests := []struct {
		label    string
		expected bool
	}{
		{"feature", true},
		{"auth", true},
		{"agent:claude-1", true},
		{"bug", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.label, func(t *testing.T) {
			if got := task.HasLabel(tt.label); got != tt.expected {
				t.Errorf("HasLabel(%q) = %v, want %v", tt.label, got, tt.expected)
			}
		})
	}
}

func TestTaskFile_HasAgentLabel(t *testing.T) {
	task := &TaskFile{
		Labels: []string{"feature", "agent:claude-1", "priority:high"},
	}

	if !task.HasAgentLabel("agent") {
		t.Error("expected HasAgentLabel('agent') to be true")
	}

	if task.HasAgentLabel("worker") {
		t.Error("expected HasAgentLabel('worker') to be false")
	}
}

func TestTaskFile_GetAgentFromLabel(t *testing.T) {
	task := &TaskFile{
		Labels: []string{"feature", "agent:claude-1", "worker:bot-2"},
	}

	if got := task.GetAgentFromLabel("agent"); got != "claude-1" {
		t.Errorf("GetAgentFromLabel('agent') = %q, want 'claude-1'", got)
	}

	if got := task.GetAgentFromLabel("worker"); got != "bot-2" {
		t.Errorf("GetAgentFromLabel('worker') = %q, want 'bot-2'", got)
	}

	if got := task.GetAgentFromLabel("unknown"); got != "" {
		t.Errorf("GetAgentFromLabel('unknown') = %q, want ''", got)
	}
}

func TestTaskFile_IsAssigned(t *testing.T) {
	tests := []struct {
		name     string
		assignee string
		expected bool
	}{
		{"assigned", "alex", true},
		{"empty", "", false},
		{"null", "null", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := &TaskFile{Assignee: tt.assignee}
			if got := task.IsAssigned(); got != tt.expected {
				t.Errorf("IsAssigned() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestTaskFile_IsClaimed(t *testing.T) {
	tests := []struct {
		name     string
		agentID  string
		expected bool
	}{
		{"claimed", "claude-1", true},
		{"not claimed", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := &TaskFile{AgentID: tt.agentID}
			if got := task.IsClaimed(); got != tt.expected {
				t.Errorf("IsClaimed() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestTaskFile_GetField(t *testing.T) {
	task := &TaskFile{
		ID:          "001",
		Title:       "Test task",
		Status:      "todo",
		Priority:    "high",
		Assignee:    "alex",
		AgentID:     "claude-1",
		Created:     "2025-01-15",
		Updated:     "2025-01-18",
		Description: "Task description",
	}

	tests := []struct {
		field    string
		expected string
	}{
		{"id", "001"},
		{"ID", "001"},
		{"title", "Test task"},
		{"status", "todo"},
		{"priority", "high"},
		{"assignee", "alex"},
		{"agent_id", "claude-1"},
		{"agentid", "claude-1"},
		{"created", "2025-01-15"},
		{"updated", "2025-01-18"},
		{"description", "Task description"},
		{"unknown", ""},
	}

	for _, tt := range tests {
		t.Run(tt.field, func(t *testing.T) {
			if got := task.GetField(tt.field); got != tt.expected {
				t.Errorf("GetField(%q) = %q, want %q", tt.field, got, tt.expected)
			}
		})
	}
}

func TestParseFrontmatter(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		wantFM      string
		wantBody    string
		expectError bool
	}{
		{
			name: "standard frontmatter",
			content: `---
id: "001"
title: Test
---

Body content`,
			wantFM:   "id: \"001\"\ntitle: Test",
			wantBody: "\nBody content",
		},
		{
			name: "no body",
			content: `---
id: "001"
---`,
			wantFM:   "id: \"001\"",
			wantBody: "",
		},
		{
			name:        "no frontmatter",
			content:     "Just body content",
			expectError: true,
		},
		{
			name: "unclosed frontmatter",
			content: `---
id: "001"
No closing`,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fm, body, err := parseFrontmatter(tt.content)
			if tt.expectError {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if fm != tt.wantFM {
				t.Errorf("frontmatter = %q, want %q", fm, tt.wantFM)
			}
			if body != tt.wantBody {
				t.Errorf("body = %q, want %q", body, tt.wantBody)
			}
		})
	}
}

func TestExtractDescription(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		expected string
	}{
		{
			name:     "with description header",
			body:     "## Description\n\nTask description here.",
			expected: "Task description here.",
		},
		{
			name:     "without description header",
			body:     "Just content without header.",
			expected: "Just content without header.",
		},
		{
			name:     "empty body",
			body:     "",
			expected: "",
		},
		{
			name:     "only whitespace",
			body:     "   \n\n   ",
			expected: "",
		},
		{
			name:     "description with multiple paragraphs",
			body:     "## Description\n\nFirst paragraph.\n\nSecond paragraph.",
			expected: "First paragraph.\n\nSecond paragraph.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := extractDescription(tt.body); got != tt.expected {
				t.Errorf("extractDescription() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestTaskFile_InvalidFile(t *testing.T) {
	env, err := NewTestEnv()
	if err != nil {
		t.Fatalf("failed to create test env: %v", err)
	}
	defer env.Cleanup()

	if err := env.CreateBacklogDir(); err != nil {
		t.Fatalf("failed to create backlog dir: %v", err)
	}

	// Create invalid task file (bad YAML)
	env.CreateFile(".backlog/todo/bad-task.md", `---
id: "001
title: Missing quote
---
`)

	reader := NewTaskFileReader(env.BacklogDir)
	task := reader.ReadTaskFile(filepath.Join(env.BacklogDir, "todo", "bad-task.md"))

	if task.Valid() {
		t.Error("expected task to be invalid for bad YAML")
	}

	if task.Error() == "" {
		t.Error("expected error message to be set")
	}
}
