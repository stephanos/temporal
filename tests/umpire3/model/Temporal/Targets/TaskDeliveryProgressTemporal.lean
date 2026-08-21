import Temporal.System.TaskDeliveryProgress
import Umpire3.TemporalView

namespace Umpire3.Temporal.Targets.TaskDeliveryProgressTemporal

open Umpire3.Temporal.System.TaskDeliveryProgress

private def states : List String := ["unavailable", "ready", "completed"]

private def actions : List String := ["progress-entity", "recover-owner"]

private def recoveryTransition : TemporalTransition where
  action := "recover-owner"
  fromState := "unavailable"
  toState := "ready"

private def deliveryTransition : TemporalTransition where
  action := "progress-entity"
  fromState := "ready"
  toState := "completed"

private def recoveryFairness : TemporalFairness where
  identifier := recoveryFairnessIdentifier
  kind := "responsive"
  action := "recover-owner"
  enabledStates := ["unavailable"]

private def deliveryFairness : TemporalFairness where
  identifier := deliveryFairnessIdentifier
  kind := "responsive"
  action := "progress-entity"
  enabledStates := ["ready"]

private def progress : TemporalProgress where
  identifier := "entity.progress"
  triggerStates := ["unavailable", "ready"]
  goalStates := ["completed"]

private def artifact (variant canonicalModel : String)
    (transitions : List TemporalTransition) (fairness : List TemporalFairness) :
    TemporalArtifact where
  target := "foundation-delivery-safety"
  property := "entity.progress"
  world := "smoke"
  variant := variant
  claimScope := "canonical-model-only"
  canonicalModel := canonicalModel
  resources := [{ identifier := "workflow", kind := "workflow" }]
  liveOnlyActions := ["crash-owner"]
  states := states
  initial := "unavailable"
  actions := actions
  transitions := transitions
  fairness := fairness
  progress := progress
  bounds := { maxTraceLength := 8 }

private def encodeState : Phase → String
  | .unavailable => "unavailable"
  | .ready => "ready"
  | .completed => "completed"

private def encodeAction : Action → String
  | .recover => "recover-owner"
  | .deliver => "progress-entity"

def soundView : TemporalView behavior () where
  artifact := artifact "sound"
    "Umpire3.Temporal.System.TaskDeliveryProgress.behavior"
    [recoveryTransition, deliveryTransition]
    [recoveryFairness, deliveryFairness]
  encodeState := encodeState
  encodeAction := encodeAction
  states_complete := by
    intro identifier member
    simp [artifact, states] at member
    rcases member with rfl | rfl | rfl
    · exact ⟨.unavailable, rfl⟩
    · exact ⟨.ready, rfl⟩
    · exact ⟨.completed, rfl⟩
  actions_complete := by
    intro identifier member
    simp [artifact, actions] at member
    rcases member with rfl | rfl
    · exact ⟨.deliver, rfl⟩
    · exact ⟨.recover, rfl⟩
  initial_exact := by
    intro state
    cases state <;> decide
  step_exact := by
    intro state action nextState
    cases state <;> cases action <;> cases nextState <;>
      simp [artifact, recoveryTransition, deliveryTransition, encodeState, encodeAction, Step]

def mutatedView : TemporalView mutatedBehavior () where
  artifact := artifact "delivery-fairness-removed"
    "Umpire3.Temporal.System.TaskDeliveryProgress.mutatedBehavior"
    [recoveryTransition]
    [recoveryFairness]
  encodeState := encodeState
  encodeAction := encodeAction
  states_complete := by
    intro identifier member
    simp [artifact, states] at member
    rcases member with rfl | rfl | rfl
    · exact ⟨.unavailable, rfl⟩
    · exact ⟨.ready, rfl⟩
    · exact ⟨.completed, rfl⟩
  actions_complete := by
    intro identifier member
    simp [artifact, actions] at member
    rcases member with rfl | rfl
    · exact ⟨.deliver, rfl⟩
    · exact ⟨.recover, rfl⟩
  initial_exact := by
    intro state
    cases state <;> decide
  step_exact := by
    intro state action nextState
    cases state <;> cases action <;> cases nextState <;>
      simp [artifact, recoveryTransition, encodeState, encodeAction, MutatedStep]

def soundExport : TemporalExport where
  view := resolved_temporal% soundView
  proof := some (resolved_theorem% progressUnderFairness)

def mutatedExport : TemporalExport where
  view := resolved_temporal% mutatedView
  proof := none

end Umpire3.Temporal.Targets.TaskDeliveryProgressTemporal
