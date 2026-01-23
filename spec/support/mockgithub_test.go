package support

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestMockGitHubServer_NewServer(t *testing.T) {
	server := NewMockGitHubServer()
	defer server.Close()

	if server.URL == "" {
		t.Error("expected server URL to be set")
	}

	if server.Server == nil {
		t.Error("expected server to be created")
	}
}

func TestMockGitHubServer_GetUser(t *testing.T) {
	server := NewMockGitHubServer()
	defer server.Close()
	server.AuthenticatedUser = "testuser"

	resp, err := http.Get(server.URL + "/user")
	if err != nil {
		t.Fatalf("failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if result["login"] != "testuser" {
		t.Errorf("expected login 'testuser', got %v", result["login"])
	}
}

func TestMockGitHubServer_AuthValidation(t *testing.T) {
	server := NewMockGitHubServer()
	defer server.Close()
	server.AuthErrorEnabled = true

	// Request without auth should fail
	resp, err := http.Get(server.URL + "/user")
	if err != nil {
		t.Fatalf("failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", resp.StatusCode)
	}

	// Request with valid auth should succeed
	req, _ := http.NewRequest("GET", server.URL+"/user", nil)
	req.Header.Set("Authorization", "token valid_token")
	resp2, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("failed to make request: %v", err)
	}
	defer resp2.Body.Close()

	if resp2.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp2.StatusCode)
	}
}

func TestMockGitHubServer_ExpectedToken(t *testing.T) {
	server := NewMockGitHubServer()
	defer server.Close()
	server.ExpectedToken = "secret_token"

	// Request with wrong token should fail
	req, _ := http.NewRequest("GET", server.URL+"/user", nil)
	req.Header.Set("Authorization", "token wrong_token")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", resp.StatusCode)
	}

	// Request with correct token should succeed
	req2, _ := http.NewRequest("GET", server.URL+"/user", nil)
	req2.Header.Set("Authorization", "token secret_token")
	resp2, err := http.DefaultClient.Do(req2)
	if err != nil {
		t.Fatalf("failed to make request: %v", err)
	}
	defer resp2.Body.Close()

	if resp2.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp2.StatusCode)
	}
}

func TestMockGitHubServer_ListIssues(t *testing.T) {
	server := NewMockGitHubServer()
	defer server.Close()

	server.SetIssues([]MockGitHubIssue{
		{Number: 1, Title: "First issue", State: "open", Labels: []string{"bug"}},
		{Number: 2, Title: "Second issue", State: "open", Labels: []string{"feature"}},
		{Number: 3, Title: "Closed issue", State: "closed"},
	})

	resp, err := http.Get(server.URL + "/repos/owner/repo/issues")
	if err != nil {
		t.Fatalf("failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	var issues []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&issues); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Default is state=open, so should return 2 issues
	if len(issues) != 2 {
		t.Errorf("expected 2 issues, got %d", len(issues))
	}
}

func TestMockGitHubServer_ListIssues_AllState(t *testing.T) {
	server := NewMockGitHubServer()
	defer server.Close()

	server.SetIssues([]MockGitHubIssue{
		{Number: 1, Title: "Open issue", State: "open"},
		{Number: 2, Title: "Closed issue", State: "closed"},
	})

	resp, err := http.Get(server.URL + "/repos/owner/repo/issues?state=all")
	if err != nil {
		t.Fatalf("failed to make request: %v", err)
	}
	defer resp.Body.Close()

	var issues []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&issues); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(issues) != 2 {
		t.Errorf("expected 2 issues, got %d", len(issues))
	}
}

func TestMockGitHubServer_GetIssue(t *testing.T) {
	server := NewMockGitHubServer()
	defer server.Close()

	server.SetIssues([]MockGitHubIssue{
		{Number: 42, Title: "Test issue", State: "open", Body: "Issue body", Assignee: "alice"},
	})

	resp, err := http.Get(server.URL + "/repos/owner/repo/issues/42")
	if err != nil {
		t.Fatalf("failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	var issue map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&issue); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if issue["number"].(float64) != 42 {
		t.Errorf("expected number 42, got %v", issue["number"])
	}
	if issue["title"] != "Test issue" {
		t.Errorf("expected title 'Test issue', got %v", issue["title"])
	}
	if issue["body"] != "Issue body" {
		t.Errorf("expected body 'Issue body', got %v", issue["body"])
	}
}

func TestMockGitHubServer_GetIssue_NotFound(t *testing.T) {
	server := NewMockGitHubServer()
	defer server.Close()

	resp, err := http.Get(server.URL + "/repos/owner/repo/issues/999")
	if err != nil {
		t.Fatalf("failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", resp.StatusCode)
	}
}

func TestMockGitHubServer_CreateIssue(t *testing.T) {
	server := NewMockGitHubServer()
	defer server.Close()

	body := strings.NewReader(`{"title":"New issue","body":"Issue description","labels":["bug"]}`)
	resp, err := http.Post(server.URL+"/repos/owner/repo/issues", "application/json", body)
	if err != nil {
		t.Fatalf("failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("expected status 201, got %d", resp.StatusCode)
	}

	var issue map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&issue); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if issue["title"] != "New issue" {
		t.Errorf("expected title 'New issue', got %v", issue["title"])
	}
	if issue["number"].(float64) != 1 {
		t.Errorf("expected number 1, got %v", issue["number"])
	}
}

func TestMockGitHubServer_UpdateIssue(t *testing.T) {
	server := NewMockGitHubServer()
	defer server.Close()

	server.SetIssues([]MockGitHubIssue{
		{Number: 1, Title: "Original title", State: "open"},
	})

	body := strings.NewReader(`{"title":"Updated title","state":"closed"}`)
	req, _ := http.NewRequest("PATCH", server.URL+"/repos/owner/repo/issues/1", body)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	var issue map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&issue); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if issue["title"] != "Updated title" {
		t.Errorf("expected title 'Updated title', got %v", issue["title"])
	}
	if issue["state"] != "closed" {
		t.Errorf("expected state 'closed', got %v", issue["state"])
	}
}

func TestMockGitHubServer_Comments(t *testing.T) {
	server := NewMockGitHubServer()
	defer server.Close()

	server.SetIssues([]MockGitHubIssue{
		{Number: 1, Title: "Test issue", State: "open"},
	})

	server.SetComments(1, []MockGitHubComment{
		{ID: 1, Author: "alice", Body: "First comment"},
		{ID: 2, Author: "bob", Body: "Second comment"},
	})

	// List comments
	resp, err := http.Get(server.URL + "/repos/owner/repo/issues/1/comments")
	if err != nil {
		t.Fatalf("failed to make request: %v", err)
	}
	defer resp.Body.Close()

	var comments []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&comments); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(comments) != 2 {
		t.Errorf("expected 2 comments, got %d", len(comments))
	}

	// Create comment
	body := strings.NewReader(`{"body":"New comment"}`)
	resp2, err := http.Post(server.URL+"/repos/owner/repo/issues/1/comments", "application/json", body)
	if err != nil {
		t.Fatalf("failed to make request: %v", err)
	}
	defer resp2.Body.Close()

	if resp2.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp2.Body)
		t.Errorf("expected status 201, got %d: %s", resp2.StatusCode, respBody)
	}
}

