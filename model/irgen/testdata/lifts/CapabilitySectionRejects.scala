// Capabilities sections the lifter refuses, each at its line (fn-134.2): a capability Property
// that reads no field of its own capability, one taking a parameter no capability binds, one whose
// parameter two declared capabilities both bind, a kind declared twice, a waiver of a Property
// no declared capability brings, a waiver with an empty reason, a machine whose brought Property no
// `queries` section bounds, and a bound override of a Property not brought and of one waived.
package fixture.capabilitysectionrejects

import framework.*
import fixture.capabilitysections.{hold, release, take, three, Answer, Note, Phase}
import fixture.capabilitysections.{Holdable, Sealable, Takeable, Tasks, TaskState}

// A capability whose Property reads Takeable's field and none of its own.
final case class Idle[S](idle: S => Boolean) extends CapabilityOf[S, Nothing, Nothing]

object Idle extends CapabilityKind:
  def takenIsIdle[S](m: Declares[S])(working: S => Boolean): Property[S] =
    m.property.never(s => working(s.state))

// A capability whose Property takes `found`, which no capability binds.
final case class Lost[S](lost: S => Boolean) extends CapabilityOf[S, Nothing, Nothing]

object Lost extends CapabilityKind:
  def lostIsFound[S](m: Declares[S])(lost: S => Boolean, found: S => Boolean): Property[S] =
    m.property.never(s => lost(s.state) && !found(s.state))

// Two capabilities that bind `lit` alike.
final case class Bright[S](lit: S => Boolean) extends CapabilityOf[S, Nothing, Nothing]

object Bright extends CapabilityKind:
  def brightIsLit[S](m: Declares[S])(lit: S => Boolean): Property[S] =
    m.property.never(s => !lit(s.state))

final case class Dim[S](lit: S => Boolean) extends CapabilityOf[S, Nothing, Nothing]

object Dim extends CapabilityKind

object ReadsNoField extends Machine[TaskState, Answer, Note]:
  val init = TaskState(Phase.queued)
  def end(t: State) = Tasks.over(t)
  object rules extends Bindings(take ~> Tasks.take)
  object capabilities extends Capabilities:
    val idle: Capability = Idle(idle = Tasks.held)
    val takeable: Capability = Takeable(take = take, working = Tasks.working)
  object queries:
    capabilities.bound(three)

object UnboundParameter extends Machine[TaskState, Answer, Note]:
  val init = TaskState(Phase.queued)
  def end(t: State) = Tasks.over(t)
  object rules extends Bindings(take ~> Tasks.take)
  object capabilities extends Capabilities:
    val lost: Capability = Lost(lost = Tasks.held)
  object queries:
    capabilities.bound(three)

object BoundTwice extends Machine[TaskState, Answer, Note]:
  val init = TaskState(Phase.queued)
  def end(t: State) = Tasks.over(t)
  object rules extends Bindings(take ~> Tasks.take)
  object capabilities extends Capabilities:
    val bright: Capability = Bright(lit = Tasks.working)
    val dim: Capability = Dim(lit = Tasks.held)
  object queries:
    capabilities.bound(three)

object DeclaredTwice extends Machine[TaskState, Answer, Note]:
  val init = TaskState(Phase.queued)
  def end(t: State) = Tasks.over(t)
  object rules extends Bindings(take ~> Tasks.take)
  object capabilities extends Capabilities:
    val takeable: Capability = Takeable(take = take, working = Tasks.working)
    val again: Capability = Takeable(take = take, working = Tasks.held)
  object queries:
    capabilities.bound(three)

// Holdable without Takeable brings no heldIsNotTaken to waive.
object WaivedUnbrought extends Machine[TaskState, Answer, Note]:
  val init = TaskState(Phase.queued)
  def end(t: State) = Tasks.over(t)
  object rules extends Bindings(hold ~> Tasks.hold, release ~> Tasks.release)
  object capabilities extends Capabilities:
    val holdable: Capability = Holdable(hold = hold, release = release, held = Tasks.held)
    except(Holdable.heldIsNotTaken, because = "nothing takes this task")
  object queries:
    capabilities.bound(three)

object EmptyReason extends Machine[TaskState, Answer, Note]:
  val init = TaskState(Phase.queued)
  def end(t: State) = Tasks.over(t)
  object rules extends Bindings(hold ~> Tasks.hold, release ~> Tasks.release)
  object capabilities extends Capabilities:
    val sealable: Capability =
      Sealable(status = Tasks.phase, done = Tasks.done, refused = Answer.gone)
    except(Sealable.sealedStaysSealed, because = " ")
  object queries:
    capabilities.bound(three)

// Its capabilities bring sealedStaysSealed and sealedIsRefused, and no `queries` section bounds
// them.
object Unbounded extends Machine[TaskState, Answer, Note]:
  val init = TaskState(Phase.queued)
  def end(t: State) = Tasks.over(t)
  object rules extends Bindings(hold ~> Tasks.hold, release ~> Tasks.release)
  object capabilities extends Capabilities:
    val sealable: Capability =
      Sealable(status = Tasks.phase, done = Tasks.done, refused = Answer.gone)

object OverrideUnbrought extends Machine[TaskState, Answer, Note]:
  val init = TaskState(Phase.queued)
  def end(t: State) = Tasks.over(t)
  object rules extends Bindings(hold ~> Tasks.hold, release ~> Tasks.release)
  object capabilities extends Capabilities:
    val sealable: Capability =
      Sealable(status = Tasks.phase, done = Tasks.done, refused = Answer.gone)
  object queries:
    capabilities.bound(three, Holdable.heldIsNotTaken -> three)

object OverrideWaived extends Machine[TaskState, Answer, Note]:
  val init = TaskState(Phase.queued)
  def end(t: State) = Tasks.over(t)
  object rules extends Bindings(hold ~> Tasks.hold, release ~> Tasks.release)
  object capabilities extends Capabilities:
    val sealable: Capability =
      Sealable(status = Tasks.phase, done = Tasks.done, refused = Answer.gone)
    except(Sealable.sealedIsRefused, because = "a fixture's waiver")
  object queries:
    capabilities.bound(three, Sealable.sealedIsRefused -> three)
