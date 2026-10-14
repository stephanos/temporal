package fixture.features.relay
package workflow

import framework.dsl.*

val task = Entity(name = "task", key = "scheduledEvent")
object formBindings:
  val start = user.start.creates(task)
  val complete = user.complete.on(task)
