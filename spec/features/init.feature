Feature: Initialization
  As a user of the backlog CLI
  I want to initialize a backlog in my project directory
  So that I can start tracking tasks

  Scenario: Initialize backlog with local backend
    When I run "backlog init" with input:
      """
      2
      1

      """
    Then the exit code should be 0
    And stdout should contain "Initializing backlog"
    And stdout should contain "Created .backlog/"
    And stdout should contain "Ready!"
    And the directory ".backlog" should exist
    And the directory ".backlog/backlog" should exist
    And the directory ".backlog/todo" should exist
    And the directory ".backlog/in-progress" should exist
    And the directory ".backlog/review" should exist
    And the directory ".backlog/done" should exist
    And the directory ".backlog/.locks" should exist
    And the file ".backlog/config.yaml" should exist
    And the file ".backlog/config.yaml" should contain "backend: local"

  Scenario: Initialize backlog with GitHub backend
    When I run "backlog init" with input:
      """
      1
      owner/repo

      """
    Then the exit code should be 0
    And stdout should contain "Created .backlog/"
    And the file ".backlog/config.yaml" should exist
    And the file ".backlog/config.yaml" should contain "backend: github"
    And the file ".backlog/config.yaml" should contain "repo: owner/repo"

  Scenario: Initialize in directory with existing files
    Given a file "README.md" with content "# My Project"
    And a file "src/main.go" with content "package main"
    When I run "backlog init" with input:
      """
      2
      1

      """
    Then the exit code should be 0
    And the directory ".backlog" should exist
    And the file "README.md" should exist
    And the file "src/main.go" should exist

  Scenario: Initialize fails if .backlog already exists
    Given a fresh backlog directory
    When I run "backlog init" with input:
      """
      2
      1

      """
    Then the exit code should be 1
    And stderr should contain "already exists"

  Scenario: Initialize creates all status directories
    When I run "backlog init" with input:
      """
      2
      1

      """
    Then the exit code should be 0
    And the directory ".backlog/backlog" should exist
    And the directory ".backlog/todo" should exist
    And the directory ".backlog/in-progress" should exist
    And the directory ".backlog/review" should exist
    And the directory ".backlog/done" should exist

  Scenario: Initialize with git-based locking
    When I run "backlog init" with input:
      """
      2
      2

      """
    Then the exit code should be 0
    And the file ".backlog/config.yaml" should contain "lock_mode: git"

  Scenario: Initialize with agent ID
    When I run "backlog init" with input:
      """
      2
      1
      my-agent
      """
    Then the exit code should be 0
    And the file ".backlog/config.yaml" should contain "agent_id: my-agent"
