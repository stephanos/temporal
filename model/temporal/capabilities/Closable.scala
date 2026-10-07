// An entity's terminal statuses and the Properties every closing entity keeps.
package temporal.capabilities

import umpire.*

final case class Closable[S, P, O](status: S => P, terminal: P => Boolean, rejected: O)
    extends CapabilityOf[S, O, Nothing]

object Closable extends CapabilityKind:
  // No step moves an entity out of a terminal status, or between terminal statuses.
  // This promises neither rejection of a mutation, reporting the status nor recording close once.
  // See chasm/lib/activity/statemachine.go and chasm/lib/nexusoperation/operation_statemachine.go.
  def terminalStatesAreFinal[S, P](m: Declares[S])(
      status: S => P,
      terminal: P => Boolean
  ): Property[S] =
    m.property holdsAcross ((before, after) =>
      !terminal(status(before)) || status(after.state) == status(before)
    )

  // Every step from a terminal status keeps the state and answers the entity's rejecting outcome.
  // The outcome is entity-specific: activity answers NotFound, Nexus and schedule FailedPrecondition.
  // This does not promise rejection of a repeated request id, which the server answers as the first.
  // See chasm/lib/activity/activity.go, chasm/lib/nexusoperation/operation.go and
  // chasm/lib/scheduler/scheduler.go.
  def closedIsRejectedUniformly[S, P, O](m: Declares[S] { type Outcome = O })(
      status: S => P,
      terminal: P => Boolean,
      rejected: O
  ): Property[S] =
    m.property holdsAcross ((before, after) =>
      !terminal(status(before)) || (after.state == before && after.outcome == rejected)
    )
