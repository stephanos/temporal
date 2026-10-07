// Pausing and unpausing an entity, separately from handing its work to a worker.
package temporal.capabilities

import umpire.*

final case class Pausable[S](
    pause: ClassRef | Composed,
    unpause: ClassRef | Composed,
    paused: S => Boolean
) extends CapabilityOf[S, Nothing, Nothing]

object Pausable extends CapabilityKind:
  // No step from a paused state lands in running: no work is handed out while paused.
  // Reading Pollable's `running` brings this Property only where both capabilities are declared.
  // This promises neither what pausing held work does (activity waits as pause-requested), the
  // answer to a second pause or an unpause of a live entity, nor that an unpause resumes work.
  // See chasm/lib/activity/statemachine.go, chasm/lib/activity/tasks.go and
  // chasm/lib/scheduler/scheduler.go.
  def pausedIsNotDispatched[S](m: Declares[S])(
      paused: S => Boolean,
      running: S => Boolean
  ): Property[S] =
    m.property.never(s => running(s.state)).from(paused)
