/* The standalone activity Model: one activity started directly through StartActivityExecution, with
 * no workflow around it, grounded in chasm/lib/activity/statemachine.go. The product machine says
 * what DescribeActivityExecution reports, the protocol how the server gets there. No history
 * event is written, so every evidence line names an observation: a status read through
 * DescribeActivityExecution or a result read through PollActivityExecution. Reset is deferred, like
 * cancellation in the Nexus caller Model, and the heartbeat timeout is not modeled.
 *
 * Read top to bottom: the types; the signature (the activity and its inputs; the caller and its
 * actions, the worker's actions on the activity, its timers and deadlines; and the bounds); then
 * one object per machine, each before the machines that use it -- Product, the product machine;
 * Protocol, the protocol machine that refines it; ActivityWorker, the worker of its task queue;
 * StandaloneActivity, the protocol with that worker -- and last Files, its IR files.
 * Realization.scala realizes it; record/ holds the system contract and withTaskQueue/ the contract
 * composed with the shared task queue.
 */
package temporal
package features.standaloneactivity

import scala.annotation.unused
import umpire.*
import umpire.realize.Reason
import temporal.capabilities.{given, *}
import temporal.realize.{inconclusive, satisfied}
import shared.Bounds.{four, three}
import shared.worker.{worker as process, Phase as WorkerPhase, State as WorkerState}
import io.temporal.api.workflowservice.v1.*
import ActivityFamily.given
import Timeout.expires

// Moved from temporal.standaloneactivity; the pin keeps its Definition IDs and type names.
given DefinitionScope = DefinitionScope("temporal.standaloneactivity.Model$package$")

/** The family of the activity's machines; the system contract takes `SystemFamily`. */
object ActivityFamily:
  given family: Family = Family("temporal.activity.standalone")

/** The system contract's family; the shared task queue, first written there, keeps it too. */
object SystemFamily:
  given family: Family = Family("temporal.activity.standalone.system")

// ### Types

/** Whether the start request sets a deadline. */
enum Timeout derives Finite:
  case unset, expires

/** The worker's answer; failed(retryable) is two classes, like ApplicationFailure's flag. */
enum AttemptResult derives Finite:
  case completed
  case failed(retryable: Boolean)
  case canceled

enum Control derives Finite:
  case pause, unpause, requestCancel, terminate

/** A step's outcome, shared by both machines by name. */
enum Outcome derives Finite:
  case accepted, notFound

/** The product machine's phases: what DescribeActivityExecution shows. */
enum ProductPhase derives Finite:
  case scheduled, started, paused, cancelRequested
  case completed, failed, canceled, terminated, timedOut

final case class ProductState(phase: ProductPhase) derives Finite

enum ProductFact derives Finite:
  case statusScheduled, statusStarted, statusPaused, statusCancelRequested
  case statusCompleted, statusFailed, statusCanceled, statusTerminated, statusTimedOut

type ProductStep = Step[ProductState, Outcome, ProductFact]

/** The protocol machine's phases. It begins before the activity exists, so unstarted is one. */
enum Phase derives Finite:
  case unstarted, scheduled, backingOff, started, paused, pauseRequested, cancelRequested
  case completed, failed, canceled, terminated, timedOut

/** Which deadline fired. */
enum TimeoutType derives Finite:
  case scheduleToClose, scheduleToStart, startToClose

/** 12 phases, 3 attempt counts (`0..Protocol.attemptBound`) and 3 deadline flags: 288 states. */
final case class ProtocolState(
    phase: Phase,
    attempts: UpTo[2],
    scheduleToClose: Timeout,
    scheduleToStart: Timeout,
    startToClose: Timeout
) derives Finite

/** What the protocol machine records; `attemptCount` is named after its observation. */
enum ProtocolFact derives Finite:
  case statusScheduled, statusStarted, statusPaused, statusCancelRequested
  case statusCompleted, statusFailed, statusCanceled, statusTerminated
  case statusTimedOut(timeoutType: TimeoutType)
  case attemptCount

type ProtocolStep = Step[ProtocolState, Outcome, ProtocolFact]

