import Umpire.Property.Elab
import Testpilot.Authoring
import Umpire.Property.Tests.Correlated.Fixtures

/-! Readable temporal authoring shares the ordinary checked clause and fingerprint. -/
namespace Umpire.Property.CorrelatedTests

private def written : PropertyCorrelatedClause :=
  correlated_response% (id "test.correlated.response") at source
    whenever (.selectedActionIs request)
    eventually (.outcomeIs response)
    within 1
    correlated [id "test.run"] by (id "test.operation") closing .«partial»

#guard written == { clause 1 with trigger := .selectedActionIs request }

private def authorContext := (targetResult.toOption.map context).getD
  { definitions := [], providers := [], meanings := [] }
private def spec (clause : PropertyCorrelatedClause) : Property := {
  id := (DefinitionFamily.mk (id "test")).id "property" "property"
  source
  requires := [id "test.capability"]
  clauses := []
  correlatedRules := [clause]
}

private def surface := property% (spec written) against authorContext tracking []
private def constructor := (spec { clause 1 with trigger := .selectedActionIs request }).check authorContext
#guard surface.isOk
#guard surface.toOption.map canonicalPropertyJson == constructor.toOption.map canonicalPropertyJson
#guard surface.toOption.map (·.behaviorFingerprint) == constructor.toOption.map (·.behaviorFingerprint)

private def wrongReference : PropertyCorrelatedClause := { written with trigger := .selectedActionIs state }

/-- error: property authoring failed: -/
#guard_msgs (error, substring := true) in
#check property% (spec wrongReference) against authorContext tracking [clauseAnchor (id "test.correlated.response")]

private def unknownReference : PropertyCorrelatedClause := { written with trigger := .selectedActionIs (value "test.missing" "x") }

/-- error: property authoring failed: -/
#guard_msgs (error, substring := true) in
#check property% (spec unknownReference) against authorContext tracking [clauseAnchor (id "test.correlated.response")]

private def wrongScope : PropertyCorrelatedClause := { written with scope := [id "test.operation"] }

/-- error: property authoring failed: -/
#guard_msgs (error, substring := true) in
#check property% (spec wrongScope) against authorContext tracking [clauseAnchor (id "test.correlated.response")]

private def emptyScope : PropertyCorrelatedClause := { written with scope := [] }

/-- error: property authoring failed: -/
#guard_msgs (error, substring := true) in
#check property% (spec emptyScope) against authorContext tracking [clauseAnchor (id "test.correlated.response")]

private def wrongKey : PropertyCorrelatedClause := { written with key := id "" }

/-- error: property authoring failed: {"error":{"kind":"empty-definition-id","definitionId":"test.correlated.response","sourcePath":"Umpire/Property/Tests/Correlated.lean","offendingValue":"<empty>","relatedDefinitionIds":[""]},"role":"clause","anchor":{"sourcePath":"Umpire/Property/Tests/TemporalAuthoring.lean","line":61,"column":78,"endLine":61,"endColumn":109}} -/
#guard_msgs (error, substring := true) in
#check property% (spec wrongKey) against authorContext tracking [clauseAnchor (id "test.correlated.response")]

private def overflowBound : PropertyCorrelatedClause := { written with bound := 18446744073709551616 }

/-- error: property authoring failed: -/
#guard_msgs (error, substring := true) in
#check property% (spec overflowBound) against authorContext tracking [clauseAnchor (id "test.correlated.response")]

private def unsupportedFormula : PropertyCorrelatedClause := { written with response := .all [] }

/-- error: property authoring failed: -/
#guard_msgs (error, substring := true) in
#check property% (spec unsupportedFormula) against authorContext tracking [clauseAnchor (id "test.correlated.response")]

private def wrongStepContext : PropertyCorrelatedClause := { written with trigger := .priorStateIs state }

/-- error: property authoring failed: -/
#guard_msgs (error, substring := true) in
#check property% (spec wrongStepContext) against authorContext tracking [clauseAnchor (id "test.correlated.response")]

/-- error: unexpected type at `evalExpr` -/
#guard_msgs (error, substring := true) in
#check property% (spec written) against source tracking []

/-- error: failed to synthesize -/
#guard_msgs (error, substring := true) in
#check (correlated_response% (id "test.correlated.response") at source
  whenever (.selectedActionIs request) eventually (.outcomeIs response)
  within (-1) correlated [id "test.run"] by (id "test.operation") closing .«partial»)

/-- error: Type mismatch -/
#guard_msgs (error, substring := true) in
#check (correlated_response% (id "test.correlated.response") at source
  whenever (step "a" request) eventually (.outcomeIs response)
  within 1 correlated [id "test.run"] by (id "test.operation") closing .«partial»)

/-- error: Unknown constant -/
#guard_msgs (error, substring := true) in
#check (correlated_response% (id "test.correlated.response") at source
  whenever (.selectedActionIs request) eventually (.outcomeIs response)
  within 1 correlated [id "test.run"] by (id "test.operation") closing .terminalModel)

/-- error: Type mismatch -/
#guard_msgs (error, substring := true) in
example (raw : Case.Projection.Event) : PropertyCorrelatedClause :=
  correlated_response% (id "test.correlated.response") at source
    whenever raw eventually (.outcomeIs response)
    within 1 correlated [id "test.run"] by (id "test.operation") closing .«partial»

/-- error: Type mismatch -/
#guard_msgs (error, substring := true) in
example (effect : temporal.server.api.testpilot.v1.Instruction) : PropertyCorrelatedClause :=
  correlated_response% (id "test.correlated.response") at source
    whenever effect eventually (.outcomeIs response)
    within 1 correlated [id "test.run"] by (id "test.operation") closing .«partial»

end Umpire.Property.CorrelatedTests
