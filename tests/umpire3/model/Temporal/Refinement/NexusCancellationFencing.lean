import Temporal.Feature.NexusCancellationFencing
import Temporal.System.NexusCancellationFencing
import Umpire3.Refinement

namespace Umpire3.Temporal.Refinement.NexusCancellationFencing

namespace Feature

abbrev State := Umpire3.Temporal.Feature.NexusCancellationFencing.State
abbrev Action := Umpire3.Temporal.Feature.NexusCancellationFencing.Action
abbrev initial := Umpire3.Temporal.Feature.NexusCancellationFencing.initial
abbrev successors := Umpire3.Temporal.Feature.NexusCancellationFencing.successors
abbrev behavior := Umpire3.Temporal.Feature.NexusCancellationFencing.behavior

end Feature

namespace System

abbrev State := Umpire3.Temporal.System.NexusCancellationFencing.State
abbrev Action := Umpire3.Temporal.System.NexusCancellationFencing.Action
abbrev initial := Umpire3.Temporal.System.NexusCancellationFencing.initial
abbrev next := Umpire3.Temporal.System.NexusCancellationFencing.next
abbrev behavior := Umpire3.Temporal.System.NexusCancellationFencing.behavior
abbrev mutatedNext := Umpire3.Temporal.System.NexusCancellationFencing.mutatedNext
abbrev mutatedBehavior := Umpire3.Temporal.System.NexusCancellationFencing.mutatedBehavior
abbrev afterDispatch := Umpire3.Temporal.System.NexusCancellationFencing.afterDispatch
abbrev afterCancellationAccepted :=
  Umpire3.Temporal.System.NexusCancellationFencing.afterCancellationAccepted
abbrev afterOwnershipChange :=
  Umpire3.Temporal.System.NexusCancellationFencing.afterOwnershipChange
abbrev afterCancellationCommit :=
  Umpire3.Temporal.System.NexusCancellationFencing.afterCancellationCommit
abbrev staleReturned := Umpire3.Temporal.System.NexusCancellationFencing.staleReturned
abbrev staleSuccess := Umpire3.Temporal.System.NexusCancellationFencing.staleSuccess

end System

def project : System.State → Feature.State
  | { lifecycle := .open, .. } => .active
  | { lifecycle := .cancellationAccepted, .. } => .cancellationAccepted
  | { lifecycle := .cancelled, .. } => .cancelled
  | { lifecycle := .succeeded, .. } => .succeeded

def Projects (systemState : System.State) (featureState : Feature.State) : Prop :=
  project systemState = featureState

def actionMap : System.Action → ActionEmission Feature.Action
  | .acceptCancellation => .one .acceptCancellation
  | .commitCancellation => .one .winCancellation
  | .persistSuccess => .one .completeSuccess
  | .dispatchTask | .acquireOwnership | .returnSuccess => .stutter

theorem initialProjects {world systemState}
    (initialState : System.behavior.Initial world systemState) :
    ∃ featureState, Feature.behavior.Initial world featureState ∧
      Projects systemState featureState := by
  cases world
  change systemState = System.initial at initialState
  subst systemState
  exact ⟨Feature.initial, rfl, rfl⟩

