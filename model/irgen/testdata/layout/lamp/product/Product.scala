/* The lamp's Product: what a caller reads (fn-126 R20). The level's own file, named after its
 * folder, holds the feature's Product machine, `<Feature>Product`, which refines nothing.
 * Product-level elaborations, read by the same audience, would sit beside it, one file per subject.
 *
 * The two package clauses read the feature's package as well as this one, so its types, its
 * signature and its given Family are in scope.
 */
package fixture.features.lamp
package product

import umpire.*

/** The lamp as a caller sees it: switched on, it is lit; switched off, it is dark. */
object LampProduct extends Machine[Lamp, Outcome, Nothing]:
  val init = Lamp(lit = false)
  def end(s: State) = true

  object states extends Section:
    def dark(s: State) = !s.lit

  object effects extends Section:
    def light(s: State): List[LampStep] = enter(s.copy(lit = true))
    def darken(s: State): List[LampStep] = enter(s.copy(lit = false))

  object rules extends Rules:
    when(states.dark)(user.switchOn ~> effects.light)
    when(_.lit)(user.switchOff ~> effects.darken)

  object properties extends Section:
    val switchingOnLights = property when user.switchOn holds (after => after.state.lit)

  object queries extends Section:
    val switchedOn = scenario.actions(user.switchOn)
    // 2 states, a pinned Scenario of 1 slot within one step.
    val lights = query verify properties.switchingOnLights in switchedOn limits one total 2
