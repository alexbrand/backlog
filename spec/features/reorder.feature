Feature: Reordering Tasks
  As a user of the backlog CLI
  I want to reorder tasks within a status
  So that I can control task position within a group

  Background:
    Given a backlog with the following tasks:
      | id    | title         | status  | priority |
      | task1 | First task    | todo    | high     |
      | task2 | Second task   | todo    | high     |
      | task3 | Third task    | todo    | high     |
      | task4 | Other status  | backlog | high     |

  Scenario: Reorder task to first position
    When I run "backlog reorder task3 --first"
    Then the exit code should be 0
    And stdout should contain "task3"
    When I run "backlog list --status=todo -f json"
    Then the exit code should be 0
    And the JSON output should have "tasks[0].id" equal to "task3"

  Scenario: Reorder task to last position
    When I run "backlog reorder task1 --last"
    Then the exit code should be 0
    And stdout should contain "task1"
    When I run "backlog list --status=todo -f json"
    Then the exit code should be 0
    And the JSON output should have "tasks[2].id" equal to "task1"

  Scenario: Reorder task before another
    When I run "backlog reorder task3 --before task1"
    Then the exit code should be 0
    And stdout should contain "task3"
    When I run "backlog list --status=todo -f json"
    Then the exit code should be 0
    And the JSON output should have "tasks[0].id" equal to "task3"
    And the JSON output should have "tasks[1].id" equal to "task1"

  Scenario: Reorder task after another
    When I run "backlog reorder task1 --after task3"
    Then the exit code should be 0
    And stdout should contain "task1"
    When I run "backlog list --status=todo -f json"
    Then the exit code should be 0
    And the JSON output should have "tasks[0].id" equal to "task2"
    And the JSON output should have "tasks[1].id" equal to "task3"
    And the JSON output should have "tasks[2].id" equal to "task1"

  Scenario: Reorder with no position flag fails
    When I run "backlog reorder task1"
    Then the exit code should be 1
    And stderr should contain "one of --before, --after, --first, or --last is required"

  Scenario: Reorder with multiple position flags fails
    When I run "backlog reorder task1 --first --last"
    Then the exit code should be 1
    And stderr should contain "only one of"

  Scenario: Reorder non-existent task returns exit code 3
    When I run "backlog reorder nonexistent --first"
    Then the exit code should be 3
    And stderr should contain "not found"

  Scenario: Reorder with non-existent reference task
    When I run "backlog reorder task1 --before nonexistent"
    Then the exit code should be 1
    And stderr should contain "does not exist"

  Scenario: Reorder relative to self fails
    When I run "backlog reorder task1 --before task1"
    Then the exit code should be 1
    And stderr should contain "cannot reorder task relative to itself"

  Scenario: Reorder in JSON format
    When I run "backlog reorder task2 --first -f json"
    Then the exit code should be 0
    And the JSON output should be valid
    And the JSON output should have "id" equal to "task2"
    And stdout should contain "sort_order"

  Scenario: Reorder persists sort order across multiple operations
    When I run "backlog reorder task3 --first"
    Then the exit code should be 0
    When I run "backlog reorder task2 --after task3"
    Then the exit code should be 0
    When I run "backlog list --status=todo -f json"
    Then the exit code should be 0
    And the JSON output should have "tasks[0].id" equal to "task3"
    And the JSON output should have "tasks[1].id" equal to "task2"
    And the JSON output should have "tasks[2].id" equal to "task1"

  Scenario: Reorder preserves other task fields
    When I run "backlog reorder task1 --last"
    Then the exit code should be 0
    And the task "task1" should have title "First task"
    And the task "task1" should have priority "high"
    And the task "task1" should have status "todo"
