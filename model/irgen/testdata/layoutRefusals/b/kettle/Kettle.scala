package fixture.features.kettle

import umpire.*
import product.KettleContract
import system.KettleSystem

final case class Kettle(hot: Boolean) derives Finite

enum Outcome derives Finite:
  case accepted

object cook extends Actor:
  val boil = action(this)

given Ok[Outcome] = Ok(Outcome.accepted)

object exports:
  val kettle = irFile("kettle")(KettleContract, KettleSystem)
