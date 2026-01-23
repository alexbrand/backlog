# backlog CLI - Executable Specification Tasks

## Overview

This task list tracks the creation of a Gherkin-based executable specification using [godog](https://github.com/cucumber/godog). The goal is to build a complete spec that defines the expected behavior of the backlog CLI.

Once the spec is in place, it will serve as the foundation for TDD-based implementation: write the feature scenarios first, then implement the code to make them pass. This "spec-first" approach ensures that implementation is driven by clearly defined, testable requirements rather than ad-hoc development.

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
- [x] Scenario: List with status filter
- [x] Scenario: List with multiple status values
- [x] Scenario: List with priority filter
- [x] Scenario: List with label filter
- [x] Scenario: List with assignee filter
- [x] Scenario: List unassigned tasks
- [x] Scenario: List with limit
- [x] Scenario: List returns empty when no tasks match
- [x] Scenario: List shows hasMore flag when more tasks exist

### Feature: Showing Tasks (`show.feature`)
- [x] Scenario: Show task displays all fields
- [x] Scenario: Show task in JSON format
- [x] Scenario: Show task with comments
- [x] Scenario: Show non-existent task returns exit code 3
- [x] Scenario: Show task displays correct status directory

### Feature: Moving Tasks (`move.feature`)
- [x] Scenario Outline: Move task to each valid status
- [x] Scenario: Move task updates file location
- [x] Scenario: Move task updates frontmatter
- [x] Scenario: Move to invalid status fails
- [x] Scenario: Move non-existent task returns exit code 3
- [x] Scenario: Move task with comment flag

### Feature: Editing Tasks (`edit.feature`)
- [x] Scenario: Edit task title
- [x] Scenario: Edit task priority
- [x] Scenario: Edit task description
- [x] Scenario: Add label to task
- [x] Scenario: Remove label from task
- [x] Scenario: Edit multiple fields at once
- [x] Scenario: Edit non-existent task returns exit code 3
- [x] Scenario: Edit preserves unmodified fields

### Step Definitions - Phase 1
- [x] Implement "Given a task {string} exists with status {string}"
- [x] Implement "Given a task {string} exists with priority {string}"
- [x] Implement "Given a task {string} exists with labels {string}"
- [x] Implement "Then a task file should exist in {string} directory"
- [x] Implement "Then the task {string} should have status {string}"
- [x] Implement "Then the created task should have priority {string}"
- [x] Implement "Then the created task should have label {string}"
- [x] Implement "Then the task {string} should not have label {string}"
- [x] Implement "Then the task count should be {int}"
- [x] Implement "Then stdout should match pattern {string}"
- [x] Implement "Then the created task should have description containing {string}"
- [x] Implement "Then the task {string} should have description containing {string}"

---

## Phase 1b: Output Format Specs

### Feature: Table Output (`output_table.feature`)
- [x] Scenario: Table output has correct headers
- [x] Scenario: Table output aligns columns
- [x] Scenario: Table output truncates long titles
- [x] Scenario: Table output shows dash for empty fields

### Feature: JSON Output (`output_json.feature`)
- [x] Scenario: JSON output is valid JSON
- [x] Scenario: JSON output includes all task fields
- [x] Scenario: JSON output includes count and hasMore
- [x] Scenario: JSON error output format

### Feature: Plain Output (`output_plain.feature`)
- [x] Scenario: Plain output shows one task per line
- [x] Scenario: Plain output format for show command

---

## Phase 1c: Configuration Specs

### Feature: Configuration (`config.feature`)
- [x] Scenario: Config show displays current configuration
- [x] Scenario: Uses default workspace when not specified
- [x] Scenario: Workspace flag overrides default
- [x] Scenario: Missing config file uses defaults
- [x] Scenario: Invalid config file returns exit code 4

### Feature: Global Flags (`global_flags.feature`)
- [x] Scenario: Quiet flag suppresses non-essential output
- [x] Scenario: Verbose flag shows debug information
- [x] Scenario: Format flag changes output format
- [x] Scenario: Workspace flag selects workspace

---

## Phase 1d: Error Handling Specs

### Feature: Error Handling (`errors.feature`)
- [x] Scenario: Network error returns exit code 1 (documented for remote backends with @remote tag)
- [x] Scenario: Auth error returns exit code 1 (documented for remote backends with @remote tag)
- [x] Scenario: Not found returns exit code 3
- [x] Scenario: Config error returns exit code 4
- [x] Scenario: Error message goes to stderr
- [x] Scenario: JSON error format when --format=json

---

## Phase 2: Local Backend — Agent Coordination Specs

### Feature: Claiming Tasks (`claim.feature`)
- [x] Scenario: Claim unclaimed task succeeds
- [x] Scenario: Claim adds agent label to task
- [x] Scenario: Claim moves task to in-progress
- [x] Scenario: Claim assigns to authenticated user
- [x] Scenario: Claim already-claimed task by same agent is no-op (exit 0)
- [x] Scenario: Claim task claimed by different agent returns exit code 2
- [x] Scenario: Claim with explicit agent-id flag
- [x] Scenario: Claim uses BACKLOG_AGENT_ID environment variable
- [x] Scenario: Claim uses workspace config agent_id
- [x] Scenario: Claim uses global default agent_id
- [x] Scenario: Claim falls back to hostname
- [x] Scenario: Claim creates lock file (file mode)
- [x] Scenario: Claim non-existent task returns exit code 3

### Feature: Releasing Tasks (`release.feature`)
- [x] Scenario: Release claimed task succeeds
- [x] Scenario: Release removes agent label
- [x] Scenario: Release moves task to todo
- [x] Scenario: Release unassigns user
- [x] Scenario: Release removes lock file
- [x] Scenario: Release with comment flag
- [x] Scenario: Release task not claimed by this agent fails
- [x] Scenario: Release unclaimed task fails

### Feature: Next Task (`next.feature`)
- [x] Scenario: Next returns highest priority unclaimed task
- [x] Scenario: Next with label filter
- [x] Scenario: Next with --claim atomically claims
- [x] Scenario: Next when no tasks available returns empty
- [x] Scenario: Next skips tasks claimed by other agents
- [x] Scenario: Next respects priority ordering (urgent > high > medium > low > none)
- [x] Scenario: Next in id-only format
- [x] Scenario: Next in JSON format

### Feature: Comments (`comment.feature`)
- [x] Scenario: Add comment to task
- [x] Scenario: Add comment with body-file
- [x] Scenario: Comment appears in task file
- [x] Scenario: Comment on non-existent task returns exit code 3

### Feature: File Locking (`locking.feature`)
- [x] Scenario: Lock file contains agent and timestamps
- [x] Scenario: Stale lock is detected and ignored
- [x] Scenario: Lock TTL expiry allows reclaim
- [x] Scenario: Concurrent claim attempts — one wins, one fails

### Step Definitions - Phase 2
- [x] Implement "Given task {string} is claimed by agent {string}"
- [x] Implement "Given the agent ID is {string}"
- [x] Implement "Given the environment variable {string} is {string}"
- [x] Implement "Then task {string} should be claimed by {string}"
- [x] Implement "Then task {string} should not be claimed"
- [x] Implement "Then a lock file should exist for {string}"
- [x] Implement "Then no lock file should exist for {string}"

---

## Phase 3: Local Backend — Git Sync Specs

### Feature: Git Sync (`git_sync.feature`)
- [x] Scenario: Mutation auto-commits when git_sync enabled
- [x] Scenario: Commit message format is correct
- [x] Scenario: Sync pulls and pushes
- [x] Scenario: Sync --force overwrites local changes
- [x] Scenario: Failed push returns exit code 2

### Feature: Git-Based Claims (`git_claim.feature`)
- [x] Scenario: Claim with lock_mode git commits and pushes
- [x] Scenario: Concurrent git claims — push conflict returns exit code 2
- [x] Scenario: Release with git commits and pushes

### Step Definitions - Phase 3
- [x] Implement "Given git_sync is enabled"
- [x] Implement "Given lock_mode is {string}"
- [x] Implement "Given a remote git repository"
- [x] Implement "Then a git commit should exist with message containing {string}"
- [x] Implement "Then the remote should have the commit"

---

## Phase 4: GitHub Backend — Issues Specs

### Feature: GitHub Connection (`github_connect.feature`)
- [x] Scenario: Connect with valid token succeeds
- [x] Scenario: Connect with invalid token returns exit code 1
- [x] Scenario: Health check passes with valid connection
- [x] Scenario: Uses GITHUB_TOKEN environment variable
- [x] Scenario: Uses credentials.yaml token

### Feature: GitHub List (`github_list.feature`)
- [x] Scenario: List fetches issues from repository
- [x] Scenario: List maps issue labels to status
- [x] Scenario: List filters by status via labels
- [x] Scenario: List filters by assignee
- [x] Scenario: List respects limit

### Feature: GitHub CRUD (`github_crud.feature`)
- [x] Scenario: Add creates GitHub issue
- [x] Scenario: Show fetches issue details
- [x] Scenario: Edit updates issue fields
- [x] Scenario: Move updates status labels

### Feature: GitHub Status Mapping (`github_status.feature`)
- [x] Scenario: Default status mapping (labels)
- [x] Scenario: Custom status_map in config
- [x] Scenario: Unknown status maps to backlog with warning
- [x] Scenario: Move to unmapped status fails

### Feature: GitHub Claims (`github_claim.feature`)
- [x] Scenario: Claim adds agent label to issue
- [x] Scenario: Claim assigns issue to authenticated user
- [x] Scenario: Release removes agent label and unassigns
- [x] Scenario: Claim already-claimed issue returns exit code 2

### Feature: GitHub Comments (`github_comments.feature`)
- [x] Scenario: Comment adds issue comment via API
- [x] Scenario: Show with --comments fetches comment thread

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
| Phase 1: Local CRUD | Complete | 5 | 36 |
| Phase 1b: Output Formats | Complete | 3 | 8 |
| Phase 1c: Configuration | Complete | 2 | 16 |
| Phase 1d: Error Handling | Complete | 1 | 12 |
| Phase 2: Agent Coordination | Complete | 5 | 46 |
| Phase 3: Git Sync | Complete | 2 | 19 |
| Phase 4: GitHub Issues | In Progress | 6 | 73 |
| Phase 5: GitHub Projects | Not Started | 0 | 0 |
| Phase 6: Linear | Not Started | 0 | 0 |
| Cross-Cutting | Not Started | 0 | 0 |
