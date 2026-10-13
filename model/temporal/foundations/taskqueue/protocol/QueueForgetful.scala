package temporal
package foundations.taskqueue
package protocol

import framework.dsl.*
import framework.dsl.Finite.given

// A crash that loses the message history dropped on invoking AddActivityTask.
val _ = () => summon[CanEqual[framework.dsl.outcomes.Outcome, framework.dsl.outcomes.Outcome]]

object QueueForgetful
    extends Derived(
      TaskQueueProtocol.rebind(fault.crash ~> TaskQueueProtocol.queueing.effects.forgetfulCrash)
    ),
      NegativeControl:

  object queries:
    val forgetfulQueueQueries =
      TaskQueueProtocol.queries.providerQueries(QueueForgetful)
