package fixture.rulespartialbridge

import framework.dsl.*
import fixture.rulespartial.Library

object Bridge extends Machine[Boolean, Boolean, Nothing]:
  val init = false
  def end(s: Boolean) = s
  object effects:
    def forwarded(s: Boolean) = Library.effects.prefixed(true)(s)
    def trailing(s: Boolean) = Library.effects.trailing(s)()
  object rules extends Rules
