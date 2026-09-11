# Workflow start: conflict, reuse and request-ID retries

Design specimen. Nothing here compiles, and no module imports it.

This file tests one question: if a Model's Actions are typed against the real RPC request and
response, can the Model stay small while keeping the precision that request fields such as
`workflow_id_conflict_policy` carry? Every model row below cites the server code that produces its
result, and names the functional test that already asserts it, if one does. The findings at the end
list what the current command syntax cannot express.

Server citations were read at commit `b539005490`. No test was run to produce them.

## Scope

In scope: `WorkflowService.StartWorkflowExecution` on one workflow ID in one active namespace, with
`workflow_id_conflict_policy`, `workflow_id_reuse_policy`, `request_id` and `on_conflict_options`.

Out of scope, each named so a later specimen can pick it up: eager workflow start, cron and start
delay, versioning overrides, cross-cluster failover, the internal parent/child and orphan fields,
`SignalWithStartWorkflowExecution` (different defaults, `validator.go:204-208`) and
`UpdateWithStart` (same starter, `multioperation/api.go:97`).

File abbreviations:

| Short | Path |
| --- | --- |
| `FH` | `service/frontend/workflow_handler.go` |
| `VAL` | `chasm/lib/workflow/validator.go` |
| `DEF` | `common/enums/defaults.go` |
| `API` | `service/history/api/startworkflow/api.go` |
| `DED` | `service/history/api/workflow_id_dedup.go` |
| `MS` | `service/history/workflow/mutable_state_impl.go` |
| `WT` | `tests/workflow_test.go` |

## What the server does

The frontend applies defaults before validating (`FH:631-635`): an unspecified conflict policy
becomes `FAIL`, an unspecified reuse policy becomes `ALLOW_DUPLICATE`, and the deprecated
`TERMINATE_IF_RUNNING` with an unspecified conflict policy becomes `TERMINATE_EXISTING` plus
`ALLOW_DUPLICATE` (`DEF:14-32`). An empty `request_id` is replaced by a server-generated UUID
(`FH:6951-6954`). Validation then rejects two combinations with `InvalidArgument`: an explicit
conflict policy together with `TERMINATE_IF_RUNNING` (`VAL:115-117`), and `TERMINATE_EXISTING`
together with `REJECT_DUPLICATE` (`VAL:119-121`).

History first tries to insert a new current run (`API:212`). When a current run exists, the
conflict handler checks three things in order (`API:359-390`):

1. **Request ID.** If the request ID is recorded on the current run, the request is a retry and no
   policy is consulted. A start request ID returns that run with `started: true` and the run's
   current status, which may be closed (`API:363-368`). An attached request ID returns
   `started: false` (`API:371-378`). Only the current run's IDs are compared (`API:359-360`).
2. **Running current run.** The conflict policy decides (`DED:61-75`). `FAIL` returns
   `WorkflowExecutionAlreadyStarted` (`DED:219-221`). `USE_EXISTING` returns the running run with
   `started: false` (`API:814-823`), and when `on_conflict_options` is present it first writes one
   `WorkflowExecutionOptionsUpdated` event to that run carrying the attached request ID, callbacks
   and links (`API:770-806`, `MS:5966`). `TERMINATE_EXISTING` terminates the running run and starts
   a new one in one transaction (`API:543-579`), with termination reason
   `TerminateIfRunning WorkflowIdReusePolicy` (`DED:347-354`), and returns the new run ID only
   (`API:605-612`).
3. **Closed current run.** The reuse policy decides (`DED:78-89`, `DED:243-257`).
   `ALLOW_DUPLICATE` starts a new run. `ALLOW_DUPLICATE_FAILED_ONLY` starts one only when the
   closed run's status is `FAILED`, `CANCELED`, `TERMINATED` or `TIMED_OUT`
   (`service/history/consts/const.go:131-136`), and otherwise returns
   `WorkflowExecutionAlreadyStarted`. `REJECT_DUPLICATE` always returns
   `WorkflowExecutionAlreadyStarted`.

## Specimen

The syntax follows the respelled command surface (fn-83.15) and the typed Actions discussed with
it. Constructs the current commands do not have are marked in the findings, not here.

