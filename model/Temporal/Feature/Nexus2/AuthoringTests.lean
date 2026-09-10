import Temporal.Feature.Nexus2.Authoring
import Umpire.Planning

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

private def frontendStartSpec : PropertySpec :=
  Authoring.Baseline.propertySpec "start" frontendBaselineModel.startAction
    frontendBaselineModel.startedState frontendBaselineModel.startedOutcome
    frontendBaselineModel.startedFact

def frontendStartProperty : Except PropertyError CheckedProperty :=
  property% frontendStartSpec against frontendBaselineContext
    tracking [parentAnchor frontendStartSpec.declaration.id]

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
  property% Authoring.GuardedRace.propertySpec frontendRaceModel against frontendRaceContext
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
      (Authoring.GuardedRace.propertySpec frontendRaceModel).check frontendRaceContext with
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
  let startKernel ← (IncrementalPlannerKernel.ofCheckedQuery checked.target.id checked.start.query).toOption
  let cancelKernel ← (IncrementalPlannerKernel.ofCheckedQuery checked.target.id checked.cancel.query).toOption
  let successKernel ← (IncrementalPlannerKernel.ofCheckedQuery checked.target.id checked.success.query).toOption
  let startRun ← (plan checked.start.query startKernel).toOption
  let cancelRun ← (plan checked.cancel.query cancelKernel).toOption
  let successRun ← (plan checked.success.query successKernel).toOption; pure (startRun.result.outcome.name, cancelRun.result.outcome.name, successRun.result.outcome.name)

#guard constructorRunNames == some ("found", "found", "found")

private def guardedAdmission : Option (CheckedProperty × CheckedBehavior × CheckedQuery Race.LawStatement) := do
  let target ← Race.targetResult.toOption
  let model ← Race.modelVocabulary.toOption
  let property ← (Authoring.GuardedRace.propertySpec model).check
    (PropertyCheckContext.ofTarget target) |>.toOption
  let behavior ← (Authoring.GuardedRace.behaviorSpec model).check (.ofTarget target) |>.toOption
  let query ← (Authoring.GuardedRace.querySpec property behavior).check target |>.toOption
  pure (property, behavior, query)

#guard guardedAdmission.isSome

private def guardedBehaviorEquivalence : Option Bool := do
  let target ← Race.targetResult.toOption
  let model ← Race.modelVocabulary.toOption
  let authored ← (Authoring.GuardedRace.behaviorSpec model).check (.ofTarget target) |>.toOption
  let established ← checkBehavior (.ofTarget target) (Race.exactBehaviorDeclaration model) |>.toOption
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
    (DefinitionId × List DefinitionId × List DefinitionId × QueryForm × QueryLimits × PlannerPolicy) := do
  let (property, behavior, query) ← guardedAdmission
  pure (property.id, property.guardedClauseIds,
    behavior.requiredOccurrences.map NamedOccurrence.id, query.form, query.limits, query.policy)

#guard inspectedGuardedSurface.map (fun (propertyId, clauses, occurrences, form, limits, policy) =>
  propertyId == Authoring.GuardedRace.family.id "property" "cases" &&
    clauses == [Authoring.GuardedRace.family.id "case-group" "lifecycle"] &&
    occurrences == [
      Authoring.GuardedRace.family.id "occurrence" "request",
      Authoring.GuardedRace.family.id "occurrence" "resolution"
    ] &&
    (match form with | .select [_] => true | _ => false) &&
    limits == QueryLimits.bounded 2 2 32 && policy == PlannerPolicy.exhaustive) == some true

private def propertyErrorKind (spec : PropertySpec) : Option PropertyErrorKind := do
  let target ← Race.targetResult.toOption
  match spec.check (PropertyCheckContext.ofTarget target) with
  | .error error => some error.kind
  | .ok _ => none

private def malformedProperty : PropertySpec := {
  family := { root := DefinitionId.of "temporal.nexus2." }
  key := "malformed"
  source := Race.source
  requires := []
  clauses := []
}

#guard propertyErrorKind malformedProperty == some .invalidDefinitionId

private def duplicateClauseProperty (model : Race.ModelVocabulary) : PropertySpec :=
  let clause := PropertyClause.transitionContract
    (Authoring.GuardedRace.family.id "property" "duplicate.clause")
    (.selectedAction model.resolveAction) (.modelOutcome model.canceledOutcome)
  {
    family := Authoring.GuardedRace.family
    key := "duplicate"
    source := Race.source
    requires := [Race.capabilityId]
    clauses := [clause, clause]
  }

#guard Race.modelVocabulary.toOption.bind (fun model =>
  propertyErrorKind (duplicateClauseProperty model)) == some .duplicateDefinitionId

private def unknownReferenceProperty (model : Race.ModelVocabulary) : PropertySpec := {
  family := Authoring.GuardedRace.family
  key := "unknown-reference"
  source := Race.source
  requires := [Race.capabilityId]
  clauses := [.transitionContract
    (Authoring.GuardedRace.family.id "property" "unknown-reference.clause")
    (.selectedAction { model.resolveAction with
      definitionId := DefinitionId.of "temporal.nexus2.unknown.action" })
    (.modelOutcome model.canceledOutcome)]
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

private def missingCapabilityProperty (model : Race.ModelVocabulary) : PropertySpec := {
  family := Authoring.GuardedRace.family
  key := "missing-capability"
  source := Race.source
  requires := []
  clauses := [.transitionContract
    (Authoring.GuardedRace.family.id "property" "missing-capability.clause")
    (.selectedAction model.resolveAction) (.modelOutcome model.canceledOutcome)]
}

#guard Race.modelVocabulary.toOption.bind (fun model =>
  propertyErrorKind (missingCapabilityProperty model)) == some .undeclaredReference

private def unsupportedGuardProperty (model : Race.ModelVocabulary) : PropertySpec := {
  family := Authoring.GuardedRace.family
  key := "unsupported-guard"
  source := Race.source
  version := 2
  requires := [Race.capabilityId]
  clauses := [.sameStepCases {
    id := Authoring.GuardedRace.family.id "case-group" "unsupported-guard"
    source := Race.source
    guard := .resultingStateIs model.cancelRequestedState
    cases := [Authoring.GuardedRace.requestCase model]
  }]
}

#guard Race.modelVocabulary.toOption.bind (fun model =>
  propertyErrorKind (unsupportedGuardProperty model)) == some .invalidPredicateContext

private def emptyGroupProperty : PropertySpec := {
  family := Authoring.GuardedRace.family
  key := "empty-group"
  source := Race.source
  version := 2
  requires := [Race.capabilityId]
  clauses := [.sameStepCases {
    id := Authoring.GuardedRace.family.id "case-group" "empty"
    source := Race.source
    guard := .selectedActionIs frontendRaceModel.resolveAction
    cases := []
  }]
}

private def emptyBooleanProperty : PropertySpec := {
  family := Authoring.GuardedRace.family
  key := "empty-boolean"
  source := Race.source
  version := 2
  requires := [Race.capabilityId]
  clauses := [.sameStepCases {
    id := Authoring.GuardedRace.family.id "case-group" "empty-boolean"
    source := Race.source
    guard := .all []
    cases := [Authoring.GuardedRace.resolutionCase frontendRaceModel]
  }]
}

private def invalidUnitProperty : PropertySpec := {
  family := Authoring.GuardedRace.family
  key := "invalid-unit"
  source := Race.source
  requires := [Race.capabilityId]
  clauses := [.eventuallyWithin
    (Authoring.GuardedRace.family.id "property" "invalid-unit.clause")
    (.selectedAction frontendRaceModel.resolveAction) (.fact frontendRaceModel.terminalFact)
    (.exact { value := 1, unit := .candidateEvaluations })]
}

