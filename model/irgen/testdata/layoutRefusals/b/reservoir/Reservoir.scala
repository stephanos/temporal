// R20 (b): a two-level feature's Phase, State and Fact belong to the level files, not here.
package fixture.features.reservoir

import framework.*

enum ProductPhase:
  case empty, full

final case class ProductState(phase: ProductPhase)

enum ProductFact:
  case filled

enum Phase:
  case empty, full

final case class SystemState(phase: Phase)

enum SystemFact:
  case filled

enum Outcome:
  case accepted

object user extends Actor:
  val fill = action(this)

given Ok[Outcome] = Ok(Outcome.accepted)
