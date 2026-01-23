# backlog CLI - Implementation Tasks

## Phase 1: Local Backend — Basic CRUD

### Project Setup
- [x] Initialize Go module (`go mod init`)
- [x] Create directory structure (`cmd/backlog/`, `internal/`)
- [x] Set up Cobra CLI framework
- [x] Set up Viper for configuration

### Core Types & Interfaces
- [ ] Define `Task` struct with all fields (id, title, description, status, priority, etc.)
- [ ] Define `Backend` interface in `internal/backend/backend.go`
- [ ] Define `TaskFilters`, `TaskInput`, `TaskChanges` types
- [ ] Define status enum (`backlog`, `todo`, `in-progress`, `review`, `done`)
- [ ] Define priority enum (`urgent`, `high`, `medium`, `low`, `none`)
- [ ] Create backend registry in `internal/backend/registry.go`

### Output Formatters
- [ ] Create formatter interface in `internal/output/`
- [ ] Implement `table` formatter (default)
- [ ] Implement `json` formatter
- [ ] Implement `plain` formatter
- [ ] Implement `id-only` formatter

### Local Backend Implementation
- [ ] Create `internal/local/` package
- [ ] Implement task file parsing (YAML frontmatter + markdown body)
- [ ] Implement task file writing
- [ ] Implement `Name()` and `Version()` methods
- [ ] Implement `Connect()` — validate/create `.backlog/` structure
- [ ] Implement `List()` — scan all status directories
- [ ] Implement `Get()` — read single task file
- [ ] Implement `Create()` — generate ID, write file to `backlog/`
- [ ] Implement `Update()` — modify frontmatter/body
- [ ] Implement `Delete()` — remove task file
- [ ] Implement `Move()` — move file between status directories

### CLI Commands - Phase 1
- [ ] Implement `backlog init` — create `.backlog/` directory structure
- [ ] Implement `backlog add <title>` with flags (`--priority`, `--label`, `--description`, `--body-file`, `--status`)
- [ ] Implement `backlog list` with filters (`--status`, `--assignee`, `--priority`, `--label`, `--limit`)
- [ ] Implement `backlog show <id>` with `--comments` flag
- [ ] Implement `backlog move <id> <status>`
- [ ] Implement `backlog edit <id>` with flags (`--title`, `--priority`, `--add-label`, `--remove-label`, `--description`)

### Global Flags
- [x] Implement `--workspace` / `-w` flag
- [x] Implement `--format` / `-f` flag
- [x] Implement `--quiet` / `-q` flag
- [x] Implement `--verbose` / `-v` flag

### Configuration
- [ ] Implement config file loading (`~/.config/backlog/config.yaml`)
- [ ] Implement workspace selection logic
- [ ] Implement `backlog config show`

### Error Handling
- [ ] Define exit codes (0=success, 1=error, 2=conflict, 3=not found, 4=config error)
- [ ] Implement consistent error output format (stderr)
- [ ] Implement JSON error output when `--format=json`

---

## Phase 2: Local Backend — Agent Coordination

### Agent Identity
- [ ] Implement `--agent-id` CLI flag
- [ ] Implement `BACKLOG_AGENT_ID` environment variable support
- [ ] Implement agent ID resolution chain (flag → env → workspace config → global default → hostname)
- [ ] Add `agent_id` and `agent_label_prefix` to workspace config

### File-Based Locking
- [ ] Create `.locks/` directory structure
- [ ] Implement lock file format (agent, claimed_at, expires_at)
- [ ] Implement lock acquisition (atomic file creation)
- [ ] Implement lock release
- [ ] Implement stale lock detection (TTL expiry)
- [ ] Add `lock_mode: file` config option

### CLI Commands - Phase 2
- [ ] Implement `backlog claim <id>` — acquire lock, add agent label to frontmatter, move to `in-progress`
- [ ] Implement `backlog release <id>` — release lock, remove agent label, move to `todo`
- [ ] Implement `backlog next` — find highest priority unclaimed task
- [ ] Implement `backlog next --claim` — atomic claim
- [ ] Implement `backlog next --label=<label>` — filtered next

