// Package support provides test helpers and fixtures for the backlog CLI specs.
package support

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

// MockGitHubIssue represents an issue in the mock GitHub API.
type MockGitHubIssue struct {
	Number   int
	Title    string
	State    string
	Labels   []string
	Assignee string
	Body     string
}

// MockGitHubComment represents a comment on a GitHub issue.
type MockGitHubComment struct {
	ID     int
	Author string
	Body   string
}

// MockGitHubProjectColumn represents a column in a GitHub Project.
type MockGitHubProjectColumn struct {
	ID   string
	Name string
}

// MockGitHubProject represents a GitHub Project v2.
type MockGitHubProject struct {
	ID      int
	Title   string
	Columns []MockGitHubProjectColumn
}

// MockGitHubProjectItem represents an item (issue) in a GitHub Project.
type MockGitHubProjectItem struct {
	IssueNumber int
	ColumnID    string
}

// MockGitHubServer provides a mock implementation of the GitHub API for testing.
type MockGitHubServer struct {
	Server *httptest.Server
	URL    string

	// mu protects all fields below
	mu sync.RWMutex

	// Issues stored by issue number
	Issues map[int]*MockGitHubIssue

	// Comments stored by issue number
	Comments map[int][]MockGitHubComment

	// ExpectedToken if set, validates Authorization header
	ExpectedToken string

	// AuthErrorEnabled if true, returns 401 for invalid tokens
	AuthErrorEnabled bool

	// AuthenticatedUser is the username to return for /user endpoint
	AuthenticatedUser string

	// NextIssueNumber is the next issue number to assign
	NextIssueNumber int

	// NextCommentID is the next comment ID to assign
	NextCommentID int

	// Projects stored by project number
	Projects map[int]*MockGitHubProject

	// ProjectItems stored by project ID, maps issue number to project item
	ProjectItems map[int]map[int]*MockGitHubProjectItem

	// InvalidProjectIDs tracks project IDs that should return errors
	InvalidProjectIDs map[int]bool
}

// NewMockGitHubServer creates and starts a new mock GitHub API server.
func NewMockGitHubServer() *MockGitHubServer {
	mock := &MockGitHubServer{
		Issues:            make(map[int]*MockGitHubIssue),
		Comments:          make(map[int][]MockGitHubComment),
		AuthenticatedUser: "test-user",
		NextIssueNumber:   1,
		NextCommentID:     1,
		Projects:          make(map[int]*MockGitHubProject),
		ProjectItems:      make(map[int]map[int]*MockGitHubProjectItem),
		InvalidProjectIDs: make(map[int]bool),
	}

	mux := http.NewServeMux()

	// GET /user - authenticated user info
	mux.HandleFunc("/user", mock.handleUser)
	mux.HandleFunc("/api/v3/user", mock.handleUser)

	// POST /graphql - GraphQL API for Projects v2
	mux.HandleFunc("/graphql", mock.handleGraphQL)
	mux.HandleFunc("/api/graphql", mock.handleGraphQL)

	// GET /repos/{owner}/{repo}/issues - list issues
	// POST /repos/{owner}/{repo}/issues - create issue
	mux.HandleFunc("/repos/", mock.handleRepos)
	mux.HandleFunc("/api/v3/repos/", mock.handleRepos)

	mock.Server = httptest.NewServer(mux)
	mock.URL = mock.Server.URL

	return mock
}

// Close shuts down the mock server.
func (m *MockGitHubServer) Close() {
	if m.Server != nil {
		m.Server.Close()
	}
}

// SetProject sets a mock project with the given columns.
func (m *MockGitHubServer) SetProject(projectID int, title string, columns []MockGitHubProjectColumn) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Projects[projectID] = &MockGitHubProject{
		ID:      projectID,
		Title:   title,
		Columns: columns,
	}
	if m.ProjectItems[projectID] == nil {
		m.ProjectItems[projectID] = make(map[int]*MockGitHubProjectItem)
	}
}

