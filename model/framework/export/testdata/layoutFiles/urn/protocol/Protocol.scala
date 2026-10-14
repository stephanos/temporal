package temporal.features.urn
package protocol

import framework.dsl.*
import temporal.features.urn.realization.UrnRun

val outward = UrnRun.machine.name

object UrnProtocol extends Machine[Water, Outcome, Nothing]:
  val init = Water(hot = false)
  def end(s: State) = true

  object refinement extends Refinement(product.UrnProduct):
    def toProduct(s: State) = s

  object effects:
    def flip(s: State) = enter(s.copy(hot = !s.hot))

  object rules extends Rules:
    on(user.flip)(always ~> effects.flip)

object BoilSpare extends Derived(UrnProtocol.restrict(user.flip))

