package fixture.rulespartial

import framework.dsl.*
import framework.dsl.Finite.given

given Ok[Boolean] = Ok(true)
object hand extends Actor:
  val go = action(this)
  val six = action(this)
    .inputs[SixInputs]
// A native six-input action declares one record (fn-165) with the retired chain's names, order and
// types; the rule reads its fields into the partial helper's flat parameters. It takes the retired
// chain's lines, so every frozen position keeps its line.
final case class SixInputs(s: Boolean, state: Boolean, c: Boolean, d: Boolean, e: Boolean, f: Boolean)

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
    on(hand.six)(always ~> ((s: Boolean, in: SixInputs) => Library.effects.six(true)(s, in.s, in.state, in.c, in.d, in.e, in.f)))
