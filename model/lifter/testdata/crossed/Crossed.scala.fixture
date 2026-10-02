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

val wire: Channel[Note] =
  channel[Note]("wire", capacity = 1, order = Order.fifo, loss = Loss.reliable)

final case class Holding(inbox: Inbox[Note])

given Finite[Holding] =
  given Finite[Inbox[Note]] = wire.contents
  Finite.derived

val elsewhere: Monitor[There, Outcome, Nothing, Boolean] =
  monitor[There, Outcome, Nothing, Boolean]("elsewhere", false)((seen, _, _) => seen)(seen => seen)

val here: Machine[Here, Outcome, Nothing] =
  machine[Here, Outcome, Nothing](Family("fixture.crossed"), "here") {
    // A monitor of another state type.
    monitors(elsewhere)
    starts(Here(false))
    ends(_ => true)
  }

val holding: Machine[Holding, Outcome, Nothing] =
  machine[Holding, Outcome, Nothing](Family("fixture.crossed"), "holding") {
    starts(Holding(wire.empty))
    ends(_ => true)
    // A delivery handler that takes another message type.
    steps(wire.deliver ~> ((h: Holding, s: Signal) => List(Step(Outcome.accepted, h))))
  }
