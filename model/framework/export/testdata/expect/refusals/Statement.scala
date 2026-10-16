package framework.fixture.expect

import framework.dsl.*

// Refused under the reify plugin as without it: a discarded statement is not a requirement.
val statement = Subject
  .property("statement")
  .expect:
    println("not a requirement") // refused
    check(after.state.beating)