private def wrongKindProperty : PropertySpec := {
  family := Authoring.GuardedRace.family
  key := "wrong-kind"
  source := Race.source
  requires := [Race.capabilityId]
  clauses := [.transitionContract
    (Authoring.GuardedRace.family.id "property" "wrong-kind.clause")
    (.selectedAction frontendRaceModel.startedState) (.modelOutcome frontendRaceModel.canceledOutcome)]
}

private def invalidExceptionProperty : PropertySpec :=
  let request := Authoring.GuardedRace.requestCase frontendRaceModel
  { Authoring.GuardedRace.propertySpec frontendRaceModel with clauses := [.sameStepCases {
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
error: property authoring failed: {"error":{"kind":"invalid-definition-id","definitionId":"temporal.nexus2..property.malformed","sourcePath":"Temporal/Feature/Nexus2/Race.lean","offendingValue":"temporal.nexus2..property.malformed","relatedDefinitionIds":["temporal.nexus2..property.malformed"]},"role":"parent","anchor":{"sourcePath":"Temporal/Feature/Nexus2/AuthoringTests.lean","line":316,"column":15,"endLine":316,"endColumn":47}}
-/
#guard_msgs (error) in
#check property% malformedProperty against frontendRaceContext tracking [
  parentAnchor malformedProperty.declaration.id]

/--
error: property authoring failed: {"error":{"kind":"duplicate-definition-id","definitionId":"temporal.nexus2.cancellation-race.property.duplicate","sourcePath":"Temporal/Feature/Nexus2/Race.lean","offendingValue":"temporal.nexus2.cancellation-race.property.duplicate.clause","relatedDefinitionIds":["temporal.nexus2.cancellation-race.property.duplicate.clause"]},"role":"clause","anchor":{"sourcePath":"Temporal/Feature/Nexus2/AuthoringTests.lean","line":325,"column":15,"endLine":325,"endColumn":78}}
-/
#guard_msgs (error) in
#check property% duplicateClauseProperty frontendRaceModel against frontendRaceContext tracking [
  parentAnchor (duplicateClauseProperty frontendRaceModel).declaration.id,
  clauseAnchor (Authoring.GuardedRace.family.id "property" "duplicate.clause"),
  clauseAnchor (Authoring.GuardedRace.family.id "property" "duplicate.clause")]

/--
error: property authoring failed: {"error":{"kind":"unknown-reference","definitionId":"temporal.nexus2.cancellation-race.property.unknown-reference","sourcePath":"Temporal/Feature/Nexus2/Race.lean","offendingValue":"temporal.nexus2.unknown.action","relatedDefinitionIds":["temporal.nexus2.unknown.action"]},"role":"clause","anchor":{"sourcePath":"Temporal/Feature/Nexus2/AuthoringTests.lean","line":333,"column":15,"endLine":333,"endColumn":65}}
-/
#guard_msgs (error) in
#check property% unknownReferenceProperty frontendRaceModel against frontendRaceContext tracking [
  parentAnchor (unknownReferenceProperty frontendRaceModel).declaration.id,
  clauseAnchor (DefinitionId.of "temporal.nexus2.unknown.action")]

/--
error: property authoring failed: {"error":{"kind":"missing-capability","definitionId":"temporal.nexus2.cancellation-race.property.cases","sourcePath":"Temporal/Feature/Nexus2/Race.lean","offendingValue":"temporal.nexus2.cancellation-race.capability","relatedDefinitionIds":["temporal.nexus2.cancellation-race.capability"]},"role":"parent","anchor":{"sourcePath":"Temporal/Feature/Nexus2/AuthoringTests.lean","line":340,"column":15,"endLine":340,"endColumn":32}}
-/
#guard_msgs (error) in
#check property% Authoring.GuardedRace.propertySpec frontendRaceModel against missingProviderContext tracking [
  parentAnchor Race.capabilityId]

/--
error: property authoring failed: {"error":{"kind":"invalid-predicate-context","definitionId":"temporal.nexus2.cancellation-race.property.unsupported-guard","sourcePath":"Temporal/Feature/Nexus2/Race.lean","source":{"path":"Temporal/Feature/Nexus2/Race.lean","line":1,"column":1,"provenance":"lean-model"},"offendingValue":"guard: resulting-state","relatedDefinitionIds":["temporal.nexus2.cancellation-race.state.operation"]},"role":"case","anchor":{"sourcePath":"Temporal/Feature/Nexus2/AuthoringTests.lean","line":348,"column":13,"endLine":348,"endColumn":64}}
-/
#guard_msgs (error) in
#check property% unsupportedGuardProperty frontendRaceModel against frontendRaceContext tracking [
  parentAnchor (unsupportedGuardProperty frontendRaceModel).declaration.id,
  caseAnchor frontendRaceModel.cancelRequestedState.definitionId]

/--
error: property authoring failed: {"error":{"kind":"empty-case-group","definitionId":"temporal.nexus2.cancellation-race.property.empty-group","sourcePath":"Temporal/Feature/Nexus2/Race.lean","source":{"path":"Temporal/Feature/Nexus2/Race.lean","line":1,"column":1,"provenance":"lean-model"},"offendingValue":"temporal.nexus2.cancellation-race.case-group.empty","relatedDefinitionIds":["temporal.nexus2.cancellation-race.case-group.empty"]},"role":"clause","anchor":{"sourcePath":"Temporal/Feature/Nexus2/AuthoringTests.lean","line":356,"column":15,"endLine":356,"endColumn":69}}
-/
#guard_msgs (error) in
#check property% emptyGroupProperty against frontendRaceContext tracking [
  parentAnchor emptyGroupProperty.declaration.id,
  clauseAnchor (Authoring.GuardedRace.family.id "case-group" "empty")]

/--
error: property authoring failed: {"error":{"kind":"empty-boolean-group","definitionId":"temporal.nexus2.cancellation-race.property.empty-boolean","sourcePath":"Temporal/Feature/Nexus2/Race.lean","source":{"path":"Temporal/Feature/Nexus2/Race.lean","line":1,"column":1,"provenance":"lean-model"},"offendingValue":"all","relatedDefinitionIds":[]},"role":"parent","anchor":{"sourcePath":"Temporal/Feature/Nexus2/AuthoringTests.lean","line":363,"column":15,"endLine":363,"endColumn":50}}
-/
#guard_msgs (error) in
#check property% emptyBooleanProperty against frontendRaceContext tracking [
  parentAnchor emptyBooleanProperty.declaration.id,
  clauseAnchor (Authoring.GuardedRace.family.id "case-group" "empty-boolean")]

/--
error: property authoring failed: {"error":{"kind":"unit-mismatch","definitionId":"temporal.nexus2.cancellation-race.property.invalid-unit","sourcePath":"Temporal/Feature/Nexus2/Race.lean","offendingValue":"candidate-evaluations is not a Property position unit","relatedDefinitionIds":["temporal.nexus2.cancellation-race.action.resolve","temporal.nexus2.cancellation-race.fact.terminal"]},"role":"parent","anchor":{"sourcePath":"Temporal/Feature/Nexus2/AuthoringTests.lean","line":371,"column":15,"endLine":371,"endColumn":49}}
-/
#guard_msgs (error) in
#check property% invalidUnitProperty against frontendRaceContext tracking [
  parentAnchor invalidUnitProperty.declaration.id,
  clauseAnchor (Authoring.GuardedRace.family.id "property" "invalid-unit.clause")]

/--
error: property authoring failed: {"error":{"kind":"wrong-reference-kind","definitionId":"temporal.nexus2.cancellation-race.property.wrong-kind","sourcePath":"Temporal/Feature/Nexus2/Race.lean","offendingValue":"temporal.nexus2.cancellation-race.state.operation: expected action, found state","relatedDefinitionIds":["temporal.nexus2.cancellation-race.state.operation"]},"role":"clause","anchor":{"sourcePath":"Temporal/Feature/Nexus2/AuthoringTests.lean","line":380,"column":15,"endLine":380,"endColumn":58}}
-/
#guard_msgs (error) in
#check property% wrongKindProperty against frontendRaceContext tracking [
  parentAnchor wrongKindProperty.declaration.id,
  clauseAnchor frontendRaceModel.startedState.definitionId]

