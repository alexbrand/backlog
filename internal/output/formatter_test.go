package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/alexbrand/backlog/internal/backend"
)

func testTask() *backend.Task {
	return &backend.Task{
		ID:          "GH-123",
		Title:       "Implement auth flow",
		Description: "OAuth2 implementation details...",
		Status:      backend.StatusInProgress,
		Priority:    backend.PriorityHigh,
		Assignee:    "alex",
		Labels:      []string{"feature", "auth"},
		Created:     time.Date(2025, 1, 15, 9, 0, 0, 0, time.UTC),
		Updated:     time.Date(2025, 1, 18, 14, 30, 0, 0, time.UTC),
		URL:         "https://github.com/alexbrand/myproject/issues/123",
	}
}

func testTaskList() *backend.TaskList {
	return &backend.TaskList{
		Tasks: []backend.Task{
			{
				ID:       "GH-123",
				Title:    "Implement auth flow",
				Status:   backend.StatusInProgress,
				Priority: backend.PriorityHigh,
				Assignee: "alex",
			},
			{
				ID:       "GH-124",
				Title:    "Add rate limiting",
				Status:   backend.StatusTodo,
				Priority: backend.PriorityMedium,
			},
		},
		Count:   2,
		HasMore: false,
	}
}

func TestFormatIsValid(t *testing.T) {
	tests := []struct {
		format Format
		valid  bool
	}{
		{FormatTable, true},
		{FormatJSON, true},
		{FormatPlain, true},
		{FormatIDOnly, true},
		{Format("invalid"), false},
		{Format(""), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.format), func(t *testing.T) {
			if got := tt.format.IsValid(); got != tt.valid {
				t.Errorf("Format(%q).IsValid() = %v, want %v", tt.format, got, tt.valid)
			}
		})
	}
}

func TestNew(t *testing.T) {
	tests := []struct {
		format   Format
		expected string
	}{
		{FormatTable, "*output.TableFormatter"},
		{FormatJSON, "*output.JSONFormatter"},
		{FormatPlain, "*output.PlainFormatter"},
		{FormatIDOnly, "*output.IDOnlyFormatter"},
		{Format("unknown"), "*output.TableFormatter"}, // defaults to table
	}

	for _, tt := range tests {
		t.Run(string(tt.format), func(t *testing.T) {
			f := New(tt.format)
			typeName := strings.Replace(strings.Replace(
				strings.Replace(string(rune(0)), "", "", -1)+
					func() string { return "" }()+
					func() string {
						return strings.TrimPrefix(
							strings.Replace(
								func() string {
									var buf bytes.Buffer
									buf.WriteString("*output.")
									switch f.(type) {
									case *TableFormatter:
										buf.WriteString("TableFormatter")
									case *JSONFormatter:
										buf.WriteString("JSONFormatter")
									case *PlainFormatter:
										buf.WriteString("PlainFormatter")
									case *IDOnlyFormatter:
										buf.WriteString("IDOnlyFormatter")
									}
									return buf.String()
								}(),
								"", "", 0),
							"")
					}(),
				"", "", 0), "", "", 0)

			// Simpler type check
			switch f.(type) {
			case *TableFormatter:
				typeName = "*output.TableFormatter"
			case *JSONFormatter:
				typeName = "*output.JSONFormatter"
			case *PlainFormatter:
				typeName = "*output.PlainFormatter"
			case *IDOnlyFormatter:
				typeName = "*output.IDOnlyFormatter"
			}

			if typeName != tt.expected {
				t.Errorf("New(%q) returned %s, want %s", tt.format, typeName, tt.expected)
			}
		})
	}
}

func TestTableFormatterFormatTask(t *testing.T) {
	f := &TableFormatter{}
	var buf bytes.Buffer
	task := testTask()

	err := f.FormatTask(&buf, task)
	if err != nil {
		t.Fatalf("FormatTask() error = %v", err)
	}

	output := buf.String()

	// Check key fields are present
	if !strings.Contains(output, "GH-123") {
		t.Error("Output should contain task ID")
	}
	if !strings.Contains(output, "Implement auth flow") {
		t.Error("Output should contain task title")
	}
	if !strings.Contains(output, "in-progress") {
		t.Error("Output should contain status")
	}
	if !strings.Contains(output, "high") {
		t.Error("Output should contain priority")
	}
	if !strings.Contains(output, "@alex") {
		t.Error("Output should contain assignee")
	}
}

func TestTableFormatterFormatTaskList(t *testing.T) {
	f := &TableFormatter{}
	var buf bytes.Buffer
	list := testTaskList()

	err := f.FormatTaskList(&buf, list)
	if err != nil {
		t.Fatalf("FormatTaskList() error = %v", err)
	}

	output := buf.String()

	// Check header
	if !strings.Contains(output, "ID") {
		t.Error("Output should contain ID header")
	}
	if !strings.Contains(output, "STATUS") {
		t.Error("Output should contain STATUS header")
	}

	// Check tasks
	if !strings.Contains(output, "GH-123") {
		t.Error("Output should contain first task ID")
	}
	if !strings.Contains(output, "GH-124") {
		t.Error("Output should contain second task ID")
	}
}

