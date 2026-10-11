package temporal
package features.nexus
package workflow
package protocol

import framework.*
import framework.Finite.given CanEqual[?, ?]

// The faulty policy: after a reset the original run acknowledges the completion, and the successor
// never learns it. A deliberately wrong design.
val _ = () => summon[CanEqual[framework.outcomes.Outcome, framework.outcomes.Outcome]]

object AckByOriginal
    extends Derived(
      RejectAfterClose
        .assuming(retentionDurable)
        .rebind(
          handler.complete ~> (RejectAfterClose.closing.effects
            .deliver(Policy.ackByOriginal, Redelivery.untilAck, _, _))
        )
    ),
      NegativeControl:

  object properties:

    object liveness:
      val ackByOriginalProgress = AckByOriginal.leadsTo("outcomeReachesOwner")(
        RejectAfterClose.states.isDone,
        RejectAfterClose.states.settled,
        within = 6,
        reporting,
        deliveryFair
      )

  object queries:
    val ackByOriginalQueries = RejectAfterClose.queries.designQueries(AckByOriginal)
