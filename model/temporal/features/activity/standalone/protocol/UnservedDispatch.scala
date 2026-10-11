package temporal
package features.activity
package standalone
package protocol

import framework.*
import framework.Finite.given
import Bounds.three

val _ = () => summon[CanEqual[framework.outcomes.Outcome, framework.outcomes.Outcome]]

object UnservedDispatch
    extends DerivedComposition(ServedDispatch.withMember(_.worker -> ActivityWorker)),
      NegativeControl:

  object properties:
    val startsWithinOneStep = UnservedDispatch.leadsTo("startsWithinOneStep")(
      ServedDispatch.states.dispatchable,
      ServedDispatch.states.started,
      within = 1,
      unservedDispatchRuns
    )

  object queries:
    val anyDispatch = scenario.free
    val stoppingWorkersStartNothing =
      query verify ServedDispatch.properties.startedByPollingWorker(UnservedDispatch) in
        anyDispatch limits three
