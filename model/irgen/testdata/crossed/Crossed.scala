// Declarations whose types cross, which the framework's types refuse before anything is lifted:
// The lifter's tests build this and expect a type error at each marked line.
package fixture.crossed

import umpire.*

enum Note derives Finite:
  case ping

enum Signal derives Finite:
  case up

enum Outcome derives Finite:
  case accepted

final case class Here(on: Boolean) derives Finite

final case class There(on: Boolean) derives Finite

given Family = Family("fixture.crossed")
val wire: Channel[Note] = channel[Note](capacity = 1, order = Order.fifo, loss = Loss.reliable)

final case class Holding(inbox: Inbox[Note])

given Finite[Holding] =
  given Finite[Inbox[Note]] = wire.contents
  Finite.derived

object HereMachine extends Machine[Here, Outcome, Nothing]:
  val init = Here(false)
  def end(here: State) = true
  object rules extends Bindings()

object HoldingMachine extends Machine[Holding, Outcome, Nothing]:
  val init = Holding(wire.empty)
  def end(holding: State) = true
  // A delivery handler that takes another message type.
  object rules
      extends Bindings(wire.deliver ~> ((h: Holding, s: Signal) => List(Step(Outcome.accepted, h))))

// Membership in no members.
val nowhere: Boolean = Signal.up.in()

object ThereMachine extends Machine[There, Outcome, Nothing]:
  val init = There(false)
  def end(there: State) = true
  object rules extends Bindings()

// A replacement refinement whose map leads to states of another type than its machine's.
object CrossedRefinement extends Derived(HereMachine.refining(ThereMachine)(h => h))
