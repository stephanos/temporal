-- authoring: header
import Temporal.Case.Syntax
import Temporal.Feature.Worker.Model

/-!
# The standalone activity Model

One activity started directly through `StartActivityExecution`, with no workflow around it: the
product machine says what an activity does as `DescribeActivityExecution` reports it, the protocol
machine says how the server gets there and refines it, and the functional set runs one Query per
side effect that settles the activity. Grounded in `chasm/lib/activity/statemachine.go`; reset is
deferred, like cancellation in the Nexus caller Model, and the heartbeat timeout is not modeled.

A standalone activity writes no history event. Every fact below is a status read through
`DescribeActivityExecution`, or a result read through `PollActivityExecution`, so the evidence lines
of the machines name observations rather than events.

Read from top to bottom: vocabulary → the two machines → what they promise → what the set asks.
-/

namespace Temporal.Feature.Activity.Standalone

open Umpire
open Umpire.Command

-- authoring: entities

/-! ### Entities

An activity is named by the id the caller chose for it: every status read and every result read
carries it, and no run id or event id is needed to tell two apart. -/

entity activity
  key: activityId

-- authoring: domains

/-! ### The input domains

As in the caller Model, a constructor with a finite field contributes one class per assignment:
`failed (retryable : Bool)` is two classes, which mirrors the retryable flag of an
`ApplicationFailure`. -/

enum Timeout
  | unset
  | expires

enum AttemptResult
  | completed
  | failed (retryable : Bool)
  | canceled

enum Delivery
  | accepted
  | notFound

enum Control
  | pause
  | unpause
  | requestCancel
  | terminate

-- authoring: actions

/-! ### Actions

Parties: `caller` starts and controls the activity, `worker` runs its attempts, `system` owns the
timers. The worker's stop is an ordinary action of the `worker` party, as in every other Model. -/

action start
  party: caller
  creates: activity
  schema: temporal.api.workflowservice.v1.StartActivityExecutionRequest
  input:
    scheduleToClose: Timeout
    scheduleToStart: Timeout
    startToClose: Timeout

/-- The worker's poll receives the task for the current attempt. -/
action attemptStart
  party: worker
  on: activity
  schema: temporal.api.workflowservice.v1.PollActivityTaskQueueResponse

action attemptResult
  party: worker
  on: activity
  schema: temporal.api.workflowservice.v1.RespondActivityTaskCompletedRequest |
    temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest |
    temporal.api.workflowservice.v1.RespondActivityTaskCanceledRequest
  input:
    result: AttemptResult
  examples:
    failed (retryable := false) → ApplicationFailureNonRetryable
    failed (retryable := true) → ApplicationFailureRetryable

/-- The four caller-side controls are one action with a finite input, because they share a result:
a control on an activity that is over is not found. -/
action control
  party: caller
  on: activity
  schema: temporal.api.workflowservice.v1.PauseActivityExecutionRequest |
    temporal.api.workflowservice.v1.UnpauseActivityExecutionRequest |
    temporal.api.workflowservice.v1.RequestCancelActivityExecutionRequest |
    temporal.api.workflowservice.v1.TerminateActivityExecutionRequest
  input:
    control: Control
  results: Delivery

/-- The worker stops polling. Nothing recorded names the activity, so the machines keep their state
and record nothing at it. -/
action workerStop
  party: worker

-- authoring: observation

/-! ### The derived observation

A retried attempt writes nothing the caller can see except the attempt count that
`DescribeActivityExecution` reports, so it is the one derived observation. -/

observation attemptCount
  on: activity
  read: attempt

/-- The status reads. The Temporal evidence catalog admits history event kinds, Run Event kinds and
declared observations; a Describe status is none of the first two, so each status a machine records
is declared as an observation of the `status` field. Whether the catalog accepts several
observations over one field is not verified; this is the one place the file departs from checked
grammar into a guess about the realization layer. -/
observation statusScheduled
  on: activity
  read: status

