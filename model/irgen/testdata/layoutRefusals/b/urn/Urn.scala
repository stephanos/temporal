package fixture.features.urn

import framework.*
import product.UrnProduct
import system.UrnSystem

final case class Urn(hot: Boolean) derives Finite

enum Outcome derives Finite:
  case accepted

object cook extends Actor:
  val boil = action(this)

given Ok[Outcome] = Ok(Outcome.accepted)

object exports:
  val urn = irFile("urn")(UrnProduct, UrnSystem)
