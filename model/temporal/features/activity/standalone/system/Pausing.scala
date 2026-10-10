// Standalone activity completion after pause and resume.
package temporal
package features.activity
package standalone
package system

import framework.*
import framework.Finite.given
import temporal.realize.satisfied

val _ = () => summon[CanEqual[framework.outcomes.Outcome, framework.outcomes.Outcome]]

object Pausing extends Derived(ActivitySystem.rebind()):

  object properties:
    val completes =
      property when activityWorker.respondCompleted holds { s => // format: off
        s.state.phase == Phase.completed && s.records(Fact.statusCompleted) // scalafix:ok DisableSyntax.==; strictEquality checks native comparisons
      } // format: on

  object queries:
    val pausedThenCompleted = scenario.actions(
      client.start(),
      client.pause,
      client.unpause,
      activityWorker.poll,
      activityWorker.respondCompleted
    )

    val pauseResume =
      (query find properties.completes in pausedThenCompleted limits six)
        .expect(satisfied)
