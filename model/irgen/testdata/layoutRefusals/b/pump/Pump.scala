package fixture.features.pump

import framework.*

final case class Pump(on: Boolean)

enum Outcome:
  case accepted

object operator extends Actor:
  val start = action(this)

given Ok[Outcome] = Ok(Outcome.accepted)
