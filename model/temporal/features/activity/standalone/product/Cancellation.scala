// The standalone activity Product's Cancellation facet: the cancel request and its settlement by the worker or by activity ID.
package temporal
package features.activity
package standalone
package product

import framework.*
import framework.outcomes.{Outcome, Rejection}

trait Cancellation extends Facet[State, Outcome, Fact], Phased[State, Phase]:

  object cancellation:

    // A canceled answer settles only an activity whose cancellation was requested; By-ID keeps its
    // notFound checks and its cancellation reason.
    object rules extends Rules:
      on(client.requestCancel) {
        when[Closed] ~> rejects(Rejection.notFound)
        when(Phase.scheduled, Phase.paused) ~> enters(Phase.canceled)
        when(Phase.started) ~> enters(Phase.cancelRequested)
        when(Phase.cancelRequested) ~> rejects(Rejection.failedPrecondition)
          .because(cancellationAlreadyRequested)
      }
      on(activityWorker.respondCanceled) {
        when(Phase.cancelRequested) ~> enters(Phase.canceled)
        when(Phase.started) ~> rejects(Rejection.invalidArgument)
          .because(cancellationNotRequested)
      }
      on(completionClient.respondCanceledByID) {
        when[Closed] ~> rejects(Rejection.notFound)
        when(Phase.scheduled, Phase.paused) ~> rejects(Rejection.notFound)
        when(Phase.cancelRequested) ~> enters(Phase.canceled)
        when(Phase.started) ~> rejects(Rejection.invalidArgument)
          .because(cancellationNotRequestedByID)
      }
