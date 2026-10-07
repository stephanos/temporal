package fixture.defaultends

import umpire.*
import temporal.capabilities.*

val closingBounds = Limits(steps = 2, actions = 2, search = 32)

object ClosingEnd extends Machine[Snapshot, Answer, Nothing], Phased[Snapshot, Phase](_.phase):
  val init = DefaultEnd.init
  object rules extends Rules:
    on(hand.touch)(always ~> DefaultEnd.effects.keep)
  object capabilities extends Capabilities:
    val closable: Capability = Closable(rejected = Answer.accepted)
  object queries:
    capabilities.bound(closingBounds)

final case class OtherSnapshot(phase: Phase) derives Finite

object OtherClosing
    extends Machine[OtherSnapshot, Answer, Nothing],
      Phased[OtherSnapshot, Phase](_.phase):
  val init = OtherSnapshot(Phase.waiting)
  object effects:
    def keep(s: State) = stay[OtherSnapshot, Answer, Nothing](s)
  object rules extends Rules:
    on(hand.touch)(always ~> effects.keep)
  object capabilities extends Capabilities:
    val closable: Capability = Closable(rejected = Answer.accepted)
  object queries:
    capabilities.bound(closingBounds)

object ClosingDerived extends Derived(ClosingEnd.unmonitored):
  object capabilities extends Capabilities:
    val closable: Capability = Closable(rejected = Answer.accepted)
  object queries:
    capabilities.bound(closingBounds)

object ClosingPair
    extends Composition[Pair](_.left -> ClosingEnd, _.right -> WrittenEnd),
      Phased[Pair, Phase](_.left.phase):
  object syncs extends Syncs:
    sync(_.left -> hand.touch, _.right -> hand.touch)
  object capabilities extends Capabilities:
    val closable: Capability = Closable(rejected = "left_accepted")
  object queries:
    capabilities.bound(closingBounds)

object ClosingDerivedPair extends DerivedComposition(ClosingPair.withMember(_.right -> DerivedEnd)):
  object capabilities extends Capabilities:
    val closable: Capability = Closable(rejected = "left_accepted")
  object queries:
    capabilities.bound(closingBounds)

object ClosingDerivedAgain
    extends DerivedComposition(ClosingDerivedPair.withMember(_.right -> WrittenEnd)):
  object capabilities extends Capabilities:
    val closable: Capability = Closable(rejected = "left_accepted")
  object queries:
    capabilities.bound(closingBounds)

object ClosingUnphased extends Machine[Snapshot, Answer, Nothing]:
  private given scala.reflect.ClassTag[Nothing] = scala.reflect.ClassTag.Nothing
  val init = DefaultEnd.init
  def end(s: State) = true
  object rules extends Rules:
    on(hand.touch)(always ~> DefaultEnd.effects.keep)
  object capabilities extends Capabilities:
    val closable: Capability = Closable[Snapshot, Nothing, Answer](rejected = Answer.accepted)
  object queries:
    capabilities.bound(closingBounds)

object ClosingOpen
    extends Machine[OpenSnapshot, Answer, Nothing],
      Phased[OpenSnapshot, OpenPhase](_.phase):
  val init = OpenOverride.init
  override def end(s: State) = true
  object rules extends Rules:
    on(hand.touch)(always ~> OpenOverride.effects.keep)
  object capabilities extends Capabilities:
    val closable: Capability = Closable(rejected = Answer.accepted)
  object queries:
    capabilities.bound(closingBounds)
