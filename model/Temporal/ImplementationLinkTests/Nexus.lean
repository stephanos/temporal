import Temporal.Feature.Nexus.Operations
import Temporal.System.Nexus.ImplementationLink
import Temporal.System.Nexus.ObservationFaultTests

/-!
Composed checks for the ordinary Nexus lifecycle. Synthetic Evidence exists only to establish the
already-accepted System trace at the test boundary; the production operation consumes the typed
Observation result and never interprets raw Evidence.
-/

namespace Temporal.ImplementationLinkTests.Nexus

open Umpire
open Temporal.System.Nexus.ImplementationLink

example : checkedResult.isOk = true ∧ checked.hasCanonicalIdentity = true := by
  native_decide

example : checked.sourceTarget.id = Temporal.System.Nexus.targetId ∧
    checked.destinationTarget.id = Temporal.Feature.Nexus.Lifecycle.targetId ∧
    checked.declaration.capabilityMappings = [lifecycleCapabilityMapping] ∧
    checked.declaration.relationMappings = [] := by
  native_decide

theorem checked_link_retains_migrated_target_identity_and_fingerprints :
    checked.sourceTarget.id = Temporal.System.Nexus.target.id ∧
    checked.sourceTarget.source = Temporal.System.Nexus.target.source ∧
    checked.sourceTarget.behaviorFingerprint =
      Temporal.System.Nexus.target.behaviorFingerprint ∧
    checked.sourceTarget.behaviorFingerprint.render =
      "sha256:54d0b7ed28698c0db28e7c9de00f3c0c0998db889a50de917d100173b37cf374" ∧
    checked.destinationTarget.id = Temporal.Feature.Nexus.Lifecycle.target.id ∧
    checked.destinationTarget.source = Temporal.Feature.Nexus.Lifecycle.target.source ∧
    checked.destinationTarget.behaviorFingerprint =
      Temporal.Feature.Nexus.Lifecycle.target.behaviorFingerprint ∧
    checked.destinationTarget.behaviorFingerprint.render =
      "sha256:2dffda3904f7425aa7ef89876393dc1648edcca0a944139672b6e35dd1651d93" := by
  native_decide

theorem migrated_targets_keep_their_named_authority_seams :
    Temporal.System.Nexus.target.kernel.authoritativeInitial
      Temporal.System.Nexus.queuedSetup Temporal.System.Nexus.queuedState ∧
    Temporal.System.Nexus.target.kernel.authoritativeStep
      Temporal.System.Nexus.queuedState Temporal.System.Nexus.dispatchAction
      Temporal.System.Nexus.dispatchedResult ∧
    Temporal.Feature.Nexus.Lifecycle.target.kernel.authoritativeInitial
      Temporal.Feature.Nexus.Lifecycle.scheduledSetup
      Temporal.Feature.Nexus.Lifecycle.scheduledState ∧
    Temporal.Feature.Nexus.Lifecycle.target.kernel.authoritativeStep
      Temporal.Feature.Nexus.Lifecycle.scheduledState
      Temporal.Feature.Nexus.Lifecycle.startAction
      Temporal.Feature.Nexus.Lifecycle.startedResult ∧
    Temporal.Feature.Nexus.Lifecycle.target.kernel.authoritativeStep
      Temporal.Feature.Nexus.Lifecycle.startedState
      Temporal.Feature.Nexus.Lifecycle.cancelAction
      Temporal.Feature.Nexus.Lifecycle.canceledResult ∧
    Temporal.Feature.Nexus.Lifecycle.target.kernel.authoritativeStep
      Temporal.Feature.Nexus.Lifecycle.startedState
      Temporal.Feature.Nexus.Lifecycle.reportSuccessAction
      Temporal.Feature.Nexus.Lifecycle.succeededResult := by
  exact ⟨Temporal.System.Nexus.target_queued_initial_authoritative,
    Temporal.System.Nexus.target_queued_dispatch_authoritative,
    Temporal.Feature.Nexus.Lifecycle.target_scheduled_initial_authoritative,
    Temporal.Feature.Nexus.Lifecycle.target_scheduled_start_authoritative,
    Temporal.Feature.Nexus.Lifecycle.target_started_cancel_authoritative,
    Temporal.Feature.Nexus.Lifecycle.target_started_reportSuccess_authoritative⟩

