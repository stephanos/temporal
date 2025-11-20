# Umpire Violation Interpretation Guide

This guide explains the different types of violations that Umpire can detect and how to interpret and debug them.

## Overview

Umpire is a telemetry-based verification system that observes Temporal system behavior through OpenTelemetry traces and validates invariant properties. When a property is violated, Umpire reports a violation with details to help diagnose the issue.

All violations are automatically checked at the end of each test in `TearDownTest()`, and tests will fail if any violations are detected.

## Property Models

### 1. RetryResilienceModel

**Property**: Workflows complete successfully despite transient failures.

**What it checks**: Workflows that have been started should eventually complete within a reasonable time threshold (default: 30 seconds), even when transient failures occur.

**Violation Message**: `"workflow did not complete within expected time despite retry mechanisms"`

**When it triggers**:
- Workflow is in "started" state (not completed)
- More than 30 seconds have passed since the workflow was started

**Tags provided**:
- `namespace`: Namespace ID of the workflow
- `workflowID`: Workflow ID
- `state`: Current FSM state (should be "started")
- `startedAt`: When the workflow was started (RFC3339 timestamp)
- `lastSeenAt`: Last time an event was seen for this workflow
- `age`: How long the workflow has been running
- `threshold`: The configured threshold (default 30s)

**How to debug**:
1. Check if the workflow is genuinely stuck or just slow
2. Review logs for the workflow execution around the `startedAt` timestamp
3. Check for repeated failures or errors in the matching/history services
4. Verify that retry policies are configured correctly
5. Look for deadlocks or resource exhaustion

**Common causes**:
- Workflow code has an infinite loop or deadlock
- Worker is not processing tasks (crashed or overwhelmed)
- Task queue is misconfigured or unavailable
- Genuine bug that prevents workflow from completing

---

### 2. TaskDeliveryGuaranteesModel

**Property**: Tasks added to task queues are eventually delivered (polled or stored) within a reasonable time.

**What it checks**: Workflow tasks that have been added to a task queue should be either polled by a worker or stored to persistence within a threshold (default: 60 seconds).

**Violation Message**: `"task was not delivered within expected time"`

**When it triggers**:
- Task is in "added" state (added to task queue but not polled or stored)
- More than 60 seconds have passed since the task was added

**Tags provided**:
- `taskQueue`: Name of the task queue
- `workflowID`: Workflow ID
- `runID`: Run ID
- `state`: Current FSM state (should be "added")
- `addedAt`: When the task was added (RFC3339 timestamp)
- `age`: How long the task has been waiting
- `threshold`: The configured threshold (default 60s)

**How to debug**:
1. Check if there are any workers polling the task queue
2. Verify the task queue name matches between workflow and worker
3. Check matching service logs for errors or backlog
4. Look for resource exhaustion (too many tasks, not enough workers)
5. Check if task queue is partitioned correctly

**Common causes**:
- No workers registered for the task queue
- Worker crashed or stopped polling
- Task queue name mismatch (typo)
- Matching service overloaded or experiencing issues
- Task lost due to a bug in matching service

---

### 3. WorkflowLifecycleInvariantsModel

**Property**: Workflows follow correct lifecycle state transitions and maintain timestamp ordering invariants.

**What it checks**: Three key invariants about workflow lifecycle:
1. StartedAt timestamp must be before CompletedAt timestamp
2. LastSeenAt timestamp must be >= StartedAt timestamp
3. Completed workflows should not receive events after completion (with 1s grace period)

**Violation Messages**:

#### a) `"workflow started timestamp is after completed timestamp"`

**When it triggers**:
- Workflow has both StartedAt and CompletedAt set
- StartedAt is chronologically after CompletedAt

**Tags provided**:
- `namespace`: Namespace ID
- `workflowID`: Workflow ID
- `state`: Current FSM state
- `startedAt`: When workflow was started (RFC3339)
- `completedAt`: When workflow was completed (RFC3339)

**How to debug**:
1. This indicates a serious bug - timestamps should never be out of order
2. Check for clock skew between different service nodes
3. Review the trace data to see which service reported each timestamp
4. Look for bugs in how StartedAt/CompletedAt are set

**Common causes**:
- Clock skew between nodes (NTP issues)
- Bug in umpire entity tracking code
- Race condition in event processing

#### b) `"workflow last seen timestamp is before started timestamp"`

**When it triggers**:
- Workflow has both StartedAt and LastSeenAt set
- LastSeenAt is chronologically before StartedAt

**Tags provided**:
- `namespace`: Namespace ID
- `workflowID`: Workflow ID
- `state`: Current FSM state
- `startedAt`: When workflow was started (RFC3339)
- `lastSeenAt`: Last event timestamp (RFC3339)

