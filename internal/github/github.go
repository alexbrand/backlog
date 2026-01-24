// Package github implements a GitHub Issues backend for the backlog CLI.
// Tasks are stored as GitHub Issues with status managed via labels.
package github

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/alexbrand/backlog/internal/backend"
	"github.com/alexbrand/backlog/internal/credentials"
	gh "github.com/google/go-github/v60/github"
	"golang.org/x/oauth2"
)

const (
	// Version is the current version of the GitHub backend.
	Version = "0.1.0"

	// Name is the name of the GitHub backend.
	Name = "github"
)

// Default status label mappings for GitHub Issues.
// These can be overridden via workspace configuration.
var defaultStatusLabels = map[backend.Status][]string{
	backend.StatusBacklog:    {},               // open issue, no status label
	backend.StatusTodo:       {"ready"},        // open issue with "ready" label
	backend.StatusInProgress: {"in-progress"},  // open issue with "in-progress" label
	backend.StatusReview:     {"needs-review"}, // open issue with "needs-review" label
	backend.StatusDone:       {},               // closed issue
}

// WorkspaceConfig holds GitHub backend-specific workspace configuration.
type WorkspaceConfig struct {
	// Repo is the repository in "owner/repo" format.
	Repo string
	// Project is the optional GitHub Project number for Projects v2 integration.
	Project int
	// StatusField is the project field name for status (used with Projects v2).
	StatusField string
	// StatusMap allows custom status-to-label mappings.
	StatusMap map[backend.Status]StatusMapping
}

// StatusMapping defines how a canonical status maps to GitHub state and labels.
type StatusMapping struct {
	// State is the GitHub issue state: "open" or "closed".
	State string
	// Labels are the labels that indicate this status.
	Labels []string
}

// GitHub implements the Backend interface using GitHub Issues.
type GitHub struct {
	client           *gh.Client
	owner            string
	repo             string
	agentID          string
	agentLabelPrefix string
	statusMap        map[backend.Status]StatusMapping
	connected        bool
	ctx              context.Context
	// Projects v2 support
	projectsClient *ProjectsClient
	statusField    *ProjectField
	useProjects    bool
}

// New creates a new GitHub backend instance.
func New() *GitHub {
	return &GitHub{
		ctx: context.Background(),
	}
}

// Name returns the name of the backend.
func (g *GitHub) Name() string {
	return Name
}

// Version returns the version of the backend.
func (g *GitHub) Version() string {
	return Version
}

// Connect initializes the backend with the given configuration.
func (g *GitHub) Connect(cfg backend.Config) error {
	wsCfg, ok := cfg.Workspace.(*WorkspaceConfig)
	if !ok {
		return errors.New("invalid workspace configuration for github backend")
	}

	// Parse owner/repo
	parts := strings.SplitN(wsCfg.Repo, "/", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid repo format: expected 'owner/repo', got %q", wsCfg.Repo)
	}
	g.owner = parts[0]
	g.repo = parts[1]

	g.agentID = cfg.AgentID
	g.agentLabelPrefix = cfg.AgentLabelPrefix
	if g.agentLabelPrefix == "" {
		g.agentLabelPrefix = "agent"
	}

	// Set up status mappings
	g.statusMap = make(map[backend.Status]StatusMapping)
	if wsCfg.StatusMap != nil {
		g.statusMap = wsCfg.StatusMap
	} else {
		// Use default mappings
		for status, labels := range defaultStatusLabels {
			state := "open"
			if status == backend.StatusDone {
				state = "closed"
			}
			g.statusMap[status] = StatusMapping{
				State:  state,
				Labels: labels,
			}
		}
	}

	// Get token from credentials (env var or credentials.yaml)
	token, err := credentials.GetGitHubToken()
	if err != nil {
		return err
	}

	// Create authenticated client
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(g.ctx, ts)
	g.client = gh.NewClient(tc)

	// Check for GITHUB_API_URL environment variable for testing/enterprise
	if apiURL := os.Getenv("GITHUB_API_URL"); apiURL != "" {
		baseURL := apiURL
		if !strings.HasSuffix(baseURL, "/") {
			baseURL += "/"
		}
		var err error
		g.client, err = g.client.WithEnterpriseURLs(baseURL, baseURL)
		if err != nil {
			return fmt.Errorf("failed to set GitHub API URL: %w", err)
		}
	}

	// Initialize Projects v2 client if project is configured
	if wsCfg.Project > 0 {
		statusFieldName := wsCfg.StatusField
		if statusFieldName == "" {
			statusFieldName = "Status"
		}
		apiURL := os.Getenv("GITHUB_API_URL")
		pc, err := NewProjectsClient(g.ctx, token, g.owner, g.repo, wsCfg.Project, statusFieldName, apiURL)
		if err != nil {
			return fmt.Errorf("failed to create projects client: %w", err)
		}
		g.projectsClient = pc

		// Get and cache the status field configuration
		sf, err := pc.GetStatusField()
		if err != nil {
			return fmt.Errorf("failed to get project status field: %w", err)
		}
		g.statusField = sf
		g.useProjects = true
	}

	g.connected = true
	return nil
}

