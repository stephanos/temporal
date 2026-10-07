// Cancellation requests are recorded while the entity's work is in flight.
package temporal.capabilities

import umpire.*
import umpire.realize.RunExpectation

final case class Cancelable[S, F](
    requestCancel: ClassRef,
    requested: F,
    reach: Seq[ClassRef],
    expect: RunExpectation
) extends CapabilityOf[S, Nothing, F]

object Cancelable extends CapabilityKind:
  // A cancel request records the request and leaves the entity live. Its find starts from `reach`,
  // for the same reason as Terminable.terminateSettles.
  // This does not promise cancellation, which only the work's answer settles, the answer without
  // work in flight (activity cancels at once), or a second request's answer (Nexus answers
  // ErrCancellationAlreadyRequested).
  // See chasm/lib/activity/operator_commands.go, chasm/lib/activity/statemachine.go and
  // chasm/lib/nexusoperation/operation.go.
  def cancelIsRequested[S](m: Declares[S])(
      requestCancel: ClassRef,
      requested: m.Fact
  ): Property[S] =
    m.property when requestCancel holds (_.records(requested))