**How to debug**:
1. This indicates a timestamp ordering bug
2. Check for clock skew issues
3. Verify that LastSeenAt is being updated correctly on all events

**Common causes**:
- Clock skew between nodes
- Bug in LastSeenAt tracking
- Events processed out of order

#### c) `"workflow received events after completion"`

**When it triggers**:
- Workflow is in "completed" state
- LastSeenAt is more than 1 second after CompletedAt

**Tags provided**:
- `namespace`: Namespace ID
- `workflowID`: Workflow ID
- `state`: Current FSM state (should be "completed")
- `completedAt`: When workflow was completed (RFC3339)
- `lastSeenAt`: Last event timestamp (RFC3339)
- `delta`: Time difference between LastSeenAt and CompletedAt

**How to debug**:
1. Check what events are being sent to completed workflows
2. Review frontend/history service code for proper completion checks
3. Look for race conditions where events are in-flight during completion

**Common causes**:
- Race condition: events in-flight when workflow completes
- Client sending signals/queries to completed workflow
- Bug in workflow lifecycle management
- Improper cleanup after workflow completion

---

### 4. LostTaskModel (Pre-existing)

**Property**: Tasks added to matching are eventually polled or marked lost.

**What it checks**: Tasks should be delivered or explicitly marked as lost within 5 minutes.

---

### 5. StuckWorkflowModel (Pre-existing)

**Property**: Workflows don't get stuck without making progress.

**What it checks**: Monitors workflow progress and detects stuck executions.

---

## Rate Limiting

All property models implement rate limiting for violation reports:
- Each unique violation is reported at most once per minute
- Old violation entries are cleaned up after 10 minutes
- This prevents log spam while still alerting on persistent issues

## Configuring Thresholds

Property models have configurable thresholds:

```go
// In model Init() method, you can customize thresholds:
m.Threshold = 60 * time.Second  // TaskDeliveryGuaranteesModel
m.Threshold = 30 * time.Second  // RetryResilienceModel
```

Currently thresholds are hardcoded. In the future, they may be configurable via test setup.

## Disabling Specific Models

To disable a specific model for debugging, you can modify the umpire initialization in `tests/testcore/functional_test_base.go`:

```go
// Initialize umpire with specific models
s.umpire, err = umpire.New(umpire.Config{
    Logger:   s.Logger,
    Registry: s.GetTestCluster().EntityRegistry(),
    Models:   []string{"retryresilience"}, // Only enable specific models
})
```

By default, all registered models are enabled.

## Debugging Workflow

When a violation is detected:

1. **Review the violation details**: Check the message, model name, and tags
2. **Find the relevant logs**: Use the namespace, workflowID, and timestamps to locate relevant logs
3. **Review OTEL traces**: Export and analyze the traces for the time period
4. **Check entity state**: Use the entity registry to inspect entity states
5. **Reproduce locally**: Try to reproduce the issue in a minimal test case

## Best Practices

1. **Don't ignore violations**: They indicate real bugs or system issues
2. **Use pitcher for controlled testing**: Inject faults intentionally to verify resilience
3. **Check timestamps carefully**: Many violations relate to timing and ordering
4. **Monitor resource usage**: Many violations stem from resource exhaustion
5. **Review service logs**: Correlate violations with service-level errors

## Example Violation Output

```
--- FAIL: TestMyWorkflow (30.45s)
    functional_test_base.go:449: Umpire detected 1 violation(s):
    functional_test_base.go:451:   [retryresilience] workflow did not complete within expected time despite retry mechanisms: map[age:31.2s lastSeenAt:2025-01-19T10:30:45Z namespace:default-test-namespace startedAt:2025-01-19T10:30:14Z state:started threshold:30s workflowID:my-workflow-123]
    functional_test_base.go:454:
                Error Trace:    functional_test_base.go:454
                Error:          Should be empty, but was [{retryresilience workflow did not complete within expected time despite retry mechanisms map[age:31.2s ...]}]
                Test:           TestMyWorkflow
                Messages:       Umpire detected 1 violation(s)
```

## Getting Help

If you encounter violations that you don't understand or believe are false positives:

1. Review this guide for the specific violation type
2. Check the relevant property model source code in `tools/umpire/model/`
3. Ask in #temporal-dev or file an issue with full violation details
4. Include relevant logs, traces, and test code for investigation

## Future Improvements

- Configurable thresholds per test
- Severity levels (warning vs. error)
- Violation suppression for known issues
- Integration with continuous monitoring
- Automated root cause analysis hints