observation statusStarted
  on: activity
  read: status

observation statusPaused
  on: activity
  read: status

observation statusCancelRequested
  on: activity
  read: status

observation statusCompleted
  on: activity
  read: status

observation statusFailed
  on: activity
  read: status

observation statusCanceled
  on: activity
  read: status

observation statusTerminated
  on: activity
  read: status

observation statusTimedOut
  on: activity
  read: status

-- authoring: product

/-! ### The product machine

What the caller sees through `DescribeActivityExecution`, with no account of how: a retry reads
as scheduled again, and a pause requested of a running attempt reads as started until the worker
yields. -/

enum ProductPhase
  | scheduled
  | started
  | paused
  | cancelRequested
  | completed
  | failed
  | canceled
  | terminated
  | timedOut

structure ProductState where
  phase : ProductPhase
  deriving BEq, DecidableEq, Repr, Finite

enum ProductOutcome
  | accepted
  | notFound

enum ProductFact
  | statusScheduled
  | statusStarted
  | statusPaused
  | statusCancelRequested
  | statusCompleted
  | statusFailed
  | statusCanceled
  | statusTerminated
  | statusTimedOut

private def productStep (phase : ProductPhase) (recorded : ProductFact) :
    List (Step ProductState ProductOutcome ProductFact) :=
  [{ outcome := .accepted, state := { phase }, facts := [recorded] }]

/-- The phases the product machine ends on. -/
def productTerminal (state : ProductState) : Bool :=
  state.phase == .completed || state.phase == .failed || state.phase == .canceled ||
    state.phase == .terminated || state.phase == .timedOut

/-- A worker takes the attempt of a scheduled activity. -/
def attemptStartStep (state : ProductState) :
    List (Step ProductState ProductOutcome ProductFact) :=
  if state.phase != .scheduled then [] else productStep .started .statusStarted

/-- The worker's answer to the attempt. Unlike the Nexus caller, a retryable failure is visible
here: `DescribeActivityExecution` reads `SCHEDULED` again with a higher attempt count
(`TransitionRescheduled`), and under a cancel request it settles the activity as canceled. The
backoff between the two is what the protocol machine adds. A canceled answer settles only an
activity whose cancellation was requested. -/
def attemptResultStep (state : ProductState) (result : AttemptResult) :
    List (Step ProductState ProductOutcome ProductFact) :=
  if state.phase != .started && state.phase != .cancelRequested then [] else
  match result with
  | .completed => productStep .completed .statusCompleted
  | .failed false => productStep .failed .statusFailed
  | .failed true =>
      if state.phase == .cancelRequested then productStep .canceled .statusCanceled
      else productStep .scheduled .statusScheduled
  | .canceled =>
      if state.phase == .cancelRequested then productStep .canceled .statusCanceled else []

/-- A control on an activity that is over is not found and changes nothing. -/
def controlStep (state : ProductState) (control : Control) :
    List (Step ProductState ProductOutcome ProductFact) :=
  if productTerminal state then
    [{ outcome := .notFound, state, facts := [] }]
  else
    match control with
    | .pause =>
        if state.phase == .scheduled || state.phase == .started then
          productStep .paused .statusPaused
        else []
    | .unpause =>
        if state.phase == .paused then productStep .scheduled .statusScheduled else []
    | .requestCancel =>
        if state.phase == .scheduled || state.phase == .started || state.phase == .paused ||
            state.phase == .cancelRequested then
          productStep .cancelRequested .statusCancelRequested
        else []
    | .terminate => productStep .terminated .statusTerminated

/-- The worker stopping is a fault the Run records and the activity does not feel. -/
def workerStopStep (_state : ProductState) :
    List (Step ProductState ProductOutcome ProductFact) := []

