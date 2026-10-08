// The activity kind: what its forms share, the deadline types, the worker's actions
// on an activity, the timers and the deadlines. Each form owns its entity, its other declarations
// and its exports, and binds the worker's actions to its entity (standalone/Standalone.scala).
package temporal
package features.activity

import umpire.*
import shared.worker.worker as process

// ### Types

// Whether the start request sets a deadline.
enum Timeout derives Finite:
  case unset, expires

// Whether a failed attempt may be retried.
enum Failure derives Finite:
  case fatal, retryable

// Which deadline fired.
enum TimeoutType derives Finite:
  case scheduleToClose, scheduleToStart, startToClose, heartbeat

// ### Signature

// A failed attempt's retry policy.
val failure = input[Failure]

// The shared worker's actions on an activity: its poll receives the task for the current attempt,
// and each answer is one RPC. The worker's stop is the worker's own action, `process.stop`:
// nothing it records names the activity, so the activity's machines keep their state. Each form
// binds them to its activity, as standalone/Standalone.scala does.
object worker:
  val poll = action(process)
  val heartbeat = action("recordHeartbeat", process)
  val respondCompleted = action(process)
  val respondFailed = action(process)
    .input(failure)
    .example(Failure.fatal, "ApplicationFailureNonRetryable")
    .example(Failure.retryable, "ApplicationFailureRetryable")
  val respondCanceled = action(process)

// One of the activity's deadlines firing, as the product machine sees it, and its dispatch delays.
object timers:
  val timeout = timer
  val startDelay = timer
  val backoff = timer

// The System's deadlines, each armed by the start's input of its name.
object deadline:
  val scheduleToClose = timer
  val scheduleToStart = timer
  val startToClose = timer
  val heartbeat = timer