example : Temporal.Feature.Nexus.Lifecycle.target.kernel.authoritativeInitial
    Temporal.Feature.Nexus.Lifecycle.scheduledSetup
    Temporal.Feature.Nexus.Lifecycle.scheduledState := by
  simpa [witness] using witness.initialForward Temporal.System.Nexus.queuedSetup
    Temporal.System.Nexus.queuedState
    Temporal.System.Nexus.target_queued_initial_authoritative

example : Temporal.Feature.Nexus.Lifecycle.target.kernel.authoritativeStep
    Temporal.Feature.Nexus.Lifecycle.startedState
    Temporal.Feature.Nexus.Lifecycle.cancelAction
    Temporal.Feature.Nexus.Lifecycle.canceledResult := by
  simpa [witness, Temporal.System.Nexus.cancellationRecordedResult,
    Temporal.Feature.Nexus.Lifecycle.canceledResult] using
    witness.stepForward Temporal.System.Nexus.runningState
    Temporal.System.Nexus.recordCancellationAction
    Temporal.System.Nexus.cancellationRecordedResult
    Temporal.System.Nexus.target_running_cancellation_authoritative

private def id (value : String) : DefinitionId := DefinitionId.of value

def source : SourceLocation := {
  path := "Temporal/ImplementationLinkTests/Nexus.lean"
  line := 1
  column := 1
  provenance := "lean-test"
}

def profileId : DefinitionId := id "temporal.test.nexus.system.profile"
def evidenceKind : DefinitionId := id "temporal.test.nexus.system.evidence.lifecycle"
def phaseField : DefinitionId := id "temporal.test.nexus.system.field.phase"
def stateField : DefinitionId := id "temporal.test.nexus.system.field.state"
def actionField : DefinitionId := id "temporal.test.nexus.system.field.action"
def outcomeField : DefinitionId := id "temporal.test.nexus.system.field.outcome"
def observationField : DefinitionId := id "temporal.test.nexus.system.field.observation"

def evidenceProfile : EvidenceProfileDeclaration := {
  id := profileId
  source
  kinds := [{
    id := evidenceKind
    fields := [
      { id := phaseField, valueType := .text },
      { id := stateField, valueType := .text },
      { id := actionField, valueType := .text },
      { id := outcomeField, valueType := .text },
      { id := observationField, valueType := .text }
    ]
  }]
}

private def field (fieldId : DefinitionId) : ObservationExpression :=
  .field { kind := evidenceKind, field := fieldId }

private def stepCondition : ObservationExpressionAuthoring :=
  .portable (.equals (field phaseField) (.text "step"))

def stateRuleId : DefinitionId := id "temporal.test.nexus.system.rule.state"
def startMappingId : DefinitionId := id "temporal.test.nexus.system.observation.start"
def cancellationMappingId : DefinitionId :=
  id "temporal.test.nexus.system.observation.cancellation"
def successfulCompletionMappingId : DefinitionId :=
  id "temporal.test.nexus.system.observation.successful-completion"
def dispatchRuleId : DefinitionId := id "temporal.test.nexus.system.rule.action.dispatch"
def cancellationRuleId : DefinitionId :=
  id "temporal.test.nexus.system.rule.action.record-cancellation"
def completionRuleId : DefinitionId :=
  id "temporal.test.nexus.system.rule.action.record-completion"
def outcomeRuleId : DefinitionId := id "temporal.test.nexus.system.rule.outcome"
def observationRuleId : DefinitionId := id "temporal.test.nexus.system.rule.observation"

