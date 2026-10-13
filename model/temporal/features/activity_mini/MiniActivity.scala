// The small activity Model, by subtraction from the standalone activity. It keeps one of each
// hard instance the real features have: recorded status phases, init/end, a verify and a find in
// the Product; two facets whose deadline guard reads the other's phase, their capabilities and a
// reasoned except waiver; three start inputs, maxAttempts bounded at two, cancellation, the
// worker's answers and the shared deadline timers in this signature; a Protocol refinement with a
// verify through it, UpTo[2] attempts, a shared scheduled status and a monitor; Retryable and
// Expirable with an overriding reason; one task queue composition and sync; a bounded lossy answer
// channel with a FailureModel, a NegativeControl, an environment assumption and a hole; restricted
// dispatch progress under assume.fair(worker); and a realization with seven evidence kinds, typed
// deadline presets and one satisfied and one reasoned inconclusive live expectation.
//
// The tree shows where each lives (model/README.md, "The layers"): a level folder's files sort by
// facet (FacetName) or design family (FamilyName), one machine per file;
// model/framework/export/testdata/layout/lamp is the template layout.
package temporal
package features.activity
package mini

import framework.dsl.*
import temporal.actors.client.Client
import temporal.features.activity.{Timeout, worker}

enum FacetName:
  case dispatching, completion, answering

// The design families its level folders group by besides the facets (fn-160 R7).
enum FamilyName:
  case answer, queue

enum MaxAttempts:
  case one, two

object MaxAttempts:
  type Bound = 2
  val bound: Bound = 2

val activity = Entity(key = "activityId")
val scheduleToStart = input[Timeout](default = Timeout.unset)
val startToClose = input[Timeout](default = Timeout.unset)
val maxAttempts = input[MaxAttempts](default = MaxAttempts.one)

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
