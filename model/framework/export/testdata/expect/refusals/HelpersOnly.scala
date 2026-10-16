package framework.fixture.expect

import framework.dsl.*

// Refused under the reify plugin as without it: a block of helper vals states no requirement.
val helpersOnly = Subject
  .property("helpersOnly")
  .across
  .expect:
    val restarted = before.phase == Phase.done // refused
