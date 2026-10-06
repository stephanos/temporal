package fixture.features.relay

import umpire.*

enum Outcome derives Finite:
  case accepted

val task = Entity()

object user extends Actor:
  val start = action(this)
  val complete = action(this).on(task)

given Ok[Outcome] = Ok(Outcome.accepted)
