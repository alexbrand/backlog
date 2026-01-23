Feature: Error Handling
  As a user of the backlog CLI
  I want consistent error handling and exit codes
  So that I can reliably detect and handle errors in scripts and automation

  # Exit code reference:
  # 0 - Success
  # 1 - General error (network, auth, invalid input)
  # 2 - Conflict (task already claimed, state conflict)
  # 3 - Not found (task doesn't exist)
  # 4 - Configuration error

  # Note: Network and auth errors cannot be tested with the local backend.
  # These scenarios document the expected behavior for remote backends (GitHub, Linear).

  Scenario: Not found error returns exit code 3
    Given a fresh backlog directory
    When I run "backlog show nonexistent-task"
    Then the exit code should be 3
    And stderr should contain "not found"

  Scenario: Not found error when moving non-existent task
    Given a fresh backlog directory
    When I run "backlog move nonexistent-task done"
    Then the exit code should be 3
    And stderr should contain "not found"

  Scenario: Not found error when editing non-existent task
    Given a fresh backlog directory
    When I run "backlog edit nonexistent-task --title=NewTitle"
    Then the exit code should be 3
    And stderr should contain "not found"

  Scenario: Config error returns exit code 4
    Given a fresh backlog directory
    And a config file with the following content:
      """
      this is not valid yaml: [
      """
    When I run "backlog list"
    Then the exit code should be 4
    And stderr should contain "config"

  Scenario: Config error for malformed workspace reference
    Given a fresh backlog directory
    And a config file with the following content:
      """
      version: 1
      workspaces: "not a map"
      """
    When I run "backlog list"
    Then the exit code should be 4
    And stderr should contain "config"

  Scenario: Error message goes to stderr not stdout
    Given a fresh backlog directory
    When I run "backlog show nonexistent-task"
    Then the exit code should be 3
    And stderr should contain "not found"
    And stdout should not contain "error"
    And stdout should not contain "not found"

  Scenario: JSON error format when --format=json for not found
    Given a fresh backlog directory
    When I run "backlog show nonexistent-task -f json"
    Then the exit code should be 3
    And the JSON output should be valid
    And the JSON output should have "error" as an object
    And the JSON output should have "error.code" equal to "NOT_FOUND"
    And the JSON output should have "error.message" containing "not found"

  Scenario: JSON error format when --format=json for config error
    Given a fresh backlog directory
    And a config file with the following content:
      """
      this is not valid yaml: [
      """
    When I run "backlog list -f json"
    Then the exit code should be 4
    And the JSON output should be valid
    And the JSON output should have "error" as an object
    And the JSON output should have "error.code" equal to "CONFIG_ERROR"
    And the JSON output should have "error.message" containing "config"

  Scenario: Invalid status value returns exit code 1
    Given a backlog with the following tasks:
      | id    | title       | status | priority |
      | task1 | Test task   | todo   | medium   |
    When I run "backlog move task1 invalid-status"
    Then the exit code should be 1
    And stderr should contain "invalid"

  Scenario: JSON error format for invalid input
    Given a backlog with the following tasks:
      | id    | title       | status | priority |
      | task1 | Test task   | todo   | medium   |
    When I run "backlog move task1 invalid-status -f json"
    Then the exit code should be 1
    And the JSON output should be valid
    And the JSON output should have "error" as an object
    And the JSON output should have "error.code" equal to "INVALID_INPUT"

  # The following scenarios document expected behavior for remote backends.
  # They are marked with @remote tag to indicate they require remote backend testing.

  @remote @github @linear
  Scenario: Network error returns exit code 1
    # This scenario cannot be tested with the local backend.
    # Expected behavior for remote backends:
    # - When a network request fails (timeout, DNS resolution, connection refused)
    # - Exit code should be 1
    # - Error message should indicate network issue
    # - JSON format should have error.code = "NETWORK_ERROR"
    Given a workspace configured for a remote backend
    And the network is unavailable
    When I run "backlog list"
    Then the exit code should be 1
    And stderr should contain "network"

  @remote @github @linear
  Scenario: Auth error returns exit code 1
    # This scenario cannot be tested with the local backend.
    # Expected behavior for remote backends:
    # - When authentication fails (invalid token, expired credentials)
    # - Exit code should be 1
    # - Error message should indicate authentication issue
    # - JSON format should have error.code = "AUTH_ERROR"
    Given a workspace configured for a remote backend
    And the authentication token is invalid
    When I run "backlog list"
    Then the exit code should be 1
    And stderr should contain "auth"
