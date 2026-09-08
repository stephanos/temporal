import Umpire.Property.Authoring
import Testpilot.Authoring
import Umpire.Property.Tests.Scoped.Fixtures

/-! Readable temporal authoring shares the ordinary checked clause and fingerprint. -/
namespace Umpire.Property.ScopedTests

private def written : PropertyScopedClause :=
  bounded_response% (id "test.scoped.response") at source
    whenever (.selectedActionIs request)
    eventually (.modelOutcomeIs response)
    within 1 on .operationTransitions
    scoped [id "test.run"] by (id "test.operation") closing .runtimePrefix

#guard written == { clause 1 with trigger := .selectedActionIs request }

private def authorContext := (targetResult.toOption.map context).getD
  { definitions := [], providers := [], meanings := [] }
private def spec (clause : PropertyScopedClause) : PropertySpec := {
  family := { root := id "test" }
  key := "property"
  source
  requires := [id "test.capability"]
  clauses := []
  scopedClauses := [clause]
}

private def surface := property% (spec written) against authorContext tracking []
private def constructor := (spec { clause 1 with trigger := .selectedActionIs request }).check authorContext
#guard surface.isOk
#guard surface.toOption.map canonicalPropertyJson == constructor.toOption.map canonicalPropertyJson
#guard surface.toOption.map (·.behaviorFingerprint) == constructor.toOption.map (·.behaviorFingerprint)

private def wrongReference : PropertyScopedClause := { written with trigger := .selectedActionIs state }

/-- error: property authoring failed: -/
#guard_msgs (error, substring := true) in
#check property% (spec wrongReference) against authorContext tracking [clauseAnchor (id "test.scoped.response")]

private def unknownReference : PropertyScopedClause := { written with trigger := .selectedActionIs (value "test.missing" "x") }

/-- error: property authoring failed: -/
#guard_msgs (error, substring := true) in
#check property% (spec unknownReference) against authorContext tracking [clauseAnchor (id "test.scoped.response")]

private def wrongScope : PropertyScopedClause := { written with scope := [id "test.operation"] }

/-- error: property authoring failed: -/
#guard_msgs (error, substring := true) in
#check property% (spec wrongScope) against authorContext tracking [clauseAnchor (id "test.scoped.response")]

private def emptyScope : PropertyScopedClause := { written with scope := [] }

/-- error: property authoring failed: -/
#guard_msgs (error, substring := true) in
#check property% (spec emptyScope) against authorContext tracking [clauseAnchor (id "test.scoped.response")]

private def wrongKey : PropertyScopedClause := { written with key := id "" }

/-- error: property authoring failed: {"error":{"kind":"empty-definition-id","definitionId":"test.scoped.response","sourcePath":"Umpire/Property/Tests/Scoped.lean","offendingValue":"<empty>","relatedDefinitionIds":[""]},"role":"clause","anchor":{"sourcePath":"Umpire/Property/Tests/TemporalAuthoring.lean","line":62,"column":78,"endLine":62,"endColumn":105}} -/
#guard_msgs (error, substring := true) in
#check property% (spec wrongKey) against authorContext tracking [clauseAnchor (id "test.scoped.response")]

private def overflowBound : PropertyScopedClause := { written with bound := 18446744073709551616 }

/-- error: property authoring failed: -/
#guard_msgs (error, substring := true) in
#check property% (spec overflowBound) against authorContext tracking [clauseAnchor (id "test.scoped.response")]

private def unsupportedFormula : PropertyScopedClause := { written with response := .all [] }

/-- error: property authoring failed: -/
#guard_msgs (error, substring := true) in
#check property% (spec unsupportedFormula) against authorContext tracking [clauseAnchor (id "test.scoped.response")]

private def wrongStepContext : PropertyScopedClause := { written with trigger := .priorStateIs state }

/-- error: property authoring failed: -/
#guard_msgs (error, substring := true) in
#check property% (spec wrongStepContext) against authorContext tracking [clauseAnchor (id "test.scoped.response")]

/-- error: unexpected type at `evalExpr` -/
#guard_msgs (error, substring := true) in
#check property% (spec written) against source tracking []

/-- error: Unknown constant -/
#guard_msgs (error, substring := true) in
#check (bounded_response% (id "test.scoped.response") at source
  whenever (.selectedActionIs request) eventually (.modelOutcomeIs response)
  within 1 on .logicalTime scoped [id "test.run"] by (id "test.operation") closing .runtimePrefix)

/-- error: failed to synthesize -/
#guard_msgs (error, substring := true) in
#check (bounded_response% (id "test.scoped.response") at source
  whenever (.selectedActionIs request) eventually (.modelOutcomeIs response)
  within (-1) on .operationTransitions scoped [id "test.run"] by (id "test.operation") closing .runtimePrefix)

/-- error: Type mismatch -/
#guard_msgs (error, substring := true) in
#check (bounded_response% (id "test.scoped.response") at source
  whenever (step "a" request) eventually (.modelOutcomeIs response)
  within 1 on .operationTransitions scoped [id "test.run"] by (id "test.operation") closing .runtimePrefix)

/-- error: Unknown constant -/
#guard_msgs (error, substring := true) in
#check (bounded_response% (id "test.scoped.response") at source
  whenever (.selectedActionIs request) eventually (.modelOutcomeIs response)
  within 1 on .operationTransitions scoped [id "test.run"] by (id "test.operation") closing .terminalModel)

/-- error: Type mismatch -/
#guard_msgs (error, substring := true) in
example (raw : Observation.Projection.Event) : PropertyScopedClause :=
  bounded_response% (id "test.scoped.response") at source
    whenever raw eventually (.modelOutcomeIs response)
    within 1 on .operationTransitions scoped [id "test.run"] by (id "test.operation") closing .runtimePrefix

/-- error: Type mismatch -/
#guard_msgs (error, substring := true) in
example (effect : temporal.server.api.testpilot.v1.Instruction) : PropertyScopedClause :=
  bounded_response% (id "test.scoped.response") at source
    whenever effect eventually (.modelOutcomeIs response)
    within 1 on .operationTransitions scoped [id "test.run"] by (id "test.operation") closing .runtimePrefix

end Umpire.Property.ScopedTests
