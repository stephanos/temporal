// A selected timer fires only inside its armed role and settles from before-state retry policy.
package temporal.capabilities

import umpire.*
import scala.annotation.unused
import scala.reflect.{ClassTag, TypeTest}

final case class Deadline[S, P, R, F <: Product](
    timer: ClassRef,
    armed: S => Boolean,
    timeout: F,
    timeoutFacts: Seq[F],
    retryable: Boolean = false,
    retriesRemaining: Option[S => Boolean] = None,
    pendingPause: Option[S => Boolean] = None,
    pendingCancel: Option[S => Boolean] = None
)(using
    @unused phasing: Phasing[S, P],
    @unused finitePhase: Finite[P],
    finiteFact: Finite[F],
    @unused covered: TypeTest[P, R],
    @unused timedOut: TypeTest[P, TimedOut],
    @unused phaseType: ClassTag[P],
    @unused coveredType: ClassTag[R]
) extends CapabilityOf[S, Nothing, F]:
  require(
    retryable == retriesRemaining.nonEmpty,
    "retryable requires exactly one eligibility binding"
  )
  require(
    retryable || (pendingPause.isEmpty && pendingCancel.isEmpty),
    "pending controls require a retryable deadline"
  )
  require(timeoutFacts.nonEmpty, "timeout fact family must be nonempty")
  require(
    timeoutFacts.distinct.size == timeoutFacts.size,
    "timeout fact family contains duplicates"
  )
  require(timeoutFacts.contains(timeout), "timeout fact family must contain the bound timeout")
  require(
    timeoutFacts.toSet == finiteFact.values.filter(_.productPrefix == timeout.productPrefix).toSet,
    "timeout fact family must contain exactly every finite value of the bound timeout constructor"
  )

object Deadline extends CapabilityKind:
  private def noPending[S](@unused state: S): Boolean = false

  def firesInWindow[S, P, R](m: Declares[S])(
      timer: ClassRef,
      armed: S => Boolean
  )(using
      phasing: Phasing[S, P],
      finite: Finite[P],
      covered: TypeTest[P, R],
      phaseType: ClassTag[P],
      coveredType: ClassTag[R]
  ): Property[S] =
    m.property.when(timer) holdsAcross ((before, _) =>
      armed(before) && phasing.roleCases[R](m.name).contains(phasing.phase(before))
    )

  def deadlineTimesOut[S, P](m: Declares[S])(
      timer: ClassRef,
      retryable: Boolean,
      timeout: m.Fact,
      retriesRemaining: S => Boolean = noPending[S],
      pendingCancel: S => Boolean = noPending[S]
  )(using
      phasing: Phasing[S, P],
      finite: Finite[P],
      timedOut: TypeTest[P, TimedOut],
      phaseType: ClassTag[P]
  ): Property[S] =
    m.property.when(timer) holdsAcross ((before, after) =>
      !(pendingCancel(before) || !retryable || !retriesRemaining(before)) ||
        (phasing.roleCases[TimedOut](m.name).contains(phasing.phase(after.state)) &&
          after.records(timeout))
    )

  def deadlineReturnsToWaiting[S, P](m: Declares[S])(
      timer: ClassRef,
      retryable: Boolean,
      retriesRemaining: S => Boolean,
      timeoutFacts: Seq[m.Fact],
      pendingPause: S => Boolean = noPending[S],
      pendingCancel: S => Boolean = noPending[S]
  )(using
      phasing: Phasing[S, P],
      finite: Finite[P],
      waiting: TypeTest[P, Waiting],
      phaseType: ClassTag[P]
  ): Property[S] =
    m.property.when(timer) holdsAcross ((before, after) =>
      !(retryable && !pendingCancel(before) && retriesRemaining(before) && !pendingPause(before)) ||
        (phasing.roleCases[Waiting](m.name).contains(phasing.phase(after.state)) &&
          timeoutFacts.forall(fact => !after.records(fact)))
    )

  def deadlinePauses[S, P](m: Declares[S])(
      timer: ClassRef,
      retryable: Boolean,
      retriesRemaining: S => Boolean,
      pendingPause: S => Boolean,
      timeoutFacts: Seq[m.Fact],
      pendingCancel: S => Boolean = noPending[S]
  )(using
      phasing: Phasing[S, P],
      finite: Finite[P],
      suspended: TypeTest[P, Suspended],
      phaseType: ClassTag[P]
  ): Property[S] =
    m.property.when(timer) holdsAcross ((before, after) =>
      !(retryable && !pendingCancel(before) && retriesRemaining(before) && pendingPause(before)) ||
        (phasing.roleCases[Suspended](m.name).contains(phasing.phase(after.state)) &&
          timeoutFacts.forall(fact => !after.records(fact)))
    )