// SetProjectItem adds an issue to a project in a specific column.
func (m *MockGitHubServer) SetProjectItem(projectID int, issueNumber int, columnID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.ProjectItems[projectID] == nil {
		m.ProjectItems[projectID] = make(map[int]*MockGitHubProjectItem)
	}
	m.ProjectItems[projectID][issueNumber] = &MockGitHubProjectItem{
		IssueNumber: issueNumber,
		ColumnID:    columnID,
	}
}

// GetProjectItem retrieves a project item for assertions.
func (m *MockGitHubServer) GetProjectItem(projectID int, issueNumber int) *MockGitHubProjectItem {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if items, ok := m.ProjectItems[projectID]; ok {
		return items[issueNumber]
	}
	return nil
}

// GetProject retrieves a project for assertions.
func (m *MockGitHubServer) GetProject(projectID int) *MockGitHubProject {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.Projects[projectID]
}

// SetInvalidProjectID marks a project ID as invalid (will return error).
func (m *MockGitHubServer) SetInvalidProjectID(projectID int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.InvalidProjectIDs[projectID] = true
}

// SetIssues sets the mock issues.
func (m *MockGitHubServer) SetIssues(issues []MockGitHubIssue) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Issues = make(map[int]*MockGitHubIssue)
	for i := range issues {
		issue := issues[i]
		m.Issues[issue.Number] = &issue
		if issue.Number >= m.NextIssueNumber {
			m.NextIssueNumber = issue.Number + 1
		}
	}
}

// SetComments sets the mock comments for an issue.
func (m *MockGitHubServer) SetComments(issueNumber int, comments []MockGitHubComment) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Comments[issueNumber] = comments
}

// GetIssue retrieves an issue by number for assertions.
// Returns nil if the issue does not exist.
func (m *MockGitHubServer) GetIssue(number int) *MockGitHubIssue {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.Issues[number]
}

// validateAuth checks the Authorization header and returns an error response if invalid.
func (m *MockGitHubServer) validateAuth(w http.ResponseWriter, r *http.Request) bool {
	m.mu.RLock()
	expectedToken := m.ExpectedToken
	authErrorEnabled := m.AuthErrorEnabled
	m.mu.RUnlock()

	auth := r.Header.Get("Authorization")

	// If auth error is enabled, check for "invalid" tokens
	if authErrorEnabled {
		if auth == "" || auth == "token invalid_token" || auth == "Bearer invalid_token" {
			m.writeError(w, http.StatusUnauthorized, "Bad credentials", "Bad credentials")
			return false
		}
	}

	// If expected token is set, validate it
	if expectedToken != "" {
		expectedAuth := "token " + expectedToken
		expectedAuthBearer := "Bearer " + expectedToken
		if auth != expectedAuth && auth != expectedAuthBearer {
			m.writeError(w, http.StatusUnauthorized, "Bad credentials", "Bad credentials")
			return false
		}
	}

	return true
}

// handleUser handles GET /user requests.
func (m *MockGitHubServer) handleUser(w http.ResponseWriter, r *http.Request) {
	if !m.validateAuth(w, r) {
		return
	}

	m.mu.RLock()
	user := m.AuthenticatedUser
	m.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"login": user,
		"id":    1,
		"type":  "User",
	})
}

