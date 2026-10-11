// Standalone activity cancellation request and worker settlement.
package temporal
package features.activity
package standalone
package system

import framework.*
import framework.Finite.given
import Bounds.four

val _ = () => summon[CanEqual[framework.outcomes.Outcome, framework.outcomes.Outcome]]

object Cancellation extends Derived(ActivitySystem.rebind()):

  object properties:
    val cancelRequestedWhileStarted =
      property when client.requestCancel holds { s => // format: off
        s.state.phase == Phase.cancelRequested && s.records(Fact.statusCancelRequested)
      } // format: on

    val canceledByWorker =
      property when worker.respondCanceled holds { s => // format: off
        s.state.phase == Phase.canceled && s.records(Fact.statusCanceled)
      } // format: on

  object queries:
    val cancelRequestedThenCanceled = scenario.actions(
      client.start(),
      worker.poll,
      client.requestCancel,
      worker.respondCanceled
    )

    val cancel =
      query find properties.canceledByWorker in cancelRequestedThenCanceled limits four
    // Asks the one Property no functional Query asks, over the path that takes a cancel request.
    val cancelRequest =
      query find properties.cancelRequestedWhileStarted in cancelRequestedThenCanceled limits
        four
