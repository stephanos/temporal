import Temporal.Feature.Nexus.Lifecycle
import Temporal.System.Nexus.Core
import Umpire.ImplementationLink
import Umpire.Property

/-!
# Nexus lifecycle Implementation Link

This is the sole production leaf that imports both the independently authored Nexus System
mechanism and Feature product meaning. It declares and proves the bounded forward correspondence;
neither base module imports or redefines the other.
-/

namespace Temporal.System.Nexus.ImplementationLink

open Umpire

private def id (value : String) : DefinitionId := DefinitionId.of value

def source : SourceLocation := {
  path := "Temporal/System/Nexus/ImplementationLink.lean"
  line := 1
  column := 1
  provenance := "lean-model"
}

def implementationLinkId : DefinitionId :=
  id "temporal.system.nexus.lifecycle.implementation-link"

def mapSetup : Temporal.System.Nexus.ExecutionSetup → List RoleBinding
  | .queued => Temporal.Feature.Nexus.Lifecycle.scheduledSetup
  | .running => Temporal.Feature.Nexus.Lifecycle.startedSetup

def mapState (state : ModelValue) : ModelValue :=
  if state = Temporal.System.Nexus.queuedState then
    Temporal.Feature.Nexus.Lifecycle.scheduledState
  else if state = Temporal.System.Nexus.runningState then
    Temporal.Feature.Nexus.Lifecycle.startedState
  else if state = Temporal.System.Nexus.cancellationRecordedState then
    Temporal.Feature.Nexus.Lifecycle.canceledState
  else
    Temporal.Feature.Nexus.Lifecycle.succeededState

def mapAction (action : ModelValue) : ModelValue :=
  if action = Temporal.System.Nexus.dispatchAction then
    Temporal.Feature.Nexus.Lifecycle.startAction
  else if action = Temporal.System.Nexus.recordCancellationAction then
    Temporal.Feature.Nexus.Lifecycle.cancelAction
  else
    Temporal.Feature.Nexus.Lifecycle.reportSuccessAction

def mapOutcome (outcome : ModelValue) : ModelValue :=
  if outcome = Temporal.System.Nexus.dispatchedOutcome then
    Temporal.Feature.Nexus.Lifecycle.startedOutcome
  else if outcome = Temporal.System.Nexus.cancellationRecordedOutcome then
    Temporal.Feature.Nexus.Lifecycle.canceledOutcome
  else
    Temporal.Feature.Nexus.Lifecycle.succeededOutcome

def mapObservation (observation : ModelValue) : ModelValue :=
  if observation = Temporal.System.Nexus.runningObservation then
    Temporal.Feature.Nexus.Lifecycle.startedObservation
  else if observation = Temporal.System.Nexus.cancellationRecordedObservation then
    Temporal.Feature.Nexus.Lifecycle.canceledObservation
  else
    Temporal.Feature.Nexus.Lifecycle.succeededObservation

@[simp] theorem mapSetup_queued :
    mapSetup Temporal.System.Nexus.queuedSetup =
      Temporal.Feature.Nexus.Lifecycle.scheduledSetup := by native_decide

@[simp] theorem mapSetup_running :
    mapSetup Temporal.System.Nexus.runningSetup =
      Temporal.Feature.Nexus.Lifecycle.startedSetup := by native_decide

@[simp] theorem mapState_queued :
    mapState Temporal.System.Nexus.queuedState =
      Temporal.Feature.Nexus.Lifecycle.scheduledState := by native_decide

@[simp] theorem mapState_running :
    mapState Temporal.System.Nexus.runningState =
      Temporal.Feature.Nexus.Lifecycle.startedState := by native_decide

@[simp] theorem mapState_cancellationRecorded :
    mapState Temporal.System.Nexus.cancellationRecordedState =
      Temporal.Feature.Nexus.Lifecycle.canceledState := by native_decide

@[simp] theorem mapState_completionRecorded :
    mapState Temporal.System.Nexus.completionRecordedState =
      Temporal.Feature.Nexus.Lifecycle.succeededState := by native_decide

@[simp] theorem mapAction_dispatch :
    mapAction Temporal.System.Nexus.dispatchAction =
      Temporal.Feature.Nexus.Lifecycle.startAction := by native_decide

@[simp] theorem mapAction_recordCancellation :
    mapAction Temporal.System.Nexus.recordCancellationAction =
      Temporal.Feature.Nexus.Lifecycle.cancelAction := by native_decide

