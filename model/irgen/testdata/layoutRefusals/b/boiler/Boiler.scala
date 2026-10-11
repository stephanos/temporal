package fixture.features.boiler

import framework.*

final case class Boiler(hot: Boolean) derives Finite

enum Outcome derives Finite:
  case accepted

object cook extends Actor:
  val boil = action(this)

given Ok[Outcome] = Ok(Outcome.accepted)
