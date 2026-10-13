package fixture.features.relay
package standalone

import framework.dsl.*

val task = Entity(name = "task", key = "operationId")
object formBindings:
  val start = user.start.creates(task)
  val complete = user.complete.on(task)
