import Temporal.Feature.Nexus2.Authoring
import Umpire.Search
import Umpire.Search.Branches

/-! Executable constructor admission, equivalence, identity, diagnostics, and trust fixtures. -/

namespace Temporal.Feature.Nexus2.AuthoringTests

open Umpire
open Temporal.Feature.Nexus2

private def frontendBaselineContext : PropertyCheckContext := {
  definitions := Lifecycle.definitions
  providers := [{
    id := Lifecycle.lifecycleProvider.contract.id
    version := Lifecycle.lifecycleProvider.contract.version
    behaviorVersion := Lifecycle.lifecycleProvider.contract.behaviorVersion
  }]
  meanings := Lifecycle.lifecycleProvider.meanings.map fun meaning =>
    (Lifecycle.lifecycleProvider.contract.id, meaning)
}

private def frontendBaselineModel : Cancellation.ModelVocabulary :=
  Cancellation.modelVocabulary.toOption.get (by decide)

private def frontendStartSpec : Property :=
  Authoring.Baseline.authoredProperty "start" frontendBaselineModel.startAction
    frontendBaselineModel.startedState frontendBaselineModel.startedOutcome
    frontendBaselineModel.startedFact

def frontendStartProperty : Except PropertyError CheckedProperty :=
  property% frontendStartSpec against frontendBaselineContext
    tracking [parentAnchor frontendStartSpec.id]

private def frontendStartEquivalence : Bool :=
  match frontendStartProperty, Authoring.Baseline.checkBaseline with
  | .ok frontend, .ok checked =>
      frontend.id == checked.start.property.id &&
        frontend.behaviorFingerprint == checked.start.property.behaviorFingerprint
  | _, _ => false

#guard frontendStartEquivalence

private def frontendRaceContext : PropertyCheckContext := {
  definitions := Race.definitions
  providers := [{
    id := Race.provider.contract.id
    version := Race.provider.contract.version
    behaviorVersion := Race.provider.contract.behaviorVersion
  }]
  meanings := Race.provider.meanings.map fun meaning => (Race.provider.contract.id, meaning)
}

private def frontendRaceModel : Race.ModelVocabulary :=
  Race.modelVocabulary.toOption.get (by decide)

def frontendGuardedProperty : Except PropertyError CheckedProperty :=
  property% Authoring.GuardedRace.authoredProperty frontendRaceModel against frontendRaceContext
    tracking [
      parentAnchor (Authoring.GuardedRace.family.id "property" "cases"),
      clauseAnchor (Authoring.GuardedRace.family.id "case-group" "lifecycle"),
      caseAnchor (Authoring.GuardedRace.family.id "case" "request"),
      clauseAnchor (Authoring.GuardedRace.family.id "case" "request.state"),
      clauseAnchor (Authoring.GuardedRace.family.id "case" "request.terminal"),
      caseAnchor (Authoring.GuardedRace.family.id "case" "resolve"),
      clauseAnchor (Authoring.GuardedRace.family.id "case" "resolve.terminal")
    ]

private def frontendGuardedEquivalence : Bool :=
  match frontendGuardedProperty,
      (Authoring.GuardedRace.authoredProperty frontendRaceModel).check frontendRaceContext with
  | .ok frontend, .ok constructor =>
      frontend.id == constructor.id && frontend.clauses == constructor.clauses &&
        frontend.behaviorFingerprint == constructor.behaviorFingerprint
  | _, _ => false

#guard frontendGuardedEquivalence

private def baselineEquivalence : Bool :=
  match Authoring.Baseline.checkBaseline, Cancellation.checkBaseline with
  | .ok authored, .ok established =>
      authored.target.id == established.target.id &&
      authored.target.behaviorTable == established.target.behaviorTable &&
      authored.target.providers.map (fun provider => provider.id) ==
        established.target.providers.map (fun provider => provider.id) &&
      authored.start.property.id == established.start.property.id &&
      authored.start.property.behaviorFingerprint == established.start.property.behaviorFingerprint &&
      authored.start.behavior.id == established.start.behavior.id &&
      authored.start.behavior.behaviorFingerprint == established.start.behavior.behaviorFingerprint &&
      authored.start.query.id == established.start.query.id &&
      authored.start.query.behaviorFingerprint == established.start.query.behaviorFingerprint &&
      authored.cancel.property.behaviorFingerprint == established.cancel.property.behaviorFingerprint &&
      authored.cancel.behavior.id == established.cancel.behavior.id &&
      authored.cancel.behavior.behaviorFingerprint == established.cancel.behavior.behaviorFingerprint &&
      authored.cancel.query.id == established.cancel.query.id &&
      authored.cancel.query.behaviorFingerprint == established.cancel.query.behaviorFingerprint &&
      authored.success.query.behaviorFingerprint == established.success.query.behaviorFingerprint
  | _, _ => false

#guard baselineEquivalence

private def constructorRunNames : Option (String × String × String) := do
  let checked ← Authoring.Baseline.checkBaseline.toOption
  let startKernel ← (SearchView.ofCheckedQuery checked.target.id checked.start.query).toOption
  let cancelKernel ← (SearchView.ofCheckedQuery checked.target.id checked.cancel.query).toOption
  let successKernel ← (SearchView.ofCheckedQuery checked.target.id checked.success.query).toOption
  let startRun ← (search checked.start.query startKernel).toOption
  let cancelRun ← (search checked.cancel.query cancelKernel).toOption
  let successRun ← (search checked.success.query successKernel).toOption; pure (startRun.result.outcome.name, cancelRun.result.outcome.name, successRun.result.outcome.name)

#guard constructorRunNames == some ("found", "found", "found")

private def guardedAdmission : Option (CheckedProperty × CheckedScenario × CheckedQuery Race.LawStatement) := do
  let target ← Race.targetResult.toOption
  let model ← Race.modelVocabulary.toOption
  let property ← (Authoring.GuardedRace.authoredProperty model).check
    (PropertyCheckContext.ofTarget target) |>.toOption
  let behavior ← (Authoring.GuardedRace.authoredScenario model).check (.ofTarget target) |>.toOption
  let query ← Query.check (.ofTarget target) (Authoring.GuardedRace.authoredQuery property behavior) |>.toOption
  pure (property, behavior, query)

#guard guardedAdmission.isSome

private def guardedBehaviorEquivalence : Option Bool := do
  let target ← Race.targetResult.toOption
  let model ← Race.modelVocabulary.toOption
  let authored ← (Authoring.GuardedRace.authoredScenario model).check (.ofTarget target) |>.toOption
  let established ← Scenario.check (.ofTarget target) (Race.exactBehaviorDeclaration model) |>.toOption
  pure (authored.id == established.id &&
    authored.requiredOccurrences == established.requiredOccurrences &&
    authored.actionsExactly == established.actionsExactly &&
    authored.behaviorFingerprint == established.behaviorFingerprint)

#guard guardedBehaviorEquivalence == some true

private def raceAlternativesAndProvider : Option (Nat × List DefinitionId) := do
  let target ← Race.targetResult.toOption
  let alternatives := target.machine.steps
    (ModelValue.named Race.operationStateId "cancel-requested")
    (ModelValue.named Race.resolveActionId "resolve")
  pure (alternatives.length, target.providers.map (fun provider => provider.id))

#guard raceAlternativesAndProvider == some (2, [Race.providerId])

private def inspectedGuardedSurface : Option
    (DefinitionId × List DefinitionId × List DefinitionId × Query.Form × Limits × PlannerPolicy) := do
  let (property, behavior, query) ← guardedAdmission
  pure (property.id, property.guardedClauseIds,
    behavior.requiredOccurrences.map Scenario.Step.id, query.form, query.limits, query.policy)

#guard inspectedGuardedSurface.map (fun (propertyId, clauses, occurrences, form, limits, policy) =>
  propertyId == Authoring.GuardedRace.family.id "property" "cases" &&
    clauses == [Authoring.GuardedRace.family.id "case-group" "lifecycle"] &&
    occurrences == [
      Authoring.GuardedRace.family.id "occurrence" "request",
      Authoring.GuardedRace.family.id "occurrence" "resolution"
    ] &&
    (match form with | .pick [_] => true | _ => false) &&
    limits == Limits.bounded 2 2 32 && policy == PlannerPolicy.exhaustive) == some true

private def propertyErrorKind (spec : Property) : Option PropertyErrorKind := do
  let target ← Race.targetResult.toOption
  match spec.check (PropertyCheckContext.ofTarget target) with
  | .error error => some error.kind
  | .ok _ => none

private def malformedProperty : Property := {
  id := (DefinitionFamily.mk (DefinitionId.of "temporal.nexus2.")).id "property" "malformed"
  source := Race.source
  requires := []
  clauses := []
}

#guard propertyErrorKind malformedProperty == some .invalidDefinitionId

private def duplicateClauseProperty (model : Race.ModelVocabulary) : Property :=
  let clause := PropertyClause.transitionContract
    (Authoring.GuardedRace.family.id "property" "duplicate.clause")
    (.selectedAction model.resolveAction) (.outcome model.canceledOutcome)
  {
    id := (Authoring.GuardedRace.family).id "property" "duplicate"
    source := Race.source
    requires := [Race.capabilityId]
    clauses := [clause, clause]
  }

