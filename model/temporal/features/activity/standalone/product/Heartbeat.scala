// The standalone activity Product's Heartbeat facet: the worker's heartbeat and its deadline.
package temporal
package features.activity
package standalone
package product

import framework.*
import framework.outcomes.{Outcome, Rejection}

trait Heartbeat extends Facet[State, Outcome, Fact], Phased[State, Phase]:
  object heartbeat:
    object effects:
      def heartbeat(s: State) = enter(s, Fact.heartbeatReceived)
      val heartbeatRetry = choice
      val heartbeatPaused = choice
      val heartbeatExhausted = choice
      def heartbeatExpires(s: State) = choose(
        heartbeatRetry -> enter(
          s.copy(phase = Phase.scheduled),
          Fact.statusScheduled,
          Fact.heartbeatTimedOut
        ),
        heartbeatPaused -> enter(
          s.copy(phase = Phase.paused),
          Fact.statusPaused,
          Fact.heartbeatTimedOut
        ),
        heartbeatExhausted -> heartbeatTimeOut(s)
      )
      def heartbeatTimeOut(s: State) =
        enter(s.copy(phase = Phase.timedOut), Fact.statusTimedOut, Fact.heartbeatTimedOut)

    object rules extends Rules:
      on(activityWorker.heartbeat) {
        when[Held] ~> effects.heartbeat
        when(Phase.scheduled, Phase.paused) ~> rejects(Rejection.notFound)
        when[Closed] ~> rejects(Rejection.notFound)
      }
      on(deadline.heartbeat) {
        when(Phase.started) ~> effects.heartbeatExpires
        when(Phase.cancelRequested) ~> effects.heartbeatTimeOut
      }
