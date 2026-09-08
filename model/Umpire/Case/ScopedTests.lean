import Umpire.Case.Tests.ScopedFixtures

namespace Umpire.Case.ScopedTests
open ScopedFixtures
open Umpire.Property.ScopedTests
open temporal.server.api.testpilot.v1

private def first := scenarios[0]
private def accepts (change : ScopedContract → ScopedContract) : Bool :=
  ((capability first).bind (fun wire => Testpilot.Scoped.decode (change wire))).isOk

#guard !accepts (fun wire => { wire with version := 2 })
#guard !accepts (fun wire => { wire with version := 0 })
#guard !accepts (fun wire => { wire with clauses := wire.clauses.map fun clause => { clause with bound := -1 } })
#guard !accepts (fun wire => { wire with clauses := wire.clauses.map fun clause => {
  clause with clock := .SCOPED_CLOCK_UNSPECIFIED } })
#guard !accepts (fun wire => { wire with limits := wire.limits.map fun limits => { limits with max_events := 0 } })
#guard !accepts (fun wire => { wire with projection_rules := wire.projection_rules.map fun rule => {
  rule with outputs := rule.outputs.map fun output => { output with prior_state := wire.initial_state } } })

private def admitted (change : ScopedEvidence → ScopedEvidence) : Bool :=
  (do
    let compiled ← Testpilot.Scoped.decode (← capability first)
    let initial ← compiled.start scope
    let _ ← initial.observe 2 (change (evidence 0 "both"))
    pure () : Except String Unit).isOk

#guard !admitted (fun event => { event with identity := event.identity.map fun identity => { identity with source := "unknown" } })
#guard !admitted (fun event => { event with identity := event.identity.map fun identity => { identity with scope := #[] } })
#guard !admitted (fun event => { event with fields := #[{ field_id := "unknown" }] })
#guard !admitted (fun event => { event with parents := event.identity.toArray })

#guard !(capability { first with bound := 9223372036854775808 }).isOk

private def boundary (projectionWork obligationWork support : Int64) (eventSize : Int64 := 512) : Bool :=
  (do
    let wire ← capability first
    let wire := { wire with limits := wire.limits.map fun limits => { limits with
      max_projection_work := projectionWork, max_obligation_work := obligationWork, max_support := support,
      max_event_bytes := eventSize } }
    let compiled ← Testpilot.Scoped.decode wire
    let initial ← compiled.start scope
    let _ ← initial.observe 2 (evidence 0 "both")
    pure () : Except String Unit).isOk

#guard boundary 2960 49 5 36
#guard !boundary 2960 49 5 35
#guard boundary 2960 49 5
#guard !boundary 2959 49 5
#guard !boundary 2960 48 5
#guard !boundary 2960 49 4
#guard (do
  let target ← targetResult.toOption
  let projection ← (plan target).toOption
  pure (projection.canonicalBehavior.startsWith "[\"checked-projection/v2\"")) == some true

/-- info: 'Umpire.Case.Scoped.Lowered.window_property' depends on axioms: [propext, Classical.choice, Quot.sound] -/
#guard_msgs in
#print axioms Umpire.Case.Scoped.Lowered.window_property
/-- info: 'Umpire.Case.Scoped.Lowered.observed_property' depends on axioms: [propext, Classical.choice, Quot.sound] -/
#guard_msgs in
#print axioms Umpire.Case.Scoped.Lowered.observed_property
/-- info: 'Umpire.Case.ScopedProofs.endpoint_agrees' depends on axioms: [propext] -/
#guard_msgs in
#print axioms Umpire.Case.ScopedProofs.endpoint_agrees
/-- info: 'Umpire.Case.Scoped.Lowered.evidence_validation' depends on axioms: [propext, Classical.choice, Quot.sound] -/
#guard_msgs in
#print axioms Umpire.Case.Scoped.Lowered.evidence_validation
/-- info: 'Testpilot.Scoped.Run.observe' depends on axioms: [propext, Classical.choice, Quot.sound] -/
#guard_msgs in
#print axioms Testpilot.Scoped.Run.observe

end Umpire.Case.ScopedTests
