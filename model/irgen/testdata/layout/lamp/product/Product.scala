// The lamp's Product: what a caller reads (fn-126 R20). The level's own file, named after its
// folder, holds the feature's Product machine, `<Feature>Product`, which refines nothing.
// Product-level elaborations, read by the same audience, would sit beside it, one file per subject.
//
// The two package clauses read the feature's package as well as this one, so its types and its
// signature are in scope.
package fixture.features.lamp
package product

import framework.*

enum Phase derives Finite:
  case dark, lit

final case class State(phase: Phase) derives Finite

enum Fact derives Finite:
  case switchedOn, switchedOff

// The lamp as a caller sees it: switched on, it is lit; switched off, it is dark.
object LampProduct extends Machine[product.State, Outcome, Fact]:
  val init = product.State(phase = Phase.dark)
  def end(s: State) = true

  object states:
    def dark(s: State) = s.phase == Phase.dark

  object effects:
    def light(s: State) = enter(s.copy(phase = Phase.lit), Fact.switchedOn)
    def darken(s: State) = enter(s.copy(phase = Phase.dark), Fact.switchedOff)

  object rules extends Rules:
    on(user.switchOn)(where(states.dark) ~> effects.light)
    on(user.switchOff)(where(_.phase == Phase.lit) ~> effects.darken)

  object properties:
    val switchingOnLights = property when user.switchOn holds
      (after => after.state.phase == Phase.lit)

  object queries:
    val switchedOn = scenario.actions(user.switchOn)
    // 2 states, a pinned Scenario of 1 slot within one step.
    val lights = query verify properties.switchingOnLights in switchedOn limits one total 2

object OnlyOn extends Derived(LampProduct.restrict(user.switchOn))
