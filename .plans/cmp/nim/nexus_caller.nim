## The Nexus caller-side Model
##
## One workflow-scheduled Nexus operation, as the caller sees it: the product machine says what an
## operation does, the protocol machine says how the server gets there and refines it, and the
## functional set runs one Query per side effect that settles the operation, once per value of the
## implementation switch. Step functions and predicates rather than rows; no cancellation and no
## concurrency-limit setup parameter.
##
## Read from top to bottom: vocabulary -> the two machines -> what they promise -> what the set asks.

import std/options
import umpire
import worker as Worker

# ── Entities ──────────────────────────────────────────────────────────────────────────────────────
#
# An operation is scheduled by a caller workflow, and recorded data names one by its scheduled
# event: every history event of the operation carries that event's id.

entity workflow

entity operation:
  refer:
    caller: workflow
  key: scheduledEvent

# ── The input domains ─────────────────────────────────────────────────────────────────────────────
#
# A class is one member of a domain, and a constructor that carries finite fields contributes one
# class per assignment of them: `handlerError(retryable: bool)` is one constructor and two classes,
# which is the granularity an example is written at and what mirrors a protobuf oneof.
#
# A payload-free domain is a Nim enum. A domain with a payload is an object variant; its kind enum
# is `{.pure.}` so the nullary constructors below can carry the constructors' own names.

type
  Timeout* = enum
    unset
    expires

  ReplyKind* {.pure.} = enum
    syncSuccess, async, operationFailed, operationCanceled, handlerError

  Reply* = object
    case kind*: ReplyKind
    of ReplyKind.handlerError:
      retryable*: bool
    else:
      discard

  Resolution* = enum
    succeeded
    failed
    canceled

  Delivery* = enum
    accepted
    notFound

finite Reply

const
  syncSuccess* = Reply(kind: ReplyKind.syncSuccess)
  async* = Reply(kind: ReplyKind.async)
  operationFailed* = Reply(kind: ReplyKind.operationFailed)
  operationCanceled* = Reply(kind: ReplyKind.operationCanceled)

proc handlerError*(retryable: bool): Reply =
  Reply(kind: ReplyKind.handlerError, retryable: retryable)

# ── Actions ───────────────────────────────────────────────────────────────────────────────────────
#
# Parties are names the feature declares by using them: `caller`, `handler`, `network`, `worker`.
# The reserved party `system` is the server. A fault is an ordinary action of a declared party, and
# a timer is `system` behavior the machine owns, so neither is a separate kind.

action schedule:
  party: caller
  creates: operation
  schema: temporal.api.command.v1.ScheduleNexusOperationCommandAttributes
  input:
    scheduleToClose: Timeout
    scheduleToStart: Timeout
    startToClose: Timeout

action handlerReply:
  party: handler
  on: operation
  schema: temporal.api.nexus.v1.StartOperationResponse | temporal.api.nexus.v1.HandlerError
  input:
    reply: Reply
  examples:
    handlerError(retryable = false) -> BadRequest
    handlerError(retryable = true) -> Internal

## The Nexus HTTP completion carries no protobuf message, so it declares no schema and its classes
## are names the realization interprets.
action complete:
  party: handler
  on: operation
  input:
    resolution: Resolution
  results: Delivery

action transportFault:
  party: network
  on: operation

## The handler's worker stops polling. An action that names no entity is behavior no entity
## records: the Run records the fault, but nothing recorded names the operation, so the machines
## keep their state and record nothing at it.
action workerStop:
  party: worker

# ── The derived observation ───────────────────────────────────────────────────────────────────────
#
# A retryable attempt failure writes no history event, so the attempt count is read back through
# `DescribeWorkflowExecution`. Every other evidence name resolves against the realization's catalog,
# which is why only a derived observation is declared.

observation pendingAttempts:
  on: operation
  read: attempts

# ── The product machine ───────────────────────────────────────────────────────────────────────────
#
# What an operation does, with no account of how. Every Property written against it is carried to
# the protocol machine by the refinement declared there.

type
  ProductPhase* = enum
    scheduled
    started
    succeeded
    failed
    canceled
    timedOut

  ProductState* = object
    phase*: ProductPhase

  ProductOutcome* = enum
    accepted
    notFound

  ProductFact* = enum
    nexusOperationScheduled
    nexusOperationStarted
    nexusOperationCompleted
    nexusOperationFailed
    nexusOperationCanceled
    nexusOperationTimedOut

  ProductStep* = Step[ProductState, ProductOutcome, ProductFact]

