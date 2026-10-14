// A feature of one level, whose machine sits in a product/ folder all the same.
package fixture.features.tap

import framework.dsl.*

final case class Tap(open: Boolean)

enum Outcome:
  case accepted

object plumber extends Actor:
  val turn = action(this)

given Ok[Outcome] = Ok(Outcome.accepted)
