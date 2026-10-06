package fixture.features.relay
package standalone

import umpire.*

object exports:
  val relayStandalone = irFile("relay-standalone")(system.RelaySystem)