#guard Race.modelVocabulary.toOption.bind (fun model =>
  propertyErrorKind (duplicateClauseProperty model)) == some .duplicateDefinitionId

private def unknownReferenceProperty (model : Race.ModelVocabulary) : Property := {
  id := (Authoring.GuardedRace.family).id "property" "unknown-reference"
  source := Race.source
  requires := [Race.capabilityId]
  clauses := [.transitionContract
    (Authoring.GuardedRace.family.id "property" "unknown-reference.clause")
    (.selectedAction { model.resolveAction with
      definitionId := DefinitionId.of "temporal.nexus2.unknown.action" })
    (.outcome model.canceledOutcome)]
}

#guard Race.modelVocabulary.toOption.bind (fun model =>
  propertyErrorKind (unknownReferenceProperty model)) == some .unknownReference

private def unknownReferenceDiagnostic : Option (PropertyErrorKind × List DefinitionId) := do
  let target ← Race.targetResult.toOption
  let model ← Race.modelVocabulary.toOption
  match (unknownReferenceProperty model).check (PropertyCheckContext.ofTarget target) with
  | .error error => some (error.kind, error.relatedDefinitionIds)
  | .ok _ => none

#guard unknownReferenceDiagnostic == some (
  .unknownReference, [DefinitionId.of "temporal.nexus2.unknown.action"])

private def missingCapabilityProperty (model : Race.ModelVocabulary) : Property := {
  id := (Authoring.GuardedRace.family).id "property" "missing-capability"
  source := Race.source
  requires := []
  clauses := [.transitionContract
    (Authoring.GuardedRace.family.id "property" "missing-capability.clause")
    (.selectedAction model.resolveAction) (.outcome model.canceledOutcome)]
}

#guard Race.modelVocabulary.toOption.bind (fun model =>
  propertyErrorKind (missingCapabilityProperty model)) == some .undeclaredReference

private def unsupportedGuardProperty (model : Race.ModelVocabulary) : Property := {
  id := (Authoring.GuardedRace.family).id "property" "unsupported-guard"
  source := Race.source
  version := 2
  requires := [Race.capabilityId]
  clauses := [.branches {
    id := Authoring.GuardedRace.family.id "case-group" "unsupported-guard"
    source := Race.source
    guard := .resultingStateIs model.cancelRequestedState
    cases := [Authoring.GuardedRace.requestCase model]
  }]
}

#guard Race.modelVocabulary.toOption.bind (fun model =>
  propertyErrorKind (unsupportedGuardProperty model)) == some .invalidPredicateContext

private def emptyGroupProperty : Property := {
  id := (Authoring.GuardedRace.family).id "property" "empty-group"
  source := Race.source
  version := 2
  requires := [Race.capabilityId]
  clauses := [.branches {
    id := Authoring.GuardedRace.family.id "case-group" "empty"
    source := Race.source
    guard := .selectedActionIs frontendRaceModel.resolveAction
    cases := []
  }]
}

private def emptyBooleanProperty : Property := {
  id := (Authoring.GuardedRace.family).id "property" "empty-boolean"
  source := Race.source
  version := 2
  requires := [Race.capabilityId]
  clauses := [.branches {
    id := Authoring.GuardedRace.family.id "case-group" "empty-boolean"
    source := Race.source
    guard := .all []
    cases := [Authoring.GuardedRace.resolutionCase frontendRaceModel]
  }]
}

private def invalidUnitProperty : Property := {
  id := (Authoring.GuardedRace.family).id "property" "invalid-unit"
  source := Race.source
  requires := [Race.capabilityId]
  clauses := [.eventuallyWithin
    (Authoring.GuardedRace.family.id "property" "invalid-unit.clause")
    (.selectedAction frontendRaceModel.resolveAction) (.fact frontendRaceModel.terminalFact)
    { value := 1, unit := .search }]
}

private def wrongKindProperty : Property := {
  id := (Authoring.GuardedRace.family).id "property" "wrong-kind"
  source := Race.source
  requires := [Race.capabilityId]
  clauses := [.transitionContract
    (Authoring.GuardedRace.family.id "property" "wrong-kind.clause")
    (.selectedAction frontendRaceModel.startedState) (.outcome frontendRaceModel.canceledOutcome)]
}

private def invalidExceptionProperty : Property :=
  let request := Authoring.GuardedRace.requestCase frontendRaceModel
  { Authoring.GuardedRace.authoredProperty frontendRaceModel with clauses := [.branches {
      id := Authoring.GuardedRace.family.id "case-group" "invalid-exception"
      source := Race.source
      guard := .selectedActionIs frontendRaceModel.requestCancelAction
      cases := [{ request with exception := some {
        id := Authoring.GuardedRace.family.id "exception" "invalid-result-guard"
        source := Race.source
        condition := .resultingStateIs frontendRaceModel.cancelRequestedState
      }}]
    }] }

private def missingProviderContext : PropertyCheckContext :=
  { frontendRaceContext with providers := [] }

/--
error: property authoring failed: {"error":{"kind":"invalid-definition-id","definitionId":"temporal.nexus2..property.malformed","sourcePath":"Temporal/Feature/Nexus2/Race.lean","offendingValue":"temporal.nexus2..property.malformed","relatedDefinitionIds":["temporal.nexus2..property.malformed"]},"role":"parent","anchor":{"sourcePath":"Temporal/Feature/Nexus2/AuthoringTests.lean","line":308,"column":15,"endLine":308,"endColumn":35}}
-/
#guard_msgs (error) in
#check property% malformedProperty against frontendRaceContext tracking [
  parentAnchor malformedProperty.id]

/--
error: property authoring failed: {"error":{"kind":"duplicate-definition-id","definitionId":"temporal.nexus2.cancellation-race.property.duplicate","sourcePath":"Temporal/Feature/Nexus2/Race.lean","offendingValue":"temporal.nexus2.cancellation-race.property.duplicate.clause","relatedDefinitionIds":["temporal.nexus2.cancellation-race.property.duplicate.clause"]},"role":"clause","anchor":{"sourcePath":"Temporal/Feature/Nexus2/AuthoringTests.lean","line":317,"column":15,"endLine":317,"endColumn":78}}
-/
#guard_msgs (error) in
#check property% duplicateClauseProperty frontendRaceModel against frontendRaceContext tracking [
  parentAnchor (duplicateClauseProperty frontendRaceModel).id,
  clauseAnchor (Authoring.GuardedRace.family.id "property" "duplicate.clause"),
  clauseAnchor (Authoring.GuardedRace.family.id "property" "duplicate.clause")]

/--
error: property authoring failed: {"error":{"kind":"unknown-reference","definitionId":"temporal.nexus2.cancellation-race.property.unknown-reference","sourcePath":"Temporal/Feature/Nexus2/Race.lean","offendingValue":"temporal.nexus2.unknown.action","relatedDefinitionIds":["temporal.nexus2.unknown.action"]},"role":"clause","anchor":{"sourcePath":"Temporal/Feature/Nexus2/AuthoringTests.lean","line":325,"column":15,"endLine":325,"endColumn":65}}
-/
#guard_msgs (error) in
#check property% unknownReferenceProperty frontendRaceModel against frontendRaceContext tracking [
  parentAnchor (unknownReferenceProperty frontendRaceModel).id,
  clauseAnchor (DefinitionId.of "temporal.nexus2.unknown.action")]

/--
error: property authoring failed: {"error":{"kind":"missing-capability","definitionId":"temporal.nexus2.cancellation-race.property.cases","sourcePath":"Temporal/Feature/Nexus2/Race.lean","offendingValue":"temporal.nexus2.cancellation-race.capability","relatedDefinitionIds":["temporal.nexus2.cancellation-race.capability"]},"role":"parent","anchor":{"sourcePath":"Temporal/Feature/Nexus2/AuthoringTests.lean","line":332,"column":15,"endLine":332,"endColumn":32}}
-/
#guard_msgs (error) in
#check property% Authoring.GuardedRace.authoredProperty frontendRaceModel against missingProviderContext tracking [
  parentAnchor Race.capabilityId]

/--
error: property authoring failed: {"error":{"kind":"invalid-predicate-context","definitionId":"temporal.nexus2.cancellation-race.property.unsupported-guard","sourcePath":"Temporal/Feature/Nexus2/Race.lean","source":{"path":"Temporal/Feature/Nexus2/Race.lean","line":1,"column":1,"provenance":"lean-model"},"offendingValue":"before: resulting-state","relatedDefinitionIds":["temporal.nexus2.cancellation-race.state.operation"]},"role":"case","anchor":{"sourcePath":"Temporal/Feature/Nexus2/AuthoringTests.lean","line":340,"column":13,"endLine":340,"endColumn":64}}
-/
#guard_msgs (error) in
#check property% unsupportedGuardProperty frontendRaceModel against frontendRaceContext tracking [
  parentAnchor (unsupportedGuardProperty frontendRaceModel).id,
  caseAnchor frontendRaceModel.cancelRequestedState.definitionId]