/--
error: property authoring failed: {"error":{"kind":"invalid-predicate-context","definitionId":"temporal.nexus2.cancellation-race.property.cases","sourcePath":"Temporal/Feature/Nexus2/Race.lean","source":{"path":"Temporal/Feature/Nexus2/Race.lean","line":1,"column":1,"provenance":"lean-model"},"offendingValue":"guard: resulting-state","relatedDefinitionIds":["temporal.nexus2.cancellation-race.state.operation"]},"role":"exception","anchor":{"sourcePath":"Temporal/Feature/Nexus2/AuthoringTests.lean","line":389,"column":18,"endLine":389,"endColumn":69}}
-/
#guard_msgs (error) in
#check property% invalidExceptionProperty against frontendRaceContext tracking [
  parentAnchor invalidExceptionProperty.declaration.id,
  caseAnchor (Authoring.GuardedRace.family.id "case" "request"),
  exceptionAnchor frontendRaceModel.cancelRequestedState.definitionId]

/--
error: expected parentAnchor, caseAnchor, exceptionAnchor, or clauseAnchor
-/
#guard_msgs (error) in
#check property% malformedProperty against frontendRaceContext tracking [
  unsupportedOperator malformedProperty.declaration.id]

private def emptyCaseProperty (model : Race.ModelVocabulary) : PropertySpec :=
  let request := Authoring.GuardedRace.requestCase model
  { Authoring.GuardedRace.propertySpec model with clauses := [.sameStepCases {
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
error: property authoring failed: {"error":{"kind":"empty-case","definitionId":"temporal.nexus2.cancellation-race.property.cases","sourcePath":"Temporal/Feature/Nexus2/Race.lean","source":{"path":"Temporal/Feature/Nexus2/Race.lean","line":1,"column":1,"provenance":"lean-model"},"offendingValue":"temporal.nexus2.cancellation-race.case.empty","relatedDefinitionIds":["temporal.nexus2.cancellation-race.case-group.empty-case","temporal.nexus2.cancellation-race.case.empty"]},"role":"case","anchor":{"sourcePath":"Temporal/Feature/Nexus2/AuthoringTests.lean","line":418,"column":13,"endLine":418,"endColumn":61}}
-/
#guard_msgs (error) in
#check property% emptyCaseProperty frontendRaceModel against frontendRaceContext tracking [
  parentAnchor (emptyCaseProperty frontendRaceModel).declaration.id,
  clauseAnchor (Authoring.GuardedRace.family.id "case-group" "empty-case"),
  caseAnchor (Authoring.GuardedRace.family.id "case" "empty")]

private def behaviorErrorKind (spec : ExactSequenceSpec) : Option BehaviorErrorKind := do
  let target ← Race.targetResult.toOption
  match spec.check (.ofTarget target) with
  | .error error => some error.kind
  | .ok _ => none

private def duplicateOccurrenceBehavior (model : Race.ModelVocabulary) : ExactSequenceSpec := {
  Authoring.GuardedRace.behaviorSpec model with
  key := "duplicate-occurrence"
  occurrences := [
    { key := "same", action := model.requestCancelAction.definitionId },
    { key := "same", action := model.resolveAction.definitionId }
  ]
}

#guard Race.modelVocabulary.toOption.bind (fun model =>
  behaviorErrorKind (duplicateOccurrenceBehavior model)) == some .duplicateDefinitionId

private def contradictoryBehavior (model : Race.ModelVocabulary) : ExactSequenceSpec := {
  Authoring.GuardedRace.behaviorSpec model with
  key := "contradictory"
  setup := (Authoring.GuardedRace.behaviorSpec model).setup ++ [{
    id := Authoring.GuardedRace.family.id "setup" "not-started"
    relation := .different
    left := .role Race.operationRoleId
    right := .value model.startedState
  }]
}

#guard Race.modelVocabulary.toOption.bind (fun model =>
  behaviorErrorKind (contradictoryBehavior model)) == none

#guard Race.modelVocabulary.toOption.bind (fun model => do
  let target ← Race.targetResult.toOption
  let checked ← (contradictoryBehavior model).check (.ofTarget target) |>.toOption
  pure checked.isUnsatisfiable) == some true

private def queryErrorKind
    (target : QueryModel Race.LawStatement)
    (spec : QuerySpec) : Option QueryErrorKind :=
  match spec.check target with
  | .error error => some error.kind
  | .ok _ => none

private def wrongTargetQuery : Option QueryErrorKind := do
  let target ← Race.targetResult.toOption
  let (property, behavior, _) ← guardedAdmission
  let spec := { Authoring.GuardedRace.querySpec property behavior with
    target := DefinitionId.of "temporal.nexus2.other.target" }
  queryErrorKind target spec

#guard wrongTargetQuery == some .targetMismatch

private def wrongTargetDiagnostic : Option (QueryErrorKind × List DefinitionId) := do
  let target ← Race.targetResult.toOption
  let (property, behavior, _) ← guardedAdmission
  let other := DefinitionId.of "temporal.nexus2.other.target"
  let spec := { Authoring.GuardedRace.querySpec property behavior with target := other }
  match spec.check target with
  | .error error => some (error.kind, error.relatedDefinitionIds)
  | .ok _ => none

#guard wrongTargetDiagnostic == some (
  .targetMismatch, [DefinitionId.of "temporal.nexus2.cancellation-race.target",
    DefinitionId.of "temporal.nexus2.other.target"])

