package temporal.feature.nexus.caller

// authoring: header

/* The Nexus caller-side Model
 *
 * One workflow-scheduled Nexus operation, as the caller sees it: the product machine says what an
 * operation does, the protocol machine says how the server gets there and refines it, and the
 * functional set runs one Query per side effect that settles the operation, once per value of the
 * implementation switch. Step functions and predicates rather than rows, no cancellation (fn-79)
 * and no concurrency-limit setup parameter.
 *
 * Read from top to bottom: vocabulary -> the two machines -> what they promise -> what the set asks.
 */

import umpire.*
import temporal.feature.worker as Worker
import Worker.{workerStop, serve, WorkerState}

// authoring: entities

/* Entities
 *
 * An operation is scheduled by a caller workflow, and recorded data names one by its scheduled
 * event: every history event of the operation carries that event's id. */

val workflow: Entity = entity("workflow")

val operation: Entity = entity("operation") refer ("caller" -> workflow) key "scheduledEvent"

// authoring: domains

/* The input domains
 *
 * A class is one member of a domain, and a constructor that carries finite fields contributes one
 * class per assignment of them: `handlerError(retryable: Boolean)` is one constructor and two
 * classes, which is the granularity an example is written at and what mirrors a protobuf oneof.
 * `derives Finite` is what enumerates them; enums get `CanEqual` for free. */

enum Timeout derives Finite:
  case unset, expires
export Timeout.{unset, expires}

enum Reply derives Finite:
  case syncSuccess, async, operationFailed, operationCanceled
  case handlerError(retryable: Boolean)

enum Resolution derives Finite:
  case succeeded, failed, canceled

enum Delivery derives Finite:
  case accepted, notFound

// authoring: actions

/* Actions
 *
 * Parties are names the feature declares by using them: `caller`, `handler`, `network`, `worker`.
 * The reserved party `system` is the server. A fault is an ordinary action of a declared party,
 * and a timer is `system` behavior the machine owns, so neither is a separate kind. Each `input`
 * line extends the action's tuple type, so `schedule` is `Action[(Timeout, Timeout, Timeout)]`. */

val schedule: Action[(Timeout, Timeout, Timeout)] =
  action("schedule")
    .party(caller)
    .creates(operation)
    .schema("temporal.api.command.v1.ScheduleNexusOperationCommandAttributes")
    .input[Timeout]("scheduleToClose")
    .input[Timeout]("scheduleToStart")
    .input[Timeout]("startToClose")

val handlerReply: Action[Reply *: EmptyTuple] =
  action("handlerReply")
    .party(handler)
    .on(operation)
    .schema("temporal.api.nexus.v1.StartOperationResponse | temporal.api.nexus.v1.HandlerError")
    .input[Reply]("reply")
    .examples(
      Tuple1(Reply.handlerError(retryable = false)) -> "BadRequest",
      Tuple1(Reply.handlerError(retryable = true)) -> "Internal",
    )

/** The Nexus HTTP completion carries no protobuf message, so it declares no schema and its classes
  * are names the realization interprets. */
val complete: Action[Resolution *: EmptyTuple] =
  action("complete")
    .party(handler)
    .on(operation)
    .input[Resolution]("resolution")
    .results[Delivery]

val transportFault: Action[EmptyTuple] =
  action("transportFault") party network on operation

/** The handler's worker stops polling. An action that names no entity is behavior no entity
  * records: the Run records the fault, but nothing recorded names the operation, so the machines
  * keep their state and record nothing at it. Shared with the worker module, so the composition
  * below can `sync` it. */
// (`workerStop` is imported from `temporal.feature.worker`.)

// authoring: observation

/* The derived observation
 *
 * A retryable attempt failure writes no history event, so the attempt count is read back through
 * `DescribeWorkflowExecution`. Every other evidence name resolves against the realization's
 * catalog, which is why only a derived observation is declared. */

val pendingAttempts: Observation = observation("pendingAttempts") on operation read "attempts"

// authoring: product

/* The product machine
 *
 * What an operation does, with no account of how. Every Property written against it is carried
 * to the protocol machine by the refinement declared there. */

