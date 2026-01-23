Feature: Listing Tasks
  As a user of the backlog CLI
  I want to list tasks from my backlog
  So that I can see what work needs to be done

  Scenario: List all tasks (excludes done by default)
    Given a backlog with the following tasks:
      | id    | title           | status      | priority |
      | task1 | First task      | todo        | high     |
      | task2 | Second task     | in-progress | medium   |
      | task3 | Completed task  | done        | low      |
      | task4 | Backlog task    | backlog     | none     |
    When I run "backlog list"
    Then the exit code should be 0
    And stdout should contain "First task"
    And stdout should contain "Second task"
    And stdout should contain "Backlog task"
    And stdout should not contain "Completed task"

  Scenario: List tasks in table format (default)
    Given a backlog with the following tasks:
      | id    | title           | status      | priority | assignee |
      | task1 | First task      | todo        | high     | alice    |
      | task2 | Second task     | in-progress | medium   |          |
      | task3 | Third task      | backlog     | low      | bob      |
    When I run "backlog list"
    Then the exit code should be 0
    And stdout should contain "ID"
    And stdout should contain "STATUS"
    And stdout should contain "PRIORITY"
    And stdout should contain "TITLE"
    And stdout should contain "ASSIGNEE"
    And stdout should contain "task1"
    And stdout should contain "todo"
    And stdout should contain "high"
    And stdout should contain "First task"
    And stdout should contain "alice"
    And stdout should contain "task2"
    And stdout should contain "in-progress"
    And stdout should contain "medium"
    And stdout should contain "Second task"
    And stdout should contain "task3"
    And stdout should contain "backlog"
    And stdout should contain "low"
    And stdout should contain "Third task"
    And stdout should contain "bob"

  Scenario: List tasks in JSON format
    Given a backlog with the following tasks:
      | id    | title           | status      | priority | assignee | labels        |
      | task1 | First task      | todo        | high     | alice    | feature,api   |
      | task2 | Second task     | in-progress | medium   |          | bug           |
      | task3 | Third task      | backlog     | low      | bob      |               |
    When I run "backlog list -f json"
    Then the exit code should be 0
    And the JSON output should be valid
    And the JSON output should have "tasks" as an array
    And the JSON output should have "count" equal to "3"
    And the JSON output should have "hasMore" equal to "false"
    And the JSON output should have "tasks[0].id" equal to "task1"
    And the JSON output should have "tasks[0].title" equal to "First task"
    And the JSON output should have "tasks[0].status" equal to "todo"
    And the JSON output should have "tasks[0].priority" equal to "high"
    And the JSON output should have "tasks[0].assignee" equal to "alice"
    And the JSON output should have array "tasks[0].labels" containing "feature"
    And the JSON output should have array "tasks[0].labels" containing "api"

  Scenario: List tasks in plain format
    Given a backlog with the following tasks:
      | id    | title           | status      | priority |
      | task1 | First task      | todo        | high     |
      | task2 | Second task     | in-progress | medium   |
      | task3 | Third task      | backlog     | low      |
    When I run "backlog list -f plain"
    Then the exit code should be 0
    And stdout should contain "task1"
    And stdout should contain "First task"
    And stdout should contain "task2"
    And stdout should contain "Second task"
    And stdout should contain "task3"
    And stdout should contain "Third task"
    And stdout should not contain "ID"
    And stdout should not contain "STATUS"

  Scenario: List tasks in id-only format
    Given a backlog with the following tasks:
      | id    | title           | status      | priority |
      | task1 | First task      | todo        | high     |
      | task2 | Second task     | in-progress | medium   |
      | task3 | Third task      | backlog     | low      |
    When I run "backlog list -f id-only"
    Then the exit code should be 0
    And stdout should contain "task1"
    And stdout should contain "task2"
    And stdout should contain "task3"
    And stdout should not contain "First task"
    And stdout should not contain "Second task"
    And stdout should not contain "Third task"
    And stdout should not contain "ID"
    And stdout should not contain "STATUS"
    And stdout should not contain "PRIORITY"
    And stdout should not contain "TITLE"

  Scenario: List with status filter
    Given a backlog with the following tasks:
      | id    | title           | status      | priority |
      | task1 | First task      | todo        | high     |
      | task2 | Second task     | in-progress | medium   |
      | task3 | Third task      | backlog     | low      |
      | task4 | Fourth task     | todo        | low      |
      | task5 | Fifth task      | done        | high     |
    When I run "backlog list --status=todo"
    Then the exit code should be 0
    And stdout should contain "First task"
    And stdout should contain "Fourth task"
    And stdout should not contain "Second task"
    And stdout should not contain "Third task"
    And stdout should not contain "Fifth task"

  Scenario: List with multiple status values
    Given a backlog with the following tasks:
      | id    | title           | status      | priority |
      | task1 | First task      | todo        | high     |
      | task2 | Second task     | in-progress | medium   |
      | task3 | Third task      | backlog     | low      |
      | task4 | Fourth task     | review      | low      |
      | task5 | Fifth task      | done        | high     |
    When I run "backlog list --status=todo,in-progress"
    Then the exit code should be 0
    And stdout should contain "First task"
    And stdout should contain "Second task"
    And stdout should not contain "Third task"
    And stdout should not contain "Fourth task"
    And stdout should not contain "Fifth task"
