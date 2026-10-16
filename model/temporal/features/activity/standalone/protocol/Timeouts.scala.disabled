// The standalone activity Protocol's Timeouts facet: the scheduling and attempt deadlines, with the
// terminal deadline witnesses, both orders of competing scheduling deadlines, and the retry after an
// attempt timeout.
package temporal
package features.activity
package standalone
package protocol

import framework.*
import framework.Finite.given
import framework.outcomes.Outcome
import temporal.capabilities.system.*
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
          ActivityProtocol.states.afterRetry(s, ActivityProtocol.states.landingPhase(s)),
          ActivityProtocol.states.landingStatus(s),
          Fact.attemptCount
        )
          .because(ActivityProtocol.states.nominalTimeWindow)

      def timeOut(s: State, t: TimeoutType) =
        enter(s.copy(phase = Phase.timedOut), Fact.statusTimedOut(t))
          .because(ActivityProtocol.states.nominalTimeWindow)

    object rules extends Rules:
      // Started and not over: the phases a deadline can fire in.
      on(deadline.scheduleToClose) {
        when[Live].where(ActivityProtocol.states.scheduleToCloseArmed) ~> (effects.timeOut(
          _,
          TimeoutType.scheduleToClose
        ))
      }
      // Waiting for a worker: the phases before an attempt is held, which schedule-to-start covers.
      on(deadline.scheduleToStart) {
        when[Waiting].where(ActivityProtocol.states.scheduleToStartArmed) ~> (effects.timeOut(
          _,
          TimeoutType.scheduleToStart
        ))
      }
      on(deadline.startToClose) {
        when(Phase.started).where(s =>
          ActivityProtocol.states.startToCloseArmed(s) && ActivityProtocol.states.retriesRemaining(
            s
          )
        ) ~> effects.backOffAfterDeadline
        when(Phase.pauseRequested).where(s =>
          ActivityProtocol.states.startToCloseArmed(s) && ActivityProtocol.states.retriesRemaining(
            s
          )
        ) ~> effects.backOffAfterDeadline
        when(Phase.started, Phase.pauseRequested, Phase.cancelRequested)
          .where(s =>
            ActivityProtocol.states.startToCloseArmed(s) && ActivityProtocol.states.endsTerminally(
              s
            )
          ) ~> (effects.timeOut(
          _,
          TimeoutType.startToClose
        ))
        when(Phase.resetRequested).where(ActivityProtocol.states.startToCloseArmed) ~>
          reset.effects.applyReset
        when(Phase.resetKeepingPause).where(ActivityProtocol.states.startToCloseArmed) ~>
          reset.effects.applyReset
      }

    object properties:

      object safety:
        // Terminal deadline witnesses record which deadline settled the activity. Start-to-close
        // uses a one-attempt start.
        val scheduleToStartFires = property when deadline.scheduleToStart holds { s =>
          s.state.phase == Phase.timedOut &&
          s.records(Fact.statusTimedOut(TimeoutType.scheduleToStart))
        }

        val scheduleToCloseTimesOut = property when deadline.scheduleToClose holds { s =>
          s.state.phase == Phase.timedOut &&
          s.records(Fact.statusTimedOut(TimeoutType.scheduleToClose))
        }

    // The terminal scheduling deadlines and the attempt's own, retryable one; a pending reset
    // applies as start-to-close ends the held attempt, and schedule-to-close keeps its terminal law.
    object capabilities extends Capabilities:
      val scheduleToCloseDeadline: Capability = Expirable[State, Phase, Live, Fact](
        timer = deadline.scheduleToClose,
        armed = ActivityProtocol.states.scheduleToCloseArmed,
        timeout = Fact.statusTimedOut(TimeoutType.scheduleToClose),
        timeoutFacts = ActivityProtocol.states.timeoutFacts
      )
      val scheduleToStartDeadline: Capability = Expirable[State, Phase, Waiting, Fact](
        timer = deadline.scheduleToStart,
        armed = ActivityProtocol.states.scheduleToStartArmed,
        timeout = Fact.statusTimedOut(TimeoutType.scheduleToStart),
        timeoutFacts = ActivityProtocol.states.timeoutFacts
      )
      val startToCloseDeadline: Capability = Expirable[State, Phase, Held, Fact](
        timer = deadline.startToClose,
        armed = ActivityProtocol.states.startToCloseArmed,
        timeout = Fact.statusTimedOut(TimeoutType.startToClose),
        timeoutFacts = ActivityProtocol.states.timeoutFacts,
        retryable = true,
        retriesRemaining = Some(ActivityProtocol.states.retriesRemaining),
        pendingPause = Some(ActivityProtocol.states.pendingPause),
        pendingCancel = Some(ActivityProtocol.states.pendingCancel)
      )
      overriding(
        (Expirable.deadlineTimesOut[State, Phase]: AnyRef) ->
          (ActivityProtocol.properties.resetDeadlineTimesOut: AnyRef),
        because = ActivityProtocol.states.pendingResetReason,
        of = Seq(startToCloseDeadline)
      )
      overriding(
        (Expirable.deadlineReturnsToWaiting[State, Phase]: AnyRef) ->
          (ActivityProtocol.properties.resetDeadlineReturnsToWaiting: AnyRef),
        because = ActivityProtocol.states.pendingResetReason,
        of = Seq(startToCloseDeadline)
      )
      overriding(
        (Expirable.deadlinePauses[State, Phase]: AnyRef) ->
          (ActivityProtocol.properties.resetDeadlinePauses: AnyRef),
        because = ActivityProtocol.states.pendingResetReason,
        of = Seq(startToCloseDeadline)
      )

    object queries:
      val scheduleToStartExpires = scenario.actions(
        client.start(scheduleToStart := expires),
        process.stop,
        deadline.scheduleToStart
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
      val scheduleToStartTimeout =
        (query find properties.safety.scheduleToStartFires in scheduleToStartExpires limits three)
          .live(
            inconclusive(
              Reason.neverEvaluated,
              because =
                "each of the three deadlines records the timed-out status under one evidence name, so on an execution where another deadline fired the claim is never read"
            )
          )
      val startToCloseTimeout =
        witness(
          client.start(startToClose := expires, maxAttempts := MaxAttempts.one),
          activityWorker.poll,
          deadline.startToClose
        )
          .records(Fact.statusTimedOut(TimeoutType.startToClose))
          .ends(_.phase == Phase.timedOut)
          .limits(three)
      val competingTimers = Vector(
        query("competingTimers.scheduleToStartFirst") find
          properties.safety.scheduleToStartFires in
          bothDeadlinesStartFirst limits three,
        query("competingTimers.scheduleToCloseFirst") find
          properties.safety.scheduleToCloseTimesOut in
          bothDeadlinesCloseFirst limits three
      )
      // Both full-State finds predict explanationsDisagree from status/attempt-only evidence.
      // The shared live gate must check actual Property assessments and reasons.
      // The second delivery after an attempt timeout: a realization whose second delivery confirms
      // a timeout, not a failed answer, runs these.
      val retryAfterTimeout =
        witness(
          client.start(startToClose := expires, maxAttempts := MaxAttempts.two),
          activityWorker.poll,
          deadline.startToClose,
          timers.backoff,
          activityWorker.poll,
          activityWorker.respondCompleted
        )
          .records(Fact.statusCompleted)
          .ends(
            _ == ActivityProtocol.states.completedOnRetry.copy(
              startToClose = Timeout.expires,
              maxAttempts = MaxAttempts.two
            )
          )
          .limits(six)
          .live(
            inconclusive(
              Reason.explanationsDisagree,
              because =
                "the full-state claim is evidenced by status and attempt count alone, so the executions that explain the Run disagree on the state it fixes"
            )
          )
      val retryExhaustion =
        witness(
          client.start(startToClose := expires, maxAttempts := MaxAttempts.two),
          activityWorker.poll,
          deadline.startToClose,
          timers.backoff,
          activityWorker.poll,
          activityWorker.respondFailed(Failure.retryable)
        )
          .records(Fact.statusFailed)
          .ends(
            _ == ActivityProtocol.states.completedOnRetry.copy(
              phase = Phase.failed,
              startToClose = Timeout.expires,
              maxAttempts = MaxAttempts.two
            )
          )
          .limits(six)
          .live(
            inconclusive(
              Reason.explanationsDisagree,
              because =
                "the full-state claim is evidenced by status and attempt count alone, so the executions that explain the Run disagree on the state it fixes"
            )
          )