@[simp] theorem mapAction_recordCompletion :
    mapAction Temporal.System.Nexus.recordCompletionAction =
      Temporal.Feature.Nexus.Lifecycle.reportSuccessAction := by native_decide

@[simp] theorem mapOutcome_dispatched :
    mapOutcome Temporal.System.Nexus.dispatchedOutcome =
      Temporal.Feature.Nexus.Lifecycle.startedOutcome := by native_decide

@[simp] theorem mapOutcome_cancellationRecorded :
    mapOutcome Temporal.System.Nexus.cancellationRecordedOutcome =
      Temporal.Feature.Nexus.Lifecycle.canceledOutcome := by native_decide

@[simp] theorem mapOutcome_completionRecorded :
    mapOutcome Temporal.System.Nexus.completionRecordedOutcome =
      Temporal.Feature.Nexus.Lifecycle.succeededOutcome := by native_decide

@[simp] theorem mapObservation_running :
    mapObservation Temporal.System.Nexus.runningObservation =
      Temporal.Feature.Nexus.Lifecycle.startedObservation := by native_decide

@[simp] theorem mapObservation_cancellationRecorded :
    mapObservation Temporal.System.Nexus.cancellationRecordedObservation =
      Temporal.Feature.Nexus.Lifecycle.canceledObservation := by native_decide

@[simp] theorem mapObservation_completionRecorded :
    mapObservation Temporal.System.Nexus.completionRecordedObservation =
      Temporal.Feature.Nexus.Lifecycle.succeededObservation := by native_decide

def sourceCapabilityReference : ImplementationSemanticReference :=
  (implementationSemanticReference? Temporal.System.Nexus.target
    Temporal.System.Nexus.lifecycleCapabilityId .capability).get (by native_decide)

def destinationCapabilityReference : ImplementationSemanticReference :=
  (implementationSemanticReference? Temporal.Feature.Nexus.Lifecycle.target
    Temporal.Feature.Nexus.Lifecycle.lifecycleCapabilityId .capability).get (by native_decide)

def lifecycleCapabilityMapping : ImplementationSemanticMapping := {
  source := sourceCapabilityReference
  destination := destinationCapabilityReference
}

def declaration : ImplementationLinkDeclaration
    Temporal.System.Nexus.ExecutionSetup ModelValue ModelValue ModelValue ModelValue
    (List RoleBinding) ModelValue ModelValue ModelValue ModelValue := {
  id := implementationLinkId
  source
  sourceTarget := .ofTarget Temporal.System.Nexus.target
  destinationTarget := .ofTarget Temporal.Feature.Nexus.Lifecycle.target
  setupMappings := [
    { source := Temporal.System.Nexus.queuedSetup,
      destination := Temporal.Feature.Nexus.Lifecycle.scheduledSetup },
    { source := Temporal.System.Nexus.runningSetup,
      destination := Temporal.Feature.Nexus.Lifecycle.startedSetup }
  ]
  stateMappings := [
    { source := Temporal.System.Nexus.queuedState,
      destination := Temporal.Feature.Nexus.Lifecycle.scheduledState },
    { source := Temporal.System.Nexus.runningState,
      destination := Temporal.Feature.Nexus.Lifecycle.startedState },
    { source := Temporal.System.Nexus.cancellationRecordedState,
      destination := Temporal.Feature.Nexus.Lifecycle.canceledState },
    { source := Temporal.System.Nexus.completionRecordedState,
      destination := Temporal.Feature.Nexus.Lifecycle.succeededState }
  ]
  actionMappings := [
    { source := Temporal.System.Nexus.dispatchAction,
      destination := Temporal.Feature.Nexus.Lifecycle.startAction },
    { source := Temporal.System.Nexus.recordCancellationAction,
      destination := Temporal.Feature.Nexus.Lifecycle.cancelAction },
    { source := Temporal.System.Nexus.recordCompletionAction,
      destination := Temporal.Feature.Nexus.Lifecycle.reportSuccessAction }
  ]
  outcomeMappings := [
    { source := Temporal.System.Nexus.dispatchedOutcome,
      destination := Temporal.Feature.Nexus.Lifecycle.startedOutcome },
    { source := Temporal.System.Nexus.cancellationRecordedOutcome,
      destination := Temporal.Feature.Nexus.Lifecycle.canceledOutcome },
    { source := Temporal.System.Nexus.completionRecordedOutcome,
      destination := Temporal.Feature.Nexus.Lifecycle.succeededOutcome }
  ]
  observationMappings := [
    { source := Temporal.System.Nexus.runningObservation,
      destination := Temporal.Feature.Nexus.Lifecycle.startedObservation },
    { source := Temporal.System.Nexus.cancellationRecordedObservation,
      destination := Temporal.Feature.Nexus.Lifecycle.canceledObservation },
    { source := Temporal.System.Nexus.completionRecordedObservation,
      destination := Temporal.Feature.Nexus.Lifecycle.succeededObservation }
  ]
  relationMappings := []
  capabilityMappings := [lifecycleCapabilityMapping]
  applicationLimit := { value := 3, unit := .semanticTransitions }
  documentation := "The pure Nexus System lifecycle forward-simulates Feature lifecycle meaning."
}