// Disconnect closes the backend connection.
func (g *GitHub) Disconnect() error {
	g.connected = false
	g.client = nil
	return nil
}

// HealthCheck verifies the backend is accessible.
func (g *GitHub) HealthCheck() (backend.HealthStatus, error) {
	start := time.Now()

	if !g.connected {
		return backend.HealthStatus{
			OK:      false,
			Message: "not connected",
			Latency: time.Since(start),
		}, nil
	}

	// Try to get the repository to verify access
	_, resp, err := g.client.Repositories.Get(g.ctx, g.owner, g.repo)
	latency := time.Since(start)

	if err != nil {
		msg := fmt.Sprintf("failed to access repository: %v", err)
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			msg = "repository not found or not accessible"
		} else if resp != nil && resp.StatusCode == http.StatusUnauthorized {
			msg = "authentication failed - check GITHUB_TOKEN"
		}
		return backend.HealthStatus{
			OK:      false,
			Message: msg,
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
func (g *GitHub) List(filters backend.TaskFilters) (*backend.TaskList, error) {
	if !g.connected {
		return nil, errors.New("not connected")
	}

	// Build list options
	opts := &gh.IssueListByRepoOptions{
		State: "all", // We'll filter by status ourselves
		ListOptions: gh.ListOptions{
			PerPage: 100,
		},
	}

	// Apply assignee filter
	if filters.Assignee != "" {
		if filters.Assignee == "@me" {
			opts.Assignee = "" // Will be set after getting current user
		} else if filters.Assignee == "unassigned" {
			opts.Assignee = "none"
		} else {
			opts.Assignee = filters.Assignee
		}
	}

	// Apply label filters
	if len(filters.Labels) > 0 {
		opts.Labels = filters.Labels
	}

	// Fetch issues
	issues, _, err := g.client.Issues.ListByRepo(g.ctx, g.owner, g.repo, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to list issues: %w", err)
	}

	// Initialize as empty slice (not nil) so JSON encoding produces [] not null
	tasks := []backend.Task{}
	for _, issue := range issues {
		// Skip pull requests (GitHub API returns them as issues too)
		if issue.IsPullRequest() {
			continue
		}

		task := g.issueToTask(issue)

		// Apply status filter
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

		// Apply priority filter
		if len(filters.Priority) > 0 {
			found := false
			for _, p := range filters.Priority {
				if task.Priority == p {
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
func (g *GitHub) Get(id string) (*backend.Task, error) {
	if !g.connected {
		return nil, errors.New("not connected")
	}

	issueNum, err := g.parseIssueNumber(id)
	if err != nil {
		return nil, err
	}

	issue, _, err := g.client.Issues.Get(g.ctx, g.owner, g.repo, issueNum)
	if err != nil {
		return nil, fmt.Errorf("failed to get issue: %w", err)
	}

	return g.issueToTask(issue), nil
}

// Create creates a new task and returns it.
func (g *GitHub) Create(input backend.TaskInput) (*backend.Task, error) {
	if !g.connected {
		return nil, errors.New("not connected")
	}

	// Build issue request
	issueReq := &gh.IssueRequest{
		Title: gh.String(input.Title),
	}

	if input.Description != "" {
		issueReq.Body = gh.String(input.Description)
	}

	// Build labels
	var labels []string
	labels = append(labels, input.Labels...)

	// Add status labels only if not using project-based status
	status := input.Status
	if status == "" {
		status = backend.StatusBacklog
	}
	if !g.useProjects {
		if mapping, ok := g.statusMap[status]; ok {
			labels = append(labels, mapping.Labels...)
		}
	}

	// Add priority label if set
	if input.Priority != "" && input.Priority != backend.PriorityNone {
		labels = append(labels, "priority:"+string(input.Priority))
	}

	if len(labels) > 0 {
		issueReq.Labels = &labels
	}

	if input.Assignee != "" {
		issueReq.Assignees = &[]string{input.Assignee}
	}

	// Create the issue
	issue, _, err := g.client.Issues.Create(g.ctx, g.owner, g.repo, issueReq)
	if err != nil {
		return nil, fmt.Errorf("failed to create issue: %w", err)
	}

	// Add issue to project and set status if using Projects v2
	if g.useProjects {
		issueNodeID, err := g.projectsClient.GetIssueNodeID(issue.GetNumber())
		if err != nil {
			// Log warning but don't fail - issue was created
			return g.issueToTask(issue), nil
		}

		itemID, err := g.projectsClient.AddIssueToProject(issueNodeID)
		if err != nil {
			// Log warning but don't fail - issue was created
			return g.issueToTask(issue), nil
		}

		// Set the status on the project item
		optionID, err := g.projectsClient.MapStatusToOptionID(status, g.statusField)
		if err == nil {
			g.projectsClient.UpdateProjectItemStatus(itemID, optionID)
		}
	}

	return g.issueToTask(issue), nil
}

// Update modifies an existing task and returns the updated task.
func (g *GitHub) Update(id string, changes backend.TaskChanges) (*backend.Task, error) {
	if !g.connected {
		return nil, errors.New("not connected")
	}

	issueNum, err := g.parseIssueNumber(id)
	if err != nil {
		return nil, err
	}

	// Get current issue to get current labels
	issue, _, err := g.client.Issues.Get(g.ctx, g.owner, g.repo, issueNum)
	if err != nil {
		return nil, fmt.Errorf("failed to get issue: %w", err)
	}

	issueReq := &gh.IssueRequest{}

	if changes.Title != nil {
		issueReq.Title = changes.Title
	}
	if changes.Description != nil {
		issueReq.Body = changes.Description
	}
	if changes.Assignee != nil {
		if *changes.Assignee == "" {
			issueReq.Assignees = &[]string{}
		} else {
			issueReq.Assignees = &[]string{*changes.Assignee}
		}
	}

	// Handle label changes
	if len(changes.AddLabels) > 0 || len(changes.RemoveLabels) > 0 {
		currentLabels := make(map[string]bool)
		for _, label := range issue.Labels {
			currentLabels[label.GetName()] = true
		}

		// Add new labels
		for _, l := range changes.AddLabels {
			currentLabels[l] = true
		}

		// Remove labels
		for _, l := range changes.RemoveLabels {
			delete(currentLabels, l)
		}

		labels := make([]string, 0, len(currentLabels))
		for l := range currentLabels {
			labels = append(labels, l)
		}
		issueReq.Labels = &labels
	}

	// Handle priority change
	if changes.Priority != nil {
		currentLabels := make(map[string]bool)
		for _, label := range issue.Labels {
			// Remove existing priority labels
			if !strings.HasPrefix(label.GetName(), "priority:") {
				currentLabels[label.GetName()] = true
			}
		}
		// Add new priority label
		if *changes.Priority != backend.PriorityNone {
			currentLabels["priority:"+string(*changes.Priority)] = true
		}

		labels := make([]string, 0, len(currentLabels))
		for l := range currentLabels {
			labels = append(labels, l)
		}
		issueReq.Labels = &labels
	}

	updatedIssue, _, err := g.client.Issues.Edit(g.ctx, g.owner, g.repo, issueNum, issueReq)
	if err != nil {
		return nil, fmt.Errorf("failed to update issue: %w", err)
	}

	return g.issueToTask(updatedIssue), nil
}

// Delete removes a task by ID (closes the issue).
func (g *GitHub) Delete(id string) error {
	if !g.connected {
		return errors.New("not connected")
	}

	issueNum, err := g.parseIssueNumber(id)
	if err != nil {
		return err
	}

	// Close the issue (GitHub doesn't support true deletion via API)
	_, _, err = g.client.Issues.Edit(g.ctx, g.owner, g.repo, issueNum, &gh.IssueRequest{
		State: gh.String("closed"),
	})
	if err != nil {
		return fmt.Errorf("failed to close issue: %w", err)
	}

	return nil
}

// Move transitions a task to a new status.
func (g *GitHub) Move(id string, status backend.Status) (*backend.Task, error) {
	if !g.connected {
		return nil, errors.New("not connected")
	}

	if !status.IsValid() {
		return nil, fmt.Errorf("invalid status: %s", status)
	}

	issueNum, err := g.parseIssueNumber(id)
	if err != nil {
		return nil, err
	}

	// Get current issue
	issue, _, err := g.client.Issues.Get(g.ctx, g.owner, g.repo, issueNum)
	if err != nil {
		return nil, fmt.Errorf("failed to get issue: %w", err)
	}

	// Update project status if using Projects v2
	if g.useProjects {
		if err := g.updateProjectStatus(issueNum, status); err != nil {
			return nil, fmt.Errorf("failed to update project status: %w", err)
		}
	}

	// Build new labels: remove status labels, add new status labels
	// Only update labels if not using project-based status
	newLabels := make([]string, 0)
	for _, label := range issue.Labels {
		labelName := label.GetName()
		isStatusLabel := false
		for _, mapping := range g.statusMap {
			for _, l := range mapping.Labels {
				if labelName == l {
					isStatusLabel = true
					break
				}
			}
			if isStatusLabel {
				break
			}
		}
		if !isStatusLabel {
			newLabels = append(newLabels, labelName)
		}
	}

	// Add new status labels only if not using project-based status
	if !g.useProjects {
		if mapping, ok := g.statusMap[status]; ok {
			newLabels = append(newLabels, mapping.Labels...)
		}
	}

	// Determine state
	state := "open"
	if status == backend.StatusDone {
		state = "closed"
	}

	// Update the issue
	updatedIssue, _, err := g.client.Issues.Edit(g.ctx, g.owner, g.repo, issueNum, &gh.IssueRequest{
		State:  gh.String(state),
		Labels: &newLabels,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update issue: %w", err)
	}

	return g.issueToTask(updatedIssue), nil
}

// updateProjectStatus updates the status of an issue in GitHub Projects v2.
func (g *GitHub) updateProjectStatus(issueNum int, status backend.Status) error {
	// Get the project item for this issue
	item, err := g.projectsClient.GetProjectItemByIssue(issueNum)
	if err != nil {
		// Issue might not be in the project - try to add it
		issueNodeID, err := g.projectsClient.GetIssueNodeID(issueNum)
		if err != nil {
			return fmt.Errorf("failed to get issue node ID: %w", err)
		}
		itemID, err := g.projectsClient.AddIssueToProject(issueNodeID)
		if err != nil {
			return fmt.Errorf("failed to add issue to project: %w", err)
		}
		item = &ProjectItem{ID: itemID, IssueNumber: issueNum}
	}

	// Map status to project option ID
	optionID, err := g.projectsClient.MapStatusToOptionID(status, g.statusField)
	if err != nil {
		return err
	}

	// Update the project item status
	if err := g.projectsClient.UpdateProjectItemStatus(item.ID, optionID); err != nil {
		return err
	}

	return nil
}

// Assign assigns a task to a user.
func (g *GitHub) Assign(id string, assignee string) (*backend.Task, error) {
	return g.Update(id, backend.TaskChanges{Assignee: &assignee})
}

// Unassign removes the assignee from a task.
func (g *GitHub) Unassign(id string) (*backend.Task, error) {
	empty := ""
	return g.Update(id, backend.TaskChanges{Assignee: &empty})
}

// ListComments returns all comments for a task.
func (g *GitHub) ListComments(id string) ([]backend.Comment, error) {
	if !g.connected {
		return nil, errors.New("not connected")
	}

	issueNum, err := g.parseIssueNumber(id)
	if err != nil {
		return nil, err
	}

	ghComments, _, err := g.client.Issues.ListComments(g.ctx, g.owner, g.repo, issueNum, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list comments: %w", err)
	}

	comments := make([]backend.Comment, len(ghComments))
	for i, c := range ghComments {
		comments[i] = backend.Comment{
			ID:      fmt.Sprintf("%d", c.GetID()),
			Author:  c.GetUser().GetLogin(),
			Body:    c.GetBody(),
			Created: c.GetCreatedAt().Time,
		}
	}

	return comments, nil
}

// AddComment adds a comment to a task.
func (g *GitHub) AddComment(id string, body string) (*backend.Comment, error) {
	if !g.connected {
		return nil, errors.New("not connected")
	}

	issueNum, err := g.parseIssueNumber(id)
	if err != nil {
		return nil, err
	}

	comment, _, err := g.client.Issues.CreateComment(g.ctx, g.owner, g.repo, issueNum, &gh.IssueComment{
		Body: gh.String(body),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create comment: %w", err)
	}

	return &backend.Comment{
		ID:      id, // Use task ID as the comment ID for consistency
		Author:  comment.GetUser().GetLogin(),
		Body:    comment.GetBody(),
		Created: comment.GetCreatedAt().Time,
	}, nil
}

// Claim claims a task for an agent.
// Implements the backend.Claimer interface.
func (g *GitHub) Claim(id string, agentID string) (*backend.ClaimResult, error) {
	if !g.connected {
		return nil, errors.New("not connected")
	}

	if agentID == "" {
		agentID = g.agentID
	}

	issueNum, err := g.parseIssueNumber(id)
	if err != nil {
		return nil, err
	}

	// Get current issue
	issue, _, err := g.client.Issues.Get(g.ctx, g.owner, g.repo, issueNum)
	if err != nil {
		return nil, fmt.Errorf("failed to get issue: %w", err)
	}

	// Check for existing agent labels
	agentLabelPrefix := g.agentLabelPrefix + ":"
	for _, label := range issue.Labels {
		if strings.HasPrefix(label.GetName(), agentLabelPrefix) {
			claimedBy := strings.TrimPrefix(label.GetName(), agentLabelPrefix)
			if claimedBy == agentID {
				// Already claimed by this agent
				return &backend.ClaimResult{
					Task:         g.issueToTask(issue),
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

	// Update project status if using Projects v2
	if g.useProjects {
		if err := g.updateProjectStatus(issueNum, backend.StatusInProgress); err != nil {
			return nil, fmt.Errorf("failed to update project status: %w", err)
		}
	}

	// Build new labels: existing + agent label + in-progress status (if not using projects)
	newLabels := make([]string, 0)
	for _, label := range issue.Labels {
		labelName := label.GetName()
		// Remove existing status labels
		isStatusLabel := false
		for _, mapping := range g.statusMap {
			for _, l := range mapping.Labels {
				if labelName == l {
					isStatusLabel = true
					break
				}
			}
			if isStatusLabel {
				break
			}
		}
		if !isStatusLabel {
			newLabels = append(newLabels, labelName)
		}
	}

	// Add agent label
	newLabels = append(newLabels, agentLabelPrefix+agentID)
	// Add in-progress status labels only if not using project-based status
	if !g.useProjects {
		if mapping, ok := g.statusMap[backend.StatusInProgress]; ok {
			newLabels = append(newLabels, mapping.Labels...)
		}
	}

	// Update the issue with assignment
	updatedIssue, _, err := g.client.Issues.Edit(g.ctx, g.owner, g.repo, issueNum, &gh.IssueRequest{
		Labels:    &newLabels,
		Assignees: &[]string{g.getAssigneeForClaim()},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to claim issue: %w", err)
	}

	return &backend.ClaimResult{
		Task:         g.issueToTask(updatedIssue),
		AlreadyOwned: false,
	}, nil
}

// Release releases a claimed task back to todo status.
// Implements the backend.Claimer interface.
func (g *GitHub) Release(id string) error {
	if !g.connected {
		return errors.New("not connected")
	}

	issueNum, err := g.parseIssueNumber(id)
	if err != nil {
		return err
	}

	// Get current issue
	issue, _, err := g.client.Issues.Get(g.ctx, g.owner, g.repo, issueNum)
	if err != nil {
		return fmt.Errorf("failed to get issue: %w", err)
	}

	// Check if the issue is claimed and by whom
	agentLabelPrefix := g.agentLabelPrefix + ":"
	var claimedBy string
	for _, label := range issue.Labels {
		if strings.HasPrefix(label.GetName(), agentLabelPrefix) {
			claimedBy = strings.TrimPrefix(label.GetName(), agentLabelPrefix)
			break
		}
	}

	// If not claimed, error
	if claimedBy == "" {
		return &ReleaseError{TaskID: id, Message: "task is not claimed"}
	}

	// If claimed by a different agent, error
	currentAgent := g.agentID
	if claimedBy != currentAgent {
		return &ReleaseError{
			TaskID:       id,
			Message:      fmt.Sprintf("task %s is claimed by different agent %s, not by %s", id, claimedBy, currentAgent),
			ClaimedBy:    claimedBy,
			CurrentAgent: currentAgent,
		}
	}

	// Update project status if using Projects v2
	if g.useProjects {
		if err := g.updateProjectStatus(issueNum, backend.StatusTodo); err != nil {
			return fmt.Errorf("failed to update project status: %w", err)
		}
	}

	// Build new labels: remove agent labels and status labels, add todo status (if not using projects)
	newLabels := make([]string, 0)
	for _, label := range issue.Labels {
		labelName := label.GetName()
		// Skip agent labels
		if strings.HasPrefix(labelName, agentLabelPrefix) {
			continue
		}
		// Skip status labels
		isStatusLabel := false
		for _, mapping := range g.statusMap {
			for _, l := range mapping.Labels {
				if labelName == l {
					isStatusLabel = true
					break
				}
			}
			if isStatusLabel {
				break
			}
		}
		if !isStatusLabel {
			newLabels = append(newLabels, labelName)
		}
	}

	// Add todo status labels only if not using project-based status
	if !g.useProjects {
		if mapping, ok := g.statusMap[backend.StatusTodo]; ok {
			newLabels = append(newLabels, mapping.Labels...)
		}
	}

	// Update the issue: remove assignees, update labels
	_, _, err = g.client.Issues.Edit(g.ctx, g.owner, g.repo, issueNum, &gh.IssueRequest{
		Labels:    &newLabels,
		Assignees: &[]string{}, // Remove all assignees
	})
	if err != nil {
		return fmt.Errorf("failed to release issue: %w", err)
	}

	return nil
}

// Helper functions

// parseIssueNumber extracts the issue number from an ID string.
// Supports formats: "123", "GH-123", "#123".
func (g *GitHub) parseIssueNumber(id string) (int, error) {
	// Remove common prefixes
	id = strings.TrimPrefix(id, "GH-")
	id = strings.TrimPrefix(id, "#")

	var num int
	_, err := fmt.Sscanf(id, "%d", &num)
	if err != nil {
		return 0, fmt.Errorf("invalid issue ID: %s", id)
	}
	return num, nil
}

// issueToTask converts a GitHub Issue to a backend Task.
func (g *GitHub) issueToTask(issue *gh.Issue) *backend.Task {
	task := &backend.Task{
		ID:      fmt.Sprintf("GH-%d", issue.GetNumber()),
		Title:   issue.GetTitle(),
		Created: issue.GetCreatedAt().Time,
		Updated: issue.GetUpdatedAt().Time,
		URL:     issue.GetHTMLURL(),
		Meta:    make(map[string]any),
	}

	// Description from body
	task.Description = issue.GetBody()

	// Assignee
	if len(issue.Assignees) > 0 {
		task.Assignee = issue.Assignees[0].GetLogin()
	}

	// Labels - include all labels (agent labels, status labels, custom labels)
	// Priority is extracted separately
	var labels []string
	var priority backend.Priority = backend.PriorityNone
	for _, label := range issue.Labels {
		name := label.GetName()
		// Extract priority
		if strings.HasPrefix(name, "priority:") {
			priority = backend.Priority(strings.TrimPrefix(name, "priority:"))
			continue
		}
		// Include all labels (status labels, agent labels, custom labels)
		labels = append(labels, name)
	}
	task.Labels = labels
	task.Priority = priority

	// Determine status from state and labels
	task.Status = g.determineStatus(issue)

	// Store original issue number in meta
	task.Meta["issue_number"] = issue.GetNumber()

	return task
}

// determineStatus determines the canonical status from a GitHub issue.
func (g *GitHub) determineStatus(issue *gh.Issue) backend.Status {
	if issue.GetState() == "closed" {
		return backend.StatusDone
	}

	// If using Projects v2, get status from project field
	if g.useProjects {
		item, err := g.projectsClient.GetProjectItemByIssue(issue.GetNumber())
		if err == nil && item.FieldValueStr != "" {
			return g.projectsClient.MapOptionToStatus(item.FieldValueStr)
		}
		// Fall through to label-based status if not in project or no status set
	}

	// Check labels for status
	issueLabels := make(map[string]bool)
	for _, label := range issue.Labels {
		issueLabels[label.GetName()] = true
	}

	// Check each status mapping (in order of priority)
	statusOrder := []backend.Status{
		backend.StatusReview,
		backend.StatusInProgress,
		backend.StatusTodo,
		backend.StatusBacklog,
	}

	for _, status := range statusOrder {
		mapping := g.statusMap[status]
		if len(mapping.Labels) == 0 {
			continue
		}
		// Check if issue has all required labels for this status
		hasAll := true
		for _, l := range mapping.Labels {
			if !issueLabels[l] {
				hasAll = false
				break
			}
		}
		if hasAll {
			return status
		}
	}

	// Default to backlog for open issues with no status labels
	return backend.StatusBacklog
}

// getAssigneeForClaim returns the username to assign when claiming.
// This should be the authenticated user.
func (g *GitHub) getAssigneeForClaim() string {
	// Try to get the authenticated user
	user, _, err := g.client.Users.Get(g.ctx, "")
	if err != nil {
		return g.agentID // Fall back to agent ID
	}
	return user.GetLogin()
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

// ReleaseError is returned when a release operation fails due to ownership issues.
type ReleaseError struct {
	TaskID       string
	Message      string
	ClaimedBy    string
	CurrentAgent string
}

func (e *ReleaseError) Error() string {
	return e.Message
}

// Register registers the GitHub backend with the registry.
func Register() {
	backend.Register(Name, func() backend.Backend {
		return New()
	})
}
