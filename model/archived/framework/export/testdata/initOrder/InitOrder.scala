// The feature file of a fixture feature (fn-126 R2, R3), lifted with the files beside it. Its
// Switch reads as a feature file is read; the rest do not, at the lines the lint refuses. (c): a
// type after a machine object, a section out of its order, a Scenario after a Query. (d): a
// Property at the top level, in its machine's object rather than its `properties`, or over another
// object's machine; a machine in a type's companion or in an object of the signature; a Property in
// an object of a machine object that is none of its sections; a Query over another object's
// Scenario; a val in exports that is no IR file. (b): Dimmer's `capabilities` reads the realization
// (Realization.scala), which reads it back through its base class.
package fixture.features.initorder

import framework.dsl.*

final case class Lamp(lit: Boolean)

enum Outcome:
  case accepted

final case class Bulb(lit: Boolean)

// A type's companion is no machine object.
object Bulb:
  object BulbLamp extends Derived(Switch.unmonitored)

val flip = action(Actor("fixture"))
val one = Limits(steps = 1, actions = 1, search = 8)

// A Model declaration at the top level.
def loose: Property[Lamp] = Switch.property expect(check(after.state.lit))

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
    object safety:
      val turnsOn = property expect(check(after.state.lit))

  // The removed legacy section occupied these lines; keep later fixture locations
  // stable so their diagnostics remain easy to compare.

  object queries:
    val flipped = scenario.actions(flip)
    val asked = query verify properties.safety.turnsOn in flipped limits one

object Backwards extends Machine[Lamp, Outcome, Nothing]:
  val init = Lamp(false)
  def end(lamp: State) = true

  object rules extends Rules

  object queries:
    val stays = scenario.free

  object properties:
    object safety:
      val unlit = property expect(check(!after.state.lit))
final case class Late(lit: Boolean)

object Misplaced extends Machine[Lamp, Outcome, Nothing]:
  val init = Lamp(true)
  def end(lamp: State) = true

  object extras:
    val dark = property expect(check(!after.state.lit))

  object rules extends Rules

  val lit = property expect(check(after.state.lit))
  object properties:
    object safety:
      val switchLit = Switch.property expect(check(after.state.lit))

// A Scenario after a Query, and a Query over another object's Scenario.
object Asked extends Machine[Lamp, Outcome, Nothing]:
  val init = Lamp(false)
  def end(lamp: State) = true

  object rules extends Rules
  object properties:
    object safety:
      val stays = property expect(check(!after.state.lit))

  object queries:
    val first = query verify properties.safety.stays in scenario("any").free limits one
    val late = scenario.free
    val borrowed = query verify properties.safety.stays in Switch.queries.flipped limits one

object Dimmer extends Machine[Lamp, Outcome, Nothing]:
  val init = Lamp(false)
  def end(lamp: State) = true

  object rules extends Rules

  object capabilities extends Dimmed(this)
