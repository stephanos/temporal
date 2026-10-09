package fixture.features.kiln
package product

import framework.*

object KilnProduct extends Machine[Kiln, Outcome, Nothing]:
  val init = Kiln(hot = false)
  def end(s: State) = true

  object effects:
    def fired(s: State): List[KilnStep] = enter(s.copy(hot = true))

  object rules extends Rules:
    on(potter.fire)(where(!_.hot) ~> effects.fired)

  // A section of a name none of a machine's sections has.
  object timers:
    val cooling = timer

  // An object of the machine's that is none of its sections.
  object helpers:
    def cold(s: Kiln) = !s.hot