def observationDeclaration
    (mappingId actionRuleId actionDefinitionId : DefinitionId) : ObservationMappingDeclaration := {
  id := mappingId
  source
  profile := profileId
  rules := [
    {
      id := stateRuleId
      output := Temporal.System.Nexus.operationStateId
      outputKind := .state
      value := .portable (field stateField)
    },
    {
      id := actionRuleId
      output := actionDefinitionId
      outputKind := .action
      value := .portable (field actionField)
      condition := some stepCondition
    },
    {
      id := outcomeRuleId
      output := Temporal.System.Nexus.transitionOutcomeId
      outputKind := .outcome
      value := .portable (field outcomeField)
      condition := some stepCondition
    },
    {
      id := observationRuleId
      output := Temporal.System.Nexus.lifecycleObservationId
      outputKind := .observation
      value := .portable (field observationField)
      condition := some stepCondition
    }
  ]
  ordering := [
    { before := actionRuleId, after := outcomeRuleId },
    { before := outcomeRuleId, after := stateRuleId },
    { before := stateRuleId, after := observationRuleId }
  ]
  closures := [{ kind := evidenceKind }]
  dispositions := [
    { field := { kind := evidenceKind, field := phaseField }, disposition := .retain },
    { field := { kind := evidenceKind, field := stateField }, disposition := .retain },
    { field := { kind := evidenceKind, field := actionField }, disposition := .retain },
    { field := { kind := evidenceKind, field := outcomeField }, disposition := .retain },
    { field := { kind := evidenceKind, field := observationField }, disposition := .retain }
  ]
  evidenceBound := { value := 2, unit := .evidenceRecords }
}

def observationContext : ObservationCheckContext :=
  ObservationCheckContext.ofTarget Temporal.System.Nexus.target [evidenceProfile]

def startPlanResult : Except ObservationError CheckedObservationPlan :=
  checkObservation observationContext <|
    observationDeclaration startMappingId dispatchRuleId Temporal.System.Nexus.dispatchActionId

private theorem startPlanResult_isSome : startPlanResult.toOption.isSome = true := by
  native_decide

def startPlan : CheckedObservationPlan :=
  startPlanResult.toOption.get startPlanResult_isSome

def cancellationPlanResult : Except ObservationError CheckedObservationPlan :=
  checkObservation observationContext <| observationDeclaration cancellationMappingId
    cancellationRuleId Temporal.System.Nexus.recordCancellationActionId

private theorem cancellationPlanResult_isSome : cancellationPlanResult.toOption.isSome = true := by
  native_decide

def cancellationPlan : CheckedObservationPlan :=
  cancellationPlanResult.toOption.get cancellationPlanResult_isSome

def successfulCompletionPlanResult : Except ObservationError CheckedObservationPlan :=
  checkObservation observationContext <| observationDeclaration successfulCompletionMappingId
    completionRuleId Temporal.System.Nexus.recordCompletionActionId

private theorem successfulCompletionPlanResult_isSome :
    successfulCompletionPlanResult.toOption.isSome = true := by
  native_decide

def successfulCompletionPlan : CheckedObservationPlan :=
  successfulCompletionPlanResult.toOption.get successfulCompletionPlanResult_isSome

private def textField (fieldId : DefinitionId) (value : String) : EvidenceFieldValue := {
  field := fieldId
  value := .text value
}

private def initialRecord
    (recordId : DefinitionId)
    (state : ModelValue) : SyntheticEvidenceRecord := {
  id := recordId
  profile := profileId
  profileVersion := 1
  kind := evidenceKind
  sequence := 1
  fields := [textField phaseField "initial", textField stateField state.value]
}

private def stepRecord
    (recordId parentId : DefinitionId)
    (action outcome resultingState observation : ModelValue) : SyntheticEvidenceRecord := {
  id := recordId
  profile := profileId
  profileVersion := 1
  kind := evidenceKind
  sequence := 2
  causalParents := [parentId]
  fields := [
    textField phaseField "step",
    textField stateField resultingState.value,
    textField actionField action.value,
    textField outcomeField outcome.value,
    textField observationField observation.value
  ]
}

private def oneStepEvidence
    (initialId stepId : DefinitionId)
    (initialState action outcome resultingState observation : ModelValue) : EvidenceBundle := {
  profile := profileId
  profileVersion := 1
  records := [
    stepRecord stepId initialId action outcome resultingState observation,
    initialRecord initialId initialState
  ]
  closures := [{ kind := evidenceKind, lastSequence := 2 }]
}

