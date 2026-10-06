package fixture.features.relay
package workflow

import umpire.*

object exports:
  val relayWorkflow = irFile("relay-workflow")(system.RelaySystem)
