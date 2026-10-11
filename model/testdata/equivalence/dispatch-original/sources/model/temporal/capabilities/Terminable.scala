// Terminating a live entity settles it in the same step.
package temporal.capabilities

import framework.*
import framework.realize.RunExpectation

final case class Terminable[S, F](
    terminate: ClassRef,
    settled: F,
    reach: Seq[ClassRef],
    expect: RunExpectation
) extends CapabilityOf[S, Nothing, F]

object Terminable extends CapabilityKind:
  // A terminate of a live entity settles it as terminated in one step and records that.
  // Its find starts from the live state reached by `reach`: a terminate of a closed entity is
  // Closable's to answer, and a transition Property restricted by `when` is unsupported.
  // This promises neither the answer after close or to a second terminate, nor recording a reason
  // and identity (the schedule records only close). Activity answers a repeated request id OK;
  // Nexus also refuses another id by name.
  // See chasm/lib/activity/activity.go, chasm/lib/activity/statemachine.go and
  // chasm/lib/nexusoperation/operation_statemachine.go.
  def terminateSettles[S](m: Declares[S])(terminate: ClassRef, settled: m.Fact): Property[S] =
    m.property when terminate holds (_.records(settled))
