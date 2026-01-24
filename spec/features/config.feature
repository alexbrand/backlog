Feature: Configuration
  As a user of the backlog CLI
  I want to configure workspaces and settings
  So that I can customize the tool for my workflow

  Scenario: Config show displays current configuration
    Given a fresh backlog directory
    And a config file with the following content:
      """
      version: 1
      defaults:
        format: table
        workspace: local
        agent_id: test-agent
      workspaces:
        local:
          backend: local
          path: ./.backlog
          default: true
      """
    When I run "backlog config show"
    Then the exit code should be 0
    And stdout should contain "version: 1"
    And stdout should contain "workspace: local"
    And stdout should contain "backend: local"

  Scenario: Uses default workspace when not specified
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
    When I run "backlog list -f json"
    Then the exit code should be 0
    And the JSON output should be valid
    And the JSON output should have "tasks[0].id" equal to "task1"

  Scenario: Workspace flag overrides default
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

  Scenario: Missing config file uses defaults
    Given a fresh backlog directory
    And the config file is removed
    And a backlog with the following tasks:
      | id    | title           | status | priority |
      | task1 | Default task    | todo   | high     |
    When I run "backlog list -f json"
    Then the exit code should be 0
    And the JSON output should be valid
    And the JSON output should have "tasks[0].id" equal to "task1"

  Scenario: Invalid config file returns exit code 4
    Given a fresh backlog directory
    And a config file with the following content:
      """
      this is not valid yaml: [
      """
    When I run "backlog list"
    Then the exit code should be 4
    And stderr should contain "config"

  Scenario: Config init creates new configuration file
    Given a fresh backlog directory
    And HOME is set to the test directory
    When I run "backlog config init" with input:
      """
      myworkspace
      2
      ./.backlog
      1

      """
    Then the exit code should be 0
    And stdout should contain "Configuration saved to"
    And the file ".config/backlog/config.yaml" should exist
    And the file ".config/backlog/config.yaml" should contain "workspace: myworkspace"
    And the file ".config/backlog/config.yaml" should contain "backend: local"

  Scenario: Config init overwrites existing file when confirmed
    Given a fresh backlog directory
    And HOME is set to the test directory
    And a file ".config/backlog/config.yaml" with content "version: 1"
    When I run "backlog config init" with input:
      """
      newworkspace
      2
      ./.backlog
      1

      y
      """
    Then the exit code should be 0
    And stdout should contain "Configuration saved to"
    And the file ".config/backlog/config.yaml" should contain "workspace: newworkspace"

  Scenario: Config init aborts when overwrite not confirmed
    Given a fresh backlog directory
    And HOME is set to the test directory
    And a file ".config/backlog/config.yaml" with content "version: 1"
    When I run "backlog config init" with input:
      """
      newworkspace
      2
      ./.backlog
      1

      n
      """
    Then the exit code should be 0
    And stdout should contain "Aborted"
    And the file ".config/backlog/config.yaml" should contain "version: 1"

  Scenario: Config init aborts when user presses Enter at overwrite prompt
    Given a fresh backlog directory
    And HOME is set to the test directory
    And a file ".config/backlog/config.yaml" with content "version: 1"
    When I run "backlog config init" with input:
      """
      newworkspace
      2
      ./.backlog
      1


      """
    Then the exit code should be 0
    And stdout should contain "Aborted"
    And the file ".config/backlog/config.yaml" should contain "version: 1"

  Scenario: Config init updates project-local config when it exists
    Given a fresh backlog directory
    And HOME is set to the test directory
    And a config file with the following content:
      """
      version: 1
      defaults:
        workspace: old
      workspaces:
        old:
          backend: local
          path: ./.backlog
          default: true
      """
    When I run "backlog config init" with input:
      """
      newworkspace
      2
      ./.backlog
      1

      y
      """
    Then the exit code should be 0
    And stdout should contain "Configuration saved to"
    # The project-local config should be updated, not the global one
    And the file ".backlog/config.yaml" should contain "workspace: newworkspace"