/--
error: property authoring failed: {"error":{"kind":"empty-case-group","definitionId":"temporal.nexus2.cancellation-race.property.empty-group","sourcePath":"Temporal/Feature/Nexus2/Race.lean","source":{"path":"Temporal/Feature/Nexus2/Race.lean","line":1,"column":1,"provenance":"lean-model"},"offendingValue":"temporal.nexus2.cancellation-race.case-group.empty","relatedDefinitionIds":["temporal.nexus2.cancellation-race.case-group.empty"]},"role":"clause","anchor":{"sourcePath":"Temporal/Feature/Nexus2/AuthoringTests.lean","line":348,"column":15,"endLine":348,"endColumn":69}}
-/
#guard_msgs (error) in
#check property% emptyGroupProperty against frontendRaceContext tracking [
  parentAnchor emptyGroupProperty.id,
  clauseAnchor (Authoring.GuardedRace.family.id "case-group" "empty")]

/--
error: property authoring failed: {"error":{"kind":"empty-boolean-group","definitionId":"temporal.nexus2.cancellation-race.property.empty-boolean","sourcePath":"Temporal/Feature/Nexus2/Race.lean","source":{"path":"Temporal/Feature/Nexus2/Race.lean","line":1,"column":1,"provenance":"lean-model"},"offendingValue":"all","relatedDefinitionIds":[]},"role":"parent","anchor":{"sourcePath":"Temporal/Feature/Nexus2/AuthoringTests.lean","line":355,"column":15,"endLine":355,"endColumn":38}}
-/
#guard_msgs (error) in
#check property% emptyBooleanProperty against frontendRaceContext tracking [
  parentAnchor emptyBooleanProperty.id,
  clauseAnchor (Authoring.GuardedRace.family.id "case-group" "empty-boolean")]

/--
error: property authoring failed: {"error":{"kind":"unit-mismatch","definitionId":"temporal.nexus2.cancellation-race.property.invalid-unit","sourcePath":"Temporal/Feature/Nexus2/Race.lean","offendingValue":"search is not a Property position unit","relatedDefinitionIds":["temporal.nexus2.cancellation-race.action.resolve","temporal.nexus2.cancellation-race.fact.terminal"]},"role":"parent","anchor":{"sourcePath":"Temporal/Feature/Nexus2/AuthoringTests.lean","line":363,"column":15,"endLine":363,"endColumn":37}}
-/
#guard_msgs (error) in
#check property% invalidUnitProperty against frontendRaceContext tracking [
  parentAnchor invalidUnitProperty.id,
  clauseAnchor (Authoring.GuardedRace.family.id "property" "invalid-unit.clause")]

/--
error: property authoring failed: {"error":{"kind":"wrong-reference-kind","definitionId":"temporal.nexus2.cancellation-race.property.wrong-kind","sourcePath":"Temporal/Feature/Nexus2/Race.lean","offendingValue":"temporal.nexus2.cancellation-race.state.operation: expected action, found state","relatedDefinitionIds":["temporal.nexus2.cancellation-race.state.operation"]},"role":"clause","anchor":{"sourcePath":"Temporal/Feature/Nexus2/AuthoringTests.lean","line":372,"column":15,"endLine":372,"endColumn":58}}
-/
#guard_msgs (error) in
#check property% wrongKindProperty against frontendRaceContext tracking [
  parentAnchor wrongKindProperty.id,
  clauseAnchor frontendRaceModel.startedState.definitionId]

/--
error: property authoring failed: {"error":{"kind":"invalid-predicate-context","definitionId":"temporal.nexus2.cancellation-race.property.cases","sourcePath":"Temporal/Feature/Nexus2/Race.lean","source":{"path":"Temporal/Feature/Nexus2/Race.lean","line":1,"column":1,"provenance":"lean-model"},"offendingValue":"before: resulting-state","relatedDefinitionIds":["temporal.nexus2.cancellation-race.state.operation"]},"role":"exception","anchor":{"sourcePath":"Temporal/Feature/Nexus2/AuthoringTests.lean","line":381,"column":18,"endLine":381,"endColumn":69}}
-/
#guard_msgs (error) in
#check property% invalidExceptionProperty against frontendRaceContext tracking [
  parentAnchor invalidExceptionProperty.id,
  caseAnchor (Authoring.GuardedRace.family.id "case" "request"),
  exceptionAnchor frontendRaceModel.cancelRequestedState.definitionId]

/--
error: expected parentAnchor, caseAnchor, exceptionAnchor, or clauseAnchor
-/
#guard_msgs (error) in
#check property% malformedProperty against frontendRaceContext tracking [
  unsupportedOperator malformedProperty.id]

private def emptyCaseProperty (model : Race.ModelVocabulary) : Property :=
  let request := Authoring.GuardedRace.requestCase model
  { Authoring.GuardedRace.authoredProperty model with clauses := [.branches {
      id := Authoring.GuardedRace.family.id "case-group" "empty-case"
      source := Race.source
      guard := .selectedActionIs model.requestCancelAction
      cases := [{ request with
        id := Authoring.GuardedRace.family.id "case" "empty"
        clauses := []
        temporalClauses := []
      }]
    }] }

/--
error: property authoring failed: {"error":{"kind":"empty-case","definitionId":"temporal.nexus2.cancellation-race.property.cases","sourcePath":"Temporal/Feature/Nexus2/Race.lean","source":{"path":"Temporal/Feature/Nexus2/Race.lean","line":1,"column":1,"provenance":"lean-model"},"offendingValue":"temporal.nexus2.cancellation-race.case.empty","relatedDefinitionIds":["temporal.nexus2.cancellation-race.case-group.empty-case","temporal.nexus2.cancellation-race.case.empty"]},"role":"case","anchor":{"sourcePath":"Temporal/Feature/Nexus2/AuthoringTests.lean","line":410,"column":13,"endLine":410,"endColumn":61}}
-/
#guard_msgs (error) in
#check property% emptyCaseProperty frontendRaceModel against frontendRaceContext tracking [
  parentAnchor (emptyCaseProperty frontendRaceModel).id,
  clauseAnchor (Authoring.GuardedRace.family.id "case-group" "empty-case"),
  caseAnchor (Authoring.GuardedRace.family.id "case" "empty")]

private def scenarioErrorKind (spec : Scenario) : Option ScenarioErrorKind := do
  let target ← Race.targetResult.toOption
  match spec.check (.ofTarget target) with
  | .error error => some error.kind
  | .ok _ => none

private def duplicateOccurrenceBehavior (model : Race.ModelVocabulary) : Scenario :=
  { (Authoring.GuardedRace.authoredScenario model).withSteps [
      { id := Authoring.GuardedRace.family.id "occurrence" "same",
        action := model.requestCancelAction.definitionId },
      { id := Authoring.GuardedRace.family.id "occurrence" "same",
        action := model.resolveAction.definitionId }
    ] with id := Authoring.GuardedRace.family.id "behavior" "duplicate-occurrence" }

#guard Race.modelVocabulary.toOption.bind (fun model =>
  scenarioErrorKind (duplicateOccurrenceBehavior model)) == some .duplicateDefinitionId

private def contradictoryBehavior (model : Race.ModelVocabulary) : Scenario := {
  Authoring.GuardedRace.authoredScenario model with
  id := Authoring.GuardedRace.family.id "behavior" "contradictory"
  setup := (Authoring.GuardedRace.authoredScenario model).setup ++ [{
    id := Authoring.GuardedRace.family.id "setup" "not-started"
    relation := .different
    left := .role Race.operationRoleId
    right := .value model.startedState
  }]
}

#guard Race.modelVocabulary.toOption.bind (fun model =>
  scenarioErrorKind (contradictoryBehavior model)) == none

#guard Race.modelVocabulary.toOption.bind (fun model => do
  let target ← Race.targetResult.toOption
  let checked ← (contradictoryBehavior model).check (.ofTarget target) |>.toOption
  pure checked.isUnsatisfiable) == some true

private def queryErrorKind
    (target : QueryModel Race.LawStatement)
    (declaration : Query) : Option QueryErrorKind :=
  match Query.check (.ofTarget target) declaration with
  | .error error => some error.kind
  | .ok _ => none

private def wrongTargetQuery : Option QueryErrorKind := do
  let target ← Race.targetResult.toOption
  let (property, behavior, _) ← guardedAdmission
  let spec := { Authoring.GuardedRace.authoredQuery property behavior with
    target := DefinitionId.of "temporal.nexus2.other.target" }
  queryErrorKind target spec

#guard wrongTargetQuery == some .targetMismatch

private def wrongTargetDiagnostic : Option (QueryErrorKind × List DefinitionId) := do
  let target ← Race.targetResult.toOption
  let (property, behavior, _) ← guardedAdmission
  let other := DefinitionId.of "temporal.nexus2.other.target"
  let spec := { Authoring.GuardedRace.authoredQuery property behavior with target := other }
  match Query.check (.ofTarget target) spec with
  | .error error => some (error.kind, error.relatedDefinitionIds)
  | .ok _ => none

#guard wrongTargetDiagnostic == some (
  .targetMismatch, [DefinitionId.of "temporal.nexus2.cancellation-race.target",
    DefinitionId.of "temporal.nexus2.other.target"])

