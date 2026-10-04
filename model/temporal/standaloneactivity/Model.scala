/* The standalone activity Model: one activity started directly through StartActivityExecution, with
 * no workflow around it. The product machine says what an activity does as
 * DescribeActivityExecution reports it, the protocol machine says how the server gets there and
 * refines it, and the functional Queries are one per side effect that settles the activity.
 * Grounded in chasm/lib/activity/statemachine.go; reset is deferred, like cancellation in the Nexus
 * caller Model, and the heartbeat timeout is not modeled.
 *
 * A standalone activity writes no history event. Every fact below is a status read through
 * DescribeActivityExecution, or a result read through PollActivityExecution, so the evidence lines
 * of the machines name observations rather than events.
 *
 * Files are split by kind: this one declares the vocabulary, the two machines and their composition
 * with the worker; Properties.scala says what they promise and Queries.scala what the Queries ask. The
 * system contract is split by subject into admission/ and compositions/, which repeat those files.
 */
package temporal
package standaloneactivity

import scala.annotation.unused
import umpire.*
import worker.{serve, workerStop, State as WorkerState}
import io.temporal.api.workflowservice.v1.*
import ActivityFamily.given

/**
 * The family of the activity's machines. The system contract in admission/ and compositions/ has a
 * family of its own, so each file imports the one its declarations take.
 */
object ActivityFamily:
  given family: Family = Family("temporal.activity.standalone")

/**
 * The family of the system contract's machines and compositions. The shared task queue keeps the same
 * family through its own given, since it was first written in the system contract.
 */
object SystemFamily:
  given family: Family = Family("temporal.activity.standalone.system")

// The caller starts and controls the activity; the worker party runs its attempts; system owns the
// timers. The worker's stop is the worker's own action, `workerStop`: nothing it records names the
// activity, so the activity's machines keep their state at it.
val caller: Party = Party("caller")

// ### Entities

/**
 * Named by the id the caller chose for it: every status read and every result read carries it, and
 * no run id or event id is needed to tell two apart.
 */
val activity: Entity = Entity("activity", key = "activityId")

// ### The input domains
//
// As in the caller Model, a variant with a finite field contributes one class per assignment:
// failed(retryable) is two classes, which mirrors the retryable flag of an ApplicationFailure.

/** Whether the start request sets a deadline. */
enum Timeout derives Finite:
  case unset, expires

/**
 * The worker's answer to an attempt. `canceled` is the worker's canceled answer, spelled the way
 * the Temporal API spells it.
 */
enum AttemptResult derives Finite:
  case completed
  case failed(retryable: Boolean)
  case canceled

/** One of the caller's four controls. */
enum Control derives Finite:
  case pause, unpause, requestCancel, terminate

// ### Actions

/**
 * The inputs of the start request, the worker's answer and a control. They live in an object of their
 * own because the deadline timers and the control action already take their names.
 */
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
val attemptStart = action(worker.party)
  .on(activity)
  .schema[PollActivityTaskQueueResponse]

val attemptResult = action(worker.party)
  .on(activity)
  .input(Inputs.result)
  .schema[RespondActivityTaskCompletedRequest]
  .schema[RespondActivityTaskFailedRequest]
  .schema[RespondActivityTaskCanceledRequest]
  .example(AttemptResult.failed(false), "ApplicationFailureNonRetryable")
  .example(AttemptResult.failed(true), "ApplicationFailureRetryable")

/**
 * The four caller-side controls are one action with a finite input, because they share a result: a
 * control on an activity that is over is not found. The result text names that delivery of the
 * control; it is metadata of the action, not a domain any state holds.
 */
val control = action(caller)
  .on(activity)
  .input(Inputs.control)
  .schema[PauseActivityExecutionRequest]
  .schema[UnpauseActivityExecutionRequest]
  .schema[RequestCancelActivityExecutionRequest]
  .schema[TerminateActivityExecutionRequest]
  .results("Delivery")

// ### The derived observation
//
// A retried attempt writes nothing the caller can see except the attempt count that
// DescribeActivityExecution reports, so it is the one derived observation. Each status a machine
// records is an observation of the status field: nine observations over that one field, which an
// evidence catalog may not accept as distinct. They are declared here as data, and the question is
// left to the realization.

val attemptCount: Observation = Observation("attemptCount", activity, "attempt")

/** A step's outcome, shared by both machines by name. */
enum Outcome derives Finite:
  case accepted, notFound

given Accepted[Outcome] = Accepted(Outcome.accepted)

// ### The product machine
//
// What the caller sees through DescribeActivityExecution, with no account of how: a retry reads as
// scheduled again, and a pause requested of a running attempt reads as started until the worker
// yields.

enum ProductPhase derives Finite:
  case scheduled, started, paused, cancelRequested, completed, failed, canceled, terminated,
    timedOut

