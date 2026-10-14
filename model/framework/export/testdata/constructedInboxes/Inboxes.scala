package fixture.constructedinbox

import framework.dsl.*

enum Message:
  case first, second
val first = channel[Message](capacity = 1, order = Order.fifo, loss = Loss.reliable)
val second = channel[Message](capacity = 2, order = Order.unordered, loss = Loss.reliable)
final case class First(inbox: Inbox[Message], flag: Boolean)
given firstState: Finite[First] =
  given Finite[Inbox[Message]] = first.contents
  Finite.derived
final case class Second(inbox: Inbox[Message])
given secondState: Finite[Second] =
  given Finite[Inbox[Message]] = second.contents
  Finite.derived
val small = channel[UpTo[1]](capacity = 1, order = Order.fifo, loss = Loss.reliable)
val large = channel[UpTo[3]](capacity = 1, order = Order.fifo, loss = Loss.reliable)
final case class Bounded(small: Inbox[UpTo[1]], large: Inbox[UpTo[3]])
given boundedState: Finite[Bounded] =
  given smallContents: Finite[Inbox[UpTo[1]]] = small.contents
  given largeContents: Finite[Inbox[UpTo[3]]] = large.contents
  Finite.derived
object Domains:
  opaque type Left = Int
  opaque type Right = Int
  given Finite[Left] = Finite.upTo(2)
  given Finite[Right] = Finite.upTo(2)
val left = channel[Domains.Left](capacity = 1, order = Order.fifo, loss = Loss.reliable)
val right = channel[Domains.Right](capacity = 1, order = Order.fifo, loss = Loss.reliable)
final case class Opaque(left: Inbox[Domains.Left], right: Inbox[Domains.Right])
given opaqueState: Finite[Opaque] =
  given leftContents: Finite[Inbox[Domains.Left]] = left.contents
  given rightContents: Finite[Inbox[Domains.Right]] = right.contents
  Finite.derived
enum Choice:
  case idle
  case waiting(inbox: Inbox[Message], flag: Boolean)
given choiceState: Finite[Choice] =
  given Finite[Inbox[Message]] = first.contents
  Finite.derived
final case class Nested(inner: First)
object FirstMachine extends Machine[First, Boolean, Nothing]:
  val init = First(first.empty, false)
  def end(s: State) = true
  object rules extends Bindings()
object SecondMachine extends Machine[Second, Boolean, Nothing]:
  val init = Second(second.empty)
  def end(s: State) = true
  object rules extends Bindings()
object BoundedMachine extends Machine[Bounded, Boolean, Nothing]:
  val init = Bounded(small.empty, large.empty)
  def end(s: State) = true
  object rules extends Bindings()
object OpaqueMachine extends Machine[Opaque, Boolean, Nothing]:
  val init = Opaque(left.empty, right.empty)
  def end(s: State) = true
  object rules extends Bindings()
object ChoiceMachine extends Machine[Choice, Boolean, Nothing]:
  val init = Choice.idle
  def end(s: State) = true
  object rules extends Bindings()
object NestedMachine extends Machine[Nested, Boolean, Nothing]:
  val init = Nested(First(first.empty, false))
  def end(s: State) = true
  object rules extends Bindings()