finite ProductState

proc productStep(phase: ProductPhase, recorded: ProductFact): seq[ProductStep] =
  @[ProductStep(outcome: accepted, state: ProductState(phase: phase), facts: @[recorded])]

## The handler's reply to the server's start request. An operation that has not started yet is the
## only one a reply can move.
proc handlerReplyStep*(state: ProductState, reply: Reply): seq[ProductStep] =
  if state.phase != scheduled: return @[]
  case reply.kind
  of ReplyKind.syncSuccess: productStep(succeeded, nexusOperationCompleted)
  of ReplyKind.async: productStep(started, nexusOperationStarted)
  of ReplyKind.operationFailed: productStep(failed, nexusOperationFailed)
  of ReplyKind.operationCanceled: productStep(canceled, nexusOperationCanceled)
  of ReplyKind.handlerError:
    # A retryable handler error leaves the operation where it is: the product machine does not
    # know about backing off, which is the whole of what the protocol machine adds.
    if reply.retryable: @[] else: productStep(failed, nexusOperationFailed)

## The four phases the product machine ends on.
proc productTerminal*(state: ProductState): bool =
  state.phase in {succeeded, failed, canceled, timedOut}

## An asynchronous completion. A completion that arrives after the operation is over is not found,
## and changes nothing.
proc completeStep*(state: ProductState, resolution: Resolution): seq[ProductStep] =
  if productTerminal(state):
    return @[ProductStep(outcome: notFound, state: state)]
  case resolution
  of succeeded: productStep(succeeded, nexusOperationCompleted)
  of failed: productStep(failed, nexusOperationFailed)
  of canceled: productStep(canceled, nexusOperationCanceled)

## A transport fault is an ordinary action of the network. The product machine cannot see one:
## whether a delivery was retried is the protocol's account of how, not what.
proc transportFaultStep*(state: ProductState): seq[ProductStep] = @[]

## The handler's worker stopping is a fault the Run records and the operation does not feel. The
## product machine cannot see it, like the transport fault: a step that kept the state and recorded
## nothing would be indistinguishable from a stutter, and the refinement would read every stutter
## as this step.
proc workerStopStep*(state: ProductState): seq[ProductStep] = @[]

## One of the operation's deadlines firing. Which deadline is the protocol's account of how, so the
## product machine has one timer, and it fires while the operation runs.
proc timeoutStep*(state: ProductState): seq[ProductStep] =
  case state.phase
  of scheduled, started: productStep(timedOut, nexusOperationTimedOut)
  of succeeded, failed, canceled, timedOut: @[]

machine nexusProduct:
  `for`: operation
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

# ── The protocol machine ──────────────────────────────────────────────────────────────────────────
#
# How the server gets there: the retry the product machine cannot see, the three timers the
# schedule command sets, and the attempt count a retryable failure raises. Written against the same
# actions, so a Property proved on the product machine is carried here by the refinement.
#
# The machine begins before the operation exists: a state object has no "no instance yet" member,
# so `unscheduled` is that member, and it is what makes the three deadline fields reachable at
# anything but their first value -- the schedule command is what sets them.
#
# Not here, for reasons recorded rather than silent: the `cancel` field and its rows, and the
# concurrency-limit rejection. The limit exists -- one dynamic-config key per implementation, and
# the schedule command fails the workflow task at it without writing a `NexusOperationScheduled`
# event -- but a step function does not read the setup, the key and value differ per switch value,
# and the rejection names no operation, so it is not modeled until a Query needs it.

type
  Phase* = enum
    unscheduled
    scheduled
    backingOff
    started
    succeeded
    failed
    canceled
    timedOut

  ## Which timer fired. The history event records it, so a Contract that did not check it would
  ## pass a run that timed out on the wrong deadline.
  TimeoutType* = enum
    scheduleToClose
    scheduleToStart
    startToClose

## The attempt count is bounded by the Limits in the design; nothing wires the Limits into a
## machine's state, so the bound is written here and the saturating successor keeps a retry inside
## it.
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
    nexusOperationScheduled, nexusOperationStarted, nexusOperationCompleted,
    nexusOperationFailed, nexusOperationCanceled, nexusOperationTimedOut, pendingAttempts

  ProtocolFact* = object
    case kind*: ProtocolFactKind
    of ProtocolFactKind.nexusOperationTimedOut:
      timeoutType*: TimeoutType
    else:
      discard

  ProtocolStep* = Step[ProtocolState, ProtocolOutcome, ProtocolFact]

