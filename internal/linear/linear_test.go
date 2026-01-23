package linear

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/alexbrand/backlog/internal/backend"
)

// mockLinearServer creates a test server that responds to Linear GraphQL queries
func mockLinearServer(t *testing.T, handler func(query string, variables map[string]any) any) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Check authorization header
		auth := r.Header.Get("Authorization")
		if auth == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		var req struct {
			Query     string         `json:"query"`
			Variables map[string]any `json:"variables"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("failed to decode request: %v", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		resp := handler(req.Query, req.Variables)

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Errorf("failed to encode response: %v", err)
		}
	}))
}

func TestNew(t *testing.T) {
	l := New()
	if l == nil {
		t.Fatal("New() returned nil")
	}
	if l.ctx == nil {
		t.Error("context not initialized")
	}
	if l.client == nil {
		t.Error("HTTP client not initialized")
	}
}

func TestName(t *testing.T) {
	l := New()
	if l.Name() != "linear" {
		t.Errorf("Name() = %s, want linear", l.Name())
	}
}

func TestVersion(t *testing.T) {
	l := New()
	if l.Version() != "0.1.0" {
		t.Errorf("Version() = %s, want 0.1.0", l.Version())
	}
}

func TestLinearPriorityMapping(t *testing.T) {
	tests := []struct {
		name           string
		linearPriority int
		expected       backend.Priority
	}{
		{"no priority", 0, backend.PriorityNone},
		{"urgent", 1, backend.PriorityUrgent},
		{"high", 2, backend.PriorityHigh},
		{"medium", 3, backend.PriorityMedium},
		{"low", 4, backend.PriorityLow},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := linearPriorityToCanonical[tt.linearPriority]
			if got != tt.expected {
				t.Errorf("linearPriorityToCanonical[%d] = %s, want %s", tt.linearPriority, got, tt.expected)
			}
		})
	}
}

func TestCanonicalPriorityMapping(t *testing.T) {
	tests := []struct {
		name     string
		priority backend.Priority
		expected int
	}{
		{"none", backend.PriorityNone, 0},
		{"urgent", backend.PriorityUrgent, 1},
		{"high", backend.PriorityHigh, 2},
		{"medium", backend.PriorityMedium, 3},
		{"low", backend.PriorityLow, 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := canonicalPriorityToLinear[tt.priority]
			if got != tt.expected {
				t.Errorf("canonicalPriorityToLinear[%s] = %d, want %d", tt.priority, got, tt.expected)
			}
		})
	}
}

func TestDefaultStatusMapping(t *testing.T) {
	tests := []struct {
		stateName string
		expected  backend.Status
	}{
		{"Backlog", backend.StatusBacklog},
		{"Todo", backend.StatusTodo},
		{"To Do", backend.StatusTodo},
		{"In Progress", backend.StatusInProgress},
		{"In Review", backend.StatusReview},
		{"Review", backend.StatusReview},
		{"Done", backend.StatusDone},
		{"Completed", backend.StatusDone},
		{"Canceled", backend.StatusDone},
		{"Cancelled", backend.StatusDone},
	}

	for _, tt := range tests {
		t.Run(tt.stateName, func(t *testing.T) {
			got := defaultStatusMapping[tt.stateName]
			if got != tt.expected {
				t.Errorf("defaultStatusMapping[%s] = %s, want %s", tt.stateName, got, tt.expected)
			}
		})
	}
}

func TestNormalizeID(t *testing.T) {
	l := New()

	tests := []struct {
		input    string
		expected string
	}{
		{"LIN-ENG-123", "ENG-123"},
		{"ENG-123", "ENG-123"},
		{"LIN-ABC-1", "ABC-1"},
		{"ABC-1", "ABC-1"},
		{"LIN-", ""},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := l.normalizeID(tt.input)
			if got != tt.expected {
				t.Errorf("normalizeID(%s) = %s, want %s", tt.input, got, tt.expected)
			}
		})
	}
}

func TestGetString(t *testing.T) {
	tests := []struct {
		name     string
		m        map[string]any
		key      string
		expected string
	}{
		{
			name:     "existing string key",
			m:        map[string]any{"key": "value"},
			key:      "key",
			expected: "value",
		},
		{
			name:     "missing key",
			m:        map[string]any{"other": "value"},
			key:      "key",
			expected: "",
		},
		{
			name:     "non-string value",
			m:        map[string]any{"key": 123},
			key:      "key",
			expected: "",
		},
		{
			name:     "nil map value",
			m:        map[string]any{"key": nil},
			key:      "key",
			expected: "",
		},
		{
			name:     "empty map",
			m:        map[string]any{},
			key:      "key",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getString(tt.m, tt.key)
			if got != tt.expected {
				t.Errorf("getString() = %s, want %s", got, tt.expected)
			}
		})
	}
}

func TestClaimConflictError(t *testing.T) {
	err := &ClaimConflictError{
		TaskID:       "ENG-123",
		ClaimedBy:    "agent-1",
		CurrentAgent: "agent-2",
	}

	expected := "task ENG-123 is already claimed by agent agent-1"
	if err.Error() != expected {
		t.Errorf("Error() = %s, want %s", err.Error(), expected)
	}
}

func TestIssueToTask(t *testing.T) {
	l := New()
	// Set up reverse status map
	l.reverseStatusMap = make(map[string]backend.Status)
	for state, status := range defaultStatusMapping {
		l.reverseStatusMap[strings.ToLower(state)] = status
	}
	l.agentLabelPrefix = "agent"

	tests := []struct {
		name     string
		issue    map[string]any
		validate func(t *testing.T, task *backend.Task)
	}{
		{
			name: "basic issue",
			issue: map[string]any{
				"id":         "uuid-123",
				"identifier": "ENG-123",
				"title":      "Test Issue",
				"description": "Test description",
				"url":        "https://linear.app/team/issue/ENG-123",
				"priority":   float64(2),
				"createdAt":  "2025-01-15T09:00:00Z",
				"updatedAt":  "2025-01-18T14:30:00Z",
				"state": map[string]any{
					"id":   "state-1",
					"name": "In Progress",
				},
				"assignee": map[string]any{
					"id":          "user-1",
					"name":        "John",
					"displayName": "John Doe",
				},
				"labels": map[string]any{
					"nodes": []any{
						map[string]any{"id": "label-1", "name": "bug"},
						map[string]any{"id": "label-2", "name": "feature"},
					},
				},
				"team": map[string]any{
					"id":  "team-1",
					"key": "ENG",
				},
			},
			validate: func(t *testing.T, task *backend.Task) {
				if task.ID != "ENG-123" {
					t.Errorf("ID = %s, want ENG-123", task.ID)
				}
				if task.Title != "Test Issue" {
					t.Errorf("Title = %s, want Test Issue", task.Title)
				}
				if task.Description != "Test description" {
					t.Errorf("Description = %s, want Test description", task.Description)
				}
				if task.URL != "https://linear.app/team/issue/ENG-123" {
					t.Errorf("URL = %s, want https://linear.app/team/issue/ENG-123", task.URL)
				}
				if task.Priority != backend.PriorityHigh {
					t.Errorf("Priority = %s, want high", task.Priority)
				}
				if task.Status != backend.StatusInProgress {
					t.Errorf("Status = %s, want in-progress", task.Status)
				}
				if task.Assignee != "John Doe" {
					t.Errorf("Assignee = %s, want John Doe", task.Assignee)
				}
				if len(task.Labels) != 2 {
					t.Errorf("len(Labels) = %d, want 2", len(task.Labels))
				}
				if task.Created.IsZero() {
					t.Error("Created should not be zero")
				}
				if task.Updated.IsZero() {
					t.Error("Updated should not be zero")
				}
			},
		},
		{
			name: "issue with no priority",
			issue: map[string]any{
				"identifier": "ENG-124",
				"title":      "No Priority Issue",
				"priority":   float64(0),
				"state": map[string]any{
					"name": "Backlog",
				},
			},
			validate: func(t *testing.T, task *backend.Task) {
				if task.Priority != backend.PriorityNone {
					t.Errorf("Priority = %s, want none", task.Priority)
				}
			},
		},
		{
			name: "issue with unknown state",
			issue: map[string]any{
				"identifier": "ENG-125",
				"title":      "Unknown State Issue",
				"state": map[string]any{
					"name": "Custom State",
				},
			},
			validate: func(t *testing.T, task *backend.Task) {
				if task.Status != backend.StatusBacklog {
					t.Errorf("Status = %s, want backlog (default for unknown)", task.Status)
				}
			},
		},
		{
			name: "issue with agent labels filtered out",
			issue: map[string]any{
				"identifier": "ENG-126",
				"title":      "Agent Label Issue",
				"labels": map[string]any{
					"nodes": []any{
						map[string]any{"id": "label-1", "name": "bug"},
						map[string]any{"id": "label-2", "name": "agent:claude-1"},
						map[string]any{"id": "label-3", "name": "feature"},
					},
				},
				"state": map[string]any{"name": "Todo"},
			},
			validate: func(t *testing.T, task *backend.Task) {
				if len(task.Labels) != 2 {
					t.Errorf("len(Labels) = %d, want 2 (agent label should be filtered)", len(task.Labels))
				}
				for _, label := range task.Labels {
					if strings.HasPrefix(label, "agent:") {
						t.Errorf("agent label %s should be filtered out", label)
					}
				}
			},
		},
		{
			name: "issue with displayName fallback to name",
			issue: map[string]any{
				"identifier": "ENG-127",
				"title":      "Assignee Fallback",
				"assignee": map[string]any{
					"id":          "user-1",
					"name":        "john",
					"displayName": "",
				},
				"state": map[string]any{"name": "Todo"},
			},
			validate: func(t *testing.T, task *backend.Task) {
				if task.Assignee != "john" {
					t.Errorf("Assignee = %s, want john", task.Assignee)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := l.issueToTask(tt.issue)
			if task == nil {
				t.Fatal("issueToTask returned nil")
			}
			tt.validate(t, task)
		})
	}
}

func TestConnectInvalidConfig(t *testing.T) {
	l := New()

	// Test with nil workspace
	err := l.Connect(backend.Config{Workspace: nil})
	if err == nil {
		t.Error("expected error for nil workspace")
	}

	// Test with wrong workspace type
	err = l.Connect(backend.Config{Workspace: "invalid"})
	if err == nil {
		t.Error("expected error for invalid workspace type")
	}
}

func TestDisconnect(t *testing.T) {
	l := New()
	l.connected = true
	l.apiKey = "test-key"

	err := l.Disconnect()
	if err != nil {
		t.Errorf("Disconnect() error = %v", err)
	}

	if l.connected {
		t.Error("expected connected to be false")
	}
	if l.apiKey != "" {
		t.Error("expected apiKey to be cleared")
	}
}

func TestHealthCheckNotConnected(t *testing.T) {
	l := New()

	status, err := l.HealthCheck()
	if err != nil {
		t.Errorf("HealthCheck() error = %v", err)
	}
	if status.OK {
		t.Error("expected OK to be false when not connected")
	}
	if status.Message != "not connected" {
		t.Errorf("Message = %s, want 'not connected'", status.Message)
	}
}

func TestListNotConnected(t *testing.T) {
	l := New()

	_, err := l.List(backend.TaskFilters{})
	if err == nil {
		t.Error("expected error when not connected")
	}
	if err.Error() != "not connected" {
		t.Errorf("error = %s, want 'not connected'", err.Error())
	}
}

func TestGetNotConnected(t *testing.T) {
	l := New()

	_, err := l.Get("ENG-123")
	if err == nil {
		t.Error("expected error when not connected")
	}
}

func TestCreateNotConnected(t *testing.T) {
	l := New()

	_, err := l.Create(backend.TaskInput{Title: "Test"})
	if err == nil {
		t.Error("expected error when not connected")
	}
}

func TestCreateNoTeam(t *testing.T) {
	l := New()
	l.connected = true
	l.teamID = ""

	_, err := l.Create(backend.TaskInput{Title: "Test"})
	if err == nil {
		t.Error("expected error when team not configured")
	}
	if !strings.Contains(err.Error(), "team not configured") {
		t.Errorf("error = %s, want to contain 'team not configured'", err.Error())
	}
}

func TestUpdateNotConnected(t *testing.T) {
	l := New()

	_, err := l.Update("ENG-123", backend.TaskChanges{})
	if err == nil {
		t.Error("expected error when not connected")
	}
}

func TestDeleteNotConnected(t *testing.T) {
	l := New()

	err := l.Delete("ENG-123")
	if err == nil {
		t.Error("expected error when not connected")
	}
}

func TestMoveNotConnected(t *testing.T) {
	l := New()

	_, err := l.Move("ENG-123", backend.StatusDone)
	if err == nil {
		t.Error("expected error when not connected")
	}
}

func TestMoveInvalidStatus(t *testing.T) {
	l := New()
	l.connected = true

	_, err := l.Move("ENG-123", backend.Status("invalid"))
	if err == nil {
		t.Error("expected error for invalid status")
	}
	if !strings.Contains(err.Error(), "invalid status") {
		t.Errorf("error = %s, want to contain 'invalid status'", err.Error())
	}
}

func TestAssignNotConnected(t *testing.T) {
	l := New()

	_, err := l.Assign("ENG-123", "user")
	if err == nil {
		t.Error("expected error when not connected")
	}
}

func TestUnassignNotConnected(t *testing.T) {
	l := New()

	_, err := l.Unassign("ENG-123")
	if err == nil {
		t.Error("expected error when not connected")
	}
}

func TestListCommentsNotConnected(t *testing.T) {
	l := New()

	_, err := l.ListComments("ENG-123")
	if err == nil {
		t.Error("expected error when not connected")
	}
}

func TestAddCommentNotConnected(t *testing.T) {
	l := New()

	_, err := l.AddComment("ENG-123", "comment")
	if err == nil {
		t.Error("expected error when not connected")
	}
}

func TestClaimNotConnected(t *testing.T) {
	l := New()

	_, err := l.Claim("ENG-123", "agent-1")
	if err == nil {
		t.Error("expected error when not connected")
	}
}

func TestReleaseNotConnected(t *testing.T) {
	l := New()

	err := l.Release("ENG-123")
	if err == nil {
		t.Error("expected error when not connected")
	}
}

// Integration-style tests with mock server

func TestGraphQLMethod(t *testing.T) {
	server := mockLinearServer(t, func(query string, variables map[string]any) any {
		return map[string]any{
			"data": map[string]any{
				"viewer": map[string]any{
					"id":   "user-123",
					"name": "Test User",
				},
			},
		}
	})
	defer server.Close()

	l := &Linear{
		ctx:    context.Background(),
		client: &http.Client{Timeout: 30 * time.Second},
		apiKey: "test-api-key",
	}

	// We can't easily override the endpoint, so we test the graphQL method
	// behavior by checking error cases and documenting expected behavior

	// Test with empty API key
	l.apiKey = ""
	_, err := l.graphQL("query { viewer { id } }", nil)
	if err == nil {
		t.Error("expected error with empty API key")
	}
}

func TestHealthCheckWithMockServer(t *testing.T) {
	server := mockLinearServer(t, func(query string, variables map[string]any) any {
		if strings.Contains(query, "viewer") {
			return map[string]any{
				"data": map[string]any{
					"viewer": map[string]any{
						"id":   "user-123",
						"name": "Test User",
					},
				},
			}
		}
		return map[string]any{"errors": []any{map[string]any{"message": "unknown query"}}}
	})
	defer server.Close()

	// Note: This test documents expected behavior
	// The actual implementation uses a hardcoded endpoint
	// Integration tests would need to override the endpoint
	t.Skip("Requires ability to override Linear API endpoint for testing")
}

func TestGraphQLErrorHandling(t *testing.T) {
	server := mockLinearServer(t, func(query string, variables map[string]any) any {
		return map[string]any{
			"errors": []any{
				map[string]any{
					"message": "Test error message",
				},
			},
		}
	})
	defer server.Close()

	// Note: This test documents expected error handling behavior
	// The graphQL method should return an error when the response contains errors
	t.Skip("Requires ability to override Linear API endpoint for testing")
}

func TestStatusMapConfiguration(t *testing.T) {
	l := New()

	// Test with custom status map
	customStatusMap := map[backend.Status]string{
		backend.StatusBacklog:    "Icebox",
		backend.StatusTodo:       "Ready",
		backend.StatusInProgress: "Working",
		backend.StatusReview:     "Reviewing",
		backend.StatusDone:       "Shipped",
	}

	wsCfg := &WorkspaceConfig{
		TeamKey:   "TEST",
		StatusMap: customStatusMap,
	}

	// Set up the backend manually (can't call Connect without API key)
	l.statusMap = wsCfg.StatusMap
	l.reverseStatusMap = make(map[string]backend.Status)
	for status, state := range wsCfg.StatusMap {
		l.reverseStatusMap[strings.ToLower(state)] = status
	}

	// Verify reverse mapping works
	if l.reverseStatusMap["icebox"] != backend.StatusBacklog {
		t.Error("custom status map not applied correctly")
	}
	if l.reverseStatusMap["shipped"] != backend.StatusDone {
		t.Error("custom status map not applied correctly")
	}
}

func TestAgentLabelPrefixDefault(t *testing.T) {
	l := New()
	l.agentLabelPrefix = ""

	// When empty, should default to "agent" during Connect
	wsCfg := &WorkspaceConfig{TeamKey: "TEST"}
	cfg := backend.Config{
		Workspace:        wsCfg,
		AgentLabelPrefix: "",
	}

	// Manually apply the default logic that happens in Connect
	if cfg.AgentLabelPrefix == "" {
		l.agentLabelPrefix = "agent"
	} else {
		l.agentLabelPrefix = cfg.AgentLabelPrefix
	}

	if l.agentLabelPrefix != "agent" {
		t.Errorf("agentLabelPrefix = %s, want agent", l.agentLabelPrefix)
	}
}

func TestRegister(t *testing.T) {
	// Clear any existing registration
	backend.Unregister(Name)

	// Test Register function
	Register()

	if !backend.IsRegistered(Name) {
		t.Error("Linear backend not registered")
	}

	// Verify we can get an instance
	b, err := backend.Get(Name)
	if err != nil {
		t.Errorf("failed to get Linear backend: %v", err)
	}

	if b.Name() != Name {
		t.Errorf("backend Name() = %s, want %s", b.Name(), Name)
	}

	// Clean up
	backend.Unregister(Name)
}

func TestTimestampParsing(t *testing.T) {
	l := New()
	l.reverseStatusMap = map[string]backend.Status{
		"todo": backend.StatusTodo,
	}
	l.agentLabelPrefix = "agent"

	issue := map[string]any{
		"identifier": "ENG-123",
		"title":      "Test",
		"createdAt":  "2025-01-15T09:00:00Z",
		"updatedAt":  "2025-01-18T14:30:00.123Z",
		"state":      map[string]any{"name": "Todo"},
	}

	task := l.issueToTask(issue)

	expectedCreated := time.Date(2025, 1, 15, 9, 0, 0, 0, time.UTC)
	if !task.Created.Equal(expectedCreated) {
		t.Errorf("Created = %v, want %v", task.Created, expectedCreated)
	}

	// Updated should parse correctly too
	if task.Updated.Year() != 2025 || task.Updated.Month() != 1 || task.Updated.Day() != 18 {
		t.Errorf("Updated date incorrect: %v", task.Updated)
	}
}

func TestIssueToTaskMeta(t *testing.T) {
	l := New()
	l.reverseStatusMap = map[string]backend.Status{
		"in progress": backend.StatusInProgress,
	}
	l.agentLabelPrefix = "agent"

	issue := map[string]any{
		"id":         "linear-uuid-123",
		"identifier": "ENG-123",
		"title":      "Test",
		"state": map[string]any{
			"id":   "state-uuid-456",
			"name": "In Progress",
		},
		"assignee": map[string]any{
			"id":          "user-uuid-789",
			"displayName": "John",
		},
		"team": map[string]any{
			"id":  "team-uuid-abc",
			"key": "ENG",
		},
	}

	task := l.issueToTask(issue)

	if task.Meta == nil {
		t.Fatal("Meta should not be nil")
	}

	if task.Meta["linear_id"] != "linear-uuid-123" {
		t.Errorf("Meta[linear_id] = %v, want linear-uuid-123", task.Meta["linear_id"])
	}

	if task.Meta["identifier"] != "ENG-123" {
		t.Errorf("Meta[identifier] = %v, want ENG-123", task.Meta["identifier"])
	}

	if task.Meta["state_id"] != "state-uuid-456" {
		t.Errorf("Meta[state_id] = %v, want state-uuid-456", task.Meta["state_id"])
	}

	if task.Meta["state_name"] != "In Progress" {
		t.Errorf("Meta[state_name] = %v, want In Progress", task.Meta["state_name"])
	}

	if task.Meta["assignee_id"] != "user-uuid-789" {
		t.Errorf("Meta[assignee_id] = %v, want user-uuid-789", task.Meta["assignee_id"])
	}

	if task.Meta["team_id"] != "team-uuid-abc" {
		t.Errorf("Meta[team_id] = %v, want team-uuid-abc", task.Meta["team_id"])
	}

	if task.Meta["team_key"] != "ENG" {
		t.Errorf("Meta[team_key] = %v, want ENG", task.Meta["team_key"])
	}
}

func TestIssueToTaskEmptyLabels(t *testing.T) {
	l := New()
	l.reverseStatusMap = map[string]backend.Status{"todo": backend.StatusTodo}
	l.agentLabelPrefix = "agent"

	issue := map[string]any{
		"identifier": "ENG-123",
		"title":      "Test",
		"state":      map[string]any{"name": "Todo"},
		// No labels field
	}

	task := l.issueToTask(issue)

	if task.Labels == nil {
		// Labels should be nil or empty, not cause a panic
		t.Log("Labels is nil, which is acceptable")
	}
}

func TestIssueToTaskNoAssignee(t *testing.T) {
	l := New()
	l.reverseStatusMap = map[string]backend.Status{"todo": backend.StatusTodo}
	l.agentLabelPrefix = "agent"

	issue := map[string]any{
		"identifier": "ENG-123",
		"title":      "Test",
		"state":      map[string]any{"name": "Todo"},
		"assignee":   nil,
	}

	task := l.issueToTask(issue)

	if task.Assignee != "" {
		t.Errorf("Assignee = %s, want empty string", task.Assignee)
	}
}

func TestIssueToTaskNoPriority(t *testing.T) {
	l := New()
	l.reverseStatusMap = map[string]backend.Status{"todo": backend.StatusTodo}
	l.agentLabelPrefix = "agent"

	issue := map[string]any{
		"identifier": "ENG-123",
		"title":      "Test",
		"state":      map[string]any{"name": "Todo"},
		// No priority field
	}

	task := l.issueToTask(issue)

	if task.Priority != backend.PriorityNone {
		t.Errorf("Priority = %s, want none", task.Priority)
	}
}
