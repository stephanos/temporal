package temporal
package features.activity
package standalone
package protocol

import framework.*
import framework.Finite.given
import actors.worker.worker as process

val _ = () => summon[CanEqual[framework.outcomes.Outcome, framework.outcomes.Outcome]]

object ServingWorker extends Derived(actors.worker.Polling.restrict(process.serve))

// ### The passing design: every dispatchable attempt starts within one step.
