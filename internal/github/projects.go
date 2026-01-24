// Package github implements a GitHub Issues backend for the backlog CLI.
// This file contains GraphQL API support for GitHub Projects v2.
package github

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/alexbrand/backlog/internal/backend"
	"github.com/shurcooL/githubv4"
	"golang.org/x/oauth2"
)

// ProjectsClient handles GitHub Projects v2 operations via GraphQL API.
type ProjectsClient struct {
	client      *githubv4.Client
	ctx         context.Context
	owner       string
	repo        string
	projectNum  int
	statusField string
}

// ProjectFieldValue represents a field value option in a project.
type ProjectFieldValue struct {
	ID   string
	Name string
}

// ProjectField represents a field in a GitHub Project.
type ProjectField struct {
	ID      string
	Name    string
	Options []ProjectFieldValue // For single-select fields
}

// ProjectItem represents an item (issue/PR) in a GitHub Project.
type ProjectItem struct {
	ID            string
	FieldValueID  string // Current value of status field
	FieldValueStr string // Current value as string
	IssueNumber   int
	IssueTitle    string
	IssueState    string
	IssueURL      string
}

// NewProjectsClient creates a new GraphQL client for GitHub Projects v2.
// If apiURL is provided, it will be used as the GraphQL endpoint (for testing/enterprise).
func NewProjectsClient(ctx context.Context, token, owner, repo string, projectNum int, statusField, apiURL string) (*ProjectsClient, error) {
	if token == "" {
		return nil, errors.New("github token is required")
	}
	if projectNum <= 0 {
		return nil, errors.New("project number must be positive")
	}

	src := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	httpClient := oauth2.NewClient(ctx, src)

	var client *githubv4.Client
	if apiURL != "" {
		// Use custom GraphQL endpoint (for testing or GitHub Enterprise)
		graphqlURL := apiURL
		if !strings.HasSuffix(graphqlURL, "/") {
			graphqlURL += "/"
		}
		graphqlURL += "graphql"
		client = githubv4.NewEnterpriseClient(graphqlURL, httpClient)
	} else {
		client = githubv4.NewClient(httpClient)
	}

	return &ProjectsClient{
		client:      client,
		ctx:         ctx,
		owner:       owner,
		repo:        repo,
		projectNum:  projectNum,
		statusField: statusField,
	}, nil
}

// GetProjectID returns the GraphQL node ID for the project.
func (p *ProjectsClient) GetProjectID() (string, error) {
	var query struct {
		Repository struct {
			ProjectV2 struct {
				ID githubv4.ID
			} `graphql:"projectV2(number: $projectNumber)"`
		} `graphql:"repository(owner: $owner, name: $repo)"`
	}

	variables := map[string]any{
		"owner":         githubv4.String(p.owner),
		"repo":          githubv4.String(p.repo),
		"projectNumber": githubv4.Int(p.projectNum),
	}

	if err := p.client.Query(p.ctx, &query, variables); err != nil {
		return "", fmt.Errorf("failed to get project ID: %w", err)
	}

	return string(query.Repository.ProjectV2.ID.(string)), nil
}

// GetStatusField returns the status field configuration for the project.
func (p *ProjectsClient) GetStatusField() (*ProjectField, error) {
	projectID, err := p.GetProjectID()
	if err != nil {
		return nil, err
	}

	// Query for project fields
	var query struct {
		Node struct {
			ProjectV2 struct {
				Fields struct {
					Nodes []struct {
						ProjectV2Field struct {
							ID   githubv4.ID
							Name githubv4.String
						} `graphql:"... on ProjectV2Field"`
						ProjectV2SingleSelectField struct {
							ID      githubv4.ID
							Name    githubv4.String
							Options []struct {
								ID   githubv4.ID
								Name githubv4.String
							}
						} `graphql:"... on ProjectV2SingleSelectField"`
					}
				} `graphql:"fields(first: 50)"`
			} `graphql:"... on ProjectV2"`
		} `graphql:"node(id: $projectId)"`
	}

	variables := map[string]any{
		"projectId": githubv4.ID(projectID),
	}

	if err := p.client.Query(p.ctx, &query, variables); err != nil {
		return nil, fmt.Errorf("failed to get project fields: %w", err)
	}

	// Find the status field
	targetFieldName := p.statusField
	if targetFieldName == "" {
		targetFieldName = "Status" // Default field name
	}

	for _, field := range query.Node.ProjectV2.Fields.Nodes {
		// Check single-select fields (which is what Status typically is)
		if string(field.ProjectV2SingleSelectField.Name) == targetFieldName {
			pf := &ProjectField{
				ID:      string(field.ProjectV2SingleSelectField.ID.(string)),
				Name:    string(field.ProjectV2SingleSelectField.Name),
				Options: make([]ProjectFieldValue, len(field.ProjectV2SingleSelectField.Options)),
			}
			for i, opt := range field.ProjectV2SingleSelectField.Options {
				pf.Options[i] = ProjectFieldValue{
					ID:   string(opt.ID.(string)),
					Name: string(opt.Name),
				}
			}
			return pf, nil
		}
	}

	return nil, fmt.Errorf("status field %q not found in project", targetFieldName)
}

