package fixture.features.placement

import framework.*
import fixture.witness.{worker, Input, Note, Subject}

object Placed extends Derived(Subject.restrict(worker.begin, worker.finish)):
  object properties:
    val invariant = property holds (_.records(Note.other))
  object queries:
    val path = scenario.actions(worker.begin(), worker.finish(Input.first))
    val reached = witness(worker.begin(), worker.finish(Input.first)).records(Note.completed)
    val checked =
      query verify properties.invariant in path limits Limits(
        "checked",
        steps = 2,
        actions = 2,
        search = 64
      )
