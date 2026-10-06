package fixture.features.pan

import umpire.*

final case class Light(lit: Boolean) derives Finite

enum Outcome derives Finite:
  case accepted

object user extends Actor:
  val flip = action(this)

given Ok[Outcome] = Ok(Outcome.accepted)

object exports:
  val pan = irFile("pan")(system.PanImplementation)
