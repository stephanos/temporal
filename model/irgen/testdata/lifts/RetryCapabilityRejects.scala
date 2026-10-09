package fixture.retrycapabilityrejects

import umpire.*
import temporal.capabilities.Retries
import fixture.retrycapabilities.*
import fixture.retrycapabilities.given

object NoPhase extends Machine[WaitingState, Answer, Nothing]:
  import WaitingFailures.given
  val init = WaitingState(SimplePhase.waiting, UpTo(0), UpTo(0))
  def end(s: WaitingState): Boolean = s.phase == SimplePhase.failed
  object rules extends Bindings(worker.fail ~> WaitingSteps.fail)
  object capabilities extends Capabilities:
    val retries: Capability = Retries[WaitingState, SimplePhase, 2](
      worker.fail,
      true,
      WaitingSteps.attempts,
      WaitingSteps.maxOne,
      WaitingSteps.remainingOne
    )
  object queries:
    capabilities.bound(three)

def zero[S](s: S): Int = 0
def maximum[S](s: S): Option[UpTo[2]] = Some(UpTo[2](1))
def remaining[S](s: S): Boolean = true

enum WithoutWaiting derives Finite:
  case plain
  case failed extends WithoutWaiting, Failed

object NoWaiting
    extends Machine[WithoutWaiting, Answer, Nothing],
      Phased[WithoutWaiting, WithoutWaiting](p => p):
  val init = WithoutWaiting.plain
  def keep(s: WithoutWaiting): List[Step[WithoutWaiting, Answer, Nothing]] = stay(s)
  object rules extends Bindings(worker.fail ~> keep)
  object capabilities extends Capabilities:
    val retries: Capability = Retries(worker.fail, true, zero, maximum, remaining)
  object queries:
    capabilities.bound(three)

enum WithoutFailed derives Finite:
  case waiting extends WithoutFailed, Waiting
  case plain

object NoFailed
    extends Machine[WithoutFailed, Answer, Nothing],
      Phased[WithoutFailed, WithoutFailed](p => p):
  val init = WithoutFailed.waiting
  override def end(s: WithoutFailed): Boolean = s == WithoutFailed.plain
  def keep(s: WithoutFailed): List[Step[WithoutFailed, Answer, Nothing]] = stay(s)
  object rules extends Bindings(worker.fail ~> keep)
  object capabilities extends Capabilities:
    val retries: Capability = Retries(worker.fail, true, zero, maximum, remaining)
  object queries:
    capabilities.bound(three)

object NoSuspended
    extends Machine[WaitingState, Answer, Nothing],
      Phased[WaitingState, SimplePhase](_.phase):
  val init = WaitingState(SimplePhase.waiting, UpTo(0), UpTo(0))
  object rules extends Bindings(worker.fail ~> WaitingSteps.fail)
  object capabilities extends Capabilities:
    val retries: Capability = Retries(
      worker.fail,
      true,
      WaitingSteps.attempts,
      WaitingSteps.maxOne,
      WaitingSteps.remainingOne,
      pendingPause = Some(WaitingSteps.pending)
    )
  object queries:
    capabilities.bound(three)

object NoCanceled
    extends Machine[WaitingState, Answer, Nothing],
      Phased[WaitingState, SimplePhase](_.phase):
  val init = WaitingState(SimplePhase.waiting, UpTo(0), UpTo(0))
  object rules extends Bindings(worker.fail ~> WaitingSteps.fail)
  object capabilities extends Capabilities:
    val retries: Capability = Retries(
      worker.fail,
      true,
      WaitingSteps.attempts,
      WaitingSteps.maxOne,
      WaitingSteps.remainingOne,
      pendingCancel = Some(WaitingSteps.pending)
    )
  object queries:
    capabilities.bound(three)

object UnboundFailure
    extends Machine[WaitingState, Answer, Nothing],
      Phased[WaitingState, SimplePhase](_.phase):
  val init = WaitingState(SimplePhase.waiting, UpTo(0), UpTo(0))
  object rules extends Bindings(worker.fail ~> WaitingSteps.fail)
  object capabilities extends Capabilities:
    val retries: Capability = Retries(
      worker.foreignFailure,
      true,
      WaitingSteps.attempts,
      WaitingSteps.maxOne,
      WaitingSteps.remainingOne
    )
  object queries:
    capabilities.bound(three)

object UnnamedCount
    extends Machine[WaitingState, Answer, Nothing],
      Phased[WaitingState, SimplePhase](_.phase):
  val init = WaitingState(SimplePhase.waiting, UpTo(0), UpTo(0))
  object rules extends Bindings(worker.fail ~> WaitingSteps.fail)
  object capabilities extends Capabilities:
    val retries: Capability = Retries(
      worker.fail,
      true,
      s => s.attempts + 1,
      WaitingSteps.maxOne,
      WaitingSteps.remainingOne
    )
  object queries:
    capabilities.bound(three)

