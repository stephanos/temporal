package fixture.naming

import framework.dsl.*
import framework.realize.*

enum Fact:
  case recorded

object Machine extends framework.dsl.Machine[Boolean, Boolean, Fact]:
  val init = false
  def end(s: Boolean): Boolean = s
  object rules extends Rules

val original: Instruction = Instruction.Hold("gate")
val alias: Instruction = original

def scripted(call: Instruction): Script =
  script("controller", Activation.Controller)(everyCase(call))

val realization = Realization(
  machine = Machine,
  producer = "fixture.naming",
  producerVersion = "1",
  roles = Vector.empty,
  correlation =
    Correlation("projection", "run", "operation", "observation", 8, 8, 2, 16, 1000, 512),
  scripts = Vector(scripted(original))
)
