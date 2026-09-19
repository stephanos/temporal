import Temporal.Case.Syntax

/-!
# The Nexus caller-side Model

One workflow-scheduled Nexus operation, as the caller sees it: the product machine says what an
operation does, the protocol machine says how the server gets there and refines it, and the
functional set runs one Query per side effect that settles the operation, once per value of the
implementation switch. `DESIGN.md` section 3 is this file's specimen, written in the landed grammar:
step functions and predicates rather than rows, no cancellation (fn-79) and no concurrency-limit
setup parameter (`UMPIRE4_RESEARCH_NEXUS_MODEL.md` section 1).

The regions `AUTHORING.md` (fn-85 .13) quotes are marked `-- authoring: <name>`; a region runs to
the next marker. The drift test reads the markers, so a quoted block and the Model cannot part.

Read from top to bottom: vocabulary → the two machines → what they promise → what the set asks.
-/

namespace Temporal.Feature.Nexus.Caller

open Umpire
open Umpire.Command

-- authoring: entities

/-! ### Entities

An operation is scheduled by a caller workflow, and recorded data names one by its scheduled event:
every history event of the operation carries that event's id. -/

entity workflow

entity operation
  refer:
    caller: workflow
  key: scheduledEvent

-- authoring: domains

/-! ### The input domains

A class is one member of a domain, and a constructor that carries finite fields contributes one
class per assignment of them: `handlerError (retryable : Bool)` is one constructor and two classes,
which is the granularity an example is written at and what mirrors a protobuf oneof. -/

enum Timeout
  | unset
  | expires

enum Reply
  | syncSuccess
  | async
  | operationFailed
  | operationCanceled
  | handlerError (retryable : Bool)

enum Resolution
  | succeeded
  | failed
  | canceled

enum Delivery
  | accepted
  | notFound

-- authoring: actions

/-! ### Actions

Parties are names the feature declares by using them: `caller`, `handler`, `network`, `worker`.
The reserved party `system` is the server. A fault is an ordinary action of a declared party, and a
timer is `system` behavior the machine owns, so neither is a separate kind. -/

action schedule
  party: caller
  creates: operation
  schema: temporal.api.command.v1.ScheduleNexusOperationCommandAttributes
  input:
    scheduleToClose: Timeout
    scheduleToStart: Timeout
    startToClose: Timeout

action handlerReply
  party: handler
  on: operation
  schema: temporal.api.nexus.v1.StartOperationResponse | temporal.api.nexus.v1.HandlerError
  input:
    reply: Reply
  examples:
    handlerError (retryable := false) → BadRequest
    handlerError (retryable := true) → Internal

/-- The Nexus HTTP completion carries no protobuf message, so it declares no schema and its classes
are names the realization interprets. -/
action complete
  party: handler
  on: operation
  input:
    resolution: Resolution
  results: Delivery

action transportFault
  party: network
  on: operation

/-- The handler's worker stops polling. An action that names no entity is behavior no entity
records: the Run records the fault, but nothing recorded names the operation, so the machines keep
their state and record nothing at it. -/
action workerStop
  party: worker

-- authoring: observation

/-! ### The derived observation

A retryable attempt failure writes no history event, so the attempt count is read back through
`DescribeWorkflowExecution`. Every other evidence name resolves against the realization's catalog,
which is why only a derived observation is declared. -/

observation pendingAttempts
  on: operation
  read: attempts

-- authoring: product

/-! ### The product machine

What an operation does, with no account of how. Every Property written against it is carried to
the protocol machine by the refinement declared there. -/

enum ProductPhase
  | scheduled
  | started
  | succeeded
  | failed
  | canceled
  | timedOut

structure ProductState where
  phase : ProductPhase
  deriving BEq, DecidableEq, Repr, Finite

enum ProductOutcome
  | accepted
  | notFound

enum ProductFact
  | nexusOperationScheduled
  | nexusOperationStarted
  | nexusOperationCompleted
  | nexusOperationFailed
  | nexusOperationCanceled
  | nexusOperationTimedOut

