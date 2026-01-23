package support

import (
	"testing"
)

func TestParseJSON_ValidJSON(t *testing.T) {
	jsonStr := `{"name": "test", "count": 42}`
	result := ParseJSON(jsonStr)

	if !result.Valid() {
		t.Errorf("Expected valid JSON, got error: %s", result.Error())
	}
}

func TestParseJSON_InvalidJSON(t *testing.T) {
	jsonStr := `{not valid json}`
	result := ParseJSON(jsonStr)

	if result.Valid() {
		t.Error("Expected invalid JSON, but Valid() returned true")
	}
	if result.Error() == "" {
		t.Error("Expected error message, got empty string")
	}
}

func TestJSONResult_GetString(t *testing.T) {
	jsonStr := `{"id": "GH-123", "title": "Test task"}`
	result := ParseJSON(jsonStr)

	if got := result.GetString("id"); got != "GH-123" {
		t.Errorf("GetString('id') = %q, want %q", got, "GH-123")
	}

	if got := result.GetString("title"); got != "Test task" {
		t.Errorf("GetString('title') = %q, want %q", got, "Test task")
	}

	if got := result.GetString("nonexistent"); got != "" {
		t.Errorf("GetString('nonexistent') = %q, want empty string", got)
	}
}

func TestJSONResult_GetInt(t *testing.T) {
	jsonStr := `{"count": 42, "zero": 0}`
	result := ParseJSON(jsonStr)

	if got := result.GetInt("count"); got != 42 {
		t.Errorf("GetInt('count') = %d, want %d", got, 42)
	}

	if got := result.GetInt("zero"); got != 0 {
		t.Errorf("GetInt('zero') = %d, want %d", got, 0)
	}

	if got := result.GetInt("nonexistent"); got != 0 {
		t.Errorf("GetInt('nonexistent') = %d, want %d", got, 0)
	}
}

func TestJSONResult_GetBool(t *testing.T) {
	jsonStr := `{"hasMore": true, "isEmpty": false}`
	result := ParseJSON(jsonStr)

	if got := result.GetBool("hasMore"); got != true {
		t.Errorf("GetBool('hasMore') = %v, want %v", got, true)
	}

	if got := result.GetBool("isEmpty"); got != false {
		t.Errorf("GetBool('isEmpty') = %v, want %v", got, false)
	}

	if got := result.GetBool("nonexistent"); got != false {
		t.Errorf("GetBool('nonexistent') = %v, want %v", got, false)
	}
}

func TestJSONResult_GetArray(t *testing.T) {
	jsonStr := `{"labels": ["feature", "auth", "high-priority"]}`
	result := ParseJSON(jsonStr)

	arr := result.GetArray("labels")
	if arr == nil {
		t.Fatal("GetArray('labels') returned nil")
	}

	if len(arr) != 3 {
		t.Errorf("len(labels) = %d, want %d", len(arr), 3)
	}

	if arr[0] != "feature" {
		t.Errorf("labels[0] = %v, want %q", arr[0], "feature")
	}
}

func TestJSONResult_GetNestedPath(t *testing.T) {
	jsonStr := `{
		"task": {
			"id": "GH-123",
			"metadata": {
				"created_by": "alex"
			}
		}
	}`
	result := ParseJSON(jsonStr)

	if got := result.GetString("task.id"); got != "GH-123" {
		t.Errorf("GetString('task.id') = %q, want %q", got, "GH-123")
	}

	if got := result.GetString("task.metadata.created_by"); got != "alex" {
		t.Errorf("GetString('task.metadata.created_by') = %q, want %q", got, "alex")
	}
}

func TestJSONResult_GetArrayIndex(t *testing.T) {
	jsonStr := `{
		"tasks": [
			{"id": "GH-001", "title": "First"},
			{"id": "GH-002", "title": "Second"},
			{"id": "GH-003", "title": "Third"}
		]
	}`
	result := ParseJSON(jsonStr)

	if got := result.GetString("tasks[0].id"); got != "GH-001" {
		t.Errorf("GetString('tasks[0].id') = %q, want %q", got, "GH-001")
	}

	if got := result.GetString("tasks[1].title"); got != "Second" {
		t.Errorf("GetString('tasks[1].title') = %q, want %q", got, "Second")
	}

	if got := result.GetString("tasks[2].id"); got != "GH-003" {
		t.Errorf("GetString('tasks[2].id') = %q, want %q", got, "GH-003")
	}

	// Out of bounds
	if got := result.GetString("tasks[10].id"); got != "" {
		t.Errorf("GetString('tasks[10].id') = %q, want empty string", got)
	}
}