// GetProjectItemByIssue returns the project item for a given issue number.
func (p *ProjectsClient) GetProjectItemByIssue(issueNumber int) (*ProjectItem, error) {
	projectID, err := p.GetProjectID()
	if err != nil {
		return nil, err
	}

	statusField, err := p.GetStatusField()
	if err != nil {
		return nil, err
	}

	// Query project items to find the one matching this issue
	var query struct {
		Node struct {
			ProjectV2 struct {
				Items struct {
					Nodes []struct {
						ID      githubv4.ID
						Content struct {
							Issue struct {
								Number   githubv4.Int
								Title    githubv4.String
								State    githubv4.String
								URL      githubv4.String
								Typename string `graphql:"__typename"`
							} `graphql:"... on Issue"`
						}
						FieldValues struct {
							Nodes []struct {
								ProjectV2ItemFieldSingleSelectValue struct {
									Field struct {
										ProjectV2SingleSelectField struct {
											ID githubv4.ID
										} `graphql:"... on ProjectV2SingleSelectField"`
									}
									OptionID githubv4.ID
									Name     githubv4.String
								} `graphql:"... on ProjectV2ItemFieldSingleSelectValue"`
							}
						} `graphql:"fieldValues(first: 20)"`
					}
					PageInfo struct {
						HasNextPage githubv4.Boolean
						EndCursor   githubv4.String
					}
				} `graphql:"items(first: 100, after: $cursor)"`
			} `graphql:"... on ProjectV2"`
		} `graphql:"node(id: $projectId)"`
	}

	var cursor *githubv4.String

	for {
		variables := map[string]any{
			"projectId": githubv4.ID(projectID),
			"cursor":    cursor,
		}

		if err := p.client.Query(p.ctx, &query, variables); err != nil {
			return nil, fmt.Errorf("failed to get project items: %w", err)
		}

		for _, item := range query.Node.ProjectV2.Items.Nodes {
			if int(item.Content.Issue.Number) == issueNumber {
				pi := &ProjectItem{
					ID:          string(item.ID.(string)),
					IssueNumber: issueNumber,
					IssueTitle:  string(item.Content.Issue.Title),
					IssueState:  string(item.Content.Issue.State),
					IssueURL:    string(item.Content.Issue.URL),
				}

				// Find the status field value
				for _, fv := range item.FieldValues.Nodes {
					fieldID := fv.ProjectV2ItemFieldSingleSelectValue.Field.ProjectV2SingleSelectField.ID
					if fieldID != nil && string(fieldID.(string)) == statusField.ID {
						if fv.ProjectV2ItemFieldSingleSelectValue.OptionID != nil {
							pi.FieldValueID = string(fv.ProjectV2ItemFieldSingleSelectValue.OptionID.(string))
						}
						pi.FieldValueStr = string(fv.ProjectV2ItemFieldSingleSelectValue.Name)
						break
					}
				}

				return pi, nil
			}
		}

		if !bool(query.Node.ProjectV2.Items.PageInfo.HasNextPage) {
			break
		}
		cursor = &query.Node.ProjectV2.Items.PageInfo.EndCursor
	}

	return nil, fmt.Errorf("issue #%d not found in project", issueNumber)
}

// UpdateProjectItemStatus updates the status field of a project item.
func (p *ProjectsClient) UpdateProjectItemStatus(itemID string, optionID string) error {
	projectID, err := p.GetProjectID()
	if err != nil {
		return err
	}

	statusField, err := p.GetStatusField()
	if err != nil {
		return err
	}

	var mutation struct {
		UpdateProjectV2ItemFieldValue struct {
			ProjectV2Item struct {
				ID githubv4.ID
			}
		} `graphql:"updateProjectV2ItemFieldValue(input: $input)"`
	}

	input := githubv4.UpdateProjectV2ItemFieldValueInput{
		ProjectID: githubv4.ID(projectID),
		ItemID:    githubv4.ID(itemID),
		FieldID:   githubv4.ID(statusField.ID),
		Value: githubv4.ProjectV2FieldValue{
			SingleSelectOptionID: (*githubv4.String)(&optionID),
		},
	}

	if err := p.client.Mutate(p.ctx, &mutation, input, nil); err != nil {
		return fmt.Errorf("failed to update project item status: %w", err)
	}

	return nil
}

