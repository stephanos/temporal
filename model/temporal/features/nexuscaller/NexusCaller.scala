/* The Nexus caller-side Model: one workflow-scheduled Nexus operation, as the caller sees it. The
 * product machine says what an operation does, the protocol machine says how the server gets there
 * and refines it, and the functional Queries are one per side effect that settles the operation.
 * No cancellation (fn-79) and no concurrency-limit setup parameter.
 *
 * Read top to bottom: the types; the signature (the parties, the entities, the inputs, actions and
 * timers, the derived observation and the bounds); then one object per machine, each before the
 * machines that use it -- Product, the product machine; Protocol, the protocol machine that refines
 * it; HandlerWorker, the handler's worker; NexusCaller, the protocol with that worker; Control, the
 * forged control a caller must refuse -- and last Files, its IR files. Realization.scala realizes
 * it; closepolicy/ holds the close and reset designs.
 */
package temporal
package features.nexuscaller

import scala.annotation.unused
import umpire.*
import umpire.realize.{Alternative, Cleanup, Conformance, Disposition, Exploration, Reason}
import umpire.realize.{PropertyOutcome, RunExpectation, Variation}
import temporal.realize.{inconclusive, satisfied}
import io.temporal.api.command.v1.ScheduleNexusOperationCommandAttributes
import io.temporal.api.nexus.v1.{HandlerError, StartOperationResponse}
import shared.Bounds.{four, three}
import shared.worker.{serve, workerStop, Phase as WorkerPhase, State as WorkerState}
import CallerFamily.given
import Timeout.expires

// Moved from temporal.nexuscaller; the pin keeps its Definition IDs and type names.
given DefinitionScope = DefinitionScope("temporal.nexuscaller.Model$package$")

/** The family of the caller's machines; the control takes its own, in `object Control`. */
object CallerFamily:
  given family: Family = Family("temporal.nexus.caller")

// ### Types
//
// A class is one member of a domain, and a constructor that carries finite fields contributes one
// class per assignment of them: handlerError(retryable) is one constructor and two classes, which
// is the granularity an example is written at and what mirrors a protobuf oneof.

/** Whether the schedule command sets a deadline. */
enum Timeout derives Finite:
  case unset, expires

/** The handler's reply to the server's start request. */
enum Reply derives Finite:
  case syncSuccess, async, operationFailed, operationCanceled
  case handlerError(retryable: Boolean)

/** How an asynchronous completion settles the operation. */
enum Resolution derives Finite:
  case succeeded, failed, canceled

/**
 * A step's outcome. The product and protocol machines share the two members, and an outcome reads
 * as the refined machine's outcome of the same name.
 */
enum Outcome derives Finite:
  case accepted, notFound

/** The product machine's phases: what an operation does. */
enum ProductPhase derives Finite:
  case scheduled, started, succeeded, failed, canceled, timedOut

final case class ProductState(phase: ProductPhase) derives Finite

enum ProductFact derives Finite:
  case nexusOperationScheduled, nexusOperationStarted, nexusOperationCompleted,
    nexusOperationFailed,
    nexusOperationCanceled, nexusOperationTimedOut

type ProductStep = Step[ProductState, Outcome, ProductFact]

/** The protocol machine's phases. It begins before the operation exists, so unscheduled is one. */
enum Phase derives Finite:
  case unscheduled, scheduled, backingOff, started, succeeded, failed, canceled, timedOut

/**
 * Which timer fired. The history event records it, so a Contract that did not check it would pass a
 * run that timed out on the wrong deadline.
 */
enum TimeoutType derives Finite:
  case scheduleToClose, scheduleToStart, startToClose

/** The attempt count is `0..attemptBound`. */
final case class ProtocolState(
    phase: Phase,
    attempts: Int,
    scheduleToClose: Timeout,
    scheduleToStart: Timeout,
    startToClose: Timeout
)

enum ProtocolFact derives Finite:
  case nexusOperationScheduled, nexusOperationStarted, nexusOperationCompleted,
    nexusOperationFailed,
    nexusOperationCanceled
  case nexusOperationTimedOut(timeoutType: TimeoutType)

  /** The attempt count, read through the observation of that name: no history event records it. */
  case pendingAttempts

