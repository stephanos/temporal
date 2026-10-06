// R20 (b): a two-level feature's Phase, State and Fact belong to the level files, not here.
package fixture.features.reservoir

import umpire.*
import product.ReservoirProduct
import system.ReservoirSystem

enum ProductPhase derives Finite:
  case empty, full

final case class ProductState(phase: ProductPhase) derives Finite

enum ProductFact derives Finite:
  case filled

enum Phase derives Finite:
  case empty, full

final case class SystemState(phase: Phase) derives Finite

enum SystemFact derives Finite:
  case filled

enum Outcome derives Finite:
  case accepted

object user extends Actor:
  val fill = action(this)

given Ok[Outcome] = Ok(Outcome.accepted)

object exports:
  val reservoir = irFile("reservoir")(ReservoirProduct, ReservoirSystem)
