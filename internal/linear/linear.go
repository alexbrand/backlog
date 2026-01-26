// Package linear implements a Linear backend for the backlog CLI.
// Tasks are stored as Linear Issues with status managed via workflow states.
package linear

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/alexbrand/backlog/internal/backend"
	"github.com/alexbrand/backlog/internal/credentials"
)

const (
	// Version is the current version of the Linear backend.
	Version = "0.1.0"

	// Name is the name of the Linear backend.
	Name = "linear"

	// defaultLinearAPIEndpoint is the default Linear GraphQL API endpoint.
	defaultLinearAPIEndpoint = "https://api.linear.app/graphql"
)

// Priority mapping from Linear's numeric priority (0-4) to canonical priority.
// Linear priority: 0 = No priority, 1 = Urgent, 2 = High, 3 = Medium, 4 = Low
var linearPriorityToCanonical = map[int]backend.Priority{
	0: backend.PriorityNone,
	1: backend.PriorityUrgent,
	2: backend.PriorityHigh,
	3: backend.PriorityMedium,
	4: backend.PriorityLow,
}

var canonicalPriorityToLinear = map[backend.Priority]int{
	backend.PriorityNone:   0,
	backend.PriorityUrgent: 1,
	backend.PriorityHigh:   2,
	backend.PriorityMedium: 3,
	backend.PriorityLow:    4,
}

// Default status mappings from Linear workflow state names to canonical statuses.
// These are common names used in Linear - can be overridden via workspace config.
var defaultStatusMapping = map[string]backend.Status{
	"Backlog":     backend.StatusBacklog,
	"Todo":        backend.StatusTodo,
	"To Do":       backend.StatusTodo,
	"In Progress": backend.StatusInProgress,
	"In Review":   backend.StatusReview,
	"Review":      backend.StatusReview,
	"Done":        backend.StatusDone,
	"Completed":   backend.StatusDone,
	"Canceled":    backend.StatusDone,
	"Cancelled":   backend.StatusDone,
}

// WorkspaceConfig holds Linear backend-specific workspace configuration.
type WorkspaceConfig struct {
	// TeamKey is the team key (e.g., "ENG") for filtering issues.
	TeamKey string
	// StatusMap allows custom status-to-workflow-state mappings.
	StatusMap map[backend.Status]string
}

// Linear implements the Backend interface using Linear Issues.
type Linear struct {
	client           *http.Client
	apiKey           string
	apiEndpoint      string
	teamKey          string
	teamID           string
	agentID          string
	agentLabelPrefix string
	statusMap        map[backend.Status]string
	reverseStatusMap map[string]backend.Status
	connected        bool
	ctx              context.Context
}

// New creates a new Linear backend instance.
func New() *Linear {
	// Check for LINEAR_API_URL environment variable for testing/custom endpoints
	apiEndpoint := os.Getenv("LINEAR_API_URL")
	if apiEndpoint == "" {
		apiEndpoint = defaultLinearAPIEndpoint
	}
	return &Linear{
		ctx:         context.Background(),
		client:      &http.Client{Timeout: 30 * time.Second},
		apiEndpoint: apiEndpoint,
	}
}

// Name returns the name of the backend.
func (l *Linear) Name() string {
	return Name
}

// Version returns the version of the backend.
func (l *Linear) Version() string {
	return Version
}

// Connect initializes the backend with the given configuration.
func (l *Linear) Connect(cfg backend.Config) error {
	wsCfg, ok := cfg.Workspace.(*WorkspaceConfig)
	if !ok {
		return errors.New("invalid workspace configuration for linear backend")
	}

	l.teamKey = wsCfg.TeamKey
	l.agentID = cfg.AgentID
	l.agentLabelPrefix = cfg.AgentLabelPrefix
	if l.agentLabelPrefix == "" {
		l.agentLabelPrefix = "agent"
	}

	// Set up status mappings
	l.statusMap = make(map[backend.Status]string)
	l.reverseStatusMap = make(map[string]backend.Status)

	if wsCfg.StatusMap != nil {
		l.statusMap = wsCfg.StatusMap
		for status, state := range wsCfg.StatusMap {
			l.reverseStatusMap[strings.ToLower(state)] = status
		}
	} else {
		// Use default mappings
		for state, status := range defaultStatusMapping {
			l.reverseStatusMap[strings.ToLower(state)] = status
		}
		// Create reverse mapping for status -> state name
		l.statusMap[backend.StatusBacklog] = "Backlog"
		l.statusMap[backend.StatusTodo] = "Todo"
		l.statusMap[backend.StatusInProgress] = "In Progress"
		l.statusMap[backend.StatusReview] = "In Review"
		l.statusMap[backend.StatusDone] = "Done"
	}

	// Get API key from credentials (env var or credentials.yaml)
	apiKey, err := credentials.GetLinearAPIKey()
	if err != nil {
		return err
	}
	l.apiKey = apiKey

	// Verify connection and get team ID if team key is specified
	if l.teamKey != "" {
		teamID, err := l.getTeamID(l.teamKey)
		if err != nil {
			return fmt.Errorf("failed to get team: %w", err)
		}
		l.teamID = teamID
	}

	l.connected = true
	return nil
}

// Disconnect closes the backend connection.
func (l *Linear) Disconnect() error {
	l.connected = false
	l.apiKey = ""
	return nil
}

// HealthCheck verifies the backend is accessible.
func (l *Linear) HealthCheck() (backend.HealthStatus, error) {
	start := time.Now()

	if !l.connected {
		return backend.HealthStatus{
			OK:      false,
			Message: "not connected",
			Latency: time.Since(start),
		}, nil
	}

	// Try to get the viewer (authenticated user) to verify access
	query := `query { viewer { id name } }`
	_, err := l.graphQL(query, nil)
	latency := time.Since(start)

	if err != nil {
		return backend.HealthStatus{
			OK:      false,
			Message: fmt.Sprintf("failed to access Linear API: %v", err),
			Latency: latency,
		}, nil
	}

	return backend.HealthStatus{
		OK:      true,
		Message: "ok",
		Latency: latency,
	}, nil
}

