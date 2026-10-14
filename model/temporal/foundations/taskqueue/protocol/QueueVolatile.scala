package temporal
package foundations.taskqueue
package protocol

import framework.dsl.*
import framework.dsl.Finite.given

// A crash that wipes the tasks matching persisted.
val _ = () => summon[CanEqual[framework.dsl.outcomes.Outcome, framework.dsl.outcomes.Outcome]]

object QueueVolatile
    extends Derived(
      TaskQueueProtocol.rebind(fault.crash ~> TaskQueueProtocol.queueing.effects.volatileCrash)
    ),
      NegativeControl:

  object queries:
    val volatileQueueQueries = TaskQueueProtocol.queries.providerQueries(QueueVolatile)
