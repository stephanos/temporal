// Closing an entity; Closed phases define the terminal statuses its Properties read.
package temporal.capabilities

import framework.*
import framework.Finite.given
import scala.annotation.unused
import scala.reflect.{ClassTag, TypeTest}

val _ = () => summon[CanEqual[framework.outcomes.Outcome, framework.outcomes.Outcome]]

final case class Closable[S, P, O](rejected: O)(using
    @unused phasing: Phasing[S, P],
    @unused finite: Finite[P],
    @unused witness: TypeTest[P, Closed],
    @unused phaseType: ClassTag[P]
) extends CapabilityOf[S, O, Nothing]
// format: off
object Closable extends CapabilityKind:
  type Closed = framework.Closed
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
        phasing.phase(after.state) == phasing.phase(before) // scalafix:ok DisableSyntax.==; strictEquality checks native comparisons
    ) // format: on
  // format: off
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
      phaseType: ClassTag[P], stateEquality: CanEqual[S, S], outcomeEquality: CanEqual[O, O]
  ): Property[S] =
    m.property holdsAcross ((before, after) =>
      !phasing.roleCases[Closed](m.name).contains(phasing.phase(before)) ||
        (after.state == before && after.outcome == rejected) // scalafix:ok DisableSyntax.==; strictEquality checks native comparisons
    ) // format: on
