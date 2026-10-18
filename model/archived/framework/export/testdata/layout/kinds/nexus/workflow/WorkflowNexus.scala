package fixture.features.nexus
package workflow

import framework.dsl.*

enum Outcome:
  case accepted

object user extends Actor:
  val complete = action(this)

given Ok[Outcome] = Ok(Outcome.accepted)