def startEvidence : EvidenceBundle := oneStepEvidence
  (id "temporal.test.nexus.system.start.initial")
  (id "temporal.test.nexus.system.start.step")
  Temporal.System.Nexus.queuedState
  Temporal.System.Nexus.dispatchAction
  Temporal.System.Nexus.dispatchedOutcome
  Temporal.System.Nexus.runningState
  Temporal.System.Nexus.runningObservation

def cancellationEvidence : EvidenceBundle := oneStepEvidence
  (id "temporal.test.nexus.system.cancellation.initial")
  (id "temporal.test.nexus.system.cancellation.step")
  Temporal.System.Nexus.runningState
  Temporal.System.Nexus.recordCancellationAction
  Temporal.System.Nexus.cancellationRecordedOutcome
  Temporal.System.Nexus.cancellationRecordedState
  Temporal.System.Nexus.cancellationRecordedObservation

def successfulCompletionEvidence : EvidenceBundle := oneStepEvidence
  (id "temporal.test.nexus.system.successful-completion.initial")
  (id "temporal.test.nexus.system.successful-completion.step")
  Temporal.System.Nexus.runningState
  Temporal.System.Nexus.recordCompletionAction
  Temporal.System.Nexus.completionRecordedOutcome
  Temporal.System.Nexus.completionRecordedState
  Temporal.System.Nexus.completionRecordedObservation

def startObservation : ObservationResult := evaluateEvidence startPlan startEvidence
def cancellationObservation : ObservationResult :=
  evaluateEvidence cancellationPlan cancellationEvidence
def successfulCompletionObservation : ObservationResult :=
  evaluateEvidence successfulCompletionPlan successfulCompletionEvidence

def startResult : FeaturePropertyResult := evaluateFeatureProperty
  Temporal.System.Nexus.queuedSetup
  Temporal.Feature.Nexus.Operations.AsyncStart.property
  startObservation

def cancellationResult : FeaturePropertyResult := evaluateFeatureProperty
  Temporal.System.Nexus.runningSetup
  Temporal.Feature.Nexus.Operations.Cancellation.property
  cancellationObservation

def successfulCompletionResult : FeaturePropertyResult := evaluateFeatureProperty
  Temporal.System.Nexus.runningSetup
  Temporal.Feature.Nexus.Operations.SuccessfulCompletion.property
  successfulCompletionObservation

private def applicationShape
    (result : FeaturePropertyResult) : Option
      (Temporal.System.Nexus.ExecutionSetup × List RoleBinding ×
        ModelTrace ModelValue ModelValue ModelValue ModelValue ×
        List ModelCoordinate × Bool) :=
  result.evaluated?.map fun evaluated =>
    let application := evaluated.application
    (application.sourceSetup,
      application.destinationSetup,
      application.trace,
      application.evidenceLinks.map ImplementationLinkEvidenceLink.coordinate,
      application.evidenceLinks.all fun evidenceLink =>
        evidenceLink.implementationLinkId == implementationLinkId &&
          evidenceLink.implementationLinkBehaviorFingerprint == checked.behaviorFingerprint &&
          evidenceLink.sourceTarget == .ofTarget Temporal.System.Nexus.target &&
          evidenceLink.destinationTarget ==
            .ofTarget Temporal.Feature.Nexus.Lifecycle.target &&
          evidenceLink.identity != behaviorFingerprintOf "")

private def expectedCoordinates : List ModelCoordinate := [
  .initialState,
  .selectedAction 1,
  .modelOutcome 1,
  .resultingState 1,
  .observation 1 1
]