enum ProductPhase derives Finite:
  case scheduled, started, succeeded, failed, canceled, timedOut

final case class ProductState(phase: ProductPhase) derives Finite, CanEqual

enum ProductOutcome derives Finite:
  case accepted, notFound

enum ProductFact derives Finite:
  case nexusOperationScheduled, nexusOperationStarted, nexusOperationCompleted,
    nexusOperationFailed, nexusOperationCanceled, nexusOperationTimedOut

type ProductStep = Step[ProductState, ProductOutcome, ProductFact]

private def productStep(phase: ProductPhase, recorded: ProductFact): List[ProductStep] =
  List(Step(ProductOutcome.accepted, ProductState(phase), List(recorded)))

/** The handler's reply to the server's start request. An operation that has not started yet is
  * the only one a reply can move. The `match` is exhaustive or the file does not compile under
  * `-Werror`. */
def handlerReplyStep(state: ProductState, reply: Reply): List[ProductStep] =
  if state.phase != ProductPhase.scheduled then Nil
  else reply match
    case Reply.syncSuccess => productStep(ProductPhase.succeeded, ProductFact.nexusOperationCompleted)
    case Reply.async => productStep(ProductPhase.started, ProductFact.nexusOperationStarted)
    case Reply.operationFailed => productStep(ProductPhase.failed, ProductFact.nexusOperationFailed)
    case Reply.operationCanceled => productStep(ProductPhase.canceled, ProductFact.nexusOperationCanceled)
    // A retryable handler error leaves the operation where it is: the product machine does not
    // know about backing off, which is the whole of what the protocol machine adds.
    case Reply.handlerError(true) => Nil
    case Reply.handlerError(false) => productStep(ProductPhase.failed, ProductFact.nexusOperationFailed)

/** The four phases the product machine ends on. */
def productTerminal(state: ProductState): Boolean = state.phase match
  case ProductPhase.succeeded | ProductPhase.failed | ProductPhase.canceled | ProductPhase.timedOut => true
  case ProductPhase.scheduled | ProductPhase.started => false

/** An asynchronous completion. A completion that arrives after the operation is over is not
  * found, and changes nothing. */
def completeStep(state: ProductState, resolution: Resolution): List[ProductStep] =
  if productTerminal(state) then List(Step(ProductOutcome.notFound, state, Nil))
  else resolution match
    case Resolution.succeeded => productStep(ProductPhase.succeeded, ProductFact.nexusOperationCompleted)
    case Resolution.failed => productStep(ProductPhase.failed, ProductFact.nexusOperationFailed)
    case Resolution.canceled => productStep(ProductPhase.canceled, ProductFact.nexusOperationCanceled)

/** A transport fault is an ordinary action of the network. The product machine cannot see one:
  * whether a delivery was retried is the protocol's account of how, not what. */
def transportFaultStep(state: ProductState): List[ProductStep] = Nil

/** The handler's worker stopping is a fault the Run records and the operation does not feel. The
  * product machine cannot see it, like the transport fault: a step that kept the state and
  * recorded nothing would be indistinguishable from a stutter, and the refinement would read every
  * stutter as this step. */
def workerStopStep(state: ProductState): List[ProductStep] = Nil

/** One of the operation's deadlines firing. Which deadline is the protocol's account of how, so
  * the product machine has one timer, and it fires while the operation runs. */
val timeout: Action[EmptyTuple] = timer("timeout")

def timeoutStep(state: ProductState): List[ProductStep] = state.phase match
  case ProductPhase.scheduled | ProductPhase.started =>
    productStep(ProductPhase.timedOut, ProductFact.nexusOperationTimedOut)
  case _ => Nil