finite ProtocolState

const
  nexusOperationScheduled* = ProtocolFact(kind: ProtocolFactKind.nexusOperationScheduled)
  nexusOperationStarted* = ProtocolFact(kind: ProtocolFactKind.nexusOperationStarted)
  nexusOperationCompleted* = ProtocolFact(kind: ProtocolFactKind.nexusOperationCompleted)
  nexusOperationFailed* = ProtocolFact(kind: ProtocolFactKind.nexusOperationFailed)
  nexusOperationCanceled* = ProtocolFact(kind: ProtocolFactKind.nexusOperationCanceled)
  pendingAttempts* = ProtocolFact(kind: ProtocolFactKind.pendingAttempts)

proc nexusOperationTimedOut*(timeoutType: TimeoutType): ProtocolFact =
  ProtocolFact(kind: ProtocolFactKind.nexusOperationTimedOut, timeoutType: timeoutType)

## The four phases the design ends on. A completion that arrives after one of them is not found.
proc terminalPhase*(phase: Phase): bool =
  phase in {succeeded, failed, canceled, timedOut}

## Scheduled and not yet over: the phases a completion resolves and a timer can fire in.
proc running*(phase: Phase): bool =
  phase in {scheduled, backingOff, started}

proc moves(state: ProtocolState, phase: Phase, recorded: seq[ProtocolFact]): seq[ProtocolStep] =
  var next = state
  next.phase = phase
  @[ProtocolStep(outcome: accepted, state: next, facts: recorded)]

## The caller's schedule command. It names the operation's three deadlines, and every one of them
## is a state field because whether a timer fires is a question about the operation and not about
## the command that started it.
proc scheduleStep*(state: ProtocolState,
    scheduleToClose, scheduleToStart, startToClose: Timeout): seq[ProtocolStep] =
  if state.phase != unscheduled: return @[]
  @[ProtocolStep(outcome: accepted,
    state: ProtocolState(phase: scheduled, attempts: 0, scheduleToClose: scheduleToClose,
      scheduleToStart: scheduleToStart, startToClose: startToClose),
    facts: @[nexusOperationScheduled])]

## The handler's reply to the server's start request. What the product machine cannot see is the
## last arm: a retryable failure backs the operation off and raises its attempt count, and the count
## is read back through the `pendingAttempts` observation because no history event records it.
proc protocolHandlerReplyStep*(state: ProtocolState, reply: Reply): seq[ProtocolStep] =
  if state.phase != scheduled: return @[]
  case reply.kind
  of ReplyKind.syncSuccess: moves(state, succeeded, @[nexusOperationCompleted])
  of ReplyKind.async: moves(state, started, @[nexusOperationStarted])
  of ReplyKind.operationFailed: moves(state, failed, @[nexusOperationFailed])
  of ReplyKind.operationCanceled: moves(state, canceled, @[nexusOperationCanceled])
  of ReplyKind.handlerError:
    if not reply.retryable:
      moves(state, failed, @[nexusOperationFailed])
    else:
      var next = state
      next.phase = backingOff
      next.attempts = saturatingSucc(state.attempts)
      @[ProtocolStep(outcome: accepted, state: next, facts: @[pendingAttempts])]

## A transport fault is the same failure arriving as a dropped delivery rather than as a reply.
proc protocolTransportFaultStep*(state: ProtocolState): seq[ProtocolStep] =
  if state.phase != scheduled: return @[]
  var next = state
  next.phase = backingOff
  next.attempts = saturatingSucc(state.attempts)
  @[ProtocolStep(outcome: accepted, state: next, facts: @[pendingAttempts])]

## The handler's worker stopping is a fault the Run records and the operation does not feel, so the
## step keeps the state and records nothing. On a path it is confirmed by the evidence of the step
## after it, and the Case says so in a Known Gap.
proc protocolWorkerStopStep*(state: ProtocolState): seq[ProtocolStep] =
  @[ProtocolStep(outcome: accepted, state: state)]