/-- Start, cancel, and successful completion translate completely with positional Evidence Links. -/
example : ([
    applicationShape startResult,
    applicationShape cancellationResult,
    applicationShape successfulCompletionResult
  ] == [
    some (Temporal.System.Nexus.queuedSetup,
      Temporal.Feature.Nexus.Lifecycle.scheduledSetup,
      Temporal.Feature.Nexus.Operations.AsyncStart.intendedTrace.trace,
      expectedCoordinates, true),
    some (Temporal.System.Nexus.runningSetup,
      Temporal.Feature.Nexus.Lifecycle.startedSetup,
      Temporal.Feature.Nexus.Operations.Cancellation.intendedTrace.trace,
      expectedCoordinates, true),
    some (Temporal.System.Nexus.runningSetup,
      Temporal.Feature.Nexus.Lifecycle.startedSetup,
      Temporal.Feature.Nexus.Operations.SuccessfulCompletion.intendedTrace.trace,
      expectedCoordinates, true)
  ]) = true := by
  native_decide

/-- Composition invokes the unchanged Feature evaluator only after successful translation. -/
example : [
    startResult.evaluated?.map EvaluatedFeatureProperty.evaluation,
    cancellationResult.evaluated?.map EvaluatedFeatureProperty.evaluation,
    successfulCompletionResult.evaluated?.map EvaluatedFeatureProperty.evaluation
  ] = [
    some (evaluateProperty Temporal.Feature.Nexus.Operations.AsyncStart.property
      Temporal.Feature.Nexus.Operations.AsyncStart.intendedTrace.trace),
    some (evaluateProperty Temporal.Feature.Nexus.Operations.Cancellation.property
      Temporal.Feature.Nexus.Operations.Cancellation.intendedTrace.trace),
    some (evaluateProperty Temporal.Feature.Nexus.Operations.SuccessfulCompletion.property
      Temporal.Feature.Nexus.Operations.SuccessfulCompletion.intendedTrace.trace)
  ] ∧ [
    startResult.evaluated?.map (fun result => result.evaluation.satisfied),
    cancellationResult.evaluated?.map (fun result => result.evaluation.satisfied),
    successfulCompletionResult.evaluated?.map (fun result => result.evaluation.satisfied)
  ] = [some true, some true, some true] := by
  native_decide

def missingClosureObservation : ObservationResult :=
  evaluateEvidence startPlan { startEvidence with closures := [] }

def observationFailureResult : FeaturePropertyResult := evaluateFeatureProperty
  Temporal.System.Nexus.queuedSetup
  Temporal.Feature.Nexus.Operations.AsyncStart.property
  missingClosureObservation

def wrongSetupResult : FeaturePropertyResult := evaluateFeatureProperty
  Temporal.System.Nexus.runningSetup
  Temporal.Feature.Nexus.Operations.AsyncStart.property
  startObservation

def impossibleTransitionEvidence : EvidenceBundle := oneStepEvidence
  (id "temporal.test.nexus.system.impossible.initial")
  (id "temporal.test.nexus.system.impossible.step")
  Temporal.System.Nexus.queuedState
  Temporal.System.Nexus.recordCompletionAction
  Temporal.System.Nexus.completionRecordedOutcome
  Temporal.System.Nexus.completionRecordedState
  Temporal.System.Nexus.completionRecordedObservation

def impossibleTransitionResult : FeaturePropertyResult := evaluateFeatureProperty
  Temporal.System.Nexus.queuedSetup
  Temporal.Feature.Nexus.Operations.SuccessfulCompletion.property
  (evaluateEvidence successfulCompletionPlan impossibleTransitionEvidence)

private def acceptedTrace? : ObservationResult → Option EvidenceBackedTrace
  | .accepted trace => some trace
  | _ => none

def startTrace : EvidenceBackedTrace :=
  (acceptedTrace? startObservation).get (by native_decide)

def missingCoordinateObservation : ObservationResult := .accepted {
  startTrace with evidenceLinks := startTrace.evidenceLinks.tail
}

def missingCoordinateResult : FeaturePropertyResult := evaluateFeatureProperty
  Temporal.System.Nexus.queuedSetup
  Temporal.Feature.Nexus.Operations.AsyncStart.property
  missingCoordinateObservation

private def driftMeaning (meaning : MeaningProvision) : MeaningProvision :=
  if meaning.definitionId == Temporal.System.Nexus.operationStateId then
    { meaning with canonicalBehavior := "temporal-system-nexus-lifecycle-state/drift" }
  else
    meaning

