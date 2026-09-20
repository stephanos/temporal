import Temporal.System.Nexus.ImplementationLink

/-!
Composed checks for the ordinary Nexus lifecycle. Synthetic Evidence exists only to establish the
already-accepted System trace at the test boundary; the production operation consumes the typed
Observation result and never interprets raw Evidence.

The destination is the Target the link derives from the Caller Model's product machine, so the
Feature meaning every check lands on is the product machine's own: its states, its recorded facts,
and the routes its Queries find.
-/

namespace TemporalModelTests.Nexus.ImplementationLink

open Umpire
open Temporal.System.Nexus.ImplementationLink

example : checkedResult.isOk = true ∧ checked.hasCanonicalIdentity = true := by
  native_decide

example : checked.sourceTarget.id = Temporal.System.Nexus.targetId ∧
    checked.destinationTarget.id = productTargetId ∧
    checked.declaration.capabilityMappings = [lifecycleCapabilityMapping] ∧
    checked.declaration.relationMappings = [] := by
  native_decide

theorem checked_link_retains_target_identity_and_fingerprints :
    checked.sourceTarget.id = Temporal.System.Nexus.target.id ∧
    checked.sourceTarget.source = Temporal.System.Nexus.target.source ∧
    checked.sourceTarget.behaviorFingerprint =
      Temporal.System.Nexus.target.behaviorFingerprint ∧
    checked.sourceTarget.behaviorFingerprint.render =
      "sha256:9a131c48af0f15669b5f414754389046129da313f07774abbab76eaab25372b4" ∧
    checked.destinationTarget.id = productTarget.id ∧
    checked.destinationTarget.source = productTarget.source ∧
    checked.destinationTarget.behaviorFingerprint = productTarget.behaviorFingerprint ∧
    checked.destinationTarget.behaviorFingerprint.render =
      "sha256:663970a8c4be23dcfb9b622ddaff325f835ff2d5c1a7ba5031754b225978f30d" := by
  native_decide

/-- The product Target is the product machine's rows: its terminal closure is the machine's
`ends:`, its starts are `scheduled` and `started`, and its rows are the machine's own. -/
example : productTable.terminalConditions = [Temporal.Feature.Nexus.Caller.nexusProduct.ends] ∧
    productTarget.terminalConditions.map List.length =
      [Temporal.Feature.Nexus.Caller.nexusProduct.ends.length] ∧
    productTable.initial.flatMap (·.states) = [{ phase := .scheduled }, { phase := .started }] ∧
    productTarget.isTerminal succeededState = true ∧
    productTarget.isTerminal canceledState = true ∧
    productTarget.isTerminal startedState = false ∧
    productTarget.machine.initialStates productSetup = [scheduledState, startedState] := by
  native_decide

theorem targets_keep_their_named_authority_seams :
    Temporal.System.Nexus.target.machine.authoritativeInitial
      Temporal.System.Nexus.queuedSetup Temporal.System.Nexus.queuedState ∧
    Temporal.System.Nexus.target.machine.authoritativeStep
      Temporal.System.Nexus.queuedState Temporal.System.Nexus.dispatchAction
      Temporal.System.Nexus.dispatchedResult ∧
    productTarget.machine.authoritativeInitial productSetup scheduledState ∧
    productTarget.machine.authoritativeStep scheduledState asyncReplyAction startedResult ∧
    productTarget.machine.authoritativeStep startedState cancelCompletionAction canceledResult ∧
    productTarget.machine.authoritativeStep startedState successCompletionAction
      succeededResult := by
  refine ⟨Temporal.System.Nexus.target_queued_initial_authoritative,
    Temporal.System.Nexus.target_queued_dispatch_authoritative, ?_, ?_, ?_, ?_⟩
  · simpa [morphism] using initialForward Temporal.System.Nexus.queuedSetup
      Temporal.System.Nexus.queuedState Temporal.System.Nexus.target_queued_initial_authoritative
  · simpa [morphism, ValueTranslation.mapStep, Step.map, Temporal.System.Nexus.dispatchedResult,
      startedResult] using stepForward Temporal.System.Nexus.queuedState
      Temporal.System.Nexus.dispatchAction Temporal.System.Nexus.dispatchedResult
      Temporal.System.Nexus.target_queued_dispatch_authoritative
  · simpa [morphism, ValueTranslation.mapStep, Step.map,
      Temporal.System.Nexus.cancellationRecordedResult, canceledResult] using
      stepForward Temporal.System.Nexus.runningState
      Temporal.System.Nexus.recordCancellationAction
      Temporal.System.Nexus.cancellationRecordedResult
      Temporal.System.Nexus.target_running_cancellation_authoritative
  · simpa [morphism, ValueTranslation.mapStep, Step.map,
      Temporal.System.Nexus.completionRecordedResult, succeededResult] using
      stepForward Temporal.System.Nexus.runningState
      Temporal.System.Nexus.recordCompletionAction
      Temporal.System.Nexus.completionRecordedResult
      Temporal.System.Nexus.target_running_completion_authoritative