theorem requiredCoverage : ImplementationLinkRequiredCoverage declaration
    Temporal.System.Nexus.target mapSetup mapState mapAction mapOutcome mapObservation := {
  setup := by
    intro value admitted
    change value = Temporal.System.Nexus.queuedSetup ∨
      value = Temporal.System.Nexus.runningSetup at admitted
    rcases admitted with rfl | rfl
    · apply Or.inl
      change ImplementationValueMapping.mk Temporal.System.Nexus.queuedSetup
        (mapSetup Temporal.System.Nexus.queuedSetup) ∈ declaration.setupMappings
      rw [mapSetup_queued]
      exact List.mem_cons.mpr (.inl rfl)
    · apply Or.inl
      change ImplementationValueMapping.mk Temporal.System.Nexus.runningSetup
        (mapSetup Temporal.System.Nexus.runningSetup) ∈ declaration.setupMappings
      rw [mapSetup_running]
      exact List.mem_cons.mpr (.inr (List.mem_singleton.mpr rfl))
  state := by
    intro value admitted
    change value = Temporal.System.Nexus.queuedState ∨
      value = Temporal.System.Nexus.runningState ∨
      value = Temporal.System.Nexus.cancellationRecordedState ∨
      value = Temporal.System.Nexus.completionRecordedState at admitted
    rcases admitted with rfl | rfl | rfl | rfl
    · apply Or.inl
      change ImplementationValueMapping.mk Temporal.System.Nexus.queuedState
        (mapState Temporal.System.Nexus.queuedState) ∈ declaration.stateMappings
      rw [mapState_queued]
      exact List.mem_cons.mpr (.inl rfl)
    · apply Or.inl
      change ImplementationValueMapping.mk Temporal.System.Nexus.runningState
        (mapState Temporal.System.Nexus.runningState) ∈ declaration.stateMappings
      rw [mapState_running]
      exact List.mem_cons.mpr (.inr (List.mem_cons.mpr (.inl rfl)))
    · apply Or.inl
      change ImplementationValueMapping.mk Temporal.System.Nexus.cancellationRecordedState
        (mapState Temporal.System.Nexus.cancellationRecordedState) ∈ declaration.stateMappings
      rw [mapState_cancellationRecorded]
      exact List.mem_cons.mpr (.inr (List.mem_cons.mpr (.inr
        (List.mem_cons.mpr (.inl rfl)))))
    · apply Or.inl
      change ImplementationValueMapping.mk Temporal.System.Nexus.completionRecordedState
        (mapState Temporal.System.Nexus.completionRecordedState) ∈ declaration.stateMappings
      rw [mapState_completionRecorded]
      exact List.mem_cons.mpr (.inr (List.mem_cons.mpr (.inr
        (List.mem_cons.mpr (.inr (List.mem_singleton.mpr rfl))))))
  action := by
    intro value admitted
    change value = Temporal.System.Nexus.dispatchAction ∨
      value = Temporal.System.Nexus.recordCancellationAction ∨
      value = Temporal.System.Nexus.recordCompletionAction at admitted
    rcases admitted with rfl | rfl | rfl
    · apply Or.inl
      change ImplementationValueMapping.mk Temporal.System.Nexus.dispatchAction
        (mapAction Temporal.System.Nexus.dispatchAction) ∈ declaration.actionMappings
      rw [mapAction_dispatch]
      exact List.mem_cons.mpr (.inl rfl)
    · apply Or.inl
      change ImplementationValueMapping.mk Temporal.System.Nexus.recordCancellationAction
        (mapAction Temporal.System.Nexus.recordCancellationAction) ∈ declaration.actionMappings
      rw [mapAction_recordCancellation]
      exact List.mem_cons.mpr (.inr (List.mem_cons.mpr (.inl rfl)))
    · apply Or.inl
      change ImplementationValueMapping.mk Temporal.System.Nexus.recordCompletionAction
        (mapAction Temporal.System.Nexus.recordCompletionAction) ∈ declaration.actionMappings
      rw [mapAction_recordCompletion]
      exact List.mem_cons.mpr (.inr (List.mem_cons.mpr (.inr
        (List.mem_singleton.mpr rfl))))
  outcome := by
    intro value admitted
    change value = Temporal.System.Nexus.dispatchedOutcome ∨
      value = Temporal.System.Nexus.cancellationRecordedOutcome ∨
      value = Temporal.System.Nexus.completionRecordedOutcome at admitted
    rcases admitted with rfl | rfl | rfl
    · apply Or.inl
      change ImplementationValueMapping.mk Temporal.System.Nexus.dispatchedOutcome
        (mapOutcome Temporal.System.Nexus.dispatchedOutcome) ∈ declaration.outcomeMappings
      rw [mapOutcome_dispatched]
      exact List.mem_cons.mpr (.inl rfl)
    · apply Or.inl
      change ImplementationValueMapping.mk Temporal.System.Nexus.cancellationRecordedOutcome
        (mapOutcome Temporal.System.Nexus.cancellationRecordedOutcome) ∈ declaration.outcomeMappings
      rw [mapOutcome_cancellationRecorded]
      exact List.mem_cons.mpr (.inr (List.mem_cons.mpr (.inl rfl)))
    · apply Or.inl
      change ImplementationValueMapping.mk Temporal.System.Nexus.completionRecordedOutcome
        (mapOutcome Temporal.System.Nexus.completionRecordedOutcome) ∈ declaration.outcomeMappings
      rw [mapOutcome_completionRecorded]
      exact List.mem_cons.mpr (.inr (List.mem_cons.mpr (.inr
        (List.mem_singleton.mpr rfl))))
  observation := by
    intro value admitted
    change value = Temporal.System.Nexus.runningObservation ∨
      value = Temporal.System.Nexus.cancellationRecordedObservation ∨
      value = Temporal.System.Nexus.completionRecordedObservation at admitted
    rcases admitted with rfl | rfl | rfl
    · apply Or.inl
      change ImplementationValueMapping.mk Temporal.System.Nexus.runningObservation
        (mapObservation Temporal.System.Nexus.runningObservation) ∈ declaration.observationMappings
      rw [mapObservation_running]
      exact List.mem_cons.mpr (.inl rfl)
    · apply Or.inl
      change ImplementationValueMapping.mk Temporal.System.Nexus.cancellationRecordedObservation
        (mapObservation Temporal.System.Nexus.cancellationRecordedObservation) ∈
          declaration.observationMappings
      rw [mapObservation_cancellationRecorded]
      exact List.mem_cons.mpr (.inr (List.mem_cons.mpr (.inl rfl)))
    · apply Or.inl
      change ImplementationValueMapping.mk Temporal.System.Nexus.completionRecordedObservation
        (mapObservation Temporal.System.Nexus.completionRecordedObservation) ∈
          declaration.observationMappings
      rw [mapObservation_completionRecorded]
      exact List.mem_cons.mpr (.inr (List.mem_cons.mpr (.inr
        (List.mem_singleton.mpr rfl))))
  relation := by native_decide
  capability := by native_decide
}

