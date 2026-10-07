package fixture.phasecapabilities

import umpire.*
import temporal.capabilities.*

val bounds = Limits(steps = 2, actions = 3, search = 64)

enum Phase derives Finite:
  case waiting extends Phase, Waiting
  case paused extends Phase, Suspended
  case pausedWhileHeld extends Phase, Held
  case started extends Phase, Held
  case done extends Phase, Succeeded

enum Answer derives Finite:
  case accepted

final case class Snapshot(phase: Phase) derives Finite
final case class OtherSnapshot(phase: Phase) derives Finite
final case class Pair(left: Snapshot, right: OtherSnapshot)

given Ok[Answer] = Ok(Answer.accepted)

object user extends Actor:
  val pause = action(this)
  val unpause = action(this)

object worker extends Actor:
  val poll = action(this)

object Direct extends Machine[Snapshot, Answer, Nothing], Phased[Snapshot, Phase](_.phase):
  val init = Snapshot(Phase.waiting)
  object effects:
    def keep(s: State) = stay[Snapshot, Answer, Nothing](s)
  object rules extends Rules:
    on(user.pause, user.unpause, worker.poll)(always ~> effects.keep)
  object capabilities extends Capabilities:
    val pausable: Capability = Pausable(pause = user.pause, unpause = user.unpause)
    val pollable: Capability = Pollable(dispatch = worker.poll)
  object queries:
    capabilities.bound(bounds)

object OtherDirect
    extends Machine[OtherSnapshot, Answer, Nothing],
      Phased[OtherSnapshot, Phase](_.phase):
  val init = OtherSnapshot(Phase.waiting)
  object effects:
    def keep(s: State) = stay[OtherSnapshot, Answer, Nothing](s)
  object rules extends Rules:
    on(user.pause, user.unpause, worker.poll)(always ~> effects.keep)
  object capabilities extends Capabilities:
    val pausable: Capability = Pausable(pause = user.pause, unpause = user.unpause)
    val pollable: Capability = Pollable(dispatch = worker.poll)
  object queries:
    capabilities.bound(bounds)

object DirectDerived extends Derived(Direct.unmonitored):
  object capabilities extends Capabilities:
    val pausable: Capability = Pausable(pause = user.pause, unpause = user.unpause)
    val pollable: Capability = Pollable(dispatch = worker.poll)
  object queries:
    capabilities.bound(bounds)

abstract class PairCapabilities(c: Composition[Pair])(using
    Declaring[Pair, String, String],
    Phasing[Pair, Phase]
) extends Capabilities:
  val pausable: Capability = Pausable(
    pause = c.own(_.left, user.pause),
    unpause = c.own(_.left, user.unpause)
  )
  val pollable: Capability = Pollable(dispatch = c.own(_.left, worker.poll))

object DirectPair
    extends Composition[Pair](_.left -> Direct, _.right -> OtherDirect),
      Phased[Pair, Phase](_.left.phase):
  object syncs extends Syncs
  object capabilities extends PairCapabilities(this)
  object queries:
    capabilities.bound(bounds)

object DirectDerivedPair extends DerivedComposition(DirectPair.withMember(_.right -> OtherDirect)):
  object capabilities extends PairCapabilities(this)
  object queries:
    capabilities.bound(bounds)

enum NoPausedPhase derives Finite:
  case waiting extends NoPausedPhase, Waiting
  case started extends NoPausedPhase, Held
  case done extends NoPausedPhase, Succeeded

final case class NoPaused(phase: NoPausedPhase) derives Finite

object MissingSuspended
    extends Machine[NoPaused, Answer, Nothing],
      Phased[NoPaused, NoPausedPhase](_.phase):
  val init = NoPaused(NoPausedPhase.waiting)
  object effects:
    def keep(s: State) = stay[NoPaused, Answer, Nothing](s)
  object rules extends Rules:
    on(user.pause, user.unpause, worker.poll)(always ~> effects.keep)
  object capabilities extends Capabilities:
    val pausable: Capability = Pausable(pause = user.pause, unpause = user.unpause)
  object queries:
    capabilities.bound(bounds)

enum NoHeldPhase derives Finite:
  case waiting extends NoHeldPhase, Waiting
  case paused extends NoHeldPhase, Suspended
  case done extends NoHeldPhase, Succeeded

final case class NoHeld(phase: NoHeldPhase) derives Finite

object MissingHeld extends Machine[NoHeld, Answer, Nothing], Phased[NoHeld, NoHeldPhase](_.phase):
  val init = NoHeld(NoHeldPhase.waiting)
  object effects:
    def keep(s: State) = stay[NoHeld, Answer, Nothing](s)
  object rules extends Rules:
    on(user.pause, user.unpause, worker.poll)(always ~> effects.keep)
  object capabilities extends Capabilities:
    val pausable: Capability = Pausable(pause = user.pause, unpause = user.unpause)
    val pollable: Capability = Pollable(dispatch = worker.poll)
  object queries:
    capabilities.bound(bounds)
