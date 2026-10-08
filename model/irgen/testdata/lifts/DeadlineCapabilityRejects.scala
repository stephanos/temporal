package fixture.deadlinecapabilityrejects

import umpire.*
import temporal.capabilities.Deadline
import fixture.deadlinecapabilities.*
import fixture.deadlinecapabilities.given

final case class FamilyQuantifier[S, F <: Product](
    timeout: F,
    timeoutFacts: Seq[F],
    positive: Option[S => Boolean] = None,
    doubled: Option[S => Boolean] = None
) extends CapabilityOf[S, Nothing, F]

object FamilyQuantifier extends CapabilityKind:
  def positive[S](m: Declares[S])(timeoutFacts: Seq[m.Fact], positive: S => Boolean): Property[S] =
    m.property holdsAcross ((before, after) =>
      positive(before) && timeoutFacts.forall(fact => after.records(fact))
    )
  def doubled[S](m: Declares[S])(timeoutFacts: Seq[m.Fact], doubled: S => Boolean): Property[S] =
    m.property holdsAcross ((before, after) =>
      doubled(before) && timeoutFacts.forall(fact => !(!after.records(fact)))
    )

object PositiveFamily extends Derived(DeadlineTimers.restrict(startExpired)):
  object capabilities extends Capabilities:
    val family: Capability = FamilyQuantifier[DeadlineState, Fact](
      Fact.timeout(TimeoutType.start),
      timeoutFacts,
      positive = Some(Steps.armed)
    )
  object queries:
    capabilities.bound(three)

object DoubledFamily extends Derived(DeadlineTimers.restrict(startExpired)):
  object capabilities extends Capabilities:
    val family: Capability = FamilyQuantifier[DeadlineState, Fact](
      Fact.timeout(TimeoutType.start),
      timeoutFacts,
      doubled = Some(Steps.armed)
    )
  object queries:
    capabilities.bound(three)

object CompleteSubtypeBinding extends Derived(DeadlineTimers.restrict(startExpired)):
  given Finite[Fact.timeout] =
    Finite.of(Fact.timeout(TimeoutType.schedule), Fact.timeout(TimeoutType.start))
  object capabilities extends Capabilities:
    val deadline: Capability = Deadline[DeadlineState, Phase, Held, Fact.timeout](
      startExpired,
      Steps.armed,
      Fact.timeout(TimeoutType.start),
      Seq(Fact.timeout(TimeoutType.schedule), Fact.timeout(TimeoutType.start))
    )
  object queries:
    capabilities.bound(three)

enum ForeignFact derives Finite:
  case timeout

trait Marker
enum MarkedFact derives Finite:
  case timeout(kind: TimeoutType) extends MarkedFact, Marker
  case unrelated

object CompleteIntersectionBinding
    extends Machine[SimplePhase, Answer, MarkedFact],
      Phased[SimplePhase, SimplePhase](p => p):
  val init = SimplePhase.waiting
  val evidence: PartialFunction[MarkedFact, String] = { case MarkedFact.timeout(_) => "timeout" }
  given Finite[MarkedFact & Marker] =
    Finite.of(MarkedFact.timeout(TimeoutType.schedule), MarkedFact.timeout(TimeoutType.start))
  def armed(s: SimplePhase): Boolean = s == SimplePhase.waiting
  def expire(s: SimplePhase): List[Step[SimplePhase, Answer, MarkedFact]] =
    if armed(s) then enter(SimplePhase.timedOut, MarkedFact.timeout(TimeoutType.start)) else Nil
  object rules extends Bindings(startExpired ~> expire)
  object capabilities extends Capabilities:
    val deadline: Capability = Deadline[SimplePhase, SimplePhase, Waiting, MarkedFact & Marker](
      startExpired,
      armed,
      MarkedFact.timeout(TimeoutType.start),
      Seq(MarkedFact.timeout(TimeoutType.schedule), MarkedFact.timeout(TimeoutType.start))
    )
  object queries:
    capabilities.bound(three)