final case class ProductState(phase: ProductPhase) derives Finite

/** A status the caller reads. */
enum ProductFact derives Finite:
  case statusScheduled, statusStarted, statusPaused, statusCancelRequested, statusCompleted,
    statusFailed,
    statusCanceled, statusTerminated, statusTimedOut

type ProductStep = Step[ProductState, Outcome, ProductFact]

/**
 * The product machine's vocabulary: its status sets, which its step functions and its promises read
 * by name, and its step functions, named after the actions they answer.
 */
object Product:
  /** The phases the product machine ends on. */
  def terminal(s: ProductState): Boolean =
    s.phase.in(
      ProductPhase.completed,
      ProductPhase.failed,
      ProductPhase.canceled,
      ProductPhase.terminated,
      ProductPhase.timedOut
    )

  /** A paused activity, which no worker is given. */
  def paused(s: ProductState): Boolean = s.phase == ProductPhase.paused

  /** An activity a worker runs an attempt of. */
  def running(s: ProductState): Boolean = s.phase == ProductPhase.started

  /** Where a worker holds the attempt, so its answer settles the activity. */
  def held(s: ProductState): Boolean =
    s.phase.in(ProductPhase.started, ProductPhase.cancelRequested)

  /** Where a pause takes effect: before an attempt starts or while one runs. */
  def pausable(s: ProductState): Boolean = s.phase.in(ProductPhase.scheduled, ProductPhase.started)

  /** A worker takes the attempt of a scheduled activity. */
  def attemptStart(s: ProductState): List[ProductStep] =
    if s.phase != ProductPhase.scheduled then disabled
    else accept(ProductState(ProductPhase.started), ProductFact.statusStarted)

  /**
   * The worker's answer to the attempt. Unlike the Nexus caller, a retryable failure is visible here:
   * DescribeActivityExecution reads SCHEDULED again with a higher attempt count
   * (TransitionRescheduled), and under a cancel request it settles the activity as canceled. The
   * backoff between the two is what the protocol machine adds. A canceled answer settles only an
   * activity whose cancellation was requested.
   */
  def attemptResult(s: ProductState, result: AttemptResult): List[ProductStep] =
    if !held(s) then disabled
    else
      result match
        case AttemptResult.completed =>
          accept(ProductState(ProductPhase.completed), ProductFact.statusCompleted)
        case AttemptResult.failed(retryable) =>
          if !retryable then accept(ProductState(ProductPhase.failed), ProductFact.statusFailed)
          else if s.phase == ProductPhase.cancelRequested then
            accept(ProductState(ProductPhase.canceled), ProductFact.statusCanceled)
          else accept(ProductState(ProductPhase.scheduled), ProductFact.statusScheduled)
        case AttemptResult.canceled =>
          if s.phase == ProductPhase.cancelRequested then
            accept(ProductState(ProductPhase.canceled), ProductFact.statusCanceled)
          else disabled

  /**
   * A control on an activity that is over is not found and changes nothing. A pause of an activity
   * that is paused or cancel-requested, and an unpause of one that is not paused, are the server's
   * FailedPrecondition; the protocol machine lists them.
   */
  def control(s: ProductState, c: Control): List[ProductStep] =
    if terminal(s) then List(Step(Outcome.notFound, s))
    else
      c match
        case Control.pause =>
          if pausable(s) then accept(ProductState(ProductPhase.paused), ProductFact.statusPaused)
          else disabled
        case Control.unpause =>
          if paused(s) then
            accept(ProductState(ProductPhase.scheduled), ProductFact.statusScheduled)
          else disabled
        case Control.requestCancel =>
          accept(ProductState(ProductPhase.cancelRequested), ProductFact.statusCancelRequested)
        case Control.terminate =>
          accept(ProductState(ProductPhase.terminated), ProductFact.statusTerminated)

  /** The worker stopping is a fault the Run records and the activity does not feel. */
  def workerStop(@unused s: ProductState): List[ProductStep] = disabled

  /** One of the activity's deadlines firing. Which deadline is the protocol's account of how. */
  def timeout(s: ProductState): List[ProductStep] =
    if terminal(s) then disabled
    else accept(ProductState(ProductPhase.timedOut), ProductFact.statusTimedOut)

val timeout = timer

/** The product machine. Every status it records is confirmed by the status observation of its name. */
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

// ### The protocol machine
//
// How the server gets there: the retry the product machine cannot see, the pause a running attempt
// turns into a pause request, the three timers the start request sets, and the attempt count. The
// machine begins before the activity exists, so unstarted is a phase and the start request is what
// sets the deadlines.

enum Phase derives Finite:
  case unstarted, scheduled, backingOff, started, paused, pauseRequested, cancelRequested,
    completed, failed,
    canceled, terminated, timedOut