// List returns tasks matching the given filters.
func (l *Linear) List(filters backend.TaskFilters) (*backend.TaskList, error) {
	if !l.connected {
		return nil, errors.New("not connected")
	}

	// Build GraphQL query with filters
	query := `
		query ListIssues($first: Int, $filter: IssueFilter) {
			issues(first: $first, filter: $filter) {
				nodes {
					id
					identifier
					title
					description
					priority
					sortOrder
					url
					createdAt
					updatedAt
					state {
						id
						name
					}
					assignee {
						id
						name
						displayName
					}
					labels {
						nodes {
							id
							name
						}
					}
					team {
						id
						key
					}
				}
				pageInfo {
					hasNextPage
				}
			}
		}
	`

	// Build filter
	filter := make(map[string]any)

	// Team filter
	if l.teamID != "" {
		filter["team"] = map[string]any{"id": map[string]any{"eq": l.teamID}}
	}

	// Assignee filter
	if filters.Assignee != "" {
		if filters.Assignee == "@me" {
			filter["assignee"] = map[string]any{"isMe": map[string]any{"eq": true}}
		} else if filters.Assignee == "unassigned" {
			filter["assignee"] = map[string]any{"null": true}
		}
		// Note: filtering by specific assignee name would require looking up the user ID first
	}

	// Label filter
	if len(filters.Labels) > 0 {
		labelFilters := make([]map[string]any, len(filters.Labels))
		for i, label := range filters.Labels {
			labelFilters[i] = map[string]any{"name": map[string]any{"eq": label}}
		}
		if len(labelFilters) == 1 {
			filter["labels"] = labelFilters[0]
		} else {
			filter["labels"] = map[string]any{"and": labelFilters}
		}
	}

	// Priority filter
	if len(filters.Priority) > 0 {
		priorities := make([]int, 0, len(filters.Priority))
		for _, p := range filters.Priority {
			if lp, ok := canonicalPriorityToLinear[p]; ok {
				priorities = append(priorities, lp)
			}
		}
		if len(priorities) > 0 {
			filter["priority"] = map[string]any{"in": priorities}
		}
	}

	// Limit
	first := 100
	if filters.Limit > 0 && filters.Limit < 100 {
		first = filters.Limit
	}

	variables := map[string]any{
		"first": first,
	}
	if len(filter) > 0 {
		variables["filter"] = filter
	}

	result, err := l.graphQL(query, variables)
	if err != nil {
		return nil, fmt.Errorf("failed to list issues: %w", err)
	}

	// Parse response
	data, ok := result["data"].(map[string]any)
	if !ok {
		return nil, errors.New("unexpected response format")
	}

	issuesData, ok := data["issues"].(map[string]any)
	if !ok {
		return nil, errors.New("unexpected response format: missing issues")
	}

	nodes, ok := issuesData["nodes"].([]any)
	if !ok {
		return nil, errors.New("unexpected response format: missing nodes")
	}

	pageInfo, _ := issuesData["pageInfo"].(map[string]any)
	hasMore, _ := pageInfo["hasNextPage"].(bool)

	tasks := make([]backend.Task, 0, len(nodes))
	for _, node := range nodes {
		issue, ok := node.(map[string]any)
		if !ok {
			continue
		}

		task := l.issueToTask(issue)

		// Apply status filter (done after mapping since we need to map state names)
		if len(filters.Status) > 0 {
			found := false
			for _, s := range filters.Status {
				if task.Status == s {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		// Exclude done unless explicitly included
		if !filters.IncludeDone && task.Status == backend.StatusDone {
			continue
		}

		tasks = append(tasks, *task)
	}

	// Sort by sortOrder (lower = higher on the board)
	sort.Slice(tasks, func(i, j int) bool {
		si, _ := tasks[i].Meta["sort_order"].(float64)
		sj, _ := tasks[j].Meta["sort_order"].(float64)
		return si < sj
	})

	// Apply limit after filtering
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
func (l *Linear) Get(id string) (*backend.Task, error) {
	if !l.connected {
		return nil, errors.New("not connected")
	}

	// Normalize ID (remove LIN- prefix if present)
	issueID := l.normalizeID(id)

	query := `
		query GetIssue($id: String!) {
			issue(id: $id) {
				id
				identifier
				title
				description
				priority
				url
				createdAt
				updatedAt
				state {
					id
					name
				}
				assignee {
					id
					name
					displayName
				}
				labels {
					nodes {
						id
						name
					}
				}
				team {
					id
					key
				}
			}
		}
	`

	result, err := l.graphQL(query, map[string]any{"id": issueID})
	if err != nil {
		return nil, fmt.Errorf("failed to get issue: %w", err)
	}

	data, ok := result["data"].(map[string]any)
	if !ok {
		return nil, errors.New("unexpected response format")
	}

	issue, ok := data["issue"].(map[string]any)
	if !ok || issue == nil {
		return nil, fmt.Errorf("issue %s not found", id)
	}

	return l.issueToTask(issue), nil
}

// Create creates a new task and returns it.
func (l *Linear) Create(input backend.TaskInput) (*backend.Task, error) {
	if !l.connected {
		return nil, errors.New("not connected")
	}

	if l.teamID == "" {
		return nil, errors.New("team not configured - set 'team' in workspace config")
	}

	mutation := `
		mutation CreateIssue($input: IssueCreateInput!) {
			issueCreate(input: $input) {
				success
				issue {
					id
					identifier
					title
					description
					priority
					url
					createdAt
					updatedAt
					state {
						id
						name
					}
					assignee {
						id
						name
						displayName
					}
					labels {
						nodes {
							id
							name
						}
					}
					team {
						id
						key
					}
				}
			}
		}
	`

	issueInput := map[string]any{
		"title":  input.Title,
		"teamId": l.teamID,
	}

	if input.Description != "" {
		issueInput["description"] = input.Description
	}

	// Set priority
	if input.Priority != "" && input.Priority != backend.PriorityNone {
		if lp, ok := canonicalPriorityToLinear[input.Priority]; ok {
			issueInput["priority"] = lp
		}
	}

	// Set initial status/state
	if input.Status != "" {
		stateID, err := l.getStateIDForStatus(input.Status)
		if err == nil {
			issueInput["stateId"] = stateID
		}
	}

	// Set labels
	if len(input.Labels) > 0 {
		labelIDs, err := l.getLabelIDs(input.Labels)
		if err == nil && len(labelIDs) > 0 {
			issueInput["labelIds"] = labelIDs
		}
	}

	result, err := l.graphQL(mutation, map[string]any{"input": issueInput})
	if err != nil {
		return nil, fmt.Errorf("failed to create issue: %w", err)
	}

	data, ok := result["data"].(map[string]any)
	if !ok {
		return nil, errors.New("unexpected response format")
	}

	createResult, ok := data["issueCreate"].(map[string]any)
	if !ok {
		return nil, errors.New("unexpected response format: missing issueCreate")
	}

	success, _ := createResult["success"].(bool)
	if !success {
		return nil, errors.New("failed to create issue")
	}

	issue, ok := createResult["issue"].(map[string]any)
	if !ok {
		return nil, errors.New("unexpected response format: missing issue")
	}

	return l.issueToTask(issue), nil
}

// Update modifies an existing task and returns the updated task.
func (l *Linear) Update(id string, changes backend.TaskChanges) (*backend.Task, error) {
	if !l.connected {
		return nil, errors.New("not connected")
	}

	issueID := l.normalizeID(id)

	// First get the current issue to get its Linear UUID
	issue, err := l.getIssueByIdentifier(issueID)
	if err != nil {
		return nil, err
	}

	linearID, ok := issue["id"].(string)
	if !ok {
		return nil, errors.New("failed to get issue ID")
	}

	mutation := `
		mutation UpdateIssue($id: String!, $input: IssueUpdateInput!) {
			issueUpdate(id: $id, input: $input) {
				success
				issue {
					id
					identifier
					title
					description
					priority
					url
					createdAt
					updatedAt
					state {
						id
						name
					}
					assignee {
						id
						name
						displayName
					}
					labels {
						nodes {
							id
							name
						}
					}
					team {
						id
						key
					}
				}
			}
		}
	`

	issueInput := make(map[string]any)

	if changes.Title != nil {
		issueInput["title"] = *changes.Title
	}

	if changes.Description != nil {
		issueInput["description"] = *changes.Description
	}

	if changes.Priority != nil {
		if lp, ok := canonicalPriorityToLinear[*changes.Priority]; ok {
			issueInput["priority"] = lp
		}
	}

	// Handle label changes
	if len(changes.AddLabels) > 0 || len(changes.RemoveLabels) > 0 {
		// Get current label IDs
		currentLabels := make(map[string]string) // name -> id
		if labelsData, ok := issue["labels"].(map[string]any); ok {
			if nodes, ok := labelsData["nodes"].([]any); ok {
				for _, n := range nodes {
					if label, ok := n.(map[string]any); ok {
						name, _ := label["name"].(string)
						id, _ := label["id"].(string)
						currentLabels[name] = id
					}
				}
			}
		}

		// Add new labels
		addIDs, _ := l.getLabelIDs(changes.AddLabels)
		for _, id := range addIDs {
			found := false
			for _, existingID := range currentLabels {
				if existingID == id {
					found = true
					break
				}
			}
			if !found {
				currentLabels["_add_"+id] = id
			}
		}

		// Remove labels
		for _, name := range changes.RemoveLabels {
			delete(currentLabels, name)
		}

		// Build final label ID list
		labelIDs := make([]string, 0, len(currentLabels))
		for _, id := range currentLabels {
			labelIDs = append(labelIDs, id)
		}
		issueInput["labelIds"] = labelIDs
	}

	if len(issueInput) == 0 {
		// Nothing to update
		return l.issueToTask(issue), nil
	}

	result, err := l.graphQL(mutation, map[string]any{
		"id":    linearID,
		"input": issueInput,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update issue: %w", err)
	}

	data, ok := result["data"].(map[string]any)
	if !ok {
		return nil, errors.New("unexpected response format")
	}

	updateResult, ok := data["issueUpdate"].(map[string]any)
	if !ok {
		return nil, errors.New("unexpected response format: missing issueUpdate")
	}

	success, _ := updateResult["success"].(bool)
	if !success {
		return nil, errors.New("failed to update issue")
	}

	updatedIssue, ok := updateResult["issue"].(map[string]any)
	if !ok {
		return nil, errors.New("unexpected response format: missing issue")
	}

	return l.issueToTask(updatedIssue), nil
}

// Delete removes a task by ID (archives the issue in Linear).
func (l *Linear) Delete(id string) error {
	if !l.connected {
		return errors.New("not connected")
	}

	issueID := l.normalizeID(id)

	// Get Linear UUID
	issue, err := l.getIssueByIdentifier(issueID)
	if err != nil {
		return err
	}

	linearID, ok := issue["id"].(string)
	if !ok {
		return errors.New("failed to get issue ID")
	}

	// Archive the issue (Linear doesn't support true deletion)
	mutation := `
		mutation ArchiveIssue($id: String!) {
			issueArchive(id: $id) {
				success
			}
		}
	`

	result, err := l.graphQL(mutation, map[string]any{"id": linearID})
	if err != nil {
		return fmt.Errorf("failed to archive issue: %w", err)
	}

	data, ok := result["data"].(map[string]any)
	if !ok {
		return errors.New("unexpected response format")
	}

	archiveResult, ok := data["issueArchive"].(map[string]any)
	if !ok {
		return errors.New("unexpected response format: missing issueArchive")
	}

	success, _ := archiveResult["success"].(bool)
	if !success {
		return errors.New("failed to archive issue")
	}

	return nil
}

// Move transitions a task to a new status.
func (l *Linear) Move(id string, status backend.Status) (*backend.Task, error) {
	if !l.connected {
		return nil, errors.New("not connected")
	}

	if !status.IsValid() {
		return nil, fmt.Errorf("invalid status: %s", status)
	}

	issueID := l.normalizeID(id)

	// Get Linear UUID
	issue, err := l.getIssueByIdentifier(issueID)
	if err != nil {
		return nil, err
	}

	linearID, ok := issue["id"].(string)
	if !ok {
		return nil, errors.New("failed to get issue ID")
	}

	// Get the state ID for the target status
	stateID, err := l.getStateIDForStatus(status)
	if err != nil {
		return nil, fmt.Errorf("failed to find workflow state for status %s: %w", status, err)
	}

	mutation := `
		mutation UpdateIssueState($id: String!, $input: IssueUpdateInput!) {
			issueUpdate(id: $id, input: $input) {
				success
				issue {
					id
					identifier
					title
					description
					priority
					url
					createdAt
					updatedAt
					state {
						id
						name
					}
					assignee {
						id
						name
						displayName
					}
					labels {
						nodes {
							id
							name
						}
					}
					team {
						id
						key
					}
				}
			}
		}
	`

	result, err := l.graphQL(mutation, map[string]any{
		"id":    linearID,
		"input": map[string]any{"stateId": stateID},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update issue state: %w", err)
	}

	data, ok := result["data"].(map[string]any)
	if !ok {
		return nil, errors.New("unexpected response format")
	}

	updateResult, ok := data["issueUpdate"].(map[string]any)
	if !ok {
		return nil, errors.New("unexpected response format: missing issueUpdate")
	}

	success, _ := updateResult["success"].(bool)
	if !success {
		return nil, errors.New("failed to update issue state")
	}

	updatedIssue, ok := updateResult["issue"].(map[string]any)
	if !ok {
		return nil, errors.New("unexpected response format: missing issue")
	}

	return l.issueToTask(updatedIssue), nil
}

// Assign assigns a task to a user.
func (l *Linear) Assign(id string, assignee string) (*backend.Task, error) {
	if !l.connected {
		return nil, errors.New("not connected")
	}

	issueID := l.normalizeID(id)

	// Get Linear UUID
	issue, err := l.getIssueByIdentifier(issueID)
	if err != nil {
		return nil, err
	}

	linearID, ok := issue["id"].(string)
	if !ok {
		return nil, errors.New("failed to get issue ID")
	}

	// Get user ID for assignee
	userID, err := l.getUserID(assignee)
	if err != nil {
		return nil, fmt.Errorf("failed to find user %s: %w", assignee, err)
	}

	mutation := `
		mutation AssignIssue($id: String!, $input: IssueUpdateInput!) {
			issueUpdate(id: $id, input: $input) {
				success
				issue {
					id
					identifier
					title
					description
					priority
					url
					createdAt
					updatedAt
					state {
						id
						name
					}
					assignee {
						id
						name
						displayName
					}
					labels {
						nodes {
							id
							name
						}
					}
					team {
						id
						key
					}
				}
			}
		}
	`

	result, err := l.graphQL(mutation, map[string]any{
		"id":    linearID,
		"input": map[string]any{"assigneeId": userID},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to assign issue: %w", err)
	}

	data, ok := result["data"].(map[string]any)
	if !ok {
		return nil, errors.New("unexpected response format")
	}

	updateResult, ok := data["issueUpdate"].(map[string]any)
	if !ok {
		return nil, errors.New("unexpected response format: missing issueUpdate")
	}

	success, _ := updateResult["success"].(bool)
	if !success {
		return nil, errors.New("failed to assign issue")
	}

	updatedIssue, ok := updateResult["issue"].(map[string]any)
	if !ok {
		return nil, errors.New("unexpected response format: missing issue")
	}

	return l.issueToTask(updatedIssue), nil
}

// Unassign removes the assignee from a task.
func (l *Linear) Unassign(id string) (*backend.Task, error) {
	if !l.connected {
		return nil, errors.New("not connected")
	}

	issueID := l.normalizeID(id)

	// Get Linear UUID
	issue, err := l.getIssueByIdentifier(issueID)
	if err != nil {
		return nil, err
	}

	linearID, ok := issue["id"].(string)
	if !ok {
		return nil, errors.New("failed to get issue ID")
	}

	mutation := `
		mutation UnassignIssue($id: String!, $input: IssueUpdateInput!) {
			issueUpdate(id: $id, input: $input) {
				success
				issue {
					id
					identifier
					title
					description
					priority
					url
					createdAt
					updatedAt
					state {
						id
						name
					}
					assignee {
						id
						name
						displayName
					}
					labels {
						nodes {
							id
							name
						}
					}
					team {
						id
						key
					}
				}
			}
		}
	`

	// Setting assigneeId to null unassigns the issue
	result, err := l.graphQL(mutation, map[string]any{
		"id":    linearID,
		"input": map[string]any{"assigneeId": nil},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to unassign issue: %w", err)
	}

	data, ok := result["data"].(map[string]any)
	if !ok {
		return nil, errors.New("unexpected response format")
	}

	updateResult, ok := data["issueUpdate"].(map[string]any)
	if !ok {
		return nil, errors.New("unexpected response format: missing issueUpdate")
	}

	success, _ := updateResult["success"].(bool)
	if !success {
		return nil, errors.New("failed to unassign issue")
	}

	updatedIssue, ok := updateResult["issue"].(map[string]any)
	if !ok {
		return nil, errors.New("unexpected response format: missing issue")
	}

	return l.issueToTask(updatedIssue), nil
}

// ListComments returns all comments for a task.
func (l *Linear) ListComments(id string) ([]backend.Comment, error) {
	if !l.connected {
		return nil, errors.New("not connected")
	}

	issueID := l.normalizeID(id)

	query := `
		query GetIssueComments($id: String!) {
			issue(id: $id) {
				comments {
					nodes {
						id
						body
						createdAt
						user {
							id
							name
							displayName
						}
					}
				}
			}
		}
	`

	result, err := l.graphQL(query, map[string]any{"id": issueID})
	if err != nil {
		return nil, fmt.Errorf("failed to get comments: %w", err)
	}

	data, ok := result["data"].(map[string]any)
	if !ok {
		return nil, errors.New("unexpected response format")
	}

	issue, ok := data["issue"].(map[string]any)
	if !ok || issue == nil {
		return nil, fmt.Errorf("issue %s not found", id)
	}

	commentsData, ok := issue["comments"].(map[string]any)
	if !ok {
		return []backend.Comment{}, nil
	}

	nodes, ok := commentsData["nodes"].([]any)
	if !ok {
		return []backend.Comment{}, nil
	}

	comments := make([]backend.Comment, 0, len(nodes))
	for _, node := range nodes {
		c, ok := node.(map[string]any)
		if !ok {
			continue
		}

		comment := backend.Comment{
			ID:   getString(c, "id"),
			Body: getString(c, "body"),
		}

		if createdAt := getString(c, "createdAt"); createdAt != "" {
			if t, err := time.Parse(time.RFC3339, createdAt); err == nil {
				comment.Created = t
			}
		}

		if user, ok := c["user"].(map[string]any); ok {
			comment.Author = getString(user, "displayName")
			if comment.Author == "" {
				comment.Author = getString(user, "name")
			}
		}

		comments = append(comments, comment)
	}

	return comments, nil
}

// AddComment adds a comment to a task.
func (l *Linear) AddComment(id string, body string) (*backend.Comment, error) {
	if !l.connected {
		return nil, errors.New("not connected")
	}

	issueID := l.normalizeID(id)

	// Get Linear UUID
	issue, err := l.getIssueByIdentifier(issueID)
	if err != nil {
		return nil, err
	}

	linearID, ok := issue["id"].(string)
	if !ok {
		return nil, errors.New("failed to get issue ID")
	}

	mutation := `
		mutation CreateComment($input: CommentCreateInput!) {
			commentCreate(input: $input) {
				success
				comment {
					id
					body
					createdAt
					user {
						id
						name
						displayName
					}
				}
			}
		}
	`

	result, err := l.graphQL(mutation, map[string]any{
		"input": map[string]any{
			"issueId": linearID,
			"body":    body,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create comment: %w", err)
	}

	data, ok := result["data"].(map[string]any)
	if !ok {
		return nil, errors.New("unexpected response format")
	}

	createResult, ok := data["commentCreate"].(map[string]any)
	if !ok {
		return nil, errors.New("unexpected response format: missing commentCreate")
	}

	success, _ := createResult["success"].(bool)
	if !success {
		return nil, errors.New("failed to create comment")
	}

	c, ok := createResult["comment"].(map[string]any)
	if !ok {
		return nil, errors.New("unexpected response format: missing comment")
	}

	comment := &backend.Comment{
		ID:   id, // Use task ID as the comment ID for consistency
		Body: getString(c, "body"),
	}

	if createdAt := getString(c, "createdAt"); createdAt != "" {
		if t, err := time.Parse(time.RFC3339, createdAt); err == nil {
			comment.Created = t
		}
	}

	if user, ok := c["user"].(map[string]any); ok {
		comment.Author = getString(user, "displayName")
		if comment.Author == "" {
			comment.Author = getString(user, "name")
		}
	}

	return comment, nil
}

// Claim claims a task for an agent.
// Implements the backend.Claimer interface.
func (l *Linear) Claim(id string, agentID string) (*backend.ClaimResult, error) {
	if !l.connected {
		return nil, errors.New("not connected")
	}

	if agentID == "" {
		agentID = l.agentID
	}

	issueID := l.normalizeID(id)

	// Get the issue
	issue, err := l.getIssueByIdentifier(issueID)
	if err != nil {
		return nil, err
	}

	linearID, ok := issue["id"].(string)
	if !ok {
		return nil, errors.New("failed to get issue ID")
	}

	// Check for existing agent labels
	agentLabelPrefix := l.agentLabelPrefix + ":"
	if labelsData, ok := issue["labels"].(map[string]any); ok {
		if nodes, ok := labelsData["nodes"].([]any); ok {
			for _, n := range nodes {
				if label, ok := n.(map[string]any); ok {
					name := getString(label, "name")
					if strings.HasPrefix(name, agentLabelPrefix) {
						claimedBy := strings.TrimPrefix(name, agentLabelPrefix)
						if claimedBy == agentID {
							// Already claimed by this agent
							return &backend.ClaimResult{
								Task:         l.issueToTask(issue),
								AlreadyOwned: true,
							}, nil
						}
						// Claimed by another agent
						return nil, &ClaimConflictError{
							TaskID:       id,
							ClaimedBy:    claimedBy,
							CurrentAgent: agentID,
						}
					}
				}
			}
		}
	}

	// Get or create the agent label
	agentLabelName := agentLabelPrefix + agentID
	agentLabelID, err := l.getOrCreateLabel(agentLabelName)
	if err != nil {
		return nil, fmt.Errorf("failed to get/create agent label: %w", err)
	}

	// Get current label IDs
	labelIDs := []string{agentLabelID}
	if labelsData, ok := issue["labels"].(map[string]any); ok {
		if nodes, ok := labelsData["nodes"].([]any); ok {
			for _, n := range nodes {
				if label, ok := n.(map[string]any); ok {
					if id := getString(label, "id"); id != "" {
						labelIDs = append(labelIDs, id)
					}
				}
			}
		}
	}

	// Get the state ID for in-progress
	stateID, err := l.getStateIDForStatus(backend.StatusInProgress)
	if err != nil {
		return nil, fmt.Errorf("failed to find in-progress state: %w", err)
	}

	// Get the current user ID for assignment
	viewerID, err := l.getViewerID()
	if err != nil {
		return nil, fmt.Errorf("failed to get current user: %w", err)
	}

	// Update the issue: add agent label, set to in-progress, assign to current user
	mutation := `
		mutation ClaimIssue($id: String!, $input: IssueUpdateInput!) {
			issueUpdate(id: $id, input: $input) {
				success
				issue {
					id
					identifier
					title
					description
					priority
					url
					createdAt
					updatedAt
					state {
						id
						name
					}
					assignee {
						id
						name
						displayName
					}
					labels {
						nodes {
							id
							name
						}
					}
					team {
						id
						key
					}
				}
			}
		}
	`

	result, err := l.graphQL(mutation, map[string]any{
		"id": linearID,
		"input": map[string]any{
			"labelIds":   labelIDs,
			"stateId":    stateID,
			"assigneeId": viewerID,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to claim issue: %w", err)
	}

	data, ok := result["data"].(map[string]any)
	if !ok {
		return nil, errors.New("unexpected response format")
	}

	updateResult, ok := data["issueUpdate"].(map[string]any)
	if !ok {
		return nil, errors.New("unexpected response format: missing issueUpdate")
	}

	success, _ := updateResult["success"].(bool)
	if !success {
		return nil, errors.New("failed to claim issue")
	}

	updatedIssue, ok := updateResult["issue"].(map[string]any)
	if !ok {
		return nil, errors.New("unexpected response format: missing issue")
	}

	return &backend.ClaimResult{
		Task:         l.issueToTask(updatedIssue),
		AlreadyOwned: false,
	}, nil
}

// Release releases a claimed task back to todo status.
// Implements the backend.Claimer interface.
func (l *Linear) Release(id string) error {
	if !l.connected {
		return errors.New("not connected")
	}

	issueID := l.normalizeID(id)

	// Get the issue
	issue, err := l.getIssueByIdentifier(issueID)
	if err != nil {
		return err
	}

	linearID, ok := issue["id"].(string)
	if !ok {
		return errors.New("failed to get issue ID")
	}

	// Check who currently claims this task
	agentLabelPrefix := l.agentLabelPrefix + ":"
	claimedBy := ""
	labelIDs := []string{}
	if labelsData, ok := issue["labels"].(map[string]any); ok {
		if nodes, ok := labelsData["nodes"].([]any); ok {
			for _, n := range nodes {
				if label, ok := n.(map[string]any); ok {
					name := getString(label, "name")
					if strings.HasPrefix(name, agentLabelPrefix) {
						claimedBy = strings.TrimPrefix(name, agentLabelPrefix)
					} else {
						if id := getString(label, "id"); id != "" {
							labelIDs = append(labelIDs, id)
						}
					}
				}
			}
		}
	}

	// Check if the task is claimed
	if claimedBy == "" {
		return &ReleaseConflictError{
			TaskID:     issueID,
			NotClaimed: true,
		}
	}

	// Check if the task is claimed by the current agent
	if claimedBy != l.agentID {
		return &ReleaseConflictError{
			TaskID:       issueID,
			ClaimedBy:    claimedBy,
			CurrentAgent: l.agentID,
		}
	}

	// Get the state ID for todo
	stateID, err := l.getStateIDForStatus(backend.StatusTodo)
	if err != nil {
		return fmt.Errorf("failed to find todo state: %w", err)
	}

	// Update the issue: remove agent labels, set to todo, unassign
	mutation := `
		mutation ReleaseIssue($id: String!, $input: IssueUpdateInput!) {
			issueUpdate(id: $id, input: $input) {
				success
			}
		}
	`

	result, err := l.graphQL(mutation, map[string]any{
		"id": linearID,
		"input": map[string]any{
			"labelIds":   labelIDs,
			"stateId":    stateID,
			"assigneeId": nil,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to release issue: %w", err)
	}

	data, ok := result["data"].(map[string]any)
	if !ok {
		return errors.New("unexpected response format")
	}

	updateResult, ok := data["issueUpdate"].(map[string]any)
	if !ok {
		return errors.New("unexpected response format: missing issueUpdate")
	}

	success, _ := updateResult["success"].(bool)
	if !success {
		return errors.New("failed to release issue")
	}

	return nil
}

// Helper functions

// graphQL executes a GraphQL query/mutation against the Linear API.
func (l *Linear) graphQL(query string, variables map[string]any) (map[string]any, error) {
	body := map[string]any{
		"query": query,
	}
	if variables != nil {
		body["variables"] = variables
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(l.ctx, "POST", l.apiEndpoint, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", l.apiKey)

	resp, err := l.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error: %s - %s", resp.Status, string(respBody))
	}

	var result map[string]any
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Check for GraphQL errors
	if errors, ok := result["errors"].([]any); ok && len(errors) > 0 {
		if errObj, ok := errors[0].(map[string]any); ok {
			if msg, ok := errObj["message"].(string); ok {
				return nil, fmt.Errorf("GraphQL error: %s", msg)
			}
		}
		return nil, fmt.Errorf("GraphQL error: %v", errors)
	}

	return result, nil
}

// getTeamID fetches the team ID for a given team key.
func (l *Linear) getTeamID(key string) (string, error) {
	query := `
		query GetTeam($key: String!) {
			team(id: $key) {
				id
				name
				key
			}
		}
	`

	result, err := l.graphQL(query, map[string]any{"key": key})
	if err != nil {
		return "", err
	}

	data, ok := result["data"].(map[string]any)
	if !ok {
		return "", errors.New("unexpected response format")
	}

	team, ok := data["team"].(map[string]any)
	if !ok || team == nil {
		return "", fmt.Errorf("team %s not found", key)
	}

	teamID, ok := team["id"].(string)
	if !ok {
		return "", errors.New("failed to get team ID")
	}

	return teamID, nil
}

// getStateIDForStatus finds the workflow state ID that matches the given canonical status.
func (l *Linear) getStateIDForStatus(status backend.Status) (string, error) {
	// Get the expected state name for this status
	stateName := l.statusMap[status]
	if stateName == "" {
		return "", fmt.Errorf("no mapping for status %s", status)
	}

	query := `
		query GetWorkflowStates($teamId: ID) {
			workflowStates(filter: { team: { id: { eq: $teamId } } }) {
				nodes {
					id
					name
					type
				}
			}
		}
	`

	variables := map[string]any{}
	if l.teamID != "" {
		variables["teamId"] = l.teamID
	}

	result, err := l.graphQL(query, variables)
	if err != nil {
		return "", err
	}

	data, ok := result["data"].(map[string]any)
	if !ok {
		return "", errors.New("unexpected response format")
	}

	states, ok := data["workflowStates"].(map[string]any)
	if !ok {
		return "", errors.New("unexpected response format: missing workflowStates")
	}

	nodes, ok := states["nodes"].([]any)
	if !ok {
		return "", errors.New("unexpected response format: missing nodes")
	}

	// Look for exact match first
	for _, node := range nodes {
		state, ok := node.(map[string]any)
		if !ok {
			continue
		}
		name := getString(state, "name")
		if strings.EqualFold(name, stateName) {
			return getString(state, "id"), nil
		}
	}

	// Fall back to type-based matching
	stateType := ""
	switch status {
	case backend.StatusBacklog:
		stateType = "backlog"
	case backend.StatusTodo:
		stateType = "unstarted"
	case backend.StatusInProgress:
		stateType = "started"
	case backend.StatusReview:
		stateType = "started" // Review is also a started state
	case backend.StatusDone:
		stateType = "completed"
	}

	for _, node := range nodes {
		state, ok := node.(map[string]any)
		if !ok {
			continue
		}
		t := getString(state, "type")
		if strings.EqualFold(t, stateType) {
			fallbackName := getString(state, "name")
			// Check if the fallback state maps back to a different canonical status
			if mappedStatus, ok := l.reverseStatusMap[strings.ToLower(fallbackName)]; ok && mappedStatus != status {
				return "", fmt.Errorf("no %q workflow state found in Linear (expected %q); the closest match %q maps to %q instead. Add an %q state in your Linear team's workflow settings", status, stateName, fallbackName, mappedStatus, stateName)
			}
			return getString(state, "id"), nil
		}
	}

	return "", fmt.Errorf("no workflow state found for status %s", status)
}

// getIssueByIdentifier fetches an issue by its identifier (e.g., "ENG-123").
func (l *Linear) getIssueByIdentifier(identifier string) (map[string]any, error) {
	query := `
		query GetIssue($id: String!) {
			issue(id: $id) {
				id
				identifier
				title
				description
				priority
				url
				createdAt
				updatedAt
				state {
					id
					name
				}
				assignee {
					id
					name
					displayName
				}
				labels {
					nodes {
						id
						name
					}
				}
				team {
					id
					key
				}
			}
		}
	`

	result, err := l.graphQL(query, map[string]any{"id": identifier})
	if err != nil {
		return nil, err
	}

	data, ok := result["data"].(map[string]any)
	if !ok {
		return nil, errors.New("unexpected response format")
	}

	issue, ok := data["issue"].(map[string]any)
	if !ok || issue == nil {
		return nil, fmt.Errorf("issue %s not found", identifier)
	}

	return issue, nil
}

// getUserID fetches the user ID for a given username/email.
func (l *Linear) getUserID(name string) (string, error) {
	query := `
		query GetUsers {
			users {
				nodes {
					id
					name
					displayName
					email
				}
			}
		}
	`

	result, err := l.graphQL(query, nil)
	if err != nil {
		return "", err
	}

	data, ok := result["data"].(map[string]any)
	if !ok {
		return "", errors.New("unexpected response format")
	}

	users, ok := data["users"].(map[string]any)
	if !ok {
		return "", errors.New("unexpected response format: missing users")
	}

	nodes, ok := users["nodes"].([]any)
	if !ok {
		return "", errors.New("unexpected response format: missing nodes")
	}

	nameLower := strings.ToLower(name)
	for _, node := range nodes {
		user, ok := node.(map[string]any)
		if !ok {
			continue
		}
		if strings.EqualFold(getString(user, "name"), name) ||
			strings.EqualFold(getString(user, "displayName"), name) ||
			strings.EqualFold(getString(user, "email"), name) ||
			strings.Contains(strings.ToLower(getString(user, "name")), nameLower) ||
			strings.Contains(strings.ToLower(getString(user, "displayName")), nameLower) {
			return getString(user, "id"), nil
		}
	}

	return "", fmt.Errorf("user %s not found", name)
}

// getViewerID fetches the current authenticated user's ID.
func (l *Linear) getViewerID() (string, error) {
	query := `query { viewer { id } }`

	result, err := l.graphQL(query, nil)
	if err != nil {
		return "", err
	}

	data, ok := result["data"].(map[string]any)
	if !ok {
		return "", errors.New("unexpected response format")
	}

	viewer, ok := data["viewer"].(map[string]any)
	if !ok {
		return "", errors.New("unexpected response format: missing viewer")
	}

	return getString(viewer, "id"), nil
}

// getLabelIDs fetches label IDs for the given label names.
func (l *Linear) getLabelIDs(names []string) ([]string, error) {
	if len(names) == 0 {
		return nil, nil
	}

	query := `
		query GetLabels($teamId: ID) {
			issueLabels(filter: { team: { id: { eq: $teamId } } }) {
				nodes {
					id
					name
				}
			}
		}
	`

	variables := map[string]any{}
	if l.teamID != "" {
		variables["teamId"] = l.teamID
	}

	result, err := l.graphQL(query, variables)
	if err != nil {
		return nil, err
	}

	data, ok := result["data"].(map[string]any)
	if !ok {
		return nil, errors.New("unexpected response format")
	}

	labels, ok := data["issueLabels"].(map[string]any)
	if !ok {
		return nil, errors.New("unexpected response format: missing issueLabels")
	}

	nodes, ok := labels["nodes"].([]any)
	if !ok {
		return nil, errors.New("unexpected response format: missing nodes")
	}

	// Build name -> ID map
	labelMap := make(map[string]string)
	for _, node := range nodes {
		label, ok := node.(map[string]any)
		if !ok {
			continue
		}
		name := getString(label, "name")
		id := getString(label, "id")
		labelMap[strings.ToLower(name)] = id
	}

	ids := make([]string, 0, len(names))
	for _, name := range names {
		if id, ok := labelMap[strings.ToLower(name)]; ok {
			ids = append(ids, id)
		}
	}

	return ids, nil
}

// getOrCreateLabel gets an existing label or creates it if it doesn't exist.
func (l *Linear) getOrCreateLabel(name string) (string, error) {
	// First try to find existing label
	ids, err := l.getLabelIDs([]string{name})
	if err == nil && len(ids) > 0 {
		return ids[0], nil
	}

	// Create the label
	mutation := `
		mutation CreateLabel($input: IssueLabelCreateInput!) {
			issueLabelCreate(input: $input) {
				success
				issueLabel {
					id
					name
				}
			}
		}
	`

	input := map[string]any{
		"name": name,
	}
	if l.teamID != "" {
		input["teamId"] = l.teamID
	}

	result, err := l.graphQL(mutation, map[string]any{"input": input})
	if err != nil {
		return "", err
	}

	data, ok := result["data"].(map[string]any)
	if !ok {
		return "", errors.New("unexpected response format")
	}

	createResult, ok := data["issueLabelCreate"].(map[string]any)
	if !ok {
		return "", errors.New("unexpected response format: missing issueLabelCreate")
	}

	label, ok := createResult["issueLabel"].(map[string]any)
	if !ok {
		return "", errors.New("unexpected response format: missing issueLabel")
	}

	return getString(label, "id"), nil
}

// normalizeID removes the LIN- prefix from an ID if present.
func (l *Linear) normalizeID(id string) string {
	id = strings.TrimPrefix(id, "LIN-")
	return id
}

// issueToTask converts a Linear Issue to a backend Task.
func (l *Linear) issueToTask(issue map[string]any) *backend.Task {
	task := &backend.Task{
		ID:          getString(issue, "identifier"),
		Title:       getString(issue, "title"),
		Description: getString(issue, "description"),
		URL:         getString(issue, "url"),
		Meta:        make(map[string]any),
	}

	// Parse timestamps
	if createdAt := getString(issue, "createdAt"); createdAt != "" {
		if t, err := time.Parse(time.RFC3339, createdAt); err == nil {
			task.Created = t
		}
	}
	if updatedAt := getString(issue, "updatedAt"); updatedAt != "" {
		if t, err := time.Parse(time.RFC3339, updatedAt); err == nil {
			task.Updated = t
		}
	}

	// Priority (Linear uses 0-4)
	if priority, ok := issue["priority"].(float64); ok {
		task.Priority = linearPriorityToCanonical[int(priority)]
	} else {
		task.Priority = backend.PriorityNone
	}

	// State/Status
	if state, ok := issue["state"].(map[string]any); ok {
		stateName := getString(state, "name")
		if status, ok := l.reverseStatusMap[strings.ToLower(stateName)]; ok {
			task.Status = status
		} else {
			task.Status = backend.StatusBacklog // Default for unknown states
		}
		task.Meta["state_id"] = getString(state, "id")
		task.Meta["state_name"] = stateName
	}

	// Assignee
	if assignee, ok := issue["assignee"].(map[string]any); ok {
		task.Assignee = getString(assignee, "displayName")
		if task.Assignee == "" {
			task.Assignee = getString(assignee, "name")
		}
		task.Meta["assignee_id"] = getString(assignee, "id")
	}

	// Labels (including agent labels)
	if labelsData, ok := issue["labels"].(map[string]any); ok {
		if nodes, ok := labelsData["nodes"].([]any); ok {
			labels := make([]string, 0, len(nodes))
			for _, n := range nodes {
				if label, ok := n.(map[string]any); ok {
					name := getString(label, "name")
					labels = append(labels, name)
				}
			}
			task.Labels = labels
		}
	}

	// Sort order (used for board position ordering)
	if sortOrder, ok := issue["sortOrder"].(float64); ok {
		task.Meta["sort_order"] = sortOrder
	}

	// Store Linear ID in meta
	task.Meta["linear_id"] = getString(issue, "id")
	task.Meta["identifier"] = getString(issue, "identifier")

	// Store team info
	if team, ok := issue["team"].(map[string]any); ok {
		task.Meta["team_id"] = getString(team, "id")
		task.Meta["team_key"] = getString(team, "key")
	}

	return task
}

// getString safely gets a string value from a map.
func getString(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
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

// ReleaseConflictError represents an error when trying to release a task that isn't claimed
// by the current agent.
type ReleaseConflictError struct {
	TaskID       string
	ClaimedBy    string
	CurrentAgent string
	NotClaimed   bool
}

func (e *ReleaseConflictError) Error() string {
	if e.NotClaimed {
		return fmt.Sprintf("task %s is not claimed", e.TaskID)
	}
	return fmt.Sprintf("task %s is claimed by different agent %s, not by %s", e.TaskID, e.ClaimedBy, e.CurrentAgent)
}

// Register registers the Linear backend with the registry.
func Register() {
	backend.Register(Name, func() backend.Backend {
		return New()
	})
}
