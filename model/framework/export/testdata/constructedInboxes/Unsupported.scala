package fixture.inboxunsupported

import framework.dsl.*

enum Message:
  case ping
val wire = channel[Message](capacity = 1, order = Order.fifo, loss = Loss.reliable)
def contents[M](value: Channel[M]): Finite[Inbox[M]] = value.contents
final case class Forwarded(box: Inbox[Message])
given forwarded: Finite[Forwarded] =
  given inbox: Finite[Inbox[Message]] = contents(wire)
  Finite.derived
val aliasedContents = wire.contents
final case class Aliased(box: Inbox[Message])
given aliased: Finite[Aliased] =
  given inbox: Finite[Inbox[Message]] = aliasedContents
  Finite.derived
object ForwardedMachine extends Machine[Forwarded, Boolean, Nothing]:
  val init = Forwarded(wire.empty)
  def end(s: State) = true
  object rules extends Bindings()
object AliasedMachine extends Machine[Aliased, Boolean, Nothing]:
  val init = Aliased(wire.empty)
  def end(s: State) = true
  object rules extends Bindings()