// AddIssueToProject adds an issue to the project and returns the project item ID.
func (p *ProjectsClient) AddIssueToProject(issueNodeID string) (string, error) {
	projectID, err := p.GetProjectID()
	if err != nil {
		return "", err
	}

	var mutation struct {
		AddProjectV2ItemById struct {
			Item struct {
				ID githubv4.ID
			}
		} `graphql:"addProjectV2ItemById(input: $input)"`
	}

	input := githubv4.AddProjectV2ItemByIdInput{
		ProjectID: githubv4.ID(projectID),
		ContentID: githubv4.ID(issueNodeID),
	}

	if err := p.client.Mutate(p.ctx, &mutation, input, nil); err != nil {
		return "", fmt.Errorf("failed to add issue to project: %w", err)
	}

	return string(mutation.AddProjectV2ItemById.Item.ID.(string)), nil
}

// GetIssueNodeID returns the GraphQL node ID for an issue by its number.
func (p *ProjectsClient) GetIssueNodeID(issueNumber int) (string, error) {
	var query struct {
		Repository struct {
			Issue struct {
				ID githubv4.ID
			} `graphql:"issue(number: $number)"`
		} `graphql:"repository(owner: $owner, name: $repo)"`
	}

	variables := map[string]any{
		"owner":  githubv4.String(p.owner),
		"repo":   githubv4.String(p.repo),
		"number": githubv4.Int(issueNumber),
	}

	if err := p.client.Query(p.ctx, &query, variables); err != nil {
		return "", fmt.Errorf("failed to get issue node ID: %w", err)
	}

	return string(query.Repository.Issue.ID.(string)), nil
}

// ListProjectItems returns all project items with their status values.
func (p *ProjectsClient) ListProjectItems() ([]ProjectItem, error) {
	projectID, err := p.GetProjectID()
	if err != nil {
		return nil, err
	}

	statusField, err := p.GetStatusField()
	if err != nil {
		return nil, err
	}

	var items []ProjectItem

	var query struct {
		Node struct {
			ProjectV2 struct {
				Items struct {
					Nodes []struct {
						ID      githubv4.ID
						Content struct {
							Issue struct {
								Number   githubv4.Int
								Title    githubv4.String
								State    githubv4.String
								URL      githubv4.String
								Typename string `graphql:"__typename"`
							} `graphql:"... on Issue"`
						}
						FieldValues struct {
							Nodes []struct {
								ProjectV2ItemFieldSingleSelectValue struct {
									Field struct {
										ProjectV2SingleSelectField struct {
											ID githubv4.ID
										} `graphql:"... on ProjectV2SingleSelectField"`
									}
									OptionID githubv4.ID
									Name     githubv4.String
								} `graphql:"... on ProjectV2ItemFieldSingleSelectValue"`
							}
						} `graphql:"fieldValues(first: 20)"`
					}
					PageInfo struct {
						HasNextPage githubv4.Boolean
						EndCursor   githubv4.String
					}
				} `graphql:"items(first: 100, after: $cursor)"`
			} `graphql:"... on ProjectV2"`
		} `graphql:"node(id: $projectId)"`
	}

	var cursor *githubv4.String

	for {
		variables := map[string]any{
			"projectId": githubv4.ID(projectID),
			"cursor":    cursor,
		}

		if err := p.client.Query(p.ctx, &query, variables); err != nil {
			return nil, fmt.Errorf("failed to list project items: %w", err)
		}

		for _, item := range query.Node.ProjectV2.Items.Nodes {
			// Only include issues (not PRs or drafts)
			if item.Content.Issue.Number == 0 {
				continue
			}

			pi := ProjectItem{
				ID:          string(item.ID.(string)),
				IssueNumber: int(item.Content.Issue.Number),
				IssueTitle:  string(item.Content.Issue.Title),
				IssueState:  string(item.Content.Issue.State),
				IssueURL:    string(item.Content.Issue.URL),
			}

			// Find the status field value
			for _, fv := range item.FieldValues.Nodes {
				fieldID := fv.ProjectV2ItemFieldSingleSelectValue.Field.ProjectV2SingleSelectField.ID
				if fieldID != nil && string(fieldID.(string)) == statusField.ID {
					if fv.ProjectV2ItemFieldSingleSelectValue.OptionID != nil {
						pi.FieldValueID = string(fv.ProjectV2ItemFieldSingleSelectValue.OptionID.(string))
					}
					pi.FieldValueStr = string(fv.ProjectV2ItemFieldSingleSelectValue.Name)
					break
				}
			}

			items = append(items, pi)
		}

		if !bool(query.Node.ProjectV2.Items.PageInfo.HasNextPage) {
			break
		}
		cursor = &query.Node.ProjectV2.Items.PageInfo.EndCursor
	}

	return items, nil
}

