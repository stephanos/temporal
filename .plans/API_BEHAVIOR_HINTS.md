# API behavior hints: inventory and interface

This is fn-118 R1: every wait, poll, retry, interval and timeout that the realizations, the lowering
and the Testpilot Temporal Driver hold, the API fact each one rests on, and the hint that would carry
it. It also records the decisions task fn-118.1 owes the later tasks: which hints are adopted, their
exact IR fields, how the lowering tells a write from a read, and the helper interface the shared
Temporal kit (fn-112.9) must leave open.

Line numbers are at commit `63462b09d4` (source fingerprints at the end). fn-112.9 and fn-114 move
the realization code, so later tasks re-locate a line by the command id this document names.

## Three kinds of wait

Every wait is one of three kinds.

- **(a) Visibility.** A read after a write, where the question is whether the write's effect is
  visible to the read at once or only eventually.
- **(b) Asynchronous cause.** A read, or a Driver await, that waits for something no command of its
  own script does: another script's performed command (an activity worker's answer, a workflow
  task, a Nexus handler's reply), a server timer at a deadline the realization set, or a server
  delivery of a task to a worker. A declared wait bound per kind of cause bounds it.
- **(c) Instruction timeout.** A timeout written on a performed command.

## Inventory

### Realizations

| # | Where | Command / value | Kind | Waits for, and the API fact | Proposed hint |
| --- | --- | --- | --- | --- | --- |
| W-1 | `standaloneactivity/Realization.scala:287-303` `awaitStatus`, interval `:301` | `Poll` of `DescribeActivityExecution` until `info.status` equals a value, every 250 ms; timeout unwritten (Profile default 10,000 ms) | helper | The shared form of W-2 to W-8 | read helper takes the typed read and the condition only (interface below) |
| W-2 | `:379-388` `await-paused` (`when control(pause)`) | W-1 for `PAUSED` | (a) | `PauseActivityExecution` on a `SCHEDULED` activity moves it to `PAUSED` in the same transaction (`chasm/lib/activity/operator_commands.go:312-313`, `statemachine.go:219-229`), and Describe reads the same component (`handler.go:200-203`). Visible at once. The path keeps the activity `SCHEDULED`: on a `STARTED` one the pause is `PAUSE_REQUESTED` and Describe reports `RUNNING` (`statemachine.go:231-239`, `responses.go:26-28`), which is why the script stops the worker (`:305-314`, `:327-332`) | visibility `PauseActivityExecution -> DescribeActivityExecution`, at once: one read |
| W-3 | `:426-435` `await-completed` (`when attemptResult(completed)`) | W-1 for `COMPLETED` | (b) | The activity script's `Finish` (RespondActivityTaskCompleted). The answer applies `COMPLETED` in the same transaction (`service/history/handler.go:425-428`, `chasm/lib/activity/activity.go:463`), so the answer is visible at once, but the controller cannot know when the worker answers | cause bounds `delivery` + `activityAnswer`; visibility `activityAnswer -> DescribeActivityExecution`, at once |
| W-4 | `:436-445` `await-failed` (`when attemptResult(failed(false))`) | W-1 for `FAILED` | (b) | `AttemptFailure` non-retryable (RespondActivityTaskFailed, `handler.go:476-479`, `activity.go:513`), at once | as W-3 |
| W-5 | `:446-455` `await-canceled` (`when attemptResult(canceled)`) | W-1 for `CANCELED` | (b) | `AttemptCanceled` (RespondActivityTaskCanceled, `handler.go:527-530`, `activity.go:533-539`), at once. No Case today | as W-3; not needed by a Case |
| W-6 | `:456-465` `await-terminated` (`when control(terminate)`) | W-1 for `TERMINATED` | (a) | `TerminateActivityExecution` applies `TERMINATED` in the same transaction (`chasm/lib/activity/handler.go:333`, `statemachine.go:161-170`); the API says "Immediate; does not wait for worker acknowledgment" (`enums/v1/activity.pb.go:56`) | visibility `TerminateActivityExecution -> DescribeActivityExecution`, at once: one read |
| W-7 | `:466-475` `await-timed-out` (`when scheduleToClose, scheduleToStart, startToClose`) | W-1 for `TIMED_OUT` | (b) | A server timer at the deadline the start request set (W-12). Timeout tasks fire at or after the deadline from the timer queue (`chasm/lib/activity/tasks.go:154-168`, `:198-216`, `:238-266`; `service/history/timer_queue_active_task_executor.go:1134`); a timer closer than now plus `history.timerProcessorMaxTimeShift` (default 1 s) is pushed out to it (`service/history/shard/task_key_manager.go:100`). No upper bound on timer latency is stated in code | cause bound `timer` = deadline + declared slack |
| W-8 | `:679-688` held race `await-paused` | W-1 for `PAUSED` | (a) | As W-2. The dispatch is held (`:667`), so no worker took the activity | as W-2 |
| W-9 | `nexuscaller/Realization.scala:296-308` `await-scheduled`, interval `:306` | `Poll` of `GetWorkflowExecutionHistory` until a `NexusOperationScheduled` event is present, every 250 ms; timeout unwritten (10,000 ms) | (b) | The workflow script's `WorkflowCommand` (RespondWorkflowTaskCompleted). The scheduled event is never buffered and is persisted before the respond returns (`service/history/historybuilder/event_store.go:333`, `respondworkflowtaskcompleted/api.go:698`), so it is visible at once, but the controller cannot know when the workflow task runs | cause bound `workflowTask`; visibility `workflowTask -> GetWorkflowExecutionHistory`, at once |
| W-10 | `:286-294` `pending-attempts`, interval `:292` (`when handlerReply(handlerError(true))`) | `Poll` of `DescribeWorkflowExecution` until a pending operation has `attempt == 1`, every 250 ms; timeout unwritten (10,000 ms) | (b) and (a) eventual | The handler's retryable error (RespondNexusTaskFailed). Matching hands the reply to the waiting dispatch and returns (`service/matching/matching_engine.go:2897-2909`); history's invocation task records the attempt afterwards (`service/history/hsm/nexusoperations/executors.go:339`, `statemachine.go:278-282`). So the reply is visible to Describe **eventually**. The retry then waits the backoff, 1 s initial with 20 % jitter (`hsm/nexusoperations/config.go:122-131`), which is the window the poll has to see `attempt == 1` before the retried reply succeeds | cause bound `handlerReply`; visibility `handlerReply -> DescribeWorkflowExecution`, eventually within a bound |
| W-11 | `:225-234` `await-close`; `wait_new_event` `:197-200`, close filter `:229-231` | `GetWorkflowExecutionHistory` long poll that resolves when the workflow closes; timeout unwritten (10,000 ms) | (b) | The workflow's close. The server blocks while the workflow runs (`service/history/api/getworkflowexecutionhistory/api.go:231,275`; `get_workflow_util.go:129,245`), at most `history.longPollExpirationInterval` (default 20 s, `common/dynamicconfig/constants.go:1896-1898`), and returns an empty page with a token on expiry | candidate "blocking read": not adopted; keeps its explicit form (see decisions) |
| W-12 | `standaloneactivity/Realization.scala:232` `deadlineSeconds` = 2 s; `:238` `longSeconds` = 300 s (also `:652-653`) | Deadlines the start request sets | server timer | 2 s is the deadline a timeout path expires; 300 s satisfies the server's refusal of a start with no start-to-close or schedule-to-close deadline (`:234-237`) and is never reached | the `timer` bound's deadline term (2 s); 300 s is no wait |
| W-13 | `nexuscaller/Realization.scala:346-355` `deadline` = 2 s | Schedule-to-start / start-to-close of the schedule command | server timer | Waited on by W-15 and W-11, not by a poll | as W-12 |
| W-14 | `:468-471` `await-completion-authority` (`AwaitLearned`) | Waits for the handle an async reply publishes; Profile default 10,000 ms | (b) | The handler's async reply, which the Driver publishes into a slot (`common/testing/testpilot/temporal/server/handle.go:174-207`). No server read | not covered (decisions) |
| W-15 | `:521-530` `await-nexus-operation` (`AwaitCommand`) | Waits for the operation's outcome; Profile default 10,000 ms | (b) | The handler's reply, the controller's completion, a server retry or a deadline. A workflow timer of the resolved timeout (`worker/interpreter.go:117`) | not covered (decisions) |
| W-16 | `:416-417` `replyBinding` (`respond-async`, `respond-sync`, `respond-failed`, `respond-error-retryable`, `respond-error`) | `NexusReply` with `timeoutMs = 5000` | (c) | **Unexplained, and inert.** The Driver never reads the timeout of a handler reply (`worker/interpreter.go:368-408`); a late reply is ended by the 30 s run ceiling, not by this value. It came from the hand-written Lean template (`b52e3b5225`) with no stated reason | none; recommended for deletion in fn-118.5 (owner) |
| W-17 | `:533-542` `finish-workflow`, `:538` | `Finish` with `timeoutMs = 5000` | (c) | **Unexplained, and inert**, as W-16: the Driver never reads the timeout of a workflow `Finish` (`worker/interpreter.go:97-101`) | as W-16 |
| W-18 | `:241-262` `history` | `GetWorkflowExecutionHistory` with `wait_new_event = true`, inherited from `historyAssignments` | none | **Unexplained, and inert.** A first page without the close filter queries from the first event and returns at once (`getworkflowexecutionhistory/api.go:270-273`); it runs after W-11 anyway | none; the flag can go when fn-118.5 touches the file (owner) |
| W-19 | `standaloneactivity/Realization.scala:319-332`, `:400-403`; `nexuscaller/Realization.scala:456-463` | `Fault` worker stop / resume | none | Driver controls, not API calls. Bounded by the Profile default and the SDK's worker stop timeout (`worker/outage.go:217-231`) | none |
| W-20 | `standaloneactivity/Realization.scala:667`, `:692-703`, `:791-798` | `Hold` / `Release` / `admissionResponseLoss` | (b), Driver | The Driver waits until the server's dispatch is held (`temporal/control/delivery.go:61-83`), for admission's decision (`control/activity.go:259-283`) and, for the lost response, for the server's same-request retry (`control/activity.go:194-213`). Profile default 10,000 ms | none: Driver controls are not API reads |

