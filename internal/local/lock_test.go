package local

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/alexbrand/backlog/internal/backend"
)

func TestParseLockFile(t *testing.T) {
	content := `agent: test-agent
claimed_at: 2025-01-15T10:00:00Z
expires_at: 2025-01-15T10:30:00Z
`
	lock, err := parseLockFile([]byte(content))
	if err != nil {
		t.Fatalf("parseLockFile() error = %v", err)
	}

	if lock.Agent != "test-agent" {
		t.Errorf("Agent = %q, want %q", lock.Agent, "test-agent")
	}

	expectedClaimed := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)
	if !lock.ClaimedAt.Equal(expectedClaimed) {
		t.Errorf("ClaimedAt = %v, want %v", lock.ClaimedAt, expectedClaimed)
	}

	expectedExpires := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)
	if !lock.ExpiresAt.Equal(expectedExpires) {
		t.Errorf("ExpiresAt = %v, want %v", lock.ExpiresAt, expectedExpires)
	}
}

func TestFormatLockFile(t *testing.T) {
	lock := &LockFile{
		Agent:     "my-agent",
		ClaimedAt: time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC),
		ExpiresAt: time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC),
	}

	content := formatLockFile(lock)

	// Parse it back
	parsed, err := parseLockFile([]byte(content))
	if err != nil {
		t.Fatalf("round-trip failed: %v", err)
	}

	if parsed.Agent != lock.Agent {
		t.Errorf("Agent = %q, want %q", parsed.Agent, lock.Agent)
	}
	if !parsed.ClaimedAt.Equal(lock.ClaimedAt) {
		t.Errorf("ClaimedAt = %v, want %v", parsed.ClaimedAt, lock.ClaimedAt)
	}
	if !parsed.ExpiresAt.Equal(lock.ExpiresAt) {
		t.Errorf("ExpiresAt = %v, want %v", parsed.ExpiresAt, lock.ExpiresAt)
	}
}

