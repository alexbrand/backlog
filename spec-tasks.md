# backlog CLI - Executable Specification Tasks

## Overview

This plan covers implementing a Gherkin-based executable specification using [godog](https://github.com/cucumber/godog) to verify the backlog CLI is built to spec.

---

## Phase 0: Test Infrastructure Setup

### Project Setup
- [x] Add godog dependency (`go get github.com/cucumber/godog`)
- [x] Create `spec/` directory structure
- [x] Create `spec/features/` for `.feature` files
- [x] Create `spec/steps/` for step definitions
- [x] Create `spec/support/` for test helpers and fixtures
- [x] Create `spec/main_test.go` — godog test runner entry point

### Test Harness
- [x] Implement temp directory setup/teardown for isolated test runs
- [x] Implement CLI runner helper (executes `backlog` commands, captures stdout/stderr/exit code)
- [x] Implement fixture loader for pre-built `.backlog/` directories
- [x] Implement JSON output parser for structured assertions
- [x] Implement task file reader for verifying file state
- [x] Implement config file generator for test workspaces

### Common Step Definitions
- [x] Implement "Given a fresh backlog directory" — creates temp `.backlog/`
- [x] Implement "Given a backlog with the following tasks:" — table-based setup
- [x] Implement "When I run {string}" — executes CLI command
- [x] Implement "Then the exit code should be {int}"
- [x] Implement "Then stdout should contain {string}"
- [x] Implement "Then stderr should contain {string}"
- [x] Implement "Then stdout should be empty"
- [x] Implement "Then stderr should be empty"
- [x] Implement "Then the output should match:" — docstring comparison
- [x] Implement "Then the JSON output should have {string} equal to {string}"

---

## Phase 1: Local Backend — Basic CRUD Specs

### Feature: Initialization (`init.feature`)
- [x] Scenario: Initialize backlog in empty directory
- [x] Scenario: Initialize backlog in directory with existing files
- [x] Scenario: Initialize fails if `.backlog/` already exists
- [x] Scenario: Initialize creates all status directories
- [x] Scenario: Initialize creates config.yaml

### Feature: Adding Tasks (`add.feature`)
- [x] Scenario: Add task with title only
- [x] Scenario: Add task with priority flag
- [x] Scenario: Add task with multiple labels
- [x] Scenario: Add task with description flag
- [x] Scenario: Add task with body-file flag
- [x] Scenario: Add task with explicit status
- [x] Scenario: Add task generates unique ID
- [x] Scenario: Add task outputs created task ID
- [x] Scenario: Add task with JSON output format
- [x] Scenario Outline: Add task with each priority level

### Feature: Listing Tasks (`list.feature`)
- [x] Scenario: List all tasks (excludes done by default)
- [x] Scenario: List tasks in table format (default)
- [x] Scenario: List tasks in JSON format
- [x] Scenario: List tasks in plain format
- [x] Scenario: List tasks in id-only format
- [ ] Scenario: List with status filter
- [ ] Scenario: List with multiple status values
- [ ] Scenario: List with priority filter
- [ ] Scenario: List with label filter
- [ ] Scenario: List with assignee filter
- [ ] Scenario: List unassigned tasks
- [ ] Scenario: List with limit
- [ ] Scenario: List returns empty when no tasks match
- [ ] Scenario: List shows hasMore flag when more tasks exist

### Feature: Showing Tasks (`show.feature`)
- [ ] Scenario: Show task displays all fields
- [ ] Scenario: Show task in JSON format
- [ ] Scenario: Show task with comments
- [ ] Scenario: Show non-existent task returns exit code 3
- [ ] Scenario: Show task displays correct status directory

### Feature: Moving Tasks (`move.feature`)
- [ ] Scenario Outline: Move task to each valid status
- [ ] Scenario: Move task updates file location
- [ ] Scenario: Move task updates frontmatter
- [ ] Scenario: Move to invalid status fails
- [ ] Scenario: Move non-existent task returns exit code 3
- [ ] Scenario: Move task with comment flag

### Feature: Editing Tasks (`edit.feature`)
- [ ] Scenario: Edit task title
- [ ] Scenario: Edit task priority
- [ ] Scenario: Edit task description
- [ ] Scenario: Add label to task
- [ ] Scenario: Remove label from task
- [ ] Scenario: Edit multiple fields at once
- [ ] Scenario: Edit non-existent task returns exit code 3
- [ ] Scenario: Edit preserves unmodified fields

### Step Definitions - Phase 1
- [ ] Implement "Given a task {string} exists with status {string}"
- [ ] Implement "Given a task {string} exists with priority {string}"
- [ ] Implement "Given a task {string} exists with labels {string}"
- [x] Implement "Then a task file should exist in {string} directory"
- [ ] Implement "Then the task {string} should have status {string}"
- [x] Implement "Then the created task should have priority {string}"
- [x] Implement "Then the created task should have label {string}"
- [ ] Implement "Then the task {string} should not have label {string}"
- [x] Implement "Then the task count should be {int}"
- [x] Implement "Then stdout should match pattern {string}"
- [x] Implement "Then the created task should have description containing {string}"

---

## Phase 1b: Output Format Specs

### Feature: Table Output (`output_table.feature`)
- [ ] Scenario: Table output has correct headers
- [ ] Scenario: Table output aligns columns
- [ ] Scenario: Table output truncates long titles
- [ ] Scenario: Table output shows dash for empty fields

### Feature: JSON Output (`output_json.feature`)
- [ ] Scenario: JSON output is valid JSON
- [ ] Scenario: JSON output includes all task fields
- [ ] Scenario: JSON output includes count and hasMore
- [ ] Scenario: JSON error output format

### Feature: Plain Output (`output_plain.feature`)
- [ ] Scenario: Plain output shows one task per line
- [ ] Scenario: Plain output format for show command

---

## Phase 1c: Configuration Specs

### Feature: Configuration (`config.feature`)
- [ ] Scenario: Config show displays current configuration
- [ ] Scenario: Uses default workspace when not specified
- [ ] Scenario: Workspace flag overrides default
- [ ] Scenario: Missing config file uses defaults
- [ ] Scenario: Invalid config file returns exit code 4

### Feature: Global Flags (`global_flags.feature`)
- [ ] Scenario: Quiet flag suppresses non-essential output
- [ ] Scenario: Verbose flag shows debug information
- [ ] Scenario: Format flag changes output format
- [ ] Scenario: Workspace flag selects workspace

---

## Phase 1d: Error Handling Specs

### Feature: Error Handling (`errors.feature`)
- [ ] Scenario: Network error returns exit code 1
- [ ] Scenario: Auth error returns exit code 1
- [ ] Scenario: Not found returns exit code 3
- [ ] Scenario: Config error returns exit code 4
- [ ] Scenario: Error message goes to stderr
- [ ] Scenario: JSON error format when --format=json

---

## Phase 2: Local Backend — Agent Coordination Specs

### Feature: Claiming Tasks (`claim.feature`)
- [ ] Scenario: Claim unclaimed task succeeds
- [ ] Scenario: Claim adds agent label to task
- [ ] Scenario: Claim moves task to in-progress
- [ ] Scenario: Claim assigns to authenticated user
- [ ] Scenario: Claim already-claimed task by same agent is no-op (exit 0)
- [ ] Scenario: Claim task claimed by different agent returns exit code 2
- [ ] Scenario: Claim with explicit agent-id flag
- [ ] Scenario: Claim uses BACKLOG_AGENT_ID environment variable
- [ ] Scenario: Claim uses workspace config agent_id
- [ ] Scenario: Claim uses global default agent_id
- [ ] Scenario: Claim falls back to hostname
- [ ] Scenario: Claim creates lock file (file mode)
- [ ] Scenario: Claim non-existent task returns exit code 3

### Feature: Releasing Tasks (`release.feature`)
- [ ] Scenario: Release claimed task succeeds
- [ ] Scenario: Release removes agent label
- [ ] Scenario: Release moves task to todo
- [ ] Scenario: Release unassigns user
- [ ] Scenario: Release removes lock file
- [ ] Scenario: Release with comment flag
- [ ] Scenario: Release task not claimed by this agent fails
- [ ] Scenario: Release unclaimed task fails

### Feature: Next Task (`next.feature`)
- [ ] Scenario: Next returns highest priority unclaimed task
- [ ] Scenario: Next with label filter
- [ ] Scenario: Next with --claim atomically claims
- [ ] Scenario: Next when no tasks available returns empty
- [ ] Scenario: Next skips tasks claimed by other agents
- [ ] Scenario: Next respects priority ordering (urgent > high > medium > low > none)
- [ ] Scenario: Next in id-only format
- [ ] Scenario: Next in JSON format

### Feature: Comments (`comment.feature`)
- [ ] Scenario: Add comment to task
- [ ] Scenario: Add comment with body-file
- [ ] Scenario: Comment appears in task file
- [ ] Scenario: Comment on non-existent task returns exit code 3

### Feature: File Locking (`locking.feature`)
- [ ] Scenario: Lock file contains agent and timestamps
- [ ] Scenario: Stale lock is detected and ignored
- [ ] Scenario: Lock TTL expiry allows reclaim
- [ ] Scenario: Concurrent claim attempts — one wins, one fails

### Step Definitions - Phase 2
- [ ] Implement "Given task {string} is claimed by agent {string}"
- [ ] Implement "Given the agent ID is {string}"
- [ ] Implement "Given the environment variable {string} is {string}"
- [ ] Implement "Then task {string} should be claimed by {string}"
- [ ] Implement "Then task {string} should not be claimed"
- [ ] Implement "Then a lock file should exist for {string}"
- [ ] Implement "Then no lock file should exist for {string}"

---

## Phase 3: Local Backend — Git Sync Specs

### Feature: Git Sync (`git_sync.feature`)
- [ ] Scenario: Mutation auto-commits when git_sync enabled
- [ ] Scenario: Commit message format is correct
- [ ] Scenario: Sync pulls and pushes
- [ ] Scenario: Sync --force overwrites local changes
- [ ] Scenario: Failed push returns exit code 2

### Feature: Git-Based Claims (`git_claim.feature`)
- [ ] Scenario: Claim with lock_mode git commits and pushes
- [ ] Scenario: Concurrent git claims — push conflict returns exit code 2
- [ ] Scenario: Release with git commits and pushes

### Step Definitions - Phase 3
- [ ] Implement "Given git_sync is enabled"
- [ ] Implement "Given lock_mode is {string}"
- [ ] Implement "Given a remote git repository"
- [ ] Implement "Then a git commit should exist with message containing {string}"
- [ ] Implement "Then the remote should have the commit"

---

## Phase 4: GitHub Backend — Issues Specs

### Feature: GitHub Connection (`github_connect.feature`)
- [ ] Scenario: Connect with valid token succeeds
- [ ] Scenario: Connect with invalid token returns exit code 1
- [ ] Scenario: Health check passes with valid connection
- [ ] Scenario: Uses GITHUB_TOKEN environment variable
- [ ] Scenario: Uses credentials.yaml token

### Feature: GitHub List (`github_list.feature`)
- [ ] Scenario: List fetches issues from repository
- [ ] Scenario: List maps issue labels to status
- [ ] Scenario: List filters by status via labels
- [ ] Scenario: List filters by assignee
- [ ] Scenario: List respects limit

### Feature: GitHub CRUD (`github_crud.feature`)
- [ ] Scenario: Add creates GitHub issue
- [ ] Scenario: Show fetches issue details
- [ ] Scenario: Edit updates issue fields
- [ ] Scenario: Move updates status labels

### Feature: GitHub Status Mapping (`github_status.feature`)
- [ ] Scenario: Default status mapping (labels)
- [ ] Scenario: Custom status_map in config
- [ ] Scenario: Unknown status maps to backlog with warning
- [ ] Scenario: Move to unmapped status fails

### Feature: GitHub Claims (`github_claim.feature`)
- [ ] Scenario: Claim adds agent label to issue
- [ ] Scenario: Claim assigns issue to authenticated user
- [ ] Scenario: Release removes agent label and unassigns
- [ ] Scenario: Claim already-claimed issue returns exit code 2

### Feature: GitHub Comments (`github_comments.feature`)
- [ ] Scenario: Comment adds issue comment via API
- [ ] Scenario: Show with --comments fetches comment thread

### Step Definitions - Phase 4
- [ ] Implement mock GitHub API server for testing
- [ ] Implement "Given a GitHub repository {string} with issues:"
- [ ] Implement "Given the GitHub token is {string}"
- [ ] Implement "Then the GitHub issue {string} should have label {string}"
- [ ] Implement "Then the GitHub issue {string} should be assigned to {string}"

---

## Phase 5: GitHub Backend — Projects Specs

### Feature: GitHub Projects (`github_projects.feature`)
- [ ] Scenario: Connect to repository with project
- [ ] Scenario: List shows tasks from project board
- [ ] Scenario: Move changes project column
- [ ] Scenario: Status read from project field

### Step Definitions - Phase 5
- [ ] Implement mock GraphQL API for Projects
- [ ] Implement "Given a GitHub project {int} with columns:"
- [ ] Implement "Then the project item should be in column {string}"

---

## Phase 6: Linear Backend Specs

### Feature: Linear Connection (`linear_connect.feature`)
- [ ] Scenario: Connect with valid API key succeeds
- [ ] Scenario: Connect with invalid API key returns exit code 1
- [ ] Scenario: Uses LINEAR_API_KEY environment variable

### Feature: Linear CRUD (`linear_crud.feature`)
- [ ] Scenario: List fetches issues from team
- [ ] Scenario: Add creates Linear issue
- [ ] Scenario: Show fetches issue details
- [ ] Scenario: Edit updates issue fields
- [ ] Scenario: Move changes issue state

### Feature: Linear Status Mapping (`linear_status.feature`)
- [ ] Scenario: Linear states map to canonical statuses
- [ ] Scenario: Custom state mapping in config

### Feature: Linear Claims (`linear_claim.feature`)
- [ ] Scenario: Claim adds agent label
- [ ] Scenario: Release removes agent label

### Step Definitions - Phase 6
- [ ] Implement mock Linear API for testing
- [ ] Implement "Given a Linear team {string} with issues:"
- [ ] Implement "Then the Linear issue {string} should have state {string}"

---

## Cross-Cutting Specs

### Feature: Multi-Backend (`multi_backend.feature`)
- [ ] Scenario: Switch between workspaces with different backends
- [ ] Scenario: Same command syntax works across backends
- [ ] Scenario: Output format consistent across backends

### Feature: Agent Workflow Integration (`agent_workflow.feature`)
- [ ] Scenario: Full agent workflow — next, claim, work, complete
- [ ] Scenario: Agent picks different task when claim fails
- [ ] Scenario: Agent releases task on failure
- [ ] Scenario: Multi-agent partitioning by labels

---

## CI/CD Integration

### Test Execution
- [ ] Add `make spec` target to run Gherkin specs
- [ ] Add `make spec-local` for local backend specs only
- [ ] Add `make spec-github` for GitHub backend specs (requires mock)
- [ ] Configure GitHub Actions to run specs on PR
- [ ] Generate test coverage report
- [ ] Generate HTML spec report for documentation

### Spec Documentation
- [ ] Add spec README explaining how to run tests
- [ ] Document how to add new scenarios
- [ ] Document step definition conventions
- [ ] Generate living documentation from features

---

## Progress Summary

| Phase | Status | Feature Files | Scenarios |
|-------|--------|---------------|-----------|
| Phase 0: Infrastructure | Complete | 0 | 0 |
| Phase 1: Local CRUD | In Progress | 3 | 17 |
| Phase 1b: Output Formats | Not Started | 0 | 0 |
| Phase 1c: Configuration | Not Started | 0 | 0 |
| Phase 1d: Error Handling | Not Started | 0 | 0 |
| Phase 2: Agent Coordination | Not Started | 0 | 0 |
| Phase 3: Git Sync | Not Started | 0 | 0 |
| Phase 4: GitHub Issues | Not Started | 0 | 0 |
| Phase 5: GitHub Projects | Not Started | 0 | 0 |
| Phase 6: Linear | Not Started | 0 | 0 |
| Cross-Cutting | Not Started | 0 | 0 |
