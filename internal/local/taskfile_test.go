package local

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/alexbrand/backlog/internal/backend"
)

func TestParseFrontmatterValidContent(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		wantFM      string
		wantBody    string
		wantErr     bool
		errContains string
	}{
		{
			name: "basic frontmatter with body",
			content: `---
id: "001"
title: Test Task
priority: high
---

## Description

This is the description.
`,
			wantFM:   "id: \"001\"\ntitle: Test Task\npriority: high\n",
			wantBody: "\n## Description\n\nThis is the description.\n",
			wantErr:  false,
		},
		{
			name: "frontmatter only no body",
			content: `---
id: "002"
title: Empty Body
---
`,
			wantFM:   "id: \"002\"\ntitle: Empty Body\n",
			wantBody: "",
			wantErr:  false,
		},
		{
			name: "frontmatter with all fields",
			content: `---
id: "003"
title: Full Task
priority: urgent
assignee: user1
labels:
  - bug
  - critical
created: 2025-01-15T09:00:00Z
updated: 2025-01-18T14:30:00Z
---

Body content here.
`,
			wantFM: `id: "003"
title: Full Task
priority: urgent
assignee: user1
labels:
  - bug
  - critical
created: 2025-01-15T09:00:00Z
updated: 2025-01-18T14:30:00Z
`,
			wantBody: "\nBody content here.\n",
			wantErr:  false,
		},
		{
			name: "frontmatter with Windows line endings",
			content: "---\r\nid: \"004\"\r\ntitle: Windows\r\n---\r\n\r\nBody\r\n",
			wantFM:   "id: \"004\"\ntitle: Windows\n",
			wantBody: "\nBody\n",
			wantErr:  false,
		},
		{
			name: "frontmatter with leading whitespace on delimiter",
			content: `  ---
id: "005"
title: Whitespace
---

Body
`,
			wantFM:   "id: \"005\"\ntitle: Whitespace\n",
			wantBody: "\nBody\n",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fm, body, err := parseFrontmatter([]byte(tt.content))

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error but got none")
				}
				if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("error = %q, want to contain %q", err.Error(), tt.errContains)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if string(fm) != tt.wantFM {
				t.Errorf("frontmatter =\n%q\nwant\n%q", string(fm), tt.wantFM)
			}

			if string(body) != tt.wantBody {
				t.Errorf("body =\n%q\nwant\n%q", string(body), tt.wantBody)
			}
		})
	}
}

func TestParseFrontmatterErrors(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		errContains string
	}{
		{
			name:        "empty file",
			content:     "",
			errContains: "empty file",
		},
		{
			name:        "no frontmatter delimiter",
			content:     "Just plain text\nNo frontmatter here",
			errContains: "does not start with frontmatter delimiter",
		},
		{
			name: "unclosed frontmatter",
			content: `---
id: "001"
title: Test
No closing delimiter`,
			errContains: "frontmatter not closed",
		},
		{
			name:        "only opening delimiter",
			content:     "---\n",
			errContains: "frontmatter not closed",
		},
		{
			name:        "whitespace only",
			content:     "   \n   \n",
			errContains: "does not start with frontmatter delimiter",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := parseFrontmatter([]byte(tt.content))

			if err == nil {
				t.Fatal("expected error but got none")
			}

			if !contains(err.Error(), tt.errContains) {
				t.Errorf("error = %q, want to contain %q", err.Error(), tt.errContains)
			}
		})
	}
}