type ProtocolStep = Step[ProtocolState, Outcome, ProtocolFact]

final case class NexusCallerState(operation: ProtocolState, worker: WorkerState)

// ### Signature
//
// Parties are names the feature declares by using them. The reserved party system is the server.
// A fault is an ordinary action of a declared party, and a timer is system behavior the machine
// owns, so neither is a separate kind.

val caller = Party()
val handler = Party()
val network = Party()

// An operation is scheduled by a caller workflow, and recorded data names one by its scheduled
// event: every history event of the operation carries that event's id.

val workflow = Entity()
val operation = Entity(key = "scheduledEvent", refer = Map("caller" -> workflow))

/** The inputs, apart because the deadline timers take their names. */
object Inputs:
  val scheduleToClose = input[Timeout]
  val scheduleToStart = input[Timeout]
  val startToClose = input[Timeout]
  val reply = input[Reply]
  val resolution = input[Resolution]

val schedule = action(caller)
  .input(Inputs.scheduleToClose)
  .input(Inputs.scheduleToStart)
  .input(Inputs.startToClose)
  .creates(operation)
  .schema[ScheduleNexusOperationCommandAttributes]

val handlerReply = action(handler)
  .on(operation)
  .input(Inputs.reply)
  .schema[StartOperationResponse]
  .schema[HandlerError]
  .example(Reply.handlerError(false), "BadRequest")
  .example(Reply.handlerError(true), "Internal")

/**
 * The Nexus HTTP completion carries no protobuf message, so it declares no schema and its classes
 * are names the realization interprets. The result text is metadata of the action, not a domain a
 * state holds.
 */
val complete = action(handler).on(operation).input(Inputs.resolution).results("Delivery")

val transportFault = action(network) on operation

// The handler's worker stopping is the worker's own action, `workerStop`: an action that names no
// entity is behavior no entity records. The Run records the fault, but nothing recorded names the
// operation, so the machines keep their state and record nothing at it.

// A retryable attempt failure writes no history event, so the attempt count is read back through
// DescribeWorkflowExecution. Every other evidence name resolves against the realization's catalog,
// which is why only a derived observation is declared.

val pendingAttempts = Observation(on = operation, read = "attempts")

given Ok[Outcome] = Ok(Outcome.accepted)

/** One of the operation's deadlines firing, as the product machine sees it. */
val timeout = timer

/** The protocol's backoff timer and its three deadlines. */
val backoff = timer
val scheduleToClose = timer
val scheduleToStart = timer
val startToClose = timer

given Finite[ProtocolState] =
  // The lifter reads this Int bound; the Go model evaluator uses it to enumerate protocol states.
  given Finite[Int] = Finite.upTo(Protocol.attemptBound)
  Finite.derived

// The bounds of the Queries, beside three and four (shared.Bounds). Nine actions are enabled before
// the operation is scheduled and eleven once it is, so an exact sequence of two is found among
// ninety-nine candidates, one of three among about a thousand and one of four among about ten
// thousand.
val two = Limits(steps = 2, actions = 2, search = 512)
val control = Limits(steps = 8, actions = 8, search = 262144)

// ### The product machine
//
// What an operation does, with no account of how. Every Property written against it is carried to
// the protocol machine by the refinement declared there.

