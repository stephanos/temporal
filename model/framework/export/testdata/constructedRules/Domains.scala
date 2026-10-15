package fixture.rulesdomains

import framework.dsl.*

given Ok[Boolean] = Ok(true)
enum Mode:
  case plain
  case tagged(flag: Boolean)
final case class Pair(left: Boolean, right: Boolean)
val low = UpTo[1](0)
object hand extends Actor:
  val bool = action(this).input[Boolean]("flag")
  val range = action(this).input[UpTo[1]]("n")
  val enumValue = action(this).input[Mode]("mode")
  val record = action(this).input[Pair]("pair")
  val nested = action(this).input[Boolean]("x").input[Boolean]("y")
  val partial = action(this)

object Domains extends Machine[Boolean, Boolean, Nothing]:
  val init = false
  def end(s: Boolean) = s
  object effects:
    def keep(s: Boolean) = stay[Boolean, Boolean, Nothing](s)
    def prefixed(prefix: Boolean)(s: Boolean) = stay[Boolean, Boolean, Nothing](prefix && s)
  object rules extends Rules:
    on(hand.bool(false))(where(s => !s) ~> effects.keep)
    on(hand.bool)(where(s => s) ~> effects.keep)
    on(hand.range(low))(where(s => !s) ~> effects.keep)
    on(hand.range)(where(s => s) ~> effects.keep)
    on(hand.enumValue(Mode.tagged(false)))(where(s => !s) ~> effects.keep)
    on(hand.enumValue)(where(s => s) ~> effects.keep)
    on(hand.record(Pair(false, true)))(where(s => !s) ~> effects.keep)
    on(hand.record)(where(s => s) ~> effects.keep)
    on(hand.nested(false, true))(where(s => !s) ~> effects.keep)
    on(hand.nested)(where(s => s) ~> effects.keep)
    on(hand.partial)(always ~> effects.prefixed(true))