// handleRepos handles requests to /repos/{owner}/{repo}/...
func (m *MockGitHubServer) handleRepos(w http.ResponseWriter, r *http.Request) {
	if !m.validateAuth(w, r) {
		return
	}

	path := r.URL.Path
	// Strip /api/v3 prefix if present (for enterprise URL compatibility)
	path = strings.TrimPrefix(path, "/api/v3")

	// Parse the path: /repos/{owner}/{repo}/...
	// Match patterns:
	// /repos/{owner}/{repo}/issues
	// /repos/{owner}/{repo}/issues/{number}
	// /repos/{owner}/{repo}/issues/{number}/comments
	// /repos/{owner}/{repo}/issues/{number}/labels
	// /repos/{owner}/{repo}/issues/{number}/labels/{name}

	issuesListPattern := regexp.MustCompile(`^/repos/[^/]+/[^/]+/issues$`)
	issuePattern := regexp.MustCompile(`^/repos/[^/]+/[^/]+/issues/(\d+)$`)
	commentsPattern := regexp.MustCompile(`^/repos/[^/]+/[^/]+/issues/(\d+)/comments$`)
	labelsPattern := regexp.MustCompile(`^/repos/[^/]+/[^/]+/issues/(\d+)/labels$`)
	labelPattern := regexp.MustCompile(`^/repos/[^/]+/[^/]+/issues/(\d+)/labels/(.+)$`)

	switch {
	case issuesListPattern.MatchString(path):
		m.handleIssuesList(w, r)
	case commentsPattern.MatchString(path):
		matches := commentsPattern.FindStringSubmatch(path)
		issueNumber, _ := strconv.Atoi(matches[1])
		m.handleComments(w, r, issueNumber)
	case labelsPattern.MatchString(path):
		matches := labelsPattern.FindStringSubmatch(path)
		issueNumber, _ := strconv.Atoi(matches[1])
		m.handleLabels(w, r, issueNumber)
	case labelPattern.MatchString(path):
		matches := labelPattern.FindStringSubmatch(path)
		issueNumber, _ := strconv.Atoi(matches[1])
		labelName := matches[2]
		m.handleLabel(w, r, issueNumber, labelName)
	case issuePattern.MatchString(path):
		matches := issuePattern.FindStringSubmatch(path)
		issueNumber, _ := strconv.Atoi(matches[1])
		m.handleIssue(w, r, issueNumber)
	default:
		m.writeError(w, http.StatusNotFound, "Not Found", "Not Found")
	}
}

// handleIssuesList handles GET/POST /repos/{owner}/{repo}/issues
func (m *MockGitHubServer) handleIssuesList(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		m.listIssues(w, r)
	case http.MethodPost:
		m.createIssue(w, r)
	default:
		m.writeError(w, http.StatusMethodNotAllowed, "Method Not Allowed", "Method Not Allowed")
	}
}

// listIssues handles GET /repos/{owner}/{repo}/issues
func (m *MockGitHubServer) listIssues(w http.ResponseWriter, r *http.Request) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Parse query parameters
	query := r.URL.Query()
	state := query.Get("state")
	if state == "" {
		state = "open" // Default to open issues
	}
	labels := query.Get("labels")
	assignee := query.Get("assignee")
	perPage := 30
	if pp := query.Get("per_page"); pp != "" {
		if n, err := strconv.Atoi(pp); err == nil && n > 0 {
			perPage = n
		}
	}

	var issues []map[string]interface{}
	for _, issue := range m.Issues {
		// Filter by state
		if state != "all" && issue.State != state {
			continue
		}

		// Filter by labels
		if labels != "" {
			requiredLabels := strings.Split(labels, ",")
			hasAllLabels := true
			for _, required := range requiredLabels {
				found := false
				for _, label := range issue.Labels {
					if label == strings.TrimSpace(required) {
						found = true
						break
					}
				}
				if !found {
					hasAllLabels = false
					break
				}
			}
			if !hasAllLabels {
				continue
			}
		}

		// Filter by assignee
		if assignee != "" && issue.Assignee != assignee {
			continue
		}

		issues = append(issues, m.issueToJSON(issue))
	}

	// Apply limit
	if len(issues) > perPage {
		// Set Link header for pagination
		w.Header().Set("Link", `<https://api.github.com/next>; rel="next"`)
		issues = issues[:perPage]
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(issues)
}

