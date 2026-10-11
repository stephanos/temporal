// The standalone activity Protocol's Retry facet: a worker's or a By-ID failure, which backs a
// held attempt off, fails it or applies a pending reset, with the failure retry claims.
package temporal
package features.activity
package standalone
package protocol

import framework.*
import framework.Finite.given
import framework.outcomes.{Outcome, Rejection}
import framework.realize.Reason
import temporal.capabilities.system.*
import temporal.realize.{inconclusive, satisfied}
import Bounds.{six, three}

val _ = () => summon[CanEqual[framework.outcomes.Outcome, framework.outcomes.Outcome]]

trait Retry extends Facet[State, Outcome, Fact], Phased[State, Phase]:
  this: Reset & Cancellation =>

  object retry:

    object effects:
      def fail(s: State) = enter(s.copy(phase = Phase.failed), Fact.statusFailed)

      // A retryable failure backs a started attempt off: read as scheduled again, one attempt higher.
      def backOff(s: State) =
        enter(
          ActivityProtocol.states.afterRetry(s, ActivityProtocol.states.landingPhase(s)),
          ActivityProtocol.states.landingStatus(s),
          Fact.attemptCount
        )
          .because("a retryable attempt backs off; the client reads scheduled again")

      // A pause requested during the attempt takes effect on its delayed retry (model.go:130-137).
      def backOffPaused(s: State) =
        enter(
          ActivityProtocol.states.afterRetry(s, ActivityProtocol.states.landingPhase(s)),
          ActivityProtocol.states.landingStatus(s),
          Fact.attemptCount
        )

    // A worker's answer settles the attempt it holds. A retryable failure backs a started attempt
    // off, settles a cancel-requested one as canceled and lands a pause-requested one in paused
    // (TransitionAttemptFailedWhilePauseRequested). A pending reset applies on either failure,
    // before its retryability is read. Held failure settlement must match the By-ID rows; By-ID
    // keeps its notFound checks.
    object rules extends Rules:
      on(activityWorker.respondFailed) {
        when(Phase.resetRequested) ~> reset.effects.applyReset
        when(Phase.resetKeepingPause) ~> reset.effects.applyReset
      }
      on(activityWorker.respondFailed(Failure.fatal)) {
        when(Phase.started, Phase.pauseRequested, Phase.cancelRequested) ~> effects.fail
      }
      on(activityWorker.respondFailed(Failure.retryable)) {
        when(Phase.started).where(ActivityProtocol.states.retriesRemaining) ~> effects.backOff
        when(Phase.cancelRequested) ~> cancellation.effects.cancel
        when(Phase.pauseRequested).where(ActivityProtocol.states.retriesRemaining) ~>
          effects.backOffPaused
        when(Phase.started, Phase.pauseRequested).where(ActivityProtocol.states.exhausted) ~>
          effects.fail
      }
      on(completionClient.respondFailedByID) {
        when(Phase.unstarted) ~> rejects(Rejection.notFound)
        when[Closed] ~> rejects(Rejection.notFound)
        when(Phase.scheduled, Phase.paused) ~> rejects(Rejection.notFound)
        when(Phase.resetRequested) ~> reset.effects.applyReset
        when(Phase.resetKeepingPause) ~> reset.effects.applyReset
      }
      on(completionClient.respondFailedByID(Failure.fatal)) {
        when(Phase.started, Phase.pauseRequested, Phase.cancelRequested) ~> effects.fail
      }
      on(completionClient.respondFailedByID(Failure.retryable)) {
        when(Phase.started).where(ActivityProtocol.states.retriesRemaining) ~> effects.backOff
        when(Phase.pauseRequested).where(ActivityProtocol.states.retriesRemaining) ~>
          effects.backOffPaused
        when(Phase.started, Phase.pauseRequested).where(ActivityProtocol.states.exhausted) ~>
          effects.fail
        when(Phase.cancelRequested) ~> cancellation.effects.cancel
      }

    object properties:
      val nonRetryableFails =
        property when activityWorker.respondFailed(Failure.fatal) holds { s =>
          s.state.phase == Phase.failed && s.records(Fact.statusFailed)
        }

      // The attempt count saturates at `ActivityProtocol.states.attemptBound`, so the claim is bounded
      // by it: a completion on any later attempt than the second reads as this one.
      val retryCompletes =
        property when activityWorker.respondCompleted holds { s =>
          s.state == ActivityProtocol.states.completedOnRetry && s.records(Fact.statusCompleted)
        }
      // The pinned exhaustion path takes this action twice, so both failures must satisfy the
      // Property: the exact first retry, then the exact exhausted settlement.
      val retryExhausts = property when activityWorker.respondFailed(Failure.retryable) holds { s =>
        (s.state == ActivityProtocol.states.completedOnRetry.copy(
          phase = Phase.scheduled,
          dispatch = Dispatch.backoff,
          attempts = UpTo(1),
          maxAttempts = MaxAttempts.two
        ) && s.records(Fact.statusScheduled) && s.records(Fact.attemptCount)) ||
        (s.state == ActivityProtocol.states.completedOnRetry.copy(
          phase = Phase.failed,
          maxAttempts = MaxAttempts.two
        ) &&
          s.records(Fact.statusFailed))
      }

      val fatalFailureByID =
        property when completionClient.respondFailedByID(Failure.fatal) holds { after =>
          after.state.phase == Phase.failed && after.records(Fact.statusFailed)
        }

    // The retryable and the fatal failure, each a Retryable instance; a pending reset applies first,
    // as a failure ends the held attempt, whatever its retryability or the policy left
    // (attempt.go:201-216, statemachine.go:317-345).
    object capabilities extends Capabilities:
      val retryableFailure: Capability = Retryable[State, Phase, MaxAttempts.Bound](
        failure = activityWorker.respondFailed(Failure.retryable),
        retryable = true,
        attemptCount = ActivityProtocol.states.attemptCount,
        maximumAttempts = ActivityProtocol.states.maximumAttempts,
        retriesRemaining = ActivityProtocol.states.retriesRemaining,
        pendingPause = Some(ActivityProtocol.states.pendingPause),
        pendingCancel = Some(ActivityProtocol.states.pendingCancel)
      )
      val fatalFailure: Capability = Retryable[State, Phase, MaxAttempts.Bound](
        failure = activityWorker.respondFailed(Failure.fatal),
        retryable = false,
        attemptCount = ActivityProtocol.states.attemptCount,
        maximumAttempts = ActivityProtocol.states.maximumAttempts,
        retriesRemaining = ActivityProtocol.states.retriesRemaining,
        pendingPause = Some(ActivityProtocol.states.pendingPause),
        pendingCancel = Some(ActivityProtocol.states.pendingCancel)
      )
      overriding(
        Retryable.failureReturnsToWaiting[State, Phase] ->
          ActivityProtocol.properties.resetFailureReturnsToWaiting,
        because = ActivityProtocol.states.pendingResetReason,
        of = Seq(retryableFailure, fatalFailure)
      )
      overriding(
        Retryable
          .failureEndsFailed[State, Phase] -> ActivityProtocol.properties.resetFailureEndsFailed,
        because = ActivityProtocol.states.pendingResetReason,
        of = Seq(retryableFailure, fatalFailure)
      )
      overriding(
        Retryable.failurePauses[State, Phase] -> ActivityProtocol.properties.resetFailurePauses,
        because = ActivityProtocol.states.pendingResetReason,
        of = Seq(retryableFailure, fatalFailure)
      )

    object queries:
      val nonRetryable = scenario.actions(
        client.start(),
        activityWorker.poll,
        activityWorker.respondFailed(Failure.fatal)
      )
      val retriedThenCompleted = scenario.actions(
        client.start(),
        activityWorker.poll,
        activityWorker.respondFailed(Failure.retryable),
        timers.backoff,
        activityWorker.poll,
        activityWorker.respondCompleted
      )
      val exhausted = scenario.actions(
        client.start(maxAttempts := MaxAttempts.two),
        activityWorker.poll,
        activityWorker.respondFailed(Failure.retryable),
        timers.backoff,
        activityWorker.poll,
        activityWorker.respondFailed(Failure.retryable)
      )
      val heldFatalFailure = scenario.actions(
        client.start(),
        activityWorker.poll,
        completionClient.respondFailedByID(Failure.fatal)
      )

      val nonRetryableFailure =
        (query find properties.nonRetryableFails in nonRetryable limits three).live(satisfied)
      val retry =
        (query find properties.retryCompletes in retriedThenCompleted limits six)
          .live(
            inconclusive(
              Reason.explanationsDisagree,
              because =
                "the claim fixes the whole state the activity ends in, its deadlines included, and the start's answer is one fact for every start class, so the executions that explain the Run disagree"
            )
          )
      val retryExhaustionByFailures =
        query verify properties.retryExhausts in exhausted limits six
      // A closed activity answers a repeated By-ID call NotFound and records nothing, so the
      // explanations of a By-ID witness disagree.
      val heldFailedByID =
        (query find properties.fatalFailureByID in heldFatalFailure limits three)
          .total(16848)
          .live(
            inconclusive(
              Reason.explanationsDisagree,
              because =
                "a closed activity answers a repeated By-ID call NotFound and records nothing, so a Run cannot rule out a silent rejected repeat: the executions that explain it disagree"
            )
          )
