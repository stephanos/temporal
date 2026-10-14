// Failure settlement and the entity's retry policy. Eligibility reads the state before failure.
package temporal.capabilities

import framework.*
import scala.annotation.unused
import scala.reflect.{ClassTag, TypeTest}

final case class Retries[S, P, N <: Int](
    failure: ClassRef,
    retryable: Boolean,
    attemptCount: S => Int,
    maximumAttempts: S => Option[UpTo[N]],
    retriesRemaining: S => Boolean,
    pendingPause: Option[S => Boolean] = None,
    pendingCancel: Option[S => Boolean] = None
)(using
    @unused phasing: Phasing[S, P],
    @unused finite: Finite[P],
    @unused waiting: TypeTest[P, Waiting],
    @unused failed: TypeTest[P, Failed],
    @unused phaseType: ClassTag[P]
) extends CapabilityOf[S, Nothing, Nothing]

object Retries extends CapabilityKind:
  type Waiting = framework.Waiting
  type Failed = framework.Failed

  private def noPending[S](@unused state: S): Boolean = false

  // Retryable cancellation takes priority over exhaustion and pause; fatal failures stay Failed.
  // See chasm/lib/activity/statemachine.go and chasm/lib/nexusoperation/operation_statemachine.go.
  def failureReturnsToWaiting[S, P](m: Declares[S])(
      failure: ClassRef,
      retryable: Boolean,
      retriesRemaining: S => Boolean,
      pendingPause: S => Boolean = noPending[S],
      pendingCancel: S => Boolean = noPending[S]
  )(using
      phasing: Phasing[S, P],
      finite: Finite[P],
      waiting: TypeTest[P, Waiting],
      phaseType: ClassTag[P]
  ): Property[S] =
    m.property.when(failure) holdsAcross ((before, after) =>
      !(retryable && !pendingCancel(before) && retriesRemaining(before) && !pendingPause(before)) ||
        phasing.roleCases[Waiting](m.name).contains(phasing.phase(after.state))
    )

  def failureEndsFailed[S, P](m: Declares[S])(
      failure: ClassRef,
      retryable: Boolean,
      retriesRemaining: S => Boolean,
      pendingCancel: S => Boolean = noPending[S]
  )(using
      phasing: Phasing[S, P],
      finite: Finite[P],
      failed: TypeTest[P, Failed],
      phaseType: ClassTag[P]
  ): Property[S] =
    m.property.when(failure) holdsAcross ((before, after) =>
      !(!retryable || (!pendingCancel(before) && !retriesRemaining(before))) ||
        phasing.roleCases[Failed](m.name).contains(phasing.phase(after.state))
    )

  def failurePauses[S, P](m: Declares[S])(
      failure: ClassRef,
      retryable: Boolean,
      retriesRemaining: S => Boolean,
      pendingPause: S => Boolean,
      pendingCancel: S => Boolean = noPending[S]
  )(using
      phasing: Phasing[S, P],
      finite: Finite[P],
      suspended: TypeTest[P, Suspended],
      phaseType: ClassTag[P]
  ): Property[S] =
    m.property.when(failure) holdsAcross ((before, after) =>
      !(retryable && !pendingCancel(before) && retriesRemaining(before) && pendingPause(before)) ||
        phasing.roleCases[Suspended](m.name).contains(phasing.phase(after.state))
    )

  def failureCancels[S, P](m: Declares[S])(
      failure: ClassRef,
      retryable: Boolean,
      pendingCancel: S => Boolean
  )(using
      phasing: Phasing[S, P],
      finite: Finite[P],
      canceled: TypeTest[P, Canceled],
      phaseType: ClassTag[P]
  ): Property[S] =
    m.property.when(failure) holdsAcross ((before, after) =>
      !(retryable && pendingCancel(before)) ||
        phasing.roleCases[Canceled](m.name).contains(phasing.phase(after.state))
    )

  // None is explicit unlimited policy. A finite maximum is independent of the count's domain.
  def attemptCountIsWithinPolicy[S, N <: Int](m: Declares[S])(
      attemptCount: S => Int,
      maximumAttempts: S => Option[UpTo[N]]
  ): Property[S] =
    m.property holds (after =>
      maximumAttempts(after.state) match
        case None          => true
        case Some(maximum) => attemptCount(after.state) <= maximum
    )
