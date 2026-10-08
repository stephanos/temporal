// The standalone activity Model: one activity started directly through StartActivityExecution, with
// no workflow around it, grounded in chasm/lib/activity/statemachine.go. The product machine says
// what DescribeActivityExecution reports, the System how the server gets there. No history
// event is written, so every evidence line names an observation: a status read through
// DescribeActivityExecution or a result read through PollActivityExecution. Reset is deferred, like
// cancellation in the Nexus client Model, and the heartbeat timeout is not modeled.
//
// Update this Model independently of the implementation. When conformance fails, ask a human
// rather than fitting the Model to the code.
//
// The feature has two levels, each in a folder of its own, because different people read them
// (model/irgen/testdata/layout/lamp is the template):
//
//   - ../Activity.scala: the kind's types, the worker's actions, its timers and deadlines;
//   - this file: the form's types; the signature (the activity and its inputs; the client and its
//     actions, the worker's actions bound to the activity; and the bounds); and last exports, its
//     IR files;
//   - product/Product.scala: Product Phase, State and Fact; ActivityProduct, the product machine,
//     what a client reads;
//   - system/System.scala: System Phase, State, Fact, timer and composition types; ActivitySystem,
//     the System machine that refines it; ActivityWorker, the worker of its task queue; and
//     StandaloneActivity, the System with that worker;
//   - system/Record.scala: the history record of the activity, and its designs;
//   - system/WithTaskQueue.scala: the contract's designs composed with the shared task queue.
//   - system/Realization.scala: the executable System realizations of StandaloneActivity,
//     HeldDispatch and LostStartAnswer.
//
// A machine object reads its header (entity, init, end, evidence), then its sections in order:
// states, refinement, effects, monitors, rules, properties, capabilities and queries. A composition
// reads end, then states, syncs, properties, capabilities and queries.
package temporal
package features.activity
package standalone

import umpire.*
import product.ActivityProduct
import system.{ActivitySystem, StandaloneActivity}

// ### Signature

enum MaxAttempts derives Finite:
  case unlimited, one, two

object MaxAttempts:
  type Bound = 2
  val bound: Bound = 2

// Named by the id the client chose: every read carries it, so no run id or event id is needed.
val activity = Entity(key = "activityId")

// The start's inputs, which the deadline timers no longer collide with.
val scheduleToClose = input[Timeout]
val scheduleToStart = input[Timeout]
val startToClose = input[Timeout]
val startDelay = input[Timeout]
val maxAttempts = input[MaxAttempts]

// Who acts, and on what: each action is declared in the object of who takes it, and named after
// where it is declared, `temporal.features.activity.standalone.client.start`.

// The client starts and controls the activity.
object client extends Client:
  val start = action(this)
    .input(scheduleToClose)
    .input(scheduleToStart)
    .input(startToClose)
    .input(startDelay)
    .input(maxAttempts)
    .creates(activity)

  // Each public control is its own RPC. They share a result: on an activity that is over, each is
  // not found. The result text is metadata of each action, not a domain a state holds.
  val pause = action(this).on(activity).results("Delivery")
  val unpause = action(this).on(activity).results("Delivery")
  val requestCancel = action(this).on(activity).results("Delivery")
  val terminate = action(this).on(activity).results("Delivery")

// The kind's worker actions (../Activity.scala) bound to this activity. The timers and deadlines
// are the kind's.
object worker:
  val poll = temporal.features.activity.worker.poll.on(activity)
  val respondCompleted = temporal.features.activity.worker.respondCompleted.on(activity)
  val respondFailed = temporal.features.activity.worker.respondFailed.on(activity)
  val respondCanceled = temporal.features.activity.worker.respondCanceled.on(activity)

// A retry shows the client only the attempt count DescribeActivityExecution reports. The statuses
// observe one status field; whether a catalog tells them apart is left to the realization.
val attemptCount = Observation(on = activity, read = "attempt")

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
    ActivityProduct.capabilities,
    ActivitySystem.capabilities,
    ActivitySystem.queries,
    system.TimeoutRetry.queries,
    StandaloneActivity.queries,
    system.Standalone,
    system.RetryAfterTimeout
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
    system.HeldDelivery,
    system.LostStartAnswer.queries,
    system.LostAdmissionResponse
  )
