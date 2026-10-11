// The standalone activity System composed with its task queue's worker.
package temporal
package features.activity
package standalone
package protocol

import framework.*
import framework.Finite.given
import actors.worker.{worker as process, State as WorkerState}

val _ = () => summon[CanEqual[framework.outcomes.Outcome, framework.outcomes.Outcome]]
// The System machine and its worker, as the standalone activity composition holds them.
final case class StandaloneActivityState(activity: State, worker: WorkerState)

// ### The worker of the activity's task queue, as the activity sees it: its stop and its serving.

object ActivityWorker extends Derived(actors.worker.Polling.restrict(process.stop, process.serve))

// ### With the worker of its task queue, the stop is the worker's own phase change and every
// attempt start is the worker serving, so an attempt has a row only while the worker polls.
