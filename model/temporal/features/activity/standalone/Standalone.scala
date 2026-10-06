/* The standalone activity Model: one activity started directly through StartActivityExecution, with
 * no workflow around it, grounded in chasm/lib/activity/statemachine.go. The product machine says
 * what DescribeActivityExecution reports, the System how the server gets there. No history
 * event is written, so every evidence line names an observation: a status read through
 * DescribeActivityExecution or a result read through PollActivityExecution. Reset is deferred, like
 * cancellation in the Nexus client Model, and the heartbeat timeout is not modeled.
 *
 * Update this Model independently of the implementation. When conformance fails, ask a human
 * rather than fitting the Model to the code.
 *
 * The feature has two levels, each in a folder of its own, because different people read them
 * (model/irgen/testdata/layout/lamp is the template):
 *
 *   - this file: the shared types; the signature (the activity and its inputs; the client and its
 *     actions, the worker's actions on the activity, its timers and deadlines; and the bounds); and
 *     last exports, its IR files;
 *   - product/Product.scala: Product Phase, State and Fact; ActivityProduct, the product machine,
 *     what a client reads;
 *   - system/System.scala: System Phase, State, Fact, timer and composition types; ActivitySystem,
 *     the System machine that refines it; ActivityWorker, the worker of its task queue; and
 *     StandaloneActivity, the System with that worker;
 *   - system/Record.scala: the history record of the activity, and its designs;
 *   - system/WithTaskQueue.scala: the contract's designs composed with the shared task queue.
 *   - system/Realization.scala: the executable System realizations of StandaloneActivity,
 *     HeldDispatch and LostStartAnswer.
 *
 * A machine object reads its header (entity, init, end, evidence), then its sections in order:
 * states, refinement, effects, monitors, rules, properties, implements and queries. A composition
 * reads end, then states, syncs, properties, implements and queries.
 */
package temporal
package features.activity.standalone

import umpire.*
import shared.worker.worker as process
import io.temporal.api.workflowservice.v1.*
import product.ActivityProduct
import system.{ActivityRealization, ActivitySystem, StandaloneActivity}

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

// ### Signature

/** Named by the id the client chose: every read carries it, so no run id or event id is needed. */
val activity = Entity(key = "activityId")

// The start's inputs, which the deadline timers no longer collide with, and the worker's answer.
val scheduleToClose = input[Timeout]
val scheduleToStart = input[Timeout]
val startToClose = input[Timeout]
val result = input[AttemptResult]

/** The control's input, apart because inside `client` its name is the control action. */
object Inputs:
  val control = input[Control]

// Who acts, and on what: each action is declared in the object of who takes it, and named after
// where it is declared, `temporal.features.activity.standalone.client.start`.

/** The client starts and controls the activity. */
object client extends Client:
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
 * and its answer settles it. The worker's stop is the worker's own action, `process.stop`:
 * nothing it records names the activity, so the activity's machines keep their state.
 */
object worker:
  val poll = action(process).on(activity).schema[PollActivityTaskQueueResponse]

  val respond = action(process)
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

/** The System's three deadlines, each armed by the start's input of its name. */
object deadline:
  val scheduleToClose = timer
  val scheduleToStart = timer
  val startToClose = timer

// A retry shows the client only the attempt count DescribeActivityExecution reports. The statuses
// observe one status field; whether a catalog tells them apart is left to the realization.
val attemptCount = Observation(on = activity, read = "attempt")

given Ok[Outcome] = Ok(Outcome.accepted)

// The bounds of the levels' Queries and the history record's, beside three and four (shared.Bounds).
val five = Limits(steps = 5, actions = 5, search = 65536)
val six = Limits(steps = 6, actions = 6, search = 262144)
val eight = Limits(steps = 8, actions = 8, search = 262144)

// ### The checked-in IR files of the standalone activity Models (umpire.irFile).

object exports:
  // The activity Model. Its cross-entity Query, stoppedWorkerStartsNothing, carries the composition
  // and its claim.
  val activityStandalone = irFile("activity-standalone")(
    StandaloneActivity,
    ActivityProduct,
    ActivityProduct.implements,
    ActivitySystem.implements,
    ActivitySystem.queries,
    StandaloneActivity.queries,
    ActivityRealization.standalone
  )

  // Its history record, the admission designs, and the shared task queue's providers it composes.
  // A composition no Query runs over is a root of its own.
  val activityStandaloneRecord = irFile("activity-standalone-record")(
    system.ActivityRecord.queries,
    system.TrustingActivityRecord.queries,
    shared.taskqueue.system.TaskQueueSystem.queries,
    shared.taskqueue.system.ForgetfulQueue.queries,
    shared.taskqueue.system.VolatileQueue.queries,
    shared.taskqueue.system.LossyMatchingQueue.queries,
    system.RecordOverQueue.queries,
    system.TrustingRecordOverQueue.queries,
    system.RecordOverMatching.queries,
    system.TrustingRecordOverMatching.queries,
    system.RecordOverLossyMatching.queries,
    system.RecordOverForgetful,
    system.RecordOverVolatile
  )

  // The held race a server is run through, and the realization that runs it. It is a Model of its
  // own, so the history record's Queries are the ones its checkers were given.
  val activityStandaloneRace = irFile("activity-standalone-race")(
    system.HeldDispatch.queries,
    ActivityRealization.heldDelivery,
    system.LostStartAnswer.queries,
    ActivityRealization.lostAdmissionResponse
  )
