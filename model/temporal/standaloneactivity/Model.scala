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
 * Unlike the Nexus caller, the activity has no Stainless kernel: its domains, states and step
 * functions are declared here in ordinary Scala.
 * They come in this order: vocabulary, the two machines, what they promise,
 * what the Queries ask.
 */
package temporal
package standaloneactivity

import umpire.*
import worker.State as WorkerState

val Family: umpire.Family = umpire.Family("temporal.activity.standalone")

// The caller starts and controls the activity; the worker party runs its attempts; system owns the
// timers. The worker's stop is an ordinary action of the worker party, as in every other Model.
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

/** What a control reports. */
enum Delivery derives Finite:
  case accepted, notFound

/** One of the caller's four controls. */
enum Control derives Finite:
  case pause, unpause, requestCancel, terminate

// ### Actions

val start = action("start", caller)
  .input[Timeout]("scheduleToClose")
  .input[Timeout]("scheduleToStart")
  .input[Timeout]("startToClose")
  .creates(activity)
  .schema("temporal.api.workflowservice.v1.StartActivityExecutionRequest")

/** The worker's poll receives the task for the current attempt. */
val attemptStart = action("attemptStart", worker.party)
  .on(activity)
  .schema("temporal.api.workflowservice.v1.PollActivityTaskQueueResponse")

val attemptResult = action("attemptResult", worker.party)
  .on(activity)
  .input[AttemptResult]("result")
  .schema(
    "temporal.api.workflowservice.v1.RespondActivityTaskCompletedRequest",
    "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest",
    "temporal.api.workflowservice.v1.RespondActivityTaskCanceledRequest"
  )
  .example(AttemptResult.failed(false), "ApplicationFailureNonRetryable")
  .example(AttemptResult.failed(true), "ApplicationFailureRetryable")

/**
 * The four caller-side controls are one action with a finite input, because they share a result: a
 * control on an activity that is over is not found.
 */
val control = action("control", caller)
  .on(activity)
  .input[Control]("control")
  .schema(
    "temporal.api.workflowservice.v1.PauseActivityExecutionRequest",
    "temporal.api.workflowservice.v1.UnpauseActivityExecutionRequest",
    "temporal.api.workflowservice.v1.RequestCancelActivityExecutionRequest",
    "temporal.api.workflowservice.v1.TerminateActivityExecutionRequest"
  )
  .results("Delivery")

/**
 * The worker stops polling. Nothing recorded names the activity, so the machines keep their state
 * and record nothing at it.
 */
val workerStop = worker.workerStop

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

private def productStep(phase: ProductPhase, recorded: ProductFact): List[ProductStep] =
  List(Step(Outcome.accepted, ProductState(phase), List(recorded)))

/** The phases the product machine ends on. */
def productTerminal(s: ProductState): Boolean = s.phase match
  case ProductPhase.completed | ProductPhase.failed | ProductPhase.canceled |
      ProductPhase.terminated | ProductPhase.timedOut =>
    true
  case ProductPhase.scheduled | ProductPhase.started | ProductPhase.paused |
      ProductPhase.cancelRequested =>
    false

/** A worker takes the attempt of a scheduled activity. */
def attemptStartStep(s: ProductState): List[ProductStep] =
  if s.phase != ProductPhase.scheduled then Nil
  else productStep(ProductPhase.started, ProductFact.statusStarted)

/**
 * The worker's answer to the attempt. Unlike the Nexus caller, a retryable failure is visible here:
 * DescribeActivityExecution reads SCHEDULED again with a higher attempt count
 * (TransitionRescheduled), and under a cancel request it settles the activity as canceled. The
 * backoff between the two is what the protocol machine adds. A canceled answer settles only an
 * activity whose cancellation was requested.
 */
def attemptResultStep(s: ProductState, result: AttemptResult): List[ProductStep] =
  if s.phase != ProductPhase.started && s.phase != ProductPhase.cancelRequested then Nil
  else
    result match
      case AttemptResult.completed =>
        productStep(ProductPhase.completed, ProductFact.statusCompleted)
      case AttemptResult.failed(retryable) =>
        if !retryable then productStep(ProductPhase.failed, ProductFact.statusFailed)
        else if s.phase == ProductPhase.cancelRequested then
          productStep(ProductPhase.canceled, ProductFact.statusCanceled)
        else productStep(ProductPhase.scheduled, ProductFact.statusScheduled)
      case AttemptResult.canceled =>
        if s.phase == ProductPhase.cancelRequested then
          productStep(ProductPhase.canceled, ProductFact.statusCanceled)
        else Nil

