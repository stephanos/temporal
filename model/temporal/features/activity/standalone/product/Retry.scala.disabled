// The standalone activity Product's Retry facet: a held attempt's failure, retried, failed or settled by a pending reset.
package temporal
package features.activity
package standalone
package product

import framework.*
import framework.outcomes.{Outcome, Rejection}

trait Retry extends Facet[State, Outcome, Fact], Phased[State, Phase]:

  object retry:

    object effects:
      val retryScheduled = choice
      val retryExhausted = choice
      val retryPaused = choice
      def serviceRetry(s: State) = choose(
        retryScheduled -> enters(Phase.scheduled)(s),
        retryPaused -> enters(Phase.paused)(s),
        retryExhausted -> enters(Phase.failed)(s)
      )
      // A held attempt's failure ends the activity, or applies a pending reset whatever the
      // failure's retryability: the activity is scheduled again, or paused where a pause was kept.
      val failureFatal = choice
      val resetApplied = choice
      val resetAppliedPaused = choice
      def failOrReset(s: State) = choose(
        failureFatal -> enters(Phase.failed)(s),
        resetApplied -> enters(Phase.scheduled)(s),
        resetAppliedPaused -> enters(Phase.paused)(s)
      )

    // A retryable failure is retried, or canceled under a cancel request. Held failure, reset and
    // retry settlement must match the By-ID rows; By-ID keeps its notFound checks.
    object rules extends Rules:
      on(activityWorker.respondFailed(Failure.fatal)) {
        when(Phase.started) ~> effects.failOrReset
        when(Phase.cancelRequested) ~> enters(Phase.failed)
      }
      on(activityWorker.respondFailed(Failure.retryable)) {
        when(Phase.started) ~> effects.serviceRetry
        when(Phase.cancelRequested) ~> enters(Phase.canceled)
      }
      on(completionClient.respondFailedByID) {
        when[Closed] ~> rejects(Rejection.notFound)
        when(Phase.scheduled, Phase.paused) ~> rejects(Rejection.notFound)
      }
      on(completionClient.respondFailedByID(Failure.fatal)) {
        when(Phase.started) ~> effects.failOrReset
        when(Phase.cancelRequested) ~> enters(Phase.failed)
      }
      on(completionClient.respondFailedByID(Failure.retryable)) {
        when(Phase.started) ~> effects.serviceRetry
        when(Phase.cancelRequested) ~> enters(Phase.canceled)
      }