private def wrongUnitQuery : Option QueryErrorKind := do
  let target ← Race.targetResult.toOption
  let (property, behavior, _) ← guardedAdmission
  let declaration := { Authoring.GuardedRace.authoredQuery property behavior with
    limits := {
      steps := { value := 2, unit := .actions }
      actions := { value := 2, unit := .actions }
      search := { value := 32, unit := .search }
    }
  }
  match Query.check (.ofTarget target) declaration with
  | .error error => pure error.kind
  | .ok _ => none

#guard wrongUnitQuery == some .unitMismatch

/--
error: Fields missing: `unit`

Hint: Add missing fields:
  
  ̲ ̲ ̲ ̲ ̲ ̲ ̲ ̲ ̲ ̲ ̲ ̲ ̲ ̲ ̲ ̲ ̲ ̲ ̲ ̲ ̲ ̲ ̲ ̲ ̲ ̲ ̲ ̲ ̲ ̲u̲n̲i̲t̲ ̲:̲=̲ ̲_̲
-/
#guard_msgs (error) in
def missingUnit : Limit := { value := 1 }

private def missingReplacementAnalysis : Option
    (BranchStatus × DefinitionId × List DefinitionId × List DefinitionId ×
      Option ModelValue × Option ModelValue) := do
  let target ← Race.targetResult.toOption
  let model ← Race.modelVocabulary.toOption
  let spec := Authoring.GuardedRace.withoutReplacement model
  let property ← spec.check (PropertyCheckContext.ofTarget target) |>.toOption
  let behavior ← (Authoring.GuardedRace.authoredScenario model).check (.ofTarget target) |>.toOption
  let query ← Query.check (.ofTarget target) (Authoring.GuardedRace.authoredQuery property behavior) |>.toOption
  let kernel ← SearchView.ofCheckedQuery query.target.id query |>.toOption
  let result := analyzeBranches query kernel
  let finding ← result.findings.find? fun finding => finding.kind == .missingReplacement
  pure (result.status, finding.parentId, finding.caseIds,
    finding.exceptions.map CheckedPropertyUnless.id,
    finding.priorState, finding.selectedAction)

#guard Race.modelVocabulary.toOption.map (fun model =>
  missingReplacementAnalysis == some (
    .exhaustive,
    Authoring.GuardedRace.family.id "case-group" "missing-replacement",
    [Authoring.GuardedRace.family.id "case" "request"],
    [Authoring.GuardedRace.family.id "exception" "request.already-cancel-requested"],
    some model.startedState,
    some model.requestCancelAction)) == some true

private def sourceIndependentIdentity : Option Bool := do
  let checked ← Authoring.Baseline.checkBaseline.toOption
  let moved := { Authoring.Baseline.authoredProperty "start" checked.model.startAction
    checked.model.startedState checked.model.startedOutcome checked.model.startedFact with
    source := { path := "Moved/Without/Semantic/Change.lean", line := 900, column := 3 }
    documentation := "Renamed Lean declaration and edited prose only."
  }
  let moved ← moved.check (PropertyCheckContext.ofTarget checked.target) |>.toOption
  pure (moved.id == checked.start.property.id &&
    moved.behaviorFingerprint == checked.start.property.behaviorFingerprint)

#guard sourceIndependentIdentity == some true

private def changedBehaviorFingerprint : Option Bool := do
  let checked ← Authoring.Baseline.checkBaseline.toOption
  let changed := Authoring.Baseline.authoredProperty "start" checked.model.startAction
    checked.model.canceledState checked.model.startedOutcome checked.model.startedFact
  let changed ← changed.check (PropertyCheckContext.ofTarget checked.target) |>.toOption
  pure (changed.behaviorFingerprint != checked.start.property.behaviorFingerprint)

#guard changedBehaviorFingerprint == some true

private def allQueryForms : Option (List String) := do
  let target ← Race.targetResult.toOption
  let (property, behavior, _) ← guardedAdmission
  let base := Authoring.GuardedRace.authoredQuery property behavior
  [.verify property, .find property, .findViolation property, .pick [property]].mapM fun form => do
    let checked ← (Query.check (.ofTarget target) { base with form }).toOption
    pure checked.form.name

#guard allQueryForms == some ["verify", "find", "find-violation", "pick"]

private def frontendRaceBehaviorContext : ScenarioCheckContext := {
  definitions := Race.definitions
}

def frontendGuardedBehavior : Except ScenarioError CheckedScenario :=
  scenario% Authoring.GuardedRace.authoredScenario frontendRaceModel
    against frontendRaceBehaviorContext tracking [
      scenarioParent (Authoring.GuardedRace.authoredScenario frontendRaceModel).id,
      setupAnchor (Authoring.GuardedRace.family.id "setup" "started"),
      occurrenceAnchor (Authoring.GuardedRace.family.id "occurrence" "request"),
      occurrenceAnchor (Authoring.GuardedRace.family.id "occurrence" "resolution")]

private def frontendGuardedBehaviorEquivalence : Bool :=
  match frontendGuardedBehavior,
      (Authoring.GuardedRace.authoredScenario frontendRaceModel).check frontendRaceBehaviorContext with
  | .ok frontend, .ok constructor =>
      frontend.id == constructor.id &&
        frontend.canonicalMetadata == constructor.canonicalMetadata &&
        frontend.behaviorFingerprint == constructor.behaviorFingerprint
  | _, _ => false

#guard frontendGuardedBehaviorEquivalence

private def frontendRaceTarget : QueryModel Race.LawStatement :=
  Race.targetResult.toOption.get (by native_decide)

private def frontendCheckedProperty : CheckedProperty :=
  frontendGuardedProperty.toOption.get (by native_decide)

private def frontendCheckedScenario : CheckedScenario :=
  frontendGuardedBehavior.toOption.get (by native_decide)

private def frontendGuardedAuthoredQuery : Query :=
  Authoring.GuardedRace.authoredQuery frontendCheckedProperty frontendCheckedScenario

def frontendGuardedQuery : Except QueryError (CheckedQuery Race.LawStatement) :=
  query% frontendGuardedAuthoredQuery against frontendRaceTarget tracking [
    queryParent (Authoring.GuardedRace.family.id "query" "case-analysis"),
    targetAnchor Race.targetId,
    propertyAnchor (Authoring.GuardedRace.family.id "property" "cases"),
    scenarioAnchor (Authoring.GuardedRace.family.id "behavior" "request-then-resolve"),
    limitsAnchor (Authoring.GuardedRace.family.id "query" "case-analysis")]

private def frontendGuardedAdmission : Option
    (CheckedProperty × CheckedScenario × CheckedQuery Race.LawStatement) := do
  let query ← frontendGuardedQuery.toOption
  let property ← frontendGuardedProperty.toOption
  let behavior ← frontendGuardedBehavior.toOption
  if query.target.id != frontendRaceTarget.id then none else
  pure (property, behavior, query)

private def frontendGuardedQueryEquivalence : Option Bool := do
  let (_, _, frontend) ← frontendGuardedAdmission
  let (_, _, constructor) ← guardedAdmission
  pure (frontend.id == constructor.id &&
    frontend.canonicalMetadata == constructor.canonicalMetadata &&
    frontend.behaviorFingerprint == constructor.behaviorFingerprint &&
    frontend.form == constructor.form && frontend.limits == constructor.limits &&
    frontend.policy == constructor.policy &&
    frontend.modelProviders == constructor.modelProviders)

#guard frontendGuardedQueryEquivalence == some true

private def frontendGuardedQueryOutcome : Option (String × String) := do
  let (_, _, frontend) ← frontendGuardedAdmission
  let (_, _, constructor) ← guardedAdmission
  let frontendKernel ← SearchView.ofCheckedQuery frontend.target.id frontend |>.toOption
  let constructorKernel ← SearchView.ofCheckedQuery constructor.target.id constructor |>.toOption
  let frontendRun ← (search frontend frontendKernel).toOption
  let constructorRun ← (search constructor constructorKernel).toOption; pure (frontendRun.result.outcome.name, constructorRun.result.outcome.name)

#guard frontendGuardedQueryOutcome == some ("found", "found")

private def malformedBehavior : Scenario := {
  Authoring.GuardedRace.authoredScenario frontendRaceModel with
  id := (DefinitionFamily.mk (DefinitionId.of "temporal.nexus2.")).id "behavior" "malformed"
}

private def unknownActionId : DefinitionId :=
  DefinitionId.of "temporal.nexus2.unknown.action"

private def unknownActionBehavior : Scenario :=
  { (Authoring.GuardedRace.authoredScenario frontendRaceModel).withSteps
      [{ id := Authoring.GuardedRace.family.id "occurrence" "unknown",
         action := unknownActionId }] with
    id := Authoring.GuardedRace.family.id "behavior" "unknown-action" }

private def wrongKindBehavior : Scenario :=
  { (Authoring.GuardedRace.authoredScenario frontendRaceModel).withSteps
      [{ id := Authoring.GuardedRace.family.id "occurrence" "wrong-kind",
         action := frontendRaceModel.startedState.definitionId }] with
    id := Authoring.GuardedRace.family.id "behavior" "wrong-kind" }