object UnnamedPause
    extends Machine[ControlState, Answer, Nothing],
      Phased[ControlState, ControlPhase](_.phase):
  val init = ControlState(ControlPhase.waiting, UpTo(0), false, false)
  object rules extends Bindings(worker.fail ~> ControlSteps.fail)
  object capabilities extends Capabilities:
    val retries: Capability = Retries(
      worker.fail,
      true,
      ControlSteps.attempts,
      ControlSteps.maxOne,
      ControlSteps.remainingOne,
      pendingPause = Some(s => s.attempts > 0)
    )
  object queries:
    capabilities.bound(three)

object MalformedPause
    extends Machine[WaitingState, Answer, Nothing],
      Phased[WaitingState, SimplePhase](_.phase):
  val init = WaitingState(SimplePhase.waiting, UpTo(0), UpTo(0))
  def selectedPause: Option[WaitingState => Boolean] = Some(WaitingSteps.pending)
  object rules extends Bindings(worker.fail ~> WaitingSteps.fail)
  object capabilities extends Capabilities:
    val retries: Capability = Retries(
      worker.fail,
      true,
      WaitingSteps.attempts,
      WaitingSteps.maxOne,
      WaitingSteps.remainingOne,
      pendingPause = selectedPause
    )
  object queries:
    capabilities.bound(three)

object ComputedClassification
    extends Machine[WaitingState, Answer, Nothing],
      Phased[WaitingState, SimplePhase](_.phase):
  val init = WaitingState(SimplePhase.waiting, UpTo(0), UpTo(0))
  object rules extends Bindings(worker.fail ~> WaitingSteps.fail)
  object capabilities extends Capabilities:
    val retries: Capability = Retries(
      worker.fail,
      WaitingSteps.pending(init),
      WaitingSteps.attempts,
      WaitingSteps.maxOne,
      WaitingSteps.remainingOne
    )
  object queries:
    capabilities.bound(three)

object AmbiguousWaiver
    extends Machine[WaitingState, Answer, Nothing],
      Phased[WaitingState, SimplePhase](_.phase):
  val init = WaitingState(SimplePhase.waiting, UpTo(0), UpTo(0))
  object rules
      extends Bindings(worker.fail ~> WaitingSteps.fail, network.fault ~> WaitingSteps.fault)
  object capabilities extends Capabilities:
    val first: Capability = Retries(
      worker.fail,
      true,
      WaitingSteps.attempts,
      WaitingSteps.maxOne,
      WaitingSteps.remainingOne
    )
    val second: Capability = Retries(
      network.fault,
      true,
      WaitingSteps.otherAttempts,
      WaitingSteps.unlimited,
      WaitingSteps.remainingUnlimited
    )
    except(
      Retries.failureEndsFailed[WaitingState, SimplePhase],
      because = "a fixture's ambiguous waiver"
    )
  object queries:
    capabilities.bound(three)

object AmbiguousClaim
    extends Machine[WaitingState, Answer, Nothing],
      Phased[WaitingState, SimplePhase](_.phase):
  val init = WaitingState(SimplePhase.waiting, UpTo(0), UpTo(0))
  object rules
      extends Bindings(worker.fail ~> WaitingSteps.fail, network.fault ~> WaitingSteps.fault)
  object capabilities extends Capabilities:
    val first: Capability = Retries(
      worker.fail,
      true,
      WaitingSteps.attempts,
      WaitingSteps.maxOne,
      WaitingSteps.remainingOne
    )
    val second: Capability = Retries(
      network.fault,
      true,
      WaitingSteps.otherAttempts,
      WaitingSteps.unlimited,
      WaitingSteps.remainingUnlimited
    )
  object queries:
    capabilities.bound(three)
    val start = scenario.free
    val ambiguous = query verify capabilities.claim(
      Retries.failureEndsFailed[WaitingState, SimplePhase]
    ) in start limits three

final case class Foreign[S](own: S => Boolean, pending: Option[S => Boolean] = None)
    extends CapabilityOf[S, Nothing, Nothing]

object Foreign extends CapabilityKind:
  def readsForeign[S](m: Declares[S])(own: S => Boolean, shared: S => Boolean): Property[S] =
    m.property holds (after => own(after.state) && shared(after.state))

final case class First[S](shared: S => Boolean) extends CapabilityOf[S, Nothing, Nothing]
object First extends CapabilityKind
final case class Second[S](shared: S => Boolean) extends CapabilityOf[S, Nothing, Nothing]
object Second extends CapabilityKind

