# Product Requirements Document: `backlog` CLI

**Version:** 0.1  
**Date:** January 2025  
**Author:** Alex Brand  

---

## Overview

`backlog` is a command-line tool for managing tasks across multiple issue tracking backends. It provides a unified, agent-friendly interface that abstracts away provider-specific APIs, enabling both humans and AI agents to manage backlogs through simple, composable commands.

---

## Problem Statement

AI agents working on codebases need to interact with issue trackers, but current options present friction:

- **GitHub Projects**: Limited CLI support; moving cards requires raw GraphQL
- **Linear**: Good API, but no widely-adopted CLI
- **Provider lock-in**: Switching backends means learning new tools and APIs

Additionally, teams with multiple agents need coordination primitives (claiming, locking) that most trackers don't natively support.

---

## Goals

1. **Unified interface**: One CLI that works identically across GitHub, Linear, and other backends
2. **Agent-first design**: Predictable output formats (JSON, plain text), atomic operations, clear exit codes
3. **Human-friendly**: Intuitive commands, sensible defaults, good DX for manual use
4. **Extensibility**: Plugin architecture for adding new backends without modifying core
5. **Coordination support**: Built-in primitives for multi-agent workflows (claim, release, lock)

---

## Non-Goals

- Full feature parity with every backend's web UI
- Real-time sync or webhooks (out of scope for v1)
- GUI or TUI—this is CLI-only
- Project/board creation and admin (use native tools)

---

## User Personas

### Agent (Primary)

An AI coding agent (Claude Code, Cursor, Aider, etc.) that needs to:

- Fetch the next task to work on
- Update task status as work progresses  
- Add notes or comments to tasks
- Claim tasks to prevent conflicts with other agents

### Developer (Secondary)

A human developer who prefers CLI workflows and wants to:

- Quickly check what's in the backlog
- Create tasks without leaving the terminal
- Triage and prioritize from the command line

---

## Core Concepts

### Task

A work item with:

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | Unique identifier (backend-specific format) |
| `title` | string | Short summary |
| `description` | string | Full description (markdown) |
| `status` | enum | `backlog`, `todo`, `in-progress`, `review`, `done` |
| `priority` | enum | `urgent`, `high`, `medium`, `low`, `none` |
| `assignee` | string? | Username or agent ID |
| `labels` | string[] | Tags/labels |
| `created` | datetime | Creation timestamp |
| `updated` | datetime | Last modified timestamp |
| `url` | string | Web URL to view in browser |
| `meta` | object | Backend-specific fields |

### Backend

A provider adapter that implements the `Backend` interface. Ships with:

- `github` — GitHub Issues (with optional Projects integration)
- `linear` — Linear API
- `local` — Filesystem/git-based (for offline or custom workflows)

### Workspace

A configured connection to a specific backend instance:

```yaml
# ~/.config/backlog/config.yaml
workspaces:
  main:
    backend: github
    repo: alexbrand/myproject
    default: true
  
  work:
    backend: linear
    team: ENG
    
  offline:
    backend: local
    path: ./.backlog
```

---

## Command Reference

### Global Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--workspace` | `-w` | Target workspace (default: `default: true` workspace) |
| `--format` | `-f` | Output format: `table`, `json`, `plain`, `id-only` |
| `--quiet` | `-q` | Suppress non-essential output |
| `--verbose` | `-v` | Show debug information |

### Commands

#### `backlog list`

List tasks with optional filtering.

```bash
backlog list                          # all non-done tasks
backlog list --status=todo            # filter by status
backlog list --assignee=@me           # my tasks
backlog list --assignee=unassigned    # unclaimed tasks
backlog list --priority=high,urgent   # multiple values
backlog list --label=bug              # by label
backlog list --limit=10               # pagination
backlog list -f json                  # JSON output for agents
```

**Output (table, default):**

```
ID       STATUS       PRIORITY  TITLE                        ASSIGNEE
GH-123   in-progress  high      Implement auth flow          @alex
GH-124   todo         medium    Add rate limiting            —
GH-125   backlog      low       Update docs                  —
```

**Output (json):**

```json
{
  "tasks": [
    {
      "id": "GH-123",
      "title": "Implement auth flow",
      "status": "in-progress",
      "priority": "high",
      "assignee": "alex",
      "labels": ["feature", "auth"],
      "url": "https://github.com/..."
    }
  ],
  "count": 3,
  "hasMore": false
}
```

