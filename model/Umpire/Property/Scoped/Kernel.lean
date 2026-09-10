import Umpire.Property.Evaluate
import Shared.ScopedObligation

/-! Property facade for the shared passive obligation kernel and its checked window theorems. -/
namespace Umpire.Property.Scoped

abbrev Coordinate := Shared.ScopedObligation.Coordinate
abbrev Coordinate.mk := Shared.ScopedObligation.Coordinate.mk
abbrev Obligation := Shared.ScopedObligation.Obligation
abbrev Obligation.consume := Shared.ScopedObligation.Obligation.consume
abbrev Obligation.consumeMany := Shared.ScopedObligation.Obligation.consumeMany
abbrev consume := Shared.ScopedObligation.consume
abbrev consumeMany := Shared.ScopedObligation.consumeMany
abbrev consumeMany_append := Shared.ScopedObligation.consumeMany_append
abbrev Obligation.resolved_immutable := Shared.ScopedObligation.Obligation.resolved_immutable
abbrev Obligation.pending_satisfied := Shared.ScopedObligation.Obligation.pending_satisfied
abbrev consumeMany_distributes := Shared.ScopedObligation.consumeMany_distributes
abbrev closedReference := Shared.ScopedObligation.closedReference
abbrev closed_agrees := Shared.ScopedObligation.closed_agrees

/-- A deliberate finite close treats remaining obligations as failed; an incomplete prefix does not. -/
def close (endpoint : PropertyScopedEndpoint) (obligations : List Obligation) :
    PropertyEndpointAnswer :=
  if obligations.contains .violated then .violated
  else if obligations.all (fun obligation => decide (obligation = .satisfied)) then .satisfied
  else match endpoint with
    | .deliberatelyClosed => .violated
    | .runtimePrefix => .unresolved


end Umpire.Property.Scoped
