/* The standalone activity Model: one activity started directly through StartActivityExecution, with
 * no workflow around it, grounded in chasm/lib/activity/statemachine.go. The product machine says
 * what DescribeActivityExecution reports, the protocol how the server gets there. No history
 * event is written, so every evidence line names an observation: a status read through
 * DescribeActivityExecution or a result read through PollActivityExecution. Reset is deferred, like
 * cancellation in the Nexus caller Model, and the heartbeat timeout is not modeled.
 */
package temporal
package standaloneactivity

import scala.annotation.unused
import umpire.*
import worker.{serve, workerStop, State as WorkerState}
import io.temporal.api.workflowservice.v1.*
import ActivityFamily.given

/** The family of the activity's machines; the system contract takes `SystemFamily`. */
object ActivityFamily:
  given family: Family = Family("temporal.activity.standalone")

/** The system contract's family; the shared task queue, first written there, keeps it too. */
object SystemFamily:
  given family: Family = Family("temporal.activity.standalone.system")

// The caller starts and controls the activity. The worker's stop is the worker's own action,
// `workerStop`: nothing it records names the activity, so the activity's machines keep their state.
val caller: Party = Party()

/** Named by the id the caller chose: every read carries it, so no run id or event id is needed. */
val activity: Entity = Entity(key = "activityId")

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

/** The inputs, apart because the deadline timers and the control action take their names. */
object Inputs:
  val scheduleToClose = input[Timeout]
  val scheduleToStart = input[Timeout]
  val startToClose = input[Timeout]
  val result = input[AttemptResult]
  val control = input[Control]

val start = action(caller)
  .input(Inputs.scheduleToClose)
  .input(Inputs.scheduleToStart)
  .input(Inputs.startToClose)
  .creates(activity)
  .schema[StartActivityExecutionRequest]

/** The worker's poll receives the task for the current attempt. */
val attemptStart = action(worker.party).on(activity).schema[PollActivityTaskQueueResponse]

val attemptResult = action(worker.party)
  .on(activity)
  .input(Inputs.result)
  .schema[RespondActivityTaskCompletedRequest]
  .schema[RespondActivityTaskFailedRequest]
  .schema[RespondActivityTaskCanceledRequest]
  .example(AttemptResult.failed(false), "ApplicationFailureNonRetryable")
  .example(AttemptResult.failed(true), "ApplicationFailureRetryable")

// The four controls are one action because they share a result: on an activity that is over, a
// control is not found. The result text is metadata of the action, not a domain a state holds.
val control = action(caller)
  .on(activity)
  .input(Inputs.control)
  .schema[PauseActivityExecutionRequest]
  .schema[UnpauseActivityExecutionRequest]
  .schema[RequestCancelActivityExecutionRequest]
  .schema[TerminateActivityExecutionRequest]
  .results("Delivery")

// A retry shows the caller only the attempt count DescribeActivityExecution reports. The statuses
// observe one status field; whether a catalog tells them apart is left to the realization.
val attemptCount: Observation = Observation(on = activity, read = "attempt")

/** A step's outcome, shared by both machines by name. */
enum Outcome derives Finite:
  case accepted, notFound

given Accepted[Outcome] = Accepted(Outcome.accepted)

// ### The product machine: what DescribeActivityExecution shows, with no account of how. A retry
// reads as scheduled again, a pause of a running attempt as started until the worker yields.

enum ProductPhase derives Finite:
  case scheduled, started, paused, cancelRequested
  case completed, failed, canceled, terminated, timedOut

final case class ProductState(phase: ProductPhase) derives Finite

enum ProductFact derives Finite:
  case statusScheduled, statusStarted, statusPaused, statusCancelRequested
  case statusCompleted, statusFailed, statusCanceled, statusTerminated, statusTimedOut

type ProductStep = Step[ProductState, Outcome, ProductFact]

