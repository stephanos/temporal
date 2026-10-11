// Standalone activity heartbeat completion, retry and exhaustion.
package temporal
package features.activity
package standalone
package system

import framework.*
import framework.Finite.given
import temporal.realize.satisfied
import Bounds.{eight, four}
import Timeout.expires

val _ = () => summon[CanEqual[framework.outcomes.Outcome, framework.outcomes.Outcome]]

object HeartbeatCompletion extends Derived(ActivitySystem.unmonitored):

  object properties:
    val heartbeatCompletes = property when activityWorker.respondCompleted holds {
      s => // format: off
      s.state.phase == Phase.completed && s.records(Fact.statusCompleted) // scalafix:ok DisableSyntax.==; strictEquality checks native comparisons
    } // format: on

  object queries:
    val heartbeatCompleted = scenario.actions(
      client.start(heartbeat := expires),
      activityWorker.poll,
      activityWorker.heartbeat,
      activityWorker.respondCompleted
    )
    val heartbeatThenCompletes =
      (query find properties.heartbeatCompletes in heartbeatCompleted limits four)
        .total(22464)
        .expect(satisfied)

object HeartbeatRetry extends Derived(ActivitySystem.unmonitored):

  object properties:
    val heartbeatRetryCompletes = property when activityWorker.respondCompleted holds {
      s => // format: off
      s.state.phase == Phase.completed && s.records(Fact.statusCompleted) // scalafix:ok DisableSyntax.==; strictEquality checks native comparisons
    } // format: on

  object queries:
    val heartbeatRetried = scenario.actions(
      client.start(heartbeat := expires, maxAttempts := MaxAttempts.two),
      activityWorker.poll,
      activityWorker.heartbeat,
      deadline.heartbeat,
      timers.backoff,
      activityWorker.poll,
      activityWorker.respondCompleted
    )
    val heartbeatTimeoutRetriesThenCompletes =
      (query find properties.heartbeatRetryCompletes in heartbeatRetried limits eight)
        .total(39312)
        .expect(satisfied)

object HeartbeatExhaustion extends Derived(ActivitySystem.unmonitored):

  object properties:
    val heartbeatExhausts = property.when(deadline.heartbeat) holds (after =>
      after.records(Fact.heartbeatTimedOut) &&
        (ActivitySystem.states.retriesRemaining(after.state) || // format: off
          (after.state.phase == Phase.timedOut && after.records( // scalafix:ok DisableSyntax.==; strictEquality checks native comparisons
            Fact.statusTimedOut(TimeoutType.heartbeat)
          ))) // format: on
    )

  object queries:
    val heartbeatExhausted = scenario.actions(
      client.start(heartbeat := expires, maxAttempts := MaxAttempts.one),
      activityWorker.poll,
      activityWorker.heartbeat,
      deadline.heartbeat
    )
    val heartbeatTimeoutExhausts =
      (query find properties.heartbeatExhausts in heartbeatExhausted limits four)
        .total(22464)
        .expect(satisfied)
