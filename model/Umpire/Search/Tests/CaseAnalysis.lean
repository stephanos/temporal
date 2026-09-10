import Umpire.Search.Tests.Fixtures

/-! Bounded guarded-case analysis over a generic finite Planning fixture. -/

namespace Umpire.SearchTests.CaseAnalysis

open Umpire
open Umpire.SearchTests

private def analysisCapability : DefinitionId := id "planner.capability.analysis"
private def absentCapability : DefinitionId := id "planner.capability.absent-input"
private def absentAction : DefinitionId := id "planner.action.absent"

private def analysisMeaning
    (definitionId : DefinitionId)
    (kind : DefinitionKind) : Meaning := {
  definitionId
  kind
  behaviorVersion := definitionId.value ++ "/analysis-v1"
}

private def analysisContext : PropertyCheckContext := {
  definitions := [
    metadata analysisCapability .capability "planner-analysis/v1",
    metadata absentCapability .capability "planner-absent-input/v1",
    metadata phase .state "planner-phase/v1",
    metadata request .action "planner-request/v1",
    metadata absentAction .action "planner-absent/v1",
    metadata accepted .outcome "planner-accepted/v1",
    metadata observed .fact "planner-observed/v1"
  ]
  providers := [{
    id := analysisCapability
    version := 1
    behaviorVersion := "planner-analysis/v1"
  }, {
    id := absentCapability
    version := 1
    behaviorVersion := "planner-absent-input/v1"
  }]
  meanings := [
    (analysisCapability, analysisMeaning phase .state),
    (analysisCapability, analysisMeaning request .action),
    (analysisCapability, analysisMeaning absentAction .action),
    (analysisCapability, analysisMeaning accepted .outcome),
    (analysisCapability, analysisMeaning observed .fact),
    (absentCapability, analysisMeaning phase .state),
    (absentCapability, analysisMeaning absentAction .action),
    (absentCapability, analysisMeaning accepted .outcome),
    (absentCapability, analysisMeaning observed .fact)
  ]
}

private def atom
    (field : PropertyPredicateField)
    (reference : DefinitionId)
    (literal : PropertyLiteral) : PropertyPredicate :=
  .atom { field, reference, constraint := .equals literal }

private def requestGuard : PropertyPredicate :=
  atom .selectedAction request (.text requestValue.value)

private def initialGuard : PropertyPredicate :=
  atom .priorState phase (.text initial.value)

private def completedGuard : PropertyPredicate :=
  atom .priorState phase (.text completed.value)

private def absentActionGuard : PropertyPredicate :=
  atom .selectedAction absentAction (.text "absent")

private def stateExpectation (clauseId : String) : PropertySameStepClause := {
  id := id clauseId
  source
  expectation := atom .resultingState phase (.text completed.value)
}

private def temporalClause : PropertyTemporalClause :=
  .eventuallyWithin (id "planner.property.analysis.case.normal.temporal") source
    (PropertyPattern.exact .selectedAction request requestValue.value)
    (PropertyPattern.exact .observation observed observedValue.value)
    { value := 0, unit := .semanticTransitions }

private def normalCase : PropertyBranch := {
  id := id "planner.property.analysis.case.normal"
  source
  guard := initialGuard
  clauses := [stateExpectation "planner.property.analysis.case.normal.state"]
  temporalClauses := [temporalClause]
}

private def overlappingCase : PropertyBranch := {
  id := id "planner.property.analysis.case.overlap"
  source
  guard := requestGuard
  clauses := [stateExpectation "planner.property.analysis.case.overlap.state"]
}

private def excludedCase : PropertyBranch := {
  normalCase with
  id := id "planner.property.analysis.case.excluded"
  exception := some {
    id := id "planner.property.analysis.case.excluded.exception"
    source
    condition := initialGuard
  }
}

private def absentInputCase : PropertyBranch := {
  id := id "planner.property.analysis.case.absent-input"
  source
  guard := initialGuard
  clauses := [stateExpectation "planner.property.analysis.case.absent-input.state"]
}

private def group
    (cases : List PropertyBranch)
    (complete : Bool := true)
    (exclusive : Bool := true)
    (guard : PropertyPredicate := requestGuard)
    (exception : Option PropertyUnless := none) : PropertyBranches := {
  id := id "planner.property.analysis.group"
  source
  guard
  exception
  cases
  complete
  exclusive
}

private def authoredProperty (selectedGroup : PropertyBranches) : Property := {
  id := id "planner.property.analysis"
  source
  version := 2
  requires := [analysisCapability]
  clauses := [.branches selectedGroup]
}

private def Property.checked? (selectedGroup : PropertyBranches) : Option CheckedProperty :=
  (Property.check analysisContext ((authoredProperty selectedGroup))).toOption

private def analysisTarget : QueryModel (fun _ => True) := target 0

private def analysisQuery
    (property : CheckedProperty)
    (budget : Nat := 8) : CheckedQuery (fun _ => True) :=
  let form := QueryForm.select [property]
  {
    checkedQuery 0 form .exhaustive budget with
    form
    quantifier := form.quantifier
    claim := form.claim
    target := analysisTarget
    completeness := (CheckedQueryModel.ofTarget analysisTarget).completeness
  }

