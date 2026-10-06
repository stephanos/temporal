// The activity kind: what its forms share, the deadline types, the worker's answer and its actions
// on an activity, the timers and the deadlines. Each form owns its entity, its other declarations
// and its exports, and binds the worker's actions to its entity (standalone/Standalone.scala).
package temporal
package features.activity

import umpire.*
import shared.worker.worker as process
import io.temporal.api.workflowservice.v1.*

// ### Types

// Whether the start request sets a deadline.
enum Timeout derives Finite:
  case unset, expires

// The worker's answer; failed(retryable) is two classes, like ApplicationFailure's flag.
enum AttemptResult derives Finite:
  case completed
  case failed(retryable: Boolean)
  case canceled

// Which deadline fired.
enum TimeoutType derives Finite:
  case scheduleToClose, scheduleToStart, startToClose

// ### Signature

// The worker's answer.
val result = input[AttemptResult]

// The shared worker's actions on an activity: its poll receives the task for the current attempt,
// and its answer settles it. The worker's stop is the worker's own action, `process.stop`:
// nothing it records names the activity, so the activity's machines keep their state. Each form
// binds them to its activity, `worker.poll.on(activity)`.
object worker:
  val poll = action(process).schema[PollActivityTaskQueueResponse]

  val respond = action(process)
    .input(result)
    .schema[RespondActivityTaskCompletedRequest]
    .schema[RespondActivityTaskFailedRequest]
    .schema[RespondActivityTaskCanceledRequest]
    .example(AttemptResult.failed(false), "ApplicationFailureNonRetryable")
    .example(AttemptResult.failed(true), "ApplicationFailureRetryable")

// One of the activity's deadlines firing, as the product machine sees it, and the backoff.
object timers:
  val timeout = timer
  val backoff = timer

// The System's three deadlines, each armed by the start's input of its name.
object deadline:
  val scheduleToClose = timer
  val scheduleToStart = timer
  val startToClose = timer