val nexusProduct: Machine[ProductState, ProductOutcome, ProductFact] =
  machine[ProductState, ProductOutcome, ProductFact]("nexusProduct"):
    forEntity(operation)
    starts(ProductPhase.scheduled)
    ends(ProductPhase.succeeded, ProductPhase.failed, ProductPhase.canceled, ProductPhase.timedOut)
    timers(timeout)
    evidence:
      case ProductFact.nexusOperationScheduled => "nexusOperationScheduled"
      case ProductFact.nexusOperationStarted => "nexusOperationStarted"
      case ProductFact.nexusOperationCompleted => "nexusOperationCompleted"
      case ProductFact.nexusOperationFailed => "nexusOperationFailed"
      case ProductFact.nexusOperationCanceled => "nexusOperationCanceled"
      case ProductFact.nexusOperationTimedOut => "nexusOperationTimedOut"
    steps(
      handlerReply ~> handlerReplyStep,
      complete ~> completeStep,
      transportFault ~> transportFaultStep,
      workerStop ~> workerStopStep,
      timeout ~> timeoutStep,
    )

// authoring: protocol

/* The protocol machine
 *
 * How the server gets there: the retry the product machine cannot see, the three timers the
 * schedule command sets, and the attempt count a retryable failure raises. Written against the
 * same actions, so a Property proved on the product machine is carried here by the refinement.
 *
 * The machine begins before the operation exists: a state type has no "no instance yet" member,
 * so `unscheduled` is that member, and it is what makes the three deadline fields reachable at
 * anything but their first value -- the schedule command is what sets them.
 *
 * Not here, for reasons recorded rather than silent: the `cancel` field and its rows (fn-79), and
 * the concurrency-limit rejection. The limit exists -- one dynamic-config key per implementation,
 * and the schedule command fails the workflow task at it without writing a
 * `NexusOperationScheduled` event -- but a step function does not read the setup, the key and
 * value differ per switch value, and the rejection names no operation, so it is not modeled until
 * a Query needs it. */

enum Phase derives Finite:
  case unscheduled, scheduled, backingOff, started, succeeded, failed, canceled, timedOut

/** Which timer fired. The history event records it, so a Contract that did not check it would
  * pass a run that timed out on the wrong deadline. */
enum TimeoutType derives Finite:
  case scheduleToClose, scheduleToStart, startToClose

/** The attempt count is bounded by the Limits in the design; nothing wires the Limits into a
  * machine's state, so the bound is written here and the saturating successor keeps a retry
  * inside it. `Bounded[2]` is `0..2`, and `Attempts(3)` is a compile error. */
inline val attemptBound = 2
type Attempts = Bounded[attemptBound.type]
object Attempts:
  inline def apply(inline n: Int): Attempts = Bounded[attemptBound.type](n)

final case class ProtocolState(
    phase: Phase,
    attempts: Attempts,
    scheduleToClose: Timeout,
    scheduleToStart: Timeout,
    startToClose: Timeout,
) derives Finite, CanEqual

enum ProtocolOutcome derives Finite:
  case accepted, notFound

enum ProtocolFact derives Finite:
  case nexusOperationScheduled, nexusOperationStarted, nexusOperationCompleted,
    nexusOperationFailed, nexusOperationCanceled
  case nexusOperationTimedOut(timeoutType: TimeoutType)
  case pendingAttempts

type ProtocolStep = Step[ProtocolState, ProtocolOutcome, ProtocolFact]

/** The four phases the design ends on. A completion that arrives after one of them is not found. */
def terminalPhase(phase: Phase): Boolean = phase match
  case Phase.succeeded | Phase.failed | Phase.canceled | Phase.timedOut => true
  case Phase.unscheduled | Phase.scheduled | Phase.backingOff | Phase.started => false

/** Scheduled and not yet over: the phases a completion resolves and a timer can fire in. */
def running(phase: Phase): Boolean = phase match
  case Phase.scheduled | Phase.backingOff | Phase.started => true
  case _ => false

private def moves(state: ProtocolState, phase: Phase, recorded: List[ProtocolFact]): List[ProtocolStep] =
  List(Step(ProtocolOutcome.accepted, state.copy(phase = phase), recorded))

/** The caller's schedule command. It names the operation's three deadlines, and every one of them
  * is a state field because whether a timer fires is a question about the operation and not about
  * the command that started it. */
