// Standalone activity retry and exhaustion after an attempt timeout.
package temporal
package features.activity
package standalone
package system

import framework.*
import framework.Finite.given
import framework.realize.Reason
import temporal.realize.inconclusive
import Timeout.expires

val _ = () => summon[CanEqual[framework.outcomes.Outcome, framework.outcomes.Outcome]]

// The same System state and rules, with a realization whose second delivery confirms a timeout,
// not a failed answer. One kind confirms all the occurrences it names, so the failure-retry kind
// cannot evidence a path on which no failed answer occurs.
object TimeoutRetry extends Derived(ActivitySystem.unmonitored):

  object properties:
    val completesAfterTimeout = property when activityWorker.respondCompleted holds {
      s => // format: off
      s.state == RetryFailures.states.completedOnRetry.copy( // scalafix:ok DisableSyntax.==; strictEquality checks native comparisons
        startToClose = Timeout.expires,
        maxAttempts = MaxAttempts.two
      ) && s.records(Fact.statusCompleted) // format: on
    } // format: off
    val failsAfterTimeout = property when activityWorker.respondFailed(Failure.retryable) holds { s =>
      s.state == RetryFailures.states.completedOnRetry.copy( // scalafix:ok DisableSyntax.==; strictEquality checks native comparisons
        phase = Phase.failed,
        startToClose = Timeout.expires,
        maxAttempts = MaxAttempts.two
      ) && s.records(Fact.statusFailed) // format: on
    }

  object queries:
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
    // Both full-State finds predict explanationsDisagree from status/attempt-only evidence.
    // The shared live gate must check actual Property assessments and reasons.
    val retryAfterTimeout =
      (query find properties.completesAfterTimeout in timedOutThenCompleted limits six)
        .expect(inconclusive(Reason.explanationsDisagree))
    val retryExhaustion =
      (query find properties.failsAfterTimeout in timedOutThenFailed limits six)
        .expect(inconclusive(Reason.explanationsDisagree))
