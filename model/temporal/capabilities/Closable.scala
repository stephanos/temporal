// An entity's terminal statuses and the Properties every closing entity keeps.
package temporal.capabilities

import umpire.*
import scala.annotation.unused
import scala.reflect.{ClassTag, TypeTest}

final case class Closable[S, P, O](rejected: O)(using
    @unused phasing: Phasing[S, P],
    @unused finite: Finite[P],
    @unused witness: TypeTest[P, Closed],
    @unused phaseType: ClassTag[P]
) extends CapabilityOf[S, O, Nothing]

object Closable extends CapabilityKind:
  type Closed = umpire.Closed
  // No step moves an entity out of a terminal status, or between terminal statuses.
  // This promises neither rejection of a mutation, reporting the status nor recording close once.
  // See chasm/lib/activity/statemachine.go and chasm/lib/nexusoperation/operation_statemachine.go.
  def terminalStatesAreFinal[S, P](m: Declares[S])(using
      phasing: Phasing[S, P],
      finite: Finite[P],
      witness: TypeTest[P, Closed],
      phaseType: ClassTag[P]
  ): Property[S] =
    m.property holdsAcross ((before, after) =>
      !phasing.roleCases[Closed](m.name).contains(phasing.phase(before)) ||
        phasing.phase(after.state) == phasing.phase(before)
    )

  // Every step from a terminal status keeps the state and answers the entity's rejecting outcome.
  // The outcome is entity-specific: activity answers NotFound, Nexus and schedule FailedPrecondition.
  // This does not promise rejection of a repeated request id, which the server answers as the first.
  // See chasm/lib/activity/activity.go, chasm/lib/nexusoperation/operation.go and
  // chasm/lib/scheduler/scheduler.go.
  def closedIsRejectedUniformly[S, P, O](m: Declares[S] { type Outcome = O })(
      rejected: O
  )(using
      phasing: Phasing[S, P],
      finite: Finite[P],
      witness: TypeTest[P, Closed],
      phaseType: ClassTag[P]
  ): Property[S] =
    m.property holdsAcross ((before, after) =>
      !phasing.roleCases[Closed](m.name).contains(phasing.phase(before)) ||
        (after.state == before && after.outcome == rejected)
    )
