package fixture.features.boiler

import framework.*

final case class Boiler(hot: Boolean)

enum Outcome:
  case accepted

object cook extends Actor:
  val boil = action(this)

given Ok[Outcome] = Ok(Outcome.accepted)