object Product:
  import ProductPhase.*
  import ProductFact.*

  def terminal(s: ProductState): Boolean =
    s.phase.in(completed, failed, canceled, terminated, timedOut)

  /** A paused activity, which no worker is given. */
  def paused(s: ProductState): Boolean = s.phase == ProductPhase.paused

  def running(s: ProductState): Boolean = s.phase == started

  /** Where a worker holds the attempt, so its answer settles the activity. */
  def held(s: ProductState): Boolean = s.phase.in(started, cancelRequested)

  /** Where a pause takes effect: before an attempt starts or while one runs. */
  def pausable(s: ProductState): Boolean = s.phase.in(scheduled, started)

  def attemptStart(s: ProductState): List[ProductStep] =
    if s.phase != scheduled then disabled else accept(ProductState(started), statusStarted)

  /**
   * Unlike the Nexus caller, a retryable failure reads SCHEDULED again with a higher attempt count
   * (TransitionRescheduled), or canceled under a cancel request; the protocol adds the backoff. A
   * canceled answer settles only an activity whose cancellation was requested.
   */
  def attemptResult(s: ProductState, result: AttemptResult): List[ProductStep] =
    if !held(s) then disabled
    else
      result match
        case AttemptResult.completed         => accept(ProductState(completed), statusCompleted)
        case AttemptResult.failed(retryable) =>
          if !retryable then accept(ProductState(failed), statusFailed)
          else if s.phase == cancelRequested then accept(ProductState(canceled), statusCanceled)
          else accept(ProductState(scheduled), statusScheduled)
        case AttemptResult.canceled =>
          if s.phase == cancelRequested then accept(ProductState(canceled), statusCanceled)
          else disabled

  /**
   * A control on an activity that is over is not found. A pause of a paused or cancel-requested
   * activity, or an unpause of one not paused, is FailedPrecondition; the protocol lists them.
   */
  def control(s: ProductState, c: Control): List[ProductStep] =
    if terminal(s) then List(Step(Outcome.notFound, s))
    else
      c match
        case Control.pause =>
          if pausable(s) then accept(ProductState(ProductPhase.paused), statusPaused) else disabled
        case Control.unpause =>
          if paused(s) then accept(ProductState(scheduled), statusScheduled) else disabled
        case Control.requestCancel => accept(ProductState(cancelRequested), statusCancelRequested)
        case Control.terminate     => accept(ProductState(terminated), statusTerminated)

  /** The worker stopping is a fault the Run records and the activity does not feel. */
  def workerStop(@unused s: ProductState): List[ProductStep] = disabled

  /** One of the activity's deadlines firing. Which deadline is the protocol's account of how. */
  def timeout(s: ProductState): List[ProductStep] =
    if terminal(s) then disabled else accept(ProductState(timedOut), statusTimedOut)

val timeout = timer

/** Every status the product machine records is confirmed by the status observation of its name. */
val activityProduct = machine[ProductState, Outcome, ProductFact] {
  forEntity(activity)
  starts(ProductState(ProductPhase.scheduled))
  ends(Product.terminal)
  steps(
    attemptStart ~> Product.attemptStart,
    attemptResult ~> Product.attemptResult,
    control ~> Product.control,
    workerStop ~> Product.workerStop,
    timeout ~> Product.timeout
  )
}

// ### The protocol machine adds the retry, the pause request, the timers and the attempt count. It
// begins before the activity exists, so unstarted is a phase and the start sets the deadlines.

enum Phase derives Finite:
  case unstarted, scheduled, backingOff, started, paused, pauseRequested, cancelRequested
  case completed, failed, canceled, terminated, timedOut

/** Which deadline fired. */
enum TimeoutType derives Finite:
  case scheduleToClose, scheduleToStart, startToClose

/** Bounds the attempt count, as the type of `ProtocolState.attempts` does. */
val attemptBound: Int = 2

/** 12 phases, 3 attempt counts (`0..attemptBound`) and 3 deadline flags: 288 states. */
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