private def productStep (phase : ProductPhase) (recorded : ProductFact) :
    List (Step ProductState ProductOutcome ProductFact) :=
  [{ outcome := .accepted, state := { phase }, facts := [recorded] }]

/-- The handler's reply to the server's start request. An operation that has not started yet is the
only one a reply can move. -/
def handlerReplyStep (state : ProductState) (reply : Reply) :
    List (Step ProductState ProductOutcome ProductFact) :=
  if state.phase != .scheduled then [] else
  match reply with
  | .syncSuccess => productStep .succeeded .nexusOperationCompleted
  | .async => productStep .started .nexusOperationStarted
  | .operationFailed => productStep .failed .nexusOperationFailed
  | .operationCanceled => productStep .canceled .nexusOperationCanceled
  -- A retryable handler error leaves the operation where it is: the product machine does not know
  -- about backing off, which is the whole of what the protocol machine adds.
  | .handlerError true => []
  | .handlerError false => productStep .failed .nexusOperationFailed

/-- The four phases the product machine ends on. -/
def productTerminal (state : ProductState) : Bool :=
  state.phase == .succeeded || state.phase == .failed || state.phase == .canceled ||
    state.phase == .timedOut

/-- An asynchronous completion. A completion that arrives after the operation is over is not found,
and changes nothing. -/
def completeStep (state : ProductState) (resolution : Resolution) :
    List (Step ProductState ProductOutcome ProductFact) :=
  if productTerminal state then
    [{ outcome := .notFound, state, facts := [] }]
  else
    match resolution with
    | .succeeded => productStep .succeeded .nexusOperationCompleted
    | .failed => productStep .failed .nexusOperationFailed
    | .canceled => productStep .canceled .nexusOperationCanceled

/-- A transport fault is an ordinary action of the network. The product machine cannot see one:
whether a delivery was retried is the protocol's account of how, not what. -/
def transportFaultStep (_state : ProductState) :
    List (Step ProductState ProductOutcome ProductFact) := []

/-- The handler's worker stopping is a fault the Run records and the operation does not feel. The
product machine cannot see it, like the transport fault: a step that kept the state and recorded
nothing would be indistinguishable from a stutter, and the refinement would read every stutter as
this step. -/
def workerStopStep (_state : ProductState) :
    List (Step ProductState ProductOutcome ProductFact) := []

/-- One of the operation's deadlines firing. Which deadline is the protocol's account of how, so the
product machine has one timer, and it fires while the operation runs. -/
def timeoutStep (state : ProductState) :
    List (Step ProductState ProductOutcome ProductFact) :=
  if state.phase == .scheduled || state.phase == .started then
    productStep .timedOut .nexusOperationTimedOut
  else []

machine nexusProduct
  for: operation
  state: ProductState
  starts: [scheduled]
  ends: [succeeded, failed, canceled, timedOut]
  timers: [timeout]
  evidence:
    nexusOperationStarted: nexusOperationStarted
    nexusOperationCompleted: nexusOperationCompleted
    nexusOperationFailed: nexusOperationFailed
    nexusOperationCanceled: nexusOperationCanceled
    nexusOperationTimedOut: nexusOperationTimedOut
  steps:
    handlerReply: handlerReplyStep
    complete: completeStep
    transportFault: transportFaultStep
    workerStop: workerStopStep
    timeout: timeoutStep

-- authoring: protocol

/-! ### The protocol machine

How the server gets there: the retry the product machine cannot see, the three timers the schedule
command sets, and the attempt count a retryable failure raises. Written against the same actions,
so a Property proved on the product machine is carried here by the refinement.

The machine begins before the operation exists: a state structure has no "no instance yet" member,
so `unscheduled` is that member, and it is what makes the three deadline fields reachable at
anything but their first value -- the schedule command is what sets them.