object Product:
  import ProductPhase.*

  /** The four phases the product machine ends on. */
  def productTerminal(s: ProductState) = s.phase.in(succeeded, failed, canceled, timedOut)

  object effects:
    import ProductFact.*

    /**
     * The handler's reply to the server's start request. An operation that has not started yet is
     * the only one a reply can move.
     */
    def handlerReplyStep(s: ProductState, reply: Reply) =
      if s.phase != scheduled then disabled
      else
        reply match
          case Reply.syncSuccess       => enter(ProductState(succeeded), nexusOperationCompleted)
          case Reply.async             => enter(ProductState(started), nexusOperationStarted)
          case Reply.operationFailed   => enter(ProductState(failed), nexusOperationFailed)
          case Reply.operationCanceled => enter(ProductState(canceled), nexusOperationCanceled)
          // A retryable handler error leaves the operation where it is: the product machine does not
          // know about backing off, which is the whole of what the protocol machine adds.
          case Reply.handlerError(retryable) =>
            if retryable then disabled else enter(ProductState(failed), nexusOperationFailed)

    /**
     * An asynchronous completion. A completion that arrives after the operation is over is not
     * found, and changes nothing.
     */
    def completeStep(s: ProductState, resolution: Resolution) =
      if productTerminal(s) then List(Step(Outcome.notFound, s))
      else
        resolution match
          case Resolution.succeeded => enter(ProductState(succeeded), nexusOperationCompleted)
          case Resolution.failed    => enter(ProductState(failed), nexusOperationFailed)
          case Resolution.canceled  => enter(ProductState(canceled), nexusOperationCanceled)

    /**
     * A transport fault is an ordinary action of the network. The product machine cannot see one:
     * whether a delivery was retried is the protocol's account of how, not what.
     */
    def transportFaultStep(@unused s: ProductState): List[ProductStep] = disabled

    /**
     * The handler's worker stopping is a fault the Run records and the operation does not feel. The
     * product machine cannot see it, like the transport fault: a step that kept the state and
     * recorded nothing would be indistinguishable from a stutter, and the refinement would read
     * every stutter as this step.
     */
    def workerStopStep(@unused s: ProductState): List[ProductStep] = disabled

    /**
     * One of the operation's deadlines firing. Which deadline is the protocol's account of how, so
     * the product machine has one timer, and it fires while the operation runs.
     */
    def timeoutStep(s: ProductState) =
      if s.phase.in(scheduled, started) then enter(ProductState(timedOut), nexusOperationTimedOut)
      else disabled

  /** Every fact the product machine records is confirmed by the history event of its name. */
  val nexusProduct = machine[ProductState, Outcome, ProductFact] {
    forEntity(operation)
    starts(ProductState(ProductPhase.scheduled))
    ends(productTerminal)
    steps(
      handlerReply ~> effects.handlerReplyStep,
      complete ~> effects.completeStep,
      transportFault ~> effects.transportFaultStep,
      workerStop ~> effects.workerStopStep,
      timeout ~> effects.timeoutStep
    )
  }

  // A same-step claim names the action it is about under `when` and holds of the step that action
  // produces; a transition claim holds of the state before and the step after. A functional Query
  // realizes a same-step claim, because the Case's Contract is the claim's clause triggered by the
  // action the Case performs; a transition claim is searched and verified, never realized.

  object properties:
    /**
     * Once an operation is over, no step changes its phase. Declared on the product machine and read
     * on the protocol machine through the map.
     */
    val terminalIsFinal = nexusProduct.property.once(productTerminal).keeps(_.phase)

// ### The protocol machine
//
// How the server gets there: the retry the product machine cannot see, the three timers the
// schedule command sets, and the attempt count a retryable failure raises. Written against the same
// actions, so a Property proved on the product machine is carried here by the refinement.
//
// The machine begins before the operation exists: a state structure has no "no instance yet"
// member, so unscheduled is that member, and it is what makes the three deadline fields reachable
// at anything but their first value -- the schedule command is what sets them.
//
// Not here, for reasons recorded rather than silent: the cancel field and its rows (fn-79), and the
// concurrency-limit rejection, which names no operation and is not modeled until a Query needs it.