def witness : ImplementationLinkWitness declaration Temporal.System.Nexus.target
    Temporal.Feature.Nexus.Lifecycle.target := {
  index := implementationLinkWitnessIndex declaration Temporal.System.Nexus.target
    Temporal.Feature.Nexus.Lifecycle.target
  mapSetup
  mapState
  mapAction
  mapOutcome
  mapObservation
  initialForward := by
    intro setup state admitted
    change Temporal.System.Nexus.authoritativeInitial setup state at admitted
    rcases Temporal.System.Nexus.authoritativeInitial_cases setup state admitted with
      ⟨rfl, rfl⟩ | ⟨rfl, rfl⟩
    · simpa only [mapSetup_queued, mapState_queued] using
        Temporal.Feature.Nexus.Lifecycle.target_scheduled_initial_authoritative
    · simpa only [mapSetup_running, mapState_running] using
        Temporal.Feature.Nexus.Lifecycle.target_started_initial_authoritative
  stepForward := by
    intro state action result admitted
    change Temporal.System.Nexus.authoritativeStep state action result at admitted
    rcases Temporal.System.Nexus.authoritativeStep_cases state action result admitted with
      ⟨rfl, rfl, rfl⟩ | ⟨rfl, rfl, rfl⟩ | ⟨rfl, rfl, rfl⟩
    · simpa only [mapState_queued, mapAction_dispatch, Temporal.System.Nexus.dispatchedResult,
        mapOutcome_dispatched, mapState_running, List.map_cons, mapObservation_running,
        List.map_nil, Temporal.Feature.Nexus.Lifecycle.startedResult] using
        Temporal.Feature.Nexus.Lifecycle.target_scheduled_start_authoritative
    · simpa only [mapState_running, mapAction_recordCancellation,
        Temporal.System.Nexus.cancellationRecordedResult, mapOutcome_cancellationRecorded,
        mapState_cancellationRecorded, List.map_cons, mapObservation_cancellationRecorded,
        List.map_nil, Temporal.Feature.Nexus.Lifecycle.canceledResult] using
        Temporal.Feature.Nexus.Lifecycle.target_started_cancel_authoritative
    · simpa only [mapState_running, mapAction_recordCompletion,
        Temporal.System.Nexus.completionRecordedResult, mapOutcome_completionRecorded,
        mapState_completionRecorded, List.map_cons, mapObservation_completionRecorded,
        List.map_nil, Temporal.Feature.Nexus.Lifecycle.succeededResult] using
        Temporal.Feature.Nexus.Lifecycle.target_started_reportSuccess_authoritative
  requiredCoverage
}

