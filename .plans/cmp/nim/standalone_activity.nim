## The standalone activity Model
##
## A Temporal activity started directly through `StartActivityExecution`, with no workflow.
## Grounded in `chasm/lib/activity/statemachine.go` and `proto/v1/activity_state.proto`. Reset is
## deferred (like cancellation in the Nexus Model) and not modeled. Heartbeat timeout is not modeled.
## Standalone activities write no history events, so every fact is a status read through
## `DescribeActivityExecution` or a result read through `PollActivityExecution`.
##
## Read from top to bottom: vocabulary -> the two machines -> what they promise -> what the set asks.

import std/options
import umpire
import worker as Worker
from nexus_caller import Timeout, unset, expires

# ── Entities ──────────────────────────────────────────────────────────────────────────────────────

entity activity:
  key: activityId

# ── The input domains ─────────────────────────────────────────────────────────────────────────────
#
# `Timeout` is the Nexus Model's. `failed(retryable: bool)` is one constructor and two classes, as
# `handlerError` is there.

type
  AttemptResultKind* {.pure.} = enum
    completed, failed, canceled

  AttemptResult* = object
    case kind*: AttemptResultKind
    of AttemptResultKind.failed:
      retryable*: bool
    else:
      discard

  Delivery* = enum
    accepted
    notFound

  Control* = enum
    pause
    unpause
    requestCancel
    terminate

finite AttemptResult

const
  completed* = AttemptResult(kind: AttemptResultKind.completed)
  canceled* = AttemptResult(kind: AttemptResultKind.canceled)

proc failed*(retryable: bool): AttemptResult =
  AttemptResult(kind: AttemptResultKind.failed, retryable: retryable)

# ── Actions ───────────────────────────────────────────────────────────────────────────────────────
#
# The caller starts and controls the activity; the worker's poll receives the task and its
# response settles the attempt. The worker stopping is a fault that names no entity.

action start:
  party: caller
  creates: activity
  schema: temporal.api.workflowservice.v1.StartActivityExecutionRequest
  input:
    scheduleToClose: Timeout
    scheduleToStart: Timeout
    startToClose: Timeout

## The worker's poll receives the task (`PollActivityTaskQueue`).
action attemptStart:
  party: worker
  on: activity
  schema: temporal.api.workflowservice.v1.PollActivityTaskQueueResponse

action attemptResult:
  party: worker
  on: activity
  schema: temporal.api.workflowservice.v1.RespondActivityTaskCompletedRequest |
    temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest |
    temporal.api.workflowservice.v1.RespondActivityTaskCanceledRequest
  input:
    result: AttemptResult
  examples:
    failed(retryable = false) -> ApplicationFailure nonRetryable
    failed(retryable = true) -> ApplicationFailure retryable

action control:
  party: caller
  on: activity
  schema: temporal.api.workflowservice.v1.PauseActivityExecutionRequest |
    temporal.api.workflowservice.v1.UnpauseActivityExecutionRequest |
    temporal.api.workflowservice.v1.RequestCancelActivityExecutionRequest |
    temporal.api.workflowservice.v1.TerminateActivityExecutionRequest
  input:
    control: Control
  results: Delivery

action workerStop:
  party: worker

# ── The derived observation ───────────────────────────────────────────────────────────────────────
#
# No history event records an attempt, so the count is read through `DescribeActivityExecution`.

observation attemptCount:
  on: activity
  read: attempt

# ── The product machine ───────────────────────────────────────────────────────────────────────────
#
# What the caller sees through Describe. Unlike the Nexus product machine, this one sees a retry:
# a retryable failure puts the activity back to SCHEDULED in Describe (`TransitionRescheduled`), so
# it is a step here and not something only the protocol adds. Every Property written against it is
# carried to the protocol machine by the refinement declared there.

type
  ProductPhase* = enum
    scheduled
    started
    paused
    cancelRequested
    completed
    failed
    canceled
    terminated
    timedOut

  ProductState* = object
    phase*: ProductPhase

  ProductOutcome* = enum
    accepted
    notFound

  ProductFact* = enum
    statusScheduled
    statusStarted
    statusPaused
    statusCancelRequested
    statusCompleted
    statusFailed
    statusCanceled
    statusTerminated
    statusTimedOut

  ProductStep* = Step[ProductState, ProductOutcome, ProductFact]