def scheduleStep(
    state: ProtocolState,
    scheduleToClose: Timeout,
    scheduleToStart: Timeout,
    startToClose: Timeout,
): List[ProtocolStep] =
  if state.phase != Phase.unscheduled then Nil
  else
    List(Step(
      ProtocolOutcome.accepted,
      ProtocolState(Phase.scheduled, Attempts(0), scheduleToClose, scheduleToStart, startToClose),
      List(ProtocolFact.nexusOperationScheduled),
    ))

/** The handler's reply to the server's start request. What the product machine cannot see is the
  * last arm: a retryable failure backs the operation off and raises its attempt count, and the
  * count is read back through the `pendingAttempts` observation because no history event records
  * it. */
def protocolHandlerReplyStep(state: ProtocolState, reply: Reply): List[ProtocolStep] =
  if state.phase != Phase.scheduled then Nil
  else reply match
    case Reply.syncSuccess => moves(state, Phase.succeeded, List(ProtocolFact.nexusOperationCompleted))
    case Reply.async => moves(state, Phase.started, List(ProtocolFact.nexusOperationStarted))
    case Reply.operationFailed => moves(state, Phase.failed, List(ProtocolFact.nexusOperationFailed))
    case Reply.operationCanceled => moves(state, Phase.canceled, List(ProtocolFact.nexusOperationCanceled))
    case Reply.handlerError(false) => moves(state, Phase.failed, List(ProtocolFact.nexusOperationFailed))
    case Reply.handlerError(true) =>
      List(Step(
        ProtocolOutcome.accepted,
        state.copy(phase = Phase.backingOff, attempts = state.attempts.saturatingSucc),
        List(ProtocolFact.pendingAttempts),
      ))

/** A transport fault is the same failure arriving as a dropped delivery rather than as a reply. */
def protocolTransportFaultStep(state: ProtocolState): List[ProtocolStep] =
  if state.phase != Phase.scheduled then Nil
  else
    List(Step(
      ProtocolOutcome.accepted,
      state.copy(phase = Phase.backingOff, attempts = state.attempts.saturatingSucc),
      List(ProtocolFact.pendingAttempts),
    ))

/** The handler's worker stopping is a fault the Run records and the operation does not feel, so
  * the step keeps the state and records nothing. On a path it is confirmed by the evidence of the
  * step after it, and the Case says so in a Known Gap. */
def protocolWorkerStopStep(state: ProtocolState): List[ProtocolStep] =
  List(Step(ProtocolOutcome.accepted, state, Nil))

/** An asynchronous completion. Before a start, the server records a Started event first, which is
  * why the evidence is two facts and not one -- and why the product machine, which has no
  * `backingOff` phase to have skipped, could write the completion alone. */
def protocolCompleteStep(state: ProtocolState, resolution: Resolution): List[ProtocolStep] =
  if terminalPhase(state.phase) then List(Step(ProtocolOutcome.notFound, state, Nil))
  else if state.phase == Phase.unscheduled then Nil
  else
    val startedFirst: List[ProtocolFact] =
      if state.phase == Phase.started then Nil else List(ProtocolFact.nexusOperationStarted)
    resolution match
      case Resolution.succeeded => moves(state, Phase.succeeded, startedFirst :+ ProtocolFact.nexusOperationCompleted)
      case Resolution.failed => moves(state, Phase.failed, startedFirst :+ ProtocolFact.nexusOperationFailed)
      case Resolution.canceled => moves(state, Phase.canceled, startedFirst :+ ProtocolFact.nexusOperationCanceled)

/** The backoff timer. It is what makes `backingOff` a phase the operation leaves rather than a
  * state it is stuck in, and it records nothing: a retry writes no history event. */
val backoff: Action[EmptyTuple] = timer("backoff")
val scheduleToClose: Action[EmptyTuple] = timer("scheduleToClose")
val scheduleToStart: Action[EmptyTuple] = timer("scheduleToStart")
val startToClose: Action[EmptyTuple] = timer("startToClose")

def backoffStep(state: ProtocolState): List[ProtocolStep] =
  if state.phase != Phase.backingOff then Nil else moves(state, Phase.scheduled, Nil)

