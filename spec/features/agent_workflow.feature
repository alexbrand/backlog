Feature: Agent Workflow Integration
  As an AI agent using the backlog CLI
  I want to execute common agent workflows
  So that I can efficiently process tasks from a shared backlog

  # These scenarios test end-to-end agent workflows as described in the PRD.
  # They verify that agents can:
  # 1. Pick up and complete tasks
  # 2. Handle claim conflicts gracefully
  # 3. Release tasks when blocked
  # 4. Partition work by labels

  @agent-workflow
  Scenario: Full agent workflow - next, claim, work, complete
    Given a fresh backlog directory
    And a backlog with the following tasks:
      | id    | title               | status | priority | labels   |
      | task1 | Implement feature A | todo   | high     | backend  |
      | task2 | Implement feature B | todo   | medium   | frontend |
    And the environment variable "BACKLOG_AGENT_ID" is "worker-agent-1"
    # Step 1: Get next task and claim it atomically
    When I run "backlog next --claim -f json"
    Then the exit code should be 0
    And the JSON output should be valid
    And the JSON output should have "id" equal to "task1"
    And the JSON output should have "status" equal to "in-progress"
    And the task "task1" should have label "agent:worker-agent-1"
    # Step 2: Add a work-in-progress comment
    When I run "backlog comment task1 'Starting implementation'"
    Then the exit code should be 0
    And the task "task1" should have comment containing "Starting implementation"
    # Step 3: Complete the task
    When I run "backlog move task1 done --comment='Completed in commit abc123'"
    Then the exit code should be 0
    And the task "task1" should have status "done"
    And the task "task1" should have comment containing "Completed in commit abc123"
    # Step 4: Verify task is no longer available for next
    When I run "backlog next -f id-only"
    Then the exit code should be 0
    And stdout should contain "task2"
    And stdout should not contain "task1"

  @agent-workflow
  Scenario: Agent picks different task when claim fails
    Given a fresh backlog directory
    And a backlog with the following tasks:
      | id    | title               | status      | priority | assignee | labels   | agent_id        |
      | task1 | High priority task  | in-progress | urgent   | user1    | backend  | competing-agent |
      | task2 | Medium priority task| todo        | high     |          | backend  |                 |
      | task3 | Low priority task   | todo        | medium   |          | backend  |                 |
    And the environment variable "BACKLOG_AGENT_ID" is "worker-agent-2"
    # Attempt to claim task already claimed by another agent
    When I run "backlog claim task1"
    Then the exit code should be 2
    And stderr should contain "already claimed"
    # Fall back to next available task
    When I run "backlog next --claim -f json"
    Then the exit code should be 0
    And the JSON output should be valid
    And the JSON output should have "id" equal to "task2"
    And the task "task2" should have label "agent:worker-agent-2"
    And the task "task2" should have status "in-progress"

  @agent-workflow
  Scenario: Agent releases task on failure
    Given a fresh backlog directory
    And a backlog with the following tasks:
      | id    | title               | status      | priority | assignee | labels                  | agent_id     |
      | task1 | External API task   | in-progress | high     | agent    | integration,agent:my-agent | my-agent  |
    And the environment variable "BACKLOG_AGENT_ID" is "my-agent"
    And task "task1" is claimed by agent "my-agent"
    # Agent encounters a blocker and needs to release the task
    When I run "backlog release task1 --comment='Blocked: external API requires credentials'"
    Then the exit code should be 0
    And stdout should contain "Released"
    And the task "task1" should have status "todo"
    And the task "task1" should not have label "agent:my-agent"
    And the task "task1" should have comment containing "Blocked: external API requires credentials"
    # Task is now available for other agents
    When I run "backlog list --assignee=unassigned -f id-only"
    Then the exit code should be 0
    And stdout should contain "task1"

  @agent-workflow
  Scenario: Multi-agent partitioning by labels
    Given a fresh backlog directory
    And a backlog with the following tasks:
      | id    | title               | status | priority | labels          |
      | task1 | Backend API work    | todo   | high     | backend,api     |
      | task2 | Frontend UI work    | todo   | high     | frontend,ui     |
      | task3 | Backend DB work     | todo   | medium   | backend,db      |
      | task4 | Frontend styles     | todo   | medium   | frontend,css    |
    # Backend agent only sees backend tasks
    Given the environment variable "BACKLOG_AGENT_ID" is "backend-agent"
    When I run "backlog next --label=backend -f json"
    Then the exit code should be 0
    And the JSON output should be valid
    And the JSON output should have "id" equal to "task1"
    When I run "backlog next --label=backend --claim"
    Then the exit code should be 0
    And the task "task1" should have label "agent:backend-agent"
    # Frontend agent only sees frontend tasks
    Given the environment variable "BACKLOG_AGENT_ID" is "frontend-agent"
    When I run "backlog next --label=frontend -f json"
    Then the exit code should be 0
    And the JSON output should be valid
    And the JSON output should have "id" equal to "task2"
    When I run "backlog next --label=frontend --claim"
    Then the exit code should be 0
    And the task "task2" should have label "agent:frontend-agent"
    # Verify both agents claimed their respective tasks
    When I run "backlog list --status=in-progress -f json"
    Then the exit code should be 0
    And the JSON output should have "count" equal to "2"
    # Verify remaining tasks are still partitioned correctly
    When I run "backlog list --label=backend --assignee=unassigned -f id-only"
    Then the exit code should be 0
    And stdout should contain "task3"
    And stdout should not contain "task1"
    When I run "backlog list --label=frontend --assignee=unassigned -f id-only"
    Then the exit code should be 0
    And stdout should contain "task4"
    And stdout should not contain "task2"