object IncompleteIntersectionBinding
    extends Derived(CompleteIntersectionBinding.restrict(startExpired)):
  given Finite[MarkedFact & Marker] = Finite.of(MarkedFact.timeout(TimeoutType.start))
  object capabilities extends Capabilities:
    val deadline: Capability = Deadline[SimplePhase, SimplePhase, Waiting, MarkedFact & Marker](
      startExpired,
      CompleteIntersectionBinding.armed,
      MarkedFact.timeout(TimeoutType.start),
      Seq(MarkedFact.timeout(TimeoutType.start))
    )
  object queries:
    capabilities.bound(three)

object IncompleteSubtypeBinding extends Derived(DeadlineTimers.restrict(startExpired)):
  given Finite[Fact.timeout] = Finite.of(Fact.timeout(TimeoutType.start))
  object capabilities extends Capabilities:
    val deadline: Capability = Deadline[DeadlineState, Phase, Held, Fact.timeout](
      startExpired,
      Steps.armed,
      Fact.timeout(TimeoutType.start),
      Seq(Fact.timeout(TimeoutType.start))
    )
  object queries:
    capabilities.bound(three)

object WrongTypedFamily extends Derived(DeadlineTimers.restrict(startExpired)):
  object capabilities extends Capabilities:
    val deadline: Capability = Deadline[DeadlineState, Phase, Held, Fact](
      startExpired,
      Steps.armed,
      Fact.timeout(TimeoutType.start),
      Seq(
        Fact.timeout(TimeoutType.schedule),
        Fact.timeout(TimeoutType.start),
        ForeignFact.timeout.asInstanceOf[Fact]
      )
    )
  object queries:
    capabilities.bound(three)

object NoPhase extends Machine[DeadlineState, Answer, Fact]:
  import DeadlineTimers.given
  val init = DeadlineTimers.init
  def end(s: DeadlineState): Boolean = s.phase == Phase.timedOut
  val evidence: PartialFunction[Fact, String] = { case Fact.timeout(_) => "timeout" }
  object rules extends Bindings(startExpired ~> Steps.start)
  object capabilities extends Capabilities:
    val deadline: Capability = Deadline[DeadlineState, Phase, Held, Fact](
      startExpired,
      Steps.armed,
      Fact.timeout(TimeoutType.start),
      timeoutFacts
    )
  object queries:
    capabilities.bound(three)

object NoCovered extends Derived(SimpleDeadlines.restrict(startExpired)):
  object capabilities extends Capabilities:
    val deadline: Capability = Deadline[SimplePhase, SimplePhase, Held, Fact](
      startExpired,
      SimpleDeadlines.armed,
      Fact.timeout(TimeoutType.start),
      timeoutFacts
    )
  object queries:
    capabilities.bound(three)

enum WithoutTimedOut derives Finite:
  case waiting extends WithoutTimedOut, Waiting
  case plain

object NoTimedOut
    extends Machine[WithoutTimedOut, Answer, Fact],
      Phased[WithoutTimedOut, WithoutTimedOut](p => p):
  val init = WithoutTimedOut.waiting
  override def end(s: WithoutTimedOut): Boolean = s == WithoutTimedOut.plain
  val evidence: PartialFunction[Fact, String] = { case Fact.timeout(_) => "timeout" }
  def armed(s: WithoutTimedOut): Boolean = s == WithoutTimedOut.waiting
  def keep(s: WithoutTimedOut): List[Step[WithoutTimedOut, Answer, Fact]] = stay(s)
  object rules extends Bindings(startExpired ~> keep)
  object capabilities extends Capabilities:
    val deadline: Capability = Deadline[WithoutTimedOut, WithoutTimedOut, Waiting, Fact](
      startExpired,
      armed,
      Fact.timeout(TimeoutType.start),
      timeoutFacts
    )
  object queries:
    capabilities.bound(three)

