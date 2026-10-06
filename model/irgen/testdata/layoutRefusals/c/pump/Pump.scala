// A shared feature: its root feature file may name no IR file.
package fixture.shared.pump

import umpire.*

final case class Pump(running: Boolean) derives Finite

enum Outcome derives Finite:
  case accepted

type PumpStep = Step[Pump, Outcome, Nothing]

object operator extends Actor:
  val start = action(this)

given Ok[Outcome] = Ok(Outcome.accepted)
