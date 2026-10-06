package fixture.features.cistern

import umpire.*

final case class Light(lit: Boolean) derives Finite

enum Outcome derives Finite:
  case accepted

object user extends Actor:
  val flip = action(this)

given Ok[Outcome] = Ok(Outcome.accepted)

object exports:
  val cistern = irFile("cistern")(product.CisternContract)
