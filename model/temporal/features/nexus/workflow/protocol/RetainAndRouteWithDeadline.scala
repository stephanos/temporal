package temporal
package features.nexus
package workflow
package protocol
import framework.*
import framework.Finite.given CanEqual[?, ?]

// The corrected design over the channel that redelivers once, with the deadline.
val _ = () => summon[CanEqual[framework.outcomes.Outcome, framework.outcomes.Outcome]]

object RetainAndRouteWithDeadline
    extends Derived(
      RetainAndRouteBoundedRetry
        .assuming(deadlineExpires)
        .extend(
          on(deadline.scheduleToClose) {
            where(RejectAfterClose.states.expirable) ~> RejectAfterClose.closing.effects.expire
          }
        )
    ):

  object properties:
    val retainAndRouteWithDeadlineProgress =
      RetainAndRouteWithDeadline.leadsTo("outcomeReachesOwner")(
        RejectAfterClose.states.isDone,
        RejectAfterClose.states.settled,
        within = 6,
        reporting,
        deliveryFair
      )

  object queries:
    val retainAndRouteWithDeadlineQueries =
      RejectAfterClose.queries.deadlineQueries(RetainAndRouteWithDeadline)