object Protocol:
  import Phase.*
  import ProtocolFact.*

  def terminal(p: Phase): Boolean = p.in(completed, failed, canceled, terminated, timedOut)

  /** Started and not over: the phases a deadline can fire in. */
  def live(p: Phase): Boolean =
    p.in(scheduled, backingOff, started, paused, pauseRequested, cancelRequested)

  /** Where a worker holds the attempt: what start-to-close covers and a worker's answer settles. */
  def held(p: Phase): Boolean = p.in(started, pauseRequested, cancelRequested)

  /** Waiting for a worker: the phases before an attempt is held, which schedule-to-start covers. */
  def waiting(p: Phase): Boolean = p.in(scheduled, backingOff)

  def saturatingSucc(a: UpTo[2]): UpTo[2] = UpTo((a + 1).min(attemptBound))

  def start(
      s: ProtocolState,
      scheduleToClose: Timeout,
      scheduleToStart: Timeout,
      startToClose: Timeout
  ): List[ProtocolStep] =
    if s.phase != Phase.unstarted then disabled
    else
      accept(
        ProtocolState(scheduled, UpTo(0), scheduleToClose, scheduleToStart, startToClose),
        statusScheduled
      )

  /** The worker's poll takes the attempt and raises the count the caller reads back. */
  def attemptStart(s: ProtocolState): List[ProtocolStep] =
    if s.phase != scheduled then disabled
    else
      accept(
        s.copy(phase = started, attempts = saturatingSucc(s.attempts)),
        statusStarted,
        ProtocolFact.attemptCount
      )

  /**
   * A retryable failure backs a started attempt off (read as scheduled again, one attempt higher),
   * settles a cancel-requested one as canceled and lands a pause-requested one in paused
   * (TransitionAttemptFailedWhilePauseRequested). A canceled answer needs a cancel request.
   */
  def attemptResult(s: ProtocolState, result: AttemptResult): List[ProtocolStep] =
    if !held(s.phase) then disabled
    else
      result match
        case AttemptResult.completed         => accept(s.copy(phase = completed), statusCompleted)
        case AttemptResult.failed(retryable) =>
          if !retryable then accept(s.copy(phase = failed), statusFailed)
          else if s.phase == cancelRequested then accept(s.copy(phase = canceled), statusCanceled)
          else if s.phase == pauseRequested then accept(s.copy(phase = paused), statusPaused)
          else
            accept(s.copy(phase = backingOff), statusScheduled, ProtocolFact.attemptCount)
              .because("a retryable failure backs off; the caller reads scheduled again")
        case AttemptResult.canceled =>
          if s.phase == cancelRequested then accept(s.copy(phase = canceled), statusCanceled)
          else disabled

  /**
   * A pause of a held attempt is a request the worker learns of on its next heartbeat, so it is its
   * own phase the caller reads as paused. Each phase a pause or an unpause is disabled in has its
   * arm: the server answers FailedPrecondition ("activity is in non-pausable state", "...
   * non-unpausable state", chasm/lib/activity/operator_commands.go), and a rejecting row would add
   * rows to the table, so they stay disabled until the behavior freeze lifts.
   */
  def control(s: ProtocolState, c: Control): List[ProtocolStep] =
    if terminal(s.phase) then List(Step(Outcome.notFound, s))
    else if s.phase == Phase.unstarted then disabled // no activity yet, so nothing to control
    else
      c match
        case Control.pause =>
          s.phase match
            case Phase.scheduled | Phase.backingOff => accept(s.copy(phase = paused), statusPaused)
            case Phase.started                      =>
              accept(s.copy(phase = pauseRequested), statusPaused)
                .because("the worker learns of the pause on its next heartbeat")
            case Phase.paused | Phase.pauseRequested => disabled // already paused, or asked to be
            case Phase.cancelRequested               => disabled // a cancel request is not pausable
            // Answered above: an activity over is not found, an unstarted one has no control.
            case Phase.unstarted | Phase.completed | Phase.failed | Phase.canceled |
                Phase.terminated | Phase.timedOut =>
              disabled
        case Control.unpause =>
          s.phase match
            case Phase.paused         => accept(s.copy(phase = scheduled), statusScheduled)
            case Phase.pauseRequested => accept(s.copy(phase = started), statusStarted)
            case Phase.scheduled | Phase.backingOff | Phase.started => disabled // not paused
            case Phase.cancelRequested                              => disabled // not paused
            // Answered above: an activity over is not found, an unstarted one has no control.
            case Phase.unstarted | Phase.completed | Phase.failed | Phase.canceled |
                Phase.terminated | Phase.timedOut =>
              disabled
        case Control.requestCancel =>
          accept(s.copy(phase = cancelRequested), statusCancelRequested)
        case Control.terminate => accept(s.copy(phase = terminated), statusTerminated)

  /** Keeps the state and records nothing; the next step's evidence confirms it, a Known Gap. */
  def workerStop(s: ProtocolState): List[ProtocolStep] = stay(s)

  /** The backoff timer. A retry writes nothing the caller can read. */
  def backoff(s: ProtocolState): List[ProtocolStep] =
    if s.phase != backingOff then disabled else accept(s.copy(phase = scheduled))

  def scheduleToClose(s: ProtocolState): List[ProtocolStep] =
    if live(s.phase) && s.scheduleToClose == Timeout.expires then
      accept(s.copy(phase = timedOut), statusTimedOut(TimeoutType.scheduleToClose))
    else disabled

  def scheduleToStart(s: ProtocolState): List[ProtocolStep] =
    if waiting(s.phase) && s.scheduleToStart == Timeout.expires then
      accept(s.copy(phase = timedOut), statusTimedOut(TimeoutType.scheduleToStart))
    else disabled

  def startToClose(s: ProtocolState): List[ProtocolStep] =
    if held(s.phase) && s.startToClose == Timeout.expires then
      accept(s.copy(phase = timedOut), statusTimedOut(TimeoutType.startToClose))
    else disabled

  /**
   * Unstarted and backing off read as scheduled. A pause request reads as started: the worker still
   * holds the attempt, its every answer is a product row from started, and the request stutters.
   */
  def productOf(s: ProtocolState): ProductState = s.phase match
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