Not here, for reasons recorded rather than silent: the `cancel` field and its rows (fn-79), and the
concurrency-limit rejection. The limit exists -- one dynamic-config key per implementation, and the
schedule command fails the workflow task at it without writing a `NexusOperationScheduled` event --
but a step function does not read the setup, the key and value differ per switch value, and the
rejection names no operation, so it is not modeled until a Query needs it. -/

enum Phase
  | unscheduled
  | scheduled
  | backingOff
  | started
  | succeeded
  | failed
  | canceled
  | timedOut

/-- Which timer fired. The history event records it, so a Contract that did not check it would pass
a run that timed out on the wrong deadline. -/
enum TimeoutType
  | scheduleToClose
  | scheduleToStart
  | startToClose

/-- The attempt count is bounded by the Limits in the design; nothing wires the Limits into a
machine's state, so the bound is written here and the saturating successor keeps a retry inside it.
-/
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
  | nexusOperationScheduled
  | nexusOperationStarted
  | nexusOperationCompleted
  | nexusOperationFailed
  | nexusOperationCanceled
  | nexusOperationTimedOut (timeoutType : TimeoutType)
  | pendingAttempts

/-- The four phases the design ends on. A completion that arrives after one of them is not found. -/
def terminalPhase (phase : Phase) : Bool :=
  phase == .succeeded || phase == .failed || phase == .canceled || phase == .timedOut

/-- Scheduled and not yet over: the phases a completion resolves and a timer can fire in. -/
def running (phase : Phase) : Bool :=
  phase == .scheduled || phase == .backingOff || phase == .started

private def moves (state : ProtocolState) (phase : Phase) (recorded : List ProtocolFact) :
    List (Step ProtocolState ProtocolOutcome ProtocolFact) :=
  [{ outcome := .accepted, state := { state with phase }, facts := recorded }]

/-- The caller's schedule command. It names the operation's three deadlines, and every one of them
is a state field because whether a timer fires is a question about the operation and not about the
command that started it. -/
def scheduleStep (state : ProtocolState)
    (scheduleToClose scheduleToStart startToClose : Timeout) :
    List (Step ProtocolState ProtocolOutcome ProtocolFact) :=
  if state.phase != .unscheduled then [] else
  [{ outcome := .accepted
     state := { phase := .scheduled, attempts := 0, scheduleToClose, scheduleToStart, startToClose }
     facts := [.nexusOperationScheduled] }]

/-- The handler's reply to the server's start request. What the product machine cannot see is the
last arm: a retryable failure backs the operation off and raises its attempt count, and the count is
read back through the `pendingAttempts` observation because no history event records it. -/
def protocolHandlerReplyStep (state : ProtocolState) (reply : Reply) :
    List (Step ProtocolState ProtocolOutcome ProtocolFact) :=
  if state.phase != .scheduled then [] else
  match reply with
  | .syncSuccess => moves state .succeeded [.nexusOperationCompleted]
  | .async => moves state .started [.nexusOperationStarted]
  | .operationFailed => moves state .failed [.nexusOperationFailed]
  | .operationCanceled => moves state .canceled [.nexusOperationCanceled]
  | .handlerError false => moves state .failed [.nexusOperationFailed]
  | .handlerError true =>
      [{ outcome := .accepted
         state := { state with phase := .backingOff, attempts := saturatingSucc state.attempts }
         facts := [.pendingAttempts] }]

/-- A transport fault is the same failure arriving as a dropped delivery rather than as a reply. -/
def protocolTransportFaultStep (state : ProtocolState) :
    List (Step ProtocolState ProtocolOutcome ProtocolFact) :=
  if state.phase != .scheduled then [] else
  [{ outcome := .accepted
     state := { state with phase := .backingOff, attempts := saturatingSucc state.attempts }
     facts := [.pendingAttempts] }]

/-- The handler's worker stopping is a fault the Run records and the operation does not feel, so
the step keeps the state and records nothing. On a path it is confirmed by the evidence of the step
after it, and the Case says so in a Known Gap. -/
def protocolWorkerStopStep (state : ProtocolState) :
    List (Step ProtocolState ProtocolOutcome ProtocolFact) :=
  [{ outcome := .accepted, state, facts := [] }]

