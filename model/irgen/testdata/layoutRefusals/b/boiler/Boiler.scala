package fixture.features.boiler

import framework.*
import product.BoilerContract
import system.BoilerImplementation

final case class Boiler(hot: Boolean) derives Finite

enum Outcome derives Finite:
  case accepted

object cook extends Actor:
  val boil = action(this)

given Ok[Outcome] = Ok(Outcome.accepted)

object exports:
  val boiler = irFile("boiler")(BoilerContract, BoilerImplementation)
