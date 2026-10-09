package fixture.defaultends

import framework.*

enum Phase derives Finite:
  case waiting extends Phase, Waiting
  case done extends Phase, Succeeded
  case failed extends Phase, Failed

enum OpenPhase derives Finite:
  case waiting extends OpenPhase, Waiting

enum Answer derives Finite:
  case accepted

final case class Snapshot(phase: Phase) derives Finite
final case class OpenSnapshot(phase: OpenPhase) derives Finite
final case class Pair(left: Snapshot, right: Snapshot)
final case class OpenPair(left: OpenSnapshot, right: OpenSnapshot)

given Ok[Answer] = Ok(Answer.accepted)

object hand extends Actor:
  val touch = action(this)

object DefaultEnd extends Machine[Snapshot, Answer, Nothing], Phased[Snapshot, Phase](_.phase):
  val init = Snapshot(Phase.waiting)
  object effects:
    def keep(s: State) = stay[Snapshot, Answer, Nothing](s)
  object rules extends Rules:
    on(hand.touch)(always ~> effects.keep)

object WrittenEnd extends Machine[Snapshot, Answer, Nothing], Phased[Snapshot, Phase](_.phase):
  val init = DefaultEnd.init
  override def end(s: State) = s.phase.in[Closed]
  object rules extends Rules:
    on(hand.touch)(always ~> DefaultEnd.effects.keep)

object DerivedEnd extends Derived(DefaultEnd.unmonitored)

object DefaultPair
    extends Composition[Pair](_.left -> DefaultEnd, _.right -> DefaultEnd),
      Phased[Pair, Phase](_.left.phase):
  object syncs extends Syncs:
    sync(_.left -> hand.touch, _.right -> hand.touch)

object DerivedPair extends Composition(DefaultPair.withMember(_.right -> DerivedEnd))

object OpenOverride
    extends Machine[OpenSnapshot, Answer, Nothing],
      Phased[OpenSnapshot, OpenPhase](_.phase):
  val init = OpenSnapshot(OpenPhase.waiting)
  override def end(s: State) = s.phase == OpenPhase.waiting
  object effects:
    def keep(s: State) = stay[OpenSnapshot, Answer, Nothing](s)
  object rules extends Rules:
    on(hand.touch)(always ~> effects.keep)

object OpenEnd
    extends Machine[OpenSnapshot, Answer, Nothing],
      Phased[OpenSnapshot, OpenPhase](_.phase):
  val init = OpenOverride.init
  object rules extends Rules:
    on(hand.touch)(always ~> OpenOverride.effects.keep)

object OpenDefaultPair
    extends Composition[OpenPair](_.left -> OpenOverride, _.right -> OpenOverride),
      Phased[OpenPair, OpenPhase](_.left.phase):
  object syncs extends Syncs:
    sync(_.left -> hand.touch, _.right -> hand.touch)

object OpenOverridePair
    extends Composition[OpenPair](_.left -> OpenOverride, _.right -> OpenOverride),
      Phased[OpenPair, OpenPhase](_.left.phase):
  override def end(s: State) = s.left.phase == OpenPhase.waiting
  object syncs extends Syncs:
    sync(_.left -> hand.touch, _.right -> hand.touch)