func TestMockGitHubServer_Labels(t *testing.T) {
	server := NewMockGitHubServer()
	defer server.Close()

	server.SetIssues([]MockGitHubIssue{
		{Number: 1, Title: "Test issue", State: "open", Labels: []string{"bug"}},
	})

	// List labels
	resp, err := http.Get(server.URL + "/repos/owner/repo/issues/1/labels")
	if err != nil {
		t.Fatalf("failed to make request: %v", err)
	}
	defer resp.Body.Close()

	var labels []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&labels); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(labels) != 1 {
		t.Errorf("expected 1 label, got %d", len(labels))
	}

	// Add labels
	body := strings.NewReader(`{"labels":["feature","enhancement"]}`)
	resp2, err := http.Post(server.URL+"/repos/owner/repo/issues/1/labels", "application/json", body)
	if err != nil {
		t.Fatalf("failed to make request: %v", err)
	}
	defer resp2.Body.Close()

	if resp2.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp2.StatusCode)
	}

	// Verify labels were added
	if err := json.NewDecoder(resp2.Body).Decode(&labels); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(labels) != 3 {
		t.Errorf("expected 3 labels after adding, got %d", len(labels))
	}

	// Delete label
	req, _ := http.NewRequest("DELETE", server.URL+"/repos/owner/repo/issues/1/labels/bug", nil)
	resp3, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("failed to make request: %v", err)
	}
	defer resp3.Body.Close()

	if resp3.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp3.StatusCode)
	}

	// Verify label was removed
	if err := json.NewDecoder(resp3.Body).Decode(&labels); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(labels) != 2 {
		t.Errorf("expected 2 labels after deletion, got %d", len(labels))
	}

	for _, label := range labels {
		if label["name"] == "bug" {
			t.Error("bug label should have been removed")
		}
	}
}

func TestMockGitHubServer_Pagination(t *testing.T) {
	server := NewMockGitHubServer()
	defer server.Close()

	// Create 5 issues
	var issues []MockGitHubIssue
	for i := 1; i <= 5; i++ {
		issues = append(issues, MockGitHubIssue{
			Number: i,
			Title:  "Issue " + string(rune('0'+i)),
			State:  "open",
		})
	}
	server.SetIssues(issues)

	// Request with limit
	resp, err := http.Get(server.URL + "/repos/owner/repo/issues?per_page=2")
	if err != nil {
		t.Fatalf("failed to make request: %v", err)
	}
	defer resp.Body.Close()

	var result []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(result) != 2 {
		t.Errorf("expected 2 issues, got %d", len(result))
	}

	// Should have Link header for pagination
	linkHeader := resp.Header.Get("Link")
	if linkHeader == "" {
		t.Error("expected Link header for pagination")
	}
}
