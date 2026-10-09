// Standalone activity dispatch eligibility through start delay and retry backoff.
package temporal
package features.activity
package standalone
package system

import framework.*
import temporal.realize.satisfied
import Bounds.four
import Timeout.expires

object DispatchEligibility extends Derived(ActivitySystem.rebind()):
  object properties:
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

    val completes =
      property when worker.respondCompleted holds { s =>
        s.state.phase == Phase.completed && s.records(Fact.statusCompleted)
      }

  object queries:
    val any = scenario.free

    val delayedThenCompleted = scenario.actions(
      client.start(startDelay := expires),
      timers.startDelay,
      worker.poll,
      worker.respondCompleted
    )

    val delayedAttemptsAreNotDispatched =
      query verify properties.dispatchRequiresReady in any limits eight
    val scheduleToStartWaitsForDispatch =
      query verify properties.scheduleToStartRequiresDispatch in any limits eight
    val startDelayedCompletion =
      (query find properties.completes in delayedThenCompleted limits four).expect(satisfied)