enum WithoutWaiting derives Finite:
  case held extends WithoutWaiting, Held
  case timedOut extends WithoutWaiting, TimedOut

object NativeTerminalRoles
    extends Machine[WithoutWaiting, Answer, Fact],
      Phased[WithoutWaiting, WithoutWaiting](p => p):
  val init = WithoutWaiting.held
  val evidence: PartialFunction[Fact, String] = { case Fact.timeout(_) => "timeout" }
  def armed(s: WithoutWaiting): Boolean = s == WithoutWaiting.held
  def expire(s: WithoutWaiting): List[Step[WithoutWaiting, Answer, Fact]] =
    if armed(s) then enter(WithoutWaiting.timedOut, Fact.timeout(TimeoutType.start)) else Nil
  object rules extends Bindings(startExpired ~> expire)
  object capabilities extends Capabilities:
    val deadline: Capability = Deadline[WithoutWaiting, WithoutWaiting, Held, Fact](
      startExpired,
      armed,
      Fact.timeout(TimeoutType.start),
      timeoutFacts
    )
  object queries:
    capabilities.bound(three)

object NoWaiting
    extends Machine[WithoutWaiting, Answer, Fact],
      Phased[WithoutWaiting, WithoutWaiting](p => p):
  val init = WithoutWaiting.held
  val evidence: PartialFunction[Fact, String] = { case Fact.timeout(_) => "timeout" }
  def armed(s: WithoutWaiting): Boolean = s == WithoutWaiting.held
  def keep(s: WithoutWaiting): List[Step[WithoutWaiting, Answer, Fact]] = stay(s)
  object rules extends Bindings(startExpired ~> keep)
  object capabilities extends Capabilities:
    val deadline: Capability = Deadline[WithoutWaiting, WithoutWaiting, Held, Fact](
      startExpired,
      armed,
      Fact.timeout(TimeoutType.start),
      timeoutFacts,
      retryable = true,
      retriesRemaining = Some(armed)
    )
  object queries:
    capabilities.bound(three)

object NoSuspended extends Derived(SimpleDeadlines.restrict(startExpired)):
  object capabilities extends Capabilities:
    val deadline: Capability = Deadline[SimplePhase, SimplePhase, Waiting, Fact](
      startExpired,
      SimpleDeadlines.armed,
      Fact.timeout(TimeoutType.start),
      timeoutFacts,
      retryable = true,
      retriesRemaining = Some(SimpleDeadlines.remaining),
      pendingPause = Some(SimpleDeadlines.armed)
    )
  object queries:
    capabilities.bound(three)

object ForeignTimer extends Derived(DeadlineTimers.restrict(startExpired)):
  object capabilities extends Capabilities:
    val deadline: Capability = Deadline[DeadlineState, Phase, Held, Fact](
      foreign.expired,
      Steps.armed,
      Fact.timeout(TimeoutType.start),
      timeoutFacts
    )
  object queries:
    capabilities.bound(three)

object UnboundTimer extends Derived(DeadlineTimers.restrict(startExpired)):
  object capabilities extends Capabilities:
    val deadline: Capability = Deadline[DeadlineState, Phase, Held, Fact](
      unboundExpired,
      Steps.armed,
      Fact.timeout(TimeoutType.start),
      timeoutFacts
    )
  object queries:
    capabilities.bound(three)

object NonTimer extends Derived(DeadlineTimers.restrict(observer.keep)):
  object capabilities extends Capabilities:
    val deadline: Capability = Deadline[DeadlineState, Phase, Held, Fact](
      observer.keep,
      Steps.armed,
      Fact.timeout(TimeoutType.start),
      timeoutFacts
    )
  object queries:
    capabilities.bound(three)

object RetryNoEligibility extends Derived(DeadlineTimers.restrict(startExpired)):
  object capabilities extends Capabilities:
    val deadline: Capability = Deadline[DeadlineState, Phase, Held, Fact](
      startExpired,
      Steps.armed,
      Fact.timeout(TimeoutType.start),
      timeoutFacts,
      retryable = true
    )
  object queries:
    capabilities.bound(three)

