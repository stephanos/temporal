package fixture.features.relay
package workflow

import umpire.*

val task = Entity(name = "task", key = "scheduledEvent")
object formBindings:
  val start = user.start.creates(task)
  val complete = user.complete.on(task)

object exports:
  val relayWorkflow = irFile("relay-workflow")(system.RelaySystem)
