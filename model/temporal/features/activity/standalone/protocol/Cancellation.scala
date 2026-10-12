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
      def cancel(s: State) =
        enter(s.copy(phase = Phase.canceled), Fact.statusCanceled)

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
        when(
          Phase.started,
          Phase.pauseRequested,
          Phase.resetRequested,
          Phase.resetKeepingPause
        ) ~>
          rejects(Rejection.invalidArgument)
            .because(cancellationNotRequested)
      }
      on(completionClient.respondCanceledByID) {
        when(Phase.unstarted) ~> rejects(Rejection.notFound)
        when[Closed] ~> rejects(Rejection.notFound)
        when(Phase.scheduled, Phase.paused) ~> rejects(Rejection.notFound)
        when(Phase.cancelRequested) ~> effects.cancel
        when(
          Phase.started,
          Phase.pauseRequested,
          Phase.resetRequested,
          Phase.resetKeepingPause
        ) ~>
          rejects(Rejection.invalidArgument)
            .because(cancellationNotRequestedByID)
      }

    object properties:

      object safety:
        val cancelRequested =
          property when client.requestCancel holds { s =>
            s.state.phase == Phase.cancelRequested && s.records(
              Fact.statusCancelRequested
            )
          }

        val canceledByWorker =
          property when activityWorker.respondCanceled holds { s =>
            s.state.phase == Phase.canceled && s.records(Fact.statusCanceled)
          }

        val canceledByID =
          property when completionClient.respondCanceledByID holds { after =>
            after.state.phase == Phase.canceled && after.records(
              Fact.statusCanceled
            )
          }

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
        query find properties.safety.canceledByWorker in cancelRequestedThenCanceled limits four
      // Asks the one Property no functional Query asks, over the path that takes a cancel request.
      val cancelRequest =
        query find properties.safety.cancelRequested in cancelRequestedThenCanceled limits four
      val heldCanceledByID =
        (query find properties.safety.canceledByID in heldCancellation limits four)
          .total(22464)
          .live(
            inconclusive(
              Reason.explanationsDisagree,
              because =
                "a closed activity answers a repeated By-ID call NotFound and records nothing, so a Run cannot rule out a silent rejected repeat: the executions that explain it disagree"
            )
          )
      val cancelIsRequested =
        (query(
          "activityProtocol.cancelIsRequested"
        ) find properties.safety.cancelRequested in heldCancellation limits four)
          .total(22464)
          .live(
            inconclusive(
              Reason.explanationsDisagree,
              because =
                "a closed activity answers a repeated By-ID call NotFound and records nothing, so a Run cannot rule out a silent rejected repeat: the executions that explain it disagree"
            )
          )
