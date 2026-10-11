package temporal
package features.activity
package mini
package product

import framework.*
import framework.Finite.given
import framework.outcomes.Outcome
import temporal.features.activity.Timeout
import temporal.capabilities.product.Pollable
import scala.annotation.unused

trait Dispatching extends Facet[State, Outcome, Fact], Phased[State, Phase]:
  object dispatching:
    object states:
      def waiting(s: State) = s.phase == Phase.scheduled

    object effects:
      def schedule(s: State, @unused scheduleToStart: Timeout, @unused startToClose: Timeout,
          @unused maxAttempts: MaxAttempts) = enter(s, Fact.statusScheduled)

    object rules extends Rules:
      on(client.start) {
        when(Phase.scheduled) ~> effects.schedule
      }
      on(activityWorker.poll) {
        when(Phase.scheduled) ~> enters(Phase.started)
      }

    object capabilities extends Capabilities:
      val pollable: Capability = Pollable(dispatch = activityWorker.poll)