/-- The facts a Query's witness records, step by step. -/
private def witnessFacts (witness : Option Scenario.Trace) : List String :=
  (witness.map fun witness => witness.trace.steps.flatMap fun step =>
    step.facts.map ModelValue.value).getD []

/-- The Caller Model's Queries find the routes the link translates onto: the asynchronous completion
route records the scheduled, started and completed facts, and the link's translated start and
completion are that route after the schedule the System's `queued` start already stands for. -/
example : (match Temporal.Feature.Nexus.Caller.asyncCompletion with
    | .ok checked => witnessFacts checked.witness
    | .error _ => []) =
      ["nexusOperationScheduled", "nexusOperationStarted", "nexusOperationCompleted"] ∧
    (match Temporal.Feature.Nexus.Caller.syncCompletion with
    | .ok checked => witnessFacts checked.witness
    | .error _ => []) = ["nexusOperationScheduled", "nexusOperationCompleted"] ∧
    [mapObservation Temporal.System.Nexus.runningObservation,
      mapObservation Temporal.System.Nexus.completionRecordedObservation].map ModelValue.value =
      ["nexusOperationStarted", "nexusOperationCompleted"] ∧
    [mapObservation Temporal.System.Nexus.cancellationRecordedObservation].map ModelValue.value =
      ["nexusOperationCanceled"] := by
  native_decide

private def id (value : String) : DefinitionId := DefinitionId.of value

