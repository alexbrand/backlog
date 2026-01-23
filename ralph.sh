#!/bin/bash
set -e

if [ -z "$1" ] || [ -z "$2" ]; then
  echo "Usage: $0 <task-file> <iterations>"
  exit 1
fi

TASK_FILE="$1"
ITERATIONS="$2"

if [ ! -f "$TASK_FILE" ]; then
  echo "Error: Task file '$TASK_FILE' not found"
  exit 1
fi

for ((i=1; i<=$ITERATIONS; i++)); do
  result=$(claude --dangerously-skip-permissions --include-partial-messages --output-format stream-json --verbose -p "@PRD.md @$TASK_FILE \
  1. Find the highest-priority task and implement it. \
  2. Run your tests and type checks. \
  3. Update the $TASK_FILE file with what was done. \
  4. Commit your changes. \
  5. Push your changes. \
  ONLY WORK ON A SINGLE TASK. \
  If the task list is complete, output <promise>COMPLETE</promise>." | pretty-claude-stream |  tee /dev/tty)

  if [[ "$result" == *"<promise>COMPLETE</promise>"* ]]; then
    echo "Task list complete after $i iterations."
    exit 0
  fi
done
