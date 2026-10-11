package fixture.actors.quiet

import framework.*

enum Outcome derives Finite:
  case accepted

object Idle extends Machine[Boolean, Outcome, Nothing]:
  val init = false
  def end(s: State) = true
  object rules extends Rules

val unrelated = Vector("ordinary data")
