/* The lamp's bulb, a zoom-in of the System level (fn-126 R20, decisions 16 and 22): a subject of
 * the level, in a file of its own beside system/System.scala and named after it. It keeps its own
 * types, and its `object refinement` says what it refines, here the System.
 */
package fixture.features.lamp
package system

import umpire.*

/** The bulb's filament, the bulb's own type: whether current through it makes it glow. */
final case class Filament(glowing: Boolean) derives Finite

type FilamentStep = Step[Filament, Outcome, Nothing]

/** The bulb in the circuit: closing the circuit heats the filament, opening it cools it. */
object Bulb extends Machine[Filament, Outcome, Nothing]:
  val init = Filament(glowing = false)
  def end(s: State) = true

  object refinement extends Refinement(LampSystem):
    def toProduct(s: State) =
      system.State(if s.glowing then Phase.closed else Phase.open)

  object effects:
    def heat(s: State): List[FilamentStep] = enter(s.copy(glowing = true))
    def cool(s: State): List[FilamentStep] = enter(s.copy(glowing = false))

  object rules extends Rules:
    on(user.switchOn)(where(!_.glowing) ~> effects.heat)
    on(user.switchOff)(where(_.glowing) ~> effects.cool)
