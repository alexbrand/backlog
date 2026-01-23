Feature: GitHub Connection
  As a user of the backlog CLI
  I want to connect to GitHub using authentication tokens
  So that I can manage tasks stored as GitHub Issues

  # Note: These scenarios test the GitHub backend connection and authentication.
  # Most scenarios require a mock GitHub API server for testing without real credentials.
  # Scenarios marked with @remote require actual GitHub API access.

  @github
  Scenario: Connect with valid token succeeds
    Given a fresh backlog directory
    And a config file with the following content:
      """
      version: 1
      defaults:
        workspace: github
      workspaces:
        github:
          backend: github
          repo: test-owner/test-repo
          api_key_env: GITHUB_TOKEN
          default: true
      """
    And the environment variable "GITHUB_TOKEN" is "ghp_valid_test_token"
    And a mock GitHub API server is running
    When I run "backlog list -f json"
    Then the exit code should be 0
    And the JSON output should be valid
    And the JSON output should have "tasks" as an array

  @github
  Scenario: Connect with invalid token returns exit code 1
    Given a fresh backlog directory
    And a config file with the following content:
      """
      version: 1
      defaults:
        workspace: github
      workspaces:
        github:
          backend: github
          repo: test-owner/test-repo
          api_key_env: GITHUB_TOKEN
          default: true
      """
    And the environment variable "GITHUB_TOKEN" is "invalid_token"
    And a mock GitHub API server is running
    And the mock GitHub API returns auth error for invalid tokens
    When I run "backlog list"
    Then the exit code should be 1
    And stderr should contain "auth"

  @github
  Scenario: Connect with invalid token returns JSON error format
    Given a fresh backlog directory
    And a config file with the following content:
      """
      version: 1
      defaults:
        workspace: github
      workspaces:
        github:
          backend: github
          repo: test-owner/test-repo
          api_key_env: GITHUB_TOKEN
          default: true
      """
    And the environment variable "GITHUB_TOKEN" is "invalid_token"
    And a mock GitHub API server is running
    And the mock GitHub API returns auth error for invalid tokens
    When I run "backlog list -f json"
    Then the exit code should be 1
    And the JSON output should be valid
    And the JSON output should have "error" as an object
    And the JSON output should have "error.code" equal to "AUTH_ERROR"
    And the JSON output should have "error.message" containing "auth"

  @github
  Scenario: Health check passes with valid connection
    Given a fresh backlog directory
    And a config file with the following content:
      """
      version: 1
      defaults:
        workspace: github
      workspaces:
        github:
          backend: github
          repo: test-owner/test-repo
          api_key_env: GITHUB_TOKEN
          default: true
      """
    And the environment variable "GITHUB_TOKEN" is "ghp_valid_test_token"
    And a mock GitHub API server is running
    When I run "backlog config health"
    Then the exit code should be 0
    And stdout should contain "github"
    And stdout should contain "healthy"

  @github
  Scenario: Health check fails with invalid token
    Given a fresh backlog directory
    And a config file with the following content:
      """
      version: 1
      defaults:
        workspace: github
      workspaces:
        github:
          backend: github
          repo: test-owner/test-repo
          api_key_env: GITHUB_TOKEN
          default: true
      """
    And the environment variable "GITHUB_TOKEN" is "invalid_token"
    And a mock GitHub API server is running
    And the mock GitHub API returns auth error for invalid tokens
    When I run "backlog config health"
    Then the exit code should be 1
    And stderr should contain "auth"

  @github
  Scenario: Uses GITHUB_TOKEN environment variable
    Given a fresh backlog directory
    And a config file with the following content:
      """
      version: 1
      defaults:
        workspace: github
      workspaces:
        github:
          backend: github
          repo: test-owner/test-repo
          api_key_env: GITHUB_TOKEN
          default: true
      """
    And the environment variable "GITHUB_TOKEN" is "ghp_env_token_12345"
    And a mock GitHub API server is running
    And the mock GitHub API expects token "ghp_env_token_12345"
    When I run "backlog list -f json"
    Then the exit code should be 0
    And the JSON output should be valid

  @github
  Scenario: Missing token returns auth error
    Given a fresh backlog directory
    And a config file with the following content:
      """
      version: 1
      defaults:
        workspace: github
      workspaces:
        github:
          backend: github
          repo: test-owner/test-repo
          api_key_env: GITHUB_TOKEN
          default: true
      """
    And the environment variable "GITHUB_TOKEN" is not set
    When I run "backlog list"
    Then the exit code should be 1
    And stderr should contain "GITHUB_TOKEN"

  @github
  Scenario: Uses credentials.yaml token when environment variable not set
    Given a fresh backlog directory
    And a config file with the following content:
      """
      version: 1
      defaults:
        workspace: github
      workspaces:
        github:
          backend: github
          repo: test-owner/test-repo
          default: true
      """
    And a credentials file with the following content:
      """
      github:
        token: ghp_credentials_token
      """
    And the environment variable "GITHUB_TOKEN" is not set
    And a mock GitHub API server is running
    And the mock GitHub API expects token "ghp_credentials_token"
    When I run "backlog list -f json"
    Then the exit code should be 0
    And the JSON output should be valid

  @github
  Scenario: Environment variable takes precedence over credentials.yaml
    Given a fresh backlog directory
    And a config file with the following content:
      """
      version: 1
      defaults:
        workspace: github
      workspaces:
        github:
          backend: github
          repo: test-owner/test-repo
          api_key_env: GITHUB_TOKEN
          default: true
      """
    And a credentials file with the following content:
      """
      github:
        token: ghp_credentials_token
      """
    And the environment variable "GITHUB_TOKEN" is "ghp_env_token_wins"
    And a mock GitHub API server is running
    And the mock GitHub API expects token "ghp_env_token_wins"
    When I run "backlog list -f json"
    Then the exit code should be 0
    And the JSON output should be valid

  @github @remote
  Scenario: Real GitHub connection with valid token
    # This scenario tests actual GitHub API connectivity.
    # It requires a real GITHUB_TOKEN with read access to a test repository.
    # Skip this test in CI unless GitHub credentials are available.
    Given a fresh backlog directory
    And a config file with the following content:
      """
      version: 1
      defaults:
        workspace: github
      workspaces:
        github:
          backend: github
          repo: test-owner/test-repo
          api_key_env: GITHUB_TOKEN
          default: true
      """
    And the environment variable "GITHUB_TOKEN" is set to a valid token
    When I run "backlog config health"
    Then the exit code should be 0
    And stdout should contain "healthy"
