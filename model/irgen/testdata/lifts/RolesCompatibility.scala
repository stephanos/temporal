package fixture.roles

import framework.Finite

@main def verifyRolePrefixes(): Unit =
  val phases = Finite[Phase].values
  val states = Finite[Job].values
  for phase <- phases do
    val (roled, listed) = (Roled.states.settled(phase), Listed.states.settled(phase))
    require(roled == listed)
  for state <- states do
    val (roled, listed) = (Roled.states.stopping(state), Listed.states.stopping(state))
    require(roled == listed)
  println(s"role prefixes equivalent: ${phases.size} phases, ${states.size} states")
