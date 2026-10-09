// Pausing and unpausing an entity. Suspended phases say where dispatch must remain blocked.
package temporal.capabilities

import framework.*
import scala.annotation.unused
import scala.reflect.{ClassTag, TypeTest}

final case class Pausable[S, P](
    pause: ClassRef | Composed,
    unpause: ClassRef | Composed
)(using
    @unused phasing: Phasing[S, P],
    @unused finite: Finite[P],
    @unused witness: TypeTest[P, Suspended],
    @unused phaseType: ClassTag[P]
) extends CapabilityOf[S, Nothing, Nothing]

object Pausable extends CapabilityKind:
  type Suspended = framework.Suspended
  // No step from a Suspended phase lands in Held: no work is handed out while paused.
  // Reading Pollable's Held role brings this Property only where both capabilities are declared.
  // This promises neither what pausing held work does (activity waits as pause-requested), the
  // answer to a second pause or an unpause of a live entity, nor that an unpause resumes work.
  // See chasm/lib/activity/statemachine.go, chasm/lib/activity/tasks.go and
  // chasm/lib/scheduler/scheduler.go.
  def pausedIsNotDispatched[S, P](m: Declares[S])(using
      phasing: Phasing[S, P],
      finite: Finite[P],
      suspended: TypeTest[P, Suspended],
      held: TypeTest[P, Pollable.Held],
      phaseType: ClassTag[P]
  ): Property[S] =
    m.property
      .never(s => phasing.roleCases[Pollable.Held](m.name).contains(phasing.phase(s.state)))
      .from(s => phasing.roleCases[Suspended](m.name).contains(phasing.phase(s)))
