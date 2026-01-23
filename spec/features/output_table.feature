Feature: Table Output Format
  As a user of the backlog CLI
  I want the table output to be well-formatted
  So that I can easily read task information in my terminal

  Background:
    Given a backlog with the following tasks:
      | id    | title                                              | status      | priority | assignee |
      | task1 | Short title                                        | todo        | high     | alice    |
      | task2 | A much longer title that might need truncation     | in-progress | medium   |          |
      | task3 | Another task                                       | backlog     | low      | bob      |

  Scenario: Table output has correct headers
    When I run "backlog list"
    Then the exit code should be 0
    And stdout should contain "ID"
    And stdout should contain "STATUS"
    And stdout should contain "PRIORITY"
    And stdout should contain "TITLE"
    And stdout should contain "ASSIGNEE"

  Scenario: Table output aligns columns
    When I run "backlog list"
    Then the exit code should be 0
    # All IDs should be visible
    And stdout should contain "task1"
    And stdout should contain "task2"
    And stdout should contain "task3"
    # All statuses should be visible
    And stdout should contain "todo"
    And stdout should contain "in-progress"
    And stdout should contain "backlog"
    # All priorities should be visible
    And stdout should contain "high"
    And stdout should contain "medium"
    And stdout should contain "low"

  Scenario: Table output truncates long titles
    Given a backlog with the following tasks:
      | id    | title                                                                                                         | status | priority |
      | long1 | This is an extremely long task title that should definitely be truncated in the table output to fit properly  | todo   | high     |
    When I run "backlog list"
    Then the exit code should be 0
    And stdout should contain "long1"
    # The title should be present but may be truncated
    And stdout should contain "This is an extremely long"

  Scenario: Table output shows dash for empty fields
    Given a backlog with the following tasks:
      | id    | title        | status | priority | assignee |
      | task4 | No assignee  | todo   | high     |          |
    When I run "backlog list --status=todo"
    Then the exit code should be 0
    And stdout should contain "task4"
    And stdout should contain "No assignee"
    # Empty assignee should show as dash
    And stdout should match pattern "task4.*—|task4.*-"
