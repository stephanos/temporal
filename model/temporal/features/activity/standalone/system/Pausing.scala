// Standalone activity completion after pause and resume.
package temporal
package features.activity
package standalone
package system

import framework.*
import temporal.realize.satisfied

object Pausing extends Derived(ActivitySystem.rebind()):
  object properties:
    val completes =
      property when worker.respondCompleted holds { s =>
        s.state.phase == Phase.completed && s.records(Fact.statusCompleted)
      }

  object queries:
    val pausedThenCompleted = scenario.actions(
      client.start(),
      client.pause,
      client.unpause,
      worker.poll,
      worker.respondCompleted
    )

    val pauseResume =
      (query find properties.completes in pausedThenCompleted limits six)
        .expect(satisfied)
