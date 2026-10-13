package fixture.features.contract

import framework.dsl.*

enum FacetName:
  case pressing, breaking
enum Outcome:
  case ok
given Ok[Outcome] = Ok(Outcome.ok)
object hand extends Actor:
  val press = action(this)
