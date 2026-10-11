package temporal
package features.nexus
package workflow
package protocol
import framework.*
import framework.Finite.given CanEqual[?, ?]

// The faulty policy the original run acknowledges, with the deadline. A deliberately wrong design.
val _ = () => summon[CanEqual[framework.outcomes.Outcome, framework.outcomes.Outcome]]

object AckByOriginalWithDeadline
    extends Derived(
      AckByOriginal
        .assuming(retryFair, deadlineExpires)
        .rebind(
          handler.complete ~> (RejectAfterClose.closing.effects
            .deliver(Policy.ackByOriginal, Redelivery.once, _, _))
        )
        .extend(
          on(deadline.scheduleToClose) {
            where(RejectAfterClose.states.expirable) ~> RejectAfterClose.closing.effects.expire
          }
        )
    ),
      NegativeControl:

  object properties:
    val ackByOriginalWithDeadlineProgress =
      AckByOriginalWithDeadline.leadsTo("outcomeReachesOwner")(
        RejectAfterClose.states.isDone,
        RejectAfterClose.states.settled,
        within = 6,
        reporting,
        deliveryFair
      )

  object queries:
    val ackByOriginalWithDeadlineQueries =
      RejectAfterClose.queries.deadlineQueries(AckByOriginalWithDeadline)