object Protocol:
  import Phase.*

  /**
   * Bounds the attempt count. Nothing wires the Limits into a machine's state, so the bound is
   * written here; the Go model evaluator checks each require precondition against it.
   */
  val attemptBound = 2

  def validAttempts(a: Int) = 0 <= a && a <= attemptBound

  /** A retry past the bound stays at it, rather than wrapping as `Fin` arithmetic would. */
  def saturatingSucc(a: Int) =
    require(validAttempts(a))
    if a < attemptBound then a + 1 else a

  /** The four phases the design ends on. A completion that arrives after one of them is not found. */
  def terminalPhase(p: Phase) = p.in(succeeded, failed, canceled, timedOut)

  /** Scheduled and not yet over: the phases a completion resolves and a timer can fire in. */
  def running(p: Phase) = p.in(scheduled, backingOff, started)

  /**
   * How a protocol state reads as a product state. A phase of the same name is that phase; backing
   * off is still scheduled, because the product machine cannot see a retry; and an operation not
   * yet scheduled reads as scheduled, because the product machine begins there. Every other field
   * is hidden, which is what a map that does not read it says.
   */
  def productOf(s: ProtocolState) = s.phase match
    case Phase.unscheduled | Phase.scheduled | Phase.backingOff =>
      ProductState(ProductPhase.scheduled)
    case Phase.started   => ProductState(ProductPhase.started)
    case Phase.succeeded => ProductState(ProductPhase.succeeded)
    case Phase.failed    => ProductState(ProductPhase.failed)
    case Phase.canceled  => ProductState(ProductPhase.canceled)
    case Phase.timedOut  => ProductState(ProductPhase.timedOut)

  /** Where every path begins: before the operation exists, with every deadline at its first value. */
  val unscheduled =
    ProtocolState(Phase.unscheduled, 0, Timeout.unset, Timeout.unset, Timeout.unset)

  object effects:
    import ProtocolFact.*

    /**
     * The caller's schedule command. It names the operation's three deadlines, and every one of them
     * is a state field because whether a timer fires is a question about the operation and not about
     * the command that started it.
     */
    def scheduleStep(
        s: ProtocolState,
        scheduleToClose: Timeout,
        scheduleToStart: Timeout,
        startToClose: Timeout
    ) =
      if s.phase != Phase.unscheduled then disabled
      else
        enter(
          ProtocolState(scheduled, 0, scheduleToClose, scheduleToStart, startToClose),
          nexusOperationScheduled
        )

    /**
     * The handler's reply to the server's start request. What the product machine cannot see is the
     * last arm: a retryable failure backs the operation off and raises its attempt count, and the
     * count is read back through the pendingAttempts observation because no history event records
     * it.
     */
    def handlerReplyStep(s: ProtocolState, reply: Reply) =
      require(validAttempts(s.attempts))
      if s.phase != scheduled then disabled
      else
        reply match
          case Reply.syncSuccess       => enter(s.copy(phase = succeeded), nexusOperationCompleted)
          case Reply.async             => enter(s.copy(phase = started), nexusOperationStarted)
          case Reply.operationFailed   => enter(s.copy(phase = failed), nexusOperationFailed)
          case Reply.operationCanceled => enter(s.copy(phase = canceled), nexusOperationCanceled)
          case Reply.handlerError(retryable) =>
            if !retryable then enter(s.copy(phase = failed), nexusOperationFailed)
            else
              enter(
                s.copy(phase = backingOff, attempts = saturatingSucc(s.attempts)),
                ProtocolFact.pendingAttempts
              )

    /** A transport fault is the same failure arriving as a dropped delivery rather than as a reply. */
    def transportFaultStep(s: ProtocolState) =
      require(validAttempts(s.attempts))
      if s.phase != scheduled then disabled
      else
        enter(
          s.copy(phase = backingOff, attempts = saturatingSucc(s.attempts)),
          ProtocolFact.pendingAttempts
        )

    /**
     * The handler's worker stopping is a fault the Run records and the operation does not feel, so
     * the step keeps the state and records nothing. On a path it is confirmed by the evidence of the
     * step after it, and the Case says so in a Known Gap.
     */
    def workerStopStep(s: ProtocolState): List[ProtocolStep] = stay(s)

    /**
     * An asynchronous completion. Before a start, the server records a Started event first, which is
     * why the evidence is two facts and not one -- and why the product machine, which has no
     * backingOff phase to have skipped, could write the completion alone.
     */
    def completeStep(s: ProtocolState, resolution: Resolution) =
      if terminalPhase(s.phase) then List(Step(Outcome.notFound, s))
      else if s.phase == Phase.unscheduled then disabled
      else
        val startedFirst =
          if s.phase != started then List(nexusOperationStarted) else Nil
        resolution match
          case Resolution.succeeded =>
            enter(s.copy(phase = succeeded), (startedFirst ++ List(nexusOperationCompleted))*)
          case Resolution.failed =>
            enter(s.copy(phase = failed), (startedFirst ++ List(nexusOperationFailed))*)
          case Resolution.canceled =>
            enter(s.copy(phase = canceled), (startedFirst ++ List(nexusOperationCanceled))*)

    /**
     * The backoff timer. It is what makes backingOff a phase the operation leaves rather than a state
     * it is stuck in, and it records nothing: a retry writes no history event.
     */
    def backoffStep(s: ProtocolState): List[ProtocolStep] =
      if s.phase != backingOff then disabled else enter(s.copy(phase = scheduled))

    /**
     * The schedule-to-close deadline covers the whole operation, so it fires in every running phase
     * -- and only when the schedule command set it.
     */
    def scheduleToCloseStep(s: ProtocolState) =
      if running(s.phase) && s.scheduleToClose == Timeout.expires then
        enter(s.copy(phase = timedOut), nexusOperationTimedOut(TimeoutType.scheduleToClose))
      else disabled

    /**
     * The schedule-to-start deadline covers the wait for the handler to accept, so it stops at the
     * start.
     */
    def scheduleToStartStep(s: ProtocolState) =
      if s.phase.in(scheduled, backingOff) && s.scheduleToStart == Timeout.expires then
        enter(s.copy(phase = timedOut), nexusOperationTimedOut(TimeoutType.scheduleToStart))
      else disabled

    /** The start-to-close deadline covers the handler's own work, so it begins at the start. */
    def startToCloseStep(s: ProtocolState) =
      if s.phase == started && s.startToClose == Timeout.expires then
        enter(s.copy(phase = timedOut), nexusOperationTimedOut(TimeoutType.startToClose))
      else disabled

  /**
   * A timeout is confirmed by the one timed-out event, whichever deadline fired, and the attempt
   * count by its observation.
   */
  val nexusProtocol = machine[ProtocolState, Outcome, ProtocolFact] {
    forEntity(operation)
    refines(Product.nexusProduct)(productOf)
    starts(unscheduled)
    ends(s => terminalPhase(s.phase))
    unobservable(backoff)
    evidence {
      case ProtocolFact.nexusOperationTimedOut(_) => "nexusOperationTimedOut"
      case ProtocolFact.pendingAttempts           => pendingAttempts.name
    }
    steps(
      schedule ~> effects.scheduleStep,
      handlerReply ~> effects.handlerReplyStep,
      complete ~> effects.completeStep,
      transportFault ~> effects.transportFaultStep,
      workerStop ~> effects.workerStopStep,
      backoff ~> effects.backoffStep,
      scheduleToClose ~> effects.scheduleToCloseStep,
      scheduleToStart ~> effects.scheduleToStartStep,
      startToClose ~> effects.startToCloseStep
    )
  }

  object properties:
    /** A synchronous reply settles the operation as succeeded, and the completed event records it. */
    val syncSucceeds = nexusProtocol.property when handlerReply(Reply.syncSuccess) holds { s =>
      s.state.phase == Phase.succeeded && s.records(ProtocolFact.nexusOperationCompleted)
    }

    /** An asynchronous reply starts the operation, and the started event records it. */
    val asyncStarts = nexusProtocol.property when handlerReply(Reply.async) holds { s =>
      s.state.phase == Phase.started && s.records(ProtocolFact.nexusOperationStarted)
    }

    /**
     * A successful completion is recorded by the completed event. Neither the phase nor the outcome
     * is fixed: a completion resolves any running phase, and accepted is every earlier step's
     * outcome too, so a clause fixing it would be answered before the completion.
     */
    val completionSucceeds = nexusProtocol.property when complete(Resolution.succeeded) holds
      (_.records(ProtocolFact.nexusOperationCompleted))

    /** A failed completion is recorded by the failed event. */
    val completionFails = nexusProtocol.property when complete(Resolution.failed) holds
      (_.records(ProtocolFact.nexusOperationFailed))

    /**
     * A non-retryable handler error settles the operation as failed, and the failed event records
     * it.
     */
    val handlerErrorFails =
      nexusProtocol.property when handlerReply(Reply.handlerError(false)) holds { s =>
        s.state.phase == Phase.failed && s.records(ProtocolFact.nexusOperationFailed)
      }

    /**
     * Succeeded on the second attempt of an operation with no deadline set. A claim fixes one state,
     * so every field is named.
     */
    val succeededOnRetry =
      ProtocolState(Phase.succeeded, 1, Timeout.unset, Timeout.unset, Timeout.unset)

    /**
     * A synchronous reply to the retried attempt settles the operation as succeeded on its second
     * attempt: the count the retryable failure raised is still one, and the completed event records
     * the reply.
     */
    val retrySucceeds = nexusProtocol.property when handlerReply(Reply.syncSuccess) holds { s =>
      s.state == succeededOnRetry && s.records(ProtocolFact.nexusOperationCompleted)
    }

    /**
     * The schedule-to-start deadline settles an operation no handler started as timed out, and the
     * timed-out event records which deadline it was.
     */
    val scheduleToStartFires = nexusProtocol.property when scheduleToStart holds { s =>
      s.state.phase == Phase.timedOut &&
      s.records(ProtocolFact.nexusOperationTimedOut(TimeoutType.scheduleToStart))
    }

    /** The start-to-close deadline settles a started operation no handler completed as timed out. */
    val startToCloseFires = nexusProtocol.property when startToClose holds { s =>
      s.state.phase == Phase.timedOut &&
      s.records(ProtocolFact.nexusOperationTimedOut(TimeoutType.startToClose))
    }

  /**
   * The paths the Queries run, then the Queries. Each path is one upstream functional test's shape,
   * from before the operation exists: the schedule command, then the side effects that settle the
   * operation. A schedule that sets no deadline is `schedule()`, each input at `unset`.
   */
  object queries:
    val syncReplied = nexusProtocol.scenario.actions(schedule(), handlerReply(Reply.syncSuccess))

    val asyncThenSucceeded = nexusProtocol.scenario
      .actions(schedule(), handlerReply(Reply.async), complete(Resolution.succeeded))

    val asyncThenFailed = nexusProtocol.scenario
      .actions(schedule(), handlerReply(Reply.async), complete(Resolution.failed))

    val nonRetryableError =
      nexusProtocol.scenario.actions(schedule(), handlerReply(Reply.handlerError(false)))

    /**
     * The retryable error backs the operation off; the backoff timer fires and records nothing; the
     * retried attempt is answered synchronously.
     */
    val retriedThenSucceeded = nexusProtocol.scenario.actions(
      schedule(),
      handlerReply(Reply.handlerError(true)),
      backoff,
      handlerReply(Reply.syncSuccess)
    )

    /**
     * The schedule command sets the schedule-to-start deadline; the handler's worker stops, so
     * nothing answers the start request; the deadline fires. The worker stops after the schedule in
     * the operation's order, where the stop changes nothing; the realization stops it before the
     * workflow starts, where the stop cannot race the dispatch.
     */
    val scheduleToStartExpires = nexusProtocol.scenario
      .actions(schedule(Inputs.scheduleToStart := expires), workerStop, scheduleToStart)

    /**
     * The schedule command sets the start-to-close deadline; the handler accepts asynchronously and
     * never completes; the deadline fires.
     */
    val startToCloseExpires = nexusProtocol.scenario
      .actions(schedule(Inputs.startToClose := expires), handlerReply(Reply.async), startToClose)

    // The design's seven: sync success, async reply then succeeded callback, async reply then failed
    // callback, non-retryable handler error, retryable handler error then sync success after one
    // backoff, schedule-to-start timeout with the handler's worker stopped, start-to-close timeout
    // after an asynchronous reply. Each finds its same-step claim on its path and is realized as a
    // Case. The product claim is verified over every trace of one path, outside `functionalQueries`,
    // because a verify Query realizes nothing.

    val syncCompletion = (query find properties.syncSucceeds in syncReplied limits two total 384)
      .expect(satisfied)
      .explore(
        Exploration(
          "nexusDeadlines",
          Vector(
            Variation(
              0,
              Vector(
                Alternative("startDeadline", 30, Vector(schedule(Inputs.startToClose := expires))),
                Alternative(
                  "scheduleDeadline",
                  20,
                  Vector(schedule(Inputs.scheduleToStart := expires))
                ),
                Alternative("unbounded", 10, Vector(schedule()))
              )
            )
          ),
          runs = 1,
          edits = 1,
          dropPrefix = true
        )
      )
    val asyncCompletion =
      (query find properties.completionSucceeds in asyncThenSucceeded limits three total 576)
        .expect(inconclusive(Reason.explanationsDisagree))
    val asyncFailure =
      (query find properties.completionFails in asyncThenFailed limits three total 576)
        .expect(inconclusive(Reason.explanationsDisagree))
    val handlerError =
      (query find properties.handlerErrorFails in nonRetryableError limits two total 384)
        .expect(inconclusive(Reason.neverEvaluated))
    val retry = (query find properties.retrySucceeds in retriedThenSucceeded limits four total 768)
      .expect(inconclusive(Reason.explanationsDisagree))
    val scheduleToStartTimeout =
      (query find properties.scheduleToStartFires in scheduleToStartExpires limits three total 576)
        .expect(inconclusive(Reason.neverEvaluated))
    val startToCloseTimeout =
      (query find properties.startToCloseFires in startToCloseExpires limits three total 576)
        .expect(inconclusive(Reason.neverEvaluated))

    /**
     * A product claim on a protocol path, read through the refinement the protocol machine declares.
     */
    val terminalHolds =
      query verify Product.properties.terminalIsFinal in asyncThenSucceeded limits three total 576

    /** The functional Queries in declaration order. */
    val functionalQueries = Vector(
      syncCompletion,
      asyncCompletion,
      asyncFailure,
      handlerError,
      retry,
      scheduleToStartTimeout,
      startToCloseTimeout
    )

