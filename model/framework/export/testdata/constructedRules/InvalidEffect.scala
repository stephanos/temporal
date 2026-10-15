package fixture.rulesinvalidclient

import framework.dsl.*
import fixture.rulesinvalids.{hand, Invalids}

object InvalidEffect extends Machine[Boolean, Boolean, Nothing]:
  val init = false
  def end(s: Boolean) = s
  object rules extends Rules:
    on(hand.go)(always ~> Invalids.effects.bad)