/--
error: scenario authoring failed: {"error":{"kind":"invalid-definition-id","definitionId":"temporal.nexus2..behavior.malformed","sourcePath":"Temporal/Feature/Nexus2/Race.lean","offendingValue":"temporal.nexus2..behavior.malformed","relatedDefinitionIds":["temporal.nexus2..behavior.malformed"]},"role":"parent","anchor":{"sourcePath":"Temporal/Feature/Nexus2/AuthoringTests.lean","line":658,"column":17,"endLine":658,"endColumn":37}}
-/
#guard_msgs (error) in
#check scenario% malformedBehavior against frontendRaceBehaviorContext tracking [
  scenarioParent malformedBehavior.id]

/--
error: scenario authoring failed: {"error":{"kind":"duplicate-definition-id","definitionId":"temporal.nexus2.cancellation-race.behavior.duplicate-occurrence","sourcePath":"Temporal/Feature/Nexus2/Race.lean","offendingValue":"temporal.nexus2.cancellation-race.occurrence.same","relatedDefinitionIds":["temporal.nexus2.cancellation-race.occurrence.same"]},"role":"occurrence","anchor":{"sourcePath":"Temporal/Feature/Nexus2/AuthoringTests.lean","line":668,"column":19,"endLine":668,"endColumn":72}}
-/
#guard_msgs (error) in
#check scenario% duplicateOccurrenceBehavior frontendRaceModel
    against frontendRaceBehaviorContext tracking [
  scenarioParent (duplicateOccurrenceBehavior frontendRaceModel).id,
  occurrenceAnchor (Authoring.GuardedRace.family.id "occurrence" "same"),
  occurrenceAnchor (Authoring.GuardedRace.family.id "occurrence" "same")]

/--
error: scenario authoring failed: {"error":{"kind":"unknown-reference","definitionId":"temporal.nexus2.cancellation-race.behavior.unknown-action","sourcePath":"Temporal/Feature/Nexus2/Race.lean","offendingValue":"temporal.nexus2.unknown.action","relatedDefinitionIds":["temporal.nexus2.unknown.action"]},"role":"action","anchor":{"sourcePath":"Temporal/Feature/Nexus2/AuthoringTests.lean","line":676,"column":15,"endLine":676,"endColumn":30}}
-/
#guard_msgs (error) in
#check scenario% unknownActionBehavior against frontendRaceBehaviorContext tracking [
  scenarioParent unknownActionBehavior.id,
  actionAnchor unknownActionId]

/--
error: scenario authoring failed: {"error":{"kind":"wrong-reference-kind","definitionId":"temporal.nexus2.cancellation-race.behavior.wrong-kind","sourcePath":"Temporal/Feature/Nexus2/Race.lean","offendingValue":"temporal.nexus2.cancellation-race.state.operation: expected action, found state","relatedDefinitionIds":["temporal.nexus2.cancellation-race.state.operation"]},"role":"action","anchor":{"sourcePath":"Temporal/Feature/Nexus2/AuthoringTests.lean","line":684,"column":15,"endLine":684,"endColumn":58}}
-/
#guard_msgs (error) in
#check scenario% wrongKindBehavior against frontendRaceBehaviorContext tracking [
  scenarioParent wrongKindBehavior.id,
  actionAnchor frontendRaceModel.startedState.definitionId]

/--
error: expected scenarioParent, setupAnchor, occurrenceAnchor, or actionAnchor
-/
#guard_msgs (error) in
#check scenario% malformedBehavior against frontendRaceBehaviorContext tracking [
  unsupportedBehaviorRole malformedBehavior.id]

private def otherTargetId : DefinitionId :=
  DefinitionId.of "temporal.nexus2.other.target"

private def mismatchedTargetQuery : Query :=
  { frontendGuardedAuthoredQuery with target := otherTargetId }

private def emptyQuery : Query :=
  { frontendGuardedAuthoredQuery with form := .pick [] }

private def missingCapabilityQuery : Query :=
  { frontendGuardedAuthoredQuery with
    form := .pick [{ frontendCheckedProperty with requires := [unknownActionId] }] }

private def invalidLimitQuery : Query :=
  { frontendGuardedAuthoredQuery with limits := Limits.bounded 2 2 0 }

private def mismatchedUnitQuery : Query :=
  { frontendGuardedAuthoredQuery with limits := {
    steps := { value := 2, unit := .actions }
    actions := { value := 2, unit := .actions }
    search := { value := 32, unit := .search }
  } }

private def duplicatePropertyQuery : Query :=
  { frontendGuardedAuthoredQuery with
    form := .pick [frontendCheckedProperty, frontendCheckedProperty] }

/--
error: query authoring failed: {"error":{"kind":"target-mismatch","definitionId":"temporal.nexus2.cancellation-race.query.case-analysis","sourcePath":"Temporal/Feature/Nexus2/Race.lean","offendingValue":"temporal.nexus2.other.target != temporal.nexus2.cancellation-race.target","relatedDefinitionIds":["temporal.nexus2.cancellation-race.target","temporal.nexus2.other.target"]},"role":"target","anchor":{"sourcePath":"Temporal/Feature/Nexus2/AuthoringTests.lean","line":726,"column":15,"endLine":726,"endColumn":28}}
-/
#guard_msgs (error) in
#check query% mismatchedTargetQuery against frontendRaceTarget tracking [
  queryParent (Authoring.GuardedRace.family.id "query" "case-analysis"),
  targetAnchor otherTargetId]

/--
error: query authoring failed: {"error":{"kind":"missing-property","definitionId":"temporal.nexus2.cancellation-race.query.case-analysis","sourcePath":"Temporal/Feature/Nexus2/Race.lean","offendingValue":"properties","relatedDefinitionIds":[]},"role":"property","anchor":{"sourcePath":"Temporal/Feature/Nexus2/AuthoringTests.lean","line":734,"column":17,"endLine":734,"endColumn":69}}
-/
#guard_msgs (error) in
#check query% emptyQuery against frontendRaceTarget tracking [
  queryParent (Authoring.GuardedRace.family.id "query" "case-analysis"),
  propertyAnchor (Authoring.GuardedRace.family.id "property" "cases")]

/--
error: query authoring failed: {"error":{"kind":"missing-capability","definitionId":"temporal.nexus2.cancellation-race.query.case-analysis","sourcePath":"Temporal/Feature/Nexus2/Race.lean","offendingValue":"temporal.nexus2.unknown.action","relatedDefinitionIds":["temporal.nexus2.cancellation-race.target","temporal.nexus2.unknown.action"]},"role":"property","anchor":{"sourcePath":"Temporal/Feature/Nexus2/AuthoringTests.lean","line":742,"column":17,"endLine":742,"endColumn":69}}
-/
#guard_msgs (error) in
#check query% missingCapabilityQuery against frontendRaceTarget tracking [
  queryParent (Authoring.GuardedRace.family.id "query" "case-analysis"),
  propertyAnchor (Authoring.GuardedRace.family.id "property" "cases")]

/--
error: query authoring failed: {"error":{"kind":"invalid-limit","definitionId":"temporal.nexus2.cancellation-race.query.case-analysis","sourcePath":"Temporal/Feature/Nexus2/Race.lean","offendingValue":"search=0","relatedDefinitionIds":[]},"role":"limits","anchor":{"sourcePath":"Temporal/Feature/Nexus2/AuthoringTests.lean","line":750,"column":15,"endLine":750,"endColumn":72}}
-/
#guard_msgs (error) in
#check query% invalidLimitQuery against frontendRaceTarget tracking [
  queryParent (Authoring.GuardedRace.family.id "query" "case-analysis"),
  limitsAnchor (Authoring.GuardedRace.family.id "query" "case-analysis")]

/--
error: query authoring failed: {"error":{"kind":"unit-mismatch","definitionId":"temporal.nexus2.cancellation-race.query.case-analysis","sourcePath":"Temporal/Feature/Nexus2/Race.lean","offendingValue":"steps:actions","relatedDefinitionIds":[]},"role":"limits","anchor":{"sourcePath":"Temporal/Feature/Nexus2/AuthoringTests.lean","line":758,"column":15,"endLine":758,"endColumn":72}}
-/
#guard_msgs (error) in
#check query% mismatchedUnitQuery against frontendRaceTarget tracking [
  queryParent (Authoring.GuardedRace.family.id "query" "case-analysis"),
  limitsAnchor (Authoring.GuardedRace.family.id "query" "case-analysis")]

/--
error: query authoring failed: {"error":{"kind":"duplicate-property","definitionId":"temporal.nexus2.cancellation-race.query.case-analysis","sourcePath":"Temporal/Feature/Nexus2/Race.lean","offendingValue":"temporal.nexus2.cancellation-race.property.cases","relatedDefinitionIds":["temporal.nexus2.cancellation-race.property.cases"]},"role":"property","anchor":{"sourcePath":"Temporal/Feature/Nexus2/AuthoringTests.lean","line":766,"column":17,"endLine":766,"endColumn":69}}
-/
#guard_msgs (error) in
#check query% duplicatePropertyQuery against frontendRaceTarget tracking [
  queryParent (Authoring.GuardedRace.family.id "query" "case-analysis"),
  propertyAnchor (Authoring.GuardedRace.family.id "property" "cases")]

/--
error: expected queryParent, targetAnchor, propertyAnchor, scenarioAnchor, limitsAnchor, or policyAnchor
-/
#guard_msgs (error) in
#check query% mismatchedTargetQuery against frontendRaceTarget tracking [
  unsupportedQueryRole (Authoring.GuardedRace.family.id "query" "case-analysis")]

