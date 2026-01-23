Feature: File Locking
  As an agent using the backlog CLI with file-based locking
  I want robust file locks that prevent concurrent access
  So that multiple agents can safely coordinate work without conflicts

  Background:
    Given a backlog with the following tasks:
      | id    | title               | status | priority | assignee | labels  | agent_id |
      | task1 | Unclaimed task      | todo   | high     |          | feature |          |
      | task2 | Claimed by agent1   | in-progress | medium | alex | bug | agent-1 |

  Scenario: Lock file contains agent and timestamps
    Given the environment variable "BACKLOG_AGENT_ID" is "test-agent"
    When I run "backlog claim task1"
    Then the exit code should be 0
    And a lock file should exist for task "task1"
    And the lock file for task "task1" should contain agent "test-agent"
    And the lock file for task "task1" should have a valid claimed_at timestamp
    And the lock file for task "task1" should have a valid expires_at timestamp
    And the lock file for task "task1" should have expires_at after claimed_at

  Scenario: Stale lock is detected and ignored
    Given task "task1" has a stale lock from agent "old-agent" that expired 1 hour ago
    And the environment variable "BACKLOG_AGENT_ID" is "new-agent"
    When I run "backlog claim task1"
    Then the exit code should be 0
    And stdout should contain "Claimed"
    And the task "task1" should have label "agent:new-agent"
    And the lock file for task "task1" should contain agent "new-agent"

  Scenario: Lock TTL expiry allows reclaim
    Given task "task1" has a lock from agent "original-agent" that expired 5 minutes ago
    And the environment variable "BACKLOG_AGENT_ID" is "reclaiming-agent"
    When I run "backlog claim task1"
    Then the exit code should be 0
    And stdout should contain "Claimed"
    And the task "task1" should have label "agent:reclaiming-agent"
    And the task "task1" should not have label "agent:original-agent"

  Scenario: Concurrent claim attempts - active lock prevents reclaim
    Given task "task1" has an active lock from agent "active-agent"
    And the environment variable "BACKLOG_AGENT_ID" is "competing-agent"
    When I run "backlog claim task1"
    Then the exit code should be 2
    And stderr should contain "already claimed"
