Feature: GitHub Projects
  As a user of the backlog CLI
  I want to manage tasks stored in GitHub Projects
  So that I can use project boards for kanban-style workflow management

  # Note: These scenarios test GitHub Projects v2 integration.
  # GitHub Projects v2 uses GraphQL API for project-specific operations.
  # Issues are still managed via REST API, but project board columns are managed via GraphQL.

  Background:
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
          project: 1
          status_field: Status
          api_key_env: GITHUB_TOKEN
          default: true
      """
    And the environment variable "GITHUB_TOKEN" is "ghp_valid_test_token"
    And a mock GitHub API server is running

  @github @projects
  Scenario: Connect to repository with project
    Given a GitHub project 1 with columns:
      | name        | id   |
      | Backlog     | COL1 |
      | Todo        | COL2 |
      | In Progress | COL3 |
      | Done        | COL4 |
    When I run "backlog config health"
    Then the exit code should be 0
    And stdout should contain "github"
    And stdout should contain "healthy"
    And stdout should contain "project"

  @github @projects
  Scenario: Connect to repository with project shows project info
    Given a GitHub project 1 with columns:
      | name        | id   |
      | Backlog     | COL1 |
      | Todo        | COL2 |
      | In Progress | COL3 |
      | Done        | COL4 |
    When I run "backlog config show -f json"
    Then the exit code should be 0
    And the JSON output should be valid
    And the JSON output should have "workspace.project" equal to "1"

  @github @projects
  Scenario: Connect to repository with invalid project returns error
    Given the mock GitHub API has no project with ID 999
    And a config file with the following content:
      """
      version: 1
      defaults:
        workspace: github
      workspaces:
        github:
          backend: github
          repo: test-owner/test-repo
          project: 999
          api_key_env: GITHUB_TOKEN
          default: true
      """
    When I run "backlog config health"
    Then the exit code should be 1
    And stderr should contain "project"

  @github @projects
  Scenario: Health check without project configured still works
    # When no project is configured, the backend should work with issues only
    When I run "backlog config health"
    Then the exit code should be 0
    And stdout should contain "github"
    And stdout should contain "healthy"
