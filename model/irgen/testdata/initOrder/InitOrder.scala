// The feature file of a fixture feature (fn-126 R2, R3), lifted with the files beside it. Its
// Switch reads as a feature file is read; the rest do not, at the lines the lint refuses. (c): a
// type after a machine object, a section out of its order, a Scenario after a Query. (d): a
// Property at the top level, in its machine's object rather than its `properties`, or over another
// object's machine; a machine in a type's companion or in an object of the signature; a Property in
// an object of a machine object that is none of its sections; a Query over another object's
// Scenario; a val in exports that is no IR file. (b): Switch's `implements` reads the realization
// (Realization.scala), which reads it back, and so does Dimmer's `capabilities`, through its base.
package fixture.features.initorder

import umpire.*

final case class Lamp(lit: Boolean) derives Finite

enum Outcome derives Finite:
  case accepted

final case class Bulb(lit: Boolean)

// A type's companion is no machine object.
object Bulb:
  object BulbLamp extends Derived(Switch.unmonitored)

val flip = action(Actor("fixture"))
val one = Limits(steps = 1, actions = 1, search = 8)

// A Model declaration at the top level.
def loose: Property[Lamp] = Switch.property holds (after => after.state.lit)

// An object of the signature holds no machine object.
object Holder:
  object Inner:
    object Dim extends Derived(Switch.unmonitored)

object Switch extends Machine[Lamp, Outcome, Nothing]:
  val init = Lamp(false)
  def end(lamp: State) = true

  object states:
    def lit(s: Lamp) = s.lit

  object effects:
    def flipped(s: Lamp): List[Step[Lamp, Outcome, Nothing]] =
      List(Step(Outcome.accepted, s.copy(lit = !s.lit)))

  object rules extends Rules:
    on(flip)(always ~> effects.flipped)

  object properties:
    val turnsOn = property holds (after => after.state.lit)

  object implements:
    val reported = SwitchRealization.reported

  object queries:
    val flipped = scenario.actions(flip)
    val asked = query verify properties.turnsOn in flipped limits one total 2

object Backwards extends Machine[Lamp, Outcome, Nothing]:
  val init = Lamp(false)
  def end(lamp: State) = true

  object rules extends Rules

  object queries:
    val stays = scenario.free

  object properties:
    val unlit = property holds (after => !after.state.lit)

final case class Late(lit: Boolean)

object Misplaced extends Machine[Lamp, Outcome, Nothing]:
  val init = Lamp(true)
  def end(lamp: State) = true

  object extras:
    val dark = property holds (after => !after.state.lit)

  object rules extends Rules

  val lit = property holds (after => after.state.lit)

  object properties:
    val switchLit = Switch.property holds (after => after.state.lit)

// A Scenario after a Query, and a Query over another object's Scenario.
object Asked extends Machine[Lamp, Outcome, Nothing]:
  val init = Lamp(false)
  def end(lamp: State) = true

  object rules extends Rules

  object properties:
    val stays = property holds (after => !after.state.lit)

  object queries:
    val first = query verify properties.stays in scenario("any").free limits one total 2
    val late = scenario.free
    val borrowed = query verify properties.stays in Switch.queries.flipped limits one total 2

object Dimmer extends Machine[Lamp, Outcome, Nothing]:
  val init = Lamp(false)
  def end(lamp: State) = true

  object rules extends Rules

  object capabilities extends Dimmed(this)

object exports:
  val switchFile = irFile("switch")(Switch)
  val note = "the switch" // exports holds IR files alone