/** A control on an activity that is over is not found and changes nothing. */
def controlStep(s: ProductState, c: Control): List[ProductStep] =
  if productTerminal(s) then List(Step(Outcome.notFound, s))
  else
    c match
      case Control.pause =>
        if s.phase == ProductPhase.scheduled || s.phase == ProductPhase.started then
          productStep(ProductPhase.paused, ProductFact.statusPaused)
        else Nil
      case Control.unpause =>
        if s.phase == ProductPhase.paused then
          productStep(ProductPhase.scheduled, ProductFact.statusScheduled)
        else Nil
      case Control.requestCancel =>
        if s.phase == ProductPhase.scheduled || s.phase == ProductPhase.started || s.phase == ProductPhase.paused ||
          s.phase == ProductPhase.cancelRequested
        then productStep(ProductPhase.cancelRequested, ProductFact.statusCancelRequested)
        else Nil
      case Control.terminate => productStep(ProductPhase.terminated, ProductFact.statusTerminated)

/** The worker stopping is a fault the Run records and the activity does not feel. */
def workerStopStep(s: ProductState): List[ProductStep] = Nil

/** One of the activity's deadlines firing. Which deadline is the protocol's account of how. */
def timeoutStep(s: ProductState): List[ProductStep] =
  if s.phase == ProductPhase.scheduled || s.phase == ProductPhase.started || s.phase == ProductPhase.cancelRequested ||
    s.phase == ProductPhase.paused
  then productStep(ProductPhase.timedOut, ProductFact.statusTimedOut)
  else Nil

val timeout = timer("timeout")