private def wrongUnitQuery : Option QueryErrorKind := do
  let target ← Race.targetResult.toOption
  let (property, behavior, _) ← guardedAdmission
  let declaration := { (Authoring.GuardedRace.querySpec property behavior).declaration with
    limits := {
      behavior := {
        transitions := { value := 2, unit := .selectedActions }
        selectedActions := { value := 2, unit := .selectedActions }
      }
      search := { value := 32, unit := .candidateEvaluations }
    }
  }
  match checkQuery (.ofTarget target) declaration with
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
    (CaseAnalysisStatus × DefinitionId × List DefinitionId × List DefinitionId ×
      Option ModelValue × Option ModelValue) := do
  let target ← Race.targetResult.toOption
  let model ← Race.modelVocabulary.toOption
  let spec := Authoring.GuardedRace.withoutReplacement model
  let property ← spec.check (PropertyCheckContext.ofTarget target) |>.toOption
  let behavior ← (Authoring.GuardedRace.behaviorSpec model).check (.ofTarget target) |>.toOption
  let query ← (Authoring.GuardedRace.querySpec property behavior).check target |>.toOption
  let kernel ← IncrementalPlannerKernel.ofCheckedQuery query.target.id query |>.toOption
  let result := analyzeCases query kernel
  let finding ← result.findings.find? fun finding => finding.kind == .missingReplacement
  pure (result.status, finding.parentId, finding.caseIds,
    finding.exceptions.map ResolvedPropertyException.id,
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
  let moved := { Authoring.Baseline.propertySpec "start" checked.model.startAction
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
  let changed := Authoring.Baseline.propertySpec "start" checked.model.startAction
    checked.model.canceledState checked.model.startedOutcome checked.model.startedFact
  let changed ← changed.check (PropertyCheckContext.ofTarget checked.target) |>.toOption
  pure (changed.behaviorFingerprint != checked.start.property.behaviorFingerprint)

#guard changedBehaviorFingerprint == some true

private def allQueryForms : Option (List QueryClaim) := do
  let target ← Race.targetResult.toOption
  let (property, behavior, _) ← guardedAdmission
  let base := Authoring.GuardedRace.querySpec property behavior
  [.verify property, .witness property, .counterexample property, .select [property]].mapM fun form => do
    let checked ← ({ base with form }).check target |>.toOption
    pure checked.claim

#guard allQueryForms == some [
  .verifiedWithinLimits, .satisfyingWitness, .violatingCounterexample, .limitedSelection]

private def frontendRaceBehaviorContext : BehaviorCheckContext := {
  definitions := Race.definitions
}

def frontendGuardedBehavior : Except BehaviorError CheckedBehavior :=
  behavior% Authoring.GuardedRace.behaviorSpec frontendRaceModel
    against frontendRaceBehaviorContext tracking [
      behaviorParent (Authoring.GuardedRace.behaviorSpec frontendRaceModel).declaration.id,
      setupAnchor (Authoring.GuardedRace.family.id "setup" "started"),
      occurrenceAnchor (Authoring.GuardedRace.family.id "occurrence" "request"),
      occurrenceAnchor (Authoring.GuardedRace.family.id "occurrence" "resolution")]

private def frontendGuardedBehaviorEquivalence : Bool :=
  match frontendGuardedBehavior,
      (Authoring.GuardedRace.behaviorSpec frontendRaceModel).check frontendRaceBehaviorContext with
  | .ok frontend, .ok constructor =>
      frontend.id == constructor.id &&
        frontend.canonicalMetadata == constructor.canonicalMetadata &&
        frontend.behaviorFingerprint == constructor.behaviorFingerprint
  | _, _ => false

#guard frontendGuardedBehaviorEquivalence

private def frontendGuardedQueryInput : Option (QueryAuthoringInput Race.LawStatement) := do
  let target ← Race.targetResult.toOption
  let property ← frontendGuardedProperty.toOption
  let behavior ← frontendGuardedBehavior.toOption
  pure (QueryAuthoringInput.ofSpec (Authoring.GuardedRace.querySpec property behavior) target)

def frontendGuardedQuery : Option (Except QueryError (CheckedQuery Race.LawStatement)) :=
  query% frontendGuardedQueryInput tracking [
    queryParent (Authoring.GuardedRace.family.id "query" "case-analysis"),
    targetAnchor Race.targetId,
    propertyAnchor (Authoring.GuardedRace.family.id "property" "cases"),
    behaviorAnchor (Authoring.GuardedRace.family.id "behavior" "request-then-resolve"),
    limitsAnchor (Authoring.GuardedRace.family.id "query" "case-analysis")]

private def frontendGuardedAdmission : Option
    (CheckedProperty × CheckedBehavior × CheckedQuery Race.LawStatement) := do
  let input ← frontendGuardedQueryInput
  let queryResult ← frontendGuardedQuery
  let query ← queryResult.toOption
  let property ← frontendGuardedProperty.toOption
  let behavior ← frontendGuardedBehavior.toOption
  if query.target.id != input.target.id then none else
  pure (property, behavior, query)

private def frontendGuardedQueryEquivalence : Option Bool := do
  let (_, _, frontend) ← frontendGuardedAdmission
  let (_, _, constructor) ← guardedAdmission
  pure (frontend.id == constructor.id &&
    frontend.canonicalMetadata == constructor.canonicalMetadata &&
    frontend.behaviorFingerprint == constructor.behaviorFingerprint &&
    frontend.form == constructor.form && frontend.quantifier == constructor.quantifier &&
    frontend.claim == constructor.claim && frontend.limits == constructor.limits &&
    frontend.policy == constructor.policy &&
    frontend.modelProviders == constructor.modelProviders)

#guard frontendGuardedQueryEquivalence == some true

private def frontendGuardedQueryOutcome : Option (String × String) := do
  let (_, _, frontend) ← frontendGuardedAdmission
  let (_, _, constructor) ← guardedAdmission
  let frontendKernel ← IncrementalPlannerKernel.ofCheckedQuery frontend.target.id frontend |>.toOption
  let constructorKernel ← IncrementalPlannerKernel.ofCheckedQuery constructor.target.id constructor |>.toOption
  let frontendRun ← (plan frontend frontendKernel).toOption
  let constructorRun ← (plan constructor constructorKernel).toOption; pure (frontendRun.result.outcome.name, constructorRun.result.outcome.name)

#guard frontendGuardedQueryOutcome == some ("found", "found")

private def malformedBehavior : ExactSequenceSpec := {
  Authoring.GuardedRace.behaviorSpec frontendRaceModel with
  family := { root := DefinitionId.of "temporal.nexus2." }
  key := "malformed"
}

private def unknownActionId : DefinitionId :=
  DefinitionId.of "temporal.nexus2.unknown.action"

private def unknownActionBehavior : ExactSequenceSpec := {
  Authoring.GuardedRace.behaviorSpec frontendRaceModel with
  key := "unknown-action"
  occurrences := [{ key := "unknown", action := unknownActionId }]
}

private def wrongKindBehavior : ExactSequenceSpec := {
  Authoring.GuardedRace.behaviorSpec frontendRaceModel with
  key := "wrong-kind"
  occurrences := [{ key := "wrong-kind", action := frontendRaceModel.startedState.definitionId }]
}

/--
error: behavior authoring failed: {"error":{"kind":"invalid-definition-id","definitionId":"temporal.nexus2..behavior.malformed","sourcePath":"Temporal/Feature/Nexus2/Race.lean","offendingValue":"temporal.nexus2..behavior.malformed","relatedDefinitionIds":["temporal.nexus2..behavior.malformed"]},"role":"parent","anchor":{"sourcePath":"Temporal/Feature/Nexus2/AuthoringTests.lean","line":668,"column":17,"endLine":668,"endColumn":49}}
-/
#guard_msgs (error) in
#check behavior% malformedBehavior against frontendRaceBehaviorContext tracking [
  behaviorParent malformedBehavior.declaration.id]

/--
error: behavior authoring failed: {"error":{"kind":"duplicate-definition-id","definitionId":"temporal.nexus2.cancellation-race.behavior.duplicate-occurrence","sourcePath":"Temporal/Feature/Nexus2/Race.lean","offendingValue":"temporal.nexus2.cancellation-race.occurrence.same","relatedDefinitionIds":["temporal.nexus2.cancellation-race.occurrence.same"]},"role":"occurrence","anchor":{"sourcePath":"Temporal/Feature/Nexus2/AuthoringTests.lean","line":678,"column":19,"endLine":678,"endColumn":72}}
-/
#guard_msgs (error) in
#check behavior% duplicateOccurrenceBehavior frontendRaceModel
    against frontendRaceBehaviorContext tracking [
  behaviorParent (duplicateOccurrenceBehavior frontendRaceModel).declaration.id,
  occurrenceAnchor (Authoring.GuardedRace.family.id "occurrence" "same"),
  occurrenceAnchor (Authoring.GuardedRace.family.id "occurrence" "same")]

/--
error: behavior authoring failed: {"error":{"kind":"unknown-reference","definitionId":"temporal.nexus2.cancellation-race.behavior.unknown-action","sourcePath":"Temporal/Feature/Nexus2/Race.lean","offendingValue":"temporal.nexus2.unknown.action","relatedDefinitionIds":["temporal.nexus2.unknown.action"]},"role":"action","anchor":{"sourcePath":"Temporal/Feature/Nexus2/AuthoringTests.lean","line":686,"column":15,"endLine":686,"endColumn":30}}
-/
#guard_msgs (error) in
#check behavior% unknownActionBehavior against frontendRaceBehaviorContext tracking [
  behaviorParent unknownActionBehavior.declaration.id,
  actionAnchor unknownActionId]

