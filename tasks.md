# backlog CLI - Implementation Tasks

## Phase 1: Local Backend — Basic CRUD

### Project Setup
- [x] Initialize Go module (`go mod init`)
- [x] Create directory structure (`cmd/backlog/`, `internal/`)
- [x] Set up Cobra CLI framework
- [x] Set up Viper for configuration

### Core Types & Interfaces
- [x] Define `Task` struct with all fields (id, title, description, status, priority, etc.)
- [x] Define `Backend` interface in `internal/backend/backend.go`
- [x] Define `TaskFilters`, `TaskInput`, `TaskChanges` types
- [x] Define status enum (`backlog`, `todo`, `in-progress`, `review`, `done`)
- [x] Define priority enum (`urgent`, `high`, `medium`, `low`, `none`)
- [x] Create backend registry in `internal/backend/registry.go`

### Output Formatters
- [x] Create formatter interface in `internal/output/`
- [x] Implement `table` formatter (default)
- [x] Implement `json` formatter
- [x] Implement `plain` formatter
- [x] Implement `id-only` formatter

### Local Backend Implementation
- [x] Create `internal/local/` package
- [x] Implement task file parsing (YAML frontmatter + markdown body)
- [x] Implement task file writing
- [x] Implement `Name()` and `Version()` methods
- [x] Implement `Connect()` — validate/create `.backlog/` structure
- [x] Implement `List()` — scan all status directories
- [x] Implement `Get()` — read single task file
- [x] Implement `Create()` — generate ID, write file to `backlog/`
- [x] Implement `Update()` — modify frontmatter/body
- [x] Implement `Delete()` — remove task file
- [x] Implement `Move()` — move file between status directories

### CLI Commands - Phase 1
- [x] Implement `backlog init` — create `.backlog/` directory structure
- [x] Implement `backlog add <title>` with flags (`--priority`, `--label`, `--description`, `--body-file`, `--status`)
- [x] Implement `backlog list` with filters (`--status`, `--assignee`, `--priority`, `--label`, `--limit`)
- [x] Implement `backlog show <id>` with `--comments` flag
- [x] Implement `backlog move <id> <status>`
- [x] Implement `backlog edit <id>` with flags (`--title`, `--priority`, `--add-label`, `--remove-label`, `--description`)

### Global Flags
- [x] Implement `--workspace` / `-w` flag
- [x] Implement `--format` / `-f` flag
- [x] Implement `--quiet` / `-q` flag
- [x] Implement `--verbose` / `-v` flag

### Configuration
- [x] Implement config file loading (`~/.config/backlog/config.yaml` and `.backlog/config.yaml`)
- [x] Implement workspace selection logic
- [x] Implement `backlog config show`

### Error Handling
- [x] Define exit codes (0=success, 1=error, 2=conflict, 3=not found, 4=config error)
- [x] Implement consistent error output format (stderr)
- [x] Implement JSON error output when `--format=json`

---

## Phase 2: Local Backend — Agent Coordination

### Agent Identity
- [x] Implement `--agent-id` CLI flag
- [x] Implement `BACKLOG_AGENT_ID` environment variable support
- [x] Implement agent ID resolution chain (flag → env → workspace config → global default → hostname)
- [x] Add `agent_id` and `agent_label_prefix` to workspace config (already in config structure)

### File-Based Locking
- [x] Create `.locks/` directory structure (created by `backlog init`)
- [x] Implement lock file format (agent, claimed_at, expires_at) — `internal/local/lock.go`
- [x] Implement lock acquisition (atomic file creation) — `Claim()` method
- [x] Implement lock release — `Release()` method
- [x] Implement stale lock detection (TTL expiry) — `isActive()` method with 30-min default TTL
- [x] Add `lock_mode: file` config option

### CLI Commands - Phase 2
- [x] Implement `backlog claim <id>` — acquire lock, add agent label to frontmatter, move to `in-progress`
- [x] Implement `backlog release <id>` — release lock, remove agent label, move to `todo`
- [x] Implement `backlog next` — find highest priority unclaimed task
- [x] Implement `backlog next --claim` — atomic claim
- [x] Implement `backlog next --label=<label>` — filtered next

### Conflict Handling
- [x] Return exit code 2 when task already claimed by another agent — `ClaimConflictError` type
- [x] Handle "already claimed by same agent" as success (no-op) — `AlreadyOwned` flag in `ClaimResult`