/-- An asynchronous completion. Before a start, the server records a Started event first, which is
why the evidence is two facts and not one -- and why the product machine, which has no `backingOff`
phase to have skipped, could write the completion alone. -/
def protocolCompleteStep (state : ProtocolState) (resolution : Resolution) :
    List (Step ProtocolState ProtocolOutcome ProtocolFact) :=
  if terminalPhase state.phase then
    [{ outcome := .notFound, state, facts := [] }]
  else if state.phase == .unscheduled then []
  else
    let startedFirst : List ProtocolFact :=
      if state.phase == .started then [] else [.nexusOperationStarted]
    match resolution with
    | .succeeded => moves state .succeeded (startedFirst ++ [.nexusOperationCompleted])
    | .failed => moves state .failed (startedFirst ++ [.nexusOperationFailed])
    | .canceled => moves state .canceled (startedFirst ++ [.nexusOperationCanceled])

/-- The backoff timer. It is what makes `backingOff` a phase the operation leaves rather than a
state it is stuck in, and it records nothing: a retry writes no history event. -/
def backoffStep (state : ProtocolState) :
    List (Step ProtocolState ProtocolOutcome ProtocolFact) :=
  if state.phase != .backingOff then [] else moves state .scheduled []

/-- The schedule-to-close deadline covers the whole operation, so it fires in every running phase --
and only when the schedule command set it. -/
def scheduleToCloseStep (state : ProtocolState) :
    List (Step ProtocolState ProtocolOutcome ProtocolFact) :=
  if running state.phase && state.scheduleToClose == .expires then
    moves state .timedOut [.nexusOperationTimedOut (timeoutType := .scheduleToClose)]
  else []

/-- The schedule-to-start deadline covers the wait for the handler to accept, so it stops at the
start. -/
def scheduleToStartStep (state : ProtocolState) :
    List (Step ProtocolState ProtocolOutcome ProtocolFact) :=
  if (state.phase == .scheduled || state.phase == .backingOff) &&
      state.scheduleToStart == .expires then
    moves state .timedOut [.nexusOperationTimedOut (timeoutType := .scheduleToStart)]
  else []

/-- The start-to-close deadline covers the handler's own work, so it begins at the start. -/
def startToCloseStep (state : ProtocolState) :
    List (Step ProtocolState ProtocolOutcome ProtocolFact) :=
  if state.phase == .started && state.startToClose == .expires then
    moves state .timedOut [.nexusOperationTimedOut (timeoutType := .startToClose)]
  else []

/-- How a protocol state reads as a product state. A phase of the same name is that phase; backing
off is still scheduled, because the product machine cannot see a retry; and an operation not yet
scheduled reads as scheduled, because the product machine begins there. Every other field is
hidden, which is what a map that does not read it says. -/
def productOf (state : ProtocolState) : ProductState :=
  { phase := match state.phase with
    | .unscheduled | .scheduled | .backingOff => .scheduled
    | .started => .started
    | .succeeded => .succeeded
    | .failed => .failed
    | .canceled => .canceled
    | .timedOut => .timedOut }

machine nexusProtocol
  for: operation
  state: ProtocolState
  refines: nexusProduct
  map: productOf
  starts: [unscheduled]
  ends: [succeeded, failed, canceled, timedOut]
  timers: [backoff, scheduleToClose, scheduleToStart, startToClose]
  unobservable: [backoff]
  evidence:
    nexusOperationScheduled: nexusOperationScheduled
    nexusOperationStarted: nexusOperationStarted
    nexusOperationCompleted: nexusOperationCompleted
    nexusOperationFailed: nexusOperationFailed
    nexusOperationCanceled: nexusOperationCanceled
    nexusOperationTimedOut: nexusOperationTimedOut
    pendingAttempts: pendingAttempts
  steps:
    schedule: scheduleStep
    handlerReply: protocolHandlerReplyStep
    complete: protocolCompleteStep
    transportFault: protocolTransportFaultStep
    workerStop: protocolWorkerStopStep
    backoff: backoffStep
    scheduleToClose: scheduleToCloseStep
    scheduleToStart: scheduleToStartStep
    startToClose: startToCloseStep