func TestJSONResult_ArrayLen(t *testing.T) {
	jsonStr := `{"tasks": [1, 2, 3], "empty": []}`
	result := ParseJSON(jsonStr)

	if got := result.ArrayLen("tasks"); got != 3 {
		t.Errorf("ArrayLen('tasks') = %d, want %d", got, 3)
	}

	if got := result.ArrayLen("empty"); got != 0 {
		t.Errorf("ArrayLen('empty') = %d, want %d", got, 0)
	}

	if got := result.ArrayLen("nonexistent"); got != -1 {
		t.Errorf("ArrayLen('nonexistent') = %d, want %d", got, -1)
	}
}

func TestJSONResult_Has(t *testing.T) {
	jsonStr := `{"id": "GH-123", "assignee": null, "nested": {"key": "value"}}`
	result := ParseJSON(jsonStr)

	if !result.Has("id") {
		t.Error("Has('id') = false, want true")
	}

	if !result.Has("assignee") {
		t.Error("Has('assignee') = false, want true (even for null)")
	}

	if !result.Has("nested.key") {
		t.Error("Has('nested.key') = false, want true")
	}

	if result.Has("nonexistent") {
		t.Error("Has('nonexistent') = true, want false")
	}
}

func TestJSONResult_Equals(t *testing.T) {
	jsonStr := `{"id": "GH-123", "count": 42, "active": true}`
	result := ParseJSON(jsonStr)

	if !result.Equals("id", "GH-123") {
		t.Error("Equals('id', 'GH-123') = false, want true")
	}

	if !result.Equals("count", 42) {
		t.Error("Equals('count', 42) = false, want true")
	}

	if result.Equals("id", "wrong") {
		t.Error("Equals('id', 'wrong') = true, want false")
	}
}

func TestJSONResult_StringEquals(t *testing.T) {
	jsonStr := `{"status": "in-progress"}`
	result := ParseJSON(jsonStr)

	if !result.StringEquals("status", "in-progress") {
		t.Error("StringEquals('status', 'in-progress') = false, want true")
	}

	if result.StringEquals("status", "done") {
		t.Error("StringEquals('status', 'done') = true, want false")
	}
}

func TestJSONResult_IntEquals(t *testing.T) {
	jsonStr := `{"count": 5}`
	result := ParseJSON(jsonStr)

	if !result.IntEquals("count", 5) {
		t.Error("IntEquals('count', 5) = false, want true")
	}

	if result.IntEquals("count", 10) {
		t.Error("IntEquals('count', 10) = true, want false")
	}
}

func TestJSONResult_BoolEquals(t *testing.T) {
	jsonStr := `{"hasMore": true}`
	result := ParseJSON(jsonStr)

	if !result.BoolEquals("hasMore", true) {
		t.Error("BoolEquals('hasMore', true) = false, want true")
	}

	if result.BoolEquals("hasMore", false) {
		t.Error("BoolEquals('hasMore', false) = true, want false")
	}
}

func TestJSONResult_Contains(t *testing.T) {
	jsonStr := `{"labels": ["feature", "auth", "bug"]}`
	result := ParseJSON(jsonStr)

	if !result.Contains("labels", "feature") {
		t.Error("Contains('labels', 'feature') = false, want true")
	}

	if !result.ContainsString("labels", "auth") {
		t.Error("ContainsString('labels', 'auth') = false, want true")
	}

	if result.ContainsString("labels", "nonexistent") {
		t.Error("ContainsString('labels', 'nonexistent') = true, want false")
	}
}

func TestJSONResult_IsNull(t *testing.T) {
	jsonStr := `{"assignee": null, "title": "Test"}`
	result := ParseJSON(jsonStr)

	if !result.IsNull("assignee") {
		t.Error("IsNull('assignee') = false, want true")
	}

	if result.IsNull("title") {
		t.Error("IsNull('title') = true, want false")
	}

	if result.IsNull("nonexistent") {
		t.Error("IsNull('nonexistent') = true, want false (doesn't exist)")
	}
}

