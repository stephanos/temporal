// The small activity Model, by subtraction from activity/standalone. Present hard instances:
// - product/Product.scala: recorded status phases, init/end, one verify and one find;
// - product/Dispatching.scala and Completion.scala: two facets; Completion's deadline guard
//   reads Dispatching's waiting phase;
// - the facets' capabilities: Pollable, Closable and one reasoned except waiver;
// - this signature: three start inputs, maxAttempts bounded at two, cancellation,
//   the worker's poll/completed/failed answers and the two shared deadline timers;
// - temporal/Bounds.scala: the named query bound (three); the find is a source witness.
package temporal
package features.activity
package mini

import framework.*
import temporal.actors.client.Client
import temporal.features.activity.{Timeout, worker}

enum FacetName:
  case dispatching, completion

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