final case class StandaloneActivityState(activity: ProtocolState, worker: WorkerState)

// ### Signature

/** Named by the id the caller chose: every read carries it, so no run id or event id is needed. */
val activity = Entity(key = "activityId")

// The start's inputs, which the deadline timers no longer collide with, and the worker's answer.
val scheduleToClose = input[Timeout]
val scheduleToStart = input[Timeout]
val startToClose = input[Timeout]
val result = input[AttemptResult]

/** The control's input, apart because inside `caller` its name is the control action. */
object Inputs:
  val control = input[Control]

// Who acts, and on what. Actor and section objects are transparent to Definition IDs, so every
// action keeps the ID the file's pin gives it.

/** The caller starts and controls the activity. */
object caller extends Actor:
  val start = action(this)
    .input(scheduleToClose)
    .input(scheduleToStart)
    .input(startToClose)
    .creates(activity)
    .schema[StartActivityExecutionRequest]

  // The four controls are one action because they share a result: on an activity that is over, a
  // control is not found. The result text is metadata of the action, not a domain a state holds.
  val control = action(this)
    .on(activity)
    .input(Inputs.control)
    .schema[PauseActivityExecutionRequest]
    .schema[UnpauseActivityExecutionRequest]
    .schema[RequestCancelActivityExecutionRequest]
    .schema[TerminateActivityExecutionRequest]
    .results("Delivery")

/**
 * The shared worker party's actions on this activity: its poll receives the task for the current
 * attempt, and its answer settles it. The worker's stop is the party's own action,
 * `process.workerStop`: nothing it records names the activity, so the activity's machines keep
 * their state.
 */
object worker extends Section:
  val attemptStart = action(process).on(activity).schema[PollActivityTaskQueueResponse]

  val attemptResult = action(process)
    .on(activity)
    .input(result)
    .schema[RespondActivityTaskCompletedRequest]
    .schema[RespondActivityTaskFailedRequest]
    .schema[RespondActivityTaskCanceledRequest]
    .example(AttemptResult.failed(false), "ApplicationFailureNonRetryable")
    .example(AttemptResult.failed(true), "ApplicationFailureRetryable")

/** One of the activity's deadlines firing, as the product machine sees it, and the backoff. */
object timers extends Section:
  val timeout = timer
  val backoff = timer

/** The protocol's three deadlines, each armed by the start's input of its name. */
object deadline extends Section:
  val scheduleToClose = timer
  val scheduleToStart = timer
  val startToClose = timer

// A retry shows the caller only the attempt count DescribeActivityExecution reports. The statuses
// observe one status field; whether a catalog tells them apart is left to the realization.
val attemptCount = Observation(on = activity, read = "attempt")

given Ok[Outcome] = Ok(Outcome.accepted)

// The bounds of these Queries and the system contract's, beside three and four (shared.Bounds).
val five = Limits(steps = 5, actions = 5, search = 65536)
val six = Limits(steps = 6, actions = 6, search = 262144)
val eight = Limits(steps = 8, actions = 8, search = 262144)

// ### The product machine: what DescribeActivityExecution shows, with no account of how. A retry
// reads as scheduled again, a pause of a running attempt as started until the worker yields.

