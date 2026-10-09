// The shared outcomes, framework.outcomes (fn-139.1): two machines on the framework's `Outcome` and
// `Rejection`, whose `enter` and `stay` answer `accepted` by the framework's given. `DoorSystem`
// writes its rejecting rules with `rejects(r)` and `rejects(r).because(text)`; `DoorProduct` writes
// the same rules with effects that spell `reject(Outcome.rejected(r), s)`. The lifter's tests
// require one step function of the two, but for names, positions and the explanation `because`
// gives. The system refines the product, whose outcomes it shares, and its capabilities section
// passes `rejected = Outcome.rejected(Rejection.notFound)`. tools/umpire/ir admits the IR.
package fixture.rejections

import framework.*
import framework.outcomes.{Outcome, Rejection}

enum Phase derives Finite:
  case shut, ajar, locked, gone

// A door's state, named apart from the machines.
final case class Door(phase: Phase) derives Finite

enum Fact derives Finite:
  case opened, closed

object visitor extends Actor:
  val open = action(this)
  val close = action(this)
  val ring = action(this)

// A door that is gone, which every request to it finds not there.
object Doors:
  def gone(d: Door): Boolean = d.phase == Phase.gone

// A door that is over answers every request `rejected` and keeps its state.
final case class Sealed[S, O](over: S => Boolean, rejected: O) extends CapabilityOf[S, O, Nothing]

object Sealed extends CapabilityKind:
  // Every step from a state that is over keeps the state and answers `rejected`.
  def overIsRejected[S](m: Declares[S])(over: S => Boolean, rejected: m.Outcome): Property[S] =
    m.property holdsAcross ((before, after) =>
      !over(before) || (after.state == before && after.outcome == rejected)
    )

val two = Limits(steps = 2, actions = 2, search = 256)

// The core spelling: each rejection an effect of its own.
object DoorProduct extends Machine[Door, Outcome, Fact], Phased[Door, Phase](_.phase):
  val init = Door(Phase.shut)
  override def end(d: State) = Doors.gone(d)

  object effects:
    def open(d: Door) = enter(d.copy(phase = Phase.ajar), Fact.opened)
    def close(d: Door) = enter(d.copy(phase = Phase.shut), Fact.closed)
    def keep(d: Door) = stay[Door, Outcome, Fact](d)
    def locked(d: Door) = reject(Outcome.rejected(Rejection.failedPrecondition), d)
    def missing(d: Door) = reject(Outcome.rejected(Rejection.notFound), d)

  object rules extends Rules:
    on(visitor.open) {
      when(Phase.shut) ~> effects.open
      when(Phase.locked) ~> effects.locked
      when(Phase.gone) ~> effects.missing
    }
    on(visitor.close) {
      when(Phase.ajar) ~> effects.close
      when(Phase.shut) ~> effects.keep
    }
    on(visitor.ring)(always ~> effects.missing)

// The same rules with `rejects`, a refinement of the product it spells.
object DoorSystem extends Machine[Door, Outcome, Fact], Phased[Door, Phase](_.phase):
  val init = Door(Phase.shut)
  override def end(d: State) = Doors.gone(d)

  object refinement extends Refinement(DoorProduct):
    def toProduct(d: Door): Door = d

  object effects:
    def open(d: Door) = enter(d.copy(phase = Phase.ajar), Fact.opened)
    def close(d: Door) = enter(d.copy(phase = Phase.shut), Fact.closed)
    def keep(d: Door) = stay[Door, Outcome, Fact](d)

  object rules extends Rules:
    on(visitor.open) {
      when(Phase.shut) ~> effects.open
      when(Phase.locked) ~> rejects(Rejection.failedPrecondition).because("the door is locked")
      when(Phase.gone) ~> rejects(Rejection.notFound)
    }
    on(visitor.close) {
      when(Phase.ajar) ~> effects.close
      when(Phase.shut) ~> effects.keep
    }
    on(visitor.ring)(always ~> rejects(Rejection.notFound))

  object capabilities extends Capabilities:
    val sealedDoor: Capability =
      Sealed(over = Doors.gone, rejected = Outcome.rejected(Rejection.notFound))

  // 4 states x 3 classes x 2 steps = 24.
  object queries:
    capabilities.bound(two)
