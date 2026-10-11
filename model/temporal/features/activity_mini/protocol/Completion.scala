package temporal
package features.activity
package mini
package protocol

import framework.*
import framework.Finite.given
import framework.outcomes.{Outcome, Rejection}
import temporal.features.activity.{Failure, deadline}
import temporal.capabilities.system.{Retryable, Expirable}

trait Completion extends Facet[State, Outcome, Fact], Phased[State, Phase]:
  this: Dispatching =>
  object completion:
    object effects:
      def fail(s: State, failure: Failure) =
        if failure == Failure.retryable && s.phase == Phase.cancelRequested then
          enters(Phase.canceled)(s)
        else if failure == Failure.retryable && ActivityProtocol.states.retriesRemaining(s) then
          enters(Phase.backingOff)(s)
        else enters(Phase.failed)(s)

    object rules extends Rules:
      on(client.cancel) {
        when[Closed] ~> rejects(Rejection.notFound)
        when(Phase.scheduled, Phase.backingOff) ~> enters(Phase.canceled)
        when(Phase.started) ~> enters(Phase.cancelRequested)
        when(Phase.cancelRequested) ~> rejects(Rejection.failedPrecondition)
      }
      on(activityWorker.respondCompleted) {
        when[Held] ~> enters(Phase.completed)
      }
      on(activityWorker.respondFailed) {
        when[Held] ~> effects.fail
      }
      on(deadline.scheduleToStart) {
        where(dispatching.states.waiting).where(ActivityProtocol.states.scheduleArmed) ~> enters(Phase.timedOut)
      }
      on(deadline.startToClose) {
        when[Held].where(ActivityProtocol.states.closeArmed) ~> enters(Phase.timedOut)
      }

    object properties:
      def retryBacksOff(m: Declares[State])(failure: ClassRef, retryable: Boolean,
          retriesRemaining: State => Boolean, pendingPause: State => Boolean,
          pendingCancel: State => Boolean): Property[State] =
        m.property.when(failure) holdsAcross ((before, after) =>
          !(retryable && retriesRemaining(before) && !pendingPause(before) && !pendingCancel(before)) ||
            after.state.phase == Phase.backingOff)

    object capabilities extends Capabilities:
      val retryable: Capability = Retryable[State, Phase, 2](
        failure = activityWorker.respondFailed(Failure.retryable), retryable = true,
        attemptCount = ActivityProtocol.states.attemptCount,
        maximumAttempts = ActivityProtocol.states.maximumAttempts,
        retriesRemaining = ActivityProtocol.states.retriesRemaining,
        pendingCancel = Some(ActivityProtocol.states.pendingCancel))
      overriding(Retryable.failureReturnsToWaiting[State, Phase] -> properties.retryBacksOff,
        because = "The mini protocol distinguishes retry backoff from dispatch-ready waiting while reporting scheduled for both.")
      val scheduleDeadline: Capability = Expirable[State, Phase, Waiting, Fact](
        timer = deadline.scheduleToStart, armed = ActivityProtocol.states.scheduleArmed,
        timeout = Fact.statusTimedOut, timeoutFacts = Seq(Fact.statusTimedOut))
