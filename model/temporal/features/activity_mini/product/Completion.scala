package temporal
package features.activity
package mini
package product

import framework.*
import framework.Finite.given
import framework.outcomes.{Outcome, Rejection}
import temporal.features.activity.{Failure, deadline}
import temporal.capabilities.product.Closable

trait Completion extends Facet[State, Outcome, Fact], Phased[State, Phase]:
  this: Dispatching =>
  object completion:
    object effects:
      val retryScheduled = choice
      val retryExhausted = choice
      def serviceRetry(s: State) = choose(
        retryScheduled -> enters(Phase.scheduled)(s),
        retryExhausted -> enters(Phase.failed)(s)
      )

    object rules extends Rules:
      on(client.cancel) {
        when[Closed] ~> rejects(Rejection.notFound)
        when(Phase.scheduled) ~> enters(Phase.canceled)
        when(Phase.started) ~> enters(Phase.cancelRequested)
        when(Phase.cancelRequested) ~> rejects(Rejection.failedPrecondition)
          .because("cancellation already requested")
      }
      on(activityWorker.respondCompleted) {
        when[Held] ~> enters(Phase.completed)
      }
      on(activityWorker.respondFailed(Failure.fatal)) {
        when[Held] ~> enters(Phase.failed)
      }
      on(activityWorker.respondFailed(Failure.retryable)) {
        when(Phase.started) ~> effects.serviceRetry
        when(Phase.cancelRequested) ~> enters(Phase.canceled)
      }
      on(deadline.scheduleToStart) {
        where(dispatching.states.waiting) ~> enters(Phase.timedOut)
      }
      on(deadline.startToClose) {
        when[Held] ~> enters(Phase.timedOut)
      }

    object capabilities extends Capabilities:
      val closable: Capability = Closable(rejected = Outcome.rejected(Rejection.notFound))
      except(Closable.closedIsRejectedUniformly[State, Phase, Outcome],
        because = "The Product omits late worker RPC replies; only client cancellation models NotFound after closure.")
