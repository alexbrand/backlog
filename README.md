# backlog

A command-line tool for managing tasks across multiple issue tracking backends. It provides a unified, agent-friendly interface that abstracts away provider-specific APIs, enabling both humans and AI agents to manage backlogs through simple, composable commands.

## Features

- **Unified interface**: One CLI that works identically across GitHub, Linear, and local file-based backends
- **Agent-first design**: Predictable output formats (JSON, plain text), atomic operations, clear exit codes
- **Human-friendly**: Intuitive commands, sensible defaults, good DX for manual use
- **Multi-agent coordination**: Built-in primitives for claiming, releasing, and locking tasks

## Installation

### Homebrew (macOS and Linux)

```bash
brew install alexbrand/tap/backlog
```

Or add the tap first:

```bash
brew tap alexbrand/tap
brew install backlog
```

### From Source

```bash
go install github.com/alexbrand/backlog/cmd/backlog@latest
```

### Download Binary

Download pre-built binaries from the [releases page](https://github.com/alexbrand/backlog/releases).

### Build Locally

```bash
git clone https://github.com/alexbrand/backlog.git
cd backlog
go build -o backlog ./cmd/backlog
```

## Quick Start

### Local Backend

Initialize a local backlog in your project:

```bash
backlog init
```

This creates a `.backlog/` directory with status folders.

Add a task:

```bash
backlog add "Implement authentication"
backlog add "Fix login bug" --priority=high --label=bug
```

List tasks:

```bash
backlog list
backlog list --status=todo
backlog list -f json
```

Move tasks through the workflow:

```bash
backlog move 001 in-progress
backlog move 001 done
```

### GitHub Backend

Configure a GitHub workspace in `~/.config/backlog/config.yaml`:

```yaml
version: 1
workspaces:
  main:
    backend: github
    repo: owner/repo
    default: true
```

Set your GitHub token:

```bash
export GITHUB_TOKEN=ghp_xxxx
```

Now all commands work against GitHub Issues:

```bash
backlog list
backlog add "New feature request"
backlog show GH-123
```

### Linear Backend

Configure a Linear workspace:

```yaml
version: 1
workspaces:
  work:
    backend: linear
    team: ENG
```

Set your Linear API key:

```bash
export LINEAR_API_KEY=lin_api_xxxx
```

## Commands

### Task Management

| Command | Description |
|---------|-------------|
| `backlog init` | Initialize a local `.backlog/` directory |
| `backlog add <title>` | Create a new task |
| `backlog list` | List tasks with optional filtering |
| `backlog show <id>` | Display full task details |
| `backlog edit <id>` | Modify task fields |
| `backlog move <id> <status>` | Transition task to a new status |
| `backlog delete <id>` | Remove a task permanently |
| `backlog reorder <id>` | Change the position of a task in the list |
| `backlog comment <id> <message>` | Add a comment to a task |

### Agent Coordination

| Command | Description |
|---------|-------------|
| `backlog claim <id>` | Claim a task for the current agent |
| `backlog release <id>` | Release a claimed task back to todo |
| `backlog next` | Get the next recommended task to work on |
| `backlog next --claim` | Get and atomically claim the next task |

### Configuration

| Command | Description |
|---------|-------------|
| `backlog config show` | Display current configuration |
| `backlog config init` | Interactive setup wizard |
| `backlog sync` | Sync local cache with remote (git backend) |

## Global Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--workspace` | `-w` | Target workspace |
| `--format` | `-f` | Output format: `table`, `json`, `plain`, `id-only` |
| `--quiet` | `-q` | Suppress non-essential output |
| `--verbose` | `-v` | Show debug information |
| `--agent-id` | | Agent identifier for claims |

## Configuration

### Config File Location

Configuration is loaded from (in order of precedence):

1. `--config` flag (explicit path)
2. `.backlog/config.yaml` (project-local)
3. `~/.config/backlog/config.yaml` (user global)

### Config Schema

```yaml
version: 1

defaults:
  format: table           # default output format
  workspace: main         # default workspace name
  agent_id: claude-1      # global default agent ID

workspaces:
  main:
    backend: github
    repo: owner/repo
    project: 1                    # optional: GitHub Project number
    status_field: Status          # project field name for status
    agent_id: claude-main         # overrides global for this workspace
    agent_label_prefix: agent     # creates "agent:claude-main" labels
    default: true

  work:
    backend: linear
    team: ENG
    api_key_env: LINEAR_API_KEY

  offline:
    backend: local
    path: ./.backlog
    lock_mode: file               # file (default) or git
    git_sync: true                # auto-commit on changes
```

### Credentials

Credentials can be provided via:

1. Environment variables: `GITHUB_TOKEN`, `LINEAR_API_KEY`
2. Credentials file: `~/.config/backlog/credentials.yaml`

```yaml
# ~/.config/backlog/credentials.yaml (chmod 600)
github:
  token: ghp_xxxx

linear:
  api_key: lin_api_xxxx
```

## Status Values

Tasks have a canonical status that maps across backends:

| Status | Description |
|--------|-------------|
| `backlog` | Not yet ready to work on |
| `todo` | Ready to be worked on |
| `in-progress` | Currently being worked on |
| `review` | Waiting for review |
| `done` | Completed |

### Status Mapping

Each backend maps these statuses to its native concepts:

| Canonical | GitHub Issues | GitHub Projects | Linear |
|-----------|---------------|-----------------|--------|
| `backlog` | open + no label | Backlog column | Backlog |
| `todo` | open + `ready` label | Todo column | Todo |
| `in-progress` | open + `in-progress` label | In Progress column | In Progress |
| `review` | open + `review` label | Review column | In Review |
| `done` | closed | Done column | Done |

Custom mappings can be configured per-workspace:

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

## Agent Integration

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

### Python Integration

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

### Agent Identity

Agent ID is resolved in priority order:

1. CLI flag: `--agent-id=claude-1`
2. Environment variable: `BACKLOG_AGENT_ID`
3. Workspace config: `workspaces.<name>.agent_id`
4. Global default: `defaults.agent_id`
5. Hostname fallback

Example for containerized agents:

```bash
export BACKLOG_AGENT_ID=claude-worker-$(hostname)
backlog claim GH-123   # uses env var
```

### Multi-Agent Partitioning

Configure separate workspaces to partition work by labels:

```yaml
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

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | General error (network, auth, invalid input) |
| 2 | Conflict (task already claimed, state conflict) |
| 3 | Not found (task doesn't exist) |
| 4 | Configuration error |

## Local Backend

The local backend stores tasks as markdown files:

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

### Task File Format

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

### Git Sync

When `git_sync: true`, every mutation auto-commits:

```bash
# After backlog claim 001
git commit -m "claim: 001-implement-auth [agent:claude-1]"
git push
```

## Development

### Running Tests

```bash
# Unit tests
go test ./...

# Integration tests (executable specification)
make spec
```

### Building

```bash
# Local build
go build -o backlog ./cmd/backlog

# Cross-compile
GOOS=darwin GOARCH=arm64 go build -o backlog-darwin-arm64 ./cmd/backlog
GOOS=linux GOARCH=amd64 go build -o backlog-linux-amd64 ./cmd/backlog
```

## License

MIT
