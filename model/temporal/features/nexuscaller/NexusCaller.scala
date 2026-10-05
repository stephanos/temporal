/* The Nexus caller-side Model: one workflow-scheduled Nexus operation, as the caller sees it. The
 * product machine says what an operation does, the protocol machine says how the server gets there
 * and refines it, and the functional Queries are one per side effect that settles the operation.
 * No cancellation (fn-79) and no concurrency-limit setup parameter.
 *
 * Read top to bottom: the types; the signature (the entities, the inputs, the parties with their
 * actions, the derived observation, the timers, the bounds and the control's choices); then one
 * object per machine, each before the machines that use it -- NexusProduct, the product machine;
 * NexusProtocol, the protocol machine that refines it; HandlerWorker, the handler's worker;
 * NexusCaller, the protocol with that worker; ForgedCompletion, the forged control a caller must
 * refuse -- and last exports, its IR files. A machine object reads its header (entity, init, end,
 * evidence), then its sections in order: states, refinement, effects, rules, properties and
 * queries. Realization.scala realizes it; closepolicy/ holds the close and reset designs.
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
import shared.worker.{worker, Phase as WorkerPhase, State as WorkerState}
import CallerFamily.given
import Timeout.expires

// Moved from temporal.nexuscaller; the pin keeps its Definition IDs and type names.
given DefinitionScope = DefinitionScope("temporal.nexuscaller.Model$package$")

/** The family of the caller's machines; the control takes `ControlFamily`. */
object CallerFamily:
  given family: Family = Family("temporal.nexus.caller")

/** The control's family, which `ForgedCompletion` names where it extends `Machine`. */
object ControlFamily:
  val family: Family = Family("temporal.nexus.control")

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
// owns, so neither is a separate kind. Each party is an actor object whose members are the actions
// it takes, and the timers are grouped in sections. Actor and section objects are transparent to
// Definition IDs, so every action keeps the ID the file's pin gives it.

// An operation is scheduled by a caller workflow, and recorded data names one by its scheduled
// event: every history event of the operation carries that event's id.

val workflow = Entity()
val operation = Entity(key = "scheduledEvent", refer = Map("caller" -> workflow))

// The inputs: the schedule's deadlines, which the deadline timers no longer collide with, the
// handler's reply and its completion's resolution.
val scheduleToClose = input[Timeout]
val scheduleToStart = input[Timeout]
val startToClose = input[Timeout]
val reply = input[Reply]
val resolution = input[Resolution]

/** The caller workflow, which schedules the operation. */
object caller extends Actor:
  val schedule = action(this)
    .input(scheduleToClose)
    .input(scheduleToStart)
    .input(startToClose)
    .creates(operation)
    .schema[ScheduleNexusOperationCommandAttributes]

/** The endpoint's handler, which replies to the start and completes the operation. */
object handler extends Actor:
  val handlerReply = action(this)
    .on(operation)
    .input(reply)
    .schema[StartOperationResponse]
    .schema[HandlerError]
    .example(Reply.handlerError(false), "BadRequest")
    .example(Reply.handlerError(true), "Internal")

  /**
   * The Nexus HTTP completion carries no protobuf message, so it declares no schema and its classes
   * are names the realization interprets. The result text is metadata of the action, not a domain a
   * state holds.
   */
  val complete = action(this).on(operation).input(resolution).results("Delivery")

/** The network between the caller and the handler, which can fail a transport. */
object network extends Actor:
  val transportFault = action(this) on operation

// The handler's worker stopping is the worker's own action, `worker.workerStop`: an action that
// names no entity is behavior no entity records. The Run records the fault, but nothing recorded
// names the operation, so the machines keep their state and record nothing at it.

// A retryable attempt failure writes no history event, so the attempt count is read back through
// DescribeWorkflowExecution. Every other evidence name resolves against the realization's catalog,
// which is why only a derived observation is declared.

val pendingAttempts = Observation(on = operation, read = "attempts")

given Ok[Outcome] = Ok(Outcome.accepted)

/** One of the operation's deadlines firing, as the product machine sees it, and the backoff. */
object timers extends Section:
  val timeout = timer
  val backoff = timer

/** The protocol's three deadlines, each armed by the schedule's input of its name. */
object deadline extends Section:
  val scheduleToClose = timer
  val scheduleToStart = timer
  val startToClose = timer

