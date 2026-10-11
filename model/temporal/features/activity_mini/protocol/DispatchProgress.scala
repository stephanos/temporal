package temporal
package features.activity
package mini
package protocol

import framework.*
import framework.Finite.given
import Bounds.three

val _ = () => summon[CanEqual[Phase, Phase]]

val workerFair = assume.fair(activityWorker.poll)

object DispatchProgress extends Derived(ActivityProtocol.restrict(activityWorker.poll)):
  object states:
    def waiting(s: State): Boolean = s.phase == Phase.scheduled
    def started(s: State): Boolean = s.phase == Phase.started
  object properties:
    val workerStarts = DispatchProgress.leadsTo("workerStarts")(
      states.waiting, states.started, within = Bounds.dispatchSteps, workerFair)
  object queries:
    val starts = witness(activityWorker.poll).records(Fact.statusStarted).limits(three)
