package fixture.features.contract
package product

import framework.dsl.*

enum Phase:
  case off
  case on extends Phase, Terminated
enum Fact:
  case changed
final case class QueueView(phase: Phase)
type State = QueueView

object ContractProduct
    extends Machine[QueueView, Outcome, Fact],
      Phased[QueueView, Phase](_.phase, (s, p) => s.copy(phase = p)),
      Pressing:
  val init = QueueView(Phase.off)
  object rules extends Facets(pressing.rules)
  object properties:
    object safety:
      val terminal = property.holds(_.state.phase.in(Phase.on))
  object queries:
    val pressed = witness(hand.press).records(Fact.changed)
