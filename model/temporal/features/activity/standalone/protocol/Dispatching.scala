// The standalone activity Protocol's Dispatching facet: the start, the worker's poll and the timers
// that make a waiting attempt dispatchable, with the dispatch eligibility claims.
package temporal
package features.activity
package standalone
package protocol

import scala.annotation.unused
import framework.*
import framework.Finite.given
import framework.outcomes.Outcome
import temporal.realize.satisfied
import Bounds.{eight, four}
import Timeout.expires

val _ = () => summon[CanEqual[framework.outcomes.Outcome, framework.outcomes.Outcome]]

trait Dispatching extends Facet[State, Outcome, Fact], Phased[State, Phase]:
  this: Completion =>

  object dispatching:

    object effects:
      // The start creates the activity, with the deadlines its inputs set.
      def schedule(
          @unused s: State,
          scheduleToClose: Timeout,
          scheduleToStart: Timeout,
          startToClose: Timeout,
          heartbeat: Timeout,
          startDelay: Timeout,
          maxAttempts: MaxAttempts
      ) =
        enter(
          protocol.State(
            phase = Phase.scheduled,
            dispatch = if startDelay == Timeout.expires then Dispatch.startDelay else Dispatch.now,
            attempts = UpTo(0),
            scheduleToClose = scheduleToClose,
            scheduleToStart = scheduleToStart,
            startToClose = startToClose,
            heartbeat = heartbeat,
            maxAttempts = maxAttempts
          ),
          Fact.statusScheduled
        )
      // This independent counter advances on delivery; Describe counts an attempt from scheduling.
      def startAttempt(s: State) =
        enter(
          s.copy(
            phase = Phase.started,
            attempts = ActivityProtocol.states.saturatingSucc(s.attempts)
          ),
          Fact.statusStarted,
          Fact.attemptCount
        )

      // A delay expires without changing status, including while paused (model.go:355-376).
      def dispatchNow(s: State) =
        enter(s.copy(dispatch = Dispatch.now)).because(ActivityProtocol.states.nominalTimeWindow)

    object rules extends Rules:
      on(client.start) {
        when(Phase.unstarted) ~> effects.schedule
      }
      on(activityWorker.poll) {
        when(Phase.scheduled).where(_.dispatch == Dispatch.now) ~> effects.startAttempt
      }
      on(timers.startDelay) {
        when[Live].where(_.dispatch == Dispatch.startDelay) ~> effects.dispatchNow
      }
      on(timers.backoff) {
        when[Live].where(_.dispatch == Dispatch.backoff) ~> effects.dispatchNow
      }

    object properties:

      object safety:
        // A dispatch delay remains pending across pause/unpause (model.go:239-244,355-376).
        val dispatchRequiresReady = property.holdsAcross { (before, after) =>
          (before.phase == Phase.scheduled && after.records(Fact.statusStarted)) implies
            (before.dispatch == Dispatch.now)
        }

        // Schedule-to-start counts from dispatch, after either delay (model.go:312-322).
        val scheduleToStartRequiresDispatch = property.holdsAcross { (before, after) =>
          after.records(Fact.statusTimedOut(TimeoutType.scheduleToStart)) implies
            (before.phase == Phase.scheduled && before.dispatch == Dispatch.now)
        }

    object queries:
      val delayedThenCompleted = scenario.actions(
        client.start(startDelay := expires),
        timers.startDelay,
        activityWorker.poll,
        activityWorker.respondCompleted
      )

      val delayedAttemptsAreNotDispatched =
        query verify properties.safety.dispatchRequiresReady in ActivityProtocol.queries.any limits eight
      val scheduleToStartWaitsForDispatch =
        query verify properties.safety.scheduleToStartRequiresDispatch in ActivityProtocol.queries.any limits eight
      val startDelayedCompletion =
        (query find completion.properties.safety.completes in delayedThenCompleted limits four)
          .live(satisfied)