object EligibilityNonretry extends Derived(DeadlineTimers.restrict(startExpired)):
  object capabilities extends Capabilities:
    val deadline: Capability = Deadline[DeadlineState, Phase, Held, Fact](
      startExpired,
      Steps.armed,
      Fact.timeout(TimeoutType.start),
      timeoutFacts,
      retriesRemaining = Some(Steps.remaining)
    )
  object queries:
    capabilities.bound(three)

object ControlNonretry extends Derived(DeadlineTimers.restrict(startExpired)):
  object capabilities extends Capabilities:
    val deadline: Capability = Deadline[DeadlineState, Phase, Held, Fact](
      startExpired,
      Steps.armed,
      Fact.timeout(TimeoutType.start),
      timeoutFacts,
      pendingCancel = Some(Steps.pendingCancel)
    )
  object queries:
    capabilities.bound(three)

object UnnamedArmed extends Derived(DeadlineTimers.restrict(startExpired)):
  object capabilities extends Capabilities:
    val deadline: Capability = Deadline[DeadlineState, Phase, Held, Fact](
      startExpired,
      s => s.armed,
      Fact.timeout(TimeoutType.start),
      timeoutFacts
    )
  object queries:
    capabilities.bound(three)

object UnnamedPause extends Derived(DeadlineTimers.restrict(startExpired)):
  object capabilities extends Capabilities:
    val deadline: Capability = Deadline[DeadlineState, Phase, Held, Fact](
      startExpired,
      Steps.armed,
      Fact.timeout(TimeoutType.start),
      timeoutFacts,
      retryable = true,
      retriesRemaining = Some(Steps.remaining),
      pendingPause = Some(s => s.pause)
    )
  object queries:
    capabilities.bound(three)

object MalformedEligibility extends Derived(DeadlineTimers.restrict(startExpired)):
  def selected: Option[DeadlineState => Boolean] = Some(Steps.remaining)
  object capabilities extends Capabilities:
    val deadline: Capability = Deadline[DeadlineState, Phase, Held, Fact](
      startExpired,
      Steps.armed,
      Fact.timeout(TimeoutType.start),
      timeoutFacts,
      retryable = true,
      retriesRemaining = selected
    )
  object queries:
    capabilities.bound(three)

object ComputedClassification extends Derived(DeadlineTimers.restrict(startExpired)):
  object capabilities extends Capabilities:
    val deadline: Capability = Deadline[DeadlineState, Phase, Held, Fact](
      startExpired,
      Steps.armed,
      Fact.timeout(TimeoutType.start),
      timeoutFacts,
      retryable = Steps.armed(init),
      retriesRemaining = Some(Steps.remaining)
    )
  object queries:
    capabilities.bound(three)

object EmptyFamily extends Derived(DeadlineTimers.restrict(startExpired)):
  object capabilities extends Capabilities:
    val deadline: Capability = Deadline[DeadlineState, Phase, Held, Fact](
      startExpired,
      Steps.armed,
      Fact.timeout(TimeoutType.start),
      Seq()
    )
  object queries:
    capabilities.bound(three)

object IncompleteFamily extends Derived(DeadlineTimers.restrict(startExpired)):
  object capabilities extends Capabilities:
    val deadline: Capability = Deadline[DeadlineState, Phase, Held, Fact](
      startExpired,
      Steps.armed,
      Fact.timeout(TimeoutType.start),
      Seq(Fact.timeout(TimeoutType.start))
    )
  object queries:
    capabilities.bound(three)

object DuplicateFamily extends Derived(DeadlineTimers.restrict(startExpired)):
  object capabilities extends Capabilities:
    val deadline: Capability = Deadline[DeadlineState, Phase, Held, Fact](
      startExpired,
      Steps.armed,
      Fact.timeout(TimeoutType.start),
      Seq(
        Fact.timeout(TimeoutType.schedule),
        Fact.timeout(TimeoutType.start),
        Fact.timeout(TimeoutType.start)
      )
    )
  object queries:
    capabilities.bound(three)