func TestParseBody(t *testing.T) {
	tests := []struct {
		name            string
		body            string
		wantDesc        string
		wantCommentLen  int
		wantFirstAuthor string
	}{
		{
			name:           "description only",
			body:           "\n## Description\n\nThis is a description.\n",
			wantDesc:       "This is a description.",
			wantCommentLen: 0,
		},
		{
			name:           "description without header",
			body:           "\nJust content without header.\n",
			wantDesc:       "Just content without header.",
			wantCommentLen: 0,
		},
		{
			name:            "description with comments",
			body:            "\n## Description\n\nTask description.\n\n## Comments\n\n### 2025-01-16 @alex\n\nFirst comment body.\n",
			wantDesc:        "Task description.",
			wantCommentLen:  1,
			wantFirstAuthor: "alex",
		},
		{
			name:            "multiple comments",
			body:            "\n## Description\n\nDesc.\n\n## Comments\n\n### 2025-01-16 @alex\n\nComment 1.\n\n### 2025-01-17 @bob\n\nComment 2.\n",
			wantDesc:        "Desc.",
			wantCommentLen:  2,
			wantFirstAuthor: "alex",
		},
		{
			name:           "empty body",
			body:           "",
			wantDesc:       "",
			wantCommentLen: 0,
		},
		{
			name:           "comments section only no description",
			body:           "\n## Comments\n\n### 2025-01-16 @user\n\nComment only.\n",
			wantDesc:       "",
			wantCommentLen: 1,
		},
		{
			name:           "multiline description",
			body:           "\n## Description\n\nLine 1.\n\nLine 2.\n\nLine 3.\n",
			wantDesc:       "Line 1.\n\nLine 2.\n\nLine 3.",
			wantCommentLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			desc, comments := parseBody([]byte(tt.body))

			if desc != tt.wantDesc {
				t.Errorf("description =\n%q\nwant\n%q", desc, tt.wantDesc)
			}

			if len(comments) != tt.wantCommentLen {
				t.Errorf("len(comments) = %d, want %d", len(comments), tt.wantCommentLen)
			}

			if tt.wantFirstAuthor != "" && len(comments) > 0 {
				if comments[0].Author != tt.wantFirstAuthor {
					t.Errorf("first comment author = %q, want %q", comments[0].Author, tt.wantFirstAuthor)
				}
			}
		})
	}
}

func TestExtractDescription(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "with header",
			content: "## Description\n\nThe actual description.",
			want:    "The actual description.",
		},
		{
			name:    "without header",
			content: "Content without description header.",
			want:    "Content without description header.",
		},
		{
			name:    "with Windows line ending header",
			content: "## Description\r\n\r\nWindows description.",
			want:    "Windows description.",
		},
		{
			name:    "empty content",
			content: "",
			want:    "",
		},
		{
			name:    "only whitespace",
			content: "   \n\n   ",
			want:    "",
		},
		{
			name:    "header with no content after",
			content: "## Description\n",
			want:    "## Description",
		},
		{
			name:    "multiline content",
			content: "## Description\n\nFirst paragraph.\n\nSecond paragraph.\n\n- List item 1\n- List item 2",
			want:    "First paragraph.\n\nSecond paragraph.\n\n- List item 1\n- List item 2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractDescription(tt.content)
			if got != tt.want {
				t.Errorf("extractDescription() =\n%q\nwant\n%q", got, tt.want)
			}
		})
	}
}

func TestParseComments(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		wantLen  int
		validate func(t *testing.T, comments []backend.Comment)
	}{
		{
			name:    "empty content",
			content: "",
			wantLen: 0,
		},
		{
			name:    "no comments",
			content: "Just some text without comment headers.",
			wantLen: 0,
		},
		{
			name:    "single comment",
			content: "\n### 2025-01-16 @alex\n\nThis is a comment.\n",
			wantLen: 1,
			validate: func(t *testing.T, comments []backend.Comment) {
				c := comments[0]
				if c.ID != "c1" {
					t.Errorf("ID = %q, want %q", c.ID, "c1")
				}
				if c.Author != "alex" {
					t.Errorf("Author = %q, want %q", c.Author, "alex")
				}
				if c.Body != "This is a comment." {
					t.Errorf("Body = %q, want %q", c.Body, "This is a comment.")
				}
				expectedDate := time.Date(2025, 1, 16, 0, 0, 0, 0, time.UTC)
				if !c.Created.Equal(expectedDate) {
					t.Errorf("Created = %v, want %v", c.Created, expectedDate)
				}
			},
		},
		{
			name: "multiple comments",
			content: `### 2025-01-16 @alex

First comment.

### 2025-01-17 @bob

Second comment.

### 2025-01-18 @charlie

Third comment.
`,
			wantLen: 3,
			validate: func(t *testing.T, comments []backend.Comment) {
				authors := []string{"alex", "bob", "charlie"}
				for i, c := range comments {
					if c.Author != authors[i] {
						t.Errorf("comment[%d].Author = %q, want %q", i, c.Author, authors[i])
					}
					expectedID := "c" + string(rune('1'+i))
					if c.ID != expectedID {
						t.Errorf("comment[%d].ID = %q, want %q", i, c.ID, expectedID)
					}
				}
			},
		},
		{
			name:    "comment with multiline body",
			content: "### 2025-01-16 @user\n\nLine 1.\n\nLine 2.\n\nLine 3.",
			wantLen: 1,
			validate: func(t *testing.T, comments []backend.Comment) {
				expected := "Line 1.\n\nLine 2.\n\nLine 3."
				if comments[0].Body != expected {
					t.Errorf("Body =\n%q\nwant\n%q", comments[0].Body, expected)
				}
			},
		},
		{
			name:    "comment with special characters in author",
			content: "### 2025-01-16 @user-123\n\nComment body.",
			wantLen: 1,
			validate: func(t *testing.T, comments []backend.Comment) {
				if comments[0].Author != "user-123" {
					t.Errorf("Author = %q, want %q", comments[0].Author, "user-123")
				}
			},
		},
		{
			name:    "comment with varied spacing",
			content: "###  2025-01-16   @alex\n\nComment.",
			wantLen: 1,
			validate: func(t *testing.T, comments []backend.Comment) {
				if comments[0].Author != "alex" {
					t.Errorf("Author = %q, want %q", comments[0].Author, "alex")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			comments := parseComments(tt.content)

			if len(comments) != tt.wantLen {
				t.Errorf("len(comments) = %d, want %d", len(comments), tt.wantLen)
			}

			if tt.validate != nil && len(comments) > 0 {
				tt.validate(t, comments)
			}
		})
	}
}