/** The schedule-to-close deadline covers the whole operation, so it fires in every running phase
  * -- and only when the schedule command set it. */
def scheduleToCloseStep(state: ProtocolState): List[ProtocolStep] =
  if running(state.phase) && state.scheduleToClose == expires then
    moves(state, Phase.timedOut, List(ProtocolFact.nexusOperationTimedOut(TimeoutType.scheduleToClose)))
  else Nil

/** The schedule-to-start deadline covers the wait for the handler to accept, so it stops at the
  * start. */
def scheduleToStartStep(state: ProtocolState): List[ProtocolStep] =
  if (state.phase == Phase.scheduled || state.phase == Phase.backingOff) && state.scheduleToStart == expires then
    moves(state, Phase.timedOut, List(ProtocolFact.nexusOperationTimedOut(TimeoutType.scheduleToStart)))
  else Nil

/** The start-to-close deadline covers the handler's own work, so it begins at the start. */
def startToCloseStep(state: ProtocolState): List[ProtocolStep] =
  if state.phase == Phase.started && state.startToClose == expires then
    moves(state, Phase.timedOut, List(ProtocolFact.nexusOperationTimedOut(TimeoutType.startToClose)))
  else Nil

/** How a protocol state reads as a product state. A phase of the same name is that phase; backing
  * off is still scheduled, because the product machine cannot see a retry; and an operation not
  * yet scheduled reads as scheduled, because the product machine begins there. Every other field
  * is hidden, which is what a map that does not read it says. */
def productOf(state: ProtocolState): ProductState = ProductState(state.phase match
  case Phase.unscheduled | Phase.scheduled | Phase.backingOff => ProductPhase.scheduled
  case Phase.started => ProductPhase.started
  case Phase.succeeded => ProductPhase.succeeded
  case Phase.failed => ProductPhase.failed
  case Phase.canceled => ProductPhase.canceled
  case Phase.timedOut => ProductPhase.timedOut)

/** The map as a type-level fact, so the machine's `refines` line and a `verify` Query that reads a
  * product Property on a protocol Scenario both find it. */
given productView: Refines[ProtocolState, ProductState] = Refines(productOf)

val nexusProtocol: Machine[ProtocolState, ProtocolOutcome, ProtocolFact] =
  machine[ProtocolState, ProtocolOutcome, ProtocolFact]("nexusProtocol"):
    forEntity(operation)
    refines(nexusProduct)
    starts(Phase.unscheduled)
    ends(Phase.succeeded, Phase.failed, Phase.canceled, Phase.timedOut)
    timers(backoff, scheduleToClose, scheduleToStart, startToClose)
    unobservable(backoff)
    // A timed-out fact of any type resolves to the one catalog event; the pattern says so.
    evidence:
      case ProtocolFact.nexusOperationScheduled => "nexusOperationScheduled"
      case ProtocolFact.nexusOperationStarted => "nexusOperationStarted"
      case ProtocolFact.nexusOperationCompleted => "nexusOperationCompleted"
      case ProtocolFact.nexusOperationFailed => "nexusOperationFailed"
      case ProtocolFact.nexusOperationCanceled => "nexusOperationCanceled"
      case ProtocolFact.nexusOperationTimedOut(_) => "nexusOperationTimedOut"
      case ProtocolFact.pendingAttempts => pendingAttempts
    steps(
      schedule ~> scheduleStep,
      handlerReply ~> protocolHandlerReplyStep,
      complete ~> protocolCompleteStep,
      transportFault ~> protocolTransportFaultStep,
      workerStop ~> protocolWorkerStopStep,
      backoff ~> backoffStep,
      scheduleToClose ~> scheduleToCloseStep,
      scheduleToStart ~> scheduleToStartStep,
      startToClose ~> startToCloseStep,
    )

// authoring: properties

/* What the machines promise
 *
 * A same-step claim names the action it is about under `when` and holds of the step that action
 * produces; a transition claim holds of the step before and the step after. A functional Query
 * realizes a same-step claim, because the Case's Contract is the claim's clause triggered by the
 * action the Case performs; a transition claim is searched and verified, never realized. The two
 * `holds` overloads tell them apart by the lambda's arity. */