// createIssue handles POST /repos/{owner}/{repo}/issues
func (m *MockGitHubServer) createIssue(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Title    string   `json:"title"`
		Body     string   `json:"body"`
		Labels   []string `json:"labels"`
		Assignee string   `json:"assignee"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		m.writeError(w, http.StatusBadRequest, "Invalid JSON", err.Error())
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	issue := &MockGitHubIssue{
		Number:   m.NextIssueNumber,
		Title:    input.Title,
		Body:     input.Body,
		Labels:   input.Labels,
		Assignee: input.Assignee,
		State:    "open",
	}
	m.NextIssueNumber++
	m.Issues[issue.Number] = issue

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(m.issueToJSON(issue))
}

// handleIssue handles GET/PATCH /repos/{owner}/{repo}/issues/{number}
func (m *MockGitHubServer) handleIssue(w http.ResponseWriter, r *http.Request, number int) {
	switch r.Method {
	case http.MethodGet:
		m.getIssue(w, r, number)
	case http.MethodPatch:
		m.updateIssue(w, r, number)
	default:
		m.writeError(w, http.StatusMethodNotAllowed, "Method Not Allowed", "Method Not Allowed")
	}
}

// getIssue handles GET /repos/{owner}/{repo}/issues/{number}
func (m *MockGitHubServer) getIssue(w http.ResponseWriter, _ *http.Request, number int) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	issue, ok := m.Issues[number]
	if !ok {
		m.writeError(w, http.StatusNotFound, "Not Found", fmt.Sprintf("Issue #%d not found", number))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(m.issueToJSON(issue))
}

// updateIssue handles PATCH /repos/{owner}/{repo}/issues/{number}
func (m *MockGitHubServer) updateIssue(w http.ResponseWriter, r *http.Request, number int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	issue, ok := m.Issues[number]
	if !ok {
		m.writeError(w, http.StatusNotFound, "Not Found", fmt.Sprintf("Issue #%d not found", number))
		return
	}

	// Read the raw body for debugging
	bodyBytes, _ := io.ReadAll(r.Body)

	var input struct {
		Title     *string  `json:"title,omitempty"`
		Body      *string  `json:"body,omitempty"`
		State     *string  `json:"state,omitempty"`
		Labels    []string `json:"labels,omitempty"`
		Assignee  *string  `json:"assignee,omitempty"`
		Assignees []string `json:"assignees,omitempty"`
	}

	if err := json.Unmarshal(bodyBytes, &input); err != nil {
		m.writeError(w, http.StatusBadRequest, "Invalid JSON", err.Error())
		return
	}

	if input.Title != nil {
		issue.Title = *input.Title
	}
	if input.Body != nil {
		issue.Body = *input.Body
	}
	if input.State != nil {
		issue.State = *input.State
	}
	// Always update labels if provided in request (even if empty)
	if len(input.Labels) > 0 {
		issue.Labels = input.Labels
	}
	if input.Assignee != nil {
		issue.Assignee = *input.Assignee
	}
	// Handle Assignees array (takes precedence over singular Assignee)
	if len(input.Assignees) > 0 {
		issue.Assignee = input.Assignees[0]
	} else if input.Assignees != nil && len(input.Assignees) == 0 {
		// Empty assignees array means unassign
		issue.Assignee = ""
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(m.issueToJSON(issue))
}

// handleComments handles GET/POST /repos/{owner}/{repo}/issues/{number}/comments
func (m *MockGitHubServer) handleComments(w http.ResponseWriter, r *http.Request, issueNumber int) {
	switch r.Method {
	case http.MethodGet:
		m.listComments(w, r, issueNumber)
	case http.MethodPost:
		m.createComment(w, r, issueNumber)
	default:
		m.writeError(w, http.StatusMethodNotAllowed, "Method Not Allowed", "Method Not Allowed")
	}
}

// listComments handles GET /repos/{owner}/{repo}/issues/{number}/comments
func (m *MockGitHubServer) listComments(w http.ResponseWriter, _ *http.Request, issueNumber int) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	comments, ok := m.Comments[issueNumber]
	if !ok {
		comments = []MockGitHubComment{}
	}

	var result []map[string]interface{}
	for _, comment := range comments {
		result = append(result, map[string]interface{}{
			"id": comment.ID,
			"user": map[string]interface{}{
				"login": comment.Author,
			},
			"body":       comment.Body,
			"created_at": time.Now().Format(time.RFC3339),
			"updated_at": time.Now().Format(time.RFC3339),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// createComment handles POST /repos/{owner}/{repo}/issues/{number}/comments
func (m *MockGitHubServer) createComment(w http.ResponseWriter, r *http.Request, issueNumber int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if issue exists
	if _, ok := m.Issues[issueNumber]; !ok {
		m.writeError(w, http.StatusNotFound, "Not Found", fmt.Sprintf("Issue #%d not found", issueNumber))
		return
	}

	var input struct {
		Body string `json:"body"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		m.writeError(w, http.StatusBadRequest, "Invalid JSON", err.Error())
		return
	}

	comment := MockGitHubComment{
		ID:     m.NextCommentID,
		Author: m.AuthenticatedUser,
		Body:   input.Body,
	}
	m.NextCommentID++

	m.Comments[issueNumber] = append(m.Comments[issueNumber], comment)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id": comment.ID,
		"user": map[string]interface{}{
			"login": comment.Author,
		},
		"body":       comment.Body,
		"created_at": time.Now().Format(time.RFC3339),
		"updated_at": time.Now().Format(time.RFC3339),
	})
}