val backoff = timer
val scheduleToClose = timer
val scheduleToStart = timer
val startToClose = timer

/** Where every path begins: before the activity exists, with every deadline at its first value. */
val unstarted: ProtocolState =
  ProtocolState(Phase.unstarted, UpTo(0), Timeout.unset, Timeout.unset, Timeout.unset)

/** A timeout is confirmed by the one status observation, whichever deadline fired. */
val activityProtocol = machine[ProtocolState, Outcome, ProtocolFact] {
  forEntity(activity)
  refines(activityProduct)(Protocol.productOf)
  starts(unstarted)
  ends(s => Protocol.terminal(s.phase))
  unobservable(backoff)
  evidence {
    case ProtocolFact.statusTimedOut(_) => "statusTimedOut"
    case ProtocolFact.attemptCount      => attemptCount.name
  }
  steps(
    start ~> Protocol.start,
    attemptStart ~> Protocol.attemptStart,
    attemptResult ~> Protocol.attemptResult,
    control ~> Protocol.control,
    workerStop ~> Protocol.workerStop,
    backoff ~> Protocol.backoff,
    scheduleToClose ~> Protocol.scheduleToClose,
    scheduleToStart ~> Protocol.scheduleToStart,
    startToClose ~> Protocol.startToClose
  )
}

val activityWorker = worker.polling.restrict(workerStop, serve)

final case class StandaloneActivityState(activity: ProtocolState, worker: WorkerState)

// With the worker of its task queue, the stop is the worker's own phase change and every attempt
// start is the worker serving, so an attempt has a row only while the worker polls.
val standaloneActivity =
  compose[StandaloneActivityState](_.activity -> activityProtocol, _.worker -> activityWorker)
    .sync(_.activity -> workerStop, _.worker -> workerStop)
    .sync(_.activity -> attemptStart, _.worker -> serve)
    .ends(s => Protocol.terminal(s.activity.phase))