/* Once an operation is over, no step changes its phase. Declared on the product machine and read
 * on the protocol machine through the map. */
val terminalIsFinal: Property[ProductState] =
  property("terminalIsFinal")(nexusProduct) holds: (before, after) =>
    !productTerminal(before.state) || after.state.phase == before.state.phase

/* A synchronous reply settles the operation as succeeded, and the completed event records it. */
val syncSucceeds: Property[ProtocolState] =
  property("syncSucceeds")(nexusProtocol) when handlerReply(Reply.syncSuccess) holds: step =>
    step.state.phase == Phase.succeeded && step.facts.contains(ProtocolFact.nexusOperationCompleted)

/* An asynchronous reply starts the operation, and the started event records it. */
val asyncStarts: Property[ProtocolState] =
  property("asyncStarts")(nexusProtocol) when handlerReply(Reply.async) holds: step =>
    step.state.phase == Phase.started && step.facts.contains(ProtocolFact.nexusOperationStarted)

/* A successful completion is recorded by the completed event. Neither the phase nor the outcome
 * is fixed: a completion resolves any running phase, and `accepted` is every earlier step's
 * outcome too, so a clause fixing it would be answered before the completion. */
val completionSucceeds: Property[ProtocolState] =
  property("completionSucceeds")(nexusProtocol) when complete(Resolution.succeeded) holds: step =>
    step.facts.contains(ProtocolFact.nexusOperationCompleted)

/* A failed completion is recorded by the failed event. */
val completionFails: Property[ProtocolState] =
  property("completionFails")(nexusProtocol) when complete(Resolution.failed) holds: step =>
    step.facts.contains(ProtocolFact.nexusOperationFailed)

/* A non-retryable handler error settles the operation as failed, and the failed event records it. */
val handlerErrorFails: Property[ProtocolState] =
  property("handlerErrorFails")(nexusProtocol) when handlerReply(Reply.handlerError(false)) holds: step =>
    step.state.phase == Phase.failed && step.facts.contains(ProtocolFact.nexusOperationFailed)

/** Succeeded on the second attempt of an operation with no deadline set. A claim fixes one state,
  * so every field is named. */
val succeededOnRetry: ProtocolState =
  ProtocolState(Phase.succeeded, Attempts(1), unset, unset, unset)

/* A synchronous reply to the retried attempt settles the operation as succeeded on its second
 * attempt: the count the retryable failure raised is still one, and the completed event records
 * the reply. */
val retrySucceeds: Property[ProtocolState] =
  property("retrySucceeds")(nexusProtocol) when handlerReply(Reply.syncSuccess) holds: step =>
    step.state == succeededOnRetry && step.facts.contains(ProtocolFact.nexusOperationCompleted)

/* The schedule-to-start deadline settles an operation no handler started as timed out, and the
 * timed-out event records which deadline it was. */
val scheduleToStartFires: Property[ProtocolState] =
  property("scheduleToStartFires")(nexusProtocol) when scheduleToStart holds: step =>
    step.state.phase == Phase.timedOut &&
      step.facts.contains(ProtocolFact.nexusOperationTimedOut(TimeoutType.scheduleToStart))

/* The start-to-close deadline settles a started operation no handler completed as timed out. */
val startToCloseFires: Property[ProtocolState] =
  property("startToCloseFires")(nexusProtocol) when startToClose holds: step =>
    step.state.phase == Phase.timedOut &&
      step.facts.contains(ProtocolFact.nexusOperationTimedOut(TimeoutType.startToClose))

// authoring: scenarios

/* The paths the Queries run
 *
 * A protocol Scenario names its classed actions with their inputs and its start by its phase.
 * Each path below is one upstream functional test's shape: the schedule command with no deadline
 * set, then the side effects that settle the operation. */

val syncReplied: Scenario[ProtocolState] =
  scenario("syncReplied")(nexusProtocol) starts Phase.unscheduled actions (
    schedule(unset, unset, unset), handlerReply(Reply.syncSuccess))

