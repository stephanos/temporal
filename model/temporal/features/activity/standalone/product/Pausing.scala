// The standalone activity Product's Pausing facet: pause and unpause; a pause of a running attempt reads as started until the worker yields.
package temporal
package features.activity
package standalone
package product

import framework.*
import framework.outcomes.{Outcome, Rejection}
import temporal.capabilities.*

trait Pausing extends Facet[State, Outcome, Fact], Phased[State, Phase]:
  object pausing:
    object effects:
      val pauseApplied = choice
      val pausePending = choice
      val pauseAlreadyRequested = choice
      // started also reads a held attempt whose pause is pending (System.refinement.toProduct).
      def pauseHeld(s: State) = choose(
        pauseApplied -> enters(Phase.paused)(s),
        pausePending -> enter(s, Fact.statusPaused),
        pauseAlreadyRequested -> reject(Outcome.rejected(Rejection.failedPrecondition), s)
          .because(pauseAlreadyRequestedReason)
      )
      val pauseWithdrawn = choice
      val unpauseNotRequested = choice
      def unpauseHeld(s: State) = choose(
        pauseWithdrawn -> enters(Phase.started)(s),
        unpauseNotRequested -> reject(Outcome.rejected(Rejection.failedPrecondition), s)
          .because(activityNotPaused)
      )

    // Closed controls answer NotFound before the live-state checks (operator_commands.go:249-254).
    object rules extends Rules:
      on(client.pause) {
        when[Closed] ~> rejects(Rejection.notFound)
        when(Phase.scheduled) ~> enters(Phase.paused)
        when(Phase.started) ~> effects.pauseHeld
        when(Phase.paused, Phase.cancelRequested) ~> rejects(Rejection.failedPrecondition)
          .because(pausedOrCancelPending)
      }
      on(client.unpause) {
        when[Closed] ~> rejects(Rejection.notFound)
        where(ActivityProduct.states.paused) ~> enters(Phase.scheduled)
        when(Phase.started) ~> effects.unpauseHeld
        when(Phase.scheduled, Phase.cancelRequested) ~> rejects(Rejection.failedPrecondition)
          .because(activityNotPaused)
      }

    object capabilities extends Capabilities:
      val pausable: Capability = Pausable(
        pause = client.pause,
        unpause = client.unpause
      )
      overriding(
        Pausable.pausedIsNotDispatched[State, Phase] ->
          ActivityProduct.properties.pausedDoesNotStartAttempt,
        because = "cancelRequested does not reveal whether a worker still holds an attempt"
      )
