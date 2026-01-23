// Package support provides test helpers and fixtures for the backlog CLI specs.
package support

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"time"
)

// MockLinearIssue represents an issue in the mock Linear API.
type MockLinearIssue struct {
	ID          string
	Identifier  string // e.g., "ENG-123"
	Title       string
	Description string
	State       string // e.g., "Todo", "In Progress", "Done"
	Priority    int    // 0 = No priority, 1 = Urgent, 2 = High, 3 = Medium, 4 = Low
	Assignee    string
	Labels      []string
	TeamKey     string // e.g., "ENG"
}

// MockLinearTeam represents a team in the mock Linear API.
type MockLinearTeam struct {
	ID   string
	Key  string
	Name string
}

// MockLinearState represents a workflow state in Linear.
type MockLinearState struct {
	ID   string
	Name string
	Type string // e.g., "backlog", "unstarted", "started", "completed", "canceled"
}

// MockLinearServer provides a mock implementation of the Linear GraphQL API for testing.
type MockLinearServer struct {
	Server *httptest.Server
	URL    string

	// mu protects all fields below
	mu sync.RWMutex

	// Issues stored by ID
	Issues map[string]*MockLinearIssue

	// Teams stored by key
	Teams map[string]*MockLinearTeam

	// States stored by name (workflow states)
	States map[string]*MockLinearState

	// ExpectedAPIKey if set, validates Authorization header
	ExpectedAPIKey string

	// AuthErrorEnabled if true, returns errors for invalid API keys
	AuthErrorEnabled bool

	// AuthenticatedUser is the username to return for viewer query
	AuthenticatedUser string

	// NextIssueNumber is the next issue number to assign
	NextIssueNumber int
}

// NewMockLinearServer creates and starts a new mock Linear API server.
func NewMockLinearServer() *MockLinearServer {
	mock := &MockLinearServer{
		Issues:            make(map[string]*MockLinearIssue),
		Teams:             make(map[string]*MockLinearTeam),
		States:            make(map[string]*MockLinearState),
		AuthenticatedUser: "test-user",
		NextIssueNumber:   1,
	}

	// Set up default team
	mock.Teams["ENG"] = &MockLinearTeam{
		ID:   "team-eng-id",
		Key:  "ENG",
		Name: "Engineering",
	}

	// Set up default workflow states
	defaultStates := []MockLinearState{
		{ID: "state-backlog", Name: "Backlog", Type: "backlog"},
		{ID: "state-todo", Name: "Todo", Type: "unstarted"},
		{ID: "state-inprogress", Name: "In Progress", Type: "started"},
		{ID: "state-review", Name: "In Review", Type: "started"},
		{ID: "state-done", Name: "Done", Type: "completed"},
	}
	for _, state := range defaultStates {
		s := state // Create a copy for the pointer
		mock.States[state.Name] = &s
	}

	mux := http.NewServeMux()

	// POST /graphql - Linear's GraphQL API endpoint
	mux.HandleFunc("/graphql", mock.handleGraphQL)

	mock.Server = httptest.NewServer(mux)
	mock.URL = mock.Server.URL

	return mock
}

// Close shuts down the mock server.
func (m *MockLinearServer) Close() {
	if m.Server != nil {
		m.Server.Close()
	}
}

// SetIssues sets the mock issues.
func (m *MockLinearServer) SetIssues(issues []MockLinearIssue) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Issues = make(map[string]*MockLinearIssue)
	for i := range issues {
		issue := issues[i]
		m.Issues[issue.ID] = &issue
	}
}

// GetIssue retrieves an issue by ID for assertions.
func (m *MockLinearServer) GetIssue(id string) *MockLinearIssue {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.Issues[id]
}

// GetIssueByIdentifier retrieves an issue by identifier (e.g., "ENG-123").
func (m *MockLinearServer) GetIssueByIdentifier(identifier string) *MockLinearIssue {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, issue := range m.Issues {
		if issue.Identifier == identifier {
			return issue
		}
	}
	return nil
}

// validateAuth checks the Authorization header and returns an error response if invalid.
func (m *MockLinearServer) validateAuth(w http.ResponseWriter, r *http.Request) bool {
	m.mu.RLock()
	expectedKey := m.ExpectedAPIKey
	authErrorEnabled := m.AuthErrorEnabled
	m.mu.RUnlock()

	auth := r.Header.Get("Authorization")

	// If auth error is enabled, check for "invalid" keys
	if authErrorEnabled {
		if auth == "" || auth == "invalid_key" || auth == "Bearer invalid_key" {
			m.writeGraphQLError(w, "Authentication required", "AUTHENTICATION_ERROR")
			return false
		}
	}

	// If expected key is set, validate it
	if expectedKey != "" {
		expectedAuth := expectedKey
		expectedAuthBearer := "Bearer " + expectedKey
		if auth != expectedAuth && auth != expectedAuthBearer {
			m.writeGraphQLError(w, "Invalid API key", "AUTHENTICATION_ERROR")
			return false
		}
	}

	return true
}