val asyncThenSucceeded: Scenario[ProtocolState] =
  scenario("asyncThenSucceeded")(nexusProtocol) starts Phase.unscheduled actions (
    schedule(unset, unset, unset), handlerReply(Reply.async), complete(Resolution.succeeded))

val asyncThenFailed: Scenario[ProtocolState] =
  scenario("asyncThenFailed")(nexusProtocol) starts Phase.unscheduled actions (
    schedule(unset, unset, unset), handlerReply(Reply.async), complete(Resolution.failed))

val nonRetryableError: Scenario[ProtocolState] =
  scenario("nonRetryableError")(nexusProtocol) starts Phase.unscheduled actions (
    schedule(unset, unset, unset), handlerReply(Reply.handlerError(false)))

/* The retryable error backs the operation off; the backoff timer fires and records nothing; the
 * retried attempt is answered synchronously. */
val retriedThenSucceeded: Scenario[ProtocolState] =
  scenario("retriedThenSucceeded")(nexusProtocol) starts Phase.unscheduled actions (
    schedule(unset, unset, unset), handlerReply(Reply.handlerError(true)), backoff,
    handlerReply(Reply.syncSuccess))

/* The schedule command sets the schedule-to-start deadline; the handler's worker stops, so nothing
 * answers the start request; the deadline fires. The worker stops after the schedule in the
 * operation's order, where the stop changes nothing; the realization stops it before the workflow
 * starts, where the stop cannot race the dispatch. */
val scheduleToStartExpires: Scenario[ProtocolState] =
  scenario("scheduleToStartExpires")(nexusProtocol) starts Phase.unscheduled actions (
    schedule(unset, expires, unset), workerStop, scheduleToStart)

/* The schedule command sets the start-to-close deadline; the handler accepts asynchronously and
 * never completes; the deadline fires. */
val startToCloseExpires: Scenario[ProtocolState] =
  scenario("startToCloseExpires")(nexusProtocol) starts Phase.unscheduled actions (
    schedule(unset, unset, expires), handlerReply(Reply.async), startToClose)

/* Nine actions are enabled before the operation is scheduled and eleven once it is, so an exact
 * sequence of two is found among ninety-nine candidates, one of three among about a thousand and
 * one of four among about ten thousand. */
val two: Limits = limits("two")(steps = 2, actions = 2, search = 512)
val three: Limits = limits("three")(steps = 3, actions = 3, search = 4096)
val four: Limits = limits("four")(steps = 4, actions = 4, search = 32768)

// authoring: queries

/* The Queries
 *
 * The design's seven: sync success, async reply then succeeded callback, async reply then failed
 * callback, non-retryable handler error, retryable handler error then sync success after one
 * backoff, schedule-to-start timeout with the handler's worker stopped, start-to-close timeout
 * after an asynchronous reply. Each finds its same-step claim on its path and is realized by the
 * set below. The product claim is verified over every trace of one path, outside the set, because
 * a `verify` Query realizes nothing.
 *
 * These are declarations; the search runs in the test module (`Pins.scala`), where `Query.pinned`
 * runs it while that module compiles. */

val syncCompletion = query("syncCompletion") find syncSucceeds in syncReplied limits two
val asyncCompletion = query("asyncCompletion") find completionSucceeds in asyncThenSucceeded limits three
val asyncFailure = query("asyncFailure") find completionFails in asyncThenFailed limits three
val handlerError = query("handlerError") find handlerErrorFails in nonRetryableError limits two
val retry = query("retry") find retrySucceeds in retriedThenSucceeded limits four
val scheduleToStartTimeout =
  query("scheduleToStartTimeout") find scheduleToStartFires in scheduleToStartExpires limits three
val startToCloseTimeout =
  query("startToCloseTimeout") find startToCloseFires in startToCloseExpires limits three
val terminalHolds = query("terminalHolds") verify terminalIsFinal in asyncThenSucceeded limits three

// authoring: set

/* The functional set
 *
 * Every party but `system` is bound: the Case drives the caller, the handler and the worker, and
 * observes the network. The set repeats over the implementation switch, so each Query's Case runs
 * once under HSM and once under CHASM. */

