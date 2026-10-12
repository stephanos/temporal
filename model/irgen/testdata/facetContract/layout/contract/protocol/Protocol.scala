package fixture.features.contract
package protocol

import framework.*

enum Phase:
  case off
  case on extends Phase, Terminated
enum Fact:
  case changed
final case class QueueDetail(phase: Phase)
type State = QueueDetail

object ContractProtocol
    extends Machine[QueueDetail, Outcome, Fact],
      Phased[QueueDetail, Phase](_.phase, (s, p) => s.copy(phase = p)),
      Pressing:
  val init = QueueDetail(Phase.off)
  object refinement extends Refinement(product.ContractProduct):
    def toProduct(s: State) =
      product.QueueView(if s.phase.in(Phase.off) then product.Phase.off else product.Phase.on)
  object monitors:
    val terminalIsSticky =
      sticky((after: Step[State, Outcome, Fact]) => after.state.phase.in(Phase.on))
  object rules extends Facets(pressing.rules)
  object queries:
    val any = scenario.free
    val terminal = query
      .verify(product.ContractProduct.properties.terminal)
      .in(any)
      .limits(Limits("one", 1, 1, 8))
