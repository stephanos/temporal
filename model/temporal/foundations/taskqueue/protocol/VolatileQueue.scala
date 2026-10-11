package temporal
package foundations.taskqueue
package protocol

import framework.*
import framework.Finite.given

// A crash that wipes the tasks matching persisted.
val _ = () => summon[CanEqual[framework.outcomes.Outcome, framework.outcomes.Outcome]]

object VolatileQueue
    extends Derived(
      TaskQueueProtocol.rebind(fault.crash ~> TaskQueueProtocol.queueing.effects.volatileCrash)
    ),
      NegativeControl:

  object queries:
    val volatileQueueQueries = TaskQueueProtocol.queries.providerQueries(VolatileQueue)