finite ProductState

proc productStep(phase: ProductPhase, recorded: ProductFact): seq[ProductStep] =
  @[ProductStep(outcome: accepted, state: ProductState(phase: phase), facts: @[recorded])]

## The five phases the product machine ends on.
proc productTerminal*(state: ProductState): bool =
  state.phase in {ProductPhase.completed, ProductPhase.failed, ProductPhase.canceled,
    terminated, timedOut}

## The worker's poll receives the task. Only a scheduled activity is dispatched.
proc attemptStartStep*(state: ProductState): seq[ProductStep] =
  if state.phase == scheduled: productStep(started, statusStarted) else: @[]

## The worker's response. A retryable failure of a started attempt reschedules it, and Describe
## reads SCHEDULED again; one after a cancel request cancels instead, as `statemachine.go` has
## CANCEL_REQUESTED a source of Canceled. A cancellation is honored only where one was requested.
proc attemptResultStep*(state: ProductState, result: AttemptResult): seq[ProductStep] =
  case state.phase
  of started, cancelRequested:
    case result.kind
    of AttemptResultKind.completed: productStep(ProductPhase.completed, statusCompleted)
    of AttemptResultKind.failed:
      if not result.retryable: productStep(ProductPhase.failed, statusFailed)
      elif state.phase == started: productStep(scheduled, statusScheduled)
      else: productStep(ProductPhase.canceled, statusCanceled)
    of AttemptResultKind.canceled:
      if state.phase == cancelRequested: productStep(ProductPhase.canceled, statusCanceled)
      else: @[]
  of scheduled, paused, ProductPhase.completed, ProductPhase.failed, ProductPhase.canceled,
      terminated, timedOut:
    @[]

## The caller's control requests. After the activity is over every one of them is not found, and
## changes nothing. A repeated cancel request is idempotent and records the status again.
proc controlStep*(state: ProductState, control: Control): seq[ProductStep] =
  if productTerminal(state):
    return @[ProductStep(outcome: notFound, state: state)]
  case control
  of pause:
    if state.phase in {scheduled, started}: productStep(paused, statusPaused) else: @[]
  of unpause:
    if state.phase == paused: productStep(scheduled, statusScheduled) else: @[]
  of requestCancel:
    if state.phase in {scheduled, started, paused, cancelRequested}:
      productStep(cancelRequested, statusCancelRequested)
    else: @[]
  of terminate:
    productStep(terminated, statusTerminated)

## The worker stopping is invisible here, as in the Nexus product machine: a kept state with no
## fact would be read as a stutter by the refinement.
proc workerStopStep*(state: ProductState): seq[ProductStep] = @[]

## One of the activity's deadlines firing. Which deadline is the protocol's account of how.
proc timeoutStep*(state: ProductState): seq[ProductStep] =
  case state.phase
  of scheduled, started, cancelRequested, paused: productStep(timedOut, statusTimedOut)
  of ProductPhase.completed, ProductPhase.failed, ProductPhase.canceled, terminated, timedOut: @[]

machine activityProduct:
  `for`: activity
  state: ProductState
  starts: [scheduled]
  ends: [completed, failed, canceled, terminated, timedOut]
  timers: [timeout]
  evidence:
    statusScheduled: statusScheduled
    statusStarted: statusStarted
    statusPaused: statusPaused
    statusCancelRequested: statusCancelRequested
    statusCompleted: statusCompleted
    statusFailed: statusFailed
    statusCanceled: statusCanceled
    statusTerminated: statusTerminated
    statusTimedOut: statusTimedOut
  steps:
    attemptStart: attemptStartStep
    attemptResult: attemptResultStep
    control: controlStep
    workerStop: workerStopStep
    timeout: timeoutStep