### Comments
- [x] Implement `backlog comment <id> <message>`
- [x] Implement `--body-file` flag for comments
- [x] Implement `--comment` flag for `move` and `release` commands

---

## Phase 3: Local Backend — Git Sync

### Git Integration
- [x] Add `lock_mode: git` config option
- [x] Add `git_sync: true` config option
- [x] Implement auto-commit on mutations (add, edit, move, claim, release, comment)
- [x] Implement commit message format (`action: id-title [agent:id]`)

### Sync Command
- [x] Implement `backlog sync` — pull, push
- [x] Implement `backlog sync --force`
- [x] Implement conflict detection via failed push (exit code 2)

### Git-Based Claims
- [x] Implement claim via pull → commit → push
- [x] Handle push failure as conflict

---

## Phase 4: GitHub Backend — Issues

### GitHub Backend Implementation
- [x] Create `internal/github/` package
- [x] Implement GitHub API client setup (go-github)
- [x] Implement `Connect()` with token authentication
- [x] Implement `HealthCheck()`
- [x] Implement `List()` — fetch issues with label filtering
- [x] Implement `Get()` — fetch single issue
- [x] Implement `Create()` — create issue
- [x] Implement `Update()` — update issue
- [x] Implement `Delete()` — close/delete issue
- [x] Implement `Move()` — update labels for status

### Status Mapping
- [x] Implement default label-based status mapping
- [x] Implement configurable `status_map` in workspace config
- [x] Handle unknown statuses (map to `backlog` with warning)

### Agent Labels
- [x] Implement `agent:<agent_id>` label management
- [x] Implement claim via label + assignment
- [x] Implement release via label removal + unassignment

### Comments
- [x] Implement `ListComments()`
- [x] Implement `AddComment()`

### CLI Integration
- [x] Integrate GitHub backend into CLI commands via `backend_helper.go`

### Configuration
- [x] Implement `backlog config init` — interactive setup
- [x] Support `GITHUB_TOKEN` environment variable
- [x] Support credentials.yaml for token storage

---

## Phase 5: GitHub Backend — Projects

### GitHub Projects v2 Integration
- [x] Implement GraphQL API client for Projects
- [x] Implement project field discovery
- [x] Implement column-based status (instead of labels)
- [x] Add `project` config option (project number)
- [x] Add `status_field` config option

### Status via Columns
- [x] Implement `Move()` via project column change
- [x] Implement status reading from project item

---

## Phase 6: Linear Backend

### Linear Backend Implementation
- [x] Create `internal/linear/` package
- [x] Implement Linear API client
- [x] Implement `Connect()` with API key authentication
- [x] Implement `HealthCheck()`
- [x] Implement `List()` — fetch issues
- [x] Implement `Get()` — fetch single issue
- [x] Implement `Create()` — create issue
- [x] Implement `Update()` — update issue
- [x] Implement `Move()` — change state

### Status Mapping
- [x] Map Linear states to canonical statuses
- [x] Implement configurable mapping

### Team & Project Support
- [x] Add `team` config option
- [x] Implement team filtering

### Agent Labels
- [x] Implement agent labels via Linear labels

---

## Cross-Cutting Concerns

### Testing
- [x] Unit tests for task file parsing
- [x] Unit tests for Linear backend
- [x] Unit tests for GitHub backend
- [x] Integration tests for CLI commands
- [ ] Test multi-agent claim scenarios

### Documentation
- [ ] Write README.md with installation and usage
- [ ] Document configuration options
- [ ] Add examples for agent integration patterns

### Build & Distribution
- [ ] Set up cross-compilation (darwin/arm64, linux/amd64, windows/amd64)
- [ ] Create GitHub Actions workflow for releases
- [ ] Set up Homebrew tap
- [ ] Publish to `go install`

---

## Progress Summary

| Phase | Status | Tasks |
|-------|--------|-------|
| Phase 1: Local CRUD | Complete | All done |
| Phase 2: Agent Coordination | Complete | All done |
| Phase 3: Git Sync | Complete | All done |
| Phase 4: GitHub Issues | Complete | All done |
| Phase 5: GitHub Projects | Complete | All done |
| Phase 6: Linear | Complete | All done |