// ### The handler's worker

object HandlerWorker:
  /**
   * The caller's view of the handler's worker: it stops and it serves. It never resumes, because an
   * action no sync line names would stay executable on its own and admit a stop, a resume and then a
   * reply; the operation's timers settle every state a stop leaves.
   */
  val handlerWorker = shared.worker.Polling.polling.restrict(workerStop, serve)

// ### The operation and the handler's worker
//
// The protocol machine's worker stop is a stutter row: the operation cannot see its handler's
// worker, so the schedule-to-start Scenario orders the stop before the request by convention.
// Composed with the worker of the handler's task queue, the stop is the worker's own phase change
// and every reply is the worker serving, so a reply has a row only while the worker polls. No
// functional Query reads the composition; it is what the cross-entity claim is verified over.

object NexusCaller:
  val pollingWorker = WorkerState(WorkerPhase.polling)

  val nexusCaller =
    compose[NexusCallerState](
      _.operation -> Protocol.nexusProtocol,
      _.worker -> HandlerWorker.handlerWorker
    )
      .sync(_.operation -> workerStop, _.worker -> workerStop)
      .sync(_.operation -> handlerReply, _.worker -> serve)
      .ends(s => Protocol.terminalPhase(s.operation.phase))

  object properties:
    /**
     * The cross-entity claim: every reply, of any class, leaves the handler's worker polling, so no
     * handler replies while its worker is stopped.
     */
    val repliedByPollingWorker = nexusCaller.property
      .whenAction(nexusCaller.synced(_.operation -> handlerReply))
      .holds(_.state.worker.phase == WorkerPhase.polling)

  object queries:
    /**
     * A retryable reply backs the operation off; the handler's worker then stops, so the retried
     * attempt is never answered and the schedule-to-start deadline fires. The start is stated: a
     * default would take the worker's from shared/worker/Worker.scala.
     */
    val repliedThenStopped = nexusCaller.scenario
      .starts(NexusCallerState(Protocol.unscheduled, pollingWorker))
      .actions(
        nexusCaller.own(_.operation, schedule(Inputs.scheduleToStart := expires)),
        nexusCaller.synced(_.operation -> handlerReply(Reply.handlerError(true))),
        nexusCaller.synced(_.operation -> workerStop),
        nexusCaller.own(_.operation, scheduleToStart)
      )

    /** The cross-entity claim, verified over that path. */
    val stoppedWorkerRepliesNothing =
      query verify properties.repliedByPollingWorker in repliedThenStopped limits four total 1536