## An asynchronous completion. Before a start, the server records a Started event first, which is
## why the evidence is two facts and not one -- and why the product machine, which has no
## `backingOff` phase to have skipped, could write the completion alone.
proc protocolCompleteStep*(state: ProtocolState, resolution: Resolution): seq[ProtocolStep] =
  if terminalPhase(state.phase):
    return @[ProtocolStep(outcome: notFound, state: state)]
  if state.phase == unscheduled: return @[]
  let startedFirst: seq[ProtocolFact] =
    if state.phase == started: @[] else: @[nexusOperationStarted]
  case resolution
  of succeeded: moves(state, succeeded, startedFirst & nexusOperationCompleted)
  of failed: moves(state, failed, startedFirst & nexusOperationFailed)
  of canceled: moves(state, canceled, startedFirst & nexusOperationCanceled)

## The backoff timer. It is what makes `backingOff` a phase the operation leaves rather than a
## state it is stuck in, and it records nothing: a retry writes no history event.
proc backoffStep*(state: ProtocolState): seq[ProtocolStep] =
  if state.phase != backingOff: return @[]
  moves(state, scheduled, @[])

## The schedule-to-close deadline covers the whole operation, so it fires in every running phase --
## and only when the schedule command set it.
proc scheduleToCloseStep*(state: ProtocolState): seq[ProtocolStep] =
  if running(state.phase) and state.scheduleToClose == expires:
    moves(state, timedOut, @[nexusOperationTimedOut(scheduleToClose)])
  else: @[]

## The schedule-to-start deadline covers the wait for the handler to accept, so it stops at the
## start.
proc scheduleToStartStep*(state: ProtocolState): seq[ProtocolStep] =
  if state.phase in {scheduled, backingOff} and state.scheduleToStart == expires:
    moves(state, timedOut, @[nexusOperationTimedOut(scheduleToStart)])
  else: @[]

## The start-to-close deadline covers the handler's own work, so it begins at the start.
proc startToCloseStep*(state: ProtocolState): seq[ProtocolStep] =
  if state.phase == started and state.startToClose == expires:
    moves(state, timedOut, @[nexusOperationTimedOut(startToClose)])
  else: @[]

## How a protocol state reads as a product state. A phase of the same name is that phase; backing
## off is still scheduled, because the product machine cannot see a retry; and an operation not yet
## scheduled reads as scheduled, because the product machine begins there. Every other field is
## hidden, which is what a map that does not read it says.
proc productOf*(state: ProtocolState): ProductState =
  ProductState(phase: (case state.phase
    of unscheduled, scheduled, backingOff: ProductPhase.scheduled
    of started: ProductPhase.started
    of succeeded: ProductPhase.succeeded
    of failed: ProductPhase.failed
    of canceled: ProductPhase.canceled
    of timedOut: ProductPhase.timedOut))

machine nexusProtocol:
  `for`: operation
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

# ── What the machines promise ─────────────────────────────────────────────────────────────────────
#
# A same-step claim names the action it is about under `when:` and holds of the step that action
# produces; a transition claim holds of the step before and the step after. A functional Query
# realizes a same-step claim, because the Case's Contract is the claim's clause triggered by the
# action the Case performs; a transition claim is searched and verified, never realized.

## Once an operation is over, no step changes its phase. Declared on the product machine and read
## on the protocol machine through the map.
property terminalIsFinal:
  machine: nexusProduct
  holds: (before, after) =>
    not productTerminal(before.state) or after.state.phase == before.state.phase

## A synchronous reply settles the operation as succeeded, and the completed event records it.
property syncSucceeds:
  machine: nexusProtocol
  `when`: handlerReply(syncSuccess)
  holds: step => step.state.phase == succeeded and nexusOperationCompleted in step.facts

## An asynchronous reply starts the operation, and the started event records it.
property asyncStarts:
  machine: nexusProtocol
  `when`: handlerReply(async)
  holds: step => step.state.phase == started and nexusOperationStarted in step.facts

## A successful completion is recorded by the completed event. Neither the phase nor the outcome
## is fixed: a completion resolves any running phase, and `accepted` is every earlier step's
## outcome too, so a clause fixing it would be answered before the completion.
property completionSucceeds:
  machine: nexusProtocol
  `when`: complete(succeeded)
  holds: step => nexusOperationCompleted in step.facts

## A failed completion is recorded by the failed event.
property completionFails:
  machine: nexusProtocol
  `when`: complete(failed)
  holds: step => nexusOperationFailed in step.facts

## A non-retryable handler error settles the operation as failed, and the failed event records it.
property handlerErrorFails:
  machine: nexusProtocol
  `when`: handlerReply(handlerError(false))
  holds: step => step.state.phase == failed and nexusOperationFailed in step.facts

