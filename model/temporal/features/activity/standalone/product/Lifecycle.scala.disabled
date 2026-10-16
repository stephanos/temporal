// The standalone activity Product's Lifecycle facet: the terminate; a closed activity stays closed.
package temporal
package features.activity
package standalone
package product

import framework.*
import framework.outcomes.{Outcome, Rejection}
import temporal.capabilities.product.*

trait Lifecycle extends Facet[State, Outcome, Fact], Phased[State, Phase]:

  object lifecycle:

    object rules extends Rules:
      on(client.terminate) {
        when[Closed] ~> rejects(Rejection.notFound)
        when(
          Phase.scheduled,
          Phase.started,
          Phase.paused,
          Phase.cancelRequested
        ) ~> enters(Phase.terminated)
      }

    object capabilities extends Capabilities:
      val closable: Capability = Closable(
        rejected = Outcome.rejected(Rejection.notFound)
      )
