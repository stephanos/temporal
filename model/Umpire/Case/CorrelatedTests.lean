import Umpire.Case.Tests.CorrelatedFixtures
import Umpire.Shared.Test

namespace Umpire.Case.CorrelatedTests
open CorrelatedFixtures
open Umpire.Property.CorrelatedTests
open temporal.server.api.testpilot.v1

private def first := scenarios[0]
private def accepts (change : CorrelatedContract → CorrelatedContract)
    (limit : CorrelatedLimits → CorrelatedLimits := id) : Bool :=
  (do Testpilot.Correlated.decode (limit (← ceilings first)) (change (← capability first))).isOk

#guard !accepts (fun wire => { wire with rules := wire.rules.map fun clause => { clause with bound := -1 } })
#guard !accepts (fun wire => { wire with rules := wire.rules.map fun clause => {
  clause with clock := .CORRELATED_CLOCK_UNSPECIFIED } })
#guard !accepts id (fun limits => { limits with max_events := 0 })
#guard !accepts (fun wire => { wire with projection_rules := wire.projection_rules.map fun rule => {
  rule with outputs := rule.outputs.map fun output => { output with prior_state := wire.initial_state } } })

private def admitted (change : CorrelatedEvidence → CorrelatedEvidence) : Bool :=
  (do
    let compiled ← Testpilot.Correlated.decode (← ceilings first) (← capability first)
    let initial ← compiled.start caseScope
    let _ ← initial.observe 2 (change (evidence 0 "both"))
    pure () : Except String Unit).isOk

#guard !admitted (fun event => { event with identity := event.identity.map fun identity => { identity with evidence_source := "unknown" } })
#guard !admitted (fun event => { event with identity := event.identity.map fun identity => { identity with scope := #[] } })
#guard !admitted (fun event => { event with identity := event.identity.map fun identity => {
  identity with scope := identity.scope.map fun binding => { binding with value := some { value := some (.unsigned_integer_value "1") } } } })
#guard admitted id
#guard !admitted (fun event => { event with fields := #[{ field_id := "unknown" }] })
#guard !admitted (fun event => { event with parents := event.identity.toArray })

#guard !(capability { first with bound := 9223372036854775808 }).isOk

private def boundary (projectionWork obligationWork support : Int64) (eventSize : Int64 := 512) : Bool :=
  (do
    let limits ← ceilings first
    let limits := { limits with
      max_projection_work := projectionWork, max_obligation_work := obligationWork, max_support := support,
      max_event_bytes := eventSize }
    let compiled ← Testpilot.Correlated.decode limits (← capability first)
    let initial ← compiled.start caseScope
    let _ ← initial.observe 2 (evidence 0 "both")
    pure () : Except String Unit).isOk

-- Projection work and event bytes count identifier bytes, so the Case-local names this evidence
-- carries set both boundaries.
#guard boundary 1760 49 5 21
#guard !boundary 1760 49 5 20
#guard boundary 1760 49 5
#guard !boundary 1759 49 5
#guard !boundary 1760 48 5
#guard !boundary 1760 49 4
#guard (do
  let target ← targetResult.toOption
  let projection ← (plan target).toOption
  pure (projection.behaviorVersion.startsWith "[\"checked-projection/v2\"")) == some true

-- `Lowered.window_property` and `Lowered.evidence_validation` are each checked once, here.
assert_axioms [Umpire.Case.Correlated.Lowered.window_property,
  Umpire.Case.Correlated.Lowered.observed_property, Umpire.Case.CorrelatedProofs.endpoint_agrees,
  Umpire.Case.Correlated.Lowered.evidence_validation, Testpilot.Correlated.Monitor.observe]
  allowing [propext, Classical.choice, Quot.sound]

end Umpire.Case.CorrelatedTests