/** The product machine. Every status it records is confirmed by the status observation of its name. */
val activityProduct: Machine[ProductState, Outcome, ProductFact] =
  machine[ProductState, Outcome, ProductFact](Family, "activityProduct") {
    forEntity(activity)
    starts(ProductState(ProductPhase.scheduled))
    ends(productTerminal)
    evidence {
      case ProductFact.statusScheduled       => "statusScheduled"
      case ProductFact.statusStarted         => "statusStarted"
      case ProductFact.statusPaused          => "statusPaused"
      case ProductFact.statusCancelRequested => "statusCancelRequested"
      case ProductFact.statusCompleted       => "statusCompleted"
      case ProductFact.statusFailed          => "statusFailed"
      case ProductFact.statusCanceled        => "statusCanceled"
      case ProductFact.statusTerminated      => "statusTerminated"
      case ProductFact.statusTimedOut        => "statusTimedOut"
    }
    steps(
      attemptStart ~> attemptStartStep,
      attemptResult ~> attemptResultStep,
      control ~> controlStep,
      workerStop ~> workerStopStep,
      timeout ~> timeoutStep
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

/** Bounds the attempt count. */
val attemptBound: Int = 2

/**
 * The protocol machine's state: 12 phases, 3 attempt counts (`0..attemptBound`) and 3 deadline flags,
 * 288 states.
 */
final case class ProtocolState(
    phase: Phase,
    attempts: Int,
    scheduleToClose: Timeout,
    scheduleToStart: Timeout,
    startToClose: Timeout
)

given Finite[ProtocolState] =
  given Finite[Int] = Finite.upTo(attemptBound)
  Finite.derived

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

def terminalPhase(p: Phase): Boolean =
  p == Phase.completed || p == Phase.failed || p == Phase.canceled || p == Phase.terminated || p == Phase.timedOut

/** Started and not over: the phases a deadline can fire in. */
def running(p: Phase): Boolean =
  p == Phase.scheduled || p == Phase.backingOff || p == Phase.started || p == Phase.paused ||
    p == Phase.pauseRequested || p == Phase.cancelRequested

/**
 * Where a worker holds the attempt: the phases a start-to-close deadline covers and a worker's answer
 * settles.
 */
def attemptHeld(p: Phase): Boolean =
  p == Phase.started || p == Phase.pauseRequested || p == Phase.cancelRequested

private def saturatingSucc(a: Int): Int = (a + 1).min(attemptBound)

private def moves(s: ProtocolState, phase: Phase, recorded: ProtocolFact*): List[ProtocolStep] =
  List(Step(Outcome.accepted, s.copy(phase = phase), recorded.toList))

def startStep(
    s: ProtocolState,
    scheduleToClose: Timeout,
    scheduleToStart: Timeout,
    startToClose: Timeout
): List[ProtocolStep] =
  if s.phase != Phase.unstarted then Nil
  else
    List(
      Step(
        Outcome.accepted,
        ProtocolState(Phase.scheduled, 0, scheduleToClose, scheduleToStart, startToClose),
        List(ProtocolFact.statusScheduled)
      )
    )

/** The worker's poll takes the attempt and raises the count the caller reads back. */
def protocolAttemptStartStep(s: ProtocolState): List[ProtocolStep] =
  if s.phase != Phase.scheduled then Nil
  else
    moves(
      s.copy(attempts = saturatingSucc(s.attempts)),
      Phase.started,
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
def protocolAttemptResultStep(s: ProtocolState, result: AttemptResult): List[ProtocolStep] =
  if !attemptHeld(s.phase) then Nil
  else
    result match
      case AttemptResult.completed => moves(s, Phase.completed, ProtocolFact.statusCompleted)
      case AttemptResult.failed(retryable) =>
        if !retryable then moves(s, Phase.failed, ProtocolFact.statusFailed)
        else if s.phase == Phase.cancelRequested then
          moves(s, Phase.canceled, ProtocolFact.statusCanceled)
        else if s.phase == Phase.pauseRequested then
          moves(s, Phase.paused, ProtocolFact.statusPaused)
        else
          List(
            Step(
              Outcome.accepted,
              s.copy(phase = Phase.backingOff),
              List(ProtocolFact.statusScheduled, ProtocolFact.attemptCount),
              because = "a retryable failure backs off; the caller reads scheduled again"
            )
          )
      case AttemptResult.canceled =>
        if s.phase == Phase.cancelRequested then
          moves(s, Phase.canceled, ProtocolFact.statusCanceled)
        else Nil

/**
 * The caller's controls. A pause of a held attempt is a request the worker learns of on its next
 * heartbeat, so it is its own phase; the caller reads it as paused either way.
 */
def protocolControlStep(s: ProtocolState, c: Control): List[ProtocolStep] =
  if terminalPhase(s.phase) then List(Step(Outcome.notFound, s))
  else if s.phase == Phase.unstarted then Nil
  else
    c match
      case Control.pause =>
        s.phase match
          case Phase.scheduled | Phase.backingOff =>
            moves(s, Phase.paused, ProtocolFact.statusPaused)
          case Phase.started =>
            List(
              Step(
                Outcome.accepted,
                s.copy(phase = Phase.pauseRequested),
                List(ProtocolFact.statusPaused),
                because = "the worker learns of the pause on its next heartbeat"
              )
            )
          case _ => Nil
      case Control.unpause =>
        s.phase match
          case Phase.paused         => moves(s, Phase.scheduled, ProtocolFact.statusScheduled)
          case Phase.pauseRequested => moves(s, Phase.started, ProtocolFact.statusStarted)
          case _                    => Nil
      case Control.requestCancel =>
        moves(s, Phase.cancelRequested, ProtocolFact.statusCancelRequested)
      case Control.terminate => moves(s, Phase.terminated, ProtocolFact.statusTerminated)

/**
 * The worker stopping keeps the state and records nothing; on a path it is confirmed by the evidence
 * of the step after it, and the Case says so in a Known Gap.
 */
def protocolWorkerStopStep(s: ProtocolState): List[ProtocolStep] = List(Step(Outcome.accepted, s))

/** The backoff timer. A retry writes nothing the caller can read. */
def backoffStep(s: ProtocolState): List[ProtocolStep] =
  if s.phase != Phase.backingOff then Nil else moves(s, Phase.scheduled)

def scheduleToCloseStep(s: ProtocolState): List[ProtocolStep] =
  if running(s.phase) && s.scheduleToClose == Timeout.expires then
    moves(s, Phase.timedOut, ProtocolFact.statusTimedOut(TimeoutType.scheduleToClose))
  else Nil

def scheduleToStartStep(s: ProtocolState): List[ProtocolStep] =
  if (s.phase == Phase.scheduled || s.phase == Phase.backingOff) && s.scheduleToStart == Timeout.expires
  then moves(s, Phase.timedOut, ProtocolFact.statusTimedOut(TimeoutType.scheduleToStart))
  else Nil

def startToCloseStep(s: ProtocolState): List[ProtocolStep] =
  if attemptHeld(s.phase) && s.startToClose == Timeout.expires then
    moves(s, Phase.timedOut, ProtocolFact.statusTimedOut(TimeoutType.startToClose))
  else Nil

/**
 * How a protocol state reads as a product state: not yet started and backing off read as scheduled;
 * a pause request reads as started, because the worker still holds the attempt and every answer it
 * can give is a row the product has from started, while the request itself is a stutter; every other
 * phase is its namesake.
 */
def productOf(s: ProtocolState): ProductState = s.phase match
  case Phase.unstarted | Phase.scheduled | Phase.backingOff => ProductState(ProductPhase.scheduled)
  case Phase.started | Phase.pauseRequested                 => ProductState(ProductPhase.started)
  case Phase.paused                                         => ProductState(ProductPhase.paused)
  case Phase.cancelRequested => ProductState(ProductPhase.cancelRequested)
  case Phase.completed       => ProductState(ProductPhase.completed)
  case Phase.failed          => ProductState(ProductPhase.failed)
  case Phase.canceled        => ProductState(ProductPhase.canceled)
  case Phase.terminated      => ProductState(ProductPhase.terminated)
  case Phase.timedOut        => ProductState(ProductPhase.timedOut)

val backoff = timer("backoff")
val scheduleToClose = timer("scheduleToClose")
val scheduleToStart = timer("scheduleToStart")
val startToClose = timer("startToClose")

/** Where every path begins: before the activity exists, with every deadline at its first value. */
val unstarted: ProtocolState =
  ProtocolState(Phase.unstarted, 0, Timeout.unset, Timeout.unset, Timeout.unset)

/** The protocol machine. */
val activityProtocol: Machine[ProtocolState, Outcome, ProtocolFact] =
  machine[ProtocolState, Outcome, ProtocolFact](Family, "activityProtocol") {
    forEntity(activity)
    refines(activityProduct)(productOf)
    starts(unstarted)
    ends(s => terminalPhase(s.phase))
    unobservable(backoff)
    evidence {
      case ProtocolFact.statusScheduled       => "statusScheduled"
      case ProtocolFact.statusStarted         => "statusStarted"
      case ProtocolFact.statusPaused          => "statusPaused"
      case ProtocolFact.statusCancelRequested => "statusCancelRequested"
      case ProtocolFact.statusCompleted       => "statusCompleted"
      case ProtocolFact.statusFailed          => "statusFailed"
      case ProtocolFact.statusCanceled        => "statusCanceled"
      case ProtocolFact.statusTerminated      => "statusTerminated"
      case ProtocolFact.statusTimedOut(_)     => "statusTimedOut"
      case ProtocolFact.attemptCount          => attemptCount.name
    }
    steps(
      start ~> startStep,
      attemptStart ~> protocolAttemptStartStep,
      attemptResult ~> protocolAttemptResultStep,
      control ~> protocolControlStep,
      workerStop ~> protocolWorkerStopStep,
      backoff ~> backoffStep,
      scheduleToClose ~> scheduleToCloseStep,
      scheduleToStart ~> scheduleToStartStep,
      startToClose ~> startToCloseStep
    )
  }

/** The activity's view of its worker: it stops and it serves. */
val activityWorker: Machine[WorkerState, worker.Outcome, worker.Fact] =
  worker.polling.restrict(Family, "activityWorker")(worker.workerStop, worker.serve)

// ### The activity and its worker
//
// Composed with the worker of the activity's task queue, the stop is the worker's own phase change
// and every attempt start is the worker serving, so an attempt has a row only while the worker
// polls.

final case class StandaloneActivityState(activity: ProtocolState, worker: WorkerState)

/** The protocol machine composed with the activity's worker. */
val standaloneActivity: Composition[StandaloneActivityState] =
  compose[StandaloneActivityState](Family, "standaloneActivity")(
    "activity" -> activityProtocol,
    "worker" -> activityWorker
  )
    .sync("workerStop", "activity" -> workerStop, "worker" -> worker.workerStop)
    .sync("attemptStart", "activity" -> attemptStart, "worker" -> worker.serve)
    .ends(s => terminalPhase(s.activity.phase))

/**
 * The product machine as it stood before the caller's controls were added: the revision the
 * generated behavior diff compares against.
 */
def productWithoutControls: Machine[ProductState, Outcome, ProductFact] =
  activityProduct.restrict(Family, "activityProduct")(
    attemptStart,
    attemptResult,
    workerStop,
    timeout
  )