/-- One of the activity's deadlines firing. Which deadline is the protocol's account of how. -/
def timeoutStep (state : ProductState) :
    List (Step ProductState ProductOutcome ProductFact) :=
  if state.phase == .scheduled || state.phase == .started || state.phase == .cancelRequested ||
      state.phase == .paused then
    productStep .timedOut .statusTimedOut
  else []

machine activityProduct
  for: activity
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

-- authoring: protocol

/-! ### The protocol machine

How the server gets there: the retry the product machine cannot see, the pause a running attempt
turns into a pause request, the three timers the start request sets, and the attempt count. The
machine begins before the activity exists, so `unstarted` is a phase and the start request is what
sets the deadlines. -/

enum Phase
  | unstarted
  | scheduled
  | backingOff
  | started
  | paused
  | pauseRequested
  | cancelRequested
  | completed
  | failed
  | canceled
  | terminated
  | timedOut

enum TimeoutType
  | scheduleToClose
  | scheduleToStart
  | startToClose

abbrev attemptBound : Nat := 2

structure ProtocolState where
  phase : Phase
  attempts : Fin (attemptBound + 1)
  scheduleToClose : Timeout
  scheduleToStart : Timeout
  startToClose : Timeout
  deriving BEq, DecidableEq, Repr, Finite

enum ProtocolOutcome
  | accepted
  | notFound

enum ProtocolFact
  | statusScheduled
  | statusStarted
  | statusPaused
  | statusCancelRequested
  | statusCompleted
  | statusFailed
  | statusCanceled
  | statusTerminated
  | statusTimedOut (timeoutType : TimeoutType)
  | attemptCount

def terminalPhase (phase : Phase) : Bool :=
  phase == .completed || phase == .failed || phase == .canceled || phase == .terminated ||
    phase == .timedOut

/-- Started and not over: the phases a deadline can fire in. -/
def running (phase : Phase) : Bool :=
  phase == .scheduled || phase == .backingOff || phase == .started || phase == .paused ||
    phase == .pauseRequested || phase == .cancelRequested

/-- A worker holds the attempt: the phases a start-to-close deadline covers and a worker's answer
settles. -/
def attemptHeld (phase : Phase) : Bool :=
  phase == .started || phase == .pauseRequested || phase == .cancelRequested

private def moves (state : ProtocolState) (phase : Phase) (recorded : List ProtocolFact) :
    List (Step ProtocolState ProtocolOutcome ProtocolFact) :=
  [{ outcome := .accepted, state := { state with phase }, facts := recorded }]

def startStep (state : ProtocolState) (scheduleToClose scheduleToStart startToClose : Timeout) :
    List (Step ProtocolState ProtocolOutcome ProtocolFact) :=
  if state.phase != .unstarted then [] else
  [{ outcome := .accepted
     state := { phase := .scheduled, attempts := 0, scheduleToClose, scheduleToStart, startToClose }
     facts := [.statusScheduled] }]

/-- The worker's poll takes the attempt and raises the count the caller reads back. -/
def protocolAttemptStartStep (state : ProtocolState) :
    List (Step ProtocolState ProtocolOutcome ProtocolFact) :=
  if state.phase != .scheduled then [] else
  [{ outcome := .accepted
     state := { state with phase := .started, attempts := saturatingSucc state.attempts }
     facts := [.statusStarted, .attemptCount] }]

/-- The worker's answer, by the phase it lands in. A retryable failure backs a started attempt off,
which the caller reads as scheduled again with a higher attempt count, settles a cancel-requested
one as canceled, and lands a pause-requested one in paused
(`TransitionAttemptFailedWhilePauseRequested`). A canceled answer is honored only under a cancel
request. -/
def protocolAttemptResultStep (state : ProtocolState) (result : AttemptResult) :
    List (Step ProtocolState ProtocolOutcome ProtocolFact) :=
  if !attemptHeld state.phase then [] else
  match result with
  | .completed => moves state .completed [.statusCompleted]
  | .failed false => moves state .failed [.statusFailed]
  | .failed true =>
      match state.phase with
      | .cancelRequested => moves state .canceled [.statusCanceled]
      | .pauseRequested => moves state .paused [.statusPaused]
      | _ => moves state .backingOff [.statusScheduled, .attemptCount]
  | .canceled =>
      if state.phase == .cancelRequested then moves state .canceled [.statusCanceled] else []

