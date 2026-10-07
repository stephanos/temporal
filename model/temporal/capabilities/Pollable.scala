// Polling hands the entity's work to a worker; `running` says where the worker holds it.
package temporal.capabilities

import umpire.*

final case class Pollable[S](dispatch: ClassRef | Composed, running: S => Boolean)
    extends CapabilityOf[S, Nothing, Nothing]

object Pollable extends CapabilityKind