# ── The protocol machine ──────────────────────────────────────────────────────────────────────────
#
# How the server gets there: the retry and backoff the product cannot see, the pause requested of a
# running attempt, the three timers the start request sets, and the attempt count. The machine
# begins before the activity exists: `unstarted` is the "no instance yet" member, and the start
# request is what sets the deadline fields.

type
  Phase* = enum
    unstarted
    scheduled
    backingOff
    started
    paused
    pauseRequested
    cancelRequested
    completed
    failed
    canceled
    terminated
    timedOut

  ## Which timer fired. Describe records it, so a Contract that did not check it would pass a run
  ## that timed out on the wrong deadline.
  TimeoutType* = enum
    scheduleToClose
    scheduleToStart
    startToClose

## The bound the saturating successor keeps the attempt count inside.
const attemptBound* = 2

type
  Attempts* = range[0 .. attemptBound]

  ProtocolState* = object
    phase*: Phase
    attempts*: Attempts
    scheduleToClose*: Timeout
    scheduleToStart*: Timeout
    startToClose*: Timeout

  ProtocolOutcome* = enum
    accepted
    notFound

  ProtocolFactKind* {.pure.} = enum
    statusScheduled, statusStarted, statusPaused, statusCancelRequested, statusCompleted,
    statusFailed, statusCanceled, statusTerminated, statusTimedOut, attemptCount

  ProtocolFact* = object
    case kind*: ProtocolFactKind
    of ProtocolFactKind.statusTimedOut:
      timeoutType*: TimeoutType
    else:
      discard

  ProtocolStep* = Step[ProtocolState, ProtocolOutcome, ProtocolFact]

finite ProtocolState

const
  statusScheduled* = ProtocolFact(kind: ProtocolFactKind.statusScheduled)
  statusStarted* = ProtocolFact(kind: ProtocolFactKind.statusStarted)
  statusPaused* = ProtocolFact(kind: ProtocolFactKind.statusPaused)
  statusCancelRequested* = ProtocolFact(kind: ProtocolFactKind.statusCancelRequested)
  statusCompleted* = ProtocolFact(kind: ProtocolFactKind.statusCompleted)
  statusFailed* = ProtocolFact(kind: ProtocolFactKind.statusFailed)
  statusCanceled* = ProtocolFact(kind: ProtocolFactKind.statusCanceled)
  statusTerminated* = ProtocolFact(kind: ProtocolFactKind.statusTerminated)
  attemptCount* = ProtocolFact(kind: ProtocolFactKind.attemptCount)

proc statusTimedOut*(timeoutType: TimeoutType): ProtocolFact =
  ProtocolFact(kind: ProtocolFactKind.statusTimedOut, timeoutType: timeoutType)

## The five phases the design ends on. A control request after one of them is not found.
proc terminalPhase*(phase: Phase): bool =
  phase in {Phase.completed, Phase.failed, Phase.canceled, Phase.terminated, Phase.timedOut}

## Started and not yet over: the phases the schedule-to-close deadline covers.
proc running*(phase: Phase): bool =
  phase in {Phase.scheduled, backingOff, Phase.started, Phase.paused, pauseRequested,
    Phase.cancelRequested}

proc moves(state: ProtocolState, phase: Phase, recorded: seq[ProtocolFact]): seq[ProtocolStep] =
  var next = state
  next.phase = phase
  @[ProtocolStep(outcome: accepted, state: next, facts: recorded)]

## The start request. It names the activity's three deadlines, and every one of them is a state
## field because whether a timer fires is a question about the activity and not about the request.
proc startStep*(state: ProtocolState,
    scheduleToClose, scheduleToStart, startToClose: Timeout): seq[ProtocolStep] =
  if state.phase != unstarted: return @[]
  @[ProtocolStep(outcome: accepted,
    state: ProtocolState(phase: Phase.scheduled, attempts: 0, scheduleToClose: scheduleToClose,
      scheduleToStart: scheduleToStart, startToClose: startToClose),
    facts: @[statusScheduled])]

