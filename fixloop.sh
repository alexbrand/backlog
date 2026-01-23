#!/bin/bash
set -e
set -o pipefail

MAX_ITERATIONS="${1:-10}"
ITERATION=0

echo "=== Fix Loop: Running until all tests pass (max $MAX_ITERATIONS iterations) ==="
echo ""

run_tests() {
  echo "--- Running tests (iteration $ITERATION) ---"

  # Create temp file for test output
  TEST_OUTPUT=$(mktemp)
  TESTS_PASSED=true

  # Run unit tests, capture output and exit code
  echo "=== Unit Tests ===" | tee "$TEST_OUTPUT"
  if ! go test -v ./... 2>&1 | tee -a "$TEST_OUTPUT"; then
    TESTS_PASSED=false
  fi

  echo "" | tee -a "$TEST_OUTPUT"
  echo "=== Spec Tests ===" | tee -a "$TEST_OUTPUT"

  # Run spec tests, capture output and exit code
  if ! make spec 2>&1 | tee -a "$TEST_OUTPUT"; then
    TESTS_PASSED=false
  fi

  LAST_TEST_OUTPUT="$TEST_OUTPUT"

  if [ "$TESTS_PASSED" = true ]; then
    rm -f "$TEST_OUTPUT"
    return 0
  else
    return 1
  fi
}

# Initial test run
ITERATION=1
if run_tests; then
  echo ""
  echo "All tests pass! No fixes needed."
  exit 0
fi

echo ""
echo "Tests failed. Starting fix loop..."
echo ""

while [ $ITERATION -le $MAX_ITERATIONS ]; do
  echo "=== Iteration $ITERATION of $MAX_ITERATIONS ==="

  # Read test output and truncate if too long (keep last 500 lines)
  TEST_CONTENT=$(tail -500 "$LAST_TEST_OUTPUT")
  rm -f "$LAST_TEST_OUTPUT"

  # Call Claude to analyze and fix
  result=$(claude --dangerously-skip-permissions --include-partial-messages --output-format stream-json --verbose -p "The following test output shows failing tests. Analyze the failures and fix the code to make them pass.

TEST OUTPUT:
\`\`\`
$TEST_CONTENT
\`\`\`

Instructions:
1. Analyze the test failures carefully.
2. Fix the code to make the tests pass.
3. Do NOT modify the tests themselves unless they are clearly wrong.
4. Focus only on fixing the failures shown above.
5. After making fixes, output <promise>FIXED</promise> to indicate you've made changes.
6. If you cannot fix the issue (e.g., it requires external changes), output <promise>BLOCKED</promise> with an explanation." | pretty-claude-stream | tee /dev/tty)

  # Check if Claude indicated it's blocked
  if [[ "$result" == *"<promise>BLOCKED</promise>"* ]]; then
    echo ""
    echo "Claude indicated it cannot fix the remaining issues."
    exit 1
  fi

  # Run tests again
  ITERATION=$((ITERATION + 1))

  if run_tests; then
    echo ""
    echo "All tests pass after $((ITERATION - 1)) fix iterations!"
    exit 0
  fi

  echo ""
  echo "Tests still failing, continuing..."
  echo ""
done

echo ""
echo "Max iterations ($MAX_ITERATIONS) reached. Tests still failing."
exit 1
