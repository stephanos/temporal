package fixture.features.pump

import framework.*

final case class Pump(on: Boolean) derives Finite

enum Outcome derives Finite:
  case accepted

object operator extends Actor:
  val start = action(this)

given Ok[Outcome] = Ok(Outcome.accepted)
