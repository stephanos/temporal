package fixture.boundedintegers

import framework.dsl.*
import framework.dsl.Finite.given

// A named bound: native state refuses an opaque Int, so the name aliases its UpTo.
object bounded:
  type Retries = UpTo[2]

final val one = 1
final case class Counters(
    zero: UpTo[0],
    first: UpTo[one.type],
    second: UpTo[3],
    retries: bounded.Retries
)
final case class Optional(first: Option[UpTo[1]], second: Option[UpTo[3]])
enum Outcome:
  case accepted

object Fields extends Machine[Counters, Outcome, Nothing]:
  val init = Counters(UpTo(0), UpTo(0), UpTo(0), UpTo(0))
  def end(s: State) = true
  object rules extends Bindings()

object Options extends Machine[Optional, Outcome, Nothing]:
  val init = Optional(None, None)
  def end(s: State) = true
  object rules extends Bindings()

val small: Channel[UpTo[1]] =
  channel[UpTo[1]](capacity = 1, order = Order.fifo, loss = Loss.reliable)
val large: Channel[UpTo[3]] =
  channel[UpTo[3]](capacity = 1, order = Order.fifo, loss = Loss.reliable)
final case class Mail(first: Inbox[UpTo[1]], second: Inbox[UpTo[3]])
given Finite[Mail] =
  given smallContents: Finite[Inbox[UpTo[1]]] = small.contents
  given largeContents: Finite[Inbox[UpTo[3]]] = large.contents
  Finite.derived

object Messages extends Machine[Mail, Outcome, Nothing]:
  val init = Mail(small.empty, large.empty)
  def end(s: State) = true
  object rules extends Bindings()