**Exit codes:**

- `0` — Success
- `1` — Error (auth, network, etc.)

---

#### `backlog show <id>`

Display full task details.

```bash
backlog show GH-123
backlog show GH-123 -f json
backlog show GH-123 --comments      # include comment thread
```

**Output:**

```
GH-123: Implement auth flow
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Status:    in-progress
Priority:  high
Assignee:  @alex
Labels:    feature, auth
Created:   2025-01-15 09:00
Updated:   2025-01-18 14:30
URL:       https://github.com/alexbrand/myproject/issues/123

## Description

Implement OAuth2 authentication flow with support for:
- Google
- GitHub
- Email/password fallback

## Acceptance Criteria

- [ ] Login endpoint works
- [ ] Token refresh implemented
- [ ] Tests passing
```

---

#### `backlog add <title>`

Create a new task.

```bash
backlog add "Implement rate limiting"
backlog add "Fix login bug" --priority=urgent --label=bug
backlog add "Refactor API" --description="Split into modules" --status=todo
backlog add "Research caching" --body-file=./task-details.md
```

**Output:**

```
Created GH-126: Implement rate limiting
```

**Output (json):**

```json
{
  "id": "GH-126",
  "title": "Implement rate limiting",
  "url": "https://github.com/..."
}
```

---

#### `backlog edit <id>`

Modify task fields.

```bash
backlog edit GH-123 --title="New title"
backlog edit GH-123 --priority=urgent
backlog edit GH-123 --add-label=blocked --remove-label=ready
backlog edit GH-123 --description="Updated description"
```

---

#### `backlog move <id> <status>`

Transition task to a new status.

```bash
backlog move GH-123 in-progress
backlog move GH-123 done
backlog move GH-123 review --comment="Ready for review"
```

**Status values:** `backlog`, `todo`, `in-progress`, `review`, `done`

---

#### `backlog claim <id>`

Claim a task for the current agent. Atomic operation for multi-agent coordination.

```bash
backlog claim GH-124
backlog claim GH-124 --agent-id=claude-2   # override agent ID
```

**Behavior:**

1. Check if task already has an `agent:*` label
2. If unclaimed:
   - Assign to authenticated user (API credential owner)
   - Add label `agent:<agent_id>` (e.g., `agent:claude-1`)
   - Move to `in-progress`
   - Return success
3. If claimed by same agent: no-op, return success
4. If claimed by different agent: return error (exit code 2)

**Exit codes:**

- `0` — Claimed successfully (or already owned by this agent)
- `1` — Error (network, auth, etc.)
- `2` — Task already claimed by another agent

This enables optimistic concurrency:

```bash
# Agent workflow
if backlog claim GH-124; then
  # work on task
  backlog move GH-124 done
else
  # pick a different task
  backlog list --status=todo --assignee=unassigned -f id-only | head -1
fi
```

---

#### `backlog release <id>`

Release a claimed task back to `todo`. Used when an agent can't complete work.

```bash
backlog release GH-124
backlog release GH-124 --comment="Blocked on external API"
```

**Behavior:**

1. Remove the `agent:*` label
2. Unassign from authenticated user
3. Move to `todo`

---

#### `backlog comment <id> <message>`

Add a comment to a task.

```bash
backlog comment GH-123 "Found the bug, working on fix"
backlog comment GH-123 --body-file=./analysis.md
```

---

#### `backlog next`

Get the next recommended task to work on. Useful for agents.

```bash
backlog next                          # highest priority unassigned task
backlog next --label=backend          # filtered
backlog next --claim                  # atomically claim it too
```

**Output (plain):**

```
GH-124
```

**Output (json):**

```json
{
  "id": "GH-124",
  "title": "Add rate limiting",
  "priority": "high",
  "url": "https://github.com/..."
}
```

---

#### `backlog sync`

Sync local cache with remote (for backends that support offline).

```bash
backlog sync
backlog sync --force
```

---

#### `backlog config`

Manage configuration.

```bash
backlog config init                   # interactive setup
backlog config show                   # display current config
backlog config set default-workspace work
backlog config add-workspace          # add new workspace interactively
```

---

## Backend Interface

New backends implement this interface:

