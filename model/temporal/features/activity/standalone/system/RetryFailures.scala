// Standalone activity failure retry and exhaustion on the unchanged lifecycle.
package temporal
package features.activity
package standalone
package system

import framework.*
import framework.Finite.given
import framework.realize.Reason
import temporal.realize.{inconclusive, satisfied}
import Bounds.{six, three}

val _ = () => summon[CanEqual[framework.outcomes.Outcome, framework.outcomes.Outcome]]

object RetryFailures extends Derived(ActivitySystem.rebind()):

  object states:
    // Completed on the second attempt of an activity with no deadline set.
    val completedOnRetry =
      ActivitySystem.init.copy(
        phase = Phase.completed,
        attempts = UpTo(ActivitySystem.states.attemptBound)
      )

  object properties:
    val nonRetryableFails =
      property when activityWorker.respondFailed(Failure.fatal) holds { s => // format: off
        s.state.phase == Phase.failed && s.records(Fact.statusFailed) // scalafix:ok DisableSyntax.==; strictEquality checks native comparisons
      } // format: on

    // The attempt count saturates at `ActivitySystem.states.attemptBound`, so the claim is bounded by it:
    // a completion on any later attempt than the second reads as this one.
    val retryCompletes =
      property when activityWorker.respondCompleted holds { s => // format: off
        s.state == states.completedOnRetry && s.records(Fact.statusCompleted) // scalafix:ok DisableSyntax.==; strictEquality checks native comparisons
      } // format: on
    // format: off
    // The pinned exhaustion path takes this action twice, so both failures must satisfy the
    // Property: the exact first retry, then the exact exhausted settlement.
    val retryExhausts = property when activityWorker.respondFailed(Failure.retryable) holds { s =>
      (s.state == states.completedOnRetry.copy( // scalafix:ok DisableSyntax.==; strictEquality checks native comparisons
        phase = Phase.scheduled,
        dispatch = Dispatch.backoff,
        attempts = UpTo(1),
        maxAttempts = MaxAttempts.two
      ) && s.records(Fact.statusScheduled) && s.records(Fact.attemptCount)) ||
      (s.state == states.completedOnRetry.copy( // scalafix:ok DisableSyntax.==; strictEquality checks native comparisons
        phase = Phase.failed,
        maxAttempts = MaxAttempts.two
      ) && // format: on
        s.records(Fact.statusFailed))
    }

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

    val nonRetryableFailure =
      (query find properties.nonRetryableFails in nonRetryable limits three).expect(satisfied)
    val retry =
      (query find properties.retryCompletes in retriedThenCompleted limits six)
        .expect(inconclusive(Reason.explanationsDisagree))
    val retryExhaustionByFailures =
      query verify properties.retryExhausts in exhausted limits six
