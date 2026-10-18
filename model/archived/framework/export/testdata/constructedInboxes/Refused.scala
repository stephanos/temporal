package fixture.constructedinbox

import framework.dsl.*

// The old reader admitted bounded opaque messages; native state refuses an opaque Inbox message
// at the state's witness and at its machine.
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
object OpaqueMachine extends Machine[Opaque, Boolean, Nothing]:
  val init = Opaque(left.empty, right.empty)
  def end(s: State) = true
  object rules extends Bindings()