func TestTableFormatterEmptyList(t *testing.T) {
	f := &TableFormatter{}
	var buf bytes.Buffer
	list := &backend.TaskList{Tasks: []backend.Task{}}

	err := f.FormatTaskList(&buf, list)
	if err != nil {
		t.Fatalf("FormatTaskList() error = %v", err)
	}

	if !strings.Contains(buf.String(), "No tasks found") {
		t.Error("Empty list should show 'No tasks found'")
	}
}

func TestJSONFormatterFormatTask(t *testing.T) {
	f := &JSONFormatter{}
	var buf bytes.Buffer
	task := testTask()

	err := f.FormatTask(&buf, task)
	if err != nil {
		t.Fatalf("FormatTask() error = %v", err)
	}

	// Parse the JSON to verify it's valid
	var result map[string]any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("Output is not valid JSON: %v", err)
	}

	if result["id"] != "GH-123" {
		t.Errorf("id = %v, want GH-123", result["id"])
	}
	if result["title"] != "Implement auth flow" {
		t.Errorf("title = %v, want Implement auth flow", result["title"])
	}
	if result["status"] != "in-progress" {
		t.Errorf("status = %v, want in-progress", result["status"])
	}
}

func TestJSONFormatterFormatTaskList(t *testing.T) {
	f := &JSONFormatter{}
	var buf bytes.Buffer
	list := testTaskList()

	err := f.FormatTaskList(&buf, list)
	if err != nil {
		t.Fatalf("FormatTaskList() error = %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("Output is not valid JSON: %v", err)
	}

	tasks, ok := result["tasks"].([]any)
	if !ok {
		t.Fatal("Result should have tasks array")
	}
	if len(tasks) != 2 {
		t.Errorf("len(tasks) = %d, want 2", len(tasks))
	}

	if result["count"].(float64) != 2 {
		t.Errorf("count = %v, want 2", result["count"])
	}
}

func TestJSONFormatterFormatError(t *testing.T) {
	f := &JSONFormatter{}
	var buf bytes.Buffer

	err := f.FormatError(&buf, "NOT_FOUND", "Task GH-999 not found", nil)
	if err != nil {
		t.Fatalf("FormatError() error = %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("Output is not valid JSON: %v", err)
	}

	errObj, ok := result["error"].(map[string]any)
	if !ok {
		t.Fatal("Result should have error object")
	}

	if errObj["code"] != "NOT_FOUND" {
		t.Errorf("code = %v, want NOT_FOUND", errObj["code"])
	}
	if errObj["message"] != "Task GH-999 not found" {
		t.Errorf("message = %v, want Task GH-999 not found", errObj["message"])
	}
}

func TestPlainFormatterFormatTask(t *testing.T) {
	f := &PlainFormatter{}
	var buf bytes.Buffer
	task := testTask()

	err := f.FormatTask(&buf, task)
	if err != nil {
		t.Fatalf("FormatTask() error = %v", err)
	}

	output := buf.String()
	// Plain format is tab-separated
	if !strings.Contains(output, "GH-123\t") {
		t.Error("Output should contain tab-separated ID")
	}
}

func TestPlainFormatterFormatTaskList(t *testing.T) {
	f := &PlainFormatter{}
	var buf bytes.Buffer
	list := testTaskList()

	err := f.FormatTaskList(&buf, list)
	if err != nil {
		t.Fatalf("FormatTaskList() error = %v", err)
	}

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 2 {
		t.Errorf("Expected 2 lines, got %d", len(lines))
	}
}

func TestIDOnlyFormatterFormatTask(t *testing.T) {
	f := &IDOnlyFormatter{}
	var buf bytes.Buffer
	task := testTask()

	err := f.FormatTask(&buf, task)
	if err != nil {
		t.Fatalf("FormatTask() error = %v", err)
	}

	output := strings.TrimSpace(buf.String())
	if output != "GH-123" {
		t.Errorf("Output = %q, want GH-123", output)
	}
}

func TestIDOnlyFormatterFormatTaskList(t *testing.T) {
	f := &IDOnlyFormatter{}
	var buf bytes.Buffer
	list := testTaskList()

	err := f.FormatTaskList(&buf, list)
	if err != nil {
		t.Fatalf("FormatTaskList() error = %v", err)
	}

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 2 {
		t.Errorf("Expected 2 lines, got %d", len(lines))
	}
	if lines[0] != "GH-123" {
		t.Errorf("First line = %q, want GH-123", lines[0])
	}
	if lines[1] != "GH-124" {
		t.Errorf("Second line = %q, want GH-124", lines[1])
	}
}

func TestTableFormatterFormatCreated(t *testing.T) {
	f := &TableFormatter{}
	var buf bytes.Buffer
	task := testTask()

	err := f.FormatCreated(&buf, task)
	if err != nil {
		t.Fatalf("FormatCreated() error = %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Created GH-123") {
		t.Error("Output should contain 'Created' with task ID")
	}
	if !strings.Contains(output, "Implement auth flow") {
		t.Error("Output should contain task title")
	}
}

func TestJSONFormatterFormatCreated(t *testing.T) {
	f := &JSONFormatter{}
	var buf bytes.Buffer
	task := testTask()

	err := f.FormatCreated(&buf, task)
	if err != nil {
		t.Fatalf("FormatCreated() error = %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("Output is not valid JSON: %v", err)
	}

	if result["id"] != "GH-123" {
		t.Errorf("id = %v, want GH-123", result["id"])
	}
	if result["title"] != "Implement auth flow" {
		t.Errorf("title = %v, want Implement auth flow", result["title"])
	}
}
