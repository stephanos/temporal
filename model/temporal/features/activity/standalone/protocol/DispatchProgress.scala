// A bounded progress promise across the activity and its task queue's worker (fn-150): a scheduled
// attempt that may be dispatched now, with its worker polling, is started within one composed step.
// It is a model check of the composition's reachable graph, not a live Case: no realization runs it.
//
// The bound is the graph's, under the restrictions this design states. The activity binds only its
// start, the worker's poll and its completion, so from a scheduled attempt no pause, cancel, reset,
// timer or By-ID answer can take a step; and the worker only serves, so it never stops. The one
// composed step left from such a state is the sync of the poll with the worker's serve, which starts
// the attempt: one step, counted once though it advances both members. Fairness makes no bound: a
// weakly fair sync that some other step could precede would still miss a deadline of one, which is
// why the design rules those steps out rather than assuming them away.
//
// The worker that may stop is the missing prerequisite. Over it the same claim fails: the worker
// stops, the activity's poll is still enabled on its own, but the sync needs the worker's serve, so
// the composed state has no step and the claim reports a deadlock (and a missed deadline) whatever the
// fairness assumption says. The cross-entity safety promise, that only a polling worker starts an
// attempt, holds over both.
//
// Read top to bottom: the members, then the passing composition and the one derived from it.
package temporal
package features.activity
package standalone
package protocol

import framework.*
import framework.Finite.given
import Bounds.three
import actors.worker.{worker as process, Phase as WorkerPhase}

val _ = () => summon[CanEqual[framework.outcomes.Outcome, framework.outcomes.Outcome]]
// The dispatch sync of each design, weakly fair: named in its claim's result, and no part of the
// bound.
val servedDispatchRuns = assume.fair(ServedDispatch.synced(_.activity -> activityWorker.poll))
val unservedDispatchRuns = assume.fair(UnservedDispatch.synced(_.activity -> activityWorker.poll))

// ### The members: the activity as far as dispatch, and a worker that never stops.

object DispatchingActivity
    extends Derived(ActivityProtocol.restrict(client.start, activityWorker.poll, activityWorker.respondCompleted))

object ServingWorker extends Derived(actors.worker.Polling.restrict(process.serve))

// ### The passing design: every dispatchable attempt starts within one step.

object ServedDispatch
    extends Composition[StandaloneActivityState](
      _.activity -> DispatchingActivity,
      _.worker -> ServingWorker
    ),
      Phased[StandaloneActivityState, Phase](_.activity.phase):

  object states:
    // The source reads both members: an attempt the worker may take now, while the worker polls.
    def dispatchable(s: State): Boolean = // format: off
      s.activity.phase == Phase.scheduled && s.activity.dispatch == Dispatch.now && s.worker.phase == WorkerPhase.polling // scalafix:ok DisableSyntax.==; strictEquality checks native comparisons

    def started(s: State): Boolean = s.activity.phase == Phase.started // scalafix:ok DisableSyntax.==; strictEquality checks native comparisons
    // format: on

  object syncs extends Syncs:
    sync(_.activity -> activityWorker.poll, _.worker -> process.serve)

  object properties:
    // The cross-entity safety promise: only a polling worker starts an attempt.
    def startedByPollingWorker(c: Composition[State]) = c
      .property("startedByPollingWorker")
      .whenAction(c.synced(_.activity -> activityWorker.poll)) // format: off
      .holds(_.state.worker.phase == WorkerPhase.polling) // scalafix:ok DisableSyntax.==; strictEquality checks native comparisons
      // format: on

    // The bounded progress promise, over both members.
    val startsWithinOneStep = ServedDispatch.leadsTo("startsWithinOneStep")(
      states.dispatchable,
      states.started,
      within = 1,
      servedDispatchRuns
    )

  object queries:
    val anyDispatch = scenario.free
    val onlyPollingWorkersStart =
      query verify properties.startedByPollingWorker(ServedDispatch) in anyDispatch limits three

// ### The missing prerequisite: a worker that may stop, and never resumes. The same claim, under the
// same kind of fairness, is refuted by a deadlock: a stopped worker leaves the sync disabled.

object UnservedDispatch
    extends DerivedComposition(ServedDispatch.withMember(_.worker -> ActivityWorker)),
      NegativeControl:

  object properties:
    val startsWithinOneStep = UnservedDispatch.leadsTo("startsWithinOneStep")(
      ServedDispatch.states.dispatchable,
      ServedDispatch.states.started,
      within = 1,
      unservedDispatchRuns
    )

  object queries:
    val anyDispatch = scenario.free
    val stoppingWorkersStartNothing =
      query verify ServedDispatch.properties.startedByPollingWorker(UnservedDispatch) in
        anyDispatch limits three