-- authoring: properties

/-! ### What the machines promise

A same-step claim names the action it is about under `when:` and holds of the step that action
produces; a transition claim holds of the step before and the step after. A functional Query
realizes a same-step claim, because the Case's Contract is the claim's clause triggered by the
action the Case performs; a transition claim is searched and verified, never realized. -/

/- Once an operation is over, no step changes its phase. Declared on the product machine and read
on the protocol machine through the map. -/
property terminalIsFinal
  machine: nexusProduct
  holds: fun before after =>
    !(productTerminal before.state) || after.state.phase == before.state.phase

/- A synchronous reply settles the operation as succeeded, and the completed event records it. -/
property syncSucceeds
  machine: nexusProtocol
  when: handlerReply (syncSuccess)
  holds: fun step =>
    step.state.phase == .succeeded && step.facts.contains .nexusOperationCompleted

/- An asynchronous reply starts the operation, and the started event records it. -/
property asyncStarts
  machine: nexusProtocol
  when: handlerReply (async)
  holds: fun step => step.state.phase == .started && step.facts.contains .nexusOperationStarted

/- A successful completion is recorded by the completed event. Neither the phase nor the outcome
is fixed: a completion resolves any running phase, and `accepted` is every earlier step's outcome
too, so a clause fixing it would be answered before the completion. -/
property completionSucceeds
  machine: nexusProtocol
  when: complete (succeeded)
  holds: fun step => step.facts.contains .nexusOperationCompleted

/- A failed completion is recorded by the failed event. -/
property completionFails
  machine: nexusProtocol
  when: complete (failed)
  holds: fun step => step.facts.contains .nexusOperationFailed

/- A non-retryable handler error settles the operation as failed, and the failed event records it. -/
property handlerErrorFails
  machine: nexusProtocol
  when: handlerReply (handlerError false)
  holds: fun step => step.state.phase == .failed && step.facts.contains .nexusOperationFailed

/-- Succeeded on the second attempt of an operation with no deadline set. A claim fixes one state, so
every field is named. -/
def succeededOnRetry : ProtocolState :=
  { phase := .succeeded, attempts := 1, scheduleToClose := .unset, scheduleToStart := .unset,
    startToClose := .unset }

/- A synchronous reply to the retried attempt settles the operation as succeeded on its second
attempt: the count the retryable failure raised is still one, and the completed event records the
reply. -/
property retrySucceeds
  machine: nexusProtocol
  when: handlerReply (syncSuccess)
  holds: fun step =>
    step.state == succeededOnRetry && step.facts.contains .nexusOperationCompleted

/- The schedule-to-start deadline settles an operation no handler started as timed out, and the
timed-out event records which deadline it was. -/
property scheduleToStartFires
  machine: nexusProtocol
  when: scheduleToStart
  holds: fun step =>
    step.state.phase == .timedOut &&
      step.facts.contains (.nexusOperationTimedOut (timeoutType := .scheduleToStart))

/- The start-to-close deadline settles a started operation no handler completed as timed out. -/
property startToCloseFires
  machine: nexusProtocol
  when: startToClose
  holds: fun step =>
    step.state.phase == .timedOut &&
      step.facts.contains (.nexusOperationTimedOut (timeoutType := .startToClose))

-- authoring: scenarios

/-! ### The paths the Queries run

A protocol Scenario names its classed actions with their inputs and its start by its phase. Each
path below is one upstream functional test's shape: the schedule command with no deadline set,
then the side effects that settle the operation. -/

scenario syncReplied
  model: nexusProtocol
  starts: unscheduled
  actions: [schedule (unset, unset, unset), handlerReply (syncSuccess)]

scenario asyncThenSucceeded
  model: nexusProtocol
  starts: unscheduled
  actions: [schedule (unset, unset, unset), handlerReply (async), complete (succeeded)]