def checkedResult := checkImplementationLink declaration Temporal.System.Nexus.target
  Temporal.Feature.Nexus.Lifecycle.target witness

private theorem checkedResult_isSome : checkedResult.toOption.isSome = true := by
  native_decide

/-- The checked first Temporal System-to-Feature correspondence. -/
def checked := checkedResult.toOption.get checkedResult_isSome

/-- The semantic layer responsible for one System-to-Feature Property result. -/
inductive FeaturePropertyLayer where
  | observation
  | implementationLink
  | property
  deriving BEq, DecidableEq, Repr

/-- Successful composition retains both the complete Implementation Link Evidence Links and the
unchanged Feature Property evaluation. -/
structure EvaluatedFeatureProperty where
  application : AppliedImplementationLink checked
  evaluation : PropertyEvaluation

/-- Observation, Implementation Link, and Feature Property outcomes remain disjoint. -/
inductive FeaturePropertyResult where
  | observationFailure (diagnostic : ObservationDiagnostic)
  | implementationLinkFailure (diagnostic : ImplementationLinkDiagnostic)
  | evaluated (result : EvaluatedFeatureProperty)

def FeaturePropertyResult.layer : FeaturePropertyResult → FeaturePropertyLayer
  | .observationFailure _ => .observation
  | .implementationLinkFailure _ => .implementationLink
  | .evaluated _ => .property

def FeaturePropertyResult.observationDiagnostic? :
    FeaturePropertyResult → Option ObservationDiagnostic
  | .observationFailure diagnostic => some diagnostic
  | _ => none

def FeaturePropertyResult.implementationLinkDiagnostic? :
    FeaturePropertyResult → Option ImplementationLinkDiagnostic
  | .implementationLinkFailure diagnostic => some diagnostic
  | _ => none

def FeaturePropertyResult.evaluated? :
    FeaturePropertyResult → Option EvaluatedFeatureProperty
  | .evaluated result => some result
  | _ => none

/-- Compose an upstream Observation result through the checked Nexus Implementation Link. Property
evaluation runs only after the source trace is re-admitted and translated successfully. -/
def evaluateFeatureProperty
    (sourceSetup : Temporal.System.Nexus.ExecutionSetup)
    (property : CheckedProperty)
    (observation : ObservationResult) : FeaturePropertyResult :=
  match observation with
  | .unknown diagnostic | .conflict diagnostic | .unsupported diagnostic =>
      .observationFailure diagnostic
  | .accepted trace =>
      match applyImplementationLink checked sourceSetup trace with
      | .applied application => .evaluated {
          application
          evaluation := evaluateProperty property application.trace
        }
      | .invalid diagnostic
      | .unknown diagnostic
      | .conflict diagnostic
      | .unsupported diagnostic => .implementationLinkFailure diagnostic

end Temporal.System.Nexus.ImplementationLink