/** Which deadline fired. */
enum TimeoutType derives Finite:
  case scheduleToClose, scheduleToStart, startToClose

/** Bounds the attempt count, as the type of `ProtocolState.attempts` does. */
val attemptBound: Int = 2

/**
 * The protocol machine's state: 12 phases, 3 attempt counts (`0..attemptBound`) and 3 deadline flags,
 * 288 states.
 */
final case class ProtocolState(
    phase: Phase,
    attempts: UpTo[2],
    scheduleToClose: Timeout,
    scheduleToStart: Timeout,
    startToClose: Timeout
) derives Finite

/**
 * What the protocol machine records. `attemptCount` is spelled as the observation it is read
 * through.
 */
enum ProtocolFact derives Finite:
  case statusScheduled, statusStarted, statusPaused, statusCancelRequested, statusCompleted,
    statusFailed,
    statusCanceled, statusTerminated
  case statusTimedOut(timeoutType: TimeoutType)
  case attemptCount

type ProtocolStep = Step[ProtocolState, Outcome, ProtocolFact]

/**
 * The protocol machine's vocabulary: its phase sets, which its step functions read by name, and its
 * step functions, named after the actions they answer, as the product machine's are.
 */
object Protocol:
  /** The phases the protocol machine ends on. */
  def terminal(p: Phase): Boolean =
    p.in(Phase.completed, Phase.failed, Phase.canceled, Phase.terminated, Phase.timedOut)

  /** Started and not over: the phases a deadline can fire in. */
  def live(p: Phase): Boolean =
    p.in(
      Phase.scheduled,
      Phase.backingOff,
      Phase.started,
      Phase.paused,
      Phase.pauseRequested,
      Phase.cancelRequested
    )

  /**
   * Where a worker holds the attempt: the phases a start-to-close deadline covers and a worker's
   * answer settles.
   */
  def held(p: Phase): Boolean = p.in(Phase.started, Phase.pauseRequested, Phase.cancelRequested)

  /** Waiting for a worker: the phases before an attempt is held, which schedule-to-start covers. */
  def waiting(p: Phase): Boolean = p.in(Phase.scheduled, Phase.backingOff)

  /** One more attempt, saturating at `attemptBound`. */
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
        ProtocolState(Phase.scheduled, UpTo(0), scheduleToClose, scheduleToStart, startToClose),
        ProtocolFact.statusScheduled
      )

  /** The worker's poll takes the attempt and raises the count the caller reads back. */
  def attemptStart(s: ProtocolState): List[ProtocolStep] =
    if s.phase != Phase.scheduled then disabled
    else
      accept(
        s.copy(phase = Phase.started, attempts = saturatingSucc(s.attempts)),
        ProtocolFact.statusStarted,
        ProtocolFact.attemptCount
      )

  /**
   * The worker's answer, by the phase it lands in. A retryable failure backs a started attempt off,
   * which the caller reads as scheduled again with a higher attempt count, settles a cancel-requested
   * one as canceled, and lands a pause-requested one in paused
   * (TransitionAttemptFailedWhilePauseRequested). A canceled answer is honored only under a cancel
   * request.
   */
  def attemptResult(s: ProtocolState, result: AttemptResult): List[ProtocolStep] =
    if !held(s.phase) then disabled
    else
      result match
        case AttemptResult.completed =>
          accept(s.copy(phase = Phase.completed), ProtocolFact.statusCompleted)
        case AttemptResult.failed(retryable) =>
          if !retryable then accept(s.copy(phase = Phase.failed), ProtocolFact.statusFailed)
          else if s.phase == Phase.cancelRequested then
            accept(s.copy(phase = Phase.canceled), ProtocolFact.statusCanceled)
          else if s.phase == Phase.pauseRequested then
            accept(s.copy(phase = Phase.paused), ProtocolFact.statusPaused)
          else
            accept(
              s.copy(phase = Phase.backingOff),
              ProtocolFact.statusScheduled,
              ProtocolFact.attemptCount
            ).because("a retryable failure backs off; the caller reads scheduled again")
        case AttemptResult.canceled =>
          if s.phase == Phase.cancelRequested then
            accept(s.copy(phase = Phase.canceled), ProtocolFact.statusCanceled)
          else disabled

  /**
   * The caller's controls. A pause of a held attempt is a request the worker learns of on its next
   * heartbeat, so it is its own phase; the caller reads it as paused either way.
   *
   * Each phase a pause or an unpause is disabled in has its arm. The server answers those requests
   * with FailedPrecondition ("activity is in non-pausable state", "... non-unpausable state",
   * chasm/lib/activity/operator_commands.go). A row that rejects them would add rows to the table, so
   * they stay disabled until the behavior freeze lifts.
   */
  def control(s: ProtocolState, c: Control): List[ProtocolStep] =
    if terminal(s.phase) then List(Step(Outcome.notFound, s))
    else if s.phase == Phase.unstarted then disabled // no activity yet, so nothing to control
    else
      c match
        case Control.pause =>
          s.phase match
            case Phase.scheduled | Phase.backingOff =>
              accept(s.copy(phase = Phase.paused), ProtocolFact.statusPaused)
            case Phase.started =>
              accept(s.copy(phase = Phase.pauseRequested), ProtocolFact.statusPaused)
                .because("the worker learns of the pause on its next heartbeat")
            case Phase.paused | Phase.pauseRequested => disabled // already paused, or asked to be
            case Phase.cancelRequested               => disabled // a cancel request is not pausable
            // Answered above: an activity that is over is not found, an unstarted one has no control.
            case Phase.unstarted | Phase.completed | Phase.failed | Phase.canceled |
                Phase.terminated | Phase.timedOut =>
              disabled
        case Control.unpause =>
          s.phase match
            case Phase.paused =>
              accept(s.copy(phase = Phase.scheduled), ProtocolFact.statusScheduled)
            case Phase.pauseRequested =>
              accept(s.copy(phase = Phase.started), ProtocolFact.statusStarted)
            case Phase.scheduled | Phase.backingOff | Phase.started => disabled // not paused
            case Phase.cancelRequested                              => disabled // not paused
            // Answered above: an activity that is over is not found, an unstarted one has no control.
            case Phase.unstarted | Phase.completed | Phase.failed | Phase.canceled |
                Phase.terminated | Phase.timedOut =>
              disabled
        case Control.requestCancel =>
          accept(s.copy(phase = Phase.cancelRequested), ProtocolFact.statusCancelRequested)
        case Control.terminate =>
          accept(s.copy(phase = Phase.terminated), ProtocolFact.statusTerminated)

  /**
   * The worker stopping keeps the state and records nothing; on a path it is confirmed by the
   * evidence of the step after it, and the Case says so in a Known Gap.
   */
  def workerStop(s: ProtocolState): List[ProtocolStep] = stay(s)

  /** The backoff timer. A retry writes nothing the caller can read. */
  def backoff(s: ProtocolState): List[ProtocolStep] =
    if s.phase != Phase.backingOff then disabled else accept(s.copy(phase = Phase.scheduled))

  def scheduleToClose(s: ProtocolState): List[ProtocolStep] =
    if live(s.phase) && s.scheduleToClose == Timeout.expires then
      accept(
        s.copy(phase = Phase.timedOut),
        ProtocolFact.statusTimedOut(TimeoutType.scheduleToClose)
      )
    else disabled

  def scheduleToStart(s: ProtocolState): List[ProtocolStep] =
    if waiting(s.phase) && s.scheduleToStart == Timeout.expires then
      accept(
        s.copy(phase = Phase.timedOut),
        ProtocolFact.statusTimedOut(TimeoutType.scheduleToStart)
      )
    else disabled

  def startToClose(s: ProtocolState): List[ProtocolStep] =
    if held(s.phase) && s.startToClose == Timeout.expires then
      accept(s.copy(phase = Phase.timedOut), ProtocolFact.statusTimedOut(TimeoutType.startToClose))
    else disabled

  /**
   * How a protocol state reads as a product state: not yet started and backing off read as
   * scheduled; a pause request reads as started, because the worker still holds the attempt and
   * every answer it can give is a row the product has from started, while the request itself is a
   * stutter; every other phase is its namesake.
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

/**
 * The protocol machine. A status is confirmed by the status observation of its name; a timeout by
 * the one status observation whichever deadline fired, and the attempt count by its observation.
 */
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

/** The activity's view of its worker: it stops and it serves. */
val activityWorker = worker.polling.restrict(workerStop, serve)

// ### The activity and its worker
//
// Composed with the worker of the activity's task queue, the stop is the worker's own phase change
// and every attempt start is the worker serving, so an attempt has a row only while the worker
// polls.

final case class StandaloneActivityState(activity: ProtocolState, worker: WorkerState)

/** The protocol machine composed with the activity's worker. */
val standaloneActivity =
  compose[StandaloneActivityState](_.activity -> activityProtocol, _.worker -> activityWorker)
    .sync("workerStop", _.activity -> workerStop, _.worker -> workerStop)
    .sync("attemptStart", _.activity -> attemptStart, _.worker -> serve)
    .ends(s => Protocol.terminal(s.activity.phase))

/**
 * The product machine as it stood before the caller's controls were added: the revision the
 * generated behavior diff compares against.
 */
def productWithoutControls: Machine[ProductState, Outcome, ProductFact] =
  activityProduct.restrict(ActivityFamily.family, "activityProduct")(
    attemptStart,
    attemptResult,
    workerStop,
    timeout
  )