/--
error: behavior authoring failed: {"error":{"kind":"wrong-reference-kind","definitionId":"temporal.nexus2.cancellation-race.behavior.wrong-kind","sourcePath":"Temporal/Feature/Nexus2/Race.lean","offendingValue":"temporal.nexus2.cancellation-race.state.operation: expected action, found state","relatedDefinitionIds":["temporal.nexus2.cancellation-race.state.operation"]},"role":"action","anchor":{"sourcePath":"Temporal/Feature/Nexus2/AuthoringTests.lean","line":694,"column":15,"endLine":694,"endColumn":58}}
-/
#guard_msgs (error) in
#check behavior% wrongKindBehavior against frontendRaceBehaviorContext tracking [
  behaviorParent wrongKindBehavior.declaration.id,
  actionAnchor frontendRaceModel.startedState.definitionId]

/--
error: expected behaviorParent, setupAnchor, occurrenceAnchor, or actionAnchor
-/
#guard_msgs (error) in
#check behavior% malformedBehavior against frontendRaceBehaviorContext tracking [
  unsupportedBehaviorRole malformedBehavior.declaration.id]

private def mapGuardedQueryInput
    (change : QueryDeclaration → QueryDeclaration) :
    Option (QueryAuthoringInput Race.LawStatement) := do
  let input ← frontendGuardedQueryInput
  pure { input with declaration := change input.declaration }

private def otherTargetId : DefinitionId :=
  DefinitionId.of "temporal.nexus2.other.target"

private def wrongTargetQueryInput : Option (QueryAuthoringInput Race.LawStatement) :=
  mapGuardedQueryInput fun declaration => { declaration with target := otherTargetId }

private def emptyQueryInput : Option (QueryAuthoringInput Race.LawStatement) :=
  mapGuardedQueryInput fun declaration => { declaration with form := .select [] }

private def missingCapabilityQueryInput : Option (QueryAuthoringInput Race.LawStatement) := do
  let input ← frontendGuardedQueryInput
  let property ← input.declaration.form.properties.head?
  pure { input with declaration := { input.declaration with
    form := .select [{ property with requires := [unknownActionId] }] } }

private def invalidLimitQueryInput : Option (QueryAuthoringInput Race.LawStatement) :=
  mapGuardedQueryInput fun declaration => { declaration with
    limits := QueryLimits.bounded 2 2 0 }

private def wrongUnitQueryInput : Option (QueryAuthoringInput Race.LawStatement) :=
  mapGuardedQueryInput fun declaration => { declaration with limits := {
    behavior := {
      transitions := { value := 2, unit := .selectedActions }
      selectedActions := { value := 2, unit := .selectedActions }
    }
    search := { value := 32, unit := .candidateEvaluations }
  } }

private def duplicatePropertyQueryInput : Option (QueryAuthoringInput Race.LawStatement) := do
  let input ← frontendGuardedQueryInput
  let property ← input.declaration.form.properties.head?
  pure { input with declaration := { input.declaration with form := .select [property, property] } }

/--
error: query authoring failed: {"error":{"kind":"target-mismatch","definitionId":"temporal.nexus2.cancellation-race.query.case-analysis","sourcePath":"Temporal/Feature/Nexus2/Race.lean","offendingValue":"temporal.nexus2.other.target != temporal.nexus2.cancellation-race.target","relatedDefinitionIds":["temporal.nexus2.cancellation-race.target","temporal.nexus2.other.target"]},"role":"target","anchor":{"sourcePath":"Temporal/Feature/Nexus2/AuthoringTests.lean","line":748,"column":15,"endLine":748,"endColumn":28}}
-/
#guard_msgs (error) in
#check query% wrongTargetQueryInput tracking [
  queryParent (Authoring.GuardedRace.family.id "query" "case-analysis"),
  targetAnchor otherTargetId]

/--
error: query authoring failed: {"error":{"kind":"missing-property","definitionId":"temporal.nexus2.cancellation-race.query.case-analysis","sourcePath":"Temporal/Feature/Nexus2/Race.lean","offendingValue":"properties","relatedDefinitionIds":[]},"role":"property","anchor":{"sourcePath":"Temporal/Feature/Nexus2/AuthoringTests.lean","line":756,"column":17,"endLine":756,"endColumn":69}}
-/
#guard_msgs (error) in
#check query% emptyQueryInput tracking [
  queryParent (Authoring.GuardedRace.family.id "query" "case-analysis"),
  propertyAnchor (Authoring.GuardedRace.family.id "property" "cases")]

/--
error: query authoring failed: {"error":{"kind":"missing-capability","definitionId":"temporal.nexus2.cancellation-race.query.case-analysis","sourcePath":"Temporal/Feature/Nexus2/Race.lean","offendingValue":"temporal.nexus2.unknown.action","relatedDefinitionIds":["temporal.nexus2.cancellation-race.target","temporal.nexus2.unknown.action"]},"role":"property","anchor":{"sourcePath":"Temporal/Feature/Nexus2/AuthoringTests.lean","line":764,"column":17,"endLine":764,"endColumn":69}}
-/
#guard_msgs (error) in
#check query% missingCapabilityQueryInput tracking [
  queryParent (Authoring.GuardedRace.family.id "query" "case-analysis"),
  propertyAnchor (Authoring.GuardedRace.family.id "property" "cases")]

/--
error: query authoring failed: {"error":{"kind":"invalid-limit","definitionId":"temporal.nexus2.cancellation-race.query.case-analysis","sourcePath":"Temporal/Feature/Nexus2/Race.lean","offendingValue":"search.candidateEvaluations=0","relatedDefinitionIds":[]},"role":"limits","anchor":{"sourcePath":"Temporal/Feature/Nexus2/AuthoringTests.lean","line":772,"column":15,"endLine":772,"endColumn":72}}
-/
#guard_msgs (error) in
#check query% invalidLimitQueryInput tracking [
  queryParent (Authoring.GuardedRace.family.id "query" "case-analysis"),
  limitsAnchor (Authoring.GuardedRace.family.id "query" "case-analysis")]

/--
error: query authoring failed: {"error":{"kind":"unit-mismatch","definitionId":"temporal.nexus2.cancellation-race.query.case-analysis","sourcePath":"Temporal/Feature/Nexus2/Race.lean","offendingValue":"behavior.transitions:selected-actions","relatedDefinitionIds":[]},"role":"limits","anchor":{"sourcePath":"Temporal/Feature/Nexus2/AuthoringTests.lean","line":780,"column":15,"endLine":780,"endColumn":72}}
-/
#guard_msgs (error) in
#check query% wrongUnitQueryInput tracking [
  queryParent (Authoring.GuardedRace.family.id "query" "case-analysis"),
  limitsAnchor (Authoring.GuardedRace.family.id "query" "case-analysis")]

/--
error: query authoring failed: {"error":{"kind":"duplicate-property","definitionId":"temporal.nexus2.cancellation-race.query.case-analysis","sourcePath":"Temporal/Feature/Nexus2/Race.lean","offendingValue":"temporal.nexus2.cancellation-race.property.cases","relatedDefinitionIds":["temporal.nexus2.cancellation-race.property.cases"]},"role":"property","anchor":{"sourcePath":"Temporal/Feature/Nexus2/AuthoringTests.lean","line":788,"column":17,"endLine":788,"endColumn":69}}
-/
#guard_msgs (error) in
#check query% duplicatePropertyQueryInput tracking [
  queryParent (Authoring.GuardedRace.family.id "query" "case-analysis"),
  propertyAnchor (Authoring.GuardedRace.family.id "property" "cases")]

/--
error: expected queryParent, targetAnchor, propertyAnchor, behaviorAnchor, limitsAnchor, or policyAnchor
-/
#guard_msgs (error) in
#check query% wrongTargetQueryInput tracking [
  unsupportedQueryRole (Authoring.GuardedRace.family.id "query" "case-analysis")]