## The worker's poll receives the task: the attempt count rises, and Describe reads both the status
## and the count.
proc protocolAttemptStartStep*(state: ProtocolState): seq[ProtocolStep] =
  if state.phase != Phase.scheduled: return @[]
  var next = state
  next.phase = Phase.started
  next.attempts = saturatingSucc(state.attempts)
  @[ProtocolStep(outcome: accepted, state: next, facts: @[statusStarted, attemptCount])]

## The worker's response, from each of the three phases an attempt can be running in. A retryable
## failure backs a started attempt off; in `cancelRequested` it cancels, faithful to
## `statemachine.go`, where CANCEL_REQUESTED is a source of Canceled; and in `pauseRequested` it
## pauses (`TransitionAttemptFailedWhilePauseRequested`). A cancellation the caller never requested
## has no row.
proc protocolAttemptResultStep*(state: ProtocolState, result: AttemptResult): seq[ProtocolStep] =
  case state.phase
  of Phase.started:
    case result.kind
    of AttemptResultKind.completed: moves(state, Phase.completed, @[statusCompleted])
    of AttemptResultKind.failed:
      if result.retryable: moves(state, backingOff, @[attemptCount])
      else: moves(state, Phase.failed, @[statusFailed])
    of AttemptResultKind.canceled: @[]
  of Phase.cancelRequested:
    case result.kind
    of AttemptResultKind.completed: moves(state, Phase.completed, @[statusCompleted])
    of AttemptResultKind.failed:
      if result.retryable: moves(state, Phase.canceled, @[statusCanceled])
      else: moves(state, Phase.failed, @[statusFailed])
    of AttemptResultKind.canceled: moves(state, Phase.canceled, @[statusCanceled])
  of pauseRequested:
    case result.kind
    of AttemptResultKind.completed: moves(state, Phase.completed, @[statusCompleted])
    of AttemptResultKind.failed:
      if result.retryable: moves(state, Phase.paused, @[statusPaused])
      else: moves(state, Phase.failed, @[statusFailed])
    of AttemptResultKind.canceled: @[]
  of unstarted, Phase.scheduled, backingOff, Phase.paused, Phase.completed, Phase.failed,
      Phase.canceled, Phase.terminated, Phase.timedOut:
    @[]

## The caller's control requests. Pausing a running attempt only requests the pause; Describe reads
## PAUSE_REQUESTED, and the fact is `statusPaused` for both because the product cannot tell them
## apart. A cancel or terminate after the activity is over is not found.
proc protocolControlStep*(state: ProtocolState, control: Control): seq[ProtocolStep] =
  case control
  of pause:
    case state.phase
    of Phase.scheduled, backingOff: moves(state, Phase.paused, @[statusPaused])
    of Phase.started: moves(state, pauseRequested, @[statusPaused])
    else: @[]
  of unpause:
    case state.phase
    of Phase.paused: moves(state, Phase.scheduled, @[statusScheduled])
    of pauseRequested: moves(state, Phase.started, @[statusStarted])
    else: @[]
  of requestCancel:
    if terminalPhase(state.phase): @[ProtocolStep(outcome: notFound, state: state)]
    elif state.phase == unstarted: @[]
    else: moves(state, Phase.cancelRequested, @[statusCancelRequested])
  of terminate:
    if terminalPhase(state.phase): @[ProtocolStep(outcome: notFound, state: state)]
    elif state.phase == unstarted: @[]
    else: moves(state, Phase.terminated, @[statusTerminated])

## The worker stopping keeps the state and records nothing; a step after it confirms it.
proc protocolWorkerStopStep*(state: ProtocolState): seq[ProtocolStep] =
  @[ProtocolStep(outcome: accepted, state: state)]

## The backoff timer returns a backed-off activity to the queue and records nothing.
proc backoffStep*(state: ProtocolState): seq[ProtocolStep] =
  if state.phase != backingOff: return @[]
  moves(state, Phase.scheduled, @[])

## The schedule-to-close deadline covers the whole activity, paused or not.
proc scheduleToCloseStep*(state: ProtocolState): seq[ProtocolStep] =
  if running(state.phase) and state.scheduleToClose == expires:
    moves(state, Phase.timedOut, @[statusTimedOut(scheduleToClose)])
  else: @[]

