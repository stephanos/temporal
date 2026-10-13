// The facet rules that land switched off (fn-161 R8, R9; model/framework/export/Switches.scala), each refused
// when its switch is on: a declared machine with rules and effects of its own (`facets-only`); a
// facet named outside its feature's vocabulary, and an action owned by differently named facets at
// the Product and the Protocol level (`facet-vocabulary`). `Agreeing` keeps both rules.
package fixture.facetrules

import framework.*

enum FacetName:
  case pressing, breaking

enum Lit:
  case off, on
  case broken extends Lit, Terminated

final case class State(light: Lit)
final case class AdmissionState(light: Lit)

enum Said:
  case ok

given Ok[Said] = Ok(Said.ok)

object hand extends Actor:
  val press = action(this)

trait Pressing extends Facet[State, Said, Nothing], Phased[State, Lit]:
  object pressing:
    object rules extends Rules:
      on(hand.press) {
        when(Lit.off) ~> enters(Lit.on)
      }

trait Breaking extends Facet[State, Said, Nothing], Phased[State, Lit]:
  object breaking:
    object rules extends Rules:
      on(hand.press) {
        when(Lit.off) ~> enters(Lit.broken)
      }

trait Smashing extends Facet[State, Said, Nothing], Phased[State, Lit]:
  object smashing:
    object rules extends Rules:
      on(hand.press) {
        when(Lit.off) ~> enters(Lit.broken)
      }

object custom:
  trait Breaking extends Facet[AdmissionState, Said, Nothing], Phased[AdmissionState, Lit]:
    object breaking:
      object rules extends Rules:
        on(hand.press) {
          when(Lit.off) ~> enters(Lit.on)
        }

object Agreeing
    extends Machine[State, Said, Nothing],
      Phased[State, Lit](_.light, (s, p) => s.copy(light = p)),
      Pressing:
  val init = State(Lit.off)
  object rules extends Facets(pressing.rules)

object Unfaceted
    extends Machine[State, Said, Nothing],
      Phased[State, Lit](_.light, (s, p) => s.copy(light = p)):
  val init = State(Lit.off)
  object rules extends Rules:
    on(hand.press) {
      when(Lit.off) ~> enters(Lit.on)
    }

object Unnamed
    extends Machine[State, Said, Nothing],
      Phased[State, Lit](_.light, (s, p) => s.copy(light = p)),
      Smashing:
  val init = State(Lit.off)
  object rules extends Facets(smashing.rules)

object Crossed
    extends Machine[State, Said, Nothing],
      Phased[State, Lit](_.light, (s, p) => s.copy(light = p)),
      Breaking:
  val init = State(Lit.off)
  object refinement extends Refinement(Agreeing):
    def toProduct(s: State) = s
  object rules extends Facets(breaking.rules)

object Design
    extends Machine[AdmissionState, Said, Nothing],
      Phased[AdmissionState, Lit](_.light, (s, p) => s.copy(light = p)),
      custom.Breaking:
  val init = AdmissionState(Lit.off)
  object refinement extends Refinement(Agreeing):
    def toProduct(s: AdmissionState) = State(s.light)
  object rules extends Facets(breaking.rules)
