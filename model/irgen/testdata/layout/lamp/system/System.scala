/* The lamp's System: how the server gets there (fn-126 R20). The level's own file, named after its
 * folder, holds the feature's System machine, `<Feature>System`, whose `object refinement` refines
 * the Product. Zoom-ins on how it keeps its promise sit beside it, one file per subject, as
 * Bulb.scala does.
 */
package fixture.features.lamp
package system

import umpire.*
import product.LampProduct

enum Phase derives Finite:
  case open, closed

final case class State(phase: Phase) derives Finite

enum Fact derives Finite:
  case circuitClosed, circuitOpened

/** The lamp as the circuit runs it: the switch closes and opens the circuit. */
object LampSystem extends Machine[system.State, Outcome, Fact]:
  val init = system.State(phase = Phase.open)
  def end(s: State) = true

  // What a caller reads of the circuit: the lamp is lit while the circuit is closed.
  object refinement extends Refinement(LampProduct):
    def toProduct(s: State) =
      product.State(if s.phase == Phase.closed then product.Phase.lit else product.Phase.dark)

  object effects:
    def close(s: State) = enter(s.copy(phase = Phase.closed), Fact.circuitClosed)
    def open(s: State) = enter(s.copy(phase = Phase.open), Fact.circuitOpened)

  object rules extends Rules:
    on(user.switchOn)(where(_.phase == Phase.open) ~> effects.close)
    on(user.switchOff)(where(_.phase == Phase.closed) ~> effects.open)

object OnlyClosed extends Derived(LampSystem.restrict(user.switchOn))