## The schedule-to-start deadline covers the wait for a worker, so it stops at the start.
proc scheduleToStartStep*(state: ProtocolState): seq[ProtocolStep] =
  if state.phase in {Phase.scheduled, backingOff} and state.scheduleToStart == expires:
    moves(state, Phase.timedOut, @[statusTimedOut(scheduleToStart)])
  else: @[]

## The start-to-close deadline covers one attempt, whatever the caller requested of it meanwhile.
proc startToCloseStep*(state: ProtocolState): seq[ProtocolStep] =
  if state.phase in {Phase.started, pauseRequested, Phase.cancelRequested} and
      state.startToClose == expires:
    moves(state, Phase.timedOut, @[statusTimedOut(startToClose)])
  else: @[]

## How a protocol state reads as a product state. Backing off is still scheduled; a requested
## pause is still started, because the attempt runs until the worker answers and only then is the
## activity paused; an activity not yet started reads as scheduled, because the product begins
## there.
proc productOf*(state: ProtocolState): ProductState =
  ProductState(phase: (case state.phase
    of unstarted, Phase.scheduled, backingOff: ProductPhase.scheduled
    of Phase.started, pauseRequested: ProductPhase.started
    of Phase.paused: ProductPhase.paused
    of Phase.cancelRequested: ProductPhase.cancelRequested
    of Phase.completed: ProductPhase.completed
    of Phase.failed: ProductPhase.failed
    of Phase.canceled: ProductPhase.canceled
    of Phase.terminated: ProductPhase.terminated
    of Phase.timedOut: ProductPhase.timedOut))

machine activityProtocol:
  `for`: activity
  state: ProtocolState
  refines: activityProduct
  map: productOf
  starts: [unstarted]
  ends: [completed, failed, canceled, terminated, timedOut]
  timers: [backoff, scheduleToClose, scheduleToStart, startToClose]
  unobservable: [backoff]
  evidence:
    statusScheduled: statusScheduled
    statusStarted: statusStarted
    statusPaused: statusPaused
    statusCancelRequested: statusCancelRequested
    statusCompleted: statusCompleted
    statusFailed: statusFailed
    statusCanceled: statusCanceled
    statusTerminated: statusTerminated
    statusTimedOut: statusTimedOut
    attemptCount: attemptCount
  steps:
    start: startStep
    attemptStart: protocolAttemptStartStep
    attemptResult: protocolAttemptResultStep
    control: protocolControlStep
    workerStop: protocolWorkerStopStep
    backoff: backoffStep
    scheduleToClose: scheduleToCloseStep
    scheduleToStart: scheduleToStartStep
    startToClose: startToCloseStep

# ── What the machines promise ─────────────────────────────────────────────────────────────────────
#
# Same-step claims are realized by the functional set; the two transition claims are verified over
# every trace of one path and never realized.

## Once an activity is over, no step changes its phase. Read on the protocol machine through the
## map.
property terminalIsFinal:
  machine: activityProduct
  holds: (before, after) =>
    not productTerminal(before.state) or after.state.phase == before.state.phase

## A completed attempt settles the activity, and Describe reads COMPLETED.
property completes:
  machine: activityProtocol
  `when`: attemptResult(completed)
  holds: step => step.state.phase == Phase.completed and statusCompleted in step.facts

## A non-retryable failure settles the activity as failed.
property nonRetryableFails:
  machine: activityProtocol
  `when`: attemptResult(failed(false))
  holds: step => step.state.phase == Phase.failed and statusFailed in step.facts

## Completed on the second attempt of an activity with no deadline set. A claim fixes one state, so
## every field is named.
const completedOnRetry* = ProtocolState(phase: Phase.completed, attempts: 2,
  scheduleToClose: unset, scheduleToStart: unset, startToClose: unset)

## The retried attempt completes the activity on its second attempt.
property retryCompletes:
  machine: activityProtocol
  `when`: attemptResult(completed)
  holds: step => step.state == completedOnRetry and statusCompleted in step.facts

