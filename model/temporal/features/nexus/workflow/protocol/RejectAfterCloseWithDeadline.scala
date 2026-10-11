package temporal
package features.nexus
package workflow
package protocol

import framework.*
import framework.Finite.given CanEqual[?, ?]

// The faulty policy that rejects after close, with the deadline. A deliberately wrong design.
val _ = () => summon[CanEqual[framework.outcomes.Outcome, framework.outcomes.Outcome]]

object RejectAfterCloseWithDeadline
    extends Derived(
      RejectAfterClose
        .assuming(retryFair, deadlineExpires)
        .rebind(
          handler.complete ~> (RejectAfterClose.closing.effects
            .deliver(Policy.rejectAfterClose, Redelivery.once, _, _))
        )
        .extend(
          on(deadline.scheduleToClose) {
            where(RejectAfterClose.states.expirable) ~> RejectAfterClose.closing.effects.expire
          }
        )
    ),
      NegativeControl:

  object properties:

    object liveness:
      val rejectAfterCloseWithDeadlineProgress =
        RejectAfterCloseWithDeadline.leadsTo("outcomeReachesOwner")(
          RejectAfterClose.states.isDone,
          RejectAfterClose.states.settled,
          within = 6,
          reporting,
          deliveryFair
        )

  object queries:
    val rejectAfterCloseWithDeadlineQueries =
      RejectAfterClose.queries.deadlineQueries(RejectAfterCloseWithDeadline)
