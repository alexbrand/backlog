Feature: Linear Connection
  As a user of the backlog CLI
  I want to connect to Linear using API keys
  So that I can manage tasks stored as Linear issues

  # Note: These scenarios test the Linear backend connection and authentication.
  # Most scenarios require a mock Linear API server for testing without real credentials.
  # Scenarios marked with @remote require actual Linear API access.

  @linear
  Scenario: Connect with valid API key succeeds
    Given a fresh backlog directory
    And a config file with the following content:
      """
      version: 1
      defaults:
        workspace: linear
      workspaces:
        linear:
          backend: linear
          team: ENG
          api_key_env: LINEAR_API_KEY
          default: true
      """
    And the environment variable "LINEAR_API_KEY" is "lin_api_valid_test_key"
    And a mock Linear API server is running
    When I run "backlog list -f json"
    Then the exit code should be 0
    And the JSON output should be valid
    And the JSON output should have "tasks" as an array

  @linear
  Scenario: Connect with invalid API key returns exit code 1
    Given a fresh backlog directory
    And a config file with the following content:
      """
      version: 1
      defaults:
        workspace: linear
      workspaces:
        linear:
          backend: linear
          team: ENG
          api_key_env: LINEAR_API_KEY
          default: true
      """
    And the environment variable "LINEAR_API_KEY" is "invalid_key"
    And a mock Linear API server is running
    And the mock Linear API returns auth error for invalid keys
    When I run "backlog list"
    Then the exit code should be 1
    And stderr should contain "auth"

  @linear
  Scenario: Connect with invalid API key returns JSON error format
    Given a fresh backlog directory
    And a config file with the following content:
      """
      version: 1
      defaults:
        workspace: linear
      workspaces:
        linear:
          backend: linear
          team: ENG
          api_key_env: LINEAR_API_KEY
          default: true
      """
    And the environment variable "LINEAR_API_KEY" is "invalid_key"
    And a mock Linear API server is running
    And the mock Linear API returns auth error for invalid keys
    When I run "backlog list -f json"
    Then the exit code should be 1
    And the JSON output should be valid
    And the JSON output should have "error" as an object
    And the JSON output should have "error.code" equal to "AUTH_ERROR"
    And the JSON output should have "error.message" containing "auth"

  @linear
  Scenario: Uses LINEAR_API_KEY environment variable
    Given a fresh backlog directory
    And a config file with the following content:
      """
      version: 1
      defaults:
        workspace: linear
      workspaces:
        linear:
          backend: linear
          team: ENG
          api_key_env: LINEAR_API_KEY
          default: true
      """
    And the environment variable "LINEAR_API_KEY" is "lin_api_env_key_12345"
    And a mock Linear API server is running
    And the mock Linear API expects key "lin_api_env_key_12345"
    When I run "backlog list -f json"
    Then the exit code should be 0
    And the JSON output should be valid

  @linear
  Scenario: Missing API key returns auth error
    Given a fresh backlog directory
    And a config file with the following content:
      """
      version: 1
      defaults:
        workspace: linear
      workspaces:
        linear:
          backend: linear
          team: ENG
          api_key_env: LINEAR_API_KEY
          default: true
      """
    And the environment variable "LINEAR_API_KEY" is not set
    When I run "backlog list"
    Then the exit code should be 1
    And stderr should contain "LINEAR_API_KEY"

  @linear
  Scenario: Uses credentials.yaml API key when environment variable not set
    Given a fresh backlog directory
    And a config file with the following content:
      """
      version: 1
      defaults:
        workspace: linear
      workspaces:
        linear:
          backend: linear
          team: ENG
          default: true
      """
    And a credentials file with the following content:
      """
      linear:
        api_key: lin_api_credentials_key
      """
    And the environment variable "LINEAR_API_KEY" is not set
    And a mock Linear API server is running
    And the mock Linear API expects key "lin_api_credentials_key"
    When I run "backlog list -f json"
    Then the exit code should be 0
    And the JSON output should be valid

  @linear
  Scenario: Environment variable takes precedence over credentials.yaml
    Given a fresh backlog directory
    And a config file with the following content:
      """
      version: 1
      defaults:
        workspace: linear
      workspaces:
        linear:
          backend: linear
          team: ENG
          api_key_env: LINEAR_API_KEY
          default: true
      """
    And a credentials file with the following content:
      """
      linear:
        api_key: lin_api_credentials_key
      """
    And the environment variable "LINEAR_API_KEY" is "lin_api_env_key_wins"
    And a mock Linear API server is running
    And the mock Linear API expects key "lin_api_env_key_wins"
    When I run "backlog list -f json"
    Then the exit code should be 0
    And the JSON output should be valid
