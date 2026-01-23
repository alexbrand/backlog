Feature: Initialization
  As a user of the backlog CLI
  I want to initialize a backlog in my project directory
  So that I can start tracking tasks

  Scenario: Initialize backlog in empty directory
    When I run "backlog init"
    Then the exit code should be 0
    And stdout should contain "Initialized"
    And the directory ".backlog" should exist
    And the directory ".backlog/backlog" should exist
    And the directory ".backlog/todo" should exist
    And the directory ".backlog/in-progress" should exist
    And the directory ".backlog/review" should exist
    And the directory ".backlog/done" should exist
    And the directory ".backlog/.locks" should exist

  Scenario: Initialize backlog in directory with existing files
    Given a file "README.md" with content "# My Project"
    And a file "src/main.go" with content "package main"
    When I run "backlog init"
    Then the exit code should be 0
    And the directory ".backlog" should exist
    And the file "README.md" should exist
    And the file "src/main.go" should exist

  Scenario: Initialize fails if .backlog already exists
    Given a fresh backlog directory
    When I run "backlog init"
    Then the exit code should be 1
    And stderr should contain "already exists"

  Scenario: Initialize creates all status directories
    When I run "backlog init"
    Then the exit code should be 0
    And the directory ".backlog/backlog" should exist
    And the directory ".backlog/todo" should exist
    And the directory ".backlog/in-progress" should exist
    And the directory ".backlog/review" should exist
    And the directory ".backlog/done" should exist

  Scenario: Initialize creates config.yaml
    When I run "backlog init"
    Then the exit code should be 0
    And the file ".backlog/config.yaml" should exist