object MixedFamily extends Derived(DeadlineTimers.restrict(startExpired)):
  object capabilities extends Capabilities:
    val deadline: Capability = Deadline[DeadlineState, Phase, Held, Fact](
      startExpired,
      Steps.armed,
      Fact.timeout(TimeoutType.start),
      Seq(Fact.timeout(TimeoutType.schedule), Fact.timeout(TimeoutType.start), Fact.unrelated)
    )
  object queries:
    capabilities.bound(three)

object WrongFamily extends Derived(DeadlineTimers.restrict(startExpired)):
  object capabilities extends Capabilities:
    val deadline: Capability = Deadline[DeadlineState, Phase, Held, Fact](
      startExpired,
      Steps.armed,
      Fact.timeout(TimeoutType.start),
      Seq(Fact.unrelated)
    )
  object queries:
    capabilities.bound(three)

object MalformedFamily extends Derived(DeadlineTimers.restrict(startExpired)):
  def selected: Seq[Fact] = timeoutFacts
  object capabilities extends Capabilities:
    val deadline: Capability = Deadline[DeadlineState, Phase, Held, Fact](
      startExpired,
      Steps.armed,
      Fact.timeout(TimeoutType.start),
      selected
    )
  object queries:
    capabilities.bound(three)

object NonconstantTimeout extends Derived(DeadlineTimers.restrict(startExpired)):
  def selected: Fact = Fact.timeout(TimeoutType.start)
  object capabilities extends Capabilities:
    val deadline: Capability = Deadline[DeadlineState, Phase, Held, Fact](
      startExpired,
      Steps.armed,
      selected,
      timeoutFacts
    )
  object queries:
    capabilities.bound(three)

object DuplicateTimeout extends Derived(DeadlineTimers.restrict(startExpired)):
  object capabilities extends Capabilities:
    val first: Capability = Deadline[DeadlineState, Phase, Held, Fact](
      startExpired,
      Steps.armed,
      Fact.timeout(TimeoutType.start),
      timeoutFacts
    )
    val second: Capability = Deadline[DeadlineState, Phase, Held, Fact](
      startExpired,
      Steps.armed,
      Fact.timeout(TimeoutType.start),
      timeoutFacts
    )
  object queries:
    capabilities.bound(three)

object AmbiguousWaiver extends Derived(DeadlineTimers.restrict(startExpired, scheduleExpired)):
  object capabilities extends Capabilities:
    val first: Capability = Deadline[DeadlineState, Phase, Held, Fact](
      startExpired,
      Steps.armed,
      Fact.timeout(TimeoutType.start),
      timeoutFacts
    )
    val second: Capability = Deadline[DeadlineState, Phase, Waiting, Fact](
      scheduleExpired(true),
      Steps.dispatchArmed,
      Fact.timeout(TimeoutType.schedule),
      timeoutFacts
    )
    except(Deadline.deadlineTimesOut[DeadlineState, Phase], because = "ambiguous fixture waiver")
  object queries:
    capabilities.bound(three)

object AmbiguousClaim extends Derived(DeadlineTimers.restrict(startExpired, scheduleExpired)):
  object capabilities extends Capabilities:
    val first: Capability = Deadline[DeadlineState, Phase, Held, Fact](
      startExpired,
      Steps.armed,
      Fact.timeout(TimeoutType.start),
      timeoutFacts
    )
    val second: Capability = Deadline[DeadlineState, Phase, Waiting, Fact](
      scheduleExpired(true),
      Steps.dispatchArmed,
      Fact.timeout(TimeoutType.schedule),
      timeoutFacts
    )
  object queries:
    capabilities.bound(three)
    val any = scenario.free
    val ambiguous = query verify capabilities.claim(
      Deadline.deadlineTimesOut[DeadlineState, Phase]
    ) in any limits three