object Product:
  import ProductPhase.*

  def phase(s: ProductState) = s.phase

  def terminal(p: ProductPhase) = p.in(completed, failed, canceled, terminated, timedOut)

  /** A path ends where the activity is over. */
  def end(s: ProductState) = terminal(s.phase)

  /** A paused activity, which no worker is given. */
  def paused(s: ProductState) = s.phase == ProductPhase.paused

  def running(s: ProductState) = s.phase == started

  /** Where a worker holds the attempt, so its answer settles the activity. */
  def held(s: ProductState) = s.phase.in(started, cancelRequested)

  /** Where a pause takes effect: before an attempt starts or while one runs. */
  def pausable(s: ProductState) = s.phase.in(scheduled, started)

  /**
   * The server code that answers a control of an activity that is over NotFound (activity.go:106).
   * Only the lifter reads a citation.
   */
  val notFoundCode = "chasm/lib/activity/activity.go"

  object effects:
    import ProductFact.*

    def attemptStart(s: ProductState) =
      if s.phase != scheduled then disabled else enter(ProductState(started), statusStarted)

    /**
     * Unlike the Nexus caller, a retryable failure reads SCHEDULED again with a higher attempt count
     * (TransitionRescheduled), or canceled under a cancel request; the protocol adds the backoff. A
     * canceled answer settles only an activity whose cancellation was requested.
     */
    def attemptResult(s: ProductState, result: AttemptResult) =
      if !held(s) then disabled
      else
        result match
          case AttemptResult.completed         => enter(ProductState(completed), statusCompleted)
          case AttemptResult.failed(retryable) =>
            if !retryable then enter(ProductState(failed), statusFailed)
            else if s.phase == cancelRequested then enter(ProductState(canceled), statusCanceled)
            else enter(ProductState(scheduled), statusScheduled)
          case AttemptResult.canceled =>
            if s.phase == cancelRequested then enter(ProductState(canceled), statusCanceled)
            else disabled

    /**
     * A control on an activity that is over is not found. A pause of a paused or cancel-requested
     * activity, or an unpause of one not paused, is FailedPrecondition; the protocol lists them.
     */
    def control(s: ProductState, c: Control) =
      if terminal(s.phase) then List(Step(Outcome.notFound, s))
      else
        c match
          case Control.pause =>
            if pausable(s) then enter(ProductState(ProductPhase.paused), statusPaused) else disabled
          case Control.unpause =>
            if paused(s) then enter(ProductState(scheduled), statusScheduled) else disabled
          case Control.requestCancel => enter(ProductState(cancelRequested), statusCancelRequested)
          case Control.terminate     => enter(ProductState(terminated), statusTerminated)

    /** The worker stopping is a fault the Run records and the activity does not feel. */
    def workerStop(@unused s: ProductState): List[ProductStep] = disabled

    /** One of the activity's deadlines firing. Which deadline is the protocol's account of how. */
    def timeout(s: ProductState) =
      if terminal(s.phase) then disabled else enter(ProductState(timedOut), statusTimedOut)

  /** Every status the product machine records is confirmed by the status observation of its name. */
  val activityProduct = machine[ProductState, Outcome, ProductFact] {
    forEntity(activity)
    starts(ProductState(ProductPhase.scheduled))
    ends(end)
    steps(
      worker.attemptStart ~> effects.attemptStart,
      worker.attemptResult ~> effects.attemptResult,
      caller.control ~> effects.control,
      process.workerStop ~> effects.workerStop,
      timers.timeout ~> effects.timeout
    )
  }

  /**
   * What the product machine is, as the laws of model/temporal/capabilities read it, and so the laws
   * it receives without listing them: it closes, and a control of an activity that is over is not
   * found, by the code `notFoundCode` cites; it pauses; and a worker's poll hands out its work. It
   * receives terminalStatesAreFinal, closedIsRejectedUniformly and pausedIsNotDispatched (pause with
   * poll), each `activityProduct.<law>`, read on the protocol through the map under the bound held
   * there.
   */
  object laws:
    val productCapabilities = capabilities(activityProduct, limits = three)(
      Closable(
        status = phase,
        terminal = terminal,
        rejected = cited(Outcome.notFound, notFoundCode)
      ),
      Pausable(
        pause = caller.control(Control.pause),
        unpause = caller.control(Control.unpause),
        paused = Product.paused
      ),
      Pollable(dispatch = worker.attemptStart, running = Product.running)
    )

// ### The protocol machine adds the retry, the pause request, the timers and the attempt count. It
// begins before the activity exists, so unstarted is a phase and the start sets the deadlines.