scenario asyncThenFailed
  model: nexusProtocol
  starts: unscheduled
  actions: [schedule (unset, unset, unset), handlerReply (async), complete (failed)]

scenario nonRetryableError
  model: nexusProtocol
  starts: unscheduled
  actions: [schedule (unset, unset, unset), handlerReply (handlerError false)]

/- The retryable error backs the operation off; the backoff timer fires and records nothing; the
retried attempt is answered synchronously. -/
scenario retriedThenSucceeded
  model: nexusProtocol
  starts: unscheduled
  actions: [schedule (unset, unset, unset), handlerReply (handlerError true), backoff,
    handlerReply (syncSuccess)]

/- The schedule command sets the schedule-to-start deadline; the handler's worker stops, so nothing
answers the start request; the deadline fires. The worker stops after the schedule in the
operation's order, where the stop changes nothing; the realization stops it before the workflow
starts, where the stop cannot race the dispatch. -/
scenario scheduleToStartExpires
  model: nexusProtocol
  starts: unscheduled
  actions: [schedule (unset, expires, unset), workerStop, scheduleToStart]

/- The schedule command sets the start-to-close deadline; the handler accepts asynchronously and
never completes; the deadline fires. -/
scenario startToCloseExpires
  model: nexusProtocol
  starts: unscheduled
  actions: [schedule (unset, unset, expires), handlerReply (async), startToClose]

/- Nine actions are enabled before the operation is scheduled and eleven once it is, so an exact
sequence of two is found among ninety-nine candidates, one of three among about a thousand and one
of four among about ten thousand. -/
limits two
  steps: 2
  actions: 2
  search: 512

limits three
  steps: 3
  actions: 3
  search: 4096

limits four
  steps: 4
  actions: 4
  search: 32768

-- authoring: queries

/-! ### The Queries

The design's seven: sync success, async reply then succeeded callback, async reply then failed
callback, non-retryable handler error, retryable handler error then sync success after one backoff,
schedule-to-start timeout with the handler's worker stopped, start-to-close timeout after an
asynchronous reply. Each finds its same-step claim on its path and is realized by the set below.
The product claim is verified over every trace of one path, outside the set, because a `verify`
Query realizes nothing. -/

query syncCompletion
  find: syncSucceeds
  in: syncReplied
  limits: two

query asyncCompletion
  find: completionSucceeds
  in: asyncThenSucceeded
  limits: three

query asyncFailure
  find: completionFails
  in: asyncThenFailed
  limits: three

query handlerError
  find: handlerErrorFails
  in: nonRetryableError
  limits: two

query retry
  find: retrySucceeds
  in: retriedThenSucceeded
  limits: four

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
  in: asyncThenSucceeded
  limits: three

-- authoring: set

/-! ### The functional set

Every party but `system` is bound: the Case drives the caller, the handler and the worker, and
observes the network. The set repeats over the implementation switch, so each Query's Case runs
once under HSM and once under CHASM. -/

set nexusCallerTests
  purpose: functional
  bind:
    caller: driven
    handler: driven
    network: observed
    worker: driven
  repeat: implementation
  queries: [syncCompletion, asyncCompletion, asyncFailure, handlerError, retry,
    scheduleToStartTimeout, startToCloseTimeout]

-- authoring: case

/-! ### The Cases

One realization serves every Query: the Producer places each class the path performs where the
realization binds it. The evidence each Case lifts is read off the machine's own `evidence:` lines
along the witness, so nothing is written twice; a step that records nothing -- the backoff timer,
the worker stop -- is confirmed by the evidence of the step after it, and the Case carries a Known
Gap naming it. Each Case is `temporal.case.nexusCallerTests.<query>` and the fixture
`nexusCallerTests-<query>-case.json`. -/

case nexusCallerCases
  realizes nexusCallerTests
  as (Temporal.Case.Realization.asyncNexus "umpire.case.service" "complete")

-- authoring: end

end Temporal.Feature.Nexus.Caller
