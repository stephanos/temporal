// The feature file of a fixture feature (fn-126 R2, R3), lifted with the files beside it. Its
// Switch reads as a feature file is read; Backwards and Misplaced do not, at the lines the lint
// refuses: (c) a type after a machine object and a section out of its order, and (d) a Property
// written in its machine's object rather than its `properties`, and one over another object's
// machine. (b): Switch's `laws` reads the realization (Realization.scala), which reads it back.
package fixture.features.initorder

import umpire.*

given Family = Family("fixture.initorder")

final case class Lamp(lit: Boolean) derives Finite

enum Outcome derives Finite:
  case accepted

val flip = action(Party("fixture"))
val one = Limits(steps = 1, actions = 1, search = 8)

object Switch:
  def lit(s: Lamp) = s.lit

  object effects:
    def flipped(s: Lamp): List[Step[Lamp, Outcome, Nothing]] =
      List(Step(Outcome.accepted, s.copy(lit = !s.lit)))

  val switch = machine[Lamp, Outcome, Nothing] {
    starts(Lamp(false))
    ends(_ => true)
    steps(flip ~> effects.flipped)
  }

  object properties:
    val turnsOn = switch.property holds (after => after.state.lit)

  object laws:
    val reported = SwitchRealization.reported

  object queries:
    val flipped = switch.scenario.actions(flip)
    val asked = query verify properties.turnsOn in flipped limits one total 2

object Backwards:
  val dark = machine[Lamp, Outcome, Nothing] {
    starts(Lamp(false))
    ends(_ => true)
  }

  object queries:
    val stays = dark.scenario.free

  object properties:
    val unlit = dark.property holds (after => !after.state.lit)

final case class Late(lit: Boolean)

object Misplaced:
  val lamp = machine[Lamp, Outcome, Nothing] {
    starts(Lamp(true))
    ends(_ => true)
  }

  val lit = lamp.property holds (after => after.state.lit)

  object properties:
    val switchLit = Switch.switch.property holds (after => after.state.lit)

object Files:
  val switchFile = irFile("switch")(Switch.switch)
