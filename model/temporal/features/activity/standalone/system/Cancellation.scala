// Standalone activity cancellation request and worker settlement.
package temporal
package features.activity
package standalone
package system

import framework.*
import Bounds.four

object Cancellation extends Derived(ActivitySystem.rebind()):
  object properties:
    val cancelRequestedWhileStarted =
      property when client.requestCancel holds { s =>
        s.state.phase == Phase.cancelRequested && s.records(Fact.statusCancelRequested)
      }

    val canceledByWorker =
      property when worker.respondCanceled holds { s =>
        s.state.phase == Phase.canceled && s.records(Fact.statusCanceled)
      }

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