private def analyzed?
    (selectedGroup : PropertyBranches)
    (budget : Nat := 8) : Option BranchAnalysisResult := do
  let property ← Property.checked? selectedGroup
  let query := analysisQuery property budget
  let kernel ← (SearchView.ofCheckedQuery query.target.id query).toOption
  pure (analyzeBranches query kernel)

private def analyzedAbsentInput? : Option BranchAnalysisResult := do
  let declaration := {
    authoredProperty (group [absentInputCase] (guard := absentActionGuard)) with
    requires := [absentCapability]
  }
  let property ← (Property.check analysisContext (declaration)).toOption
  let query := analysisQuery property
  let kernel ← (SearchView.ofCheckedQuery query.target.id query).toOption
  pure (analyzeBranches query kernel)

private def findingKinds? (selectedGroup : PropertyBranches) : Option (List BranchFindingKind) :=
  (analyzed? selectedGroup).map fun result => result.findings.map BranchFinding.kind

/-! Complete/exclusive failures are source-linked obligations; temporal clauses remain in scope. -/
#guard ((analyzed? (group [normalCase, overlappingCase])).map fun result =>
    (result.scope.targetId,
      result.scope.behaviorId,
      result.scope.properties.map AnalyzedProperty.id,
      result.status,
      result.findings.map BranchFinding.kind,
      result.observations.head?.map fun observation =>
        (observation.parentId,
          observation.cases.map CaseApplicability.caseId,
          observation.cases.head?.map fun item => item.clauses.map PropertyClauseIdentity.clauseId))) ==
  some (targetId,
    behavior.id,
    [id "planner.property.analysis"],
    BranchStatus.exhaustive,
    [BranchFindingKind.overlappingExclusiveCases],
    some (id "planner.property.analysis.group",
      [id "planner.property.analysis.case.normal", id "planner.property.analysis.case.overlap"],
      some [
        id "planner.property.analysis.case.normal.state",
        id "planner.property.analysis.case.normal.temporal"
      ]))

/-! Compatible overlap is observed without becoming an exclusivity failure. -/
#guard ((analyzed? (group [normalCase, overlappingCase] (exclusive := false))).map fun result =>
    (result.findings.isEmpty,
      result.observations.head?.map fun observation =>
        observation.cases.filter CaseApplicability.applies |>.length)) == some (true, some 2)

/-! Reachable uncovered groups, exclusions, and missing replacements remain separate findings. -/
#guard findingKinds? (group [{ normalCase with guard := completedGuard }]) ==
  some [.uncoveredCompleteGroup]

#guard ((analyzed? (group [{ normalCase with guard := completedGuard }])).bind fun result =>
    result.findings.head? |>.map fun finding =>
      (finding.caseIds,
        finding.caseSources,
        finding.clauses.map PropertyClauseIdentity.clauseId,
        finding.effectiveGuards.map CheckedPropertyPredicate.expression)) == some (
  [id "planner.property.analysis.case.normal"],
  [source],
  [id "planner.property.analysis.case.normal.state",
    id "planner.property.analysis.case.normal.temporal"],
  [requestGuard, completedGuard])

#guard findingKinds? (group [excludedCase]) == some [
  .caseExcluded,
  .missingReplacement,
  .uncoveredCompleteGroup
]

#guard findingKinds? (group [excludedCase] (complete := false)) == some [
  .caseExcluded,
  .missingReplacement
]

#guard findingKinds? (group [normalCase] (exception := some {
    id := id "planner.property.analysis.group.exception"
    source
    condition := initialGuard
  })) == some [.parentExcluded]

/-! An unreachable parent trigger is unexercised under exhaustive evidence, not covered. -/
#guard ((analyzed? (group [normalCase] (guard := completedGuard))).map fun result =>
    (result.status,
      result.findings.isEmpty,
      result.requirements.head?.map BranchRequirement.parentExercised)) ==
  some (BranchStatus.exhaustive, true, some false)

/-! Exhausted analysis retains partial scope and witnesses but cannot claim exhaustive absence. -/
#guard ((analyzed? (group [normalCase]) 1).map fun result =>
    (result.status,
      result.scope.limits.search,
      result.metadata.completeness.established,
      result.observations.isEmpty)) ==
  some (BranchStatus.limitReached,
    ({ value := 1, unit := .candidateEvaluations } : Limit), false, true)

/-! A checked predicate input that the selected Target cannot supply rejects the analysis. -/
#guard (analyzedAbsentInput?.map fun result =>
    (result.status.name,
      result.metadata.completeness.established,
      result.findings.isEmpty,
      result.observations.isEmpty)) == some ("invalid", false, true, true)

/-! Case source order cannot choose a winner or change canonical analysis results. -/
#guard (analyzed? (group [normalCase, overlappingCase])).map BranchAnalysisResult.canonicalView ==
  (analyzed? (group [overlappingCase, normalCase])).map BranchAnalysisResult.canonicalView

end Umpire.SearchTests.CaseAnalysis
