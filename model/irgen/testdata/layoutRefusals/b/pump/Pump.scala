package fixture.features.pump

import umpire.*
import product.PumpProduct
import system.PumpProtocol

final case class Pump(on: Boolean) derives Finite

enum Outcome derives Finite:
  case accepted

object operator extends Actor:
  val start = action(this)

given Ok[Outcome] = Ok(Outcome.accepted)

object exports:
  val pump = irFile("pump")(PumpProduct, PumpProtocol)
