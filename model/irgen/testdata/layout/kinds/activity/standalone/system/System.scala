package fixture.features.activity.standalone
package system

import umpire.*
import product.ActivityProduct

enum Phase derives Finite:
  case idle, done

final case class State(phase: Phase) derives Finite

enum Fact derives Finite:
  case completed

object ActivitySystem extends Machine[system.State, Outcome, Fact]:
  val init = system.State(phase = Phase.idle)
  def end(s: State) = true

  object refinement extends Refinement(ActivityProduct):
    def toProduct(s: State) =
      product.State(if s.phase == Phase.idle then product.Phase.idle else product.Phase.done)

  object effects:
    def complete(s: State) = enter(s.copy(phase = Phase.done), Fact.completed)

  object rules extends Rules:
    on(user.complete)(always ~> effects.complete)