theorem stepSimulates : StepSimulation System.behavior Feature.behavior Projects actionMap := by
  intro world systemState featureState action nextSystemState related systemStep
  cases world
  change System.State at systemState nextSystemState
  change Feature.State at featureState
  change Projects systemState featureState at related
  subst featureState
  change nextSystemState ∈ System.next .smoke systemState action at systemStep
  cases action with
  | dispatchTask =>
      simp only [System.next, Umpire3.Temporal.System.NexusCancellationFencing.next] at systemStep
      split at systemStep
      · have nextState := List.mem_singleton.mp systemStep
        subst nextSystemState
        refine ⟨project systemState, ?_, ?_⟩
        · exact Runs.nil (model := Feature.behavior.at .smoke) (project systemState)
        · cases systemState with
          | mk lifecycle _ _ _ _ => cases lifecycle <;> rfl
      · simp at systemStep
  | acceptCancellation =>
      simp only [System.next, Umpire3.Temporal.System.NexusCancellationFencing.next] at systemStep
      split at systemStep
      · rename_i accepts
        have nextState := List.mem_singleton.mp systemStep
        subst nextSystemState
        refine ⟨.cancellationAccepted, ?_, rfl⟩
        have projected : project systemState = .active := by
          cases systemState
          simp_all [project]
        rw [projected]
        apply Runs.cons (next :=
          Umpire3.Temporal.Feature.NexusCancellationFencing.State.cancellationAccepted)
        · exact Umpire3.Temporal.Feature.NexusCancellationFencing.acceptCancellationStep .smoke
        · exact Runs.nil (model := Feature.behavior.at .smoke) .cancellationAccepted
      · simp at systemStep
  | acquireOwnership =>
      simp only [System.next, Umpire3.Temporal.System.NexusCancellationFencing.next] at systemStep
      split at systemStep
      · have nextState := List.mem_singleton.mp systemStep
        subst nextSystemState
        refine ⟨project systemState, ?_, ?_⟩
        · exact Runs.nil (model := Feature.behavior.at .smoke) (project systemState)
        · cases systemState with
          | mk lifecycle _ _ _ _ => cases lifecycle <;> rfl
      · simp at systemStep
  | commitCancellation =>
      simp only [System.next, Umpire3.Temporal.System.NexusCancellationFencing.next] at systemStep
      split at systemStep
      · rename_i commits
        have nextState := List.mem_singleton.mp systemStep
        subst nextSystemState
        refine ⟨.cancelled, ?_, rfl⟩
        have projected : project systemState = .cancellationAccepted := by
          cases systemState
          simp_all [project]
        rw [projected]
        apply Runs.cons (next :=
          Umpire3.Temporal.Feature.NexusCancellationFencing.State.cancelled)
        · exact Umpire3.Temporal.Feature.NexusCancellationFencing.winCancellationStep .smoke
        · exact Runs.nil (model := Feature.behavior.at .smoke) .cancelled
      · simp at systemStep
  | returnSuccess =>
      simp only [System.next, Umpire3.Temporal.System.NexusCancellationFencing.next] at systemStep
      split at systemStep
      · have nextState := List.mem_singleton.mp systemStep
        subst nextSystemState
        refine ⟨project systemState, ?_, ?_⟩
        · exact Runs.nil (model := Feature.behavior.at .smoke) (project systemState)
        · cases systemState with
          | mk lifecycle _ _ _ _ => cases lifecycle <;> rfl
      · simp at systemStep
  | persistSuccess =>
      simp only [System.next, Umpire3.Temporal.System.NexusCancellationFencing.next] at systemStep
      split at systemStep
      · rename_i persists
        have nextState := List.mem_singleton.mp systemStep
        subst nextSystemState
        refine ⟨.succeeded, ?_, rfl⟩
        rcases persists.2.2 with active | accepted
        · have projected : project systemState = .active := by
            cases systemState
            simp_all [project]
          rw [projected]
          apply Runs.cons (next :=
            Umpire3.Temporal.Feature.NexusCancellationFencing.State.succeeded)
          · exact Umpire3.Temporal.Feature.NexusCancellationFencing.completeActiveStep .smoke
          · exact Runs.nil (model := Feature.behavior.at .smoke) .succeeded
        · have projected : project systemState = .cancellationAccepted := by
            cases systemState
            simp_all [project]
          rw [projected]
          apply Runs.cons (next :=
            Umpire3.Temporal.Feature.NexusCancellationFencing.State.succeeded)
          · exact Umpire3.Temporal.Feature.NexusCancellationFencing.completeAcceptedStep .smoke
          · exact Runs.nil (model := Feature.behavior.at .smoke) .succeeded
      · simp at systemStep

def soundSimulation : SafetySimulation System.behavior Feature.behavior where
  Relates := Projects
  mapAction := actionMap
  initial := fun _ initialState => initialProjects initialState
  step := stepSimulates

theorem mutationBreaksDeclaredSimulation :
    ¬StepSimulation System.mutatedBehavior Feature.behavior Projects actionMap := by
  intro simulation
  have related : Projects System.staleReturned
      Umpire3.Temporal.Feature.NexusCancellationFencing.State.cancelled := by
    rfl
  have transition : System.mutatedBehavior.Step .smoke System.staleReturned
      .persistSuccess System.staleSuccess := by
    change System.staleSuccess ∈
      Umpire3.Temporal.System.NexusCancellationFencing.mutatedNext
        .smoke System.staleReturned .persistSuccess
    simp [Umpire3.Temporal.System.NexusCancellationFencing.mutatedNext,
      Umpire3.Temporal.System.NexusCancellationFencing.staleReturned,
      Umpire3.Temporal.System.NexusCancellationFencing.staleSuccess,
      Umpire3.Temporal.System.NexusCancellationFencing.afterCancellationCommit,
      Umpire3.Temporal.System.NexusCancellationFencing.afterOwnershipChange,
      Umpire3.Temporal.System.NexusCancellationFencing.afterCancellationAccepted,
      Umpire3.Temporal.System.NexusCancellationFencing.afterDispatch,
      Umpire3.Temporal.System.NexusCancellationFencing.initial]
  rcases simulation System.staleReturned
      Umpire3.Temporal.Feature.NexusCancellationFencing.State.cancelled .persistSuccess
      System.staleSuccess related transition with ⟨nextFeatureState, run, projects⟩
  change Feature.State at nextFeatureState
  change project System.staleSuccess = nextFeatureState at projects
  have nextState : nextFeatureState =
      Umpire3.Temporal.Feature.NexusCancellationFencing.State.succeeded := by
    calc
      nextFeatureState = project System.staleSuccess := projects.symm
      _ = Umpire3.Temporal.Feature.NexusCancellationFencing.State.succeeded := rfl
  subst nextFeatureState
  change Runs (Feature.behavior.at .smoke)
    Umpire3.Temporal.Feature.NexusCancellationFencing.State.cancelled
    [Umpire3.Temporal.Feature.NexusCancellationFencing.Action.completeSuccess]
    Umpire3.Temporal.Feature.NexusCancellationFencing.State.succeeded at run
  exact Umpire3.Temporal.Feature.NexusCancellationFencing.cancelledCannotComplete run

end Umpire3.Temporal.Refinement.NexusCancellationFencing
