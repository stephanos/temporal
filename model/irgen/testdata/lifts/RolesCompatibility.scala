package fixture.roles

import framework.Finite

@main def verifyRolePrefixes(): Unit =
  val phases = Finite[Phase].values
  val states = Finite[Job].values
  for phase <- phases do
    require(Roled.states.settled(phase) == Listed.states.settled(phase)) // scalafix:ok DisableSyntax.==; typed equality
  for state <- states do
    require(Roled.states.stopping(state) == Listed.states.stopping(state)) // scalafix:ok DisableSyntax.==; typed equality
  println(s"role prefixes equivalent: ${phases.size} phases, ${states.size} states")
