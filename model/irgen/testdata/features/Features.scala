// A feature file's instructions, which it writes in the kit's lower-case forms (fn-133.3): the
// lifter refuses an instruction's core case class and a command written out with its name here, in
// a directory named `features`, at their lines.
package fixture.features

import framework.realize.*
import temporal.realize.*
import temporal.realize.WorkerInstruction.Fault
import temporal.features.activity.standalone.activity
import temporal.features.activity.standalone.system.ActivitySystem as activitySystem

private def realizing(command: Command | Instruction) = temporalRealization(
  machine = activitySystem,
  operation = activity,
  roles = Vector(taskQueue),
  scripts = Vector(controller(everyCase(command))),
  evidence = Vector.empty
)

private val stopWorker = Fault(taskQueue, FaultKind.workerStop)

// An instruction's core case class.
val upperCase: Realization = realizing(stopWorker)

private val stopNamed = Command("stop-named", fault(taskQueue, FaultKind.workerStop))

// A command written out with its name.
val spelledOut: Realization = realizing(stopNamed)