// handleGraphQL handles POST /graphql requests.
func (m *MockLinearServer) handleGraphQL(w http.ResponseWriter, r *http.Request) {
	if !m.validateAuth(w, r) {
		return
	}

	if r.Method != http.MethodPost {
		m.writeGraphQLError(w, "Method not allowed", "METHOD_NOT_ALLOWED")
		return
	}

	var req struct {
		Query     string                 `json:"query"`
		Variables map[string]interface{} `json:"variables"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		m.writeGraphQLError(w, "Invalid JSON: "+err.Error(), "BAD_REQUEST")
		return
	}

	// Parse and handle different GraphQL queries/mutations
	query := strings.TrimSpace(req.Query)

	// Route to appropriate handler based on query content
	switch {
	case strings.Contains(query, "viewer"):
		m.handleViewerQuery(w)
	case strings.Contains(query, "issues") || strings.Contains(query, "Issues"):
		m.handleIssuesQuery(w, req.Variables)
	case strings.Contains(query, "issue") && strings.Contains(query, "create"):
		m.handleCreateIssue(w, req.Variables)
	case strings.Contains(query, "issue") && strings.Contains(query, "update"):
		m.handleUpdateIssue(w, req.Variables)
	case strings.Contains(query, "team"):
		m.handleTeamQuery(w, req.Variables)
	default:
		// Default: return empty data for unrecognized queries
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": map[string]interface{}{},
		})
	}
}

// handleViewerQuery handles the viewer query (authenticated user info).
func (m *MockLinearServer) handleViewerQuery(w http.ResponseWriter) {
	m.mu.RLock()
	user := m.AuthenticatedUser
	m.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"data": map[string]interface{}{
			"viewer": map[string]interface{}{
				"id":    "user-id-123",
				"name":  user,
				"email": user + "@example.com",
			},
		},
	})
}

// handleIssuesQuery handles queries for listing issues.
func (m *MockLinearServer) handleIssuesQuery(w http.ResponseWriter, variables map[string]interface{}) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Build issues list
	var issueNodes []map[string]interface{}
	for _, issue := range m.Issues {
		// Apply team filter if specified
		if teamKey, ok := variables["teamKey"].(string); ok && teamKey != "" {
			if issue.TeamKey != teamKey {
				continue
			}
		}

		issueNode := m.issueToGraphQL(issue)
		issueNodes = append(issueNodes, issueNode)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"data": map[string]interface{}{
			"issues": map[string]interface{}{
				"nodes": issueNodes,
				"pageInfo": map[string]interface{}{
					"hasNextPage": false,
					"endCursor":   nil,
				},
			},
		},
	})
}

// handleCreateIssue handles issue creation mutations.
func (m *MockLinearServer) handleCreateIssue(w http.ResponseWriter, variables map[string]interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()

	input, ok := variables["input"].(map[string]interface{})
	if !ok {
		m.writeGraphQLError(w, "Invalid input", "BAD_REQUEST")
		return
	}

	// Generate new issue
	teamKey := "ENG"
	if tk, ok := input["teamId"].(string); ok {
		// Look up team by ID
		for _, team := range m.Teams {
			if team.ID == tk {
				teamKey = team.Key
				break
			}
		}
	}

	issueID := generateID()
	identifier := teamKey + "-" + string(rune('0'+m.NextIssueNumber))
	m.NextIssueNumber++

	issue := &MockLinearIssue{
		ID:         issueID,
		Identifier: identifier,
		Title:      getString(input, "title"),
		TeamKey:    teamKey,
		State:      "Backlog",
	}

	if desc, ok := input["description"].(string); ok {
		issue.Description = desc
	}
	if priority, ok := input["priority"].(float64); ok {
		issue.Priority = int(priority)
	}

	m.Issues[issueID] = issue

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"data": map[string]interface{}{
			"issueCreate": map[string]interface{}{
				"success": true,
				"issue":   m.issueToGraphQL(issue),
			},
		},
	})
}

// handleUpdateIssue handles issue update mutations.
func (m *MockLinearServer) handleUpdateIssue(w http.ResponseWriter, variables map[string]interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()

	issueID, ok := variables["id"].(string)
	if !ok {
		m.writeGraphQLError(w, "Issue ID required", "BAD_REQUEST")
		return
	}

	issue, exists := m.Issues[issueID]
	if !exists {
		m.writeGraphQLError(w, "Issue not found", "NOT_FOUND")
		return
	}

	input, ok := variables["input"].(map[string]interface{})
	if !ok {
		m.writeGraphQLError(w, "Invalid input", "BAD_REQUEST")
		return
	}

	// Apply updates
	if title, ok := input["title"].(string); ok {
		issue.Title = title
	}
	if desc, ok := input["description"].(string); ok {
		issue.Description = desc
	}
	if stateID, ok := input["stateId"].(string); ok {
		// Look up state by ID
		for _, state := range m.States {
			if state.ID == stateID {
				issue.State = state.Name
				break
			}
		}
	}
	if priority, ok := input["priority"].(float64); ok {
		issue.Priority = int(priority)
	}
	if assigneeID, ok := input["assigneeId"].(string); ok {
		issue.Assignee = assigneeID
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"data": map[string]interface{}{
			"issueUpdate": map[string]interface{}{
				"success": true,
				"issue":   m.issueToGraphQL(issue),
			},
		},
	})
}

// handleTeamQuery handles team queries.
func (m *MockLinearServer) handleTeamQuery(w http.ResponseWriter, variables map[string]interface{}) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	teamKey, _ := variables["key"].(string)

	team, exists := m.Teams[teamKey]
	if !exists {
		m.writeGraphQLError(w, "Team not found", "NOT_FOUND")
		return
	}

	// Build states list
	var stateNodes []map[string]interface{}
	for _, state := range m.States {
		stateNodes = append(stateNodes, map[string]interface{}{
			"id":   state.ID,
			"name": state.Name,
			"type": state.Type,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"data": map[string]interface{}{
			"team": map[string]interface{}{
				"id":   team.ID,
				"key":  team.Key,
				"name": team.Name,
				"states": map[string]interface{}{
					"nodes": stateNodes,
				},
			},
		},
	})
}

// issueToGraphQL converts a MockLinearIssue to GraphQL response format.
func (m *MockLinearServer) issueToGraphQL(issue *MockLinearIssue) map[string]interface{} {
	result := map[string]interface{}{
		"id":          issue.ID,
		"identifier":  issue.Identifier,
		"title":       issue.Title,
		"description": issue.Description,
		"priority":    issue.Priority,
		"createdAt":   time.Now().Format(time.RFC3339),
		"updatedAt":   time.Now().Format(time.RFC3339),
		"url":         "https://linear.app/team/" + issue.TeamKey + "/issue/" + issue.Identifier,
	}

	// Add state
	if state, exists := m.States[issue.State]; exists {
		result["state"] = map[string]interface{}{
			"id":   state.ID,
			"name": state.Name,
			"type": state.Type,
		}
	} else {
		result["state"] = map[string]interface{}{
			"id":   "state-unknown",
			"name": issue.State,
			"type": "backlog",
		}
	}

	// Add team
	if team, exists := m.Teams[issue.TeamKey]; exists {
		result["team"] = map[string]interface{}{
			"id":   team.ID,
			"key":  team.Key,
			"name": team.Name,
		}
	}

	// Add assignee if set
	if issue.Assignee != "" {
		result["assignee"] = map[string]interface{}{
			"id":   issue.Assignee,
			"name": issue.Assignee,
		}
	} else {
		result["assignee"] = nil
	}

	// Add labels
	var labelNodes []map[string]interface{}
	for _, label := range issue.Labels {
		labelNodes = append(labelNodes, map[string]interface{}{
			"id":   "label-" + label,
			"name": label,
		})
	}
	result["labels"] = map[string]interface{}{
		"nodes": labelNodes,
	}

	return result
}

// writeGraphQLError writes a GraphQL-style error response.
func (m *MockLinearServer) writeGraphQLError(w http.ResponseWriter, message, code string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK) // GraphQL returns 200 even for errors
	json.NewEncoder(w).Encode(map[string]interface{}{
		"errors": []map[string]interface{}{
			{
				"message": message,
				"extensions": map[string]interface{}{
					"code": code,
				},
			},
		},
	})
}

// Helper functions

func generateID() string {
	return "id-" + time.Now().Format("20060102150405.000000000")
}

func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}
