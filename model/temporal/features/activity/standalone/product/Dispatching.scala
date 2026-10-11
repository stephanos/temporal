// The standalone activity Product's Dispatching facet: the start and the worker's poll.
package temporal
package features.activity
package standalone
package product

import framework.*
import framework.outcomes.Outcome
import scala.annotation.unused
import temporal.capabilities.*

trait Dispatching extends Facet[State, Outcome, Fact], Phased[State, Phase]:
  object dispatching:
    object effects:
      def schedule(
          s: State,
          @unused scheduleToClose: Timeout,
          @unused scheduleToStart: Timeout,
          @unused startToClose: Timeout,
          @unused heartbeat: Timeout,
          @unused startDelay: Timeout,
          @unused maxAttempts: MaxAttempts
      ) = enter(s, Fact.statusScheduled)

    object rules extends Rules:
      on(client.start) {
        when(Phase.scheduled) ~> effects.schedule
      }
      on(activityWorker.poll) {
        when(Phase.scheduled) ~> enters(Phase.started)
      }

    object capabilities extends Capabilities:
      val pollable: Capability = Pollable(dispatch = activityWorker.poll)
