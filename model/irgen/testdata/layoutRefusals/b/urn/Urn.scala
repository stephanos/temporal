package fixture.features.urn

import framework.*

final case class Urn(hot: Boolean) derives Finite

enum Outcome derives Finite:
  case accepted

object cook extends Actor:
  val boil = action(this)

given Ok[Outcome] = Ok(Outcome.accepted)