private def unknownStateProperty : PropertySpec := {
  family := Authoring.GuardedRace.family
  key := "unknown-state"
  source := Race.source
  requires := [Race.capabilityId]
  clauses := [.transitionContract
    (Authoring.GuardedRace.family.id "property" "unknown-state.clause")
    (.selectedAction frontendRaceModel.resolveAction)
    (.resultingState { frontendRaceModel.cancelRequestedState with
      definitionId := DefinitionId.of "temporal.nexus2.unknown.state" })]
}

private def unknownResultProperty : PropertySpec := {
  family := Authoring.GuardedRace.family
  key := "unknown-result"
  source := Race.source
  requires := [Race.capabilityId]
  clauses := [.transitionContract
    (Authoring.GuardedRace.family.id "property" "unknown-result.clause")
    (.selectedAction frontendRaceModel.resolveAction)
    (.modelOutcome { frontendRaceModel.canceledOutcome with
      definitionId := DefinitionId.of "temporal.nexus2.unknown.result" })]
}

/--
error: property authoring failed: {"error":{"kind":"unknown-reference","definitionId":"temporal.nexus2.cancellation-race.property.unknown-state","sourcePath":"Temporal/Feature/Nexus2/Race.lean","offendingValue":"temporal.nexus2.unknown.state","relatedDefinitionIds":["temporal.nexus2.unknown.state"]},"role":"clause","anchor":{"sourcePath":"Temporal/Feature/Nexus2/AuthoringTests.lean","line":827,"column":15,"endLine":827,"endColumn":64}}
-/
#guard_msgs (error) in
#check property% unknownStateProperty against frontendRaceContext tracking [
  parentAnchor unknownStateProperty.declaration.id,
  clauseAnchor (DefinitionId.of "temporal.nexus2.unknown.state")]

/--
error: property authoring failed: {"error":{"kind":"unknown-reference","definitionId":"temporal.nexus2.cancellation-race.property.unknown-result","sourcePath":"Temporal/Feature/Nexus2/Race.lean","offendingValue":"temporal.nexus2.unknown.result","relatedDefinitionIds":["temporal.nexus2.unknown.result"]},"role":"clause","anchor":{"sourcePath":"Temporal/Feature/Nexus2/AuthoringTests.lean","line":835,"column":15,"endLine":835,"endColumn":65}}
-/
#guard_msgs (error) in
#check property% unknownResultProperty against frontendRaceContext tracking [
  parentAnchor unknownResultProperty.declaration.id,
  clauseAnchor (DefinitionId.of "temporal.nexus2.unknown.result")]

private def incompatibleStrategyQueryInput : Option (QueryAuthoringInput Race.LawStatement) := do
  let input ← frontendGuardedQueryInput
  let property ← input.declaration.form.properties.head?
  pure { input with declaration := { input.declaration with
    form := .verify property
    policy := .shortest
  } }

/--
error: query authoring failed: {"error":{"kind":"incompatible-strategy","definitionId":"temporal.nexus2.cancellation-race.query.case-analysis","sourcePath":"Temporal/Feature/Nexus2/Race.lean","offendingValue":"shortest","relatedDefinitionIds":[]},"role":"policy","anchor":{"sourcePath":"Temporal/Feature/Nexus2/AuthoringTests.lean","line":851,"column":15,"endLine":851,"endColumn":72}}
-/
#guard_msgs (error) in
#check query% incompatibleStrategyQueryInput tracking [
  queryParent (Authoring.GuardedRace.family.id "query" "case-analysis"),
  policyAnchor (Authoring.GuardedRace.family.id "query" "case-analysis")]

private def frontendContradictoryBehavior : Except BehaviorError CheckedBehavior :=
  behavior% contradictoryBehavior frontendRaceModel against frontendRaceBehaviorContext tracking [
    behaviorParent (contradictoryBehavior frontendRaceModel).declaration.id,
    setupAnchor (Authoring.GuardedRace.family.id "setup" "not-started")]

#guard frontendContradictoryBehavior.toOption.map CheckedBehavior.isUnsatisfiable == some true

private def invalidQueryHasNoCheckedValue : Option QueryErrorKind := do
  let input ← wrongTargetQueryInput
  match input.check with
  | .error error => some error.kind
  | .ok _ => none

#guard invalidQueryHasNoCheckedValue == some .targetMismatch

private def movedBehaviorSpec : ExactSequenceSpec := {
  Authoring.GuardedRace.behaviorSpec frontendRaceModel with
  source := { path := "Moved/Behavior.lean", line := 901, column := 4 }
}

private def movedBehaviorFrontend : Except BehaviorError CheckedBehavior :=
  behavior% movedBehaviorSpec against frontendRaceBehaviorContext tracking [
    behaviorParent movedBehaviorSpec.declaration.id]

private def behaviorSourceMetadataDifference : Option Bool := do
  let constructor ← (Authoring.GuardedRace.behaviorSpec frontendRaceModel).check
    frontendRaceBehaviorContext |>.toOption
  let moved ← movedBehaviorFrontend.toOption
  pure (moved.id == constructor.id && moved.source != constructor.source &&
    moved.canonicalMetadata != constructor.canonicalMetadata &&
    moved.behaviorFingerprint == constructor.behaviorFingerprint)

#guard behaviorSourceMetadataDifference == some true

private def movedQueryInput : Option (QueryAuthoringInput Race.LawStatement) := do
  let input ← frontendGuardedQueryInput
  pure { input with declaration := { input.declaration with
    source := { path := "Moved/Query.lean", line := 902, column := 5 }
  } }

private def movedQueryFrontend : Option (Except QueryError (CheckedQuery Race.LawStatement)) :=
  query% movedQueryInput tracking [
    queryParent (Authoring.GuardedRace.family.id "query" "case-analysis")]

private def querySourceMetadataDifference : Option Bool := do
  let input ← frontendGuardedQueryInput
  let constructor ← input.check.toOption
  let movedResult ← movedQueryFrontend
  let moved ← movedResult.toOption
  pure (moved.id == constructor.id && moved.source != constructor.source &&
    moved.canonicalMetadata == constructor.canonicalMetadata &&
    moved.behaviorFingerprint == constructor.behaviorFingerprint)

#guard querySourceMetadataDifference == some true

private def openBehaviorFrontend
    (spec : ExactSequenceSpec)
    (context : BehaviorCheckContext) : Except BehaviorError CheckedBehavior :=
  behavior% spec against context tracking [behaviorParent spec.declaration.id]

private def invalidOpenBehaviorHasNoCheckedValue : Option BehaviorErrorKind :=
  match openBehaviorFrontend malformedBehavior frontendRaceBehaviorContext with
  | .error error => some error.kind
  | .ok _ => none

#guard invalidOpenBehaviorHasNoCheckedValue == some .invalidDefinitionId

private def openQueryFrontend
    (spec : QuerySpec)
    (target : QueryModel Race.LawStatement) :
    Except QueryError (CheckedQuery Race.LawStatement) :=
  query% spec against target tracking [queryParent spec.declaration.id]

private def invalidOpenQueryHasNoCheckedValue : Option QueryErrorKind := do
  let target ← Race.targetResult.toOption
  let (property, behavior, _) ← guardedAdmission
  let spec := { Authoring.GuardedRace.querySpec property behavior with target := otherTargetId }
  match openQueryFrontend spec target with
  | .error error => some error.kind
  | .ok _ => none

#guard invalidOpenQueryHasNoCheckedValue == some .targetMismatch

/--
error: Tactic `decide` failed for proposition
-/
#guard_msgs (error, substring := true) in
example : frontendGuardedQueryInput.map
    (fun input => input.check.toOption.isSome) = some true := by
  decide +kernel

private def frontendCancelSpec : PropertySpec :=
  Authoring.Baseline.propertySpec "cancel" frontendBaselineModel.cancelAction
    frontendBaselineModel.canceledState frontendBaselineModel.canceledOutcome
    frontendBaselineModel.canceledFact

