// The feature file of a fixture feature (fn-126 R2, R3), lifted with the files beside it. Its
// Switch reads as a feature file is read; the rest do not, at the lines the lint refuses. (c): a
// type after a machine object, a section out of its order, a Scenario after a Query. (d): a
// Property at the top level, in its machine's object rather than its `properties`, or over another
// object's machine; a machine in a type's companion or in an object of the signature; a Property in
// an object of a machine object that is none of its sections; a Query over another object's
// Scenario; a val in exports that is no IR file. (b): Switch's `laws` reads the realization
// (Realization.scala), which reads it back.
package fixture.features.initorder

import umpire.*

given Family = Family("fixture.initorder")

final case class Lamp(lit: Boolean) derives Finite

enum Outcome derives Finite:
  case accepted

final case class Bulb(lit: Boolean)

// A type's companion is no machine object.
object Bulb:
  val bulb = machine[Lamp, Outcome, Nothing] { starts(Lamp(true)); ends(_ => true) }

val flip = action(Party("fixture"))
val one = Limits(steps = 1, actions = 1, search = 8)

// A Model declaration at the top level.
def loose: Property[Lamp] = Switch.switch.property holds (after => after.state.lit)

// An object of the signature holds no machine object.
object Holder:
  object Inner:
    val dim = machine[Lamp, Outcome, Nothing] { starts(Lamp(false)); ends(_ => true) }

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
  object extras:
    val dark = lamp.property holds (after => !after.state.lit)

  val lamp = machine[Lamp, Outcome, Nothing] {
    starts(Lamp(true))
    ends(_ => true)
  }

  val lit = lamp.property holds (after => after.state.lit)

  object properties:
    val switchLit = Switch.switch.property holds (after => after.state.lit)

// A Scenario after a Query, and a Query over another object's Scenario.
object Asked:
  val asked = machine[Lamp, Outcome, Nothing] { starts(Lamp(false)); ends(_ => true) }

  object properties:
    val stays = asked.property holds (after => !after.state.lit)

  object queries:
    val first = query verify properties.stays in asked.scenario("any").free limits one total 2
    val late = asked.scenario.free
    val borrowed = query verify properties.stays in Switch.queries.flipped limits one total 2

object exports:
  val switchFile = irFile("switch")(Switch.switch)
  val note = "the switch" // exports holds IR files alone
