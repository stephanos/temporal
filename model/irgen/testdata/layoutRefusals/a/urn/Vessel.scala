// R20 (a): the urn has two levels, but no root feature file (this file is not named after its
// folder) and no product/Product.scala (its Product sits in product/Base.scala); both are refused
// at the refinement (system/System.scala).
package fixture.features.urn

import framework.*

final case class Urn(full: Boolean) derives Finite

enum Outcome derives Finite:
  case accepted

type UrnStep = Step[Urn, Outcome, Nothing]
