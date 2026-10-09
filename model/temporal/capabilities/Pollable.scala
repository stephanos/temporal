// Polling hands an entity's work to a worker; Held phases say where the worker owns it.
package temporal.capabilities

import framework.*
import scala.annotation.unused
import scala.reflect.{ClassTag, TypeTest}

final case class Pollable[S, P](dispatch: ClassRef | Composed)(using
    @unused phasing: Phasing[S, P],
    @unused finite: Finite[P],
    @unused witness: TypeTest[P, Held],
    @unused phaseType: ClassTag[P]
) extends CapabilityOf[S, Nothing, Nothing]

object Pollable extends CapabilityKind:
  type Held = framework.Held