func TestJSONResult_IsArray(t *testing.T) {
	jsonStr := `{"tasks": [], "count": 0}`
	result := ParseJSON(jsonStr)

	if !result.IsArray("tasks") {
		t.Error("IsArray('tasks') = false, want true")
	}

	if result.IsArray("count") {
		t.Error("IsArray('count') = true, want false")
	}
}

func TestJSONResult_IsObject(t *testing.T) {
	jsonStr := `{"metadata": {"key": "value"}, "name": "test"}`
	result := ParseJSON(jsonStr)

	if !result.IsObject("metadata") {
		t.Error("IsObject('metadata') = false, want true")
	}

	if result.IsObject("name") {
		t.Error("IsObject('name') = true, want false")
	}
}

func TestJSONResult_GetObject(t *testing.T) {
	jsonStr := `{"error": {"code": "NOT_FOUND", "message": "Task not found"}}`
	result := ParseJSON(jsonStr)

	obj := result.GetObject("error")
	if obj == nil {
		t.Fatal("GetObject('error') returned nil")
	}

	if obj["code"] != "NOT_FOUND" {
		t.Errorf("error.code = %v, want %q", obj["code"], "NOT_FOUND")
	}
}

func TestJSONResult_ComplexPath(t *testing.T) {
	jsonStr := `{
		"data": {
			"items": [
				{"name": "first", "tags": ["a", "b"]},
				{"name": "second", "tags": ["c", "d"]}
			]
		}
	}`
	result := ParseJSON(jsonStr)

	if got := result.GetString("data.items[0].name"); got != "first" {
		t.Errorf("GetString('data.items[0].name') = %q, want %q", got, "first")
	}

	if got := result.GetString("data.items[1].tags[0]"); got != "c" {
		t.Errorf("GetString('data.items[1].tags[0]') = %q, want %q", got, "c")
	}

	if got := result.ArrayLen("data.items[0].tags"); got != 2 {
		t.Errorf("ArrayLen('data.items[0].tags') = %d, want %d", got, 2)
	}
}

func TestParsePath(t *testing.T) {
	tests := []struct {
		path     string
		expected []string
	}{
		{"tasks", []string{"tasks"}},
		{"tasks.id", []string{"tasks", "id"}},
		{"tasks[0]", []string{"tasks", "[0]"}},
		{"tasks[0].id", []string{"tasks", "[0]", "id"}},
		{"data.items[0].name", []string{"data", "items", "[0]", "name"}},
		{"[0]", []string{"[0]"}},
		{"a.b.c", []string{"a", "b", "c"}},
	}

	for _, tt := range tests {
		got := parsePath(tt.path)
		if len(got) != len(tt.expected) {
			t.Errorf("parsePath(%q) = %v, want %v", tt.path, got, tt.expected)
			continue
		}
		for i, part := range got {
			if part != tt.expected[i] {
				t.Errorf("parsePath(%q)[%d] = %q, want %q", tt.path, i, part, tt.expected[i])
			}
		}
	}
}

func TestParseArrayIndex(t *testing.T) {
	tests := []struct {
		part    string
		idx     int
		isIndex bool
	}{
		{"[0]", 0, true},
		{"[42]", 42, true},
		{"[100]", 100, true},
		{"tasks", 0, false},
		{"[]", 0, false},
		{"[abc]", 0, false},
	}

	for _, tt := range tests {
		idx, isIndex := parseArrayIndex(tt.part)
		if isIndex != tt.isIndex {
			t.Errorf("parseArrayIndex(%q) isIndex = %v, want %v", tt.part, isIndex, tt.isIndex)
		}
		if isIndex && idx != tt.idx {
			t.Errorf("parseArrayIndex(%q) idx = %d, want %d", tt.part, idx, tt.idx)
		}
	}
}

func TestParseJSONFromResult(t *testing.T) {
	cmdResult := &CommandResult{
		Stdout:   `{"id": "GH-123", "title": "Test task"}`,
		Stderr:   "",
		ExitCode: 0,
	}

	result := ParseJSONFromResult(cmdResult)
	if !result.Valid() {
		t.Errorf("Expected valid JSON, got error: %s", result.Error())
	}

	if got := result.GetString("id"); got != "GH-123" {
		t.Errorf("GetString('id') = %q, want %q", got, "GH-123")
	}
}
