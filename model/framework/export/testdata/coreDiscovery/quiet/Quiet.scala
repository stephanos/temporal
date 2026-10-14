package fixture.actors.quiet

import framework.dsl.*

enum Outcome:
  case accepted

val unrelated = Vector("ordinary data")

object Idle extends Machine[Boolean, Outcome, Nothing]:
  val init = false
  def end(s: State) = true
  object rules extends Rules
