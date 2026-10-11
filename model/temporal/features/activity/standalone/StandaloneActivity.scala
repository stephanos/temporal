// The standalone activity Model: one activity started directly through StartActivityExecution, with
// no workflow around it, grounded in chasm/lib/activity/statemachine.go. The product machine says
// what DescribeActivityExecution reports, the System how the server gets there. No history
// event is written, so every evidence line names an observation: a status read through
// DescribeActivityExecution or a result read through PollActivityExecution. A reset of a held attempt
// is deferred until the attempt ends, like a pause request.
//
// Update this Model independently of the implementation. When conformance fails, ask a human
// rather than fitting the Model to the code.
//
// The feature has two levels, each in a folder of its own, because different people read them
// (model/irgen/testdata/layout/lamp is the template):
//
//   - ../Activity.scala: the kind's types, the worker's actions, its timers and deadlines;
//   - this file: the form's types; the signature (the activity and its inputs; the client and its
//     actions, the worker's actions bound to the activity);
//   - product/Product.scala: Product Phase, State and Fact; ActivityProduct, the product machine,
//     what a client reads;
//   - system/System.scala: System Phase, Dispatch, State and Fact; ActivityProtocol, the lifecycle
//     machine that refines the Product;
//   - system/<Facet>.scala: the System's facets (fn-161), each the actions it owns, their rules and
//     effects, its capabilities and its claims: Dispatching, Pausing, Cancellation, Lifecycle,
//     Reset, Heartbeat, Completion, Retry and Timeouts;
//   - system/Dispatch.scala: the dispatch protocol, ActivityRecord and TrustingActivityRecord;
//   - system/DispatchRaces.scala: HeldDispatch and LostStartAnswer;
//   - system/DispatchWithTaskQueue.scala: the dispatch designs composed with the shared task queue;
//   - system/DispatchWithWorker.scala: StandaloneActivityState; ActivityWorker, the worker of its
//     task queue; and StandaloneActivity, the System with that worker;
//   - system/Realization.scala: the executable System realizations of StandaloneActivity,
//     HeldDispatch and LostStartAnswer, each naming the Queries it runs.
//
// A machine object reads its header (entity, init, end, evidence), then its sections in order:
// states, refinement, effects, monitors, rules, properties, capabilities and queries. A composition
// reads end, then states, syncs, properties, capabilities and queries.
package temporal
package features.activity
package standalone

import temporal.actors.client.Client
import framework.*

// The standalone activity's facet vocabulary (fn-161), shared by the Product and the System; the
// admission designs and the lost-answer race have their own.
enum FacetName:
  case dispatching, pausing, cancellation, lifecycle, reset, heartbeat, completion, retry, timeouts
  case admission, heldAdmission, responseLoss

// ### Signature

enum MaxAttempts derives Finite:
  case unlimited, one, two

object MaxAttempts:
  type Bound = 2
  val bound: Bound = 2

// Whether a reset leaves a paused activity paused: ResetActivityExecutionRequest.keep_paused.
enum ResetPause derives Finite:
  case resume, keepPaused

// Named by the id the client chose: every read carries it, so no run id or event id is needed.
val activity = Entity(key = "activityId")

// The start's inputs, which the deadline timers no longer collide with.
val scheduleToClose = input[Timeout]
val scheduleToStart = input[Timeout]
val startToClose = input[Timeout]
val heartbeat = input[Timeout]
val startDelay = input[Timeout]
val maxAttempts = input[MaxAttempts]
val pausing = input[ResetPause]

val pauseAlreadyRequestedReason = "pause already requested (chasm/lib/activity/model/model.go:232)"

val pausedOrCancelPending =
  "already paused or cancellation pending (chasm/lib/activity/model/model.go:232)"

val pausedOrControlPending =
  "already paused, or a cancellation or reset pending (chasm/lib/activity/model/model.go:232)"

val activityNotPaused = "activity is not paused (chasm/lib/activity/model/model.go:251)"

val cancellationAlreadyRequested =
  "cancellation already requested (chasm/lib/activity/model/model.go:201-202)"

val resetWithPendingCancellation =
  "cannot reset an activity with a pending cancellation (operator_commands.go:458)"

val resetAlreadyPending = "a reset is already pending (operator_commands.go:460-464)"

val resetWithPendingReset =
  "cannot reset an activity with a pending reset (operator_commands.go:460-464)"

val cancellationNotRequested =
  "cancellation was not requested (chasm/lib/activity/model/model.go:171)"

val cancellationNotRequestedByID = "cancellation was not requested"

// Who acts, and on what: each action is declared in the object of who takes it, and named after
// where it is declared, `temporal.features.activity.standalone.client.start`.

// The client starts and controls the activity.
object client extends Client:
  val start = action(this)
    .input(scheduleToClose)
    .input(scheduleToStart)
    .input(startToClose)
    .input(heartbeat)
    .input(startDelay)
    .input(maxAttempts)
    .creates(activity)

  // Each public control is its own RPC. They share a result: on an activity that is over, each is
  // not found. The result text is metadata of each action, not a domain a state holds.
  val pause = action(this).on(activity).results("Delivery")
  val unpause = action(this).on(activity).results("Delivery")
  val requestCancel = action(this).on(activity).results("Delivery")
  val terminate = action(this).on(activity).results("Delivery")
  val reset = action(this).input(pausing).on(activity).results("Delivery")

// A client that settles an activity asynchronously by its activity id, without the task token and
// independent of the worker process: the Java SDK's ActivityCompletionClient, the Go SDK's
// client.CompleteActivityByID.
object completionClient extends Client:
  val respondCompletedByID = action(this).on(activity)
  val respondFailedByID = action(this)
    .input(failure)
    .example(Failure.fatal, fatalFailureExample)
    .example(Failure.retryable, retryableFailureExample)
    .on(activity)
  val respondCanceledByID = action(this).on(activity)

// The activity kind's worker actions (../Activity.scala, `worker`) bound to this form's `activity`.
// The timers and deadlines are the kind's.
object activityWorker:
  val poll = worker.poll.on(activity)
  val heartbeat = worker.heartbeat.on(activity)
  val respondCompleted = worker.respondCompleted.on(activity)
  val respondFailed = worker.respondFailed.on(activity)
  val respondCanceled = worker.respondCanceled.on(activity)

// A retry shows the client only the attempt count DescribeActivityExecution reports. The statuses
// observe one status field; whether a catalog tells them apart is left to the realization.
val attemptCount = Observation(on = activity)
val heartbeatDetails = Observation(on = activity)
