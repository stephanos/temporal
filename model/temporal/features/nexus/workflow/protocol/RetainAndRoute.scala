package temporal
package features.nexus
package workflow
package protocol
import framework.*
import framework.Finite.given CanEqual[?, ?]

// The corrected policy: retain at the operation and route to the current owner.
val _ = () => summon[CanEqual[framework.outcomes.Outcome, framework.outcomes.Outcome]]

object RetainAndRoute
    extends Derived(
      RejectAfterClose
        .assuming(retentionDurable)
        .rebind(
          handler.complete ~> (RejectAfterClose.closing.effects
            .deliver(Policy.retainAndRoute, Redelivery.untilAck, _, _))
        )
    ):

  object properties:
    val retainAndRouteProgress = RetainAndRoute.leadsTo("outcomeReachesOwner")(
      RejectAfterClose.states.isDone,
      RejectAfterClose.states.settled,
      within = 6,
      reporting,
      deliveryFair
    )

    // A retained outcome reaches an owner only through a reset. The claim is made twice over the
    // channel that retries until acknowledged: under the recovery assumption, and without it.

    val retainedReachesOwner = RetainAndRoute.leadsTo("retainedReachesOwner")(
      RejectAfterClose.properties.awaitingOwner,
      RejectAfterClose.properties.ownerKnowsOutcome,
      within = 2,
      deliveryFair,
      recovery
    )

    val retainedWaitsWithoutRecovery = RetainAndRoute.leadsTo("retainedWaitsWithoutRecovery")(
      RejectAfterClose.properties.awaitingOwner,
      RejectAfterClose.properties.ownerKnowsOutcome,
      within = 2,
      deliveryFair
    )

  object queries:
    val retainAndRouteQueries = RejectAfterClose.queries.designQueries(RetainAndRoute)
