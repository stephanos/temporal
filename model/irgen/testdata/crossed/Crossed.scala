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

val elsewhere: Monitor[There, Outcome, Nothing, Boolean] =
  monitor[There, Outcome, Nothing, Boolean](false)((seen, _, _) => seen)(seen => seen)

val here: Machine[Here, Outcome, Nothing] =
  machine[Here, Outcome, Nothing] {
    // A monitor of another state type.
    monitors(elsewhere)
    starts(Here(false))
    ends(_ => true)
  }

val holding: Machine[Holding, Outcome, Nothing] =
  machine[Holding, Outcome, Nothing] {
    starts(Holding(wire.empty))
    ends(_ => true)
    // A delivery handler that takes another message type.
    steps(wire.deliver ~> ((h: Holding, s: Signal) => List(Step(Outcome.accepted, h))))
  }

// Membership in no members.
val nowhere: Boolean = Signal.up.in()

val there: Machine[There, Outcome, Nothing] =
  machine[There, Outcome, Nothing] {
    starts(There(false))
    ends(_ => true)
  }

// A replacement refinement whose map leads to states of another type than its machine's.
val crossedRefinement: Machine[Here, Outcome, Nothing] =
  here.refining(there)(h => h)(using Family("fixture.crossed"))
