/* The standalone activity Model: one activity started directly through StartActivityExecution, with
 * no workflow around it, grounded in chasm/lib/activity/statemachine.go. The product machine says
 * what DescribeActivityExecution reports, the protocol how the server gets there. No history
 * event is written, so every evidence line names an observation: a status read through
 * DescribeActivityExecution or a result read through PollActivityExecution. Reset is deferred, like
 * cancellation in the Nexus caller Model, and the heartbeat timeout is not modeled.
 *
 * The feature has two levels, each in a folder of its own, because different people read them
 * (model/irgen/testdata/layout/lamp is the template):
 *
 *   - this file: the types; the signature (the activity and its inputs; the caller and its actions,
 *     the worker's actions on the activity, its timers and deadlines; and the bounds); and last
 *     exports, its IR files;
 *   - product/Product.scala: ActivityProduct, the product machine, what a caller reads;
 *   - system/System.scala: ActivityProtocol, the protocol machine that refines it; ActivityWorker,
 *     the worker of its task queue; and StandaloneActivity, the protocol with that worker;
 *   - system/Record.scala: the system contract, history's record of the activity, and its designs;
 *   - system/WithTaskQueue.scala: the contract's designs composed with the shared task queue.
 *
 * A machine object reads its header (entity, init, end, evidence), then its sections in order:
 * states, refinement, effects, monitors, rules, properties, implements and queries. A composition
 * reads end, then states, syncs, properties, implements and queries. Realization.scala realizes it.
 */
package temporal
package features.standaloneactivity

import umpire.*
import shared.worker.{worker as process, State as WorkerState}
import io.temporal.api.workflowservice.v1.*
import product.ActivityProduct
import system.{ActivityProtocol, StandaloneActivity}

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

/** The protocol machine's phases. It begins before the activity exists, so unstarted is one. */
enum Phase derives Finite:
  case unstarted, scheduled, backingOff, started, paused, pauseRequested, cancelRequested
  case completed, failed, canceled, terminated, timedOut

/** Which deadline fired. */
enum TimeoutType derives Finite:
  case scheduleToClose, scheduleToStart, startToClose

/**
 * 12 phases, 3 attempt counts (`0..ActivityProtocol.attemptBound`) and 3 deadline flags: 288
 * states.
 */
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

// Who acts, and on what: each action is declared in the object of who takes it, and named after
// where it is declared, `temporal.features.standaloneactivity.caller.start`.

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
 * The shared worker's actions on this activity: its poll receives the task for the current attempt,
 * and its answer settles it. The worker's stop is the worker's own action, `process.workerStop`:
 * nothing it records names the activity, so the activity's machines keep their state.
 */
object worker:
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
object timers:
  val timeout = timer
  val backoff = timer

/** The protocol's three deadlines, each armed by the start's input of its name. */
object deadline:
  val scheduleToClose = timer
  val scheduleToStart = timer
  val startToClose = timer

// A retry shows the caller only the attempt count DescribeActivityExecution reports. The statuses
// observe one status field; whether a catalog tells them apart is left to the realization.
val attemptCount = Observation(on = activity, read = "attempt")

given Ok[Outcome] = Ok(Outcome.accepted)

// The bounds of the levels' Queries and the system contract's, beside three and four (shared.Bounds).
val five = Limits(steps = 5, actions = 5, search = 65536)
val six = Limits(steps = 6, actions = 6, search = 262144)
val eight = Limits(steps = 8, actions = 8, search = 262144)

// ### The checked-in IR files of the standalone activity Models (umpire.irFile).

object exports:
  // The activity Model. Its cross-entity Query, stoppedWorkerStartsNothing, carries the composition
  // and its claim.
  val activity = irFile("activity")(
    StandaloneActivity,
    ActivityProduct,
    ActivityProduct.implements,
    ActivityProtocol.implements,
    ActivityProtocol.queries,
    StandaloneActivity.queries,
    ActivityRealization.standalone
  )

  // Its system contract, the admission designs, and the shared task queue's providers it composes.
  // A composition no Query runs over is a root of its own.
  val activitySystem = irFile("activity-system")(
    system.CurrentAdmission.queries,
    system.StaleAdmission.queries,
    shared.taskqueue.system.MatchingQueue.queries,
    shared.taskqueue.system.ForgetfulQueue.queries,
    shared.taskqueue.system.VolatileQueue.queries,
    shared.taskqueue.system.LossyMatchingQueue.queries,
    system.CurrentOverQueue.queries,
    system.StaleOverQueue.queries,
    system.CurrentOverMatching.queries,
    system.StaleOverMatching.queries,
    system.CurrentOverLossyMatching.queries,
    system.CurrentOverForgetful,
    system.CurrentOverVolatile
  )

  // The held race a server is run through, and the realization that runs it. It is a Model of its
  // own, so the system contract's Queries are the ones its checkers were given.
  val activityRace = irFile("activity-race")(
    system.HeldAdmission.queries,
    ActivityRealization.heldDelivery,
    system.AdmissionResponseLoss.queries,
    ActivityRealization.lostAdmissionResponse
  )