object Protocol:
  import Phase.*

  /** Bounds the attempt count, as the type of `ProtocolState.attempts` does. */
  val attemptBound = 2

  def terminal(p: Phase) = p.in(completed, failed, canceled, terminated, timedOut)

  /** Started and not over: the phases a deadline can fire in. */
  def live(p: Phase) =
    p.in(scheduled, backingOff, started, paused, pauseRequested, cancelRequested)

  /** Where a worker holds the attempt: what start-to-close covers and a worker's answer settles. */
  def held(p: Phase) = p.in(started, pauseRequested, cancelRequested)

  /** Waiting for a worker: the phases before an attempt is held, which schedule-to-start covers. */
  def waiting(p: Phase) = p.in(scheduled, backingOff)

  def saturatingSucc(a: UpTo[2]): UpTo[2] = UpTo((a + 1).min(attemptBound))

  /**
   * Unstarted and backing off read as scheduled. A pause request reads as started: the worker still
   * holds the attempt, its every answer is a product row from started, and the request stutters.
   */
  def productOf(s: ProtocolState) = s.phase match
    case Phase.unstarted | Phase.scheduled | Phase.backingOff =>
      ProductState(ProductPhase.scheduled)
    case Phase.started | Phase.pauseRequested => ProductState(ProductPhase.started)
    case Phase.paused                         => ProductState(ProductPhase.paused)
    case Phase.cancelRequested                => ProductState(ProductPhase.cancelRequested)
    case Phase.completed                      => ProductState(ProductPhase.completed)
    case Phase.failed                         => ProductState(ProductPhase.failed)
    case Phase.canceled                       => ProductState(ProductPhase.canceled)
    case Phase.terminated                     => ProductState(ProductPhase.terminated)
    case Phase.timedOut                       => ProductState(ProductPhase.timedOut)

  /** Where every path begins: before the activity exists, with every deadline at its first value. */
  val unstarted =
    ProtocolState(Phase.unstarted, UpTo(0), Timeout.unset, Timeout.unset, Timeout.unset)

  object effects:
    import ProtocolFact.*

    def start(
        s: ProtocolState,
        scheduleToClose: Timeout,
        scheduleToStart: Timeout,
        startToClose: Timeout
    ) =
      if s.phase != Phase.unstarted then disabled
      else
        enter(
          ProtocolState(scheduled, UpTo(0), scheduleToClose, scheduleToStart, startToClose),
          statusScheduled
        )

    /** The worker's poll takes the attempt and raises the count the caller reads back. */
    def attemptStart(s: ProtocolState) =
      if s.phase != scheduled then disabled
      else
        enter(
          s.copy(phase = started, attempts = saturatingSucc(s.attempts)),
          statusStarted,
          ProtocolFact.attemptCount
        )

    /**
     * A retryable failure backs a started attempt off (read as scheduled again, one attempt higher),
     * settles a cancel-requested one as canceled and lands a pause-requested one in paused
     * (TransitionAttemptFailedWhilePauseRequested). A canceled answer needs a cancel request.
     */
    def attemptResult(s: ProtocolState, result: AttemptResult) =
      if !held(s.phase) then disabled
      else
        result match
          case AttemptResult.completed         => enter(s.copy(phase = completed), statusCompleted)
          case AttemptResult.failed(retryable) =>
            if !retryable then enter(s.copy(phase = failed), statusFailed)
            else if s.phase == cancelRequested then enter(s.copy(phase = canceled), statusCanceled)
            else if s.phase == pauseRequested then enter(s.copy(phase = paused), statusPaused)
            else
              enter(s.copy(phase = backingOff), statusScheduled, ProtocolFact.attemptCount)
                .because("a retryable failure backs off; the caller reads scheduled again")
          case AttemptResult.canceled =>
            if s.phase == cancelRequested then enter(s.copy(phase = canceled), statusCanceled)
            else disabled

    /**
     * A pause of a held attempt is a request the worker learns of on its next heartbeat, so it is its
     * own phase the caller reads as paused. Each phase a pause or an unpause is disabled in has its
     * arm: the server answers FailedPrecondition ("activity is in non-pausable state", "...
     * non-unpausable state", chasm/lib/activity/operator_commands.go), and a rejecting row would add
     * rows to the table, so they stay disabled until the behavior freeze lifts.
     */
    def control(s: ProtocolState, c: Control) =
      if terminal(s.phase) then List(Step(Outcome.notFound, s))
      else if s.phase == Phase.unstarted then disabled // no activity yet, so nothing to control
      else
        c match
          case Control.pause =>
            s.phase match
              case Phase.scheduled | Phase.backingOff => enter(s.copy(phase = paused), statusPaused)
              case Phase.started                      =>
                enter(s.copy(phase = pauseRequested), statusPaused)
                  .because("the worker learns of the pause on its next heartbeat")
              case Phase.paused | Phase.pauseRequested => disabled // already paused, or asked to be
              case Phase.cancelRequested => disabled // a cancel request is not pausable
              // Answered above: an activity over is not found, an unstarted one has no control.
              case Phase.unstarted | Phase.completed | Phase.failed | Phase.canceled |
                  Phase.terminated | Phase.timedOut =>
                disabled
          case Control.unpause =>
            s.phase match
              case Phase.paused         => enter(s.copy(phase = scheduled), statusScheduled)
              case Phase.pauseRequested => enter(s.copy(phase = started), statusStarted)
              case Phase.scheduled | Phase.backingOff | Phase.started => disabled // not paused
              case Phase.cancelRequested                              => disabled // not paused
              // Answered above: an activity over is not found, an unstarted one has no control.
              case Phase.unstarted | Phase.completed | Phase.failed | Phase.canceled |
                  Phase.terminated | Phase.timedOut =>
                disabled
          case Control.requestCancel =>
            enter(s.copy(phase = cancelRequested), statusCancelRequested)
          case Control.terminate => enter(s.copy(phase = terminated), statusTerminated)

    /** Keeps the state and records nothing; the next step's evidence confirms it, a Known Gap. */
    def workerStop(s: ProtocolState): List[ProtocolStep] = stay(s)

    /** The backoff timer. A retry writes nothing the caller can read. */
    def backoff(s: ProtocolState): List[ProtocolStep] =
      if s.phase != backingOff then disabled else enter(s.copy(phase = scheduled))

    def scheduleToClose(s: ProtocolState) =
      if live(s.phase) && s.scheduleToClose == Timeout.expires then
        enter(s.copy(phase = timedOut), statusTimedOut(TimeoutType.scheduleToClose))
      else disabled

    def scheduleToStart(s: ProtocolState) =
      if waiting(s.phase) && s.scheduleToStart == Timeout.expires then
        enter(s.copy(phase = timedOut), statusTimedOut(TimeoutType.scheduleToStart))
      else disabled

    def startToClose(s: ProtocolState) =
      if held(s.phase) && s.startToClose == Timeout.expires then
        enter(s.copy(phase = timedOut), statusTimedOut(TimeoutType.startToClose))
      else disabled

  /** A timeout is confirmed by the one status observation, whichever deadline fired. */
  val activityProtocol = machine[ProtocolState, Outcome, ProtocolFact] {
    forEntity(activity)
    refines(Product.activityProduct)(productOf)
    starts(unstarted)
    ends(s => terminal(s.phase))
    unobservable(timers.backoff)
    evidence {
      case ProtocolFact.statusTimedOut(_) => "statusTimedOut"
      case ProtocolFact.attemptCount      => attemptCount.name
    }
    steps(
      caller.start ~> effects.start,
      worker.attemptStart ~> effects.attemptStart,
      worker.attemptResult ~> effects.attemptResult,
      caller.control ~> effects.control,
      process.workerStop ~> effects.workerStop,
      timers.backoff ~> effects.backoff,
      deadline.scheduleToClose ~> effects.scheduleToClose,
      deadline.scheduleToStart ~> effects.scheduleToStart,
      deadline.startToClose ~> effects.startToClose
    )
  }

  /**
   * What the protocol promises of its own: the settlement claims. The cross-entity claim of the
   * activity and its worker is the composition's, and the system contract's are record/'s.
   */
  object properties:
    val completes =
      activityProtocol.property when worker.attemptResult(AttemptResult.completed) holds { s =>
        s.state.phase == Phase.completed && s.records(ProtocolFact.statusCompleted)
      }

    val nonRetryableFails =
      activityProtocol.property when worker.attemptResult(AttemptResult.failed(false)) holds { s =>
        s.state.phase == Phase.failed && s.records(ProtocolFact.statusFailed)
      }

    /** Completed on the second attempt of an activity with no deadline set. */
    val completedOnRetry =
      ProtocolState(
        Phase.completed,
        UpTo(attemptBound),
        Timeout.unset,
        Timeout.unset,
        Timeout.unset
      )

    /**
     * The attempt count saturates at `attemptBound`, so the claim is bounded by it: a completion on
     * any later attempt than the second reads as this one.
     */
    val retryCompletes =
      activityProtocol.property when worker.attemptResult(AttemptResult.completed) holds { s =>
        s.state == completedOnRetry && s.records(ProtocolFact.statusCompleted)
      }

    val cancelRequestedWhileStarted =
      activityProtocol.property when caller.control(Control.requestCancel) holds { s =>
        s.state.phase == Phase.cancelRequested && s.records(ProtocolFact.statusCancelRequested)
      }

    val canceledByWorker =
      activityProtocol.property when worker.attemptResult(AttemptResult.canceled) holds { s =>
        s.state.phase == Phase.canceled && s.records(ProtocolFact.statusCanceled)
      }

    val terminated = activityProtocol.property when caller.control(Control.terminate) holds { s =>
      s.state.phase == Phase.terminated && s.records(ProtocolFact.statusTerminated)
    }

    // Each deadline times the activity out and the status records which it was. With both
    // schedule-to-start and schedule-to-close set and no attempt started, either may fire first.
    val scheduleToStartFires = activityProtocol.property when deadline.scheduleToStart holds { s =>
      s.state.phase == Phase.timedOut &&
      s.records(ProtocolFact.statusTimedOut(TimeoutType.scheduleToStart))
    }

    val scheduleToCloseFires = activityProtocol.property when deadline.scheduleToClose holds { s =>
      s.state.phase == Phase.timedOut &&
      s.records(ProtocolFact.statusTimedOut(TimeoutType.scheduleToClose))
    }

    val startToCloseFires = activityProtocol.property when deadline.startToClose holds { s =>
      s.state.phase == Phase.timedOut &&
      s.records(ProtocolFact.statusTimedOut(TimeoutType.startToClose))
    }

  /**
   * What the protocol machine is, as the functional laws read it, which a find through its
   * realization asks: a terminate settles it, a cancel request is recorded, and
   * DescribeActivityExecution reports its status by `activityStatus`. Each law's find starts the
   * activity and stops the worker before the control, so no attempt is in flight when it lands, as
   * `terminatedWhileScheduled` does. A Run explains an unobserved control of an activity that is over
   * too, which answers notFound and records nothing, so the claim's explanations disagree. It reads
   * the realization, which reads this machine, so it waits in an object of its own until it is used.
   */
  object laws:
    val protocolCapabilities = capabilities(activityProtocol, limits = three)(
      Terminable(
        terminate = caller.control(Control.terminate),
        settled = ProtocolFact.statusTerminated,
        reach = Seq(caller.start(), process.workerStop),
        expect = inconclusive(Reason.explanationsDisagree)
      ),
      Cancelable(
        requestCancel = caller.control(Control.requestCancel),
        requested = ProtocolFact.statusCancelRequested,
        reach = Seq(caller.start(), process.workerStop),
        expect = inconclusive(Reason.explanationsDisagree)
      ),
      Describable(status = ActivityRealization.activityStatus)
    )

  /**
   * The paths, then one functional Query per side effect that settles the activity, and the Queries
   * that carry the other promises into the lifted Model. Each path starts before the activity exists;
   * a start that sets no deadline is `start()`, each input at `unset`.
   */
  object queries:
    val completed = activityProtocol.scenario
      .actions(caller.start(), worker.attemptStart, worker.attemptResult(AttemptResult.completed))

    val nonRetryable = activityProtocol.scenario
      .actions(
        caller.start(),
        worker.attemptStart,
        worker.attemptResult(AttemptResult.failed(false))
      )

    val retriedThenCompleted = activityProtocol.scenario.actions(
      caller.start(),
      worker.attemptStart,
      worker.attemptResult(AttemptResult.failed(true)),
      timers.backoff,
      worker.attemptStart,
      worker.attemptResult(AttemptResult.completed)
    )

    val cancelRequestedThenCanceled = activityProtocol.scenario.actions(
      caller.start(),
      worker.attemptStart,
      caller.control(Control.requestCancel),
      worker.attemptResult(AttemptResult.canceled)
    )

    /** The worker stops before the start, so no attempt is in flight when the caller terminates. */
    val terminatedWhileScheduled = activityProtocol.scenario
      .actions(caller.start(), process.workerStop, caller.control(Control.terminate))

    val pausedThenCompleted = activityProtocol.scenario.actions(
      caller.start(),
      caller.control(Control.pause),
      caller.control(Control.unpause),
      worker.attemptStart,
      worker.attemptResult(AttemptResult.completed)
    )

    val scheduleToStartExpires = activityProtocol.scenario
      .actions(
        caller.start(scheduleToStart := expires),
        process.workerStop,
        deadline.scheduleToStart
      )

    val startToCloseExpires = activityProtocol.scenario
      .actions(caller.start(startToClose := expires), worker.attemptStart, deadline.startToClose)

    /** Both deadlines set and no attempt started, so either may fire first. */
    val bothDeadlinesStartFirst = activityProtocol.scenario.actions(
      caller.start(scheduleToClose := expires, scheduleToStart := expires),
      deadline.scheduleToStart
    )

    val bothDeadlinesCloseFirst = activityProtocol.scenario.actions(
      caller.start(scheduleToClose := expires, scheduleToStart := expires),
      deadline.scheduleToClose
    )

    // One Query per side effect that settles the activity, apart from the Properties they find.
    val completion =
      (query find properties.completes in completed limits three total 864).expect(satisfied)
    val nonRetryableFailure =
      (query find properties.nonRetryableFails in nonRetryable limits three total 864)
        .expect(satisfied)
    val retry =
      (query find properties.retryCompletes in retriedThenCompleted limits six total 1728)
        .expect(inconclusive(Reason.explanationsDisagree))
    val cancel =
      query find properties.canceledByWorker in cancelRequestedThenCanceled limits four total 1152
    val terminate =
      (query find properties.terminated in terminatedWhileScheduled limits three total 864)
        .expect(inconclusive(Reason.explanationsDisagree))
    val pauseResume =
      (query find properties.completes in pausedThenCompleted limits six total 1440)
        .expect(satisfied)
    val scheduleToStartTimeout =
      (query find properties.scheduleToStartFires in scheduleToStartExpires limits three total 864)
        .expect(inconclusive(Reason.neverEvaluated))
    val startToCloseTimeout =
      query find properties.startToCloseFires in startToCloseExpires limits three total 864

    val all = Vector(
      completion,
      nonRetryableFailure,
      retry,
      cancel,
      terminate,
      pauseResume,
      scheduleToStartTimeout,
      startToCloseTimeout
    )

    // The Queries that carry the other promises into the lifted Model, where Go's are compared.

    /** Asks the one Property no functional Query asks, over the path that takes a cancel request. */
    val cancelRequest =
      query find properties.cancelRequestedWhileStarted in cancelRequestedThenCanceled limits
        four total 1152

    /** Neither deadline is ordered before the other: each firing is a trace of its own. */
    val competingTimers = Vector(
      query("competingTimers.scheduleToStartFirst") find properties.scheduleToStartFires in
        bothDeadlinesStartFirst limits three total 576,
      query("competingTimers.scheduleToCloseFirst") find properties.scheduleToCloseFires in
        bothDeadlinesCloseFirst limits three total 576
    )