```go
package backend

type Backend interface {
    // Identification
    Name() string
    Version() string

    // Connection
    Connect(cfg Config) error
    Disconnect() error
    HealthCheck() (HealthStatus, error)

    // Core operations
    List(filters TaskFilters) (*TaskList, error)
    Get(id string) (*Task, error)
    Create(input TaskInput) (*Task, error)
    Update(id string, changes TaskChanges) (*Task, error)
    Delete(id string) error

    // Status transitions
    Move(id string, status Status) (*Task, error)

    // Assignment
    Assign(id string, assignee string) (*Task, error)
    Unassign(id string) (*Task, error)

    // Comments
    ListComments(id string) ([]Comment, error)
    AddComment(id string, body string) (*Comment, error)
}

// Optional interfaces - backends implement if supported
type Claimer interface {
    Claim(id string, agentID string) (*ClaimResult, error)
    Release(id string) error
}

type Syncer interface {
    Sync() (*SyncResult, error)
}
```

Backends are loaded as compiled plugins or built-in:

1. Built-in: `github`, `linear`, `local`
2. External plugins: Go plugins in `~/.config/backlog/plugins/`

---

## Configuration

### File Location

- Config: `~/.config/backlog/config.yaml`
- Credentials: `~/.config/backlog/credentials.yaml` (or system keychain)
- Cache: `~/.cache/backlog/`

### Config Schema

```yaml
version: 1

defaults:
  format: table
  workspace: main
  agent_id: claude-1              # global default agent ID

workspaces:
  main:
    backend: github
    repo: alexbrand/myproject
    project: 1                    # optional: GitHub Project number
    status_field: Status          # project field name for status
    agent_id: claude-main         # overrides global for this workspace
    agent_label_prefix: agent     # creates "agent:claude-main" labels
    default: true
    
  work:
    backend: linear
    team: ENG
    api_key_env: LINEAR_API_KEY   # or use credentials.yaml
    agent_label_prefix: agent
    # inherits agent_id from defaults
    
  offline:
    backend: local
    path: ./.backlog
    lock_mode: file           # file (default) or git
    git_sync: true            # auto-commit on changes (required for lock_mode: git)
```

### Agent Identity Resolution

Agent ID is resolved in priority order:

1. **CLI flag**: `--agent-id=claude-1`
2. **Environment variable**: `BACKLOG_AGENT_ID`
3. **Workspace config**: `workspaces.<name>.agent_id`
4. **Global default**: `defaults.agent_id`
5. **Fallback**: hostname

Example for containerized agents:

```bash
export BACKLOG_AGENT_ID=claude-worker-$(hostname)
backlog claim GH-123   # uses env var
```

### Claim Behavior

When `backlog claim <id>` is called:

1. Task is assigned to the authenticated user (whoever owns the API credential)
2. Label `<agent_label_prefix>:<agent_id>` is added (e.g., `agent:claude-1`)
3. Any other `<agent_label_prefix>:*` labels are removed
4. Task is moved to `in-progress`

When filtering by agent:

```bash
backlog list --assignee=claude-1    # filters by agent label, not native assignee
backlog list --assignee=@me         # filters by authenticated user (native assignment)
```

### Credentials

```yaml
# ~/.config/backlog/credentials.yaml (chmod 600)
github:
  token: ghp_xxxx

linear:
  api_key: lin_api_xxxx
```

Or use environment variables: `GITHUB_TOKEN`, `LINEAR_API_KEY`

---

## Status Mapping

Different backends use different terminology. The CLI normalizes to:

| Canonical | GitHub Issues | GitHub Projects | Linear |
|-----------|---------------|-----------------|--------|
| `backlog` | open + no label | Backlog column | Backlog |
| `todo` | open + `ready` label | Todo column | Todo |
| `in-progress` | open + `in-progress` label | In Progress column | In Progress |
| `review` | open + `review` label | Review column | In Review |
| `done` | closed | Done column | Done |

Mappings are configurable per-workspace:

```yaml
workspaces:
  main:
    backend: github
    status_map:
      backlog: { state: open, labels: [] }
      todo: { state: open, labels: [ready] }
      in-progress: { state: open, labels: [in-progress] }
      review: { state: open, labels: [needs-review] }
      done: { state: closed }
```

---

## Error Handling

### Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | General error (network, auth, invalid input) |
| 2 | Conflict (task already claimed, state conflict) |
| 3 | Not found (task doesn't exist) |
| 4 | Configuration error |

### Error Output

Errors go to stderr in a consistent format:

```
error: Task GH-999 not found
```

With `--format=json`:

```json
{
  "error": {
    "code": "NOT_FOUND",
    "message": "Task GH-999 not found",
    "details": {}
  }
}
```

---

## Agent Integration Patterns

### Basic Workflow

```bash
#!/bin/bash
# agent-worker.sh

TASK_ID=$(backlog next --claim -f id-only)
if [ -z "$TASK_ID" ]; then
  echo "No tasks available"
  exit 0
fi

# Do work...
backlog comment "$TASK_ID" "Starting work"

# On success
backlog move "$TASK_ID" done --comment="Completed in commit abc123"

# On failure
# backlog release "$TASK_ID" --comment="Blocked: need API access"
```

### Structured Agent Loop

```python
import subprocess
import json

def get_next_task():
    result = subprocess.run(
        ["backlog", "next", "--claim", "-f", "json"],
        capture_output=True, text=True
    )
    if result.returncode == 0:
        return json.loads(result.stdout)
    return None

def complete_task(task_id, comment):
    subprocess.run([
        "backlog", "move", task_id, "done",
        "--comment", comment
    ])

def release_task(task_id, reason):
    subprocess.run([
        "backlog", "release", task_id,
        "--comment", reason
    ])
```

### Multi-Agent Partitioning

```yaml
# config.yaml
workspaces:
  backend-agent:
    backend: github
    repo: myorg/myproject
    default_filters:
      labels: [backend, api]
      
  frontend-agent:
    backend: github
    repo: myorg/myproject
    default_filters:
      labels: [frontend, ui]
```

```bash
# backend-agent runs:
backlog -w backend-agent next --claim

# frontend-agent runs:
backlog -w frontend-agent next --claim
```

---

## Technical Stack

**Language:** Go 1.22+

**Key dependencies:**

| Library | Purpose |
|---------|---------|
| `github.com/spf13/cobra` | CLI framework |
| `github.com/spf13/viper` | Configuration management |
| `github.com/google/go-github/v60` | GitHub API client |
| `gopkg.in/yaml.v3` | YAML parsing |
| `github.com/charmbracelet/lipgloss` | Terminal styling (optional) |

**Project structure:**

```
backlog/
├── cmd/
│   └── backlog/
│       └── main.go
├── internal/
│   ├── cli/              # Cobra commands
│   │   ├── root.go
│   │   ├── list.go
│   │   ├── claim.go
│   │   └── ...
│   ├── backend/          # Backend interface + registry
│   │   ├── backend.go
│   │   └── registry.go
│   ├── github/           # GitHub backend
│   ├── linear/           # Linear backend
│   ├── local/            # Local filesystem backend
│   ├── config/           # Config loading
│   └── output/           # Formatters (table, json, plain)
├── go.mod
└── go.sum
```

**Build & distribution:**

```bash
# Local build
go build -o backlog ./cmd/backlog

# Cross-compile
GOOS=darwin GOARCH=arm64 go build -o backlog-darwin-arm64 ./cmd/backlog
GOOS=linux GOARCH=amd64 go build -o backlog-linux-amd64 ./cmd/backlog
GOOS=windows GOARCH=amd64 go build -o backlog-windows-amd64.exe ./cmd/backlog
```

Distribution via:
- GitHub Releases (binaries)
- Homebrew tap
- `go install github.com/alexbrand/backlog/cmd/backlog@latest`

---

## Implementation Phases

Each phase delivers a working tool that provides standalone value.

### Phase 1: Local Backend — Basic CRUD

**Goal:** A working backlog tool for solo use, no external dependencies.

**Delivers:**
- `backlog init` — Initialize `.backlog/` directory structure
- `backlog add <title>` — Create task as markdown file
- `backlog list` — List tasks across all status directories
- `backlog show <id>` — Display task details
- `backlog move <id> <status>` — Move file between directories
- `backlog edit <id>` — Modify task fields
- Output formats: `table`, `json`, `plain`

**Value:** Usable immediately for personal task tracking or single-agent workflows.

---

### Phase 2: Local Backend — Agent Coordination

**Goal:** Multi-agent support with file-based locking.

**Delivers:**
- `backlog claim <id>` — Claim with file lock + agent label in frontmatter
- `backlog release <id>` — Release claim
- `backlog next` — Get highest priority unclaimed task
- `--agent-id` flag and config support
- Exit code 2 for conflicts
- Lock TTL and stale lock detection

**Value:** Multiple agents can safely work the same local backlog without conflicts.

---

### Phase 3: Local Backend — Git Sync

**Goal:** Distributed agents via git coordination.

**Delivers:**
- `lock_mode: git` config option
- Auto-commit on mutations
- Push/pull coordination for claims
- `backlog sync` command
- Conflict detection via failed push

**Value:** Agents on different machines can coordinate through a shared git repo.

---

### Phase 4: GitHub Backend — Issues

**Goal:** Use GitHub Issues as the backing store.

**Delivers:**
- GitHub Issues backend (label-based status)
- `backlog config init` — Interactive setup with OAuth or token
- Status mapping via labels
- Agent labels for claim/release
- All existing commands work against GitHub

**Value:** Teams already using GitHub can manage their backlog with the CLI.

---

### Phase 5: GitHub Backend — Projects

**Goal:** Native GitHub Projects (v2) integration.

**Delivers:**
- Column-based status (instead of labels)
- Custom field mapping
- Project board sync

**Value:** Better UI experience for teams using GitHub Projects for kanban.

---

### Phase 6: Linear Backend

**Goal:** Linear as a backend option.

**Delivers:**
- Linear API integration
- Status/state mapping
- Team and project filtering
- Agent labels via Linear labels

**Value:** Teams using Linear can use the same CLI workflow.

---

### Future Phases (Backlog)

- Jira backend
- `backlog watch` — Real-time updates
- Workspace templates
- Plugin system for custom backends

---

## Open Questions

1. **Should `claim` use file-based locks for the local backend, or rely on git commits?**  
   ✅ **Decided: Support both via `lock_mode` config.**
   - `lock_mode: file` (default) — Fast, no network. Best for single-machine or single-agent setups.
   - `lock_mode: git` — Distributed-safe. Requires `git_sync: true`. Claim = pull, commit, push; conflict on push = exit code 2.

2. **How to handle status values that don't map cleanly?**  
   ✅ **Decided: Lenient for reads, strict for writes.**
   - When listing/reading: Unknown backend statuses map to `backlog` (configurable via `default_unknown_status`), with a warning. Original value preserved in `meta.original_status`.
   - When writing: `backlog move X unknown-status` fails unless the status is explicitly mapped in config.
   - Users can extend the status enum via `status_map` in workspace config.

3. **Should we support bulk operations (`backlog move GH-123 GH-124 GH-125 done`)?**  
   ✅ **Decided: Not in v1.** Single-item operations only. Agents can loop. Revisit if it becomes a pain point.

4. **WebSocket/SSE for real-time updates?**  
   ✅ **Decided: Out of scope for v1, but don't preclude.** Keep `Backend` interface async-first. Can add optional `subscribe(filters): AsyncIterable<TaskEvent>` method and `backlog watch` command in v2.

---

## Success Metrics

- Agent can complete a full claim → work → complete cycle without human intervention
- Switching between GitHub and Linear requires only config changes
- Command latency under 500ms for common operations
- Zero conflicts in multi-agent scenarios when using `claim` properly

---

## Appendix: Local Backend Spec

The `local` backend stores tasks as markdown files:

```
.backlog/
├── config.yaml
├── backlog/
│   └── 001-implement-auth.md
├── todo/
├── in-progress/
├── review/
├── done/
└── .locks/
    └── 003.lock
```

**Task file format:**

```markdown
---
id: "001"
title: Implement auth flow
priority: high
assignee: null
labels: [feature, auth]
created: 2025-01-15T09:00:00Z
updated: 2025-01-18T14:30:00Z
---

## Description

OAuth2 implementation details...

## Comments

### 2025-01-16 @alex
Started research on OAuth providers.
```

**Lock file format:**

```yaml
agent: claude-1
claimed_at: 2025-01-19T10:00:00Z
expires_at: 2025-01-19T10:30:00Z
```

**Git integration:**

When `git_sync: true`, every mutation auto-commits:

```bash
# After backlog claim 001
git commit -m "claim: 001-implement-auth [agent:claude-1]"
git push
```
