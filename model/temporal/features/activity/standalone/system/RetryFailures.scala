// Standalone activity failure retry and exhaustion on the unchanged lifecycle.
package temporal
package features.activity
package standalone
package system

import framework.*
import framework.realize.Reason
import temporal.realize.{inconclusive, satisfied}
import Bounds.three

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
      property when worker.respondFailed(Failure.fatal) holds { s =>
        s.state.phase == Phase.failed && s.records(Fact.statusFailed)
      }

    // The attempt count saturates at `ActivitySystem.states.attemptBound`, so the claim is bounded by it:
    // a completion on any later attempt than the second reads as this one.
    val retryCompletes =
      property when worker.respondCompleted holds { s =>
        s.state == states.completedOnRetry && s.records(Fact.statusCompleted)
      }

    // The pinned exhaustion path takes this action twice, so both failures must satisfy the
    // Property: the exact first retry, then the exact exhausted settlement.
    val retryExhausts = property when worker.respondFailed(Failure.retryable) holds { s =>
      (s.state == states.completedOnRetry.copy(
        phase = Phase.scheduled,
        dispatch = Dispatch.backoff,
        attempts = UpTo(1),
        maxAttempts = MaxAttempts.two
      ) && s.records(Fact.statusScheduled) && s.records(Fact.attemptCount)) ||
      (s.state == states.completedOnRetry.copy(
        phase = Phase.failed,
        maxAttempts = MaxAttempts.two
      ) &&
        s.records(Fact.statusFailed))
    }

  object queries:
    val nonRetryable = scenario.actions(
      client.start(),
      worker.poll,
      worker.respondFailed(Failure.fatal)
    )
    val retriedThenCompleted = scenario.actions(
      client.start(),
      worker.poll,
      worker.respondFailed(Failure.retryable),
      timers.backoff,
      worker.poll,
      worker.respondCompleted
    )
    val exhausted = scenario.actions(
      client.start(maxAttempts := MaxAttempts.two),
      worker.poll,
      worker.respondFailed(Failure.retryable),
      timers.backoff,
      worker.poll,
      worker.respondFailed(Failure.retryable)
    )

    val nonRetryableFailure =
      (query find properties.nonRetryableFails in nonRetryable limits three).expect(satisfied)
    val retry =
      (query find properties.retryCompletes in retriedThenCompleted limits six)
        .expect(inconclusive(Reason.explanationsDisagree))
    val retryExhaustionByFailures =
      query verify properties.retryExhausts in exhausted limits six
