package fixture.phasecapabilities

import framework.*
import temporal.capabilities.*

val bounds = Limits(steps = 2, actions = 3, search = 64)

enum WorkflowPhase derives Finite:
  case waiting extends WorkflowPhase, Waiting
  case paused extends WorkflowPhase, Suspended
  case pausedWhileHeld extends WorkflowPhase, Held
  case started extends WorkflowPhase, Held
  case done extends WorkflowPhase, Succeeded

enum Result derives Finite:
  case accepted

final case class WorkState(phase: WorkflowPhase) derives Finite
final case class OtherWorkState(phase: WorkflowPhase) derives Finite
final case class Pair(left: WorkState, right: OtherWorkState)

given Ok[Result] = Ok(Result.accepted)

object user extends Actor:
  val pause = action(this)
  val unpause = action(this)

object worker extends Actor:
  val poll = action(this)

object Direct
    extends Machine[WorkState, Result, Nothing],
      Phased[WorkState, WorkflowPhase](_.phase):
  val init = WorkState(WorkflowPhase.waiting)
  object effects:
    def keep(s: State) = stay[WorkState, Result, Nothing](s)
  object rules extends Rules:
    on(user.pause, user.unpause, worker.poll)(always ~> effects.keep)
  object capabilities extends Capabilities:
    val pausable: Capability = Pausable(pause = user.pause, unpause = user.unpause)
    val pollable: Capability = Pollable(dispatch = worker.poll)
  object queries:
    capabilities.bound(bounds)

object OtherDirect
    extends Machine[OtherWorkState, Result, Nothing],
      Phased[OtherWorkState, WorkflowPhase](_.phase):
  val init = OtherWorkState(WorkflowPhase.waiting)
  object effects:
    def keep(s: State) = stay[OtherWorkState, Result, Nothing](s)
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
    Phasing[Pair, WorkflowPhase]
) extends Capabilities:
  val pausable: Capability = Pausable(
    pause = c.own(_.left, user.pause),
    unpause = c.own(_.left, user.unpause)
  )
  val pollable: Capability = Pollable(dispatch = c.own(_.left, worker.poll))

object DirectPair
    extends Composition[Pair](_.left -> Direct, _.right -> OtherDirect),
      Phased[Pair, WorkflowPhase](_.left.phase):
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
    extends Machine[NoPaused, Result, Nothing],
      Phased[NoPaused, NoPausedPhase](_.phase):
  val init = NoPaused(NoPausedPhase.waiting)
  object effects:
    def keep(s: State) = stay[NoPaused, Result, Nothing](s)
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

object MissingHeld extends Machine[NoHeld, Result, Nothing], Phased[NoHeld, NoHeldPhase](_.phase):
  val init = NoHeld(NoHeldPhase.waiting)
  object effects:
    def keep(s: State) = stay[NoHeld, Result, Nothing](s)
  object rules extends Rules:
    on(user.pause, user.unpause, worker.poll)(always ~> effects.keep)
  object capabilities extends Capabilities:
    val pausable: Capability = Pausable(pause = user.pause, unpause = user.unpause)
    val pollable: Capability = Pollable(dispatch = worker.poll)
  object queries:
    capabilities.bound(bounds)
