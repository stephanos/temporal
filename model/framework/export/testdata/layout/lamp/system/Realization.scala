// The System realization belongs beside LampSystem. A Product realization is added only when
// there is an executable Product subject; this template keeps the existing wrapper-object form.
package fixture.features.lamp
package system

import framework.realize.*
import framework.realize.Activation.Controller

object LampRealization:
  val system = Realization(
    machine = LampSystem,
    producer = family + ".testpilot",
    producerVersion = "1",
    roles = Vector.empty,
    correlation = Correlation(
      projection = family + ".projection.lamp",
      run = family + ".scope.run",
      operation = family + ".scope.lamp",
      observation = "correlated-evidence",
      events = 8,
      buffered = 8,
      keys = 2,
      support = 16,
      work = 1000,
      eventSize = 512
    ),
    scripts = Vector(Script("controller", Controller, Vector.empty))
  )
