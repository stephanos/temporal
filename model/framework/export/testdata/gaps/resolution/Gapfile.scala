// fn-167.2: an IR file run refuses a gap on a declaration no IR file lifts: `orphan` is a Property
// of a package no IR file is collected from. Built and lifted apart from the other fixtures.
package fixture.features.gapfile

import framework.*

enum Light:
  case off, on
  case broken extends Light, Terminated

enum Said:
  case ok

given Ok[Said] = Ok(Said.ok)

object hand extends Actor:
  val press = action(this)

object Gapfile extends Machine[Light, Said, Nothing], Phased[Light, Light](l => l, (_, p) => p):
  val init = Light.off
  object effects:
    def press(s: Light) = enter[Light, Said, Nothing](Light.on)
  object rules extends Rules:
    on(hand.press) {
      when(Light.off) ~> effects.press
    }
  object properties:
    object safety:
      @gap(unasked)(because = "asked below")
      val lit = property.holds(s => s.state != Light.broken)
  object queries:
    val free = scenario.free
    val one = Limits(steps = 1, actions = 1, search = 8)
    val litHolds = query verify properties.safety.lit in free limits one
