# Comparison spec: the same two Models in several languages

This directory holds one authoring sample per language for the Umpire model layer. Every sample
implements the same two behavioral Models, described here, so the samples differ only in language
and DSL mechanism. The reference is the real Lean Model at
`temporal/model/Temporal/Feature/Nexus/Caller/Model.lean` (copied to `lean/NexusCaller.lean`).

The Go Testpilot runtime, the protobuf Case format, and realizations are out of scope. A sample
stops at the point where a checked Model, its Properties, Scenarios, Queries and Sets exist and the
finite state table, search, and refinement checks can run. Where the Lean version runs those checks
at compile time, a sample may run them in a test; the README of each sample must say which.

## What every sample must contain

The samples do not need to compile. They illustrate what authoring would look like if the model
layer were written in that language, using the language's best DSL mechanism (builders, macros,
custom syntax, whatever fits). Write them as a fluent expert in the language would, close to real
code, with plausible imports and signatures, but do not spend effort on making a toolchain accept
them. Where a language can push a check to compile time (exhaustiveness, refinement, search
results, token-pinned errors), show how; where it cannot, show where the check runs instead.

Files, per language directory `cmp/<lang>/`:

- `README.md`: the DSL mechanism used and why; which checks happen at compile time, at test time,
  or not at all; what an author's error looks like for a typical mistake (a step function naming
  an undeclared action, a non-exhaustive match, a failed query); toolchain and expected
  edit-to-feedback loop; honest notes on what the language made easy or awkward.
- `umpire.<ext>` (or an `umpire/` directory): the framework surface the samples author against.
  Types and signatures for Step, Machine, finite state enumeration, refinement check, Property,
  Scenario, Limits, Query, Set, Composition, and the DSL entry points (macros, builders, syntax).
  Bodies may be sketched or elided with a comment. Enough that a reader can see how the Model
  files bind to it.
- `nexus_caller.<ext>`: Model 1 below, complete.
- `standalone_activity.<ext>`: Model 2 below, complete.
- a test file with the equivalents of the Lean `#guard` pins listed under "Pins".

Keep the Lean file's section order and the Lean file's comments where they explain semantics
(product versus protocol, stutter rows, why faults are ordinary actions). Comments may be shortened
but not dropped. Names must match this spec exactly so the samples can be compared side by side.

## Framework vocabulary (shared)

- **entity**: what a machine is about. Has an optional `key` (which recorded field names an
  instance) and optional `refer` links to other entities.
- **party**: who performs an action: `caller`, `handler`, `worker`, `network`, `operator`; `system`
  is reserved for timers.
- **action**: a named side effect of a party, optionally `creates` or `on` an entity, with typed
  finite `input` fields, optional `results` (outcome enum), optional `schema` (protobuf message
  name, a string), optional `examples` mapping an input class to a concrete realization value.
- **Step S O F**: `{ outcome : O, state : S, facts : List F }`. A step function
  `S -> inputs -> List (Step S O F)` returns `[]` when the action is not enabled.