```lean
/-- The current run of one workflow ID, as `StartWorkflowExecution` sees it. The two closed
states are the server's own partition (`service/history/consts/const.go:131-136`). -/
enum Status
  | absent
  | running
  | closedSucceeded      -- COMPLETED, CONTINUED_AS_NEW
  | closedUnsucceeded    -- FAILED, CANCELED, TERMINATED, TIMED_OUT

/-- Request IDs are compared, never interpreted, so three symbols stand for any three distinct
UUIDs. -/
enum RequestSlot
  | r1
  | r2
  | r3

action start
  rpc: WorkflowService.StartWorkflowExecution
  vary:
    workflowIdConflictPolicy: [unspecified, fail, useExisting, terminateExisting]
    workflowIdReusePolicy: [unspecified, allowDuplicate, allowDuplicateFailedOnly, rejectDuplicate, terminateIfRunning]
    onConflictOptions: [absent, attachRequestId]
  refer:
    requestId: RequestSlot
    workflowId: workflow
  fixed:
    workflowType.name: "umpire-start"
  normalize:                                                          -- DEF:14-32, FH:631-635
    workflowIdReusePolicy: terminateIfRunning and workflowIdConflictPolicy: unspecified
      → workflowIdConflictPolicy: terminateExisting, workflowIdReusePolicy: allowDuplicate
    workflowIdConflictPolicy: unspecified → fail
    workflowIdReusePolicy: unspecified → allowDuplicate
  outcomes:
    started: response.started = true
    joined: response.started = false
    alreadyStarted: failure WorkflowExecutionAlreadyStarted
    invalid: failure InvalidArgument

action close
  rpc: WorkflowService.TerminateWorkflowExecution
  refer:
    workflowId: workflow
  outcomes:
    terminated: response

model currentRun
  role: workflow
  state:
    status: Status
    startedBy: optional RequestSlot
    attached: set RequestSlot
  actions: [start, close]
  starts: [status: absent]
  steps:
    -- Frontend validation, before any state is read.
    any + start(workflowIdConflictPolicy: not unspecified, workflowIdReusePolicy: terminateIfRunning)
      → unchanged, outcome: invalid
    any + start(workflowIdConflictPolicy: terminateExisting, workflowIdReusePolicy: rejectDuplicate)
      → unchanged, outcome: invalid

    -- No current run.
    status: absent + start(requestId: r)
      → status: running, startedBy: r, attached: [], outcome: started,
        evidence: workflowExecutionStarted

    -- Retries win over every policy.
    startedBy: r + start(requestId: r)
      → unchanged, outcome: started
    attached: contains r + start(requestId: r)
      → unchanged, outcome: joined

    -- Running current run: the conflict policy decides.
    status: running + start(workflowIdConflictPolicy: fail)
      → unchanged, outcome: alreadyStarted
    status: running + start(workflowIdConflictPolicy: useExisting, onConflictOptions: absent)
      → unchanged, outcome: joined
    status: running + start(workflowIdConflictPolicy: useExisting, onConflictOptions: attachRequestId, requestId: r)
      → attached: add r, outcome: joined,
        evidence: workflowExecutionOptionsUpdated
    status: running + start(workflowIdConflictPolicy: terminateExisting, requestId: r)
      → new run with status: running, startedBy: r, attached: [], outcome: started,
        evidence: workflowExecutionTerminated on previous run, workflowExecutionStarted on new run

    -- Closed current run: the reuse policy decides.
    status: closed + start(workflowIdReusePolicy: allowDuplicate, requestId: r)
      → new run with status: running, startedBy: r, attached: [], outcome: started,
        evidence: workflowExecutionStarted on new run
    status: closedUnsucceeded + start(workflowIdReusePolicy: allowDuplicateFailedOnly, requestId: r)
      → new run with status: running, startedBy: r, attached: [], outcome: started,
        evidence: workflowExecutionStarted on new run
    status: closedSucceeded + start(workflowIdReusePolicy: allowDuplicateFailedOnly)
      → unchanged, outcome: alreadyStarted
    status: closed + start(workflowIdReusePolicy: rejectDuplicate)
      → unchanged, outcome: alreadyStarted

    status: running + close
      → status: closedUnsucceeded, outcome: terminated,
        evidence: workflowExecutionTerminated

property retryReturnsTheStartedRun
  when: start
  require:
    outcome: started implies response.runId = current.runId when startedBy = requestId

property joinReturnsTheRunningRun
  when: start
  require:
    outcome: joined implies response.runId = current.runId

property terminateStartsADifferentRun
  when: start(workflowIdConflictPolicy: terminateExisting)
  require:
    response.runId ≠ previous.runId
```

### Row sources and existing tests

| Row | Server source | Functional test |
| --- | --- | --- |
| Validation, `TERMINATE_IF_RUNNING` with a conflict policy | `VAL:115-117` | none; unit test `service/frontend/workflow_handler_test.go:907` |
| Validation, `TERMINATE_EXISTING` with `REJECT_DUPLICATE` | `VAL:119-121` | none; unit test `service/frontend/workflow_handler_test.go:925` |
| No current run | `API:212`, `API:232-238` | `WT:48` "start" |
| Retry of the start request ID | `API:363-368` | `WT:48` "start twice - same request" (`WT:92`) |
| Retry of an attached request ID | `API:371-378` | `WT:475` |
| Running, `FAIL` | `DED:219-221` | `WT:48` "fail when already started" (`WT:145`) |
| Running, `USE_EXISTING` | `API:814-823` | `WT:172` |
| Running, `USE_EXISTING` attaching the request ID | `API:770-806`, `MS:5966` | `WT:226` |
| Running, `TERMINATE_EXISTING` | `API:543-612`, `DED:347-354` | `WT:714` (minimal interval set to 0) |
| Closed, `ALLOW_DUPLICATE` | `DED:243-245` | `WT:814` asserts history size only |
| Closed, `ALLOW_DUPLICATE_FAILED_ONLY` | `DED:246-250`, `const.go:131-136` | none in one cluster; `tests/xdc/failover_test.go:548` |
| Closed, `REJECT_DUPLICATE` | `DED:251-253` | none in one cluster; `tests/xdc/failover_test.go:548` |

