Feature: Global Flags
  As a user of the backlog CLI
  I want to use global flags to customize command behavior
  So that I can control output format, verbosity, and workspace selection

  Scenario: Quiet flag suppresses non-essential output
    Given a backlog with the following tasks:
      | id    | title           | status | priority |
      | task1 | First task      | todo   | high     |
    When I run "backlog add 'New task' -q"
    Then the exit code should be 0
    And stdout should be empty

  Scenario: Quiet flag with long form
    Given a backlog with the following tasks:
      | id    | title           | status | priority |
      | task1 | First task      | todo   | high     |
    When I run "backlog add 'Another task' --quiet"
    Then the exit code should be 0
    And stdout should be empty

  Scenario: Verbose flag shows debug information
    Given a backlog with the following tasks:
      | id    | title           | status | priority |
      | task1 | First task      | todo   | high     |
    When I run "backlog list -v"
    Then the exit code should be 0
    And stderr should contain "debug"

  Scenario: Verbose flag with long form
    Given a backlog with the following tasks:
      | id    | title           | status | priority |
      | task1 | First task      | todo   | high     |
    When I run "backlog list --verbose"
    Then the exit code should be 0
    And stderr should contain "debug"

  Scenario: Format flag changes output format
    Given a backlog with the following tasks:
      | id    | title           | status | priority |
      | task1 | First task      | todo   | high     |
    When I run "backlog list -f json"
    Then the exit code should be 0
    And the JSON output should be valid
    And the JSON output should have "tasks[0].id" equal to "task1"

  Scenario: Format flag with long form
    Given a backlog with the following tasks:
      | id    | title           | status | priority |
      | task1 | First task      | todo   | high     |
    When I run "backlog list --format json"
    Then the exit code should be 0
    And the JSON output should be valid
    And the JSON output should have "tasks[0].id" equal to "task1"

  Scenario: Format flag supports table output
    Given a backlog with the following tasks:
      | id    | title           | status | priority |
      | task1 | First task      | todo   | high     |
    When I run "backlog list -f table"
    Then the exit code should be 0
    And stdout should contain "ID"
    And stdout should contain "STATUS"
    And stdout should contain "PRIORITY"
    And stdout should contain "TITLE"

  Scenario: Format flag supports plain output
    Given a backlog with the following tasks:
      | id    | title           | status | priority |
      | task1 | First task      | todo   | high     |
    When I run "backlog list -f plain"
    Then the exit code should be 0
    And stdout should contain "task1"
    And stdout should not contain "ID"
    And stdout should not contain "STATUS"

  Scenario: Format flag supports id-only output
    Given a backlog with the following tasks:
      | id    | title           | status | priority |
      | task1 | First task      | todo   | high     |
    When I run "backlog list -f id-only"
    Then the exit code should be 0
    And stdout should contain "task1"
    And stdout should not contain "First task"

  Scenario: Workspace flag selects workspace
    Given a fresh backlog directory
    And a config file with the following content:
      """
      version: 1
      defaults:
        workspace: primary
      workspaces:
        primary:
          backend: local
          path: ./.backlog
          default: true
        secondary:
          backend: local
          path: ./.backlog-secondary
      """
    And a backlog with the following tasks:
      | id    | title           | status | priority |
      | task1 | Primary task    | todo   | high     |
    When I run "backlog list -w secondary -f json"
    Then the exit code should be 0
    And the JSON output should be valid
    And the JSON output should have "count" equal to "0"

  Scenario: Workspace flag with long form
    Given a fresh backlog directory
    And a config file with the following content:
      """
      version: 1
      defaults:
        workspace: primary
      workspaces:
        primary:
          backend: local
          path: ./.backlog
          default: true
        secondary:
          backend: local
          path: ./.backlog-secondary
      """
    And a backlog with the following tasks:
      | id    | title           | status | priority |
      | task1 | Primary task    | todo   | high     |
    When I run "backlog list --workspace secondary -f json"
    Then the exit code should be 0
    And the JSON output should be valid
    And the JSON output should have "count" equal to "0"
