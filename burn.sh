#!/bin/bash
set -e

# Default values
MAX_ITERATIONS=10

usage() {
  echo "Usage: $0 [options]"
  echo ""
  echo "Burn through the backlog using Claude to implement tasks."
  echo ""
  echo "Options:"
  echo "  -n, --iterations NUM   Maximum iterations (default: $MAX_ITERATIONS)"
  echo "  -l, --label LABEL      Only work on tasks with this label"
  echo "  -h, --help             Show this help message"
  exit 1
}

LABEL_FLAG=""

# Parse arguments
while [[ $# -gt 0 ]]; do
  case $1 in
    -n|--iterations)
      MAX_ITERATIONS="$2"
      shift 2
      ;;
    -l|--label)
      LABEL_FLAG="--label=$2"
      shift 2
      ;;
    -h|--help)
      usage
      ;;
    *)
      echo "Unknown option: $1"
      usage
      ;;
  esac
done

echo "=== Backlog Burn ==="
echo "Max iterations: $MAX_ITERATIONS"
echo ""

for ((i=1; i<=MAX_ITERATIONS; i++)); do
  echo "--- Iteration $i of $MAX_ITERATIONS ---"

  # Get and claim the next highest priority task
  TASK_JSON=$(backlog next --claim $LABEL_FLAG --format json 2>/dev/null || echo "")

  if [ -z "$TASK_JSON" ] || [ "$TASK_JSON" = "null" ]; then
    echo "No more tasks available. Done!"
    exit 0
  fi

  TASK_ID=$(echo "$TASK_JSON" | jq -r '.id')
  TASK_TITLE=$(echo "$TASK_JSON" | jq -r '.title')
  TASK_DESC=$(echo "$TASK_JSON" | jq -r '.description // "No description"')

  echo "Claimed: $TASK_ID - $TASK_TITLE"
  echo ""

  # Run Claude to implement the task
  result=$(claude --dangerously-skip-permissions \
    --output-format stream-json \
    --verbose \
    -p "You are working on the backlog for this project.

CURRENT TASK: $TASK_ID
TITLE: $TASK_TITLE
DESCRIPTION:
$TASK_DESC

INSTRUCTIONS:
1. Implement the task according to its description
2. Run tests and type checks to verify your changes
3. Commit your changes with a descriptive message
4. Push your changes
5. Mark the task as done: backlog complete $TASK_ID

IMPORTANT:
- The task is already claimed - do not run 'backlog claim'
- Only work on this single task
- Follow the project's coding standards (see CLAUDE.md if it exists)
- If you discover follow-up work, TODOs, or related tasks that should be done later,
  add them to the backlog: backlog add \"Task title\" --description \"Details...\"
- If the task cannot be completed, release it: backlog release $TASK_ID --comment \"reason\"
- When finished, output <task-complete>$TASK_ID</task-complete>" \
    2>&1 | tee /dev/tty)

  # Check if task was completed
  if [[ "$result" == *"<task-complete>$TASK_ID</task-complete>"* ]]; then
    echo ""
    echo "✓ Task $TASK_ID completed"
  else
    echo ""
    echo "? Task $TASK_ID status unclear. Check: backlog show $TASK_ID"
  fi

  echo ""
  sleep 2
done

echo ""
echo "Reached max iterations ($MAX_ITERATIONS)."
echo "Remaining tasks:"
backlog list --status backlog,todo
