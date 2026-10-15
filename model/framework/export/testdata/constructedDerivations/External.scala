package fixture.derivationexternal

import framework.dsl.*
import framework.dsl.Finite.given

val wire = channel[Boolean](capacity = 1, order = Order.fifo, loss = Loss.reliable)
val unknown = hole
final case class Snapshot(flag: Boolean, messages: Inbox[Boolean])
given Finite[Snapshot] =
  given Finite[Inbox[Boolean]] = wire.contents
  Finite.derived
object hand extends Actor:
  val go = action(this)

object Source extends Machine[Snapshot, Boolean, Nothing]:
  val init = Snapshot(false, wire.empty)
  def end(s: Snapshot) = s.flag
  object effects:
    @lifted def first(s: Snapshot): List[Step[Snapshot, Boolean, Nothing]] = List(Step(true, s))
    @lifted def external(s: Snapshot): List[Step[Snapshot, Boolean, Nothing]] =
      require(s.messages.isEmpty)
      if s.flag then unknown.reached else List(Step(true, s))
  object rules extends Rules:
    on(hand.go)(always ~> effects.first)

object Rebound extends Derived(Source.rebind(hand.go ~> Source.effects.external))
object Group extends Derived(Source.rebind(on(hand.go)(always ~> Source.effects.external)))
