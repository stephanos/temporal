package fixture.foundations.coreonly

import framework.*
import framework.realize.*

enum Outcome derives Finite:
  case accepted

object Runs:
  val only = Realization(
    machine = Idle,
    producer = family + ".fixture",
    producerVersion = "1",
    roles = Vector.empty,
    correlation = Correlation("projection", "run", "operation", "observation", 8, 8, 2, 16, 1000, 512),
    scripts = Vector.empty
  )
  val unrelated = Vector("ordinary data")

object Idle extends Machine[Boolean, Outcome, Nothing]:
  val init = false
  def end(s: State) = true
  object rules extends Rules
