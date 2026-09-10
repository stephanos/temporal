import Temporal.Feature.Nexus.Success.Model
import Temporal.Case.Template

/-!
# Nexus.Success Testpilot producer

The checked model is carried into a Case; it is never compared against an expected one. Nothing
Temporal is written here any more: the Program, its history attributes, and the Definition IDs its
coordinates carry belong to the `nexusOperation` realization template, and the derivation --
correlated clauses from `require` lines, the projection read off the checked Machine along the
witness trace, provenance, and coverage -- belongs to `Umpire.Case.Producer`.

What is left is the three things this Case decides: which template realizes it, which fixture name
its identities derive from, and which recorded history event confirms each Action the Scenario
selects.
-/

namespace Temporal.Feature.Nexus.Success.Producer

open Umpire
open Umpire.Case.Compiler
open Testpilot.Authoring
open temporal.server.api.testpilot.v1

/-- The async Nexus realization: a controller-started workflow schedules one operation the handler
answers asynchronously, and the controller reads the history back. -/
def realization : Case.Producer.Realization :=
  Temporal.Case.Template.nexusOperation "umpire.case.service" "complete" .async

/-- The identity today's checked-in fixture carries. The Program ID predates the fixture-derived
convention, so it is stated rather than derived; every other identity is the derivation. -/
def identity : Case.Producer.Identity := {
  caseId := "temporal.case.async-nexus-success"
  fixture := "async-nexus"
  programId := "temporal.case.async-nexus.program" }

/-- Which recorded history event confirms each Action the Scenario selects. Both Actions are named
by their declared spelling, so a Model that renames one rejects at production rather than silently
mapping the wrong event. -/
def evidence (vocabulary : Case.Producer.Vocabulary) : List Case.Producer.EvidenceMapping := [
  { «action» := vocabulary.namedAction "awaitStart"
    eventKind := Temporal.Case.Template.NexusOperation.startedSource.eventKind },
  { «action» := vocabulary.namedAction "awaitSuccess"
    eventKind := Temporal.Case.Template.NexusOperation.completedSource.eventKind }]

/-- The checked Nexus.Success authoring bundle as the Umpire-owned Producer input. A `.umpire`
module may not import this namespace, so the conversion lives here. -/
def producerInput {Setup State Action Outcome Fact : Type}
    [BEq Setup] [BEq State] [BEq Action] [BEq Outcome] [BEq Fact]
    {«model» : Authoring.SuccessModel Setup State Action Outcome Fact}
    (checked : Authoring.CheckedModel «model») :
    Case.Producer.Input «model».lawStatement := {
  target := checked.target
  vocabulary := {
    «states» := checked.vocabulary.states
    «actions» := checked.vocabulary.actions
    «outcomes» := checked.vocabulary.outcomes
    «facts» := checked.vocabulary.facts }
  «property» := checked.property
  «scenario» := checked.behavior
  «witness» := checked.witness
  operationRole := «model».operationRoleId
  queryId := checked.query.id
  querySource := checked.query.source
  queryFingerprint := checked.query.behaviorFingerprint.render
  knownGaps := checked.query.authoredKnownGaps
  source := Authoring.source }

/-- Lower one checked Nexus.Success model into a Case. The checked values are carried, never compared
against an expected model: a different Target, Behavior, Query or Property produces different Case
bytes. Only a claim this Producer cannot realize rejects.

`required` names clauses the caller requires the Case to carry, beyond the ones the checked Property
already names. Coverage is always requested explicitly, never left to a default. -/
def produce {Setup State Action Outcome Fact : Type}
    [BEq Setup] [BEq State] [BEq Action] [BEq Outcome] [BEq Fact]
    {«model» : Authoring.SuccessModel Setup State Action Outcome Fact}
    (checked : Authoring.CheckedModel «model»)
    (required : List DefinitionId := []) :
    Except Error temporal.server.api.testpilot.v1.Case :=
  let input := producerInput checked
  Case.Producer.produce input identity realization (evidence input.vocabulary) required

private def compilerError (definitionId construct : String) : Error := {
  sourceDefinitionId := definitionId
  source := Authoring.source
  construct
}

private def checkedCompletion : Except Error (Authoring.CheckedModel lifecycle) :=
  completion.mapError fun _ =>
    compilerError "temporal.nexus.success.query.completion" "checked-completion"

/-- The checked Nexus.Success completion declaration lowered to the closed Case format. -/
def completionCase : Except Error temporal.server.api.testpilot.v1.Case := do
  produce (← checkedCompletion)

end Temporal.Feature.Nexus.Success.Producer