func TestLockIsActive(t *testing.T) {
	tests := []struct {
		name     string
		lock     *LockFile
		expected bool
	}{
		{
			name: "active lock",
			lock: &LockFile{
				ExpiresAt: time.Now().UTC().Add(10 * time.Minute),
			},
			expected: true,
		},
		{
			name: "expired lock",
			lock: &LockFile{
				ExpiresAt: time.Now().UTC().Add(-10 * time.Minute),
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.lock.isActive(); got != tt.expected {
				t.Errorf("isActive() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestClaim(t *testing.T) {
	tmpDir := t.TempDir()
	backlogDir := filepath.Join(tmpDir, ".backlog")

	// Create directory structure
	for _, dir := range []string{"backlog", "todo", "in-progress", "review", "done", ".locks"} {
		if err := os.MkdirAll(filepath.Join(backlogDir, dir), 0755); err != nil {
			t.Fatalf("failed to create dir: %v", err)
		}
	}

	l := New()
	cfg := backend.Config{
		Workspace:        &WorkspaceConfig{Path: backlogDir},
		AgentID:          "test-agent",
		AgentLabelPrefix: "agent",
	}
	if err := l.Connect(cfg); err != nil {
		t.Fatalf("Connect() error = %v", err)
	}

	// Create a task in todo
	task, err := l.Create(backend.TaskInput{
		Title:  "Test Task",
		Status: backend.StatusTodo,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Claim the task
	result, err := l.Claim(task.ID, "test-agent")
	if err != nil {
		t.Fatalf("Claim() error = %v", err)
	}

	if result.AlreadyOwned {
		t.Error("AlreadyOwned should be false for new claim")
	}

	if result.Task.Status != backend.StatusInProgress {
		t.Errorf("Task.Status = %q, want %q", result.Task.Status, backend.StatusInProgress)
	}

	// Verify lock file exists
	lock, err := l.readLock(task.ID)
	if err != nil {
		t.Fatalf("readLock() error = %v", err)
	}
	if lock == nil {
		t.Fatal("lock file should exist after claim")
	}
	if lock.Agent != "test-agent" {
		t.Errorf("lock.Agent = %q, want %q", lock.Agent, "test-agent")
	}

	// Verify agent label was added
	claimedTask, _ := l.Get(task.ID)
	hasLabel := false
	for _, label := range claimedTask.Labels {
		if label == "agent:test-agent" {
			hasLabel = true
			break
		}
	}
	if !hasLabel {
		t.Errorf("Task should have agent:test-agent label, got %v", claimedTask.Labels)
	}
}

func TestClaimAlreadyOwnedBySameAgent(t *testing.T) {
	tmpDir := t.TempDir()
	backlogDir := filepath.Join(tmpDir, ".backlog")

	// Create directory structure
	for _, dir := range []string{"backlog", "todo", "in-progress", "review", "done", ".locks"} {
		if err := os.MkdirAll(filepath.Join(backlogDir, dir), 0755); err != nil {
			t.Fatalf("failed to create dir: %v", err)
		}
	}

	l := New()
	cfg := backend.Config{
		Workspace:        &WorkspaceConfig{Path: backlogDir},
		AgentID:          "test-agent",
		AgentLabelPrefix: "agent",
	}
	if err := l.Connect(cfg); err != nil {
		t.Fatalf("Connect() error = %v", err)
	}

	// Create and claim a task
	task, _ := l.Create(backend.TaskInput{
		Title:  "Test Task",
		Status: backend.StatusTodo,
	})
	_, _ = l.Claim(task.ID, "test-agent")

	// Claim again by same agent
	result, err := l.Claim(task.ID, "test-agent")
	if err != nil {
		t.Fatalf("Claim() error = %v", err)
	}

	if !result.AlreadyOwned {
		t.Error("AlreadyOwned should be true when same agent claims again")
	}
}

func TestClaimConflict(t *testing.T) {
	tmpDir := t.TempDir()
	backlogDir := filepath.Join(tmpDir, ".backlog")

	// Create directory structure
	for _, dir := range []string{"backlog", "todo", "in-progress", "review", "done", ".locks"} {
		if err := os.MkdirAll(filepath.Join(backlogDir, dir), 0755); err != nil {
			t.Fatalf("failed to create dir: %v", err)
		}
	}

	l := New()
	cfg := backend.Config{
		Workspace:        &WorkspaceConfig{Path: backlogDir},
		AgentID:          "agent-1",
		AgentLabelPrefix: "agent",
	}
	if err := l.Connect(cfg); err != nil {
		t.Fatalf("Connect() error = %v", err)
	}

	// Create and claim a task
	task, _ := l.Create(backend.TaskInput{
		Title:  "Test Task",
		Status: backend.StatusTodo,
	})
	_, _ = l.Claim(task.ID, "agent-1")

	// Try to claim by different agent
	_, err := l.Claim(task.ID, "agent-2")
	if err == nil {
		t.Fatal("Claim() should return error when task claimed by another agent")
	}

	conflictErr, ok := err.(*ClaimConflictError)
	if !ok {
		t.Fatalf("error should be *ClaimConflictError, got %T", err)
	}
	if conflictErr.ClaimedBy != "agent-1" {
		t.Errorf("ClaimedBy = %q, want %q", conflictErr.ClaimedBy, "agent-1")
	}
}

func TestClaimExpiredLock(t *testing.T) {
	tmpDir := t.TempDir()
	backlogDir := filepath.Join(tmpDir, ".backlog")

	// Create directory structure
	for _, dir := range []string{"backlog", "todo", "in-progress", "review", "done", ".locks"} {
		if err := os.MkdirAll(filepath.Join(backlogDir, dir), 0755); err != nil {
			t.Fatalf("failed to create dir: %v", err)
		}
	}

	l := New()
	cfg := backend.Config{
		Workspace:        &WorkspaceConfig{Path: backlogDir},
		AgentID:          "new-agent",
		AgentLabelPrefix: "agent",
	}
	if err := l.Connect(cfg); err != nil {
		t.Fatalf("Connect() error = %v", err)
	}

	// Create a task
	task, _ := l.Create(backend.TaskInput{
		Title:  "Test Task",
		Status: backend.StatusTodo,
	})

	// Create an expired lock file manually
	expiredLock := &LockFile{
		Agent:     "old-agent",
		ClaimedAt: time.Now().UTC().Add(-1 * time.Hour),
		ExpiresAt: time.Now().UTC().Add(-30 * time.Minute),
	}
	_ = l.writeLock(task.ID, expiredLock)

	// New agent should be able to claim
	result, err := l.Claim(task.ID, "new-agent")
	if err != nil {
		t.Fatalf("Claim() error = %v, expected success for expired lock", err)
	}

	if result.AlreadyOwned {
		t.Error("AlreadyOwned should be false")
	}

	// Verify lock was updated
	lock, _ := l.readLock(task.ID)
	if lock.Agent != "new-agent" {
		t.Errorf("lock.Agent = %q, want %q", lock.Agent, "new-agent")
	}
}

func TestRelease(t *testing.T) {
	tmpDir := t.TempDir()
	backlogDir := filepath.Join(tmpDir, ".backlog")

	// Create directory structure
	for _, dir := range []string{"backlog", "todo", "in-progress", "review", "done", ".locks"} {
		if err := os.MkdirAll(filepath.Join(backlogDir, dir), 0755); err != nil {
			t.Fatalf("failed to create dir: %v", err)
		}
	}

	l := New()
	cfg := backend.Config{
		Workspace:        &WorkspaceConfig{Path: backlogDir},
		AgentID:          "test-agent",
		AgentLabelPrefix: "agent",
	}
	if err := l.Connect(cfg); err != nil {
		t.Fatalf("Connect() error = %v", err)
	}

	// Create and claim a task
	task, _ := l.Create(backend.TaskInput{
		Title:  "Test Task",
		Status: backend.StatusTodo,
	})
	_, _ = l.Claim(task.ID, "test-agent")

	// Release the task
	err := l.Release(task.ID)
	if err != nil {
		t.Fatalf("Release() error = %v", err)
	}

	// Verify lock file was removed
	lock, _ := l.readLock(task.ID)
	if lock != nil {
		t.Error("lock file should be removed after release")
	}

	// Verify task is back to todo
	releasedTask, _ := l.Get(task.ID)
	if releasedTask.Status != backend.StatusTodo {
		t.Errorf("Task.Status = %q, want %q", releasedTask.Status, backend.StatusTodo)
	}

	// Verify agent label was removed
	for _, label := range releasedTask.Labels {
		if label == "agent:test-agent" {
			t.Error("agent label should be removed after release")
		}
	}
}

func TestFindAgentLabels(t *testing.T) {
	l := &Local{agentLabelPrefix: "agent"}

	tests := []struct {
		name     string
		labels   []string
		expected []string
	}{
		{
			name:     "no agent labels",
			labels:   []string{"bug", "feature"},
			expected: nil,
		},
		{
			name:     "one agent label",
			labels:   []string{"bug", "agent:claude-1", "feature"},
			expected: []string{"agent:claude-1"},
		},
		{
			name:     "multiple agent labels",
			labels:   []string{"agent:agent-1", "bug", "agent:agent-2"},
			expected: []string{"agent:agent-1", "agent:agent-2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := l.findAgentLabels(tt.labels)
			if len(got) != len(tt.expected) {
				t.Errorf("findAgentLabels() = %v, want %v", got, tt.expected)
				return
			}
			for i, v := range got {
				if v != tt.expected[i] {
					t.Errorf("findAgentLabels()[%d] = %q, want %q", i, v, tt.expected[i])
				}
			}
		})
	}
}