// handleLabels handles GET/POST/PUT /repos/{owner}/{repo}/issues/{number}/labels
func (m *MockGitHubServer) handleLabels(w http.ResponseWriter, r *http.Request, issueNumber int) {
	switch r.Method {
	case http.MethodGet:
		m.listLabels(w, r, issueNumber)
	case http.MethodPost:
		m.addLabels(w, r, issueNumber)
	case http.MethodPut:
		m.setLabels(w, r, issueNumber)
	default:
		m.writeError(w, http.StatusMethodNotAllowed, "Method Not Allowed", "Method Not Allowed")
	}
}

// listLabels handles GET /repos/{owner}/{repo}/issues/{number}/labels
func (m *MockGitHubServer) listLabels(w http.ResponseWriter, _ *http.Request, issueNumber int) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	issue, ok := m.Issues[issueNumber]
	if !ok {
		m.writeError(w, http.StatusNotFound, "Not Found", fmt.Sprintf("Issue #%d not found", issueNumber))
		return
	}

	var labels []map[string]interface{}
	for _, label := range issue.Labels {
		labels = append(labels, map[string]interface{}{
			"name": label,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(labels)
}

// addLabels handles POST /repos/{owner}/{repo}/issues/{number}/labels
func (m *MockGitHubServer) addLabels(w http.ResponseWriter, r *http.Request, issueNumber int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	issue, ok := m.Issues[issueNumber]
	if !ok {
		m.writeError(w, http.StatusNotFound, "Not Found", fmt.Sprintf("Issue #%d not found", issueNumber))
		return
	}

	var input struct {
		Labels []string `json:"labels"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		m.writeError(w, http.StatusBadRequest, "Invalid JSON", err.Error())
		return
	}

	// Add labels, avoiding duplicates
	existing := make(map[string]bool)
	for _, label := range issue.Labels {
		existing[label] = true
	}
	for _, label := range input.Labels {
		if !existing[label] {
			issue.Labels = append(issue.Labels, label)
		}
	}

	var labels []map[string]interface{}
	for _, label := range issue.Labels {
		labels = append(labels, map[string]interface{}{
			"name": label,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(labels)
}

// setLabels handles PUT /repos/{owner}/{repo}/issues/{number}/labels
func (m *MockGitHubServer) setLabels(w http.ResponseWriter, r *http.Request, issueNumber int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	issue, ok := m.Issues[issueNumber]
	if !ok {
		m.writeError(w, http.StatusNotFound, "Not Found", fmt.Sprintf("Issue #%d not found", issueNumber))
		return
	}

	var input struct {
		Labels []string `json:"labels"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		m.writeError(w, http.StatusBadRequest, "Invalid JSON", err.Error())
		return
	}

	issue.Labels = input.Labels

	var labels []map[string]interface{}
	for _, label := range issue.Labels {
		labels = append(labels, map[string]interface{}{
			"name": label,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(labels)
}

// handleLabel handles DELETE /repos/{owner}/{repo}/issues/{number}/labels/{name}
func (m *MockGitHubServer) handleLabel(w http.ResponseWriter, r *http.Request, issueNumber int, labelName string) {
	if r.Method != http.MethodDelete {
		m.writeError(w, http.StatusMethodNotAllowed, "Method Not Allowed", "Method Not Allowed")
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	issue, ok := m.Issues[issueNumber]
	if !ok {
		m.writeError(w, http.StatusNotFound, "Not Found", fmt.Sprintf("Issue #%d not found", issueNumber))
		return
	}

	// Remove the label
	var newLabels []string
	for _, label := range issue.Labels {
		if label != labelName {
			newLabels = append(newLabels, label)
		}
	}
	issue.Labels = newLabels

	// Return remaining labels
	var labels []map[string]interface{}
	for _, label := range issue.Labels {
		labels = append(labels, map[string]interface{}{
			"name": label,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(labels)
}

// issueToJSON converts a MockGitHubIssue to the GitHub API JSON format.
func (m *MockGitHubServer) issueToJSON(issue *MockGitHubIssue) map[string]interface{} {
	result := map[string]interface{}{
		"number":     issue.Number,
		"title":      issue.Title,
		"state":      issue.State,
		"body":       issue.Body,
		"created_at": time.Now().Format(time.RFC3339),
		"updated_at": time.Now().Format(time.RFC3339),
		"html_url":   fmt.Sprintf("%s/test-owner/test-repo/issues/%d", m.URL, issue.Number),
		"url":        fmt.Sprintf("%s/repos/test-owner/test-repo/issues/%d", m.URL, issue.Number),
	}

	// Add labels array
	var labels []map[string]interface{}
	for i, label := range issue.Labels {
		labels = append(labels, map[string]interface{}{
			"id":   int64(i + 1),
			"name": label,
		})
	}
	result["labels"] = labels

	// Add assignee if set (both singular and plural for go-github compatibility)
	if issue.Assignee != "" {
		assigneeObj := map[string]interface{}{
			"login": issue.Assignee,
		}
		result["assignee"] = assigneeObj
		result["assignees"] = []map[string]interface{}{assigneeObj}
	} else {
		result["assignee"] = nil
		result["assignees"] = []map[string]interface{}{}
	}

	return result
}

// writeError writes a GitHub-style error response.
func (m *MockGitHubServer) writeError(w http.ResponseWriter, status int, message, documentation string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":           message,
		"documentation_url": documentation,
	})
}

// handleGraphQL handles POST /graphql requests for GitHub Projects v2.
func (m *MockGitHubServer) handleGraphQL(w http.ResponseWriter, r *http.Request) {
	if !m.validateAuth(w, r) {
		return
	}

	if r.Method != http.MethodPost {
		m.writeGraphQLError(w, "Method not allowed")
		return
	}

	var req struct {
		Query     string                 `json:"query"`
		Variables map[string]interface{} `json:"variables"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		m.writeGraphQLError(w, "Invalid JSON: "+err.Error())
		return
	}

	// Parse and handle different GraphQL queries/mutations
	query := strings.TrimSpace(req.Query)

	// Check for project query patterns
	if strings.Contains(query, "projectV2") || strings.Contains(query, "ProjectV2") {
		m.handleProjectQuery(w, query, req.Variables)
		return
	}

	// Default: return empty data
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"data": map[string]interface{}{},
	})
}

// handleProjectQuery handles GraphQL queries related to projects.
func (m *MockGitHubServer) handleProjectQuery(w http.ResponseWriter, query string, variables map[string]interface{}) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Extract project number from variables or query
	var projectNumber int
	if num, ok := variables["projectNumber"].(float64); ok {
		projectNumber = int(num)
	} else if num, ok := variables["number"].(float64); ok {
		projectNumber = int(num)
	}

	// Check if this is an invalid project ID
	if m.InvalidProjectIDs[projectNumber] {
		m.writeGraphQLError(w, fmt.Sprintf("Could not find project with number %d", projectNumber))
		return
	}

	// Check if project exists
	project, exists := m.Projects[projectNumber]
	if !exists && projectNumber > 0 {
		// If no project is set up but a query is made, return empty but valid response
		// This mimics GitHub behavior when project doesn't exist
		m.writeGraphQLError(w, fmt.Sprintf("Could not find project with number %d", projectNumber))
		return
	}

	// Build response based on query type
	if strings.Contains(query, "field") || strings.Contains(query, "Field") {
		// Query for project fields (columns)
		m.handleProjectFieldsQuery(w, project)
		return
	}

	if strings.Contains(query, "items") {
		// Query for project items
		m.handleProjectItemsQuery(w, projectNumber, project)
		return
	}

	// Default project info query
	m.handleProjectInfoQuery(w, project)
}

// handleProjectInfoQuery returns basic project information.
func (m *MockGitHubServer) handleProjectInfoQuery(w http.ResponseWriter, project *MockGitHubProject) {
	if project == nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": map[string]interface{}{
				"repository": map[string]interface{}{
					"projectV2": nil,
				},
			},
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"data": map[string]interface{}{
			"repository": map[string]interface{}{
				"projectV2": map[string]interface{}{
					"id":     fmt.Sprintf("PVT_%d", project.ID),
					"title":  project.Title,
					"number": project.ID,
				},
			},
		},
	})
}

// handleProjectFieldsQuery returns project fields (columns/status options).
func (m *MockGitHubServer) handleProjectFieldsQuery(w http.ResponseWriter, project *MockGitHubProject) {
	if project == nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": map[string]interface{}{
				"repository": map[string]interface{}{
					"projectV2": nil,
				},
			},
		})
		return
	}

	// Build field options from columns (for the Status field)
	var options []map[string]interface{}
	for _, col := range project.Columns {
		options = append(options, map[string]interface{}{
			"id":   col.ID,
			"name": col.Name,
		})
	}

	// Build the fields array with Status as a single-select field
	// and include built-in fields like Title
	fieldNodes := []map[string]interface{}{
		{
			// Title field (regular ProjectV2Field)
			"__typename": "ProjectV2Field",
			"id":         "PVTF_Title",
			"name":       "Title",
			"dataType":   "TITLE",
		},
		{
			// Status field (single-select)
			"__typename": "ProjectV2SingleSelectField",
			"id":         "PVTSSF_Status",
			"name":       "Status",
			"options":    options,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"data": map[string]interface{}{
			"node": map[string]interface{}{
				"fields": map[string]interface{}{
					"nodes": fieldNodes,
				},
			},
		},
	})
}

// handleProjectItemsQuery returns project items (issues on the board).
func (m *MockGitHubServer) handleProjectItemsQuery(w http.ResponseWriter, projectID int, project *MockGitHubProject) {
	if project == nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": map[string]interface{}{
				"repository": map[string]interface{}{
					"projectV2": nil,
				},
			},
		})
		return
	}

	// Build items list
	var items []map[string]interface{}
	if projectItems, ok := m.ProjectItems[projectID]; ok {
		for _, item := range projectItems {
			issue := m.Issues[item.IssueNumber]
			if issue == nil {
				continue
			}

			// Find column name
			columnName := ""
			for _, col := range project.Columns {
				if col.ID == item.ColumnID {
					columnName = col.Name
					break
				}
			}

			items = append(items, map[string]interface{}{
				"id": fmt.Sprintf("PVTI_%d", item.IssueNumber),
				"content": map[string]interface{}{
					"__typename": "Issue",
					"number":     issue.Number,
					"title":      issue.Title,
					"state":      strings.ToUpper(issue.State),
				},
				"fieldValueByName": map[string]interface{}{
					"__typename": "ProjectV2ItemFieldSingleSelectValue",
					"name":       columnName,
					"optionId":   item.ColumnID,
				},
			})
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"data": map[string]interface{}{
			"repository": map[string]interface{}{
				"projectV2": map[string]interface{}{
					"id":     fmt.Sprintf("PVT_%d", project.ID),
					"title":  project.Title,
					"number": project.ID,
					"items": map[string]interface{}{
						"nodes": items,
					},
				},
			},
		},
	})
}

// writeGraphQLError writes a GraphQL-style error response.
func (m *MockGitHubServer) writeGraphQLError(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"errors": []map[string]interface{}{
			{
				"message": message,
				"type":    "NOT_FOUND",
			},
		},
	})
}
