// The standalone activity Protocol's Cancellation facet: the cancel request and its settlement by
// the worker or by activity ID.
package temporal
package features.activity
package standalone
package protocol

import framework.*
import framework.Finite.given
import framework.outcomes.{Outcome, Rejection}
import framework.realize.Reason
import temporal.realize.inconclusive
import Bounds.four

val _ = () => summon[CanEqual[framework.outcomes.Outcome, framework.outcomes.Outcome]]

trait Cancellation extends Facet[State, Outcome, Fact], Phased[State, Phase]:
  object cancellation:
    object effects:
      def cancel(s: State) = enter(s.copy(phase = Phase.canceled), Fact.statusCanceled)

      def requestCancel(s: State) =
        enter(s.copy(phase = Phase.cancelRequested), Fact.statusCancelRequested)

    // Cancel > Reset > Pause: a cancellation replaces a pending reset and a requested pause, and
    // neither a pause nor a reset undoes it (model.go:156-158). A canceled answer needs a cancel
    // request; By-ID keeps its notFound checks and its cancellation reason.
    object rules extends Rules:
      on(client.requestCancel) {
        when[Closed] ~> rejects(Rejection.notFound)
        when(Phase.scheduled, Phase.paused) ~> effects.cancel
        when(
          Phase.started,
          Phase.pauseRequested,
          Phase.resetRequested,
          Phase.resetKeepingPause
        ) ~> effects.requestCancel
        when(Phase.cancelRequested) ~> rejects(Rejection.failedPrecondition)
          .because(cancellationAlreadyRequested)
      }
      on(activityWorker.respondCanceled) {
        when(Phase.cancelRequested) ~> effects.cancel
        when(Phase.started, Phase.pauseRequested, Phase.resetRequested, Phase.resetKeepingPause) ~>
          rejects(Rejection.invalidArgument)
            .because(cancellationNotRequested)
      }
      on(completionClient.respondCanceledByID) {
        when(Phase.unstarted) ~> rejects(Rejection.notFound)
        when[Closed] ~> rejects(Rejection.notFound)
        when(Phase.scheduled, Phase.paused) ~> rejects(Rejection.notFound)
        when(Phase.cancelRequested) ~> effects.cancel
        when(Phase.started, Phase.pauseRequested, Phase.resetRequested, Phase.resetKeepingPause) ~>
          rejects(Rejection.invalidArgument)
            .because(cancellationNotRequestedByID)
      }

    object properties:
      val cancelRequested =
        property when client.requestCancel holds { s => // format: off
          s.state.phase == Phase.cancelRequested && s.records(Fact.statusCancelRequested) // scalafix:ok DisableSyntax.==; strictEquality checks native comparisons
        } // format: on

      val canceledByWorker =
        property when activityWorker.respondCanceled holds { s => // format: off
          s.state.phase == Phase.canceled && s.records(Fact.statusCanceled) // scalafix:ok DisableSyntax.==; strictEquality checks native comparisons
        } // format: on

      val cancelIsRequested =
        property("activityProtocol.cancelIsRequested") when client.requestCancel holds (
          _.records(Fact.statusCancelRequested)
        )
      val canceledByID = property when completionClient.respondCanceledByID holds {
        after => // format: off
        after.state.phase == Phase.canceled && after.records(Fact.statusCanceled) // scalafix:ok DisableSyntax.==; strictEquality checks native comparisons
      } // format: on

    object queries:
      val cancelRequestedThenCanceled = scenario.actions(
        client.start(),
        activityWorker.poll,
        client.requestCancel,
        activityWorker.respondCanceled
      )
      val heldCancellation = scenario.actions(
        client.start(),
        activityWorker.poll,
        client.requestCancel,
        completionClient.respondCanceledByID
      )

      val cancel =
        query find properties.canceledByWorker in cancelRequestedThenCanceled limits four
      // Asks the one Property no functional Query asks, over the path that takes a cancel request.
      val cancelRequest =
        query find properties.cancelRequested in cancelRequestedThenCanceled limits four
      val heldCanceledByID =
        (query find properties.canceledByID in heldCancellation limits four)
          .total(22464)
          .expect(inconclusive(Reason.explanationsDisagree))
      val cancelIsRequested =
        (query(
          "activityProtocol.cancelIsRequested"
        ) find properties.cancelIsRequested in heldCancellation limits four)
          .total(22464)
          .expect(inconclusive(Reason.explanationsDisagree))
