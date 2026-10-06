// A feature of one level, whose machine sits in a product/ folder all the same.
package fixture.features.tap

import umpire.*
import product.Faucet

given Family = Family("fixture.tap")

final case class Tap(open: Boolean) derives Finite

enum Outcome derives Finite:
  case accepted

object plumber extends Actor:
  val turn = action(this)

given Ok[Outcome] = Ok(Outcome.accepted)

object exports:
  val tap = irFile("tap")(Faucet)
