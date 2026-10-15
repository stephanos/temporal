package fixture.rulesinvalids

import framework.dsl.*

given Ok[Boolean] = Ok(true)
object hand extends Actor:
  val go = action(this)
object Invalids extends Machine[Boolean, Boolean, Nothing]:
  val init = false
  def end(s: Boolean) = s
  object states:
    val bad = is(throw IllegalArgumentException("unsupported"))
  object effects:
    val bad = effect(if true then reject(false))
  object rules extends Rules:
    disabled(hand.go)
