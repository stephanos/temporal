package fixture.rulesinvalidclient

import framework.dsl.*
import fixture.rulesinvalids.{hand, Invalids}

object InvalidGuard extends Machine[Boolean, Boolean, Nothing]:
  val init = false
  def end(s: Boolean) = s
  object effects:
    def keep(s: Boolean) = List(Step[Boolean, Boolean, Nothing](true, s))
  object rules extends Rules:
    on(hand.go)(where(Invalids.states.bad) ~> effects.keep)
