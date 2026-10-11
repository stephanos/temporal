package fixture.features.activity.mini

import framework.*

enum Outcome:
  case accepted

object user extends Actor:
  val complete = action(this)

given Ok[Outcome] = Ok(Outcome.accepted)