private def unknownStateProperty : Property := {
  id := (Authoring.GuardedRace.family).id "property" "unknown-state"
  source := Race.source
  requires := [Race.capabilityId]
  clauses := [.transitionContract
    (Authoring.GuardedRace.family.id "property" "unknown-state.clause")
    (.selectedAction frontendRaceModel.resolveAction)
    (.resultingState { frontendRaceModel.cancelRequestedState with
      definitionId := DefinitionId.of "temporal.nexus2.unknown.state" })]
}

private def unknownResultProperty : Property := {
  id := (Authoring.GuardedRace.family).id "property" "unknown-result"
  source := Race.source
  requires := [Race.capabilityId]
  clauses := [.transitionContract
    (Authoring.GuardedRace.family.id "property" "unknown-result.clause")
    (.selectedAction frontendRaceModel.resolveAction)
    (.outcome { frontendRaceModel.canceledOutcome with
      definitionId := DefinitionId.of "temporal.nexus2.unknown.result" })]
}

/--
error: property authoring failed: {"error":{"kind":"unknown-reference","definitionId":"temporal.nexus2.cancellation-race.property.unknown-state","sourcePath":"Temporal/Feature/Nexus2/Race.lean","offendingValue":"temporal.nexus2.unknown.state","relatedDefinitionIds":["temporal.nexus2.unknown.state"]},"role":"clause","anchor":{"sourcePath":"Temporal/Feature/Nexus2/AuthoringTests.lean","line":803,"column":15,"endLine":803,"endColumn":64}}
-/
#guard_msgs (error) in
#check property% unknownStateProperty against frontendRaceContext tracking [
  parentAnchor unknownStateProperty.id,
  clauseAnchor (DefinitionId.of "temporal.nexus2.unknown.state")]

/--
error: property authoring failed: {"error":{"kind":"unknown-reference","definitionId":"temporal.nexus2.cancellation-race.property.unknown-result","sourcePath":"Temporal/Feature/Nexus2/Race.lean","offendingValue":"temporal.nexus2.unknown.result","relatedDefinitionIds":["temporal.nexus2.unknown.result"]},"role":"clause","anchor":{"sourcePath":"Temporal/Feature/Nexus2/AuthoringTests.lean","line":811,"column":15,"endLine":811,"endColumn":65}}
-/
#guard_msgs (error) in
#check property% unknownResultProperty against frontendRaceContext tracking [
  parentAnchor unknownResultProperty.id,
  clauseAnchor (DefinitionId.of "temporal.nexus2.unknown.result")]

private def incompatibleStrategyQuery : Query :=
  { frontendGuardedAuthoredQuery with
    form := .verify frontendCheckedProperty
    policy := .shortest }

/--
error: query authoring failed: {"error":{"kind":"incompatible-strategy","definitionId":"temporal.nexus2.cancellation-race.query.case-analysis","sourcePath":"Temporal/Feature/Nexus2/Race.lean","offendingValue":"shortest","relatedDefinitionIds":[]},"role":"policy","anchor":{"sourcePath":"Temporal/Feature/Nexus2/AuthoringTests.lean","line":824,"column":15,"endLine":824,"endColumn":72}}
-/
#guard_msgs (error) in
#check query% incompatibleStrategyQuery against frontendRaceTarget tracking [
  queryParent (Authoring.GuardedRace.family.id "query" "case-analysis"),
  policyAnchor (Authoring.GuardedRace.family.id "query" "case-analysis")]

private def frontendContradictoryBehavior : Except ScenarioError CheckedScenario :=
  scenario% contradictoryBehavior frontendRaceModel against frontendRaceBehaviorContext tracking [
    scenarioParent (contradictoryBehavior frontendRaceModel).id,
    setupAnchor (Authoring.GuardedRace.family.id "setup" "not-started")]

#guard frontendContradictoryBehavior.toOption.map CheckedScenario.isUnsatisfiable == some true

private def invalidQueryHasNoCheckedValue : Option QueryErrorKind :=
  (Query.error? mismatchedTargetQuery frontendRaceTarget).map QueryError.kind

#guard invalidQueryHasNoCheckedValue == some .targetMismatch

private def movedBehaviorSpec : Scenario := {
  Authoring.GuardedRace.authoredScenario frontendRaceModel with
  source := { path := "Moved/Behavior.lean", line := 901, column := 4 }
}

private def movedBehaviorFrontend : Except ScenarioError CheckedScenario :=
  scenario% movedBehaviorSpec against frontendRaceBehaviorContext tracking [
    scenarioParent movedBehaviorSpec.id]

private def behaviorSourceMetadataDifference : Option Bool := do
  let constructor ← (Authoring.GuardedRace.authoredScenario frontendRaceModel).check
    frontendRaceBehaviorContext |>.toOption
  let moved ← movedBehaviorFrontend.toOption
  pure (moved.id == constructor.id && moved.source != constructor.source &&
    moved.canonicalMetadata != constructor.canonicalMetadata &&
    moved.behaviorFingerprint == constructor.behaviorFingerprint)

#guard behaviorSourceMetadataDifference == some true

private def movedQuery : Query :=
  { frontendGuardedAuthoredQuery with
    source := { path := "Moved/Query.lean", line := 902, column := 5 } }

private def movedQueryFrontend : Except QueryError (CheckedQuery Race.LawStatement) :=
  query% movedQuery against frontendRaceTarget tracking [
    queryParent (Authoring.GuardedRace.family.id "query" "case-analysis")]

private def querySourceMetadataDifference : Option Bool := do
  let constructor ← (Query.check (.ofTarget frontendRaceTarget)
    frontendGuardedAuthoredQuery).toOption
  let moved ← movedQueryFrontend.toOption
  pure (moved.id == constructor.id && moved.source != constructor.source &&
    moved.canonicalMetadata == constructor.canonicalMetadata &&
    moved.behaviorFingerprint == constructor.behaviorFingerprint)

#guard querySourceMetadataDifference == some true

private def openBehaviorFrontend
    (spec : Scenario)
    (context : ScenarioCheckContext) : Except ScenarioError CheckedScenario :=
  scenario% spec against context tracking [scenarioParent spec.id]

private def invalidOpenBehaviorHasNoCheckedValue : Option ScenarioErrorKind :=
  match openBehaviorFrontend malformedBehavior frontendRaceBehaviorContext with
  | .error error => some error.kind
  | .ok _ => none

#guard invalidOpenBehaviorHasNoCheckedValue == some .invalidDefinitionId

private def openQueryFrontend
    (declaration : Query)
    (target : QueryModel Race.LawStatement) :
    Except QueryError (CheckedQuery Race.LawStatement) :=
  query% declaration against target tracking [queryParent declaration.id]

private def invalidOpenQueryHasNoCheckedValue : Option QueryErrorKind := do
  let target ← Race.targetResult.toOption
  let (property, behavior, _) ← guardedAdmission
  let spec := { Authoring.GuardedRace.authoredQuery property behavior with target := otherTargetId }
  match openQueryFrontend spec target with
  | .error error => some error.kind
  | .ok _ => none

#guard invalidOpenQueryHasNoCheckedValue == some .targetMismatch

/--
error: Tactic `decide` failed for proposition
-/
#guard_msgs (error, substring := true) in
example : (Query.check (.ofTarget frontendRaceTarget)
    frontendGuardedAuthoredQuery).toOption.isSome = true := by
  decide +kernel

private def frontendCancelSpec : Property :=
  Authoring.Baseline.authoredProperty "cancel" frontendBaselineModel.cancelAction
    frontendBaselineModel.canceledState frontendBaselineModel.canceledOutcome
    frontendBaselineModel.canceledFact

private def frontendSuccessSpec : Property :=
  Authoring.Baseline.authoredProperty "success" frontendBaselineModel.reportSuccessAction
    frontendBaselineModel.succeededState frontendBaselineModel.succeededOutcome
    frontendBaselineModel.succeededFact

def frontendCancelProperty : Except PropertyError CheckedProperty :=
  property% frontendCancelSpec against frontendBaselineContext tracking [
    parentAnchor frontendCancelSpec.id]

def frontendSuccessProperty : Except PropertyError CheckedProperty :=
  property% frontendSuccessSpec against frontendBaselineContext tracking [
    parentAnchor frontendSuccessSpec.id]

private def frontendBaselineBehaviorContext : ScenarioCheckContext := {
  definitions := Lifecycle.definitions
}

private def frontendStartBehaviorSpec : Scenario :=
  Authoring.Baseline.authoredScenario "start" "scheduled" "start"
    frontendBaselineModel.scheduledState frontendBaselineModel.startAction

private def frontendCancelBehaviorSpec : Scenario :=
  Authoring.Baseline.authoredScenario "cancel" "started-cancel" "cancel"
    frontendBaselineModel.startedState frontendBaselineModel.cancelAction

private def frontendSuccessBehaviorSpec : Scenario :=
  Authoring.Baseline.authoredScenario "success" "started-success" "success"
    frontendBaselineModel.startedState frontendBaselineModel.reportSuccessAction