## Succeeded on the second attempt of an operation with no deadline set. A claim fixes one state,
## so every field is named.
const succeededOnRetry* = ProtocolState(phase: succeeded, attempts: 1,
  scheduleToClose: unset, scheduleToStart: unset, startToClose: unset)

## A synchronous reply to the retried attempt settles the operation as succeeded on its second
## attempt: the count the retryable failure raised is still one, and the completed event records
## the reply.
property retrySucceeds:
  machine: nexusProtocol
  `when`: handlerReply(syncSuccess)
  holds: step => step.state == succeededOnRetry and nexusOperationCompleted in step.facts

## The schedule-to-start deadline settles an operation no handler started as timed out, and the
## timed-out event records which deadline it was.
property scheduleToStartFires:
  machine: nexusProtocol
  `when`: scheduleToStart
  holds: step =>
    step.state.phase == timedOut and nexusOperationTimedOut(scheduleToStart) in step.facts

## The start-to-close deadline settles a started operation no handler completed as timed out.
property startToCloseFires:
  machine: nexusProtocol
  `when`: startToClose
  holds: step =>
    step.state.phase == timedOut and nexusOperationTimedOut(startToClose) in step.facts

# ── The paths the Queries run ─────────────────────────────────────────────────────────────────────
#
# A protocol Scenario names its classed actions with their inputs and its start by its phase. Each
# path below is one upstream functional test's shape: the schedule command with no deadline set,
# then the side effects that settle the operation.

scenario syncReplied:
  model: nexusProtocol
  starts: unscheduled
  actions: [schedule(unset, unset, unset), handlerReply(syncSuccess)]

scenario asyncThenSucceeded:
  model: nexusProtocol
  starts: unscheduled
  actions: [schedule(unset, unset, unset), handlerReply(async), complete(succeeded)]

scenario asyncThenFailed:
  model: nexusProtocol
  starts: unscheduled
  actions: [schedule(unset, unset, unset), handlerReply(async), complete(failed)]

scenario nonRetryableError:
  model: nexusProtocol
  starts: unscheduled
  actions: [schedule(unset, unset, unset), handlerReply(handlerError(false))]

## The retryable error backs the operation off; the backoff timer fires and records nothing; the
## retried attempt is answered synchronously.
scenario retriedThenSucceeded:
  model: nexusProtocol
  starts: unscheduled
  actions: [schedule(unset, unset, unset), handlerReply(handlerError(true)), backoff,
    handlerReply(syncSuccess)]

## The schedule command sets the schedule-to-start deadline; the handler's worker stops, so nothing
## answers the start request; the deadline fires. The worker stops after the schedule in the
## operation's order, where the stop changes nothing; the realization stops it before the workflow
## starts, where the stop cannot race the dispatch.
scenario scheduleToStartExpires:
  model: nexusProtocol
  starts: unscheduled
  actions: [schedule(unset, expires, unset), workerStop, scheduleToStart]

## The schedule command sets the start-to-close deadline; the handler accepts asynchronously and
## never completes; the deadline fires.
scenario startToCloseExpires:
  model: nexusProtocol
  starts: unscheduled
  actions: [schedule(unset, unset, expires), handlerReply(async), startToClose]

## Nine actions are enabled before the operation is scheduled and eleven once it is, so an exact
## sequence of two is found among ninety-nine candidates, one of three among about a thousand and
## one of four among about ten thousand.
limits two:
  steps: 2
  actions: 2
  search: 512

limits three:
  steps: 3
  actions: 3
  search: 4096

limits four:
  steps: 4
  actions: 4
  search: 32768

# ── The Queries ───────────────────────────────────────────────────────────────────────────────────
#
# The design's seven: sync success, async reply then succeeded callback, async reply then failed
# callback, non-retryable handler error, retryable handler error then sync success after one
# backoff, schedule-to-start timeout with the handler's worker stopped, start-to-close timeout after
# an asynchronous reply. Each finds its same-step claim on its path and is realized by the set
# below. The product claim is verified over every trace of one path, outside the set, because a
# `verify` Query realizes nothing.
#
# Each search runs in the compile-time VM; a claim its path never reaches fails the build at the
# `find:` line.

query syncCompletion:
  find: syncSucceeds
  `in`: syncReplied
  limits: two

query asyncCompletion:
  find: completionSucceeds
  `in`: asyncThenSucceeded
  limits: three

