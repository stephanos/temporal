package fixture.rulespartial

import framework.dsl.*

given Ok[Boolean] = Ok(true)
object hand extends Actor:
  val go = action(this)
object Client extends Machine[Boolean, Boolean, Nothing]:
  val init = false
  def end(s: Boolean) = s
  object rules extends Rules:
    on(hand.go)(always ~> Library.effects.prefixed(true))

object LocalForwarder extends Machine[Boolean, Boolean, Nothing]:
  val init = false
  def end(s: Boolean) = s
  object effects:
    def keep(s: Boolean) = Library.effects.prefixed(true)(s)
  object rules extends Rules:
    on(hand.go)(always ~> effects.keep)

object StoredForwarder extends Machine[Boolean, Boolean, Nothing]:
  val init = false
  def end(s: Boolean) = s
  object rules extends Rules:
    on(hand.go)(always ~> fixture.rulespartialbridge.Bridge.effects.forwarded)
