package local

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/alexbrand/backlog/internal/backend"
)

// TestMultiAgentSequentialClaim tests the scenario where:
// Agent A claims a task, Agent B tries and fails, Agent A releases, Agent B claims successfully
func TestMultiAgentSequentialClaim(t *testing.T) {
	tmpDir := t.TempDir()
	backlogDir := filepath.Join(tmpDir, ".backlog")

	// Create directory structure
	for _, dir := range []string{"backlog", "todo", "in-progress", "review", "done", ".locks"} {
		if err := os.MkdirAll(filepath.Join(backlogDir, dir), 0755); err != nil {
			t.Fatalf("failed to create dir: %v", err)
		}
	}

	// Create two local backend instances (simulating two agents)
	agentA := New()
	agentB := New()

	cfgA := backend.Config{
		Workspace:        &WorkspaceConfig{Path: backlogDir},
		AgentID:          "agent-A",
		AgentLabelPrefix: "agent",
	}
	cfgB := backend.Config{
		Workspace:        &WorkspaceConfig{Path: backlogDir},
		AgentID:          "agent-B",
		AgentLabelPrefix: "agent",
	}

	if err := agentA.Connect(cfgA); err != nil {
		t.Fatalf("agentA Connect() error = %v", err)
	}
	if err := agentB.Connect(cfgB); err != nil {
		t.Fatalf("agentB Connect() error = %v", err)
	}

	// Create a task
	task, err := agentA.Create(backend.TaskInput{
		Title:  "Shared Task",
		Status: backend.StatusTodo,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Step 1: Agent A claims the task
	resultA, err := agentA.Claim(task.ID, "agent-A")
	if err != nil {
		t.Fatalf("Agent A Claim() error = %v", err)
	}
	if resultA.AlreadyOwned {
		t.Error("Agent A should get fresh claim, not AlreadyOwned")
	}
	if resultA.Task.Status != backend.StatusInProgress {
		t.Errorf("Task status after A's claim = %q, want %q", resultA.Task.Status, backend.StatusInProgress)
	}

	// Step 2: Agent B tries to claim - should fail with conflict
	_, err = agentB.Claim(task.ID, "agent-B")
	if err == nil {
		t.Fatal("Agent B Claim() should fail when task is claimed by Agent A")
	}
	conflictErr, ok := err.(*ClaimConflictError)
	if !ok {
		t.Fatalf("error should be *ClaimConflictError, got %T: %v", err, err)
	}
	if conflictErr.ClaimedBy != "agent-A" {
		t.Errorf("ClaimedBy = %q, want %q", conflictErr.ClaimedBy, "agent-A")
	}

	// Step 3: Agent A releases the task
	if err := agentA.Release(task.ID); err != nil {
		t.Fatalf("Agent A Release() error = %v", err)
	}

	// Verify task is back to todo
	releasedTask, _ := agentA.Get(task.ID)
	if releasedTask.Status != backend.StatusTodo {
		t.Errorf("Task status after release = %q, want %q", releasedTask.Status, backend.StatusTodo)
	}

	// Step 4: Agent B can now claim successfully
	resultB, err := agentB.Claim(task.ID, "agent-B")
	if err != nil {
		t.Fatalf("Agent B Claim() after release error = %v", err)
	}
	if resultB.AlreadyOwned {
		t.Error("Agent B should get fresh claim, not AlreadyOwned")
	}
	if resultB.Task.Status != backend.StatusInProgress {
		t.Errorf("Task status after B's claim = %q, want %q", resultB.Task.Status, backend.StatusInProgress)
	}

	// Verify agent B's label is on the task
	finalTask, _ := agentB.Get(task.ID)
	hasAgentBLabel := false
	hasAgentALabel := false
	for _, label := range finalTask.Labels {
		if label == "agent:agent-B" {
			hasAgentBLabel = true
		}
		if label == "agent:agent-A" {
			hasAgentALabel = true
		}
	}
	if !hasAgentBLabel {
		t.Errorf("Task should have agent:agent-B label, got %v", finalTask.Labels)
	}
	if hasAgentALabel {
		t.Errorf("Task should not have agent:agent-A label after B claimed, got %v", finalTask.Labels)
	}
}

// TestMultiAgentIndependentTasks tests multiple agents claiming different tasks
func TestMultiAgentIndependentTasks(t *testing.T) {
	tmpDir := t.TempDir()
	backlogDir := filepath.Join(tmpDir, ".backlog")

	// Create directory structure
	for _, dir := range []string{"backlog", "todo", "in-progress", "review", "done", ".locks"} {
		if err := os.MkdirAll(filepath.Join(backlogDir, dir), 0755); err != nil {
			t.Fatalf("failed to create dir: %v", err)
		}
	}

	// Create three local backend instances (simulating three agents)
	agents := make([]*Local, 3)
	for i := 0; i < 3; i++ {
		agents[i] = New()
		cfg := backend.Config{
			Workspace:        &WorkspaceConfig{Path: backlogDir},
			AgentID:          "agent-" + string(rune('A'+i)),
			AgentLabelPrefix: "agent",
		}
		if err := agents[i].Connect(cfg); err != nil {
			t.Fatalf("agent[%d] Connect() error = %v", i, err)
		}
	}

	// Create three tasks using agent A
	taskIDs := make([]string, 3)
	for i := 0; i < 3; i++ {
		task, err := agents[0].Create(backend.TaskInput{
			Title:  "Task " + string(rune('1'+i)),
			Status: backend.StatusTodo,
		})
		if err != nil {
			t.Fatalf("Create task %d error = %v", i, err)
		}
		taskIDs[i] = task.ID
	}

	// Each agent claims a different task
	agentNames := []string{"agent-A", "agent-B", "agent-C"}
	for i, agent := range agents {
		result, err := agent.Claim(taskIDs[i], agentNames[i])
		if err != nil {
			t.Fatalf("Agent %d Claim() error = %v", i, err)
		}
		if result.Task.Status != backend.StatusInProgress {
			t.Errorf("Task %d status = %q, want %q", i, result.Task.Status, backend.StatusInProgress)
		}
	}

	// Verify each task is claimed by the correct agent
	for i, agent := range agents {
		task, _ := agent.Get(taskIDs[i])
		expectedLabel := "agent:" + agentNames[i]
		hasCorrectLabel := false
		for _, label := range task.Labels {
			if label == expectedLabel {
				hasCorrectLabel = true
				break
			}
		}
		if !hasCorrectLabel {
			t.Errorf("Task %d should have label %q, got %v", i, expectedLabel, task.Labels)
		}
	}

	// Verify agents cannot cross-claim
	for i := 0; i < 3; i++ {
		otherTaskIdx := (i + 1) % 3
		_, err := agents[i].Claim(taskIDs[otherTaskIdx], agentNames[i])
		if err == nil {
			t.Errorf("Agent %d should not be able to claim task %d owned by agent %d", i, otherTaskIdx, otherTaskIdx)
		}
		if _, ok := err.(*ClaimConflictError); !ok {
			t.Errorf("Expected ClaimConflictError, got %T: %v", err, err)
		}
	}
}

// TestMultiAgentHandoff tests proper task handoff between agents
func TestMultiAgentHandoff(t *testing.T) {
	tmpDir := t.TempDir()
	backlogDir := filepath.Join(tmpDir, ".backlog")

	// Create directory structure
	for _, dir := range []string{"backlog", "todo", "in-progress", "review", "done", ".locks"} {
		if err := os.MkdirAll(filepath.Join(backlogDir, dir), 0755); err != nil {
			t.Fatalf("failed to create dir: %v", err)
		}
	}

	agentA := New()
	agentB := New()

	cfgA := backend.Config{
		Workspace:        &WorkspaceConfig{Path: backlogDir},
		AgentID:          "handoff-agent-A",
		AgentLabelPrefix: "agent",
	}
	cfgB := backend.Config{
		Workspace:        &WorkspaceConfig{Path: backlogDir},
		AgentID:          "handoff-agent-B",
		AgentLabelPrefix: "agent",
	}

	agentA.Connect(cfgA)
	agentB.Connect(cfgB)

	// Create a task
	task, _ := agentA.Create(backend.TaskInput{
		Title:  "Handoff Task",
		Status: backend.StatusTodo,
	})

	// Agent A claims
	agentA.Claim(task.ID, "handoff-agent-A")

	// Agent A does some work and moves to review (simulating partial completion)
	agentA.Move(task.ID, backend.StatusReview)

	// Verify task is in review but still claimed by A
	taskInReview, _ := agentA.Get(task.ID)
	if taskInReview.Status != backend.StatusReview {
		t.Errorf("Task status = %q, want %q", taskInReview.Status, backend.StatusReview)
	}

	// Agent A releases
	agentA.Release(task.ID)

	// Verify task stays in todo after release (as per release behavior)
	taskAfterRelease, _ := agentA.Get(task.ID)
	if taskAfterRelease.Status != backend.StatusTodo {
		t.Errorf("Task status after release = %q, want %q", taskAfterRelease.Status, backend.StatusTodo)
	}

	// Agent B claims the task
	resultB, err := agentB.Claim(task.ID, "handoff-agent-B")
	if err != nil {
		t.Fatalf("Agent B Claim() error = %v", err)
	}

	// Verify B owns it now
	hasAgentBLabel := false
	for _, label := range resultB.Task.Labels {
		if label == "agent:handoff-agent-B" {
			hasAgentBLabel = true
			break
		}
	}
	if !hasAgentBLabel {
		t.Errorf("Task should have agent:handoff-agent-B label after handoff, got %v", resultB.Task.Labels)
	}
}

// TestMultiAgentExpiredLockTakeover tests that an agent can take over when lock expires
func TestMultiAgentExpiredLockTakeover(t *testing.T) {
	tmpDir := t.TempDir()
	backlogDir := filepath.Join(tmpDir, ".backlog")

	// Create directory structure
	for _, dir := range []string{"backlog", "todo", "in-progress", "review", "done", ".locks"} {
		if err := os.MkdirAll(filepath.Join(backlogDir, dir), 0755); err != nil {
			t.Fatalf("failed to create dir: %v", err)
		}
	}

	agentA := New()
	agentB := New()

	cfgA := backend.Config{
		Workspace:        &WorkspaceConfig{Path: backlogDir},
		AgentID:          "expired-agent-A",
		AgentLabelPrefix: "agent",
	}
	cfgB := backend.Config{
		Workspace:        &WorkspaceConfig{Path: backlogDir},
		AgentID:          "takeover-agent-B",
		AgentLabelPrefix: "agent",
	}

	agentA.Connect(cfgA)
	agentB.Connect(cfgB)

	// Create a task
	task, _ := agentA.Create(backend.TaskInput{
		Title:  "Lock Expiry Task",
		Status: backend.StatusTodo,
	})

	// Agent A claims
	agentA.Claim(task.ID, "expired-agent-A")

	// Manually expire the lock (simulating time passing)
	expiredLock := &LockFile{
		Agent:     "expired-agent-A",
		ClaimedAt: time.Now().UTC().Add(-2 * time.Hour),
		ExpiresAt: time.Now().UTC().Add(-1 * time.Hour), // Expired 1 hour ago
	}
	agentA.writeLock(task.ID, expiredLock)

	// Agent B should now be able to claim despite A's label being present
	resultB, err := agentB.Claim(task.ID, "takeover-agent-B")
	if err != nil {
		t.Fatalf("Agent B Claim() after lock expiry error = %v", err)
	}

	if resultB.AlreadyOwned {
		t.Error("Agent B should get fresh claim, not AlreadyOwned")
	}

	// Verify B's lock is now in place
	lock, _ := agentB.readLock(task.ID)
	if lock == nil {
		t.Fatal("Lock should exist after B's claim")
	}
	if lock.Agent != "takeover-agent-B" {
		t.Errorf("Lock agent = %q, want %q", lock.Agent, "takeover-agent-B")
	}
	if !lock.isActive() {
		t.Error("B's lock should be active")
	}
}

// TestMultiAgentReleaseConflicts tests that agents cannot release each other's tasks
func TestMultiAgentReleaseConflicts(t *testing.T) {
	tmpDir := t.TempDir()
	backlogDir := filepath.Join(tmpDir, ".backlog")

	// Create directory structure
	for _, dir := range []string{"backlog", "todo", "in-progress", "review", "done", ".locks"} {
		if err := os.MkdirAll(filepath.Join(backlogDir, dir), 0755); err != nil {
			t.Fatalf("failed to create dir: %v", err)
		}
	}

	agentA := New()
	agentB := New()

	cfgA := backend.Config{
		Workspace:        &WorkspaceConfig{Path: backlogDir},
		AgentID:          "owner-agent",
		AgentLabelPrefix: "agent",
	}
	cfgB := backend.Config{
		Workspace:        &WorkspaceConfig{Path: backlogDir},
		AgentID:          "intruder-agent",
		AgentLabelPrefix: "agent",
	}

	agentA.Connect(cfgA)
	agentB.Connect(cfgB)

	// Create and claim a task as Agent A
	task, _ := agentA.Create(backend.TaskInput{
		Title:  "Protected Task",
		Status: backend.StatusTodo,
	})
	agentA.Claim(task.ID, "owner-agent")

	// Agent B tries to release A's task - should fail
	err := agentB.Release(task.ID)
	if err == nil {
		t.Fatal("Agent B should not be able to release Agent A's task")
	}

	releaseErr, ok := err.(*ReleaseConflictError)
	if !ok {
		t.Fatalf("error should be *ReleaseConflictError, got %T: %v", err, err)
	}
	if releaseErr.ClaimedBy != "owner-agent" {
		t.Errorf("ReleaseConflictError.ClaimedBy = %q, want %q", releaseErr.ClaimedBy, "owner-agent")
	}

	// Verify task is still claimed by A
	taskStillClaimed, _ := agentA.Get(task.ID)
	if taskStillClaimed.Status != backend.StatusInProgress {
		t.Errorf("Task should still be in-progress, got %q", taskStillClaimed.Status)
	}

	lock, _ := agentA.readLock(task.ID)
	if lock == nil || lock.Agent != "owner-agent" {
		t.Error("Lock should still belong to owner-agent")
	}
}

// TestMultiAgentClaimWithCustomLabelPrefix tests claiming with custom label prefixes
func TestMultiAgentClaimWithCustomLabelPrefix(t *testing.T) {
	tmpDir := t.TempDir()
	backlogDir := filepath.Join(tmpDir, ".backlog")

	// Create directory structure
	for _, dir := range []string{"backlog", "todo", "in-progress", "review", "done", ".locks"} {
		if err := os.MkdirAll(filepath.Join(backlogDir, dir), 0755); err != nil {
			t.Fatalf("failed to create dir: %v", err)
		}
	}

	agentA := New()
	agentB := New()

	// Use custom prefix "worker" instead of default "agent"
	cfgA := backend.Config{
		Workspace:        &WorkspaceConfig{Path: backlogDir},
		AgentID:          "bot-1",
		AgentLabelPrefix: "worker",
	}
	cfgB := backend.Config{
		Workspace:        &WorkspaceConfig{Path: backlogDir},
		AgentID:          "bot-2",
		AgentLabelPrefix: "worker",
	}

	agentA.Connect(cfgA)
	agentB.Connect(cfgB)

	// Create a task
	task, _ := agentA.Create(backend.TaskInput{
		Title:  "Custom Prefix Task",
		Status: backend.StatusTodo,
	})

	// Agent A claims
	result, err := agentA.Claim(task.ID, "bot-1")
	if err != nil {
		t.Fatalf("Claim() error = %v", err)
	}

	// Verify the custom prefix is used
	hasCustomLabel := false
	for _, label := range result.Task.Labels {
		if label == "worker:bot-1" {
			hasCustomLabel = true
			break
		}
	}
	if !hasCustomLabel {
		t.Errorf("Task should have worker:bot-1 label, got %v", result.Task.Labels)
	}

	// Agent B should not be able to claim
	_, err = agentB.Claim(task.ID, "bot-2")
	if err == nil {
		t.Fatal("Agent B should not be able to claim task owned by Agent A")
	}
	if _, ok := err.(*ClaimConflictError); !ok {
		t.Fatalf("Expected ClaimConflictError, got %T: %v", err, err)
	}
}

// TestMultiAgentRepeatedClaimReleaseCycles tests multiple claim/release cycles
func TestMultiAgentRepeatedClaimReleaseCycles(t *testing.T) {
	tmpDir := t.TempDir()
	backlogDir := filepath.Join(tmpDir, ".backlog")

	// Create directory structure
	for _, dir := range []string{"backlog", "todo", "in-progress", "review", "done", ".locks"} {
		if err := os.MkdirAll(filepath.Join(backlogDir, dir), 0755); err != nil {
			t.Fatalf("failed to create dir: %v", err)
		}
	}

	agentA := New()
	agentB := New()

	cfgA := backend.Config{
		Workspace:        &WorkspaceConfig{Path: backlogDir},
		AgentID:          "cycle-agent-A",
		AgentLabelPrefix: "agent",
	}
	cfgB := backend.Config{
		Workspace:        &WorkspaceConfig{Path: backlogDir},
		AgentID:          "cycle-agent-B",
		AgentLabelPrefix: "agent",
	}

	agentA.Connect(cfgA)
	agentB.Connect(cfgB)

	// Create a task
	task, _ := agentA.Create(backend.TaskInput{
		Title:  "Cycle Task",
		Status: backend.StatusTodo,
	})

	agents := []*Local{agentA, agentB}
	agentNames := []string{"cycle-agent-A", "cycle-agent-B"}

	// Alternate between agents claiming and releasing
	for cycle := 0; cycle < 4; cycle++ {
		agentIdx := cycle % 2
		agent := agents[agentIdx]
		agentName := agentNames[agentIdx]

		// Claim
		result, err := agent.Claim(task.ID, agentName)
		if err != nil {
			t.Fatalf("Cycle %d: Agent %s Claim() error = %v", cycle, agentName, err)
		}
		if result.Task.Status != backend.StatusInProgress {
			t.Errorf("Cycle %d: Task status = %q, want %q", cycle, result.Task.Status, backend.StatusInProgress)
		}

		// Verify lock
		lock, _ := agent.readLock(task.ID)
		if lock == nil || lock.Agent != agentName {
			t.Errorf("Cycle %d: Lock should belong to %s", cycle, agentName)
		}

		// Release
		if err := agent.Release(task.ID); err != nil {
			t.Fatalf("Cycle %d: Agent %s Release() error = %v", cycle, agentName, err)
		}

		// Verify no lock
		lock, _ = agent.readLock(task.ID)
		if lock != nil {
			t.Errorf("Cycle %d: Lock should be removed after release", cycle)
		}
	}
}