/-- The caller's controls. A pause of a held attempt is a request the worker learns of on its next
heartbeat, so it is its own phase; the caller reads it as paused either way. -/
def protocolControlStep (state : ProtocolState) (control : Control) :
    List (Step ProtocolState ProtocolOutcome ProtocolFact) :=
  if terminalPhase state.phase then
    [{ outcome := .notFound, state, facts := [] }]
  else if state.phase == .unstarted then []
  else
    match control with
    | .pause =>
        if state.phase == .scheduled || state.phase == .backingOff then
          moves state .paused [.statusPaused]
        else if state.phase == .started then
          moves state .pauseRequested [.statusPaused]
        else []
    | .unpause =>
        if state.phase == .paused then moves state .scheduled [.statusScheduled]
        else if state.phase == .pauseRequested then moves state .started [.statusStarted]
        else []
    | .requestCancel => moves state .cancelRequested [.statusCancelRequested]
    | .terminate => moves state .terminated [.statusTerminated]

/-- The worker stopping keeps the state and records nothing; on a path it is confirmed by the
evidence of the step after it, and the Case says so in a Known Gap. -/
def protocolWorkerStopStep (state : ProtocolState) :
    List (Step ProtocolState ProtocolOutcome ProtocolFact) :=
  [{ outcome := .accepted, state, facts := [] }]

/-- The backoff timer: a retry writes nothing the caller can read. -/
def backoffStep (state : ProtocolState) :
    List (Step ProtocolState ProtocolOutcome ProtocolFact) :=
  if state.phase != .backingOff then [] else moves state .scheduled []

def scheduleToCloseStep (state : ProtocolState) :
    List (Step ProtocolState ProtocolOutcome ProtocolFact) :=
  if running state.phase && state.scheduleToClose == .expires then
    moves state .timedOut [.statusTimedOut (timeoutType := .scheduleToClose)]
  else []

def scheduleToStartStep (state : ProtocolState) :
    List (Step ProtocolState ProtocolOutcome ProtocolFact) :=
  if (state.phase == .scheduled || state.phase == .backingOff) &&
      state.scheduleToStart == .expires then
    moves state .timedOut [.statusTimedOut (timeoutType := .scheduleToStart)]
  else []

def startToCloseStep (state : ProtocolState) :
    List (Step ProtocolState ProtocolOutcome ProtocolFact) :=
  if attemptHeld state.phase && state.startToClose == .expires then
    moves state .timedOut [.statusTimedOut (timeoutType := .startToClose)]
  else []

/-- How a protocol state reads as a product state: not yet started and backing off read as
scheduled; a pause request reads as started, because the worker still holds the attempt and every
answer it can give is a row the product has from started, while the request itself is a stutter;
every other phase is its namesake. -/
def productOf (state : ProtocolState) : ProductState :=
  { phase := match state.phase with
    | .unstarted | .scheduled | .backingOff => .scheduled
    | .started | .pauseRequested => .started
    | .paused => .paused
    | .cancelRequested => .cancelRequested
    | .completed => .completed
    | .failed => .failed
    | .canceled => .canceled
    | .terminated => .terminated
    | .timedOut => .timedOut }

machine activityProtocol
  for: activity
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

-- authoring: properties

/-! ### What the machines promise -/

property terminalIsFinal
  machine: activityProduct
  holds: fun before after =>
    !(productTerminal before.state) || after.state.phase == before.state.phase

/- A paused activity is dispatched to no worker: nothing moves it straight to started. -/
property pausedIsNotDispatched
  machine: activityProduct
  holds: fun before after =>
    before.state.phase != .paused || after.state.phase != .started

