package fixture.features.kettle

import framework.*

final case class Kettle(hot: Boolean) derives Finite

enum Outcome derives Finite:
  case accepted

object cook extends Actor:
  val boil = action(this)

given Ok[Outcome] = Ok(Outcome.accepted)
