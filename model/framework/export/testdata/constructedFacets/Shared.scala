package fixture.facetstargets

import framework.dsl.*

enum Dial:
  case low, high
final case class State(lit: Boolean)
enum Outcome:
  case accepted
given Ok[Outcome] = Ok(Outcome.accepted)
object hand extends Actor:
  val press = action(this)
  val turn = action(this).input[Dial]("dial")
  val stop = action(this)
  val reset = action(this)
object clock:
  val tick = timer
