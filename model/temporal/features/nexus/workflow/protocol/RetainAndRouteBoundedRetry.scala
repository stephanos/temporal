package temporal
package features.nexus
package workflow
package protocol

import framework.*
import framework.Finite.given CanEqual[?, ?]

// The corrected design over the channel that redelivers once.
val _ = () => summon[CanEqual[framework.outcomes.Outcome, framework.outcomes.Outcome]]

object RetainAndRouteBoundedRetry
    extends Derived(
      RetainAndRoute
        .assuming(retryFair)
        .rebind(
          handler.complete ~> (RejectAfterClose.closing.effects
            .deliver(Policy.retainAndRoute, Redelivery.once, _, _))
        )
    ):

  object properties:
    val retainAndRouteBoundedRetryProgress =
      RetainAndRouteBoundedRetry.leadsTo("outcomeReachesOwner")(
        RejectAfterClose.states.isDone,
        RejectAfterClose.states.settled,
        within = 6,
        reporting,
        deliveryFair
      )

    // Over the channel that redelivers once, at most the one redelivery comes before the reset.
    val retainedReachesOwnerBoundedRetry =
      RetainAndRouteBoundedRetry.leadsTo("retainedReachesOwner")(
        RejectAfterClose.properties.awaitingOwner,
        RejectAfterClose.properties.ownerKnowsOutcome,
        within = 2,
        deliveryFair,
        recovery
      )

  object queries:
    val retainAndRouteBoundedRetryQueries =
      RejectAfterClose.queries.safetyQueries(RetainAndRouteBoundedRetry)

// The three policies with a schedule-to-close deadline, over the channel that redelivers once.
