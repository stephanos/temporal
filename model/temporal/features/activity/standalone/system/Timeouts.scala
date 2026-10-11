// The standalone activity Protocol's Timeouts facet: the scheduling and attempt deadlines, with the
// terminal deadline witnesses, both orders of competing scheduling deadlines, and the retry after an
// attempt timeout.
package temporal
package features.activity
package standalone
package system

import framework.*
import framework.Finite.given
import framework.outcomes.Outcome
import temporal.capabilities.*
import framework.realize.Reason
import temporal.realize.inconclusive
import Bounds.{six, three}
import actors.worker.worker as process
import Timeout.expires

val _ = () => summon[CanEqual[framework.outcomes.Outcome, framework.outcomes.Outcome]]

trait Timeouts extends Facet[State, Outcome, Fact], Phased[State, Phase]:
  this: Reset =>
  object timeouts:
    object effects:
      def backOffAfterDeadline(s: State) =
        enter(
          ActivitySystem.states.afterRetry(s, ActivitySystem.states.landingPhase(s)),
          ActivitySystem.states.landingStatus(s),
          Fact.attemptCount
        )
          .because(ActivitySystem.states.nominalTimeWindow)

      def timeOut(s: State, t: TimeoutType) =
        enter(s.copy(phase = Phase.timedOut), Fact.statusTimedOut(t))
          .because(ActivitySystem.states.nominalTimeWindow)

    object rules extends Rules:
      // Started and not over: the phases a deadline can fire in.
      on(deadline.scheduleToClose) {
        when[Live].where(ActivitySystem.states.scheduleToCloseArmed) ~> (effects.timeOut(
          _,
          TimeoutType.scheduleToClose
        ))
      }
      // Waiting for a worker: the phases before an attempt is held, which schedule-to-start covers.
      on(deadline.scheduleToStart) {
        when[Waiting].where(ActivitySystem.states.scheduleToStartArmed) ~> (effects.timeOut(
          _,
          TimeoutType.scheduleToStart
        ))
      }
      on(deadline.startToClose) {
        when(Phase.started).where(s =>
          ActivitySystem.states.startToCloseArmed(s) && ActivitySystem.states.retriesRemaining(s)
        ) ~> effects.backOffAfterDeadline
        when(Phase.pauseRequested).where(s =>
          ActivitySystem.states.startToCloseArmed(s) && ActivitySystem.states.retriesRemaining(s)
        ) ~> effects.backOffAfterDeadline
        when(Phase.started, Phase.pauseRequested, Phase.cancelRequested)
          .where(s =>
            ActivitySystem.states.startToCloseArmed(s) && ActivitySystem.states.endsTerminally(s)
          ) ~> (effects.timeOut(
          _,
          TimeoutType.startToClose
        ))
        when(Phase.resetRequested).where(ActivitySystem.states.startToCloseArmed) ~>
          reset.effects.applyReset
        when(Phase.resetKeepingPause).where(ActivitySystem.states.startToCloseArmed) ~>
          reset.effects.applyReset
      }

    object properties:
      // Terminal deadline witnesses record which deadline settled the activity. Start-to-close
      // uses a one-attempt start.
      val scheduleToStartFires = property when deadline.scheduleToStart holds { s =>
        s.state.phase == Phase.timedOut && // scalafix:ok DisableSyntax.==; strictEquality checks native comparisons
        s.records(Fact.statusTimedOut(TimeoutType.scheduleToStart))
      }

      val startToCloseFires = property when deadline.startToClose holds { s =>
        s.state.phase == Phase.timedOut && // scalafix:ok DisableSyntax.==; strictEquality checks native comparisons
        s.records(Fact.statusTimedOut(TimeoutType.startToClose))
      }

      val scheduleToCloseTimesOut = property when deadline.scheduleToClose holds { s =>
        s.state.phase == Phase.timedOut && // scalafix:ok DisableSyntax.==; strictEquality checks native comparisons
        s.records(Fact.statusTimedOut(TimeoutType.scheduleToClose))
      }

      // The second delivery after an attempt timeout: a realization whose second delivery confirms
      // a timeout, not a failed answer, runs these.
      val completesAfterTimeout = property when activityWorker.respondCompleted holds {
        s => // format: off
        s.state == ActivitySystem.states.completedOnRetry.copy( // scalafix:ok DisableSyntax.==; strictEquality checks native comparisons
          startToClose = Timeout.expires,
          maxAttempts = MaxAttempts.two
        ) && s.records(Fact.statusCompleted) // format: on
      } // format: off
      val failsAfterTimeout = property when activityWorker.respondFailed(Failure.retryable) holds { s =>
        s.state == ActivitySystem.states.completedOnRetry.copy( // scalafix:ok DisableSyntax.==; strictEquality checks native comparisons
          phase = Phase.failed,
          startToClose = Timeout.expires,
          maxAttempts = MaxAttempts.two
        ) && s.records(Fact.statusFailed) // format: on
      }

    // The terminal scheduling deadlines and the attempt's own, retryable one; a pending reset
    // applies as start-to-close ends the held attempt, and schedule-to-close keeps its terminal law.
    object capabilities extends Capabilities:
      val scheduleToCloseDeadline: Capability = Expirable[State, Phase, Live, Fact](
        timer = deadline.scheduleToClose,
        armed = ActivitySystem.states.scheduleToCloseArmed,
        timeout = Fact.statusTimedOut(TimeoutType.scheduleToClose),
        timeoutFacts = ActivitySystem.states.timeoutFacts
      )
      val scheduleToStartDeadline: Capability = Expirable[State, Phase, Waiting, Fact](
        timer = deadline.scheduleToStart,
        armed = ActivitySystem.states.scheduleToStartArmed,
        timeout = Fact.statusTimedOut(TimeoutType.scheduleToStart),
        timeoutFacts = ActivitySystem.states.timeoutFacts
      )
      val startToCloseDeadline: Capability = Expirable[State, Phase, Held, Fact](
        timer = deadline.startToClose,
        armed = ActivitySystem.states.startToCloseArmed,
        timeout = Fact.statusTimedOut(TimeoutType.startToClose),
        timeoutFacts = ActivitySystem.states.timeoutFacts,
        retryable = true,
        retriesRemaining = Some(ActivitySystem.states.retriesRemaining),
        pendingPause = Some(ActivitySystem.states.pendingPause),
        pendingCancel = Some(ActivitySystem.states.pendingCancel)
      )
      overriding(
        (Expirable.deadlineTimesOut[State, Phase]: AnyRef) ->
          (ActivitySystem.properties.resetDeadlineTimesOut: AnyRef),
        because = ActivitySystem.states.pendingResetReason,
        of = Seq(startToCloseDeadline)
      )
      overriding(
        (Expirable.deadlineReturnsToWaiting[State, Phase]: AnyRef) ->
          (ActivitySystem.properties.resetDeadlineReturnsToWaiting: AnyRef),
        because = ActivitySystem.states.pendingResetReason,
        of = Seq(startToCloseDeadline)
      )
      overriding(
        (Expirable.deadlinePauses[State, Phase]: AnyRef) ->
          (ActivitySystem.properties.resetDeadlinePauses: AnyRef),
        because = ActivitySystem.states.pendingResetReason,
        of = Seq(startToCloseDeadline)
      )

    object queries:
      val scheduleToStartExpires = scenario.actions(
        client.start(scheduleToStart := expires),
        process.stop,
        deadline.scheduleToStart
      )
      val startToCloseExpires = scenario.actions(
        client.start(startToClose := expires, maxAttempts := MaxAttempts.one),
        activityWorker.poll,
        deadline.startToClose
      )
      // Neither scheduling deadline is ordered before the other, so each firing is a trace of its
      // own. Both deadlines are set and no attempt started, so either may fire first. No
      // realization runs them.
      val bothDeadlinesStartFirst = scenario.actions(
        client.start(scheduleToClose := Timeout.expires, scheduleToStart := Timeout.expires),
        deadline.scheduleToStart
      )
      val bothDeadlinesCloseFirst = scenario.actions(
        client.start(scheduleToClose := Timeout.expires, scheduleToStart := Timeout.expires),
        deadline.scheduleToClose
      )
      val timedOutThenCompleted = scenario.actions(
        client.start(startToClose := expires, maxAttempts := MaxAttempts.two),
        activityWorker.poll,
        deadline.startToClose,
        timers.backoff,
        activityWorker.poll,
        activityWorker.respondCompleted
      )
      val timedOutThenFailed = scenario.actions(
        client.start(startToClose := expires, maxAttempts := MaxAttempts.two),
        activityWorker.poll,
        deadline.startToClose,
        timers.backoff,
        activityWorker.poll,
        activityWorker.respondFailed(Failure.retryable)
      )

      val scheduleToStartTimeout =
        (query find properties.scheduleToStartFires in scheduleToStartExpires limits three)
          .expect(inconclusive(Reason.neverEvaluated))
      val startToCloseTimeout =
        query find properties.startToCloseFires in startToCloseExpires limits three
      val competingTimers = Vector(
        query("competingTimers.scheduleToStartFirst") find
          properties.scheduleToStartFires in
          bothDeadlinesStartFirst limits three,
        query("competingTimers.scheduleToCloseFirst") find
          properties.scheduleToCloseTimesOut in
          bothDeadlinesCloseFirst limits three
      )
      // Both full-State finds predict explanationsDisagree from status/attempt-only evidence.
      // The shared live gate must check actual Property assessments and reasons.
      val retryAfterTimeout =
        (query find properties.completesAfterTimeout in timedOutThenCompleted limits six)
          .expect(inconclusive(Reason.explanationsDisagree))
      val retryExhaustion =
        (query find properties.failsAfterTimeout in timedOutThenFailed limits six)
          .expect(inconclusive(Reason.explanationsDisagree))
