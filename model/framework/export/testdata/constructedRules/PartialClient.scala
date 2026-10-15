package fixture.rulespartial

import framework.dsl.*
import framework.dsl.Finite.given

given Ok[Boolean] = Ok(true)
object hand extends Actor:
  val go = action(this)
  val six = action(this)
    .input[Boolean]("s")
    .input[Boolean]("state")
    .input[Boolean]("c")
    .input[Boolean]("d")
    .input[Boolean]("e")
    .input[Boolean]("f")
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

object Leading extends Machine[Boolean, Boolean, Nothing]:
  val init = false
  def end(s: Boolean) = s
  object rules extends Rules:
    on(hand.go)(always ~> Library.effects.leading()(true))
object Middle extends Machine[Boolean, Boolean, Nothing]:
  val init = false
  def end(s: Boolean) = s
  object rules extends Rules:
    on(hand.go)(always ~> Library.effects.middle(true)())
object Repeated extends Machine[Boolean, Boolean, Nothing]:
  val init = false
  def end(s: Boolean) = s
  object rules extends Rules:
    on(hand.go)(always ~> Library.effects.repeated()()(true)())
object Trailing extends Machine[Boolean, Boolean, Nothing]:
  val init = false
  def end(s: Boolean) = s
  object rules extends Rules:
    on(hand.go)(always ~> fixture.rulespartialbridge.Bridge.effects.trailing)
object Polymorphic extends Machine[Boolean, Boolean, Nothing]:
  val init = false
  def end(s: Boolean) = s
  object rules extends Rules:
    on(hand.go)(always ~> Library.effects.polymorphic[Boolean](true))
object Multiple extends Machine[Boolean, Boolean, Nothing]:
  val init = false
  def end(s: Boolean) = s
  object rules extends Rules:
    on(hand.go)(always ~> Library.effects.multiple(true)(false))
object Six extends Machine[Boolean, Boolean, Nothing]:
  val init = false
  def end(s: Boolean) = s
  object rules extends Rules:
    on(hand.six)(always ~> Library.effects.six(true))
