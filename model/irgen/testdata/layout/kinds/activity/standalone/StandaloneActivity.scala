package fixture.features.activity
package standalone

import framework.*

enum Outcome derives Finite:
  case accepted

object user extends Actor:
  val complete = action(this)

given Ok[Outcome] = Ok(Outcome.accepted)

object exports:
  val activityStandalone =
    irFile("fixture-activity-standalone")(product.ActivityProduct, system.ActivitySystem)