/**
 * Bounds the attempt count. Nothing wires the Limits into a machine's state, so the bound is
 * written here, beside the state type's `Finite`, which reads it before any machine exists; the Go
 * model evaluator checks each require precondition against it.
 */
val attemptBound = 2

given Finite[ProtocolState] =
  // The lifter reads this Int bound; the Go model evaluator uses it to enumerate protocol states.
  given Finite[Int] = Finite.upTo(attemptBound)
  Finite.derived

// The bounds of the Queries, beside three and four (shared.Bounds). Nine actions are enabled before
// the operation is scheduled and eleven once it is, so an exact sequence of two is found among
// ninety-nine candidates, one of three among about a thousand and one of four among about ten
// thousand.
val two = Limits(steps = 2, actions = 2, search = 512)
val control = Limits(steps = 8, actions = 8, search = 262144)

// A failed callback completes the forged control's operation two ways: as the success the control
// forges, and as the failure the runtime still sends.
val forged = choice
val sent = choice

// ### The product machine
//
// What an operation does, with no account of how. Every Property written against it is carried to
// the protocol machine by the refinement declared there.

object NexusProduct extends Machine[ProductState, Outcome, ProductFact]:
  import ProductPhase.*

  val entity = operation
  val init = ProductState(scheduled)
  def end(s: State) = states.productTerminal(s)

  /** The product's phase sets. */
  object states extends Section:
    /** The four phases the product machine ends on. */
    def productTerminal(s: State) = s.phase.in(succeeded, failed, canceled, timedOut)

  /** Every fact the product machine records is confirmed by the history event of its name. */
  object effects extends Section:
    import ProductFact.*

    def succeed(s: State) = enter(s.copy(phase = succeeded), nexusOperationCompleted)

    def start(s: State) = enter(s.copy(phase = started), nexusOperationStarted)

    def fail(s: State) = enter(s.copy(phase = failed), nexusOperationFailed)

    def cancel(s: State) = enter(s.copy(phase = canceled), nexusOperationCanceled)

    /** A completion that arrives after the operation is over is not found, and changes nothing. */
    def notFound(s: State): List[ProductStep] = List(Step(Outcome.notFound, s))

    /**
     * One of the operation's deadlines firing. Which deadline is the protocol's account of how, so
     * the product machine has one timer.
     */
    def timeOut(s: State) = enter(s.copy(phase = timedOut), nexusOperationTimedOut)

  object rules extends Rules(_.phase):
    // The handler's reply to the server's start request moves only an operation that has not
    // started yet. A retryable handler error leaves the operation where it is, so no rule fires it:
    // the product machine does not know about backing off, which is the whole of what the protocol
    // machine adds.
    in(scheduled) {
      handler.handlerReply(Reply.syncSuccess) ~> effects.succeed
      handler.handlerReply(Reply.async) ~> effects.start
      handler.handlerReply(Reply.operationFailed) ~> effects.fail
      handler.handlerReply(Reply.operationCanceled) ~> effects.cancel
      handler.handlerReply(Reply.handlerError(false)) ~> effects.fail
    }

    // An asynchronous completion settles a running operation, and is not found once it is over.
    when(s => states.productTerminal(s))(handler.complete ~> effects.notFound)
    in(scheduled, started) {
      handler.complete(Resolution.succeeded) ~> effects.succeed
      handler.complete(Resolution.failed) ~> effects.fail
      handler.complete(Resolution.canceled) ~> effects.cancel
    }

    // A transport fault is an ordinary action of the network, and the handler's worker stopping is a
    // fault the Run records. The product machine sees neither: whether a delivery was retried is the
    // protocol's account of how, not what, and a step that kept the state and recorded nothing would
    // be indistinguishable from a stutter, which the refinement would read as this step.
    disabled(network.transportFault, worker.workerStop)

    // The deadline fires while the operation runs.
    in(scheduled, started)(timers.timeout ~> effects.timeOut)

  // A same-step claim names the action it is about under `when` and holds of the step that action
  // produces; a transition claim holds of the state before and the step after. A functional Query
  // realizes a same-step claim, because the Case's Contract is the claim's clause triggered by the
  // action the Case performs; a transition claim is searched and verified, never realized.

  object properties extends Section:
    /**
     * Once an operation is over, no step changes its phase. Declared on the product machine and read
     * on the protocol machine through the map.
     */
    val terminalIsFinal = property.once(states.productTerminal).keeps(_.phase)

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

