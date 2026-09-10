import Umpire.Property.Evaluate
import Shared.CorrelatedObligation

/-! Property facade for the shared passive obligation kernel and its checked window theorems. -/
namespace Umpire.Property.Correlated

abbrev Coordinate := Shared.CorrelatedObligation.Coordinate
abbrev Coordinate.mk := Shared.CorrelatedObligation.Coordinate.mk
abbrev Obligation := Shared.CorrelatedObligation.Obligation
abbrev Obligation.consume := Shared.CorrelatedObligation.Obligation.consume
abbrev Obligation.consumeMany := Shared.CorrelatedObligation.Obligation.consumeMany
abbrev consume := Shared.CorrelatedObligation.consume
abbrev consumeMany := Shared.CorrelatedObligation.consumeMany
abbrev consumeMany_append := Shared.CorrelatedObligation.consumeMany_append
abbrev Obligation.resolved_immutable := Shared.CorrelatedObligation.Obligation.resolved_immutable
abbrev Obligation.pending_satisfied := Shared.CorrelatedObligation.Obligation.pending_satisfied
abbrev consumeMany_distributes := Shared.CorrelatedObligation.consumeMany_distributes
abbrev closedReference := Shared.CorrelatedObligation.closedReference
abbrev closed_agrees := Shared.CorrelatedObligation.closed_agrees

/-- A deliberate finite close treats remaining obligations as failed; an incomplete prefix does not. -/
def close (ending : TraceEnding) (obligations : List Obligation) :
    PropertyEndpointAnswer :=
  if obligations.contains .violated then .violated
  else if obligations.all (fun obligation => decide (obligation = .satisfied)) then .satisfied
  else match ending with
    | .final => .violated
    | .«partial» => .unresolved


end Umpire.Property.Correlated
