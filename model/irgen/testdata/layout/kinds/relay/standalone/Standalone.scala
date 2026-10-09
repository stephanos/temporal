package fixture.features.relay
package standalone

import framework.*

val task = Entity(name = "task", key = "operationId")
object formBindings:
  val start = user.start.creates(task)
  val complete = user.complete.on(task)

object exports:
  val relayStandalone = irFile("relay-standalone")(system.RelaySystem)