private def frontendSuccessSpec : PropertySpec :=
  Authoring.Baseline.propertySpec "success" frontendBaselineModel.reportSuccessAction
    frontendBaselineModel.succeededState frontendBaselineModel.succeededOutcome
    frontendBaselineModel.succeededFact

def frontendCancelProperty : Except PropertyError CheckedProperty :=
  property% frontendCancelSpec against frontendBaselineContext tracking [
    parentAnchor frontendCancelSpec.declaration.id]

def frontendSuccessProperty : Except PropertyError CheckedProperty :=
  property% frontendSuccessSpec against frontendBaselineContext tracking [
    parentAnchor frontendSuccessSpec.declaration.id]

private def frontendBaselineBehaviorContext : BehaviorCheckContext := {
  definitions := Lifecycle.definitions
}

private def frontendStartBehaviorSpec : ExactSequenceSpec :=
  Authoring.Baseline.behaviorSpec "start" "scheduled" "start"
    frontendBaselineModel.scheduledState frontendBaselineModel.startAction

private def frontendCancelBehaviorSpec : ExactSequenceSpec :=
  Authoring.Baseline.behaviorSpec "cancel" "started-cancel" "cancel"
    frontendBaselineModel.startedState frontendBaselineModel.cancelAction

private def frontendSuccessBehaviorSpec : ExactSequenceSpec :=
  Authoring.Baseline.behaviorSpec "success" "started-success" "success"
    frontendBaselineModel.startedState frontendBaselineModel.reportSuccessAction

def frontendStartBehavior : Except BehaviorError CheckedBehavior :=
  behavior% frontendStartBehaviorSpec against frontendBaselineBehaviorContext tracking [
    behaviorParent frontendStartBehaviorSpec.declaration.id,
    setupAnchor (Authoring.Baseline.family.id "setup" "scheduled"),
    occurrenceAnchor (Authoring.Baseline.family.id "occurrence" "start")]

def frontendCancelBehavior : Except BehaviorError CheckedBehavior :=
  behavior% frontendCancelBehaviorSpec against frontendBaselineBehaviorContext tracking [
    behaviorParent frontendCancelBehaviorSpec.declaration.id,
    setupAnchor (Authoring.Baseline.family.id "setup" "started-cancel"),
    occurrenceAnchor (Authoring.Baseline.family.id "occurrence" "cancel")]

def frontendSuccessBehavior : Except BehaviorError CheckedBehavior :=
  behavior% frontendSuccessBehaviorSpec against frontendBaselineBehaviorContext tracking [
    behaviorParent frontendSuccessBehaviorSpec.declaration.id,
    setupAnchor (Authoring.Baseline.family.id "setup" "started-success"),
    occurrenceAnchor (Authoring.Baseline.family.id "occurrence" "success")]

private structure FrontendBaseline where
  startProperty : CheckedProperty
  startBehavior : CheckedBehavior
  startQuery : CheckedQuery Lifecycle.LawStatement
  cancelProperty : CheckedProperty
  cancelBehavior : CheckedBehavior
  cancelQuery : CheckedQuery Lifecycle.LawStatement
  successProperty : CheckedProperty
  successBehavior : CheckedBehavior
  successQuery : CheckedQuery Lifecycle.LawStatement

private def frontendBaselineAdmission : Option FrontendBaseline := do
  let constructor ← Authoring.Baseline.checkBaseline.toOption
  let startProperty ← frontendStartProperty.toOption
  let startBehavior ← frontendStartBehavior.toOption
  let startSpec := Authoring.Baseline.querySpec "start" startProperty startBehavior
  let startQuery ← (query% startSpec against constructor.target tracking [
    queryParent startSpec.declaration.id,
    targetAnchor startSpec.target,
    propertyAnchor startProperty.id,
    behaviorAnchor startBehavior.id,
    limitsAnchor startSpec.declaration.id]) |>.toOption
  let cancelProperty ← frontendCancelProperty.toOption
  let cancelBehavior ← frontendCancelBehavior.toOption
  let cancelSpec := Authoring.Baseline.querySpec "cancel" cancelProperty cancelBehavior
  let cancelQuery ← (query% cancelSpec against constructor.target tracking [
    queryParent cancelSpec.declaration.id,
    targetAnchor cancelSpec.target,
    propertyAnchor cancelProperty.id,
    behaviorAnchor cancelBehavior.id,
    limitsAnchor cancelSpec.declaration.id]) |>.toOption
  let successProperty ← frontendSuccessProperty.toOption
  let successBehavior ← frontendSuccessBehavior.toOption
  let successSpec := Authoring.Baseline.querySpec "success" successProperty successBehavior
  let successQuery ← (query% successSpec against constructor.target tracking [
    queryParent successSpec.declaration.id,
    targetAnchor successSpec.target,
    propertyAnchor successProperty.id,
    behaviorAnchor successBehavior.id,
    limitsAnchor successSpec.declaration.id]) |>.toOption
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
  let startKernel ← IncrementalPlannerKernel.ofCheckedQuery constructor.target.id
    frontend.startQuery |>.toOption
  let cancelKernel ← IncrementalPlannerKernel.ofCheckedQuery constructor.target.id
    frontend.cancelQuery |>.toOption
  let successKernel ← IncrementalPlannerKernel.ofCheckedQuery constructor.target.id
    frontend.successQuery |>.toOption
  let startRun ← (plan frontend.startQuery startKernel).toOption
  let cancelRun ← (plan frontend.cancelQuery cancelKernel).toOption
  let successRun ← (plan frontend.successQuery successKernel).toOption; pure (startRun.result.outcome, cancelRun.result.outcome, successRun.result.outcome)

private def constructorBaselineTraces : Option
    (PlanningOutcome × PlanningOutcome × PlanningOutcome) := do
  let constructor ← Authoring.Baseline.checkBaseline.toOption
  let startKernel ← IncrementalPlannerKernel.ofCheckedQuery constructor.target.id
    constructor.start.query |>.toOption
  let cancelKernel ← IncrementalPlannerKernel.ofCheckedQuery constructor.target.id
    constructor.cancel.query |>.toOption
  let successKernel ← IncrementalPlannerKernel.ofCheckedQuery constructor.target.id
    constructor.success.query |>.toOption
  let startRun ← (plan constructor.start.query startKernel).toOption
  let cancelRun ← (plan constructor.cancel.query cancelKernel).toOption
  let successRun ← (plan constructor.success.query successKernel).toOption; pure (startRun.result.outcome, cancelRun.result.outcome, successRun.result.outcome)

#guard frontendBaselineTraces == constructorBaselineTraces

/--
error: Tactic `decide` failed
-/
#guard_msgs (error, substring := true) in
example : ((Authoring.GuardedRace.behaviorSpec frontendRaceModel).check
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
    parentAnchor frontendStartSpec.declaration.id]

def frontendTenProperties : List (Except PropertyError CheckedProperty) := [
  property% frontendStartSpec against frontendBaselineContext tracking [parentAnchor frontendStartSpec.declaration.id],
  property% frontendStartSpec against frontendBaselineContext tracking [parentAnchor frontendStartSpec.declaration.id],
  property% frontendStartSpec against frontendBaselineContext tracking [parentAnchor frontendStartSpec.declaration.id],
  property% frontendStartSpec against frontendBaselineContext tracking [parentAnchor frontendStartSpec.declaration.id],
  property% frontendStartSpec against frontendBaselineContext tracking [parentAnchor frontendStartSpec.declaration.id],
  property% frontendStartSpec against frontendBaselineContext tracking [parentAnchor frontendStartSpec.declaration.id],
  property% frontendStartSpec against frontendBaselineContext tracking [parentAnchor frontendStartSpec.declaration.id],
  property% frontendStartSpec against frontendBaselineContext tracking [parentAnchor frontendStartSpec.declaration.id],
  property% frontendStartSpec against frontendBaselineContext tracking [parentAnchor frontendStartSpec.declaration.id],
  property% frontendStartSpec against frontendBaselineContext tracking [parentAnchor frontendStartSpec.declaration.id]
]

