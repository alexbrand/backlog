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

  Scenario: Initialize GitHub backend with auto-detected repo
    Given a git repository with remote "git@github.com:testowner/testrepo.git"
    When I run "backlog init" with input:
      """
      1


      """
    Then the exit code should be 0
    And stdout should contain "Detected GitHub repository: testowner/testrepo"
    And the file ".backlog/config.yaml" should contain "backend: github"
    And the file ".backlog/config.yaml" should contain "repo: testowner/testrepo"

  @github
  Scenario: Initialize GitHub backend with existing projects
    Given the environment variable "GITHUB_TOKEN" is "ghp_valid_test_token"
    And a mock GitHub API server is running
    And the mock GitHub API has the following projects:
      | number | title            |
      | 1      | Development      |
      | 2      | Sprint Planning  |
    When I run "backlog init" with input:
      """
      1
      test-owner/test-repo
      2

      """
    Then the exit code should be 0
    And stdout should contain "Found 2 project(s)"
    And stdout should contain "Development (#1)"
    And stdout should contain "Sprint Planning (#2)"
    And the file ".backlog/config.yaml" should contain "project: 2"
    And the file ".backlog/config.yaml" should contain "status_field: Status"

  @github
  Scenario: Initialize GitHub backend creating new project
    Given the environment variable "GITHUB_TOKEN" is "ghp_valid_test_token"
    And a mock GitHub API server is running
    When I run "backlog init" with input:
      """
      1
      test-owner/test-repo
      c
      My Backlog

      """
    Then the exit code should be 0
    And stdout should contain "No projects found"
    And stdout should contain "Creating project"
    And stdout should contain "created (#1)"
    And the file ".backlog/config.yaml" should contain "project: 1"
    And the file ".backlog/config.yaml" should contain "status_field: Status"

  @github
  Scenario: Initialize GitHub backend skipping project setup
    Given the environment variable "GITHUB_TOKEN" is "ghp_valid_test_token"
    And a mock GitHub API server is running
    When I run "backlog init" with input:
      """
      1
      test-owner/test-repo
      s

      """
    Then the exit code should be 0
    And stdout should contain "Skipping GitHub Projects"
    And the file ".backlog/config.yaml" should not contain "project:"

  @github
  Scenario: Initialize GitHub backend with single project suggests default
    Given the environment variable "GITHUB_TOKEN" is "ghp_valid_test_token"
    And a mock GitHub API server is running
    And the mock GitHub API has the following projects:
      | number | title    |
      | 5      | Backlog  |
    When I run "backlog init" with input:
      """
      1
      test-owner/test-repo


      """
    Then the exit code should be 0
    And stdout should contain "Found 1 project(s)"
    And stdout should contain "Backlog (#5)"
    And the file ".backlog/config.yaml" should contain "project: 5"
