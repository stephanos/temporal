package fixture.features.activity.standalone
package product

import framework.*

enum Phase derives Finite:
  case idle, done

final case class State(phase: Phase) derives Finite

enum Fact derives Finite:
  case completed

object ActivityProduct extends Machine[product.State, Outcome, Fact]:
  val init = product.State(phase = Phase.idle)
  def end(s: State) = true

  object effects:
    def complete(s: State) = enter(s.copy(phase = Phase.done), Fact.completed)

  object rules extends Rules:
    on(user.complete)(always ~> effects.complete)