// ### The worker of the activity's task queue, as the activity sees it: its stop and its serving.

object ActivityWorker:
  val activityWorker = shared.worker.Polling.polling.restrict(process.workerStop, process.serve)

// ### With the worker of its task queue, the stop is the worker's own phase change and every
// attempt start is the worker serving, so an attempt has a row only while the worker polls.

object StandaloneActivity:
  val standaloneActivity =
    compose[StandaloneActivityState](
      _.activity -> Protocol.activityProtocol,
      _.worker -> ActivityWorker.activityWorker
    )
      .sync(_.activity -> process.workerStop, _.worker -> process.workerStop)
      .sync(_.activity -> worker.attemptStart, _.worker -> process.serve)
      .ends(s => Protocol.terminal(s.activity.phase))

  object properties:
    /** The cross-entity claim: no stopped worker starts an attempt. */
    val startedByPollingWorker = standaloneActivity.property
      .whenAction(standaloneActivity.synced(_.activity -> worker.attemptStart))
      .holds(_.state.worker.phase == WorkerPhase.polling)

  object queries:
    /**
     * The first attempt fails retryably and backs off, then the worker stops, so the retry is never
     * dispatched; the attempt start makes the verification exercise the claim. The start is stated:
     * a default would take the worker's from shared/worker/Worker.scala; fn-115's golden compares positions.
     */
    val stoppedBeforeRetry = standaloneActivity.scenario
      .starts(StandaloneActivityState(Protocol.unstarted, WorkerState(WorkerPhase.polling)))
      .actions(
        standaloneActivity.own(_.activity, caller.start(scheduleToStart := expires)),
        standaloneActivity.synced(_.activity -> worker.attemptStart),
        standaloneActivity.own(_.activity, worker.attemptResult(AttemptResult.failed(true))),
        standaloneActivity.own(_.activity, timers.backoff),
        standaloneActivity.synced(_.activity -> process.workerStop),
        standaloneActivity.own(_.activity, deadline.scheduleToStart)
      )

    /** The cross-entity claim, over the path on which a stopped worker never takes the retry. */
    val stoppedWorkerStartsNothing =
      query verify properties.startedByPollingWorker in stoppedBeforeRetry limits six total 3456

