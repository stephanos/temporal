// The facet rules that land switched off (fn-161 R8, R9; model/irgen/Switches.scala), each refused
// when its switch is on: a declared machine with rules and effects of its own (`facets-only`); a
// facet named outside its feature's vocabulary, and an action owned by differently named facets at
// the Product and the Protocol level (`facet-vocabulary`). `Agreeing` keeps both rules.
package fixture.facetrules

import framework.*

enum FacetName derives Finite:
  case pressing, breaking

enum Lit derives Finite:
  case off, on, broken extends Lit, Terminated

final case class Lamp(light: Lit) derives Finite

enum Said derives Finite:
  case ok

given Ok[Said] = Ok(Said.ok)

object hand extends Actor:
  val press = action(this)

trait Pressing extends Facet[Lamp, Said, Nothing], Phased[Lamp, Lit]:
  object pressing:
    object rules extends Rules:
      on(hand.press) {
        when(Lit.off) ~> enters(Lit.on)
      }

trait Breaking extends Facet[Lamp, Said, Nothing], Phased[Lamp, Lit]:
  object breaking:
    object rules extends Rules:
      on(hand.press) {
        when(Lit.off) ~> enters(Lit.broken)
      }

trait Smashing extends Facet[Lamp, Said, Nothing], Phased[Lamp, Lit]:
  object smashing:
    object rules extends Rules:
      on(hand.press) {
        when(Lit.off) ~> enters(Lit.broken)
      }

object Agreeing
    extends Machine[Lamp, Said, Nothing],
      Phased[Lamp, Lit](_.light, (s, p) => s.copy(light = p)),
      Pressing:
  val init = Lamp(Lit.off)
  object rules extends Facets(pressing.rules)

object Unfaceted
    extends Machine[Lamp, Said, Nothing],
      Phased[Lamp, Lit](_.light, (s, p) => s.copy(light = p)):
  val init = Lamp(Lit.off)
  object rules extends Rules:
    on(hand.press) {
      when(Lit.off) ~> enters(Lit.on)
    }

object Unnamed
    extends Machine[Lamp, Said, Nothing],
      Phased[Lamp, Lit](_.light, (s, p) => s.copy(light = p)),
      Smashing:
  val init = Lamp(Lit.off)
  object rules extends Facets(smashing.rules)

object Crossed
    extends Machine[Lamp, Said, Nothing],
      Phased[Lamp, Lit](_.light, (s, p) => s.copy(light = p)),
      Breaking:
  val init = Lamp(Lit.off)
  object refinement extends Refinement(Agreeing):
    def toProduct(s: Lamp) = s
  object rules extends Facets(breaking.rules)