object NexusProtocol extends Machine[ProtocolState, Outcome, ProtocolFact]:
  import Phase.*

  val entity = operation

  /** Where every path begins: before the operation exists, with every deadline at its first value. */
  val init = ProtocolState(Phase.unscheduled, 0, Timeout.unset, Timeout.unset, Timeout.unset)
  def end(s: State) = states.terminalPhase(s.phase)

  /**
   * A timeout is confirmed by the one timed-out event, whichever deadline fired, and the attempt
   * count by its observation.
   */
  val evidence: PartialFunction[ProtocolFact, String] = {
    case ProtocolFact.nexusOperationTimedOut(_) => "nexusOperationTimedOut"
    case ProtocolFact.pendingAttempts           => pendingAttempts.name
  }

  /** The protocol's phase sets and its attempt count's arithmetic. */
  object states extends Section:
    def validAttempts(a: Int) = 0 <= a && a <= attemptBound

    /** A retry past the bound stays at it, rather than wrapping as `Fin` arithmetic would. */
    def saturatingSucc(a: Int) =
      require(validAttempts(a))
      if a < attemptBound then a + 1 else a

    /** The four phases the design ends on. A completion that arrives after one of them is not found. */
    def terminalPhase(p: Phase) = p.in(succeeded, failed, canceled, timedOut)

    /** Scheduled and not yet over: the phases a completion resolves and a timer can fire in. */
    def running(p: Phase) = p.in(scheduled, backingOff, started)

    /** Waiting for the handler to accept: what the schedule-to-start deadline covers. */
    def waiting(p: Phase) = p.in(scheduled, backingOff)

    /** Every phase once the operation is scheduled, running or over: what a completion answers. */
    def created(p: Phase) = running(p) || terminalPhase(p)

  /** The protocol refines the product: what each of its states reads as there. */
  object refinement extends Refinement(NexusProduct):
    /**
     * A phase of the same name is that phase; backing off is still scheduled, because the product
     * machine cannot see a retry; and an operation not yet scheduled reads as scheduled, because the
     * product machine begins there. Every other field is hidden, which is what a map that does not
     * read it says.
     */
    def toProduct(s: State): ProductState = s.phase match
      case Phase.unscheduled | Phase.scheduled | Phase.backingOff =>
        ProductState(ProductPhase.scheduled)
      case Phase.started   => ProductState(ProductPhase.started)
      case Phase.succeeded => ProductState(ProductPhase.succeeded)
      case Phase.failed    => ProductState(ProductPhase.failed)
      case Phase.canceled  => ProductState(ProductPhase.canceled)
      case Phase.timedOut  => ProductState(ProductPhase.timedOut)

    /** The backoff timer records nothing a Run can read: a retry writes no history event. */
    val unobservable = List(timers.backoff)

  object effects extends Section:
    import ProtocolFact.*

    /**
     * The caller's schedule command. It names the operation's three deadlines, and every one of them
     * is a state field because whether a timer fires is a question about the operation and not about
     * the command that started it.
     */
    def schedule(
        @unused s: State,
        scheduleToClose: Timeout,
        scheduleToStart: Timeout,
        startToClose: Timeout
    ) =
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
    def handlerReply(s: State, reply: Reply) =
      require(states.validAttempts(s.attempts))
      reply match
        case Reply.syncSuccess       => enter(s.copy(phase = succeeded), nexusOperationCompleted)
        case Reply.async             => enter(s.copy(phase = started), nexusOperationStarted)
        case Reply.operationFailed   => enter(s.copy(phase = failed), nexusOperationFailed)
        case Reply.operationCanceled => enter(s.copy(phase = canceled), nexusOperationCanceled)
        case Reply.handlerError(retryable) =>
          if !retryable then enter(s.copy(phase = failed), nexusOperationFailed)
          else
            enter(
              s.copy(phase = backingOff, attempts = states.saturatingSucc(s.attempts)),
              ProtocolFact.pendingAttempts
            )

    /** A transport fault is the same failure arriving as a dropped delivery rather than as a reply. */
    def backOff(s: State) =
      require(states.validAttempts(s.attempts))
      enter(
        s.copy(phase = backingOff, attempts = states.saturatingSucc(s.attempts)),
        ProtocolFact.pendingAttempts
      )

    /**
     * The handler's worker stopping is a fault the Run records and the operation does not feel, so
     * the step keeps the state and records nothing. On a path it is confirmed by the evidence of the
     * step after it, and the Case says so in a Known Gap.
     */
    def keep(s: State): List[ProtocolStep] = stay(s)

    /** A completion that arrives after the operation is over is not found, and changes nothing. */
    def notFound(s: State): List[ProtocolStep] = List(Step(Outcome.notFound, s))

    /**
     * An asynchronous completion. Before a start, the server records a Started event first, which is
     * why the evidence is two facts and not one -- and why the product machine, which has no
     * backingOff phase to have skipped, could write the completion alone.
     */
    def complete(s: State, resolution: Resolution) =
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
    def retry(s: State): List[ProtocolStep] = enter(s.copy(phase = scheduled))

    /** One of the three deadlines firing; the timed-out event records which. */
    def timeOut(s: State, t: TimeoutType) =
      enter(s.copy(phase = timedOut), nexusOperationTimedOut(t))

  object rules extends Rules(_.phase):
    in(unscheduled)(caller.schedule ~> effects.schedule)
    in(scheduled)(handler.handlerReply ~> effects.handlerReply)

    // A completion resolves any running phase, and is not found once the operation is over; an
    // operation not yet scheduled has nothing to complete.
    when(s => states.terminalPhase(s.phase))(handler.complete ~> effects.notFound)
    when(s => states.running(s.phase))(handler.complete ~> effects.complete)

    in(scheduled)(network.transportFault ~> effects.backOff)
    when(_ => true)(worker.workerStop ~> effects.keep)
    in(backingOff)(timers.backoff ~> effects.retry)

    // Each deadline fires only when the schedule command set it. Schedule-to-close covers the whole
    // operation, schedule-to-start the wait for the handler to accept, and start-to-close the
    // handler's own work.
    when(s => states.running(s.phase) && s.scheduleToClose == Timeout.expires) {
      deadline.scheduleToClose ~> (s => effects.timeOut(s, TimeoutType.scheduleToClose))
    }
    when(s => states.waiting(s.phase) && s.scheduleToStart == Timeout.expires) {
      deadline.scheduleToStart ~> (s => effects.timeOut(s, TimeoutType.scheduleToStart))
    }
    when(s => s.phase == started && s.startToClose == Timeout.expires) {
      deadline.startToClose ~> (s => effects.timeOut(s, TimeoutType.startToClose))
    }

  object properties extends Section:
    /** A synchronous reply settles the operation as succeeded, and the completed event records it. */
    val syncSucceeds = property when handler.handlerReply(Reply.syncSuccess) holds { s =>
      s.state.phase == Phase.succeeded && s.records(ProtocolFact.nexusOperationCompleted)
    }

    /** An asynchronous reply starts the operation, and the started event records it. */
    val asyncStarts = property when handler.handlerReply(Reply.async) holds { s =>
      s.state.phase == Phase.started && s.records(ProtocolFact.nexusOperationStarted)
    }

    /**
     * A successful completion is recorded by the completed event. Neither the phase nor the outcome
     * is fixed: a completion resolves any running phase, and accepted is every earlier step's
     * outcome too, so a clause fixing it would be answered before the completion.
     */
    val completionSucceeds =
      property when handler.complete(Resolution.succeeded) holds
        (_.records(ProtocolFact.nexusOperationCompleted))

    /** A failed completion is recorded by the failed event. */
    val completionFails = property when handler.complete(Resolution.failed) holds
      (_.records(ProtocolFact.nexusOperationFailed))

    /**
     * A non-retryable handler error settles the operation as failed, and the failed event records
     * it.
     */
    val handlerErrorFails =
      property when handler.handlerReply(Reply.handlerError(false)) holds { s =>
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
    val retrySucceeds = property when handler.handlerReply(Reply.syncSuccess) holds { s =>
      s.state == succeededOnRetry && s.records(ProtocolFact.nexusOperationCompleted)
    }

    /**
     * The schedule-to-start deadline settles an operation no handler started as timed out, and the
     * timed-out event records which deadline it was.
     */
    val scheduleToStartFires = property when deadline.scheduleToStart holds { s =>
      s.state.phase == Phase.timedOut &&
      s.records(ProtocolFact.nexusOperationTimedOut(TimeoutType.scheduleToStart))
    }

    /** The start-to-close deadline settles a started operation no handler completed as timed out. */
    val startToCloseFires = property when deadline.startToClose holds { s =>
      s.state.phase == Phase.timedOut &&
      s.records(ProtocolFact.nexusOperationTimedOut(TimeoutType.startToClose))
    }

  /**
   * The paths the Queries run, then the Queries. Each path is one upstream functional test's shape,
   * from before the operation exists: the schedule command, then the side effects that settle the
   * operation. A schedule that sets no deadline is `schedule()`, each input at `unset`. A path one
   * Query takes is written in it.
   */
  object queries extends Section:
    val asyncThenSucceeded = scenario
      .actions(
        caller.schedule(),
        handler.handlerReply(Reply.async),
        handler.complete(Resolution.succeeded)
      )

    // The design's seven: sync success, async reply then succeeded callback, async reply then failed
    // callback, non-retryable handler error, retryable handler error then sync success after one
    // backoff, schedule-to-start timeout with the handler's worker stopped, start-to-close timeout
    // after an asynchronous reply. Each finds its same-step claim on its path and is realized as a
    // Case. The product claim is verified over every trace of one path, outside `functionalQueries`,
    // because a verify Query realizes nothing.

    val syncCompletion = (query find properties.syncSucceeds in scenario("syncReplied").actions(
      caller.schedule(),
      handler.handlerReply(Reply.syncSuccess)
    ) limits two total 384)
      .expect(satisfied)
      .explore(
        Exploration(
          "nexusDeadlines",
          Vector(
            Variation(
              0,
              Vector(
                Alternative("startDeadline", 30, Vector(caller.schedule(startToClose := expires))),
                Alternative(
                  "scheduleDeadline",
                  20,
                  Vector(caller.schedule(scheduleToStart := expires))
                ),
                Alternative("unbounded", 10, Vector(caller.schedule()))
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
      (query find properties.completionFails in scenario("asyncThenFailed").actions(
        caller.schedule(),
        handler.handlerReply(Reply.async),
        handler.complete(Resolution.failed)
      ) limits three total 576)
        .expect(inconclusive(Reason.explanationsDisagree))
    val handlerError =
      (query find properties.handlerErrorFails in scenario("nonRetryableError").actions(
        caller.schedule(),
        handler.handlerReply(Reply.handlerError(false))
      ) limits two total 384)
        .expect(inconclusive(Reason.neverEvaluated))

    // The retryable error backs the operation off; the backoff timer fires and records nothing; the
    // retried attempt is answered synchronously.
    val retry = (query find properties.retrySucceeds in scenario("retriedThenSucceeded").actions(
      caller.schedule(),
      handler.handlerReply(Reply.handlerError(true)),
      timers.backoff,
      handler.handlerReply(Reply.syncSuccess)
    ) limits four total 768)
      .expect(inconclusive(Reason.explanationsDisagree))

    // The schedule command sets the schedule-to-start deadline; the handler's worker stops, so
    // nothing answers the start request; the deadline fires. The worker stops after the schedule in
    // the operation's order, where the stop changes nothing; the realization stops it before the
    // workflow starts, where the stop cannot race the dispatch.
    val scheduleToStartTimeout =
      (query find properties.scheduleToStartFires in scenario("scheduleToStartExpires").actions(
        caller.schedule(scheduleToStart := expires),
        worker.workerStop,
        deadline.scheduleToStart
      ) limits three total 576)
        .expect(inconclusive(Reason.neverEvaluated))

    // The schedule command sets the start-to-close deadline; the handler accepts asynchronously and
    // never completes; the deadline fires.
    val startToCloseTimeout =
      (query find properties.startToCloseFires in scenario("startToCloseExpires").actions(
        caller.schedule(startToClose := expires),
        handler.handlerReply(Reply.async),
        deadline.startToClose
      ) limits three total 576)
        .expect(inconclusive(Reason.neverEvaluated))

    /**
     * A product claim on a protocol path, read through the refinement the protocol machine declares.
     */
    val terminalHolds =
      query verify NexusProduct.properties.terminalIsFinal in asyncThenSucceeded limits three total
        576

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
//
// The caller's view of the handler's worker: it stops and it serves. It never resumes, because an
// action no sync line names would stay executable on its own and admit a stop, a resume and then a
// reply; the operation's timers settle every state a stop leaves.

object HandlerWorker
    extends Derived(shared.worker.Polling.restrict(worker.workerStop, worker.serve))

// ### The operation and the handler's worker
//
// The protocol machine's worker stop is a stutter row: the operation cannot see its handler's
// worker, so the schedule-to-start Scenario orders the stop before the request by convention.
// Composed with the worker of the handler's task queue, the stop is the worker's own phase change
// and every reply is the worker serving, so a reply has a row only while the worker polls. No
// functional Query reads the composition; it is what the cross-entity claim is verified over.

object NexusCaller
    extends Composition[NexusCallerState](
      _.operation -> NexusProtocol,
      _.worker -> HandlerWorker
    ):
  def end(s: State) = NexusProtocol.states.terminalPhase(s.operation.phase)

  object syncs extends Syncs:
    sync(_.operation -> worker.workerStop, _.worker -> worker.workerStop)
    sync(_.operation -> handler.handlerReply, _.worker -> worker.serve)

  object properties extends Section:
    /**
     * The cross-entity claim: every reply, of any class, leaves the handler's worker polling, so no
     * handler replies while its worker is stopped.
     */
    val repliedByPollingWorker = property
      .whenAction(synced(_.operation -> handler.handlerReply))
      .holds(_.state.worker.phase == WorkerPhase.polling)

  object queries extends Section:
    /**
     * The cross-entity claim, verified over the path on which a retryable reply backs the operation
     * off; the handler's worker then stops, so the retried attempt is never answered and the
     * schedule-to-start deadline fires. The start is stated: a default would take the worker's from
     * shared/worker/Worker.scala.
     */
    val stoppedWorkerRepliesNothing =
      query verify properties.repliedByPollingWorker in scenario("repliedThenStopped")
        .starts(NexusCallerState(NexusProtocol.init, WorkerState(WorkerPhase.polling)))
        .actions(
          own(_.operation, caller.schedule(scheduleToStart := expires)),
          synced(_.operation -> handler.handlerReply(Reply.handlerError(true))),
          synced(_.operation -> worker.workerStop),
          own(_.operation, deadline.scheduleToStart)
        ) limits four total 1536

// ### The control
//
// A caller design that predicts success for a failed completion, which the forged-completion Query
// must refuse. It keeps its own family, and its Definition IDs hang off this object's pin. It is the
// protocol machine without its refinement, its completion forged and an inspection added, declared
// rather than derived from NexusProtocol: a derivation lifts its source machines, and
// nexus-control.json holds this machine alone.

object ForgedCompletion
    extends Machine[ProtocolState, Outcome, ProtocolFact](using
      ControlFamily.family,
      summon,
      summon,
      summon
    ),
      NegativeControl:
  // Moved from temporal.nexuscaller; the pin keeps its Definition IDs.
  given DefinitionScope = DefinitionScope("temporal.nexuscaller.Control$")

  val entity = operation
  val init = NexusProtocol.init
  def end(s: State) = NexusProtocol.states.terminalPhase(s.phase)
  val evidence: PartialFunction[ProtocolFact, String] = {
    case ProtocolFact.nexusOperationTimedOut(_) => "nexusOperationTimedOut"
    case ProtocolFact.pendingAttempts           => pendingAttempts.name
  }
  val unobservable = List(timers.backoff)

  /**
   * The caller's inspection of its workflow, which only this control takes. The section keeps the
   * ID this object's pin gives it; inside the object its name shadows the feature's caller, whose
   * schedule is written in full.
   */
  object caller extends Section:
    val inspect = action(features.nexuscaller.caller).on(operation)

  object effects extends Section:
    def inspect(s: State): List[ProtocolStep] = stay(s)

    /**
     * The protocol's completion of an operation once scheduled: not found once it is over, and
     * resolved while it runs. One effect rather than two rules, because the forged completion names
     * both alternatives of a failed callback in every such phase, the not-found ones included.
     */
    def settle(s: State, resolution: Resolution) =
      if NexusProtocol.states.terminalPhase(s.phase) then NexusProtocol.effects.notFound(s)
      else NexusProtocol.effects.complete(s, resolution)

    // The control deliberately predicts success for a failed callback. The runtime still sends
    // failure.
    def forgedComplete(s: State, resolution: Resolution) =
      if resolution == Resolution.failed then
        choose(forged -> settle(s, Resolution.succeeded), sent -> settle(s, resolution))
      else settle(s, resolution)

  // The protocol machine's rules, its completion forged and the inspection added.
  object rules extends Rules(_.phase):
    in(Phase.unscheduled)(features.nexuscaller.caller.schedule ~> NexusProtocol.effects.schedule)
    in(Phase.scheduled)(handler.handlerReply ~> NexusProtocol.effects.handlerReply)
    when(s => NexusProtocol.states.created(s.phase))(handler.complete ~> effects.forgedComplete)
    when(_ => true)(caller.inspect ~> effects.inspect)
    in(Phase.scheduled)(network.transportFault ~> NexusProtocol.effects.backOff)
    when(_ => true)(worker.workerStop ~> NexusProtocol.effects.keep)
    in(Phase.backingOff)(timers.backoff ~> NexusProtocol.effects.retry)
    when(s => NexusProtocol.states.running(s.phase) && s.scheduleToClose == Timeout.expires) {
      deadline.scheduleToClose ~> (s =>
        NexusProtocol.effects.timeOut(s, TimeoutType.scheduleToClose)
      )
    }
    when(s => NexusProtocol.states.waiting(s.phase) && s.scheduleToStart == Timeout.expires) {
      deadline.scheduleToStart ~> (s =>
        NexusProtocol.effects.timeOut(s, TimeoutType.scheduleToStart)
      )
    }
    when(s => s.phase == Phase.started && s.startToClose == Timeout.expires) {
      deadline.startToClose ~> (s => NexusProtocol.effects.timeOut(s, TimeoutType.startToClose))
    }

  object properties extends Section:
    /**
     * A failed completion is recorded as completed: what the control predicts and no runtime sends.
     */
    val forgedSuccess = property when handler.complete(Resolution.failed) holds
      (_.records(ProtocolFact.nexusOperationCompleted))

  object queries extends Section:
    /**
     * The forged control, which every modeled execution that explains the evidence refutes, on a
     * path that inspects the operation around a failed completion.
     */
    val forgedCompletion =
      (query find properties.forgedSuccess in scenario("inspectedFailure").actions(
        features.nexuscaller.caller.schedule(),
        caller.inspect,
        handler.handlerReply(Reply.async),
        caller.inspect,
        handler.complete(Resolution.failed)
      ) limits control total 960)
        .expect(
          RunExpectation(
            Conformance.inconclusive,
            PropertyOutcome.violated,
            contract = PropertyOutcome.violated,
            disposition = Disposition.stoppedByMonitor,
            cleanup = Cleanup.succeeded,
            reason = Some(Reason.everyExplanationViolates),
            conformanceReason = Some(Reason.incomplete)
          )
        )
        .explore(
          Exploration(
            "nexusControl",
            Vector(
              Variation(
                1,
                Vector(
                  Alternative("twice", 20, Vector(caller.inspect, caller.inspect)),
                  Alternative("once", 10, Vector(caller.inspect)),
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

object exports:
  // The functional Queries and the realization that runs them are roots beside the machines: Go
  // lowers each Query's witness through the realization into a Testpilot Case (tools/umpire/lower).
  // The product claim read on a protocol path and the cross-entity Query, which carries the
  // composition with the handler's worker and its claim, are roots too.
  val nexusCaller = irFile("nexus-caller")(
    NexusProduct,
    NexusProtocol,
    HandlerWorker,
    NexusCaller,
    shared.worker.Polling,
    NexusProtocol.queries.functionalQueries,
    NexusProtocol.queries.terminalHolds,
    NexusCaller.queries.stoppedWorkerRepliesNothing,
    NexusRealization.asyncNexus
  )

  // The forged completion a caller must refuse, and the realization that offers it.
  val nexusControl =
    irFile("nexus-control")(
      ForgedCompletion.queries.forgedCompletion,
      NexusRealization.forgedCompletion
    )