- **machine**: `for` entity, `state` type (finite), `starts`, `ends`, `timers` (system actions the
  machine owns), `unobservable` (timers that record nothing), `evidence` (fact -> recorded event
  or observation name), `steps` (action -> step function), optional `refines` + `map` (abstraction
  function to another machine's state) which must be checked: for every protocol transition
  `(s, a, s')`, either `map s == map s'` (a stutter) or the product has some transition from
  `map s` to `map s'` under any action class. The Lean checker matches by mapped states, not by
  action name (a protocol timer row maps to the product's `timeout` row, for example).
- **observation**: a derived read (`on` entity, `read` field) used as evidence with no history event.
- **property**: `machine`, optional `when: action (class)` (same-step claim) and `holds` predicate;
  without `when` it is a transition claim `holds: before after -> Bool`.
- **scenario**: `model`, `starts` (state by phase), `actions` (list of classed actions in order).
- **limits**: `steps`, `actions`, `search`, as the Lean search uses them
  (`Umpire/Search.lean`): the search explores traces up to depth `min(steps, actions)`, and
  `search` is the budget of search nodes visited before it stops with "bound reached". A trace's
  length counts every action, timers included. A find query whose witness is longer than the
  depth is not found; a verify query that hits the node budget is inconclusive, not verified.
- **query**: `find: property in: scenario limits:` (realized by a set) or
  `verify: property in: scenario limits:` (searched, never realized).
- **set**: `purpose: functional | canary | exploratory`, `bind: party -> driven | observed`,
  optional `repeat: implementation` (run once per implementation switch HSM/CHASM), `queries`, or
  for exploratory: `machine`, `cover: rows | results | classMembers`, `budget: <limits>`.
- **compose**: product of machines of different entities with `sync` pairs (two member actions
  fire as one), `starts`, `ends`.

## Model 1: Nexus caller (`nexus_caller`)

Copy the semantics of the Lean file exactly. Summary:

Entities: `workflow`; `operation` (refer `caller: workflow`, key `scheduledEvent`).

Enums: `Timeout = unset | expires`; `Reply = syncSuccess | async | operationFailed |
operationCanceled | handlerError(retryable: Bool)`; `Resolution = succeeded | failed | canceled`;
`Delivery = accepted | notFound`.

Actions:
- `schedule` party caller, creates operation, schema
  `temporal.api.command.v1.ScheduleNexusOperationCommandAttributes`, inputs
  `scheduleToClose, scheduleToStart, startToClose : Timeout`.
- `handlerReply` party handler, on operation, schema
  `temporal.api.nexus.v1.StartOperationResponse | temporal.api.nexus.v1.HandlerError`, input
  `reply : Reply`, examples `handlerError(false) -> BadRequest`, `handlerError(true) -> Internal`.
- `complete` party handler, on operation, input `resolution : Resolution`, results `Delivery`.
- `transportFault` party network, on operation.
- `workerStop` party worker (no entity).

Observation: `pendingAttempts` on operation, read `attempts`.

Product machine `nexusProduct`: `ProductPhase = scheduled | started | succeeded | failed |
canceled | timedOut`; state `{ phase }`; outcome `accepted | notFound`; facts
`nexusOperationScheduled | Started | Completed | Failed | Canceled | TimedOut`.
Steps: handlerReply (only from scheduled: syncSuccess->succeeded+Completed, async->started+Started,
operationFailed->failed+Failed, operationCanceled->canceled+Canceled, handlerError(true)->[] (invisible),
handlerError(false)->failed+Failed); complete (terminal -> notFound, no facts, state kept; else
by resolution); transportFault -> []; workerStop -> []; timeout (scheduled|started ->
timedOut+TimedOut). starts [scheduled]; ends [succeeded, failed, canceled, timedOut]; timers
[timeout].

Protocol machine `nexusProtocol`: `Phase = unscheduled | scheduled | backingOff | started |
succeeded | failed | canceled | timedOut`; `TimeoutType = scheduleToClose | scheduleToStart |
startToClose`; `attemptBound = 2`; state `{ phase, attempts : 0..attemptBound, scheduleToClose,
scheduleToStart, startToClose : Timeout }`; facts as product plus
`nexusOperationTimedOut(timeoutType)` and `pendingAttempts`.
Steps: schedule (only from unscheduled; sets the three deadlines, attempts 0, fact Scheduled);
handlerReply (only from scheduled; as product, except handlerError(true) -> backingOff,
attempts+1 saturating, fact pendingAttempts); transportFault (scheduled -> backingOff, attempts+1,
pendingAttempts); workerStop (stutter: keep state, no facts, always enabled); complete (terminal ->
notFound; unscheduled -> []; else facts = [Started if phase != started] ++ [Completed|Failed|
Canceled] by resolution); backoff timer (backingOff -> scheduled, no facts); scheduleToClose
(running && field expires -> timedOut(scheduleToClose)); scheduleToStart ((scheduled|backingOff)
&& expires -> timedOut(scheduleToStart)); startToClose (started && expires ->
timedOut(startToClose)). `running = scheduled|backingOff|started`. starts [unscheduled]; ends
the four terminal phases; timers [backoff, scheduleToClose, scheduleToStart, startToClose];
unobservable [backoff]; refines nexusProduct via `productOf` (unscheduled|scheduled|backingOff ->
scheduled, others by name).

Properties: `terminalIsFinal` (product, transition: terminal before => same phase after);
`syncSucceeds` (protocol, when handlerReply(syncSuccess): phase succeeded && Completed);
`asyncStarts` (when handlerReply(async): started && Started); `completionSucceeds` (when
complete(succeeded): facts contain Completed); `completionFails` (when complete(failed): Failed);
`handlerErrorFails` (when handlerReply(handlerError false): failed && Failed); `retrySucceeds`
(when handlerReply(syncSuccess): state == {succeeded, attempts 1, all unset} && Completed);
`scheduleToStartFires` (when scheduleToStart: timedOut && TimedOut(scheduleToStart));
`startToCloseFires` (when startToClose: timedOut && TimedOut(startToClose)).

Scenarios (all on nexusProtocol, start unscheduled): `syncReplied` [schedule(unset,unset,unset),
handlerReply(syncSuccess)]; `asyncThenSucceeded` [schedule, handlerReply(async),
complete(succeeded)]; `asyncThenFailed` [..., complete(failed)]; `nonRetryableError` [schedule,
handlerReply(handlerError false)]; `retriedThenSucceeded` [schedule, handlerReply(handlerError
true), backoff, handlerReply(syncSuccess)]; `scheduleToStartExpires` [schedule(unset,expires,unset),
workerStop, scheduleToStart]; `startToCloseExpires` [schedule(unset,unset,expires),
handlerReply(async), startToClose].

Limits: `two` (2,2,512); `three` (3,3,4096); `four` (4,4,32768).

Queries: `syncCompletion` find syncSucceeds in syncReplied two; `asyncCompletion` find
completionSucceeds in asyncThenSucceeded three; `asyncFailure` find completionFails in
asyncThenFailed three; `handlerError` find handlerErrorFails in nonRetryableError two; `retry`
find retrySucceeds in retriedThenSucceeded four; `scheduleToStartTimeout` find
scheduleToStartFires in scheduleToStartExpires three; `startToCloseTimeout` find
startToCloseFires in startToCloseExpires three; `terminalHolds` verify terminalIsFinal in
asyncThenSucceeded three.

Sets: `nexusCallerTests` functional, bind caller driven, handler driven, network observed, worker
driven, repeat implementation, the seven find-queries. `nexusCallerCanary` canary, handler
observed, queries [syncCompletion, asyncCompletion]. `nexusCallerExploration` exploratory, machine
nexusProtocol, cover rows|results|classMembers, budget four.

Composition: worker entity (`Worker` module: `worker` entity key taskQueue; `WorkerState { phase :
polling | stopped }`; actions workerStop, workerResume (party worker, no entity), serve (party
worker, on worker); machine `polling`: stop polling->stopped, resume stopped->polling, serve
polling->polling; starts [polling], ends [polling, stopped]). `handlerWorker = polling restricted
to [workerStop, serve]`. `compose nexusCaller` members operation: nexusProtocol, worker:
handlerWorker; sync workerStop = operation.workerStop || worker.workerStop, handlerReply =
operation.handlerReply || worker.serve; starts [operation.unscheduled, worker.polling]; ends the
operation's four terminals. Property `repliedByPollingWorker` (when handlerReply:
state.worker.phase == polling). Scenario `repliedThenStopped` [operation.schedule(unset,expires,
unset), handlerReply(handlerError true), workerStop, operation.scheduleToStart]. Query
`stoppedWorkerRepliesNothing` verify repliedByPollingWorker in repliedThenStopped four.

## Model 2: standalone activity (`standalone_activity`)

A Temporal activity started directly through `StartActivityExecution`, with no workflow. Grounded
in `chasm/lib/activity/statemachine.go` and `proto/v1/activity_state.proto`. Reset is deferred
(like cancellation in the Nexus Model) and not modeled. Heartbeat timeout is not modeled. Standalone
activities write no history events, so every fact is a status read through
`DescribeActivityExecution` or a result read through `PollActivityExecution`.

Entities: `activity` key `activityId`.

Enums: `Timeout = unset | expires` (reuse); `AttemptResult = completed | failed(retryable: Bool) |
canceled`; `Delivery = accepted | notFound`; `Control = pause | unpause | requestCancel | terminate`.

Actions:
- `start` party caller, creates activity, schema
  `temporal.api.workflowservice.v1.StartActivityExecutionRequest`, inputs `scheduleToClose,
  scheduleToStart, startToClose : Timeout`.
- `attemptStart` party worker, on activity. The worker's poll receives the task
  (`PollActivityTaskQueue`). Schema `temporal.api.workflowservice.v1.PollActivityTaskQueueResponse`.
- `attemptResult` party worker, on activity, input `result : AttemptResult`, schema
  `RespondActivityTaskCompletedRequest | RespondActivityTaskFailedRequest |
  RespondActivityTaskCanceledRequest`, examples `failed(false) -> ApplicationFailure nonRetryable`,
  `failed(true) -> ApplicationFailure retryable`.
- `control` party caller, on activity, input `control : Control`, results `Delivery`, schema
  `PauseActivityExecutionRequest | UnpauseActivityExecutionRequest |
  RequestCancelActivityExecutionRequest | TerminateActivityExecutionRequest`.
- `workerStop` party worker (no entity).

Observation: `attemptCount` on activity, read `attempt` (from DescribeActivityExecution).

Product machine `activityProduct` (what the caller sees through Describe): `ProductPhase =
scheduled | started | paused | cancelRequested | completed | failed | canceled | terminated |
timedOut`; state `{ phase }`; outcome `accepted | notFound`; facts `statusScheduled |
statusStarted | statusPaused | statusCancelRequested | statusCompleted | statusFailed |
statusCanceled | statusTerminated | statusTimedOut`.
Steps:
- attemptStart: scheduled -> started + statusStarted; else [].
- attemptResult: from started or cancelRequested: completed -> completed+statusCompleted;
  failed(false) -> failed+statusFailed; failed(true) from started -> scheduled+statusScheduled
  (unlike Nexus, the retry is visible: Describe reads SCHEDULED again with a higher attempt count,
  matching `TransitionRescheduled`); failed(true) from cancelRequested -> canceled+statusCanceled;
  canceled -> only from cancelRequested: canceled+statusCanceled, else []. Other phases [].
- control: terminal (completed|failed|canceled|terminated|timedOut) -> notFound, keep state, no
  facts. pause: scheduled|started -> paused + statusPaused; else []. unpause: paused -> scheduled +
  statusScheduled; else []. requestCancel: scheduled|started|paused -> cancelRequested +
  statusCancelRequested; cancelRequested -> cancelRequested (idempotent, fact
  statusCancelRequested). terminate: any non-terminal -> terminated + statusTerminated.
- workerStop: [] (invisible).
- timeout (system timer): scheduled|started|cancelRequested|paused -> timedOut + statusTimedOut.
starts [scheduled]; ends [completed, failed, canceled, terminated, timedOut]; timers [timeout].

Protocol machine `activityProtocol`: `Phase = unstarted | scheduled | backingOff | started |
paused | pauseRequested | cancelRequested | completed | failed | canceled | terminated |
timedOut`; `TimeoutType = scheduleToClose | scheduleToStart | startToClose`; `attemptBound = 2`;
state `{ phase, attempts : 0..attemptBound, scheduleToClose, scheduleToStart, startToClose :
Timeout }`; facts as product plus `statusTimedOut(timeoutType)` replacing the untyped one, plus
`attemptCount`.
Steps:
- start: only from unstarted; -> scheduled, attempts 0, deadlines set, fact statusScheduled.
- attemptStart: scheduled -> started, attempts+1 saturating, facts [statusStarted, attemptCount].
- attemptResult from started: completed -> completed+statusCompleted; failed(false) ->
  failed+statusFailed; failed(true) -> backingOff, facts [attemptCount]; canceled -> [] (no cancel
  was requested). From cancelRequested: completed -> completed; failed(false) -> failed;
  failed(true) -> canceled + statusCanceled (in `statemachine.go` CANCEL_REQUESTED is a
  source of Completed, Failed, Canceled, TimedOut and Terminated, not of Rescheduled, so a
  retryable failure under a cancel request settles the activity as canceled); canceled ->
  canceled+statusCanceled. From pauseRequested: completed ->
  completed; failed(false) -> failed; failed(true) -> paused + statusPaused
  (TransitionAttemptFailedWhilePauseRequested); canceled -> []. Other phases [].
- control pause: scheduled|backingOff -> paused + statusPaused (TransitionPaused); started ->
  pauseRequested + statusPaused (the Describe status reads PAUSE_REQUESTED; use fact
  statusPaused for both, as the product cannot tell them apart); else []. unpause: paused ->
  scheduled + statusScheduled; pauseRequested -> started + statusStarted; else [].
  requestCancel: scheduled|backingOff|started|paused|pauseRequested|cancelRequested ->
  cancelRequested + statusCancelRequested; terminal -> notFound; unstarted -> []. terminate:
  non-terminal, not unstarted -> terminated + statusTerminated; terminal -> notFound.
- workerStop: stutter (keep state, no facts, always enabled).
- backoff timer: backingOff -> scheduled, no facts.
- scheduleToClose: running && expires -> timedOut(scheduleToClose) where running = scheduled |
  backingOff | started | paused | pauseRequested | cancelRequested.
- scheduleToStart: (scheduled|backingOff) && expires -> timedOut(scheduleToStart).
- startToClose: (started|pauseRequested|cancelRequested) && expires -> timedOut(startToClose).
starts [unstarted]; ends the five terminal phases; timers [backoff, scheduleToClose,
scheduleToStart, startToClose]; unobservable [backoff]; refines activityProduct via `productOf`:
unstarted|scheduled|backingOff -> scheduled; pauseRequested -> started (the worker still holds the
attempt, so every answer it can give is a product row from started, and the request itself is a
stutter); paused -> paused; others by name.

Properties: `terminalIsFinal` (product, transition); `completes` (when attemptResult(completed):
completed && statusCompleted); `nonRetryableFails` (when attemptResult(failed false): failed &&
statusFailed); `retryCompletes` (when attemptResult(completed): state == {completed, attempts 2,
all unset} && statusCompleted); `cancelRequestedWhileStarted` (when control(requestCancel):
phase cancelRequested && statusCancelRequested); `canceledByWorker` (when attemptResult(canceled):
canceled && statusCanceled); `terminated` (when control(terminate): terminated &&
statusTerminated); `pausedIsNotDispatched` (product, transition claim: before.phase == paused =>
after.phase != started); `scheduleToStartFires`; `startToCloseFires` (as Nexus, with
statusTimedOut(type)).

Scenarios (activityProtocol, start unstarted, `start(unset,unset,unset)` unless said):
`completed` [start, attemptStart, attemptResult(completed)]; `nonRetryable` [start,
attemptStart, attemptResult(failed false)]; `retriedThenCompleted` [start, attemptStart,
attemptResult(failed true), backoff, attemptStart, attemptResult(completed)];
`cancelRequestedThenCanceled` [start, attemptStart, control(requestCancel),
attemptResult(canceled)]; `terminatedWhileScheduled` [start, workerStop, control(terminate)];
`pausedThenCompleted` [start, control(pause), control(unpause), attemptStart,
attemptResult(completed)]; `scheduleToStartExpires` [start(unset,expires,unset), workerStop,
scheduleToStart]; `startToCloseExpires` [start(unset,unset,expires), attemptStart,
startToClose].

Limits: `three` (3,3,4096); `four` (4,4,32768); `six` (6,6,262144).

Queries: `completion` find completes in completed three; `nonRetryableFailure` find
nonRetryableFails in nonRetryable three; `retry` find retryCompletes in retriedThenCompleted six;
`cancel` find canceledByWorker in cancelRequestedThenCanceled four; `terminate` find terminated in
terminatedWhileScheduled three; `pauseResume` find completes in pausedThenCompleted six;
`scheduleToStartTimeout` find scheduleToStartFires in scheduleToStartExpires three;
`startToCloseTimeout` find startToCloseFires in startToCloseExpires three; `terminalHolds` verify
terminalIsFinal in completed three; `pauseHolds` verify pausedIsNotDispatched in
pausedThenCompleted six.

Sets: `standaloneActivityTests` functional, bind caller driven, worker driven, the eight
find-queries, no repeat (standalone activities are CHASM only). `standaloneActivityCanary` canary,
worker observed, queries [completion, cancel]. `standaloneActivityExploration` exploratory,
machine activityProtocol, cover rows|results|classMembers, budget four.

Composition: `activityWorker = polling restricted to [workerStop, serve]`; `compose
standaloneActivity` members activity: activityProtocol, worker: activityWorker; sync workerStop,
attemptStart = activity.attemptStart || worker.serve; starts [activity.unstarted,
worker.polling]; ends the five terminals. Property `startedByPollingWorker` (when attemptStart:
worker polling). Scenario `stoppedBeforeRetry` [activity.start(unset,expires,unset), attemptStart,
activity.attemptResult(failed true), activity.backoff, workerStop, activity.scheduleToStart].
Query `stoppedWorkerStartsNothing` verify startedByPollingWorker in stoppedBeforeRetry six.

## Pins (the test file)

Nexus: nexusProduct has 6 states, 4 ends, 12 action classes; handlerReply(handlerError true) from
scheduled is []; reachable phases from starts are all 6; nexusProtocol has 8 * 3 * 2 * 2 * 2 =
192 states, 4 * 3 * 8 = 96 ends, 8 + 6 + 3 + 1 + 1 + 4 = 23 action classes; the refinement check
passes; every find-query's scenario reaches its property; `terminalHolds` verifies.

Activity: activityProduct has 9 states, 5 ends; activityProtocol has 12 * 3 * 8 = 288 states,
5 * 3 * 8 = 120 ends; attemptResult(canceled) from started is []; the refinement check passes;
every find-query's scenario reaches its property; `terminalHolds` and `pauseHolds` verify.

(An "action class" is one constructor with one assignment of its finite inputs: `schedule` has
8, `handlerReply` 6, `complete` 3, `transportFault` 1, `workerStop` 1, timers 1 each.)

Revision note (2026-09-29): the first version of Model 2 mapped `pauseRequested` to `paused` and
made the product blind to retryable failures, which left three protocol rows with no product
counterpart. The writers caught it. The fix above is on the product side: `pauseRequested` maps to
`started`, and `attemptResult(failed true)` is a visible product row (started -> scheduled,
cancelRequested -> canceled). `pausedIsNotDispatched` still holds over the whole product table.

Second note, same day: the real Lean checker (`Umpire/Command/Refinement.lean`) is stricter than
the rule stated above. Besides mapped states, a matching product row must have the same outcome and
its facts must be among the protocol row's facts (compared by evidence name, which is why a typed
`statusTimedOut(kind)` matches an untyped `statusTimedOut`). Under that rule one Model 2 row needs
one more fact: protocol `attemptResult(failed true)` from `started` records
`[statusScheduled, attemptCount]`, not `[attemptCount]`. The Lean sample carries this; the other
samples implement the mapped-states rule as stated and are unaffected by it.

Third note, 2026-09-30: the original composition scenario `stoppedBeforeDispatch` never performed
`attemptStart`, so `stoppedWorkerStartsNothing` verified a claim that never fired and passed
vacuously. The Lean framework reports this as coverage not exercised, because `requireFiring`
defaults to false. The replacement `stoppedBeforeRetry` mirrors the Nexus `repliedThenStopped`
path: an attempt starts while the worker polls, then the worker stops before the retry.