def driftContext : ObservationCheckContext := {
  observationContext with meanings := observationContext.meanings.map driftMeaning
}

def driftPlanResult : Except ObservationError CheckedObservationPlan :=
  checkObservation driftContext <|
    observationDeclaration startMappingId dispatchRuleId Temporal.System.Nexus.dispatchActionId

private theorem driftPlanResult_isSome : driftPlanResult.toOption.isSome = true := by
  native_decide

def driftPlan : CheckedObservationPlan :=
  driftPlanResult.toOption.get driftPlanResult_isSome

def behaviorFingerprintDriftResult : FeaturePropertyResult := evaluateFeatureProperty
  Temporal.System.Nexus.queuedSetup
  Temporal.Feature.Nexus.Operations.AsyncStart.property
  (evaluateEvidence driftPlan startEvidence)

def mutatedPropertyId : DefinitionId :=
  id "temporal.test.nexus.feature.property.mutated-start"

def mutatedPropertyDeclaration : PropertyDeclaration := {
  id := mutatedPropertyId
  source := Temporal.Feature.Nexus.Operations.source
  requires := [Temporal.Feature.Nexus.Lifecycle.lifecycleCapabilityId]
  clauses := [
    .transitionContract (id "temporal.test.nexus.feature.property.mutated-start.state")
      { field := .selectedAction,
        reference := Temporal.Feature.Nexus.Lifecycle.startActionId,
        constraint := .equals Temporal.Feature.Nexus.Lifecycle.startAction.value }
      { field := .resultingState,
        reference := Temporal.Feature.Nexus.Lifecycle.operationStateId,
        constraint := .equals Temporal.Feature.Nexus.Lifecycle.succeededState.value }
  ]
}

def mutatedPropertyResult : Except PropertyError CheckedProperty :=
  checkProperty (PropertyCheckContext.ofTarget Temporal.Feature.Nexus.Lifecycle.target)
    (.portable mutatedPropertyDeclaration)

private theorem mutatedPropertyResult_isSome : mutatedPropertyResult.toOption.isSome = true := by
  native_decide

def mutatedProperty : CheckedProperty :=
  mutatedPropertyResult.toOption.get mutatedPropertyResult_isSome

def propertyFailureResult : FeaturePropertyResult := evaluateFeatureProperty
  Temporal.System.Nexus.queuedSetup mutatedProperty startObservation

/-- Each independent mutation stops at its responsible semantic layer with its exact kind. -/
example : [
    observationFailureResult.layer,
    wrongSetupResult.layer,
    impossibleTransitionResult.layer,
    missingCoordinateResult.layer,
    behaviorFingerprintDriftResult.layer,
    propertyFailureResult.layer
  ] = [
    .observation,
    .implementationLink,
    .implementationLink,
    .implementationLink,
    .implementationLink,
    .property
  ] ∧
  observationFailureResult.observationDiagnostic?.map ObservationDiagnostic.kind =
    some .missingClosure ∧
  [
    wrongSetupResult.implementationLinkDiagnostic?.map ImplementationLinkDiagnostic.kind,
    impossibleTransitionResult.implementationLinkDiagnostic?.map
      ImplementationLinkDiagnostic.kind,
    missingCoordinateResult.implementationLinkDiagnostic?.map
      ImplementationLinkDiagnostic.kind,
    behaviorFingerprintDriftResult.implementationLinkDiagnostic?.map
      ImplementationLinkDiagnostic.kind
  ] = [
    some .nonAuthoritativeSourceInitial,
    some .nonAuthoritativeSourceStep,
    some .absentCoordinate,
    some .behaviorFingerprintDrift
  ] ∧
  propertyFailureResult.evaluated?.map (fun result => result.evaluation.satisfied) = some false := by
  native_decide