### Conflict Handling
- [ ] Return exit code 2 when task already claimed by another agent
- [ ] Handle "already claimed by same agent" as success (no-op)

### Comments
- [ ] Implement `backlog comment <id> <message>`
- [ ] Implement `--body-file` flag for comments
- [ ] Implement `--comment` flag for `move` and `release` commands

---

## Phase 3: Local Backend — Git Sync

### Git Integration
- [ ] Add `lock_mode: git` config option
- [ ] Add `git_sync: true` config option
- [ ] Implement auto-commit on mutations (add, edit, move, claim, release)
- [ ] Implement commit message format (`action: id-title [agent:id]`)

### Sync Command
- [ ] Implement `backlog sync` — pull, push
- [ ] Implement `backlog sync --force`
- [ ] Implement conflict detection via failed push (exit code 2)

### Git-Based Claims
- [ ] Implement claim via pull → commit → push
- [ ] Handle push failure as conflict

---

## Phase 4: GitHub Backend — Issues

### GitHub Backend Implementation
- [ ] Create `internal/github/` package
- [ ] Implement GitHub API client setup (go-github)
- [ ] Implement `Connect()` with token authentication
- [ ] Implement `HealthCheck()`
- [ ] Implement `List()` — fetch issues with label filtering
- [ ] Implement `Get()` — fetch single issue
- [ ] Implement `Create()` — create issue
- [ ] Implement `Update()` — update issue
- [ ] Implement `Delete()` — close/delete issue
- [ ] Implement `Move()` — update labels for status

### Status Mapping
- [ ] Implement default label-based status mapping
- [ ] Implement configurable `status_map` in workspace config
- [ ] Handle unknown statuses (map to `backlog` with warning)

### Agent Labels
- [ ] Implement `agent:<agent_id>` label management
- [ ] Implement claim via label + assignment
- [ ] Implement release via label removal + unassignment

### Comments
- [ ] Implement `ListComments()`
- [ ] Implement `AddComment()`

### Configuration
- [ ] Implement `backlog config init` — interactive setup
- [ ] Support `GITHUB_TOKEN` environment variable
- [ ] Support credentials.yaml for token storage

---

## Phase 5: GitHub Backend — Projects

### GitHub Projects v2 Integration
- [ ] Implement GraphQL API client for Projects
- [ ] Implement project field discovery
- [ ] Implement column-based status (instead of labels)
- [ ] Add `project` config option (project number)
- [ ] Add `status_field` config option

### Status via Columns
- [ ] Implement `Move()` via project column change
- [ ] Implement status reading from project item

---

## Phase 6: Linear Backend

### Linear Backend Implementation
- [ ] Create `internal/linear/` package
- [ ] Implement Linear API client
- [ ] Implement `Connect()` with API key authentication
- [ ] Implement `HealthCheck()`
- [ ] Implement `List()` — fetch issues
- [ ] Implement `Get()` — fetch single issue
- [ ] Implement `Create()` — create issue
- [ ] Implement `Update()` — update issue
- [ ] Implement `Move()` — change state

### Status Mapping
- [ ] Map Linear states to canonical statuses
- [ ] Implement configurable mapping

### Team & Project Support
- [ ] Add `team` config option
- [ ] Implement team filtering

### Agent Labels
- [ ] Implement agent labels via Linear labels

---

## Cross-Cutting Concerns

### Testing
- [ ] Unit tests for task file parsing
- [ ] Unit tests for each backend
- [ ] Integration tests for CLI commands
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
| Phase 1: Local CRUD | In Progress | 7/X |
| Phase 2: Agent Coordination | Not Started | 0/X |
| Phase 3: Git Sync | Not Started | 0/X |
| Phase 4: GitHub Issues | Not Started | 0/X |
| Phase 5: GitHub Projects | Not Started | 0/X |
| Phase 6: Linear | Not Started | 0/X |