#guard frontendTenProperties.all fun result =>
  result.toOption.map (fun property => property.behaviorFingerprint) ==
    frontendStartProperty.toOption.map (fun property => property.behaviorFingerprint)

#guard constructorOneProperty.toOption.isSome
#guard constructorTenProperties.all (fun result => result.toOption.isSome)
#guard frontendOneProperty.toOption.isSome
#guard frontendTenProperties.all (fun result => result.toOption.isSome)

def constructorOneBehavior : Except BehaviorError CheckedBehavior :=
  frontendStartBehaviorSpec.check frontendBaselineBehaviorContext

def constructorTenBehaviors : List (Except BehaviorError CheckedBehavior) := [
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

def frontendOneBehavior : Except BehaviorError CheckedBehavior :=
  behavior% frontendStartBehaviorSpec against frontendBaselineBehaviorContext tracking [
    behaviorParent frontendStartBehaviorSpec.declaration.id,
    setupAnchor (Authoring.Baseline.family.id "setup" "scheduled"),
    occurrenceAnchor (Authoring.Baseline.family.id "occurrence" "start")]

def frontendTenBehaviors : List (Except BehaviorError CheckedBehavior) := [
  behavior% frontendStartBehaviorSpec against frontendBaselineBehaviorContext tracking [behaviorParent frontendStartBehaviorSpec.declaration.id],
  behavior% frontendStartBehaviorSpec against frontendBaselineBehaviorContext tracking [behaviorParent frontendStartBehaviorSpec.declaration.id],
  behavior% frontendStartBehaviorSpec against frontendBaselineBehaviorContext tracking [behaviorParent frontendStartBehaviorSpec.declaration.id],
  behavior% frontendStartBehaviorSpec against frontendBaselineBehaviorContext tracking [behaviorParent frontendStartBehaviorSpec.declaration.id],
  behavior% frontendStartBehaviorSpec against frontendBaselineBehaviorContext tracking [behaviorParent frontendStartBehaviorSpec.declaration.id],
  behavior% frontendStartBehaviorSpec against frontendBaselineBehaviorContext tracking [behaviorParent frontendStartBehaviorSpec.declaration.id],
  behavior% frontendStartBehaviorSpec against frontendBaselineBehaviorContext tracking [behaviorParent frontendStartBehaviorSpec.declaration.id],
  behavior% frontendStartBehaviorSpec against frontendBaselineBehaviorContext tracking [behaviorParent frontendStartBehaviorSpec.declaration.id],
  behavior% frontendStartBehaviorSpec against frontendBaselineBehaviorContext tracking [behaviorParent frontendStartBehaviorSpec.declaration.id],
  behavior% frontendStartBehaviorSpec against frontendBaselineBehaviorContext tracking [behaviorParent frontendStartBehaviorSpec.declaration.id]]

#guard constructorOneBehavior.toOption.isSome
#guard constructorTenBehaviors.all (fun result => result.toOption.isSome)
#guard frontendOneBehavior.toOption.isSome
#guard frontendTenBehaviors.all (fun result => result.toOption.isSome)

def constructorOneQuery : Option (Except QueryError (CheckedQuery Race.LawStatement)) :=
  frontendGuardedQueryInput.map QueryAuthoringInput.check

def constructorTenQueries : List (Option (Except QueryError (CheckedQuery Race.LawStatement))) := [
  frontendGuardedQueryInput.map QueryAuthoringInput.check,
  frontendGuardedQueryInput.map QueryAuthoringInput.check,
  frontendGuardedQueryInput.map QueryAuthoringInput.check,
  frontendGuardedQueryInput.map QueryAuthoringInput.check,
  frontendGuardedQueryInput.map QueryAuthoringInput.check,
  frontendGuardedQueryInput.map QueryAuthoringInput.check,
  frontendGuardedQueryInput.map QueryAuthoringInput.check,
  frontendGuardedQueryInput.map QueryAuthoringInput.check,
  frontendGuardedQueryInput.map QueryAuthoringInput.check,
  frontendGuardedQueryInput.map QueryAuthoringInput.check]

def frontendOneQuery : Option (Except QueryError (CheckedQuery Race.LawStatement)) :=
  query% frontendGuardedQueryInput tracking [
    queryParent (Authoring.GuardedRace.family.id "query" "case-analysis"),
    targetAnchor Race.targetId,
    propertyAnchor (Authoring.GuardedRace.family.id "property" "cases"),
    behaviorAnchor (Authoring.GuardedRace.family.id "behavior" "request-then-resolve"),
    limitsAnchor (Authoring.GuardedRace.family.id "query" "case-analysis")]

def frontendTenQueries : List (Option (Except QueryError (CheckedQuery Race.LawStatement))) := [
  query% frontendGuardedQueryInput tracking [queryParent (Authoring.GuardedRace.family.id "query" "case-analysis")],
  query% frontendGuardedQueryInput tracking [queryParent (Authoring.GuardedRace.family.id "query" "case-analysis")],
  query% frontendGuardedQueryInput tracking [queryParent (Authoring.GuardedRace.family.id "query" "case-analysis")],
  query% frontendGuardedQueryInput tracking [queryParent (Authoring.GuardedRace.family.id "query" "case-analysis")],
  query% frontendGuardedQueryInput tracking [queryParent (Authoring.GuardedRace.family.id "query" "case-analysis")],
  query% frontendGuardedQueryInput tracking [queryParent (Authoring.GuardedRace.family.id "query" "case-analysis")],
  query% frontendGuardedQueryInput tracking [queryParent (Authoring.GuardedRace.family.id "query" "case-analysis")],
  query% frontendGuardedQueryInput tracking [queryParent (Authoring.GuardedRace.family.id "query" "case-analysis")],
  query% frontendGuardedQueryInput tracking [queryParent (Authoring.GuardedRace.family.id "query" "case-analysis")],
  query% frontendGuardedQueryInput tracking [queryParent (Authoring.GuardedRace.family.id "query" "case-analysis")]]

private def queryAdmissionSucceeded
    (result : Option (Except QueryError (CheckedQuery Race.LawStatement))) : Bool :=
  result.bind Except.toOption |>.isSome

#guard queryAdmissionSucceeded constructorOneQuery
#guard constructorTenQueries.all queryAdmissionSucceeded
#guard queryAdmissionSucceeded frontendOneQuery
#guard frontendTenQueries.all queryAdmissionSucceeded

private def minimalPropertySpec : PropertySpec := {
  family := { root := DefinitionId.of "test.frontend" }
  key := "minimal"
  source := Race.source
  requires := []
  clauses := []
}

/--
error: Tactic `decide` failed
-/
#guard_msgs (error, substring := true) in
example : (PropertySpec.check {
    family := { root := DefinitionId.of "test.frontend" }
    key := "minimal"
    source := Race.source
    requires := []
    clauses := []
  } {
    definitions := [], providers := [], meanings := [] }).toOption.isSome = true := by
  decide +kernel

#print axioms Authoring.Baseline.checkBaseline
#print axioms Authoring.GuardedRace.propertySpec
#print axioms Authoring.GuardedRace.behaviorSpec
#print axioms Authoring.GuardedRace.querySpec
#print axioms PropertySpec.check
#print axioms PropertySpec.checked
#print axioms ExactSequenceSpec.check
#print axioms ExactSequenceSpec.checked
#print axioms QuerySpec.check
#print axioms QuerySpec.checked
#print axioms frontendGuardedBehavior
#print axioms frontendGuardedAdmission
#print axioms frontendBaselineAdmission
#print axioms frontendOneQuery
#print axioms Race.targetResult

end Temporal.Feature.Nexus2.AuthoringTests
