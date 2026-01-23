Feature: Adding Tasks
  As a user of the backlog CLI
  I want to add tasks to my backlog
  So that I can track work items

  Scenario: Add task with title only
    Given a fresh backlog directory
    When I run "backlog add 'Fix login bug'"
    Then the exit code should be 0
    And stdout should contain "Created"
    And a task file should exist in "backlog" directory

  Scenario: Add task with priority flag
    Given a fresh backlog directory
    When I run "backlog add 'Urgent fix' --priority=urgent"
    Then the exit code should be 0
    And a task file should exist in "backlog" directory
    And the created task should have priority "urgent"

  Scenario: Add task with multiple labels
    Given a fresh backlog directory
    When I run "backlog add 'Bug fix' --label=bug --label=critical"
    Then the exit code should be 0
    And the created task should have label "bug"
    And the created task should have label "critical"

  Scenario: Add task with description flag
    Given a fresh backlog directory
    When I run "backlog add 'New feature' --description='Detailed description here'"
    Then the exit code should be 0
    And the created task should have description containing "Detailed description"

  Scenario: Add task with body-file flag
    Given a fresh backlog directory
    And a file "task-details.md" with content "This is the task body from a file."
    When I run "backlog add 'Research caching' --body-file=task-details.md"
    Then the exit code should be 0
    And the created task should have description containing "task body from a file"

  Scenario: Add task with explicit status
    Given a fresh backlog directory
    When I run "backlog add 'Ready task' --status=todo"
    Then the exit code should be 0
    And a task file should exist in "todo" directory

  Scenario: Add task generates unique ID
    Given a fresh backlog directory
    When I run "backlog add 'First task'"
    And I run "backlog add 'Second task'"
    Then the exit code should be 0
    And the task count should be 2

  Scenario: Add task outputs created task ID
    Given a fresh backlog directory
    When I run "backlog add 'Test task'"
    Then the exit code should be 0
    And stdout should match pattern "Created [A-Za-z0-9-]+:"

  Scenario: Add task with JSON output format
    Given a fresh backlog directory
    When I run "backlog add 'JSON task' -f json"
    Then the exit code should be 0
    And the JSON output should have "title" equal to "JSON task"

  Scenario Outline: Add task with each priority level
    Given a fresh backlog directory
    When I run "backlog add 'Priority test' --priority=<priority>"
    Then the exit code should be 0
    And the created task should have priority "<priority>"

    Examples:
      | priority |
      | urgent   |
      | high     |
      | medium   |
      | low      |
      | none     |
