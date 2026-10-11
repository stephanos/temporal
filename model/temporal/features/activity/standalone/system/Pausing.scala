// The standalone activity Protocol's Pausing facet: pause and unpause, and completion after a pause.
package temporal
package features.activity
package standalone
package system

import framework.*
import framework.outcomes.{Outcome, Rejection}
import temporal.realize.satisfied
import Bounds.six

trait Pausing extends Facet[State, Outcome, Fact], Phased[State, Phase]:
  this: Completion =>
  object pausing:
    object effects:
      def pause(s: State) = enter(s.copy(phase = Phase.paused), Fact.statusPaused)

      // A pause of a held attempt is a request the worker learns of on its next heartbeat, so it is
      // its own phase the client reads as paused.
      def requestPause(s: State) =
        enter(s.copy(phase = Phase.pauseRequested), Fact.statusPaused)
          .because("the worker learns of the pause on its next heartbeat")

      def resume(s: State) = enter(s.copy(phase = Phase.scheduled), Fact.statusScheduled)

      // An unpause before the worker yields: it keeps the attempt it holds.
      def withdrawPause(s: State) = enter(s.copy(phase = Phase.started), Fact.statusStarted)

    // Closed controls answer NotFound before the live-state checks (operator_commands.go:249-254).
    object rules extends Rules:
      on(client.pause) {
        when[Closed] ~> rejects(Rejection.notFound)
        when(Phase.scheduled) ~> effects.pause
        when(Phase.started) ~> effects.requestPause
        when(
          Phase.paused,
          Phase.pauseRequested,
          Phase.cancelRequested,
          Phase.resetRequested,
          Phase.resetKeepingPause
        ) ~>
          rejects(Rejection.failedPrecondition)
            .because(pausedOrControlPending)
      }
      on(client.unpause) {
        when[Closed] ~> rejects(Rejection.notFound)
        when(Phase.paused) ~> effects.resume
        when(Phase.pauseRequested) ~> effects.withdrawPause
        when(
          Phase.scheduled,
          Phase.started,
          Phase.cancelRequested,
          Phase.resetRequested,
          Phase.resetKeepingPause
        ) ~>
          rejects(Rejection.failedPrecondition)
            .because(activityNotPaused)
      }

    object queries:
      val pausedThenCompleted = scenario.actions(
        client.start(),
        client.pause,
        client.unpause,
        activityWorker.poll,
        activityWorker.respondCompleted
      )

      val pauseResume =
        (query find completion.properties.completes in pausedThenCompleted limits six)
          .expect(satisfied)
