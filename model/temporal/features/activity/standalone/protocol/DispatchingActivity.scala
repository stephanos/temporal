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

val _ = () => summon[CanEqual[framework.outcomes.Outcome, framework.outcomes.Outcome]]
// The dispatch sync of each design, weakly fair: named in its claim's result, and no part of the
// bound.
val servedDispatchRuns = assume.fair(ServedDispatch.synced(_.activity -> activityWorker.poll))
val unservedDispatchRuns = assume.fair(UnservedDispatch.synced(_.activity -> activityWorker.poll))

// ### The members: the activity as far as dispatch, and a worker that never stops.

object DispatchingActivity
    extends Derived(ActivityProtocol.restrict(client.start, activityWorker.poll, activityWorker.respondCompleted))
