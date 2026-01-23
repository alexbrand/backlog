Feature: Moving Tasks
  As a user of the backlog CLI
  I want to move tasks between status states
  So that I can track progress of work items

  Background:
    Given a backlog with the following tasks:
      | id    | title               | status      | priority | assignee | labels   | description                  |
      | task1 | Implement feature   | backlog     | high     |          | feature  | Feature implementation       |
      | task2 | Fix bug             | todo        | urgent   | alex     | bug      | Bug fix needed               |
      | task3 | Write tests         | in-progress | medium   |          | testing  | Add test coverage            |
      | task4 | Code review         | review      | low      |          | review   | Review pending changes       |
      | task5 | Completed work      | done        | none     |          |          | Already finished             |

  Scenario Outline: Move task to each valid status
    When I run "backlog move task1 <status>"
    Then the exit code should be 0
    And stdout should contain "task1"
    And stdout should contain "<status>"
    And the task "task1" should have status "<status>"

    Examples:
      | status      |
      | backlog     |
      | todo        |
      | in-progress |
      | review      |
      | done        |

  Scenario: Move task updates file location
    When I run "backlog move task1 todo"
    Then the exit code should be 0
    And a task file should exist in "todo" directory
    And the task "task1" should be in directory "todo"

  Scenario: Move task updates frontmatter
    When I run "backlog move task1 in-progress"
    Then the exit code should be 0
    And the task "task1" should have status "in-progress"
    And the task "task1" should have title "Implement feature"
    And the task "task1" should have priority "high"

  Scenario: Move to invalid status fails
    When I run "backlog move task1 invalid-status"
    Then the exit code should be 1
    And stderr should contain "invalid status"

  Scenario: Move non-existent task returns exit code 3
    When I run "backlog move nonexistent-task todo"
    Then the exit code should be 3
    And stderr should contain "not found"

  Scenario: Move task with comment flag
    When I run "backlog move task2 in-progress --comment 'Starting work on this'"
    Then the exit code should be 0
    And the task "task2" should have status "in-progress"
    And the task "task2" should have comment containing "Starting work on this"

  Scenario: Move task in JSON format
    When I run "backlog move task1 todo -f json"
    Then the exit code should be 0
    And the JSON output should be valid
    And the JSON output should have "id" equal to "task1"
    And the JSON output should have "status" equal to "todo"

  Scenario: Move task from done back to in-progress
    When I run "backlog move task5 in-progress"
    Then the exit code should be 0
    And the task "task5" should have status "in-progress"
    And the task "task5" should be in directory "in-progress"

  Scenario: Move task preserves other fields
    When I run "backlog move task2 review"
    Then the exit code should be 0
    And the task "task2" should have status "review"
    And the task "task2" should have priority "urgent"
    And the task "task2" should have assignee "alex"
    And the task "task2" should have label "bug"
