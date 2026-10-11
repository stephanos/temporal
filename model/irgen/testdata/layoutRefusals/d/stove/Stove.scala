package fixture.features.stove

import framework.*

final case class Light(lit: Boolean)

enum Outcome:
  case accepted

object user extends Actor:
  val flip = action(this)

given Ok[Outcome] = Ok(Outcome.accepted)