// MapStatusToOptionID maps a canonical backend status to a project field option ID.
func (p *ProjectsClient) MapStatusToOptionID(status backend.Status, statusField *ProjectField) (string, error) {
	// Default mappings from canonical status to typical project column names
	defaultMappings := map[backend.Status][]string{
		backend.StatusBacklog:    {"Backlog", "📋 Backlog", "Icebox"},
		backend.StatusTodo:       {"Todo", "To Do", "📋 Todo", "Ready"},
		backend.StatusInProgress: {"In Progress", "In progress", "🏗 In progress", "Working"},
		backend.StatusReview:     {"In Review", "Review", "In review", "👀 In review"},
		backend.StatusDone:       {"Done", "✅ Done", "Completed", "Closed"},
	}

	candidates := defaultMappings[status]
	for _, opt := range statusField.Options {
		for _, candidate := range candidates {
			if opt.Name == candidate {
				return opt.ID, nil
			}
		}
	}

	// If no match found, return error with available options
	var optNames []string
	for _, opt := range statusField.Options {
		optNames = append(optNames, opt.Name)
	}
	return "", fmt.Errorf("no project column found for status %q (available: %v)", status, optNames)
}

// DiscoverFields returns all available fields in the project.
// This is useful for discovering what fields are available for configuration.
func (p *ProjectsClient) DiscoverFields() ([]ProjectField, error) {
	projectID, err := p.GetProjectID()
	if err != nil {
		return nil, err
	}

	// Query for all project fields
	var query struct {
		Node struct {
			ProjectV2 struct {
				Fields struct {
					Nodes []struct {
						ProjectV2Field struct {
							ID       githubv4.ID
							Name     githubv4.String
							DataType githubv4.String
						} `graphql:"... on ProjectV2Field"`
						ProjectV2SingleSelectField struct {
							ID      githubv4.ID
							Name    githubv4.String
							Options []struct {
								ID   githubv4.ID
								Name githubv4.String
							}
						} `graphql:"... on ProjectV2SingleSelectField"`
						ProjectV2IterationField struct {
							ID   githubv4.ID
							Name githubv4.String
						} `graphql:"... on ProjectV2IterationField"`
					}
				} `graphql:"fields(first: 50)"`
			} `graphql:"... on ProjectV2"`
		} `graphql:"node(id: $projectId)"`
	}

	variables := map[string]any{
		"projectId": githubv4.ID(projectID),
	}

	if err := p.client.Query(p.ctx, &query, variables); err != nil {
		return nil, fmt.Errorf("failed to discover project fields: %w", err)
	}

	var fields []ProjectField
	for _, field := range query.Node.ProjectV2.Fields.Nodes {
		// Check single-select fields first (most common for status)
		if field.ProjectV2SingleSelectField.ID != nil {
			pf := ProjectField{
				ID:      string(field.ProjectV2SingleSelectField.ID.(string)),
				Name:    string(field.ProjectV2SingleSelectField.Name),
				Options: make([]ProjectFieldValue, len(field.ProjectV2SingleSelectField.Options)),
			}
			for i, opt := range field.ProjectV2SingleSelectField.Options {
				pf.Options[i] = ProjectFieldValue{
					ID:   string(opt.ID.(string)),
					Name: string(opt.Name),
				}
			}
			fields = append(fields, pf)
			continue
		}

		// Check iteration fields
		if field.ProjectV2IterationField.ID != nil {
			pf := ProjectField{
				ID:   string(field.ProjectV2IterationField.ID.(string)),
				Name: string(field.ProjectV2IterationField.Name),
			}
			fields = append(fields, pf)
			continue
		}

		// Check regular fields (text, number, date)
		if field.ProjectV2Field.ID != nil {
			pf := ProjectField{
				ID:   string(field.ProjectV2Field.ID.(string)),
				Name: string(field.ProjectV2Field.Name),
			}
			fields = append(fields, pf)
		}
	}

	return fields, nil
}

// MapOptionToStatus maps a project field option name to a canonical backend status.
func (p *ProjectsClient) MapOptionToStatus(optionName string) backend.Status {
	// Map from typical project column names to canonical status
	switch optionName {
	case "Backlog", "📋 Backlog", "Icebox":
		return backend.StatusBacklog
	case "Todo", "To Do", "📋 Todo", "Ready":
		return backend.StatusTodo
	case "In Progress", "In progress", "🏗 In progress", "Working":
		return backend.StatusInProgress
	case "In Review", "Review", "In review", "👀 In review":
		return backend.StatusReview
	case "Done", "✅ Done", "Completed", "Closed":
		return backend.StatusDone
	default:
		// Default unknown columns to backlog
		return backend.StatusBacklog
	}
}