def frontendStartBehavior : Except ScenarioError CheckedScenario :=
  scenario% frontendStartBehaviorSpec against frontendBaselineBehaviorContext tracking [
    scenarioParent frontendStartBehaviorSpec.id,
    setupAnchor (Authoring.Baseline.family.id "setup" "scheduled"),
    occurrenceAnchor (Authoring.Baseline.family.id "occurrence" "start")]

def frontendCancelBehavior : Except ScenarioError CheckedScenario :=
  scenario% frontendCancelBehaviorSpec against frontendBaselineBehaviorContext tracking [
    scenarioParent frontendCancelBehaviorSpec.id,
    setupAnchor (Authoring.Baseline.family.id "setup" "started-cancel"),
    occurrenceAnchor (Authoring.Baseline.family.id "occurrence" "cancel")]

def frontendSuccessBehavior : Except ScenarioError CheckedScenario :=
  scenario% frontendSuccessBehaviorSpec against frontendBaselineBehaviorContext tracking [
    scenarioParent frontendSuccessBehaviorSpec.id,
    setupAnchor (Authoring.Baseline.family.id "setup" "started-success"),
    occurrenceAnchor (Authoring.Baseline.family.id "occurrence" "success")]

private structure FrontendBaseline where
  startProperty : CheckedProperty
  startBehavior : CheckedScenario
  startQuery : CheckedQuery Lifecycle.LawStatement
  cancelProperty : CheckedProperty
  cancelBehavior : CheckedScenario
  cancelQuery : CheckedQuery Lifecycle.LawStatement
  successProperty : CheckedProperty
  successBehavior : CheckedScenario
  successQuery : CheckedQuery Lifecycle.LawStatement

private def frontendBaselineAdmission : Option FrontendBaseline := do
  let constructor ← Authoring.Baseline.checkBaseline.toOption
  let startProperty ← frontendStartProperty.toOption
  let startBehavior ← frontendStartBehavior.toOption
  let startSpec := Authoring.Baseline.authoredQuery "start" startProperty startBehavior
  let startQuery ← (query% startSpec against constructor.target tracking [
    queryParent startSpec.id,
    targetAnchor startSpec.target,
    propertyAnchor startProperty.id,
    scenarioAnchor startBehavior.id,
    limitsAnchor startSpec.id]) |>.toOption
  let cancelProperty ← frontendCancelProperty.toOption
  let cancelBehavior ← frontendCancelBehavior.toOption
  let cancelSpec := Authoring.Baseline.authoredQuery "cancel" cancelProperty cancelBehavior
  let cancelQuery ← (query% cancelSpec against constructor.target tracking [
    queryParent cancelSpec.id,
    targetAnchor cancelSpec.target,
    propertyAnchor cancelProperty.id,
    scenarioAnchor cancelBehavior.id,
    limitsAnchor cancelSpec.id]) |>.toOption
  let successProperty ← frontendSuccessProperty.toOption
  let successBehavior ← frontendSuccessBehavior.toOption
  let successSpec := Authoring.Baseline.authoredQuery "success" successProperty successBehavior
  let successQuery ← (query% successSpec against constructor.target tracking [
    queryParent successSpec.id,
    targetAnchor successSpec.target,
    propertyAnchor successProperty.id,
    scenarioAnchor successBehavior.id,
    limitsAnchor successSpec.id]) |>.toOption
  pure {
    startProperty := startProperty
    startBehavior := startBehavior
    startQuery := startQuery
    cancelProperty := cancelProperty
    cancelBehavior := cancelBehavior
    cancelQuery := cancelQuery
    successProperty := successProperty
    successBehavior := successBehavior
    successQuery := successQuery
  }

private def frontendBaselineEquivalence : Option Bool := do
  let frontend ← frontendBaselineAdmission
  let constructor ← Authoring.Baseline.checkBaseline.toOption
  pure (frontend.startProperty.canonicalMetadata == constructor.start.property.canonicalMetadata &&
    frontend.startProperty.behaviorFingerprint == constructor.start.property.behaviorFingerprint &&
    frontend.startBehavior.canonicalMetadata == constructor.start.behavior.canonicalMetadata &&
    frontend.startBehavior.behaviorFingerprint == constructor.start.behavior.behaviorFingerprint &&
    frontend.startQuery.canonicalMetadata == constructor.start.query.canonicalMetadata &&
    frontend.startQuery.behaviorFingerprint == constructor.start.query.behaviorFingerprint &&
    frontend.cancelProperty.canonicalMetadata == constructor.cancel.property.canonicalMetadata &&
    frontend.cancelProperty.behaviorFingerprint == constructor.cancel.property.behaviorFingerprint &&
    frontend.cancelBehavior.canonicalMetadata == constructor.cancel.behavior.canonicalMetadata &&
    frontend.cancelBehavior.behaviorFingerprint == constructor.cancel.behavior.behaviorFingerprint &&
    frontend.cancelQuery.canonicalMetadata == constructor.cancel.query.canonicalMetadata &&
    frontend.cancelQuery.behaviorFingerprint == constructor.cancel.query.behaviorFingerprint &&
    frontend.successProperty.canonicalMetadata == constructor.success.property.canonicalMetadata &&
    frontend.successProperty.behaviorFingerprint == constructor.success.property.behaviorFingerprint &&
    frontend.successBehavior.canonicalMetadata == constructor.success.behavior.canonicalMetadata &&
    frontend.successBehavior.behaviorFingerprint == constructor.success.behavior.behaviorFingerprint &&
    frontend.successQuery.canonicalMetadata == constructor.success.query.canonicalMetadata &&
    frontend.successQuery.behaviorFingerprint == constructor.success.query.behaviorFingerprint)

#guard frontendBaselineEquivalence == some true

private def frontendBaselineTraces : Option (PlanningOutcome × PlanningOutcome × PlanningOutcome) := do
  let frontend ← frontendBaselineAdmission
  let constructor ← Authoring.Baseline.checkBaseline.toOption
  let startKernel ← SearchView.ofCheckedQuery constructor.target.id
    frontend.startQuery |>.toOption
  let cancelKernel ← SearchView.ofCheckedQuery constructor.target.id
    frontend.cancelQuery |>.toOption
  let successKernel ← SearchView.ofCheckedQuery constructor.target.id
    frontend.successQuery |>.toOption
  let startRun ← (search frontend.startQuery startKernel).toOption
  let cancelRun ← (search frontend.cancelQuery cancelKernel).toOption
  let successRun ← (search frontend.successQuery successKernel).toOption; pure (startRun.result.outcome, cancelRun.result.outcome, successRun.result.outcome)

private def constructorBaselineTraces : Option
    (PlanningOutcome × PlanningOutcome × PlanningOutcome) := do
  let constructor ← Authoring.Baseline.checkBaseline.toOption
  let startKernel ← SearchView.ofCheckedQuery constructor.target.id
    constructor.start.query |>.toOption
  let cancelKernel ← SearchView.ofCheckedQuery constructor.target.id
    constructor.cancel.query |>.toOption
  let successKernel ← SearchView.ofCheckedQuery constructor.target.id
    constructor.success.query |>.toOption
  let startRun ← (search constructor.start.query startKernel).toOption
  let cancelRun ← (search constructor.cancel.query cancelKernel).toOption
  let successRun ← (search constructor.success.query successKernel).toOption; pure (startRun.result.outcome, cancelRun.result.outcome, successRun.result.outcome)

#guard frontendBaselineTraces == constructorBaselineTraces

/--
error: Tactic `decide` failed
-/
#guard_msgs (error, substring := true) in
example : ((Authoring.GuardedRace.authoredScenario frontendRaceModel).check
    frontendRaceBehaviorContext).toOption.isSome = true := by
  decide +kernel

def constructorOneProperty : Except PropertyError CheckedProperty :=
  frontendStartSpec.check frontendBaselineContext

def constructorTenProperties : List (Except PropertyError CheckedProperty) :=
  [frontendStartSpec.check frontendBaselineContext,
    frontendStartSpec.check frontendBaselineContext,
    frontendStartSpec.check frontendBaselineContext,
    frontendStartSpec.check frontendBaselineContext,
    frontendStartSpec.check frontendBaselineContext,
    frontendStartSpec.check frontendBaselineContext,
    frontendStartSpec.check frontendBaselineContext,
    frontendStartSpec.check frontendBaselineContext,
    frontendStartSpec.check frontendBaselineContext,
    frontendStartSpec.check frontendBaselineContext]

def frontendOneProperty : Except PropertyError CheckedProperty :=
  property% frontendStartSpec against frontendBaselineContext tracking [
    parentAnchor frontendStartSpec.id]

def frontendTenProperties : List (Except PropertyError CheckedProperty) := [
  property% frontendStartSpec against frontendBaselineContext tracking [parentAnchor frontendStartSpec.id],
  property% frontendStartSpec against frontendBaselineContext tracking [parentAnchor frontendStartSpec.id],
  property% frontendStartSpec against frontendBaselineContext tracking [parentAnchor frontendStartSpec.id],
  property% frontendStartSpec against frontendBaselineContext tracking [parentAnchor frontendStartSpec.id],
  property% frontendStartSpec against frontendBaselineContext tracking [parentAnchor frontendStartSpec.id],
  property% frontendStartSpec against frontendBaselineContext tracking [parentAnchor frontendStartSpec.id],
  property% frontendStartSpec against frontendBaselineContext tracking [parentAnchor frontendStartSpec.id],
  property% frontendStartSpec against frontendBaselineContext tracking [parentAnchor frontendStartSpec.id],
  property% frontendStartSpec against frontendBaselineContext tracking [parentAnchor frontendStartSpec.id],
  property% frontendStartSpec against frontendBaselineContext tracking [parentAnchor frontendStartSpec.id]
]