## Findings

### What the current syntax cannot express

1. **Structured state.** The current run has a status, the request ID that started it, and the set
   of attached request IDs. A flat `State` enum over three request slots needs
   4 × 4 × 8 = 128 constructors. The Model needs state fields.
2. **Guards and patterns.** Rows match on state fields (`status: closed`, `attached: contains r`)
   and on parameter patterns: a single value, alternatives, `not unspecified`, and omitted fields as
   wildcards. Without wildcards the start Action alone has 4 × 5 × 2 × 3 = 120 instances, and
   nearly every row would be repeated per instance.
3. **Row precedence.** Retries win over every policy, and validation wins over both. Either rows
   are ordered with first-match semantics, or guards are written disjoint and a checker proves they
   are. The current table rejects two rows on the same state and Action.
4. **Request normalization.** Defaults and the deprecated-policy migration apply once per RPC before
   any row. They belong on the Action (`normalize:`), not on every row.
5. **Symbolic references.** `requestId: RequestSlot` and `workflowId: workflow` are compared, never
   interpreted. A Run binds each slot to a fresh runtime value, and Properties compare them
   relationally. Slots and `RunRef` exist in the Program; the Model side has no such binding.
6. **Run identity.** `TERMINATE_EXISTING` and the reuse rows replace the current run. Evidence then
   lands on two runs (`Terminated` on the previous one, `Started` on the new one), and Properties
   compare `response.runId` with `current.runId` and `previous.runId`. The Model has one role with
   one instance and no notion of the run behind it.
7. **Response-typed outcomes.** Outcomes are predicates over the typed response and failure:
   `started`, `runId`, `status`, the failure type. `ActionTemplate` already carries the `Response`
   and `Failure` types; no authoring surface reaches them.
8. **Environment-dependent rows.** With `history.enableWorkflowIdReuseStartTimeValidation` on, a
   start within `history.workflowIdReuseMinimalInterval` (default 1s) of the current run's start
   returns `ResourceExhausted` (`DED:260-282`, `DED:303-324`). The validation flag defaults to off
   (`API:469-473`). A Model row that depends on it needs the dynamic config value as part of its
   setup, bound by the Profile, or a Known Gap. The same holds for the callback limit that can fail
   an attach (`MS:3530-3535`).
9. **Result alternatives from concurrency.** If the running run closes between the conflict check
   and the lock on the `USE_EXISTING` attach path, history evaluates the reuse policy again and may
   start a new run (`API:824-843`). One row therefore has two possible results. The table supports
   several results per row; the command syntax emits exactly one.

### Server behavior worth a second look

These are inferred from reading the code; none was reproduced.

1. **`on_conflict_options` is silently ignored with any policy except `USE_EXISTING`.** Neither the
   frontend nor history rejects it (`API:769-770` is the only reader).
2. **`on_conflict_options` with every flag false still writes an event.** The caller checks only
   that the options are present (`API:770`), and `AddWorkflowExecutionOptionsUpdatedEvent` adds the
   event unconditionally (`MS:5963-5983`).
3. **The attach-path reuse re-check passes a stale status.** After the run is found closed under
   lock, `ResolveWorkflowIDReusePolicy` receives the `RUNNING` status recorded before the lock
   (`API:828-836`). `RUNNING` is not in the failed set, so `ALLOW_DUPLICATE_FAILED_ONLY` returns
   `WorkflowExecutionAlreadyStarted` even when the run actually failed.
4. **History accepts a combination the frontend rejects.** History migrates `TERMINATE_IF_RUNNING`
   regardless of the conflict policy (`DED:387-392`), so a request reaching history directly with
   `TERMINATE_IF_RUNNING` and `FAIL` terminates instead of failing validation.
5. **A unit subtest does not test what its name says.** In
   `service/history/history_engine2_test.go`, the subtest "and id reuse policy ALLOW_DUPLICATE"
   (`:2195`) sends `REJECT_DUPLICATE` (`:2201`).

### What this means for the sets

- **Functional.** Each row is one Query with fixed parameters, for example
  `start(workflowIdConflictPolicy: fail)` on a running run. The rows marked "none" in the table are
  the functional tests this Model would add.
- **Exploratory.** Vary the conflict policy, reuse policy, options and request slots over action
  sequences of length three or four. Findings 2, 3 and 9 above are exactly the kind of result an
  exploratory set should surface on its own.
- **Canary.** Every row is observable through the RPC response and workflow history, so the Model is
  black-box. Rows that depend on dynamic config (finding 8) run only where the Profile pins those
  values.