val nexusCallerTests: Set = set("nexusCallerTests"):
  purpose(functional)
  bind(caller -> driven, handler -> driven, network -> observed, worker -> driven)
  repeat(implementation)
  queries(syncCompletion, asyncCompletion, asyncFailure, handlerError, retry,
    scheduleToStartTimeout, startToCloseTimeout)

/* The canary set
 *
 * A canary runs a Query against a deployment that performs the handler's part itself: the handler
 * is `observed`, so the verifier reads which reply occurred and checks the machine allows it. What
 * admits a canary is that a deployment can close every gap its Case carries, and every step of
 * the sync and async completion paths records evidence; a path with a silent step -- the backoff,
 * the worker stop -- is a capability gap no deployment closes, so a canary naming it is rejected. */

val nexusCallerCanary: Set = set("nexusCallerCanary"):
  purpose(canary)
  bind(caller -> driven, handler -> observed, network -> observed, worker -> driven)
  queries(syncCompletion, asyncCompletion)

/* The exploratory set
 *
 * An exploration covers the protocol machine rather than listing Queries. Its targets are the
 * rows an exploration within the budget's steps of a start can take, the results those rows reach
 * and the members of the classes their actions claim, each in the machine's catalog order and cut
 * at the budget's search count, so the enumeration is the same on every reading. */

val nexusCallerExploration: Set = set("nexusCallerExploration"):
  purpose(exploratory)
  bind(caller -> driven, handler -> driven, network -> observed, worker -> driven)
  machine(nexusProtocol)
  cover(rows | results | classMembers)
  budget(four)

// authoring: composition

/* The operation and the handler's worker
 *
 * The protocol machine's worker stop is a stutter row: the operation cannot see its handler's
 * worker, so the schedule-to-start Scenario orders the stop before the request by convention.
 * Composed with the worker of the handler's task queue, the stop is the worker's own phase change
 * and every reply is the worker serving, so a reply has a row only while the worker polls. No set
 * names the composition; it is what the cross-entity claim is verified over. */

/** The caller's view of the handler's worker: it stops and it serves. It never resumes, because
  * an action no `sync` line names would stay executable on its own and admit a stop, a resume and
  * then a reply; the operation's timers settle every state a stop leaves. */
val handlerWorker: Machine[WorkerState, ?, ?] = Worker.polling restrict (workerStop, serve)

final case class NexusCallerState(operation: ProtocolState, worker: WorkerState) derives Finite, CanEqual

/** An `object`, so the members are named fields a Scenario can reach as `nexusCaller.operation`. */
object nexusCaller extends Compose[NexusCallerState]("nexusCaller"):
  val operation = member("operation", nexusProtocol)(_.operation)
  val worker = member("worker", handlerWorker)(_.worker)
  sync(workerStop, operation(workerStop) || worker(workerStop))
  sync(handlerReply, operation(handlerReply) || worker(serve))
  starts(operation at Phase.unscheduled, worker at Worker.Phase.polling)
  ends(operation at Phase.succeeded, operation at Phase.failed, operation at Phase.canceled,
    operation at Phase.timedOut)

/* Every reply, of any class, leaves the handler's worker polling: no handler replies while its
 * worker is stopped. */
val repliedByPollingWorker: Property[NexusCallerState] =
  property("repliedByPollingWorker")(nexusCaller) when handlerReply holds: step =>
    step.state.worker.phase == Worker.Phase.polling

/* A retryable reply backs the operation off; the handler's worker then stops, so the retried
 * attempt is never answered and the schedule-to-start deadline fires. */
val repliedThenStopped: Scenario[NexusCallerState] =
  scenario("repliedThenStopped")(nexusCaller) starts (nexusCaller.operation at Phase.unscheduled) actions (
    nexusCaller.operation(schedule)(unset, expires, unset), handlerReply(Reply.handlerError(true)),
    workerStop, nexusCaller.operation(scheduleToStart))

val stoppedWorkerRepliesNothing =
  query("stoppedWorkerRepliesNothing") verify repliedByPollingWorker in repliedThenStopped limits four

// authoring: end