/-- Failure provenance keeps Observation plan, Implementation Link, and Property identities apart. -/
example :
  observationFailureResult.observationDiagnostic?.map ObservationDiagnostic.planId =
      some startPlan.id ∧
    wrongSetupResult.implementationLinkDiagnostic?.map (fun diagnostic =>
      diagnostic.hasCanonicalIdentity &&
        diagnostic.implementationLinkId == implementationLinkId &&
        diagnostic.sourceTarget == .ofTarget Temporal.System.Nexus.target &&
        diagnostic.destinationTarget ==
          .ofTarget Temporal.Feature.Nexus.Lifecycle.target) = some true ∧
    propertyFailureResult.evaluated?.map (fun result => result.evaluation.propertyId) =
      some mutatedPropertyId ∧
    startPlan.id != implementationLinkId ∧
    implementationLinkId != mutatedPropertyId := by
  native_decide

/-- An Implementation Link failure exposes neither unknown Observation evidence nor a Property. -/
example : [
    wrongSetupResult,
    impossibleTransitionResult,
    missingCoordinateResult,
    behaviorFingerprintDriftResult
  ].all fun result =>
    result.observationDiagnostic?.isNone &&
      result.implementationLinkDiagnostic?.isSome &&
      result.evaluated?.isNone := by
  native_decide

namespace DuplicateDelivery

open Temporal.System.Nexus.ImplementationLink.CallerClosure.DuplicateDelivery

def strictApplication := applyImplementationLink
  Temporal.System.Nexus.ImplementationLink.CallerClosure.checked
  Temporal.System.Nexus.CallerClosure.setup
  Temporal.System.Nexus.ObservationFaultTests.completeTrace

def observedApplication := applyObservedTraceTranslation checkedObservedTranslation
  Temporal.System.Nexus.CallerClosure.setup
  Temporal.System.Nexus.ObservationFaultTests.completeTrace

def translatedTrace :=
  observedApplication.translated?.map TranslatedObservedTrace.trace

def propertyEvaluation := translatedTrace.map <|
  evaluateProperty Temporal.Feature.Nexus.Experimental.CallerClosure.callerClosureProperty

/-- The strict conformance path stays closed while the checked observed path carries no authority. -/
example :
    strictApplication.status = .invalid ∧
    strictApplication.diagnostic?.map ImplementationLinkDiagnostic.kind =
      some .nonAuthoritativeSourceStep ∧
    observedApplication.status = .applied ∧
    observedApplication.translated?.map (fun translation =>
      (translation.hasAuthorityClaim,
        translation.evidenceLinks.length,
        translation.evidenceLinks.all fun evidenceLink =>
          evidenceLink.implementationLinkId == observedImplementationLinkId &&
            evidenceLink.implementationLinkBehaviorFingerprint ==
              checkedObservedTranslation.behaviorFingerprint)) =
      some (false, 7, true) := by
  native_decide

/-- The unchanged Feature Property isolates the count-two uniqueness violation. -/
example : propertyEvaluation.map (fun evaluation =>
    (evaluation.propertyId,
      evaluation.satisfied,
      evaluation.clauses.map fun clause => (clause.clauseId, clause.satisfied))) = some (
    Temporal.Feature.Nexus.Experimental.CallerClosure.callerClosurePropertyId,
    false,
    [
      (DefinitionId.of "workflow-nexus.property.clause.delivery", true),
      (DefinitionId.of "workflow-nexus.property.clause.ownership", true),
      (DefinitionId.of "workflow-nexus.property.clause.uniqueness", false)
    ]) := by
  native_decide

example :
    Temporal.System.Nexus.ImplementationLink.CallerClosure.checked.hasCanonicalIdentity = true ∧
    Temporal.System.Nexus.ImplementationLink.CallerClosure.checked.declaration.id =
      Temporal.System.Nexus.ImplementationLink.CallerClosure.implementationLinkId ∧
    checkedObservedTranslation.hasCanonicalIdentity = true ∧
    checkedObservedTranslation.declaration.observationMappings.contains {
      source := sourceCancellationCountTwo
      destination := destinationCancellationCountTwo
    } := by
  native_decide

end DuplicateDelivery

end Temporal.ImplementationLinkTests.Nexus