func TestGenerateFilename(t *testing.T) {
	tests := []struct {
		id    string
		title string
		want  string
	}{
		{
			id:    "001",
			title: "Simple Task",
			want:  "001-simple-task.md",
		},
		{
			id:    "002",
			title: "",
			want:  "002.md",
		},
		{
			id:    "003",
			title: "Task with Special!@# Characters",
			want:  "003-task-with-special-characters.md",
		},
		{
			id:    "004",
			title: "Task_With_Underscores",
			want:  "004-task-with-underscores.md",
		},
		{
			id:    "005",
			title: "Multiple   Spaces",
			want:  "005-multiple-spaces.md",
		},
		{
			id:    "006",
			title: "This is a very long task title that should be truncated to fifty characters maximum",
			want:  "006-this-is-a-very-long-task-title-that-should-be-trun.md",
		},
		{
			id:    "007",
			title: "   Trimmed Title   ",
			want:  "007-trimmed-title.md",
		},
		{
			id:    "008",
			title: "123 Numbers First",
			want:  "008-123-numbers-first.md",
		},
		{
			id:    "009",
			title: "UPPERCASE TITLE",
			want:  "009-uppercase-title.md",
		},
		{
			id:    "010",
			title: "---leading-trailing---",
			want:  "010-leading-trailing.md",
		},
	}

	for _, tt := range tests {
		t.Run(tt.title, func(t *testing.T) {
			got := generateFilename(tt.id, tt.title)
			if got != tt.want {
				t.Errorf("generateFilename(%q, %q) = %q, want %q", tt.id, tt.title, got, tt.want)
			}
		})
	}
}