query asyncFailure:
  find: completionFails
  `in`: asyncThenFailed
  limits: three

# NOTE: `handlerError` is also the `Reply` constructor above. Nim has one flat scope per module, so
# this line is a redefinition as written; see README, "Where the spec's names fight Nim's scope".
query handlerError:
  find: handlerErrorFails
  `in`: nonRetryableError
  limits: two

query retry:
  find: retrySucceeds
  `in`: retriedThenSucceeded
  limits: four

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
  `in`: asyncThenSucceeded
  limits: three

# ── The functional set ────────────────────────────────────────────────────────────────────────────
#
# Every party but `system` is bound: the Case drives the caller, the handler and the worker, and
# observes the network. The set repeats over the implementation switch, so each Query's Case runs
# once under HSM and once under CHASM.

set nexusCallerTests:
  purpose: functional
  `bind`:
    caller: driven
    handler: driven
    network: observed
    worker: driven
  repeat: implementation
  queries: [syncCompletion, asyncCompletion, asyncFailure, handlerError, retry,
    scheduleToStartTimeout, startToCloseTimeout]

# ── The canary set ────────────────────────────────────────────────────────────────────────────────
#
# A canary runs a Query against a deployment that performs the handler's part itself: the handler
# is `observed`, so the verifier reads which reply occurred and checks the machine allows it. What
# admits a canary is that a deployment can close every gap its Case carries, and every step of the
# sync and async completion paths records evidence; a path with a silent step -- the backoff, the
# worker stop -- is a capability gap no deployment closes, so a canary naming it is rejected.

set nexusCallerCanary:
  purpose: canary
  `bind`:
    caller: driven
    handler: observed
    network: observed
    worker: driven
  queries: [syncCompletion, asyncCompletion]

# ── The exploratory set ───────────────────────────────────────────────────────────────────────────
#
# An exploration covers the protocol machine rather than listing Queries. Its targets are the rows
# an exploration within the budget's steps of a start can take, the results those rows reach and
# the members of the classes their actions claim, each in the machine's catalog order and cut at
# the budget's search count, so the enumeration is the same on every reading.

set nexusCallerExploration:
  purpose: exploratory
  `bind`:
    caller: driven
    handler: driven
    network: observed
    worker: driven
  machine: nexusProtocol
  cover: rows | results | classMembers
  budget: four

# ── The operation and the handler's worker ────────────────────────────────────────────────────────
#
# The protocol machine's worker stop is a stutter row: the operation cannot see its handler's
# worker, so the schedule-to-start Scenario orders the stop before the request by convention.
# Composed with the worker of the handler's task queue, the stop is the worker's own phase change
# and every reply is the worker serving, so a reply has a row only while the worker polls. No set
# names the composition; it is what the cross-entity claim is verified over.

## The caller's view of the handler's worker: it stops and it serves. It never resumes, because an
## action no `sync:` line names would stay executable on its own and admit a stop, a resume and
## then a reply; the operation's timers settle every state a stop leaves.
machine handlerWorker:
  `from`: Worker.polling
  restrict: [workerStop, serve]

type NexusCallerState* = object
  operation*: ProtocolState
  worker*: Worker.WorkerState

finite NexusCallerState

compose nexusCaller:
  `for`: [operation, Worker.worker]
  state: NexusCallerState
  members:
    operation: nexusProtocol
    worker: handlerWorker
  sync:
    workerStop: operation.workerStop || worker.workerStop
    handlerReply: operation.handlerReply || worker.serve
  starts: [operation.unscheduled, worker.polling]
  ends: [operation.succeeded, operation.failed, operation.canceled, operation.timedOut]

## Every reply, of any class, leaves the handler's worker polling: no handler replies while its
## worker is stopped.
property repliedByPollingWorker:
  machine: nexusCaller
  `when`: handlerReply
  holds: step => step.state.worker.phase == Worker.polling

## A retryable reply backs the operation off; the handler's worker then stops, so the retried
## attempt is never answered and the schedule-to-start deadline fires.
scenario repliedThenStopped:
  model: nexusCaller
  starts: operation.unscheduled
  actions: [operation.schedule(unset, expires, unset), handlerReply(handlerError(true)),
    workerStop, operation.scheduleToStart]

query stoppedWorkerRepliesNothing:
  verify: repliedByPollingWorker
  `in`: repliedThenStopped
  limits: four
