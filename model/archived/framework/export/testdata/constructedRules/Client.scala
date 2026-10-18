package fixture.rulesclient

import framework.dsl.*
import fixture.ruleslibrary.{hand, Library}

object Client extends Machine[Boolean, Boolean, Nothing]:
  val init = false
  def end(s: Boolean) = s
  object rules extends Rules:
    on(hand.go)(where(Library.states.ready) ~> Library.effects.keep)
object Contracts extends Machine[Boolean, Boolean, Nothing]:
  val init = false
  def end(s: Boolean) = s
  object rules extends Rules:
    on(hand.go)(where(Library.states.guarded) ~> Library.effects.held)
