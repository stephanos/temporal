package temporal.features.urn
package product

import framework.*
import temporal.features.urn.protocol.UrnProtocol

val upward = UrnProtocol.name

object UrnProduct extends Machine[Water, Outcome, Nothing]:
  val init = Water(hot = false)
  def end(s: State) = true

  object effects:
    def flip(s: State) = enter(s.copy(hot = !s.hot))

  object rules extends Rules:
    on(user.flip)(always ~> effects.flip)

