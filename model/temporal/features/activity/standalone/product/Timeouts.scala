// The standalone activity Product's Timeouts facet: a deadline that ends a live activity.
package temporal
package features.activity
package standalone
package product

import framework.*
import framework.outcomes.Outcome

trait Timeouts extends Facet[State, Outcome, Fact], Phased[State, Phase]:

  object timeouts:

    object rules extends Rules:
      on(timers.timeout) {
        when(Phase.scheduled, Phase.started, Phase.paused, Phase.cancelRequested) ~> enters(
          Phase.timedOut
        )
      }
