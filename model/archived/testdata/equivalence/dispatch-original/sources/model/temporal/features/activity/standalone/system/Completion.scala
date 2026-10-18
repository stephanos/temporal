// Standalone activity completion, verified on the unchanged lifecycle.
package temporal
package features.activity
package standalone
package system

import framework.*
import framework.Finite.given
import temporal.realize.satisfied
import Bounds.three

val _ = () => summon[CanEqual[framework.outcomes.Outcome, framework.outcomes.Outcome]]

object Completion extends Derived(ActivitySystem.rebind()):

  object properties:
    val completes =
      property when worker.respondCompleted holds { s => // format: off
        s.state.phase == Phase.completed && s.records(Fact.statusCompleted) // scalafix:ok DisableSyntax.==; strictEquality checks native comparisons
      } // format: on

  object queries:
    val completed = scenario.actions(
      client.start(),
      worker.poll,
      worker.respondCompleted
    )

    val completion =
      (query find properties.completes in completed limits three).expect(satisfied)
