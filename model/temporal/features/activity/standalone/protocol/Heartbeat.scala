// The standalone activity Protocol's Heartbeat facet: the worker's heartbeat and its deadline, with
// the heartbeat completion, retry and exhaustion witnesses.
package temporal
package features.activity
package standalone
package protocol

import framework.*
import framework.Finite.given
import framework.outcomes.{Outcome, Rejection}
import temporal.capabilities.*
import temporal.realize.satisfied
import Bounds.{eight, four}
import Timeout.expires

val _ = () => summon[CanEqual[framework.outcomes.Outcome, framework.outcomes.Outcome]]

trait Heartbeat extends Facet[State, Outcome, Fact], Phased[State, Phase]:
  this: Completion =>
  object heartbeat:
    object effects:
      def heartbeat(s: State) = enter(s, Fact.heartbeatReceived)
      def heartbeatBackOff(s: State) =
        enter(
          ActivityProtocol.states.afterRetry(s, ActivityProtocol.states.landingPhase(s)),
          ActivityProtocol.states.landingStatus(s),
          Fact.attemptCount,
          Fact.heartbeatTimedOut
        )
          .because(ActivityProtocol.states.nominalTimeWindow)
      def heartbeatTimeOut(s: State) =
        enter(
          s.copy(phase = Phase.timedOut),
          Fact.statusTimedOut(TimeoutType.heartbeat),
          Fact.heartbeatTimedOut
        )
          .because(ActivityProtocol.states.nominalTimeWindow)
      // A pending reset applies as the heartbeat deadline ends the held attempt.
      def applyResetOnHeartbeat(s: State) =
        enter(
          s.copy(
            phase = ActivityProtocol.states.landingPhase(s),
            attempts = UpTo(0),
            dispatch = Dispatch.now
          ),
          ActivityProtocol.states.landingStatus(s),
          Fact.attemptCount,
          Fact.heartbeatTimedOut
        ).because(ActivityProtocol.states.nominalTimeWindow)

    object rules extends Rules:
      on(activityWorker.heartbeat) {
        when[Held] ~> effects.heartbeat
        when(Phase.unstarted, Phase.scheduled, Phase.paused) ~> rejects(Rejection.notFound)
        when[Closed] ~> rejects(Rejection.notFound)
      }
      on(deadline.heartbeat) {
        when(Phase.started).where(s =>
          ActivityProtocol.states.heartbeatArmed(s) && ActivityProtocol.states.retriesRemaining(s)
        ) ~> effects.heartbeatBackOff
        when(Phase.pauseRequested).where(s =>
          ActivityProtocol.states.heartbeatArmed(s) && ActivityProtocol.states.retriesRemaining(s)
        ) ~> effects.heartbeatBackOff
        when(Phase.started, Phase.pauseRequested, Phase.cancelRequested).where(s =>
          ActivityProtocol.states.heartbeatArmed(s) && ActivityProtocol.states.endsTerminally(s)
        ) ~> effects.heartbeatTimeOut
        when(Phase.resetRequested).where(ActivityProtocol.states.heartbeatArmed) ~>
          effects.applyResetOnHeartbeat
        when(Phase.resetKeepingPause).where(ActivityProtocol.states.heartbeatArmed) ~>
          effects.applyResetOnHeartbeat
      }

    object properties:
      val heartbeatExhausts = property.when(deadline.heartbeat) holds (after =>
        after.records(Fact.heartbeatTimedOut) &&
          (ActivityProtocol.states.retriesRemaining(after.state) || // format: off
            (after.state.phase == Phase.timedOut && after.records( // scalafix:ok DisableSyntax.==; strictEquality checks native comparisons
              Fact.statusTimedOut(TimeoutType.heartbeat)
            ))) // format: on
      )

    // A pending reset applies as the heartbeat deadline ends the held attempt, before its
    // retryability or the policy is read (attempt.go:201-216, statemachine.go:317-345).
    object capabilities extends Capabilities:
      val heartbeatDeadline: Capability = Expirable[State, Phase, Held, Fact](
        timer = deadline.heartbeat,
        armed = ActivityProtocol.states.heartbeatArmed,
        timeout = Fact.statusTimedOut(TimeoutType.heartbeat),
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
        of = Seq(heartbeatDeadline)
      )
      overriding(
        (Expirable.deadlineReturnsToWaiting[State, Phase]: AnyRef) ->
          (ActivityProtocol.properties.resetDeadlineReturnsToWaiting: AnyRef),
        because = ActivityProtocol.states.pendingResetReason,
        of = Seq(heartbeatDeadline)
      )
      overriding(
        (Expirable.deadlinePauses[State, Phase]: AnyRef) ->
          (ActivityProtocol.properties.resetDeadlinePauses: AnyRef),
        because = ActivityProtocol.states.pendingResetReason,
        of = Seq(heartbeatDeadline)
      )

    object queries:
      val heartbeatCompleted = scenario.actions(
        client.start(standalone.heartbeat := expires),
        activityWorker.poll,
        activityWorker.heartbeat,
        activityWorker.respondCompleted
      )
      val heartbeatRetried = scenario.actions(
        client.start(standalone.heartbeat := expires, maxAttempts := MaxAttempts.two),
        activityWorker.poll,
        activityWorker.heartbeat,
        deadline.heartbeat,
        timers.backoff,
        activityWorker.poll,
        activityWorker.respondCompleted
      )
      val heartbeatExhausted = scenario.actions(
        client.start(standalone.heartbeat := expires, maxAttempts := MaxAttempts.one),
        activityWorker.poll,
        activityWorker.heartbeat,
        deadline.heartbeat
      )
      val heartbeatThenCompletes =
        (query find completion.properties.completes in heartbeatCompleted limits four)
          .total(22464)
          .expect(satisfied)
      val heartbeatTimeoutRetriesThenCompletes =
        (query find completion.properties.completes in heartbeatRetried limits eight)
          .total(39312)
          .expect(satisfied)
      val heartbeatTimeoutExhausts =
        (query find properties.heartbeatExhausts in heartbeatExhausted limits four)
          .total(22464)
          .expect(satisfied)
