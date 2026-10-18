package framework.fixture.expect

import framework.dsl.*

// Refused under the reify plugin as without it: a predicate does not construct a step.
val stepped: Step[Job, Said, Note] => Boolean = expect:
  check(Step(Said.accepted, after.state).state.beating) // refused