## A cancel request of a running attempt is recorded and awaits the worker.
property cancelRequestedWhileStarted:
  machine: activityProtocol
  `when`: control(requestCancel)
  holds: step =>
    step.state.phase == Phase.cancelRequested and statusCancelRequested in step.facts

## The worker honors the cancel request, and Describe reads CANCELED.
property canceledByWorker:
  machine: activityProtocol
  `when`: attemptResult(canceled)
  holds: step => step.state.phase == Phase.canceled and statusCanceled in step.facts

# NOTE: `terminated` is also a member of `Phase` and `ProductPhase`; see README, "Where the spec's
# names fight Nim's scope".
## A terminate request settles the activity at once.
property terminated:
  machine: activityProtocol
  `when`: control(terminate)
  holds: step => step.state.phase == Phase.terminated and statusTerminated in step.facts

## A paused activity is never dispatched: no step takes it straight to started.
property pausedIsNotDispatched:
  machine: activityProduct
  holds: (before, after) =>
    before.state.phase != ProductPhase.paused or after.state.phase != ProductPhase.started

## The schedule-to-start deadline settles an activity no worker picked up, and records which
## deadline it was.
property scheduleToStartFires:
  machine: activityProtocol
  `when`: scheduleToStart
  holds: step => step.state.phase == Phase.timedOut and statusTimedOut(scheduleToStart) in step.facts

## The start-to-close deadline settles a started attempt no worker answered.
property startToCloseFires:
  machine: activityProtocol
  `when`: startToClose
  holds: step => step.state.phase == Phase.timedOut and statusTimedOut(startToClose) in step.facts

# ── The paths the Queries run ─────────────────────────────────────────────────────────────────────
#
# Each path is one upstream functional test's shape: the start request with no deadline set unless
# the deadline is what the path is about, then the side effects that settle the activity.

# NOTE: `completed` is also a member of `Phase` and `ProductPhase`; see README.
scenario completed:
  model: activityProtocol
  starts: unstarted
  actions: [start(unset, unset, unset), attemptStart, attemptResult(completed)]

scenario nonRetryable:
  model: activityProtocol
  starts: unstarted
  actions: [start(unset, unset, unset), attemptStart, attemptResult(failed(false))]

## The retryable failure backs the activity off; the backoff timer fires and records nothing; the
## second attempt completes.
scenario retriedThenCompleted:
  model: activityProtocol
  starts: unstarted
  actions: [start(unset, unset, unset), attemptStart, attemptResult(failed(true)), backoff,
    attemptStart, attemptResult(completed)]

scenario cancelRequestedThenCanceled:
  model: activityProtocol
  starts: unstarted
  actions: [start(unset, unset, unset), attemptStart, control(requestCancel),
    attemptResult(canceled)]

## The worker stops, so nothing picks the task up; the caller terminates the scheduled activity.
scenario terminatedWhileScheduled:
  model: activityProtocol
  starts: unstarted
  actions: [start(unset, unset, unset), workerStop, control(terminate)]

## Paused before dispatch, unpaused, then run to completion.
scenario pausedThenCompleted:
  model: activityProtocol
  starts: unstarted
  actions: [start(unset, unset, unset), control(pause), control(unpause), attemptStart,
    attemptResult(completed)]

## The start request sets the schedule-to-start deadline; the worker stops; the deadline fires.
scenario scheduleToStartExpires:
  model: activityProtocol
  starts: unstarted
  actions: [start(unset, expires, unset), workerStop, scheduleToStart]

## The start request sets the start-to-close deadline; the attempt starts and never answers.
scenario startToCloseExpires:
  model: activityProtocol
  starts: unstarted
  actions: [start(unset, unset, expires), attemptStart, startToClose]

limits three:
  steps: 3
  actions: 3
  search: 4096

limits four:
  steps: 4
  actions: 4
  search: 32768

limits six:
  steps: 6
  actions: 6
  search: 262144

# ── The Queries ───────────────────────────────────────────────────────────────────────────────────
#
# Eight finds, one per side effect that settles or redirects the activity, and two verifies over
# the paths whose traces the transition claims are about. Each search runs in the VM at compile
# time.