### DSL

| Where | What | Note |
| --- | --- | --- |
| `model/umpire/realize/Realize.scala:371-378`, `:375` | `Command.timeoutMs`, default 0 (the run's default) | the only timeout surface; W-16, W-17 use it |
| `:433-440`, `:439`; `:474-478` | `TypedPoll.intervalMs`, `Instruction.poll(..., intervalMs)`, default 0 | the only interval surface; W-1, W-9, W-10 use it. A 0 interval is refused by the reader |
| `:441`, `:443-444` | `AwaitLearned`, `AwaitCommand` | carry no bound of their own |

### Lowering and reader

| Where | What | Note |
| --- | --- | --- |
| `tools/umpire/lower/realization.go:506-508` | copies a positive `timeout_ms` into `InstructionLimits.timeout_milliseconds` | 0 leaves the Profile default |
| `realization.go:822-853`, `:853` | lowers a `Poll` to `ReadEvidence` with the authored interval | the interval is copied, never derived |
| `lower/internal/producer/build.go:88-104` | `ReadEvidence(..., pollIntervalMilliseconds)`, `TimeoutMilliseconds` | constructors only |
| `tools/umpire/model/validate_realization.go:636-638` | refuses a negative `timeout_ms` | |
| `validate_realization.go:812-814` | refuses an interval below 1 ms | |
| `tools/umpire/lower/lower.go:368-393` | refuses a path that leaves an activity attempt unanswered to run out a deadline | not a wait; the lowering creates no wait here |

### Testpilot Temporal Driver

Paths are under `common/testing/testpilot/`. Infrastructure waits (setup, teardown, cleanup) are
listed for completeness and are out of scope.

| Where | Value | Bounds | Kind |
| --- | --- | --- | --- |
| `temporal/profile.go:83-85` `DefaultInstructionLimits` | 10,000 ms, 1 attempt | every instruction that writes no limit. Stated reason: "the most common timeout and attempts across the checked-in Temporal Cases" (a statistic, not an API fact) | default |
| `temporal/profile.go:96` `DefaultCeilings` | run 30,000 ms, cleanup 20,000 ms | the whole run and the cleanup entrypoint; "the largest value any checked-in Temporal Case declared" | ceiling |
| `tools/canary/casebinding/casebinding.go:84-90`, `:100` | the same values as local, as literals | canary Profile; it authorizes no `InjectFault` or `AwaitSlot`, so W-14, W-19, W-20 never run there | default, ceiling |
| `contract/profile.go:85-94` `InstructionDefaults.Resolve` | Case limit, else default | one resolution rule | resolution |
| `internal/execution/dataflow.go:234-248` | timeout within the run (or cleanup) ceiling | preparation | check |
| `internal/execution/evidence.go:277-282` | 0 < interval <= timeout | preparation of `ReadEvidence` | check |
| `internal/execution/runtime.go:28` | run ceiling 30 s | the ordinary schedule and every reservation wait | ceiling |
| `internal/execution/scheduler.go:649` | resolved timeout | each controller instruction, from dispatch: admission plus its wait; on expiry `TIMED_OUT` (`:737-739`) | (b)/(c) per instruction |
| `internal/execution/scheduler.go:727` | run context | the wait on worker reservations: worker-run instructions are bounded by the run ceiling, not by their own timeout | ceiling |
| `temporal/server/session.go:130-160` `PollRPC`, timer `:148` | the Case's interval | repeats a read whose condition is false; **any RPC error ends the poll** (`:138-139`): no error means "not yet" | (a)/(b) poll |
| `temporal/server/session.go:207-211` | min(timeout, max(run, cleanup)) | the context of each server effect | per instruction |
| `temporal/worker/typed.go:68` | resolved timeout of the schedule command | the Nexus operation's **schedule-to-close timeout** when the command sets none: a server timer set from the Profile default (10,000 ms) | server timer, flagged |
| `temporal/worker/interpreter.go:117` | resolved timeout | `AwaitInstruction`: a workflow timer on the operation future | (b) |
| `temporal/worker/interpreter.go:166,174`; `worker/driver.go:79-82` | 100 ms | an attempt that answers a cancellation heartbeats until the server reports the cancel request (RecordActivityTaskHeartbeat returns `cancel_requested` at once, `chasm/lib/activity/activity.go:623-624`); ends with the attempt's own deadlines | (b), Driver-internal; no Case |
| `temporal/worker/sdk.go:349-364` | session lifetime | long-polls `PollActivityExecution` for the activity's close; an empty answer means poll again | Driver-internal |
| `temporal/delivery_control.go:173-187` | Release context | re-polls `PollActivityTaskQueue` for the held delivery | Driver-internal |
| `temporal/worker/callback.go:66` | run ceiling | HTTP client of the Nexus completion callback | ceiling |
| `temporal/worker/registry.go:121-131,190-198` | run context | waits for a peer group, then reserves again | Driver-internal retry |
| `worker/api.go:23` (5 s), `worker/driver.go:211-216`, `registry.go:234,301`, `outage.go:188`, `session.go:266,447`, `routing.go:343`, `internal/execution/runtime.go:47,57,68,82,95`, `scheduler.go:84,416,498,1017`, `temporal/driver.go:233,392`, `temporal/binding/binding.go:35-36` (30 s, 10 s) | | cleanup, settle, teardown, diagnostics | infrastructure |
| `temporal/provision/provision.go:57-59,167-197` | ready 30 s every 250 ms | namespace cache readiness; `NamespaceNotFound` means not yet | infrastructure, out of scope |

No Driver code configures a gRPC retry policy (`temporal/server/driver.go:72-79`,
`binding.go:89`), and every instruction has one attempt.

## Unexplained waits, for the owner

1. **W-16, W-17: the two `timeoutMs = 5000`.** They bound nothing at runtime and no comment or commit
   states a reason. Recommendation: delete them in fn-118.5 (a listed Program difference: the
   `limits` of `finish-workflow` and the `respond-*` nodes go; no Contract changes). No per-command
   instruction-timeout hint is adopted, because no Case needs one.
2. **`worker/typed.go:68`: the Nexus schedule-to-close timer is the Profile's default instruction
   timeout** when the schedule command sets none, so a Profile change, or fn-118 R5's bound scale
   factor, changes server behavior. Recommendation: the Nexus realization sets schedule-to-close
   explicitly, or the Driver stops deriving it; the owner decides which.
3. **W-18: `wait_new_event` on the closing `history` read** is inert.
4. **W-10 rests on two server facts beyond visibility.** The poll sees `attempt == 1` only inside the
   retry backoff (at least 0.8 s), so its interval must stay well below that. And `attempt`
   increments when an attempt fails only in the HSM implementation, the default
   (`chasm/lib/nexusoperation/config.go:47-49`); with `nexusoperation.enableChasmWorkflowOperations`
   on, it increments at schedule (`chasm/lib/nexusoperation/operation_statemachine.go:22`), and the
   condition would hold before the failure. The API calls the count "approximate"
   (`workflow/v1/message.pb.go:1868-1871`).

## Writes, reads and the order between them

### The classification rule

The lowering tells a write from a read from realization structure and descriptors that exist today,
without the read-only candidate hint:

- A **read** is a `Poll` (its method is its evidence kind's `Recorded.TypedRead`/`TypedSingle`
  method), or an `Rpc` with at least one `ResponseRead` whose method the API binds to HTTP `GET`
  (`google.api.http`, on the method descriptor the lowering already resolves,
  `tools/umpire/lower/descriptor.go:54`).
- A **write** is an `Rpc` the API binds to HTTP `POST`, whatever it reads of its own answer, or a
  performed command the server acts on that is no RPC, identified by its cause kind (table below).
- **Neither:** `Fault`, `Hold`, `Release` (Driver controls), `AwaitLearned`, `AwaitCommand` (Driver
  waits), and a `GET` RPC whose response the Case does not read (`inspect-workflow`, `await-close`).
- An RPC with neither binding is refused by the lowering. None exists today.

The bindings, read from the linked `go.temporal.io/api` (`v1.63.6-0.20260909222256-20151aa90480`)
descriptors with `proto.GetExtension(method.Options(), annotations.E_Http)`:

| Method | Binding |
| --- | --- |
| StartActivityExecution, PauseActivityExecution, UnpauseActivityExecution, TerminateActivityExecution, RequestCancelActivityExecution, StartWorkflowExecution | POST |
| DescribeActivityExecution, GetWorkflowExecutionHistory, DescribeWorkflowExecution | GET |
| RespondActivityTaskCompleted / Failed / Canceled, RecordActivityTaskHeartbeat | POST (worker-side, never a controller RPC) |
| RespondWorkflowTaskCompleted, RespondNexusTaskCompleted / Failed | none (worker-side, never a controller RPC) |

### Non-RPC commands

| Command, in a script of | API method that realizes it | Cause kind |
| --- | --- | --- |
| `Finish`, `AttemptFailure`, `AttemptCanceled` in an activity script | RespondActivityTaskCompleted, RespondActivityTaskFailed, RespondActivityTaskCanceled | `activityAnswer` |
| `WorkflowCommand`, `Finish` in a workflow script | RespondWorkflowTaskCompleted | `workflowTask` |
| `NexusReply` in a Nexus handler script | RespondNexusTaskCompleted, or RespondNexusTaskFailed for a handler error | `handlerReply` |
| `NexusCompletion` in the controller | the completion callback (history's `CompleteNexusOperation`, `service/history/handler.go:2161`); no WorkflowService method | none needed: no read on an existing path follows it before a closing read |
| A step no command performs: `attemptStart` (an activity script's `starts`) | the server's dispatch to the worker | `delivery` |
| A step no command performs: `scheduleToStart`, `startToClose`, `scheduleToClose` | a server timer | `timer` |

The activity Model has no step for a retry's backoff. The second `attemptStart` stands for it, so
the `delivery` bound covers the activity's first backoff (1 s, no jitter:
`common/retrypolicy/retry_policy.go:76-80`, `common/backoff/retry.go:198-208`). The Nexus `backoff`
step is never between a read and what it waits for on an existing path, so no `retry` cause is
adopted.

### Order across scripts

The lowering orders by the path, not by script order. A read waits for the step that records the
fact its evidence kind confirms (`Evidence.records`, matched to the Model's evidence function): for
example `await-completed` waits for `attemptResult(completed)`, `await-scheduled` for `schedule`,
`pending-attempts` for `handlerReply(handlerError(true))`, `await-timed-out` for the timeout step.

- If a write of the read's own script before it performs that step, the pair's visibility decides:
  at once is one read, eventually is a bounded wait with the visibility's bound.
- Otherwise the read waits for causes. Its bound is the sum of the cause bounds of the asynchronous
  steps on the path from its script's last synchronization to that step, plus the visibility's
  bound when the write that performs the step is eventually visible. The interval is the smallest
  among them.
- A script **synchronizes** at a completed read or poll: it observed the step it waited for, and
  every write the server recorded on that execution before it is visible to the script's later
  reads of that execution. Both executions' reads go through the execution's mutable state under
  its lease (`service/history/chasm_engine.go:680-701,1254`;
  `service/history/api/describeworkflow/api.go:76`).
- R3's refusal checks every write between the read's last synchronization and the read, in path
  order across scripts: each needs a declared visibility to the read's method.
- A **closing read** (a command with `closes`) checks no pair: the realization declares that it is
  made after its sources report nothing more, and the explicit blocking read before it (W-11) is
  what makes that true. That is why `history` stays one read after `await-close`.

### Every write->read pair on the existing Cases

| Case | Read | Writes since its script last synchronized (path order) | Waits for | Visibility |
| --- | --- | --- | --- | --- |
| activity-completion | `await-completed` | StartActivityExecution; `activityAnswer` (Finish) | `delivery` + `activityAnswer` | both at once |
| activity-nonRetryableFailure | `await-failed` | StartActivityExecution; `activityAnswer` (AttemptFailure) | `delivery` + `activityAnswer` | both at once |
| activity-pauseResume | `await-paused` | StartActivityExecution; PauseActivityExecution | its own pause | both at once: one read |
| activity-pauseResume | `await-completed` | UnpauseActivityExecution; `activityAnswer` (Finish) | `delivery` + `activityAnswer` | both at once. Unpause applies `SCHEDULED` at once and dispatches through the timer queue (`statemachine.go:247-251`, `applyUnpaused` `:674-677`), which `delivery` covers |
| activity-race-admissionResponseLoss.committed | none | | | |
| activity-race-heldAdmission.staleDelivery | `await-paused` | StartActivityExecution; PauseActivityExecution (`Hold` is neither) | its own pause | both at once: one read |
| activity-retry | `await-completed` | StartActivityExecution; `activityAnswer` (AttemptFailure); `activityAnswer` (Finish) | 2 x `delivery` + 2 x `activityAnswer` | all at once; the failure applies `SCHEDULED` and the attempt count at once (`statemachine.go:107-111,684`) |
| activity-scheduleToStartTimeout | `await-timed-out` | StartActivityExecution | `timer` | at once |
| activity-terminate | `await-terminated` | StartActivityExecution; TerminateActivityExecution | its own terminate | both at once: one read |
| every Nexus Case | `await-scheduled` | StartWorkflowExecution; `workflowTask` (WorkflowCommand) | `workflowTask` | both at once (StartWorkflowExecution persists before it returns) |
| nexus-caller-retry | `pending-attempts` | `handlerReply` (retryable error), after `await-scheduled` | `handlerReply` | **eventually** |
| every Nexus Case | `history` | none: a closing read after `await-close` | | |
| nexus-control-forgedCompletion | `inspect-workflow`, `inspect-workflow-2` | no read: nothing of the response is read | | |

Pairs on realization paths no Case takes today, not declared: RequestCancelActivityExecution ->
DescribeActivityExecution (at once while no attempt runs; `CANCEL_REQUESTED` otherwise,
`operator_commands.go:268-271`); `activityAnswer` (AttemptCanceled) -> DescribeActivityExecution
(at once); RequestCancelActivityExecution -> RecordActivityTaskHeartbeat, read by the Driver's
cancellation heartbeat (at once, `activity.go:623-624`).

## Before-numbers (R8)

Command, from the repository root:

```bash
cat > /tmp/wait-budget.jq <<'JQ'
# fn-118 R8 before-numbers: per Case, the poll instructions and the declared wait budget.
# A wait is an instruction that waits for a condition or a cause: a ReadEvidence poll, an AwaitSlot,
# an AwaitInstruction, a GetWorkflowExecutionHistory long poll filtered to the close event, and any
# instruction that writes its own timeout (an instruction timeout). Its budget is the timeout it
# writes, or the Temporal Profile default (profile.go DefaultInstructionLimits, 10000 ms) when none.
# maxPollCalls is the most RPCs the polls can issue: one, then one per interval until the timeout.
def profileDefault: 10000;
def isLongPoll: (.instruction.invokeRpc.requestAssignments // [])
  | any(.target == "history_event_filter_type"
        and .value.literal.enumValue.name == "HISTORY_EVENT_FILTER_TYPE_CLOSE_EVENT");
[ .program.entrypoints[] as $e | $e.instructions[]?
  | (.instruction | keys[0]) as $kind
  | select($kind == "readEvidence" or $kind == "awaitSlot" or $kind == "awaitInstruction"
           or isLongPoll or .limits.timeoutMilliseconds != null)
  | { at: "\($e.entrypointId)/\(.instructionId)", kind: (if isLongPoll then "longPoll" else $kind end),
      budget: ((.limits.timeoutMilliseconds // profileDefault) | tonumber),
      interval: (.instruction.readEvidence.pollIntervalMilliseconds // null) } ]
| { case: input_filename | split("/") | last | rtrimstr("-case.json"),
    polls: map(select(.kind == "readEvidence")) | length,
    maxPollCalls: map(select(.kind == "readEvidence") | (.budget / (.interval | tonumber) | floor) + 1) | add // 0,
    waits: length,
    budgetMs: map(.budget) | add // 0,
    explicitMs: map(select(.kind == "finish" or .kind == "nexusHandlerReply") | .budget) | add // 0 }
JQ
jq -c -f /tmp/wait-budget.jq model/cases/*-case.json
# The totals row:
jq -c -f /tmp/wait-budget.jq model/cases/*-case.json | jq -s -c '{polls: map(.polls) | add,
  maxPollCalls: map(.maxPollCalls) | add, waits: map(.waits) | add,
  budgetMs: map(.budgetMs) | add, explicitMs: map(.explicitMs) | add}'
# The Case set counted: 4fa0f35ab786b1e54c1f8de86cd12c551646ac1770cd66a0e8811dbf71a799cf
sha256sum model/cases/*-case.json | sha256sum
```

A wait is a `ReadEvidence` poll, an `AwaitSlot`, an `AwaitInstruction`, a `GetWorkflowExecutionHistory`
long poll filtered to the close event, and any instruction that writes its own timeout. Its budget
is the timeout it writes, or the Profile default of 10,000 ms. Results at `63462b09d4`:

| Case | Polls | Most poll RPCs | Waits | Budget (ms) | Of it, explicit 5,000 ms |
| --- | ---: | ---: | ---: | ---: | ---: |
| activity-completion | 1 | 41 | 1 | 10,000 | 0 |
| activity-nonRetryableFailure | 1 | 41 | 1 | 10,000 | 0 |
| activity-pauseResume | 2 | 82 | 2 | 20,000 | 0 |
| activity-race-admissionResponseLoss.committed | 0 | 0 | 0 | 0 | 0 |
| activity-race-heldAdmission.staleDelivery | 1 | 41 | 1 | 10,000 | 0 |
| activity-retry | 1 | 41 | 1 | 10,000 | 0 |
| activity-scheduleToStartTimeout | 1 | 41 | 1 | 10,000 | 0 |
| activity-terminate | 1 | 41 | 1 | 10,000 | 0 |
| nexus-caller-asyncCompletion | 1 | 41 | 6 | 50,000 | 10,000 |
| nexus-caller-asyncFailure | 1 | 41 | 6 | 50,000 | 10,000 |
| nexus-caller-handlerError | 1 | 41 | 5 | 40,000 | 10,000 |
| nexus-caller-retry | 2 | 82 | 7 | 55,000 | 15,000 |
| nexus-caller-scheduleToStartTimeout | 1 | 41 | 4 | 35,000 | 5,000 |
| nexus-caller-startToCloseTimeout | 1 | 41 | 5 | 40,000 | 10,000 |
| nexus-caller-syncCompletion | 1 | 41 | 5 | 40,000 | 10,000 |
| nexus-control-forgedCompletion | 1 | 41 | 6 | 50,000 | 10,000 |
| **Total** | **17** | **697** | **52** | **440,000** | **80,000** |

The budget splits into 17 polls (170,000 ms), 8 close long polls (80,000), 3 slot awaits (30,000),
8 operation awaits (80,000) and 16 explicit 5,000 ms limits (80,000). Every run is also capped at
30,000 ms. When task 5 recounts, a read-once `ReadEvidence` is no poll and has no wait budget; the
script must skip it.

## Decisions

### The three formerly parked questions

1. **Visibility is declared method to method.** In every pair above the write commits all of its
   effect on the execution in one transaction, and the read reads that execution's mutable state, so
   no pair is at once for one field and eventual for another. Whether a status reads `PAUSED` or
   `RUNNING` after a pause depends on the state the path left the activity in, which the Model and
   the realization decide, not on visibility. The one field-level split found is
   DescribeWorkflowExecution's memo and search attributes, which may come from visibility
   (`describeworkflow/api.go:237-241`); no Case reads them.
2. **A bound belongs with the hint, and a Profile only scales it.** A bound states how an API behaves:
   a timer fires after its deadline, a backoff lasts about 1 s, a dispatch goes through matching.
   These differ per cause, which one Profile-wide value cannot say (today's 10,000 ms default is a
   statistic). How much slower an environment is applies to all of them alike, so a Profile carries
   one scale factor, which the Run records. The canary Profile today repeats the local values exactly
   (`casebinding.go:100`).
3. **The Testpilot IR needs a field for the hint's position.** `CaseProvenance.sources` cannot carry
   it: Testpilot reads none of the provenance (`case.proto:18-20`) while the expiry message is part of
   the Run, its rows have no key to an instruction, and the Producer writes every row at line 1
   column 1 (`tools/umpire/lower/internal/producer/producer.go:232-233`). The cost is that a moved
   hint declaration changes the bytes of the Cases whose waits it bounds.

### Adopted hints

| Hint | Adopted declarations | Needed by |
| --- | --- | --- |
| Visibility | StartActivityExecution, PauseActivityExecution, UnpauseActivityExecution, TerminateActivityExecution and `activityAnswer` -> DescribeActivityExecution, at once; StartWorkflowExecution and `workflowTask` -> GetWorkflowExecutionHistory, at once; `handlerReply` -> DescribeWorkflowExecution, eventually | the pairs table |
| Wait bounds per cause kind | `delivery`, `activityAnswer`, `workflowTask`, `handlerReply`, `timer` | every remaining poll |
| Server steps | `attemptStart` is a `delivery`; the three timeout classes of each Model are a `timer` | activity Cases |

Proposed starting values (interval / at most), for task 2 to confirm with the server citations
above: `delivery` 250 / 3,000 ms (matching, the worker's poll, the timer-queue dispatch after an
unpause, the activity's 1 s first backoff); `activityAnswer` 250 / 2,000; `workflowTask` 250 / 5,000;
`handlerReply` 250 / 5,000; `timer` 250 / 3,000 ms of slack after the 2,000 ms deadline its `ServerStep` carries;
`handlerReply -> DescribeWorkflowExecution` eventually 250 / 2,000. With them no derived poll exceeds
today's 10,000 ms, three polls become single reads (both `await-paused`, `await-terminated`), and the
projected poll count is 14.

Not adopted:

| Hint | Why | Case that would need it |
| --- | --- | --- |
| Not-yet errors | No Case tolerates an error: a poll ends on any RPC error (`session.go:138-139`), each start persists before it returns so a read after it finds the execution, and long polls say "nothing yet" with an empty answer, not an error. With no Case there is no removal test, so no refusal can be defined. The only not-yet error in the Driver is `NamespaceNotFound` in provisioning, which is infrastructure | none |
| Instruction timeout per command kind | The two authored timeouts are inert (W-16, W-17) | none |
| `retry` cause | No read waits across a retry step on an existing path; the activity's backoff is inside `delivery` | a Nexus Case that reads after `backoff` |
| Repeatable call | Every instruction has one attempt; the starts already carry `request_id` | none |
| Blocking read | `await-close` writes its long poll in the request; `await-scheduled` could long-poll the same way, and DescribeActivityExecution has a long-poll token (`chasm/lib/activity/handler.go:200-243`) | all eight Nexus Cases (W-11), if its bound were derived; every activity status poll, to wait once |
| Read-only call | The `google.api.http` binding already says which methods read | none |
| Cost of a call | | none |
| Cause bounds for Driver awaits | W-14, W-15 and the Driver control waits read no API; they keep the Profile default (R4 errors clause) | the Nexus Cases' `await-completion-authority` and `await-nexus-operation` |

### Umpire IR fields (task 2)

```proto
message Realization {
  // ... fields 1-14 unchanged ...
  // How the APIs it calls behave between calls, as the shared Temporal kit declares it.
  ApiBehavior behavior = 15;
  // The kind of cause each step class no command performs is.
  repeated ServerStep server_steps = 16;
}

message ApiBehavior {
  repeated Visibility visibility = 1;
  repeated CauseBound causes = 2;
}

// When the effect of a write is visible to a read.
message Visibility {
  // Stable, e.g. "temporal.realize.visibility.pauseActivityExecution.describeActivityExecution".
  string id = 1;
  Position position = 2;
  oneof write {
    // "/package.Service/Method", bound to HTTP POST.
    string method = 3;
    CauseKind cause = 4;
  }
  // "/package.Service/Method", bound to HTTP GET.
  string read = 5;
  // Unset: at once. Set: eventually, within this bound.
  WaitBound eventually = 6;
}

// How long a condition may take to hold, and how often a wait looks.
message WaitBound {
  Position position = 1;
  int64 interval_ms = 2;
  int64 at_most_ms = 3;
}

message CauseBound {
  string id = 1;
  Position position = 2;
  CauseKind kind = 3;
  WaitBound bound = 4;
}

enum CauseKind {
  CAUSE_KIND_UNSPECIFIED = 0;
  // An activity script's answer: RespondActivityTaskCompleted, Failed or Canceled.
  CAUSE_KIND_ACTIVITY_ANSWER = 1;
  // A workflow script's commands: RespondWorkflowTaskCompleted.
  CAUSE_KIND_WORKFLOW_TASK = 2;
  // A Nexus handler script's reply: RespondNexusTaskCompleted or Failed.
  CAUSE_KIND_HANDLER_REPLY = 3;
  // The server's dispatch of a task to a worker.
  CAUSE_KIND_DELIVERY = 4;
  // A server timer at a deadline the realization set.
  CAUSE_KIND_TIMER = 5;
}

message ServerStep {
  Position position = 1;
  ActionClass step = 2;
  CauseKind kind = 3;
  // For a timer: the deadline the realization set for this class, written from the same kit value
  // its request carries. The wait is this deadline plus the `timer` CauseBound, which is the slack.
  int64 deadline_ms = 4;
}
```

The reader refuses an unknown method (the lifter already refuses one it cannot resolve), a
`Visibility` whose write is not POST or whose read is not GET, two declarations of one pair, a
`CauseBound` declared twice for one kind, an interval or bound of zero or less, an interval
greater than its bound, and a timer `ServerStep` with no positive deadline, each at the
declaration's position. Empty fields leave existing IR bytes
and fingerprints unchanged.

### Testpilot IR fields (task 3)

```proto
message ReadEvidence {
  // ... fields 1-5 unchanged ...
  // Reads once: the condition is checked once and a false one fails the instruction.
  // poll_interval_milliseconds is then 0.
  bool once = 6;
}

message InstructionNode {
  // ... fields 1-5 unchanged ...
  // The hints a lowered wait's bound comes from. Set, the node writes its own timeout, which
  // preparation requires to equal the sum of their bounds; no Profile default applies to it.
  repeated WaitHint wait_hints = 6;
}

message WaitHint {
  // The Umpire IR id of the Visibility or CauseBound.
  string hint_id = 1;
  // Where it is declared, with its line.
  SourceLocation source = 2;
  int64 at_most_milliseconds = 3;
}
```

Preparation accepts `once` only with `poll_interval_milliseconds` 0, and a poll only with a
positive interval no greater than its timeout, as today (`evidence.go:277-282`). On expiry the
outcome's detail names the evidence kind and its `until` condition, the node's
timeout, each hint with its source, and the Profile's scale factor when it is not 1.

### Lowering (task 4)

The rules of the section on writes and reads: classify each command; for each read find the step it
waits for; emit one read (`once`) for an at-once write of its own script before it, or a poll whose
timeout is the summed bounds and whose `wait_hints` name them. Refuse, at the read's position: a
write in its window with no declared visibility to its method (naming both, or the cause kind and
the read's method); a step in its window that no command performs and no `ServerStep` names; and a
cause kind with no `CauseBound`. An explicit `Poll` stays accepted only with a recorded reason
(task 4's final rule).

## Helper interface for fn-112.9

fn-112.9 adds no hint field, derived wait or Program change. It shapes the kit so that fn-118.2 can
attach the declarations in one place and fn-118.5 can delete the literals without touching a call
site. What the kit must provide, as signatures; fn-112.9 owns the rest of its helpers:

```scala
// model/umpire/realize: a read is a typed evidence read and a typed condition. No call site writes
// an interval, a timeout or Instruction.poll. Until fn-118.5 the body passes the kit's one interval
// value (250 ms), so Case bytes stay as they are.
def await[Req, Projected](evidence: EvidenceRef[Req, Projected], role: Role)(
    assign: Vector[TypedAssignment[Req, ?]],
    until: Condition[Projected]
): Instruction
```

- Feature read helpers build on `await`, such as the activity realization's
  `awaitStatus(fact, status)` from fn-112's sketch. Its evidence kind keeps the typed method
  (`Recorded.single/read(WorkflowServiceGrpc.METHOD_*, ...)`), so the read method of a visibility
  pair is the evidence kind's method.
- `script`, `perform`, `onPath`, `always` and `command` take no timeout parameter. The Nexus
  timeouts W-16 and W-17 stay in one kit value until fn-118.5 removes them.
- The deadlines the realizations set (W-12, W-13) are kit values that requests refer to by value:
  `val deadlineSeconds: Long = 2` and `val unreachedDeadlineSeconds: Long = 300`. fn-118.2 writes the
  timer `ServerStep` deadlines from the same values.
- Every Temporal realization is built by one kit function, so fn-118.2 adds `behavior` in one place:

```scala
// model/temporal/realize
def temporalRealization(
    name: String,
    machine: Machine[?, ?, ?],
    producer: String,
    scripts: Vector[Script],
    evidence: Vector[Evidence | EvidenceRef[?, ?] | TypedEvidence[?]],
    learned: Vector[Learned] = Vector.empty,
    controls: Vector[umpire.realize.Control] = Vector.empty
): Realization
```

What fn-118.2 then adds (reserved names, so fn-112.9 does not take them):

```scala
// model/umpire/realize
// Milliseconds, as Command.timeoutMs and Poll.intervalMs are today.
final case class WaitBound(intervalMs: Long, atMostMs: Long)
enum Visible:
  case atOnce
  case eventually(bound: WaitBound)
enum CauseKind:
  case activityAnswer, workflowTask, handlerReply, delivery, timer
final class VisibilityHint private[realize] (...)
final class CauseBound private[realize] (...)
final case class ServerStep(step: ClassRef, kind: CauseKind, deadlineMs: Long = 0)
final case class ApiBehavior(visibility: Vector[VisibilityHint], causes: Vector[CauseBound])
extension [Req <: GeneratedMessage, Rsp <: GeneratedMessage](write: MethodDescriptor[Req, Rsp])
  def visibleTo[RReq <: GeneratedMessage, RRsp <: GeneratedMessage](
      read: MethodDescriptor[RReq, RRsp], when: Visible): VisibilityHint
extension (cause: CauseKind)
  def visibleTo[RReq <: GeneratedMessage, RRsp <: GeneratedMessage](
      read: MethodDescriptor[RReq, RRsp], when: Visible): VisibilityHint
  def boundedBy(bound: WaitBound): CauseBound
// Realization gains behavior: ApiBehavior and serverSteps: Vector[ServerStep], both empty by default.

// model/temporal/realize/Behavior.scala: object TemporalBehavior holds the declarations, each with a
// comment citing its server code path.
```

## Source fingerprints

SHA-256 at `63462b09d4`:

| Source | SHA-256 |
| --- | --- |
| `model/umpire/realize/Realize.scala` | `e065b2fb23c8118e02bd0e96a86d3349e8eed8be78004e3704c479bb0879e524` |
| `model/temporal/standaloneactivity/Realization.scala` | `a258bea4b7ed6c8ba0099f23caabc255d6fd71f306adee8fdd44c4e39c343040` |
| `model/temporal/nexuscaller/Realization.scala` | `68bb1d69ed223580645828c1f97c59b7d28ef8b710e4208cdb35c7809d634cd3` |
| `tools/umpire/lower/realization.go` | `6e60f4be551223dde7462b64c6e92996d235496856c519d4e7a4167ee32da459` |
| `tools/umpire/lower/internal/producer/build.go` | `0c46efe791f2f7ede2494a451ecc2e64b169a2fb29defb2fcedf923013684b21` |
| `tools/umpire/model/validate_realization.go` | `94523ed6489f5c2833665a73d59435db03686f5f22c0652f46066dc232f1ae47` |
| `common/testing/testpilot/temporal/profile.go` | `6d018a64de1874f5336beab3f1359bbde1b05a807edebfc1e5de44998ddf9e4c` |
| `common/testing/testpilot/temporal/server/session.go` | `92580973811d76b0a05aaf0512d6ece38ae351356e29bda7948becf2062b2016` |
| `common/testing/testpilot/internal/execution/evidence.go` | `3db7eb89ebe284c98c2877da9c8546f10ea24a733af73941dfc4d4ec013a9024` |
| `common/testing/testpilot/internal/execution/scheduler.go` | `ed8412e4eca0dac858c9f19a589c02cf378204e90c506e22e7d3b834929c2afe` |

`.plans/umpire-api-wait-inventory.md`, the earlier snapshot, is superseded by this document.