func TestReadTaskFileComplete(t *testing.T) {
	tmpDir := t.TempDir()
	backlogDir := filepath.Join(tmpDir, ".backlog")

	// Create status directories
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

	tests := []struct {
		name     string
		content  string
		status   backend.Status
		validate func(t *testing.T, task *backend.Task)
	}{
		{
			name: "full task with all fields",
			content: `---
id: "001"
title: Full Featured Task
priority: high
assignee: user1
labels:
  - bug
  - critical
created: 2025-01-15T09:00:00Z
updated: 2025-01-18T14:30:00Z
---

## Description

This task has all fields populated.

Multiple paragraphs are supported.

## Comments

### 2025-01-16 @alex

Working on this now.

### 2025-01-17 @bob

Review complete.
`,
			status: backend.StatusInProgress,
			validate: func(t *testing.T, task *backend.Task) {
				if task.ID != "001" {
					t.Errorf("ID = %q, want %q", task.ID, "001")
				}
				if task.Title != "Full Featured Task" {
					t.Errorf("Title = %q, want %q", task.Title, "Full Featured Task")
				}
				if task.Priority != backend.PriorityHigh {
					t.Errorf("Priority = %q, want %q", task.Priority, backend.PriorityHigh)
				}
				if task.Assignee != "user1" {
					t.Errorf("Assignee = %q, want %q", task.Assignee, "user1")
				}
				if task.Status != backend.StatusInProgress {
					t.Errorf("Status = %q, want %q", task.Status, backend.StatusInProgress)
				}
				if len(task.Labels) != 2 {
					t.Errorf("len(Labels) = %d, want 2", len(task.Labels))
				}
				if task.Meta == nil {
					t.Fatal("Meta is nil, expected comments")
				}
				comments, ok := task.Meta["comments"].([]backend.Comment)
				if !ok {
					t.Fatal("comments not found in Meta")
				}
				if len(comments) != 2 {
					t.Errorf("len(comments) = %d, want 2", len(comments))
				}
			},
		},
		{
			name: "minimal task",
			content: `---
id: "002"
title: Minimal Task
created: 2025-01-15T09:00:00Z
updated: 2025-01-15T09:00:00Z
---
`,
			status: backend.StatusBacklog,
			validate: func(t *testing.T, task *backend.Task) {
				if task.ID != "002" {
					t.Errorf("ID = %q, want %q", task.ID, "002")
				}
				if task.Title != "Minimal Task" {
					t.Errorf("Title = %q, want %q", task.Title, "Minimal Task")
				}
				// Empty priority should default to none
				if task.Priority != backend.PriorityNone {
					t.Errorf("Priority = %q, want %q (default)", task.Priority, backend.PriorityNone)
				}
				if task.Assignee != "" {
					t.Errorf("Assignee = %q, want empty", task.Assignee)
				}
				if task.Description != "" {
					t.Errorf("Description = %q, want empty", task.Description)
				}
			},
		},
		{
			name: "task with empty labels array",
			content: `---
id: "003"
title: No Labels
labels: []
created: 2025-01-15T09:00:00Z
updated: 2025-01-15T09:00:00Z
---

## Description

Task with explicitly empty labels.
`,
			status: backend.StatusTodo,
			validate: func(t *testing.T, task *backend.Task) {
				if len(task.Labels) != 0 {
					t.Errorf("len(Labels) = %d, want 0", len(task.Labels))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Write the test file
			filename := "test-task.md"
			filePath := filepath.Join(backlogDir, string(tt.status), filename)
			if err := os.WriteFile(filePath, []byte(tt.content), 0644); err != nil {
				t.Fatalf("failed to write test file: %v", err)
			}
			defer os.Remove(filePath)

			// Read and validate
			task, err := l.readTaskFile(filePath, tt.status)
			if err != nil {
				t.Fatalf("readTaskFile() error = %v", err)
			}

			tt.validate(t, task)
		})
	}
}

func TestReadTaskFileErrors(t *testing.T) {
	tmpDir := t.TempDir()
	backlogDir := filepath.Join(tmpDir, ".backlog")

	for _, status := range []string{"backlog"} {
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

	tests := []struct {
		name        string
		content     string
		errContains string
	}{
		{
			name:        "invalid frontmatter YAML",
			content:     "---\nid: [invalid yaml\n---\n",
			errContains: "unmarshal",
		},
		{
			name:        "no frontmatter",
			content:     "No frontmatter here",
			errContains: "does not start with frontmatter delimiter",
		},
		{
			name:        "unclosed frontmatter",
			content:     "---\nid: 001\ntitle: test\n",
			errContains: "not closed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filename := "test-error.md"
			filePath := filepath.Join(backlogDir, "backlog", filename)
			if err := os.WriteFile(filePath, []byte(tt.content), 0644); err != nil {
				t.Fatalf("failed to write test file: %v", err)
			}
			defer os.Remove(filePath)

			_, err := l.readTaskFile(filePath, backend.StatusBacklog)
			if err == nil {
				t.Fatal("expected error but got none")
			}

			if !contains(err.Error(), tt.errContains) {
				t.Errorf("error = %q, want to contain %q", err.Error(), tt.errContains)
			}
		})
	}
}

func TestReadTaskFileNonExistent(t *testing.T) {
	l := New()
	// Don't connect, just test file read

	_, err := l.readTaskFile("/nonexistent/path/task.md", backend.StatusBacklog)
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestWriteTaskAndReadBack(t *testing.T) {
	tmpDir := t.TempDir()
	backlogDir := filepath.Join(tmpDir, ".backlog")

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

	originalTask := &backend.Task{
		ID:          "999",
		Title:       "Roundtrip Test",
		Description: "Testing write and read.\n\nMultiple paragraphs.",
		Status:      backend.StatusReview,
		Priority:    backend.PriorityUrgent,
		Assignee:    "tester",
		Labels:      []string{"test", "roundtrip"},
		Created:     time.Date(2025, 1, 15, 9, 0, 0, 0, time.UTC),
		Updated:     time.Date(2025, 1, 18, 14, 30, 0, 0, time.UTC),
		Meta: map[string]any{
			"comments": []backend.Comment{
				{
					ID:      "c1",
					Author:  "alex",
					Body:    "First comment.",
					Created: time.Date(2025, 1, 16, 0, 0, 0, 0, time.UTC),
				},
				{
					ID:      "c2",
					Author:  "bob",
					Body:    "Second comment.",
					Created: time.Date(2025, 1, 17, 0, 0, 0, 0, time.UTC),
				},
			},
		},
	}

	// Write the task
	if err := l.writeTask(originalTask); err != nil {
		t.Fatalf("writeTask() error = %v", err)
	}

	// Read it back via Get (which uses readTaskFile internally)
	readTask, err := l.Get(originalTask.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	// Verify all fields
	if readTask.ID != originalTask.ID {
		t.Errorf("ID = %q, want %q", readTask.ID, originalTask.ID)
	}
	if readTask.Title != originalTask.Title {
		t.Errorf("Title = %q, want %q", readTask.Title, originalTask.Title)
	}
	if readTask.Description != originalTask.Description {
		t.Errorf("Description = %q, want %q", readTask.Description, originalTask.Description)
	}
	if readTask.Status != originalTask.Status {
		t.Errorf("Status = %q, want %q", readTask.Status, originalTask.Status)
	}
	if readTask.Priority != originalTask.Priority {
		t.Errorf("Priority = %q, want %q", readTask.Priority, originalTask.Priority)
	}
	if readTask.Assignee != originalTask.Assignee {
		t.Errorf("Assignee = %q, want %q", readTask.Assignee, originalTask.Assignee)
	}
	if len(readTask.Labels) != len(originalTask.Labels) {
		t.Errorf("len(Labels) = %d, want %d", len(readTask.Labels), len(originalTask.Labels))
	}

	// Verify comments
	if readTask.Meta == nil {
		t.Fatal("Meta is nil")
	}
	comments, ok := readTask.Meta["comments"].([]backend.Comment)
	if !ok {
		t.Fatal("comments not found in Meta")
	}
	if len(comments) != 2 {
		t.Errorf("len(comments) = %d, want 2", len(comments))
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestTaskFile_BlocksBlockedBy(t *testing.T) {
	l, backlogDir := setupBacklog(t)

	// Create a task with blocks/blocked_by in meta
	task := &backend.Task{
		ID:       "001",
		Title:    "Task with deps",
		Status:   backend.StatusTodo,
		Priority: backend.PriorityHigh,
		Created:  time.Now().UTC(),
		Updated:  time.Now().UTC(),
		Meta: map[string]any{
			"blocks":     []string{"002", "003"},
			"blocked_by": []string{"004"},
		},
	}

	// Write the task
	if err := l.writeTask(task); err != nil {
		t.Fatalf("writeTask() error = %v", err)
	}

	// Read it back
	filePath := filepath.Join(backlogDir, "todo", "001-task-with-deps.md")
	readTask, err := l.readTaskFile(filePath, backend.StatusTodo)
	if err != nil {
		t.Fatalf("readTaskFile() error = %v", err)
	}

	// Verify blocks
	blocks, ok := readTask.Meta["blocks"].([]string)
	if !ok {
		// YAML unmarshals as []any, check for that
		if blocksAny, ok := readTask.Meta["blocks"].([]any); ok {
			blocks = make([]string, len(blocksAny))
			for i, v := range blocksAny {
				blocks[i] = v.(string)
			}
		}
	}
	if len(blocks) != 2 || blocks[0] != "002" || blocks[1] != "003" {
		t.Errorf("blocks = %v, want [002 003]", blocks)
	}

	// Verify blocked_by
	blockedBy, ok := readTask.Meta["blocked_by"].([]string)
	if !ok {
		if blockedByAny, ok := readTask.Meta["blocked_by"].([]any); ok {
			blockedBy = make([]string, len(blockedByAny))
			for i, v := range blockedByAny {
				blockedBy[i] = v.(string)
			}
		}
	}
	if len(blockedBy) != 1 || blockedBy[0] != "004" {
		t.Errorf("blocked_by = %v, want [004]", blockedBy)
	}

	// Verify frontmatter contains the fields
	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	contentStr := string(content)
	if !contains(contentStr, "blocks:") {
		t.Error("frontmatter should contain 'blocks:'")
	}
	if !contains(contentStr, "blocked_by:") {
		t.Error("frontmatter should contain 'blocked_by:'")
	}
}