query completion:
  find: completes
  `in`: completed
  limits: three

query nonRetryableFailure:
  find: nonRetryableFails
  `in`: nonRetryable
  limits: three

query retry:
  find: retryCompletes
  `in`: retriedThenCompleted
  limits: six

query cancel:
  find: canceledByWorker
  `in`: cancelRequestedThenCanceled
  limits: four

# NOTE: `terminate` is also a member of `Control`; see README.
query terminate:
  find: terminated
  `in`: terminatedWhileScheduled
  limits: three

query pauseResume:
  find: completes
  `in`: pausedThenCompleted
  limits: six

query scheduleToStartTimeout:
  find: scheduleToStartFires
  `in`: scheduleToStartExpires
  limits: three

query startToCloseTimeout:
  find: startToCloseFires
  `in`: startToCloseExpires
  limits: three

query terminalHolds:
  verify: terminalIsFinal
  `in`: completed
  limits: three

query pauseHolds:
  verify: pausedIsNotDispatched
  `in`: pausedThenCompleted
  limits: six

# ── The functional set ────────────────────────────────────────────────────────────────────────────
#
# The Case drives the caller and the worker. No repeat over the implementation switch: standalone
# activities are CHASM only.

set standaloneActivityTests:
  purpose: functional
  `bind`:
    caller: driven
    worker: driven
  queries: [completion, nonRetryableFailure, retry, cancel, terminate, pauseResume,
    scheduleToStartTimeout, startToCloseTimeout]

# ── The canary set ────────────────────────────────────────────────────────────────────────────────
#
# A canary against a deployment that runs the worker itself: the worker is `observed`, so the
# verifier reads which result occurred and checks the machine allows it.

set standaloneActivityCanary:
  purpose: canary
  `bind`:
    caller: driven
    worker: observed
  queries: [completion, cancel]

# ── The exploratory set ───────────────────────────────────────────────────────────────────────────

set standaloneActivityExploration:
  purpose: exploratory
  `bind`:
    caller: driven
    worker: driven
  machine: activityProtocol
  cover: rows | results | classMembers
  budget: four

# ── The activity and its worker ───────────────────────────────────────────────────────────────────
#
# The protocol machine's worker stop is a stutter row. Composed with the worker of the activity's
# task queue, the stop is the worker's own phase change and every dispatch is the worker serving,
# so an attempt starts only while the worker polls.

## The caller's view of the activity's worker: it stops and it serves. It never resumes, for the
## reason the Nexus composition gives.
machine activityWorker:
  `from`: Worker.polling
  restrict: [workerStop, serve]

type StandaloneActivityState* = object
  activity*: ProtocolState
  worker*: Worker.WorkerState

finite StandaloneActivityState

compose standaloneActivity:
  `for`: [activity, Worker.worker]
  state: StandaloneActivityState
  members:
    activity: activityProtocol
    worker: activityWorker
  sync:
    workerStop: activity.workerStop || worker.workerStop
    attemptStart: activity.attemptStart || worker.serve
  starts: [activity.unstarted, worker.polling]
  ends: [activity.completed, activity.failed, activity.canceled, activity.terminated,
    activity.timedOut]

## Every dispatch leaves the worker polling: no attempt starts while the worker is stopped.
property startedByPollingWorker:
  machine: standaloneActivity
  `when`: attemptStart
  holds: step => step.state.worker.phase == Worker.polling

## The first attempt is dispatched while the worker polls and fails retryably; the worker stops
## during the backoff, so the retry is never dispatched and the schedule-to-start deadline fires.
## The path performs one `attemptStart`, so the claim is exercised rather than vacuous.
scenario stoppedBeforeRetry:
  model: standaloneActivity
  starts: activity.unstarted
  actions: [activity.start(unset, expires, unset), attemptStart,
    activity.attemptResult(failed(true)), activity.backoff, workerStop, activity.scheduleToStart]

query stoppedWorkerStartsNothing:
  verify: startedByPollingWorker
  `in`: stoppedBeforeRetry
  limits: six