property completes
  machine: activityProtocol
  when: attemptResult (completed)
  holds: fun step =>
    step.state.phase == .completed && step.facts.contains .statusCompleted

property nonRetryableFails
  machine: activityProtocol
  when: attemptResult (failed false)
  holds: fun step => step.state.phase == .failed && step.facts.contains .statusFailed

/-- Completed on the second attempt of an activity with no deadline set. -/
def completedOnRetry : ProtocolState :=
  { phase := .completed, attempts := 2, scheduleToClose := .unset, scheduleToStart := .unset,
    startToClose := .unset }

property retryCompletes
  machine: activityProtocol
  when: attemptResult (completed)
  holds: fun step =>
    step.state == completedOnRetry && step.facts.contains .statusCompleted

property cancelRequestedWhileStarted
  machine: activityProtocol
  when: control (requestCancel)
  holds: fun step =>
    step.state.phase == .cancelRequested && step.facts.contains .statusCancelRequested

property canceledByWorker
  machine: activityProtocol
  when: attemptResult (canceled)
  holds: fun step => step.state.phase == .canceled && step.facts.contains .statusCanceled

property terminated
  machine: activityProtocol
  when: control (terminate)
  holds: fun step => step.state.phase == .terminated && step.facts.contains .statusTerminated

property scheduleToStartFires
  machine: activityProtocol
  when: scheduleToStart
  holds: fun step =>
    step.state.phase == .timedOut &&
      step.facts.contains (.statusTimedOut (timeoutType := .scheduleToStart))

property startToCloseFires
  machine: activityProtocol
  when: startToClose
  holds: fun step =>
    step.state.phase == .timedOut &&
      step.facts.contains (.statusTimedOut (timeoutType := .startToClose))

-- authoring: scenarios

/-! ### The paths the Queries run -/

scenario completed
  model: activityProtocol
  starts: unstarted
  actions: [start (unset, unset, unset), attemptStart, attemptResult (completed)]

scenario nonRetryable
  model: activityProtocol
  starts: unstarted
  actions: [start (unset, unset, unset), attemptStart, attemptResult (failed false)]

scenario retriedThenCompleted
  model: activityProtocol
  starts: unstarted
  actions: [start (unset, unset, unset), attemptStart, attemptResult (failed true), backoff,
    attemptStart, attemptResult (completed)]

scenario cancelRequestedThenCanceled
  model: activityProtocol
  starts: unstarted
  actions: [start (unset, unset, unset), attemptStart, control (requestCancel),
    attemptResult (canceled)]

/- The worker stops before the start, so no attempt is in flight when the caller terminates. -/
scenario terminatedWhileScheduled
  model: activityProtocol
  starts: unstarted
  actions: [start (unset, unset, unset), workerStop, control (terminate)]

scenario pausedThenCompleted
  model: activityProtocol
  starts: unstarted
  actions: [start (unset, unset, unset), control (pause), control (unpause), attemptStart,
    attemptResult (completed)]

scenario scheduleToStartExpires
  model: activityProtocol
  starts: unstarted
  actions: [start (unset, expires, unset), workerStop, scheduleToStart]

scenario startToCloseExpires
  model: activityProtocol
  starts: unstarted
  actions: [start (unset, unset, expires), attemptStart, startToClose]

limits three
  steps: 3
  actions: 3
  search: 4096

limits four
  steps: 4
  actions: 4
  search: 32768

limits six
  steps: 6
  actions: 6
  search: 262144

-- authoring: queries

query completion
  find: completes
  in: completed
  limits: three

query nonRetryableFailure
  find: nonRetryableFails
  in: nonRetryable
  limits: three

query retry
  find: retryCompletes
  in: retriedThenCompleted
  limits: six

query cancel
  find: canceledByWorker
  in: cancelRequestedThenCanceled
  limits: four