#guard frontendTenProperties.all fun result =>
  result.toOption.map (fun property => property.behaviorFingerprint) ==
    frontendStartProperty.toOption.map (fun property => property.behaviorFingerprint)

#guard constructorOneProperty.toOption.isSome
#guard constructorTenProperties.all (fun result => result.toOption.isSome)
#guard frontendOneProperty.toOption.isSome
#guard frontendTenProperties.all (fun result => result.toOption.isSome)

def constructorOneBehavior : Except ScenarioError CheckedScenario :=
  frontendStartBehaviorSpec.check frontendBaselineBehaviorContext

def constructorTenBehaviors : List (Except ScenarioError CheckedScenario) := [
  frontendStartBehaviorSpec.check frontendBaselineBehaviorContext,
  frontendStartBehaviorSpec.check frontendBaselineBehaviorContext,
  frontendStartBehaviorSpec.check frontendBaselineBehaviorContext,
  frontendStartBehaviorSpec.check frontendBaselineBehaviorContext,
  frontendStartBehaviorSpec.check frontendBaselineBehaviorContext,
  frontendStartBehaviorSpec.check frontendBaselineBehaviorContext,
  frontendStartBehaviorSpec.check frontendBaselineBehaviorContext,
  frontendStartBehaviorSpec.check frontendBaselineBehaviorContext,
  frontendStartBehaviorSpec.check frontendBaselineBehaviorContext,
  frontendStartBehaviorSpec.check frontendBaselineBehaviorContext]

def frontendOneBehavior : Except ScenarioError CheckedScenario :=
  scenario% frontendStartBehaviorSpec against frontendBaselineBehaviorContext tracking [
    scenarioParent frontendStartBehaviorSpec.id,
    setupAnchor (Authoring.Baseline.family.id "setup" "scheduled"),
    occurrenceAnchor (Authoring.Baseline.family.id "occurrence" "start")]

def frontendTenBehaviors : List (Except ScenarioError CheckedScenario) := [
  scenario% frontendStartBehaviorSpec against frontendBaselineBehaviorContext tracking [scenarioParent frontendStartBehaviorSpec.id],
  scenario% frontendStartBehaviorSpec against frontendBaselineBehaviorContext tracking [scenarioParent frontendStartBehaviorSpec.id],
  scenario% frontendStartBehaviorSpec against frontendBaselineBehaviorContext tracking [scenarioParent frontendStartBehaviorSpec.id],
  scenario% frontendStartBehaviorSpec against frontendBaselineBehaviorContext tracking [scenarioParent frontendStartBehaviorSpec.id],
  scenario% frontendStartBehaviorSpec against frontendBaselineBehaviorContext tracking [scenarioParent frontendStartBehaviorSpec.id],
  scenario% frontendStartBehaviorSpec against frontendBaselineBehaviorContext tracking [scenarioParent frontendStartBehaviorSpec.id],
  scenario% frontendStartBehaviorSpec against frontendBaselineBehaviorContext tracking [scenarioParent frontendStartBehaviorSpec.id],
  scenario% frontendStartBehaviorSpec against frontendBaselineBehaviorContext tracking [scenarioParent frontendStartBehaviorSpec.id],
  scenario% frontendStartBehaviorSpec against frontendBaselineBehaviorContext tracking [scenarioParent frontendStartBehaviorSpec.id],
  scenario% frontendStartBehaviorSpec against frontendBaselineBehaviorContext tracking [scenarioParent frontendStartBehaviorSpec.id]]

#guard constructorOneBehavior.toOption.isSome
#guard constructorTenBehaviors.all (fun result => result.toOption.isSome)
#guard frontendOneBehavior.toOption.isSome
#guard frontendTenBehaviors.all (fun result => result.toOption.isSome)

def constructorOneQuery : Except QueryError (CheckedQuery Race.LawStatement) :=
  Query.check (.ofTarget frontendRaceTarget) frontendGuardedAuthoredQuery

def constructorTenQueries : List (Except QueryError (CheckedQuery Race.LawStatement)) := [
  Query.check (.ofTarget frontendRaceTarget) frontendGuardedAuthoredQuery,
  Query.check (.ofTarget frontendRaceTarget) frontendGuardedAuthoredQuery,
  Query.check (.ofTarget frontendRaceTarget) frontendGuardedAuthoredQuery,
  Query.check (.ofTarget frontendRaceTarget) frontendGuardedAuthoredQuery,
  Query.check (.ofTarget frontendRaceTarget) frontendGuardedAuthoredQuery,
  Query.check (.ofTarget frontendRaceTarget) frontendGuardedAuthoredQuery,
  Query.check (.ofTarget frontendRaceTarget) frontendGuardedAuthoredQuery,
  Query.check (.ofTarget frontendRaceTarget) frontendGuardedAuthoredQuery,
  Query.check (.ofTarget frontendRaceTarget) frontendGuardedAuthoredQuery,
  Query.check (.ofTarget frontendRaceTarget) frontendGuardedAuthoredQuery]

def frontendOneQuery : Except QueryError (CheckedQuery Race.LawStatement) :=
  query% frontendGuardedAuthoredQuery against frontendRaceTarget tracking [
    queryParent (Authoring.GuardedRace.family.id "query" "case-analysis"),
    targetAnchor Race.targetId,
    propertyAnchor (Authoring.GuardedRace.family.id "property" "cases"),
    scenarioAnchor (Authoring.GuardedRace.family.id "behavior" "request-then-resolve"),
    limitsAnchor (Authoring.GuardedRace.family.id "query" "case-analysis")]

def frontendTenQueries : List (Except QueryError (CheckedQuery Race.LawStatement)) := [
  query% frontendGuardedAuthoredQuery against frontendRaceTarget tracking [queryParent (Authoring.GuardedRace.family.id "query" "case-analysis")],
  query% frontendGuardedAuthoredQuery against frontendRaceTarget tracking [queryParent (Authoring.GuardedRace.family.id "query" "case-analysis")],
  query% frontendGuardedAuthoredQuery against frontendRaceTarget tracking [queryParent (Authoring.GuardedRace.family.id "query" "case-analysis")],
  query% frontendGuardedAuthoredQuery against frontendRaceTarget tracking [queryParent (Authoring.GuardedRace.family.id "query" "case-analysis")],
  query% frontendGuardedAuthoredQuery against frontendRaceTarget tracking [queryParent (Authoring.GuardedRace.family.id "query" "case-analysis")],
  query% frontendGuardedAuthoredQuery against frontendRaceTarget tracking [queryParent (Authoring.GuardedRace.family.id "query" "case-analysis")],
  query% frontendGuardedAuthoredQuery against frontendRaceTarget tracking [queryParent (Authoring.GuardedRace.family.id "query" "case-analysis")],
  query% frontendGuardedAuthoredQuery against frontendRaceTarget tracking [queryParent (Authoring.GuardedRace.family.id "query" "case-analysis")],
  query% frontendGuardedAuthoredQuery against frontendRaceTarget tracking [queryParent (Authoring.GuardedRace.family.id "query" "case-analysis")],
  query% frontendGuardedAuthoredQuery against frontendRaceTarget tracking [queryParent (Authoring.GuardedRace.family.id "query" "case-analysis")]]

private def queryAdmissionSucceeded
    (result : Except QueryError (CheckedQuery Race.LawStatement)) : Bool :=
  result.toOption.isSome

#guard queryAdmissionSucceeded constructorOneQuery
#guard constructorTenQueries.all queryAdmissionSucceeded
#guard queryAdmissionSucceeded frontendOneQuery
#guard frontendTenQueries.all queryAdmissionSucceeded

private def minimalPropertySpec : Property := {
  id := (DefinitionFamily.mk (DefinitionId.of "test.frontend")).id "property" "minimal"
  source := Race.source
  requires := []
  clauses := []
}

/--
error: Tactic `decide` failed
-/
#guard_msgs (error, substring := true) in
example : (Property.check
    { definitions := [], providers := [], meanings := [] }
    { id := (DefinitionFamily.mk (DefinitionId.of "test.frontend")).id "property" "minimal"
      source := Race.source
      requires := []
      clauses := [] }).toOption.isSome = true := by
  decide +kernel

#print axioms Authoring.Baseline.checkBaseline
#print axioms Authoring.GuardedRace.authoredProperty
#print axioms Authoring.GuardedRace.authoredScenario
#print axioms Authoring.GuardedRace.authoredQuery
#print axioms Property.check
#print axioms Property.checked
#print axioms Scenario.check
#print axioms Scenario.checked
#print axioms Query.check
#print axioms Query.checked
#print axioms frontendGuardedBehavior
#print axioms frontendGuardedAdmission
#print axioms frontendBaselineAdmission
#print axioms frontendOneQuery
#print axioms Race.targetResult

end Temporal.Feature.Nexus2.AuthoringTests