// ### The checked-in IR files of the standalone activity Models (umpire.irFile).

object Files:
  // The activity Model. Its cross-entity Query, stoppedWorkerStartsNothing, carries the composition
  // and its claim.
  val activityFile = irFile("activity")(
    StandaloneActivity.standaloneActivity,
    Product.activityProduct,
    Protocol.queries.all,
    Product.laws.productCapabilities,
    Protocol.laws.protocolCapabilities,
    Protocol.queries.cancelRequest,
    StandaloneActivity.queries.stoppedWorkerStartsNothing,
    ActivityRealization.standalone
  )

  // Its system contract, the admission designs, and the shared task queue's providers it composes.
  // A composition no Query runs over is a root of its own.
  val activitySystemFile = irFile("activity-system")(
    record.Admission.queries.currentQueries,
    record.StaleAdmission.queries.staleQueries,
    Protocol.queries.competingTimers,
    shared.taskqueue.MatchingQueue.queries.matchingQueueQueries,
    shared.taskqueue.MatchingQueue.queries.forgetfulQueueQueries,
    shared.taskqueue.MatchingQueue.queries.volatileQueueQueries,
    shared.taskqueue.MatchingQueue.queries.lossyMatchingQueueQueries,
    shared.taskqueue.MatchingQueue.queries.storageLossQuery,
    withTaskQueue.CurrentOverQueue.queries.currentOverQueueQueries,
    withTaskQueue.StaleOverQueue.queries.staleOverQueueQueries,
    withTaskQueue.CurrentOverMatching.queries.currentOverMatchingQueries,
    withTaskQueue.StaleOverMatching.queries.staleOverMatchingQueries,
    withTaskQueue.CurrentOverLossyMatching.queries.currentOverLossyMatchingQueries,
    withTaskQueue.CurrentOverForgetful.currentOverForgetful,
    withTaskQueue.CurrentOverVolatile.currentOverVolatile
  )

  // The held race a server is run through, and the realization that runs it. It is a Model of its
  // own, so the system contract's Queries are the ones its checkers were given.
  val activityRaceFile = irFile("activity-race")(
    record.HeldAdmission.queries.heldStaleDelivery,
    ActivityRealization.heldDelivery,
    record.ResponseLoss.queries.lostAdmissionResponseQuery,
    ActivityRealization.lostAdmissionResponse
  )