query terminate
  find: terminated
  in: terminatedWhileScheduled
  limits: three

query pauseResume
  find: completes
  in: pausedThenCompleted
  limits: six

query scheduleToStartTimeout
  find: scheduleToStartFires
  in: scheduleToStartExpires
  limits: three

query startToCloseTimeout
  find: startToCloseFires
  in: startToCloseExpires
  limits: three

query terminalHolds
  verify: terminalIsFinal
  in: completed
  limits: three

query pauseHolds
  verify: pausedIsNotDispatched
  in: pausedThenCompleted
  limits: six

-- authoring: set

/-! ### The sets

Standalone activities exist only under CHASM, so the functional set does not repeat over the
implementation switch. -/

set standaloneActivityTests
  purpose: functional
  bind:
    caller: driven
    worker: driven
  queries: [completion, nonRetryableFailure, retry, cancel, terminate, pauseResume,
    scheduleToStartTimeout, startToCloseTimeout]

set standaloneActivityCanary
  purpose: canary
  bind:
    caller: driven
    worker: observed
  queries: [completion, cancel]

set standaloneActivityExploration
  purpose: exploratory
  bind:
    caller: driven
    worker: driven
  machine: activityProtocol
  cover: rows | results | classMembers
  budget: four

-- authoring: case

/-! ### The Cases

No standalone-activity realization exists yet: `Temporal.Case.Realization` holds `asyncNexus`,
`workflowStart`, `workflowOutage` and `unaryRpc`, and the `case` elaborator is Nexus-specific
today. The blocks below name the realization this Model would need
(`Temporal/Case/Realization/Activity.lean`, about the size of the Nexus one) and do not elaborate
until it exists. -/

case standaloneActivityCases
  realizes standaloneActivityTests
  as (Temporal.Case.Realization.standaloneActivity "umpire.case.activity")

case standaloneActivityCanaryCases
  realizes standaloneActivityCanary
  as (Temporal.Case.Realization.standaloneActivity "umpire.case.activity")

case standaloneActivityExplorationCases
  realizes standaloneActivityExploration
  as standaloneActivityCases.realization

-- authoring: composition

/-! ### The activity and its worker

Composed with the worker of the activity's task queue, the stop is the worker's own phase change
and every attempt start is the worker serving, so an attempt has a row only while the worker
polls. -/

machine activityWorker
  from: Worker.polling
  restrict: [workerStop, serve]

structure StandaloneActivityState where
  activity : ProtocolState
  worker : Worker.WorkerState
  deriving BEq, DecidableEq, Repr

compose standaloneActivity
  for: [activity, Worker.worker]
  state: StandaloneActivityState
  members:
    activity: activityProtocol
    worker: activityWorker
  sync:
    workerStop: activity.workerStop ∥ worker.workerStop
    attemptStart: activity.attemptStart ∥ worker.serve
  starts: [activity.unstarted, worker.polling]
  ends: [activity.completed, activity.failed, activity.canceled, activity.terminated,
    activity.timedOut]

property startedByPollingWorker
  machine: standaloneActivity
  when: attemptStart
  holds: fun step => step.state.worker.phase == .polling

/- The first attempt is started by the polling worker and fails retryably; the backoff returns the
activity to scheduled; the worker then stops, so the retry is never dispatched and the
schedule-to-start deadline fires. The attempt start on the path is what makes the verification
exercise the claim rather than pass for want of a firing. -/
scenario stoppedBeforeRetry
  model: standaloneActivity
  starts: activity.unstarted
  actions: [activity.start (unset, expires, unset), attemptStart,
    activity.attemptResult (failed true), activity.backoff, workerStop, activity.scheduleToStart]

query stoppedWorkerStartsNothing
  verify: startedByPollingWorker
  in: stoppedBeforeRetry
  limits: six

-- authoring: end

end Temporal.Feature.Activity.Standalone