object AmbiguousForeign
    extends Machine[WaitingState, Answer, Nothing],
      Phased[WaitingState, SimplePhase](_.phase):
  val init = WaitingState(SimplePhase.waiting, UpTo(0), UpTo(0))
  object rules extends Bindings(worker.fail ~> WaitingSteps.fail)
  object capabilities extends Capabilities:
    val own: Capability = Foreign(own = WaitingSteps.remainingOne)
    val first: Capability = First(shared = WaitingSteps.remainingOne)
    val second: Capability = Second(shared = WaitingSteps.pending)
  object queries:
    capabilities.bound(three)

def controlEndsFailed(m: Declares[ControlState])(
    failure: ClassRef,
    retryable: Boolean,
    retriesRemaining: ControlState => Boolean,
    pendingCancel: ControlState => Boolean
): Property[ControlState] =
  m.property.when(failure) holdsAcross ((before, after) =>
    retryable || after.state.phase == ControlPhase.failed
  )

def controlPauses(m: Declares[ControlState])(
    failure: ClassRef,
    retryable: Boolean,
    retriesRemaining: ControlState => Boolean,
    pendingPause: ControlState => Boolean,
    pendingCancel: ControlState => Boolean
): Property[ControlState] =
  m.property.when(failure) holdsAcross ((before, after) =>
    after.state.phase == ControlPhase.waiting
  )

// Two Retries instances; only the first binds a pending pause, so it alone brings failurePauses.
abstract class TwoFailures(using
    Declaring[ControlState, Answer, Nothing],
    Phasing[ControlState, ControlPhase]
) extends Capabilities:
  val first: Capability = Retries(
    worker.fail,
    true,
    ControlSteps.attempts,
    ControlSteps.maxOne,
    ControlSteps.remainingOne,
    pendingPause = Some(ControlSteps.pendingPause)
  )
  val second: Capability = Retries(
    worker.fatalFailure,
    false,
    ControlSteps.attempts,
    ControlSteps.maxOne,
    ControlSteps.remainingOne
  )

object UnknownInstance
    extends Machine[ControlState, Answer, Nothing],
      Phased[ControlState, ControlPhase](_.phase):
  val init = ControlState(ControlPhase.waiting, UpTo(0), false, false)
  object rules
      extends Bindings(worker.fail ~> ControlSteps.fail, worker.fatalFailure ~> ControlSteps.fatal)
  object capabilities extends TwoFailures:
    overriding(
      Retries.failureEndsFailed[ControlState, ControlPhase] -> controlEndsFailed,
      because = "a fixture's unknown instance",
      of = Seq(ControlledFailures.capabilities.eligible)
    )
  object queries:
    capabilities.bound(three)

object NonMatchingInstance
    extends Machine[ControlState, Answer, Nothing],
      Phased[ControlState, ControlPhase](_.phase):
  val init = ControlState(ControlPhase.waiting, UpTo(0), false, false)
  object rules
      extends Bindings(worker.fail ~> ControlSteps.fail, worker.fatalFailure ~> ControlSteps.fatal)
  object capabilities extends TwoFailures:
    overriding(
      Retries.failurePauses[ControlState, ControlPhase] -> controlPauses,
      because = "a fixture's non-matching instance",
      of = Seq(second)
    )
  object queries:
    capabilities.bound(three)

object DuplicateInstance
    extends Machine[ControlState, Answer, Nothing],
      Phased[ControlState, ControlPhase](_.phase):
  val init = ControlState(ControlPhase.waiting, UpTo(0), false, false)
  object rules
      extends Bindings(worker.fail ~> ControlSteps.fail, worker.fatalFailure ~> ControlSteps.fatal)
  object capabilities extends TwoFailures:
    overriding(
      Retries.failureEndsFailed[ControlState, ControlPhase] -> controlEndsFailed,
      because = "a fixture's duplicate instance",
      of = Seq(first, first)
    )
  object queries:
    capabilities.bound(three)

object EmptySelection
    extends Machine[ControlState, Answer, Nothing],
      Phased[ControlState, ControlPhase](_.phase):
  val init = ControlState(ControlPhase.waiting, UpTo(0), false, false)
  object rules
      extends Bindings(worker.fail ~> ControlSteps.fail, worker.fatalFailure ~> ControlSteps.fatal)
  object capabilities extends TwoFailures:
    overriding(
      Retries.failureEndsFailed[ControlState, ControlPhase] -> controlEndsFailed,
      because = "a fixture's empty selection",
      of = Seq()
    )
  object queries:
    capabilities.bound(three)

object RedundantSelection
    extends Machine[ControlState, Answer, Nothing],
      Phased[ControlState, ControlPhase](_.phase):
  val init = ControlState(ControlPhase.waiting, UpTo(0), false, false)
  object rules
      extends Bindings(worker.fail ~> ControlSteps.fail, worker.fatalFailure ~> ControlSteps.fatal)
  object capabilities extends TwoFailures:
    overriding(
      Retries.failurePauses[ControlState, ControlPhase] -> controlPauses,
      because = "a fixture's redundant selection",
      of = Seq(first)
    )
  object queries:
    capabilities.bound(three)
