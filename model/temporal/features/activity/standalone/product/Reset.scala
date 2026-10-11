// The standalone activity Product's Reset facet: a reset of a waiting activity applies at once; of a held attempt it waits for the attempt to end, which Describe still reads as started, or as pause-requested where a pause is kept.
package temporal
package features.activity
package standalone
package product

import framework.*
import framework.outcomes.{Outcome, Rejection}

trait Reset extends Facet[State, Outcome, Fact], Phased[State, Phase]:
  object reset:
    object effects:
      val resetPending = choice
      val resetPausePending = choice
      val resetAlreadyRequested = choice
      def resetHeld(s: State) = choose(
        resetPending -> enters(Phase.started)(s),
        resetAlreadyRequested -> reject(Outcome.rejected(Rejection.failedPrecondition), s)
          .because(resetAlreadyPending)
      )
      def resetHeldKeepingPause(s: State) = choose(
        resetPending -> enters(Phase.started)(s),
        resetPausePending -> enter(s, Fact.statusPaused),
        resetAlreadyRequested -> reject(Outcome.rejected(Rejection.failedPrecondition), s)
          .because(resetAlreadyPending)
      )

    object rules extends Rules:
      on(client.reset) {
        when[Closed] ~> rejects(Rejection.notFound)
        when(Phase.scheduled) ~> enters(Phase.scheduled)
        when(Phase.cancelRequested) ~> rejects(Rejection.failedPrecondition)
          .because(resetWithPendingCancellation)
      }
      on(client.reset(ResetPause.resume)) {
        where(ActivityProduct.states.paused) ~> enters(Phase.scheduled)
        when(Phase.started) ~> effects.resetHeld
      }
      on(client.reset(ResetPause.keepPaused)) {
        where(ActivityProduct.states.paused) ~> enters(Phase.paused)
        when(Phase.started) ~> effects.resetHeldKeepingPause
      }
