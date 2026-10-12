// The small activity Model, by subtraction from activity/standalone. Present hard instances:
// - product/Product.scala: recorded status phases, init/end, one verify and one find;
// - product/Dispatching.scala and Completion.scala: two facets; Completion's deadline guard
//   reads Dispatching's waiting phase;
// - the facets' capabilities: Pollable, Closable and one reasoned except waiver;
// - this signature: three start inputs, maxAttempts bounded at two, cancellation,
//   the worker's poll/completed/failed answers and the two shared deadline timers;
// - temporal/Bounds.scala: the named query bound (three); the find is a source witness.
// - protocol/Protocol.scala: Product refinement and verify through it, UpTo[2] attempts,
//   backingOff's shared scheduled status, and the attemptsNeverRewind monitor;
// - protocol/Dispatching.scala and Completion.scala: dispatch/retry/deadline facets,
//   Retryable and Expirable, with retryBacksOff overriding and its reason;
// - protocol/Queue.scala and ScheduledQueue.scala: one task queue composition and one sync,
//   scheduleToStart shared by start and taskQueue.schedule;
// - protocol/Faults.scala, Answering.scala, AnswerDispatch.scala and TrustingAnswers.scala:
//   one bounded lossy worker-answer channel, delayed-answer FailureModel, stale-answer
//   NegativeControl find and verify, environment assumption, and cancelDuringBackoff hole;
// - protocol/DispatchProgress.scala: restricted dispatch leadsTo under assume.fair(worker),
//   with the step bound in temporal/Bounds.scala;
// - realization/MiniActivityRun.scala and Kit.scala: seven fact evidence kinds, typed deadline
//   presets preserving maxAttempts, one satisfied and one reasoned inconclusive live expectation.
package temporal
package features.activity
package mini

import framework.*
import temporal.actors.client.Client
import temporal.features.activity.{Timeout, worker}

enum FacetName:
  case dispatching, completion, answering

enum MaxAttempts:
  case one, two

object MaxAttempts:
  type Bound = 2
  val bound: Bound = 2

val activity = Entity(key = "activityId")
val scheduleToStart = input[Timeout]
val startToClose = input[Timeout]
val maxAttempts = input[MaxAttempts]

object client extends Client:
  val start = action(this)
    .input(scheduleToStart)
    .input(startToClose)
    .input(maxAttempts)
    .creates(activity)
  val cancel = action(this).on(activity).results("Delivery")

object activityWorker:
  val poll = worker.poll.on(activity)
  val respondCompleted = worker.respondCompleted.on(activity)
  val respondFailed = worker.respondFailed.on(activity)
