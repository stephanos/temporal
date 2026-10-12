package temporal
package foundations.taskqueue
package protocol

import framework.*
import framework.Finite.given

// A crash that loses the message history dropped on invoking AddActivityTask.
val _ = () => summon[CanEqual[framework.outcomes.Outcome, framework.outcomes.Outcome]]

object QueueForgetful
    extends Derived(
      TaskQueueProtocol.rebind(fault.crash ~> TaskQueueProtocol.queueing.effects.forgetfulCrash)
    ),
      NegativeControl:

  object queries:
    val forgetfulQueueQueries =
      TaskQueueProtocol.queries.providerQueries(QueueForgetful)
