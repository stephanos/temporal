// The standalone activity Product's Completion facet: a held attempt's completion by its worker or by activity ID.
package temporal
package features.activity
package standalone
package product

import framework.*
import framework.outcomes.{Outcome, Rejection}

trait Completion extends Facet[State, Outcome, Fact], Phased[State, Phase]:
  object completion:
    // Held completion must match the By-ID rows; By-ID keeps its notFound checks and its broader Live
    // completion.
    object rules extends Rules:
      on(activityWorker.respondCompleted) {
        when[Held] ~> enters(Phase.completed)
      }
      on(completionClient.respondCompletedByID) {
        when[Closed] ~> rejects(Rejection.notFound)
        when[Live] ~> enters(Phase.completed)
      }
