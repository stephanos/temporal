package fixture.features.nexus
package workflow

import umpire.*

enum Outcome derives Finite:
  case accepted

object user extends Actor:
  val complete = action(this)

given Ok[Outcome] = Ok(Outcome.accepted)

object exports:
  val nexusWorkflow = irFile("nexus-workflow")(product.NexusProduct, system.NexusSystem)
