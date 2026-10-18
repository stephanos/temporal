package fixture.derivationfacets

import framework.dsl.*

enum Fact:
  case opened, closed
enum Phase(val status: Fact) extends Recorded[Fact]:
  case open extends Phase(Fact.opened), Live
  case done extends Phase(Fact.closed), Closed
final case class State(left: Phase, right: Phase)
given Ok[Boolean] = Ok(true)
object hand extends Actor:
  val go = action(this)
  val more = action(this)
  val off = action(this)
object clock:
  val tick = timer