def source : SourceLocation := {
  path := "TemporalModelTests/Nexus/ImplementationLink.lean"
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
    (mappingId actionRuleId actionDefinitionId : DefinitionId) : Evidence.Reading := {
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
      outputKind := .fact
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

def observationContext : Evidence.ReadingContext :=
  Evidence.ReadingContext.ofTarget Temporal.System.Nexus.target [evidenceProfile]

def startPlanResult : Except Evidence.ReadingError Evidence.CheckedReading :=
  Evidence.checkReading observationContext <|
    observationDeclaration startMappingId dispatchRuleId Temporal.System.Nexus.dispatchActionId

private theorem startPlanResult_isSome : startPlanResult.toOption.isSome = true := by
  native_decide

def startPlan : Evidence.CheckedReading :=
  startPlanResult.toOption.get startPlanResult_isSome

def cancellationPlanResult : Except Evidence.ReadingError Evidence.CheckedReading :=
  Evidence.checkReading observationContext <| observationDeclaration cancellationMappingId
    cancellationRuleId Temporal.System.Nexus.recordCancellationActionId

private theorem cancellationPlanResult_isSome : cancellationPlanResult.toOption.isSome = true := by
  native_decide

def cancellationPlan : Evidence.CheckedReading :=
  cancellationPlanResult.toOption.get cancellationPlanResult_isSome

def successfulCompletionPlanResult : Except Evidence.ReadingError Evidence.CheckedReading :=
  Evidence.checkReading observationContext <| observationDeclaration successfulCompletionMappingId
    completionRuleId Temporal.System.Nexus.recordCompletionActionId

private theorem successfulCompletionPlanResult_isSome :
    successfulCompletionPlanResult.toOption.isSome = true := by
  native_decide

def successfulCompletionPlan : Evidence.CheckedReading :=
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
    (initialState action outcome resultingState observation : ModelValue) : SyntheticEvidence := {
  profile := profileId
  profileVersion := 1
  records := [
    stepRecord stepId initialId action outcome resultingState observation,
    initialRecord initialId initialState
  ]
  closures := [{ kind := evidenceKind, lastSequence := 2 }]
}

def startEvidence : SyntheticEvidence := oneStepEvidence
  (id "temporal.test.nexus.system.start.initial")
  (id "temporal.test.nexus.system.start.step")
  Temporal.System.Nexus.queuedState
  Temporal.System.Nexus.dispatchAction
  Temporal.System.Nexus.dispatchedOutcome
  Temporal.System.Nexus.runningState
  Temporal.System.Nexus.runningObservation

def cancellationEvidence : SyntheticEvidence := oneStepEvidence
  (id "temporal.test.nexus.system.cancellation.initial")
  (id "temporal.test.nexus.system.cancellation.step")
  Temporal.System.Nexus.runningState
  Temporal.System.Nexus.recordCancellationAction
  Temporal.System.Nexus.cancellationRecordedOutcome
  Temporal.System.Nexus.cancellationRecordedState
  Temporal.System.Nexus.cancellationRecordedObservation

def successfulCompletionEvidence : SyntheticEvidence := oneStepEvidence
  (id "temporal.test.nexus.system.successful-completion.initial")
  (id "temporal.test.nexus.system.successful-completion.step")
  Temporal.System.Nexus.runningState
  Temporal.System.Nexus.recordCompletionAction
  Temporal.System.Nexus.completionRecordedOutcome
  Temporal.System.Nexus.completionRecordedState
  Temporal.System.Nexus.completionRecordedObservation

/-! ### Properties over the product Target

Each Property is one transition contract, from the class a System step maps to onto the phase it
lands in, checked against the product Target. -/

private def transitionProperty (propertyId : DefinitionId) (action state : ModelValue) :
    Property := {
  id := propertyId
  source
  requires := [Temporal.Feature.Nexus.Caller.nexusProduct.capabilityId]
  clauses := [
    .transitionContract (id (propertyId.value ++ ".contract"))
      { field := .selectedAction, reference := action.definitionId,
        constraint := .equals action.value }
      { field := .resultingState, reference := state.definitionId,
        constraint := .equals state.value }
  ]
}

private def checkedProperty (declaration : Property) : Except PropertyError CheckedProperty :=
  Property.check (PropertyCheckContext.ofTarget productTarget) declaration

def asyncStartProperty : CheckedProperty :=
  (checkedProperty (transitionProperty (id "temporal.test.nexus.feature.property.async-start")
    asyncReplyAction startedState)).toOption.get (by native_decide)

def cancellationProperty : CheckedProperty :=
  (checkedProperty (transitionProperty (id "temporal.test.nexus.feature.property.cancellation")
    cancelCompletionAction canceledState)).toOption.get (by native_decide)

def successfulCompletionProperty : CheckedProperty :=
  (checkedProperty (transitionProperty
    (id "temporal.test.nexus.feature.property.successful-completion")
    successCompletionAction succeededState)).toOption.get (by native_decide)

/-- The product trace one System step translates onto. -/
private def productTrace (initialState action : ModelValue)
    (result : Step ModelValue ModelValue ModelValue) :
    ModelTrace ModelValue ModelValue ModelValue ModelValue := {
  initialState
  steps := [{
    selectedAction := action
    outcome := result.outcome
    state := result.state
    facts := result.facts
  }]
}

def asyncStartTrace := productTrace scheduledState asyncReplyAction startedResult
def cancellationTrace := productTrace startedState cancelCompletionAction canceledResult
def successfulCompletionTrace := productTrace startedState successCompletionAction succeededResult

def startObservation : ObservationResult := evaluateEvidence startPlan startEvidence
def cancellationObservation : ObservationResult :=
  evaluateEvidence cancellationPlan cancellationEvidence
def successfulCompletionObservation : ObservationResult :=
  evaluateEvidence successfulCompletionPlan successfulCompletionEvidence

def startResult : FeaturePropertyResult := evaluateFeatureProperty
  Temporal.System.Nexus.queuedSetup asyncStartProperty startObservation

def cancellationResult : FeaturePropertyResult := evaluateFeatureProperty
  Temporal.System.Nexus.runningSetup cancellationProperty cancellationObservation

def successfulCompletionResult : FeaturePropertyResult := evaluateFeatureProperty
  Temporal.System.Nexus.runningSetup successfulCompletionProperty successfulCompletionObservation

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
      application.evidenceSupports.map ImplementationLinkEvidenceSupport.coordinate,
      application.evidenceSupports.all fun evidenceSupport =>
        evidenceSupport.implementationLinkId == implementationLinkId &&
          evidenceSupport.implementationLinkBehaviorFingerprint == checked.behaviorFingerprint &&
          evidenceSupport.sourceTarget == .ofTarget Temporal.System.Nexus.target &&
          evidenceSupport.destinationTarget == .ofTarget productTarget &&
          evidenceSupport.identity != behaviorFingerprintOf "")

private def expectedCoordinates : List ModelCoordinate := [
  .initialState,
  .selectedAction 1,
  .outcome 1,
  .state 1,
  .fact 1 1
]

/-- Start, cancel, and successful completion translate completely with positional Evidence Links. -/
example : ([
    applicationShape startResult,
    applicationShape cancellationResult,
    applicationShape successfulCompletionResult
  ] == [
    some (Temporal.System.Nexus.queuedSetup, productSetup, asyncStartTrace,
      expectedCoordinates, true),
    some (Temporal.System.Nexus.runningSetup, productSetup, cancellationTrace,
      expectedCoordinates, true),
    some (Temporal.System.Nexus.runningSetup, productSetup, successfulCompletionTrace,
      expectedCoordinates, true)
  ]) = true := by
  native_decide

/-- Composition invokes the unchanged Feature evaluator only after successful translation. -/
example : [
    startResult.evaluated?.map EvaluatedFeatureProperty.evaluation,
    cancellationResult.evaluated?.map EvaluatedFeatureProperty.evaluation,
    successfulCompletionResult.evaluated?.map EvaluatedFeatureProperty.evaluation
  ] = [
    (evaluatePropertyOnTrace asyncStartProperty asyncStartTrace).toOption,
    (evaluatePropertyOnTrace cancellationProperty cancellationTrace).toOption,
    (evaluatePropertyOnTrace successfulCompletionProperty successfulCompletionTrace).toOption
  ] ∧ [
    startResult.evaluated?.map (fun result => result.evaluation.satisfied),
    cancellationResult.evaluated?.map (fun result => result.evaluation.satisfied),
    successfulCompletionResult.evaluated?.map (fun result => result.evaluation.satisfied)
  ] = [some true, some true, some true] := by
  native_decide

def missingClosureObservation : ObservationResult :=
  evaluateEvidence startPlan { startEvidence with closures := [] }

def observationFailureResult : FeaturePropertyResult := evaluateFeatureProperty
  Temporal.System.Nexus.queuedSetup asyncStartProperty missingClosureObservation

def wrongSetupResult : FeaturePropertyResult := evaluateFeatureProperty
  Temporal.System.Nexus.runningSetup asyncStartProperty startObservation

def impossibleTransitionEvidence : SyntheticEvidence := oneStepEvidence
  (id "temporal.test.nexus.system.impossible.initial")
  (id "temporal.test.nexus.system.impossible.step")
  Temporal.System.Nexus.queuedState
  Temporal.System.Nexus.recordCompletionAction
  Temporal.System.Nexus.completionRecordedOutcome
  Temporal.System.Nexus.completionRecordedState
  Temporal.System.Nexus.completionRecordedObservation

def impossibleStep : FeaturePropertyResult := evaluateFeatureProperty
  Temporal.System.Nexus.queuedSetup successfulCompletionProperty
  (evaluateEvidence successfulCompletionPlan impossibleTransitionEvidence)

private def acceptedTrace? : ObservationResult → Option EvidenceBackedTrace
  | .accepted trace => some trace
  | _ => none

def startTrace : EvidenceBackedTrace :=
  (acceptedTrace? startObservation).get (by native_decide)

private def uncheckedTraceOf (trace : EvidenceBackedTrace) : UncheckedEvidenceBackedTrace := {
  traceId := trace.traceId
  checkedPlan := trace.checkedPlan
  mappingId := trace.mappingId
  mappingVersion := trace.mappingVersion
  mappingDigest := trace.mappingDigest
  source := trace.source
  profileId := trace.profileId
  profileVersion := trace.profileVersion
  sourceClosed := trace.sourceClosed
  vocabulary := trace.vocabulary
  dispositions := trace.dispositions
  appliedBound := trace.appliedBound
  evidenceIdentities := trace.evidenceIdentities
  recordSupport := trace.recordSupport
  trace := trace.trace
  evidenceSupports := trace.evidenceSupports
}

private def observationResultOfAdmission
    (result : Except ObservationDiagnostic EvidenceBackedTrace) : ObservationResult :=
  match result with
  | .ok trace => .accepted trace
  | .error diagnostic =>
      match diagnostic.status with
      | .unknown => .unknown diagnostic
      | .conflict => .conflict diagnostic
      | .unsupported => .unsupported diagnostic
      | .accepted => .unknown diagnostic

def missingCoordinateObservation : ObservationResult :=
  let unchecked := uncheckedTraceOf startTrace
  observationResultOfAdmission <| validateEvidenceBackedTrace {
    unchecked with evidenceSupports := unchecked.evidenceSupports.tail
  }

def missingCoordinateResult : FeaturePropertyResult := evaluateFeatureProperty
  Temporal.System.Nexus.queuedSetup asyncStartProperty missingCoordinateObservation

private def driftMeaning (meaning : Meaning) : Meaning :=
  if meaning.definitionId == Temporal.System.Nexus.operationStateId then
    { meaning with behaviorVersion := "temporal-system-nexus-lifecycle-state/drift" }
  else
    meaning

def driftContext : Evidence.ReadingContext := {
  observationContext with meanings := observationContext.meanings.map driftMeaning
}

def driftPlanResult : Except Evidence.ReadingError Evidence.CheckedReading :=
  Evidence.checkReading driftContext <|
    observationDeclaration startMappingId dispatchRuleId Temporal.System.Nexus.dispatchActionId

private theorem driftPlanResult_isSome : driftPlanResult.toOption.isSome = true := by
  native_decide

def driftPlan : Evidence.CheckedReading :=
  driftPlanResult.toOption.get driftPlanResult_isSome

def behaviorFingerprintDriftResult : FeaturePropertyResult := evaluateFeatureProperty
  Temporal.System.Nexus.queuedSetup asyncStartProperty (evaluateEvidence driftPlan startEvidence)

def mutatedPropertyId : DefinitionId :=
  id "temporal.test.nexus.feature.property.mutated-start"

/-- The start contract with the wrong landing phase: an asynchronous reply lands in `started`,
never `succeeded`. -/
def mutatedPropertyDeclaration : Property :=
  transitionProperty mutatedPropertyId asyncReplyAction succeededState

def mutatedPropertyResult : Except PropertyError CheckedProperty :=
  checkedProperty mutatedPropertyDeclaration

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
    impossibleStep.layer,
    missingCoordinateResult.layer,
    behaviorFingerprintDriftResult.layer,
    propertyFailureResult.layer
  ] = [
    .observation,
    .implementationLink,
    .implementationLink,
    .observation,
    .implementationLink,
    .featureProperty
  ] ∧
  observationFailureResult.observationDiagnostic?.map ObservationDiagnostic.kind =
    some .missingClosure ∧
  missingCoordinateResult.observationDiagnostic?.map ObservationDiagnostic.kind =
    some .absentModelCoordinate ∧
  [
    wrongSetupResult.implementationLinkDiagnostic?.map ImplementationLinkDiagnostic.kind,
    impossibleStep.implementationLinkDiagnostic?.map
      ImplementationLinkDiagnostic.kind,
    behaviorFingerprintDriftResult.implementationLinkDiagnostic?.map
      ImplementationLinkDiagnostic.kind
  ] = [
    some .nonAuthoritativeSourceInitial,
    some .nonAuthoritativeSourceStep,
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
        diagnostic.destinationTarget == .ofTarget productTarget) = some true ∧
    propertyFailureResult.evaluated?.map (fun result => result.evaluation.propertyId) =
      some mutatedPropertyId ∧
    startPlan.id != implementationLinkId ∧
    implementationLinkId != mutatedPropertyId := by
  native_decide

/-- An Implementation Link failure exposes neither unknown Observation evidence nor a Property. -/
example : [
    wrongSetupResult,
    impossibleStep,
    behaviorFingerprintDriftResult
  ].all fun result =>
    result.observationDiagnostic?.isNone &&
      result.implementationLinkDiagnostic?.isSome &&
      result.evaluated?.isNone := by
  native_decide

/-- A malformed unchecked trace stops at Observation admission without a later-stage result. -/
example :
    missingCoordinateResult.observationDiagnostic?.map ObservationDiagnostic.kind =
        some .absentModelCoordinate ∧
      missingCoordinateResult.implementationLinkDiagnostic?.isNone ∧
      missingCoordinateResult.evaluated?.isNone := by
  native_decide

end TemporalModelTests.Nexus.ImplementationLink
