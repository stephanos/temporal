import Umpire.Case.Producer

/-!
Pins for the parts of the generic Producer that are decided before any checked Model is in hand:
the identities a fixture name derives, and how a declared spelling resolves against the checked
vocabulary. The derivation itself is pinned end to end by the checked-in fixtures, which regenerate
through `produce`.
-/

namespace Umpire.Case.Tests.Producer

open Umpire.Case.Producer

/-- The fixture name is the only identity slot: everything else derives from it. -/
private def derived : Identity := Identity.ofFixture "async-nexus"

#guard derived.caseId == "temporal.case.async-nexus"
#guard derived.programId == "temporal.case.async-nexus.program"
#guard derived.contractId == "temporal.case.async-nexus.contract"
#guard derived.runScope == "async-nexus"

/-- A stated Program ID overrides the derivation without disturbing the others; that is how a Case
whose bytes predate the convention keeps them. -/
private def stated : Identity := {
  caseId := "temporal.case.async-nexus-success"
  fixture := "async-nexus"
  programId := "temporal.case.async-nexus.program" }

#guard stated.contractId == "temporal.case.async-nexus-success.contract"
#guard stated.runScope == "async-nexus"

private def stateId : DefinitionId := .of "example.state"
private def actionId : DefinitionId := .of "example.action"

private def declared : Case.Producer.Vocabulary := {
  «states» := [ModelValue.named stateId "pending", ModelValue.named stateId "completed"]
  «actions» := [ModelValue.named actionId "awaitCompletion"]
  «outcomes» := []
  «facts» := [] }

#guard (Vocabulary.namedState declared "completed").value == "completed"
#guard (Vocabulary.stateAt declared 0).value == "pending"

/-! An unknown spelling resolves to the unknown Model Value rather than to a neighbour, so the
clause referencing it is rejected at admission instead of silently addressing the wrong member. -/
#guard Vocabulary.namedAction declared "awaitStart" == unknownValue
#guard Vocabulary.outcomeAt declared 0 == unknownValue

end Umpire.Case.Tests.Producer