// ### The control
//
// A caller design that predicts success for a failed completion, which the forged-completion Query
// must refuse. It keeps its own family, and its Definition IDs hang off this object.

object Control:
  // Moved from temporal.nexuscaller; the pin keeps its Definition IDs.
  given DefinitionScope = DefinitionScope("temporal.nexuscaller.Control$")

  given family: Family = Family("temporal.nexus.control")

  val inspect = action(caller).on(operation)

  // A failed callback completes the operation two ways: as the success the control forges, and as
  // the failure the runtime still sends.
  val forged = choice
  val sent = choice

  object effects:
    def inspectStep(s: ProtocolState): List[ProtocolStep] = stay(s)

    // The control deliberately predicts success for a failed callback. The runtime still sends
    // failure.
    def forgedComplete(s: ProtocolState, resolution: Resolution) =
      if resolution == Resolution.failed then
        choose(
          forged -> Protocol.effects.completeStep(s, Resolution.succeeded),
          sent -> Protocol.effects.completeStep(s, resolution)
        )
      else Protocol.effects.completeStep(s, resolution)

  // The protocol machine without its refinement, its completion forged and an inspection added.
  // It is declared, not derived from nexusProtocol: a derivation lifts its source machines, and
  // nexus-control.json holds this machine alone.
  val forgedCompletion = machine[ProtocolState, Outcome, ProtocolFact] {
    forEntity(operation)
    starts(Protocol.unscheduled)
    ends(s => Protocol.terminalPhase(s.phase))
    unobservable(backoff)
    evidence {
      case ProtocolFact.nexusOperationTimedOut(_) => "nexusOperationTimedOut"
      case ProtocolFact.pendingAttempts           => pendingAttempts.name
    }
    steps(
      schedule ~> Protocol.effects.scheduleStep,
      handlerReply ~> Protocol.effects.handlerReplyStep,
      complete ~> effects.forgedComplete,
      inspect ~> effects.inspectStep,
      transportFault ~> Protocol.effects.transportFaultStep,
      workerStop ~> Protocol.effects.workerStopStep,
      backoff ~> Protocol.effects.backoffStep,
      scheduleToClose ~> Protocol.effects.scheduleToCloseStep,
      scheduleToStart ~> Protocol.effects.scheduleToStartStep,
      startToClose ~> Protocol.effects.startToCloseStep
    )
  }

  object properties:
    /**
     * A failed completion is recorded as completed: what the control predicts and no runtime sends.
     */
    val forgedSuccess = forgedCompletion.property when complete(Resolution.failed) holds
      (_.records(ProtocolFact.nexusOperationCompleted))

  object queries:
    /** The control inspects the operation around a failed completion. */
    val inspectedFailure = Control.forgedCompletion.scenario.actions(
      schedule(),
      inspect,
      handlerReply(Reply.async),
      inspect,
      complete(Resolution.failed)
    )

    /** The forged control, which every modeled execution that explains the evidence refutes. */
    val forgedCompletion =
      (query find properties.forgedSuccess in inspectedFailure limits control total 960)
        .expect(
          RunExpectation(
            Conformance.inconclusive,
            PropertyOutcome.violated,
            contract = PropertyOutcome.violated,
            disposition = Disposition.stoppedByMonitor,
            cleanup = Cleanup.succeeded,
            reason = Some(Reason.everyExplanationViolates)
          )
        )
        .explore(
          Exploration(
            "nexusControl",
            Vector(
              Variation(
                1,
                Vector(
                  Alternative("twice", 20, Vector(inspect, inspect)),
                  Alternative("once", 10, Vector(inspect)),
                  Alternative("none", 0, Vector.empty)
                )
              )
            ),
            runs = 1,
            edits = 8,
            dropPrefix = true
          )
        )

// ### The checked-in IR files of the Nexus caller Model (umpire.irFile).

object Files:
  // The functional Queries and the realization that runs them are roots beside the machines: Go
  // lowers each Query's witness through the realization into a Testpilot Case (tools/umpire/lower).
  // The product claim read on a protocol path and the cross-entity Query, which carries the
  // composition with the handler's worker and its claim, are roots too.
  val nexusCallerFile = irFile("nexus-caller")(
    Product.nexusProduct,
    Protocol.nexusProtocol,
    HandlerWorker.handlerWorker,
    NexusCaller.nexusCaller,
    shared.worker.Polling.polling,
    Protocol.queries.functionalQueries,
    Protocol.queries.terminalHolds,
    NexusCaller.queries.stoppedWorkerRepliesNothing,
    NexusRealization.asyncNexus
  )

  // The forged completion a caller must refuse, and the realization that offers it.
  val nexusControlFile =
    irFile("nexus-control")(Control.queries.forgedCompletion, NexusRealization.forgedCompletion)
