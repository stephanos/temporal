import Temporal.Families.WorkflowOwnership.Feature
import Temporal.Families.WorkflowOwnership.System
import Umpire3.Refinement

namespace Umpire3.Temporal.Refinement.WorkflowOwnership

namespace Feature

abbrev State := Feature.WorkflowOwnership.State
abbrev Action := Feature.WorkflowOwnership.Action
abbrev behavior : Behavior Unit := Behavior.ofTransitionSystem Feature.WorkflowOwnership.model

end Feature

namespace System

abbrev State := System.WorkflowOwnership.State
abbrev Action := System.WorkflowOwnership.Action
abbrev behavior := System.WorkflowOwnership.behavior
abbrev mutatedBehavior := System.WorkflowOwnership.mutatedBehavior
abbrev next := System.WorkflowOwnership.next
abbrev mutatedNext := System.WorkflowOwnership.mutatedNext

end System

def bootstrapped : Feature.State :=
  Feature.WorkflowOwnership.initial.setTaskEpoch .primary .first

def dispatched : Feature.State :=
  bootstrapped.setAttempt
    .primary .primary .first .dispatched

def failed : Feature.State := dispatched.setAttemptState .primary .failed

def rejected : Feature.State :=
  Feature.WorkflowOwnership.beforeRotation.setAttemptState .primary .rejected

def project : System.State → Feature.State
  | .idle => Feature.WorkflowOwnership.initial
  | .currentDispatched => dispatched
  | .currentFailed => failed
  | .ownerRotated => Feature.WorkflowOwnership.beforeRotation
  | .staleRejected => rejected
  | .currentCompleted => Feature.WorkflowOwnership.fencedFinal
  | .staleCompleted => Feature.WorkflowOwnership.staleCompletionFinal

def Projects (systemState : System.State) (featureState : Feature.State) : Prop :=
  project systemState = featureState

def actionMap : System.Action → ActionEmission Feature.Action
  | .dispatchCurrent => .many [.bootstrap .primary .first, .dispatch .primary .primary .first]
  | .failCurrent => .one (.fail .primary)
  | .rotateOwner => .one (.rotate .primary .first .second)
  | .rejectStale => .one (.rejectStale .primary)
  | .completeCurrent => .many [.dispatch .secondary .primary .second, .complete .secondary]
  | .completeStale => .one (.complete .primary)

theorem bootstrapStep : Feature.WorkflowOwnership.model.Step Feature.WorkflowOwnership.initial
    (.bootstrap .primary .first) bootstrapped := by
  change bootstrapped ∈ Feature.WorkflowOwnership.next Feature.WorkflowOwnership.initial
    (.bootstrap .primary .first)
  decide

theorem dispatchStep : Feature.WorkflowOwnership.model.Step bootstrapped
    (.dispatch .primary .primary .first) dispatched := by
  change dispatched ∈ Feature.WorkflowOwnership.next bootstrapped
    (.dispatch .primary .primary .first)
  decide

theorem failStep : Feature.WorkflowOwnership.model.Step dispatched
    (.fail .primary) failed := by
  change failed ∈ Feature.WorkflowOwnership.next dispatched (.fail .primary)
  decide

theorem rotateStep : Feature.WorkflowOwnership.model.Step failed
    (.rotate .primary .first .second) Feature.WorkflowOwnership.beforeRotation := by
  change Feature.WorkflowOwnership.beforeRotation ∈ Feature.WorkflowOwnership.next failed
    (.rotate .primary .first .second)
  decide

theorem rejectStep : Feature.WorkflowOwnership.model.Step Feature.WorkflowOwnership.beforeRotation
    (.rejectStale .primary) rejected := by
  change rejected ∈ Feature.WorkflowOwnership.next Feature.WorkflowOwnership.beforeRotation
    (.rejectStale .primary)
  decide

def secondaryDispatched : Feature.State :=
  rejected.setAttempt .secondary .primary .second .dispatched

theorem dispatchSecondaryStep : Feature.WorkflowOwnership.model.Step rejected
    (.dispatch .secondary .primary .second) secondaryDispatched := by
  change secondaryDispatched ∈ Feature.WorkflowOwnership.next rejected
    (.dispatch .secondary .primary .second)
  decide

theorem completeSecondaryStep : Feature.WorkflowOwnership.model.Step secondaryDispatched
    (.complete .secondary) Feature.WorkflowOwnership.fencedFinal := by
  change Feature.WorkflowOwnership.fencedFinal ∈ Feature.WorkflowOwnership.next secondaryDispatched
    (.complete .secondary)
  decide

theorem mappedRun {state action nextState} (step : nextState ∈ System.next state action) :
    Runs Feature.WorkflowOwnership.model (project state) (actionMap action).actions
      (project nextState) := by
  cases state <;> cases action <;> simp [System.next,
    Umpire3.Temporal.System.WorkflowOwnership.next] at step
  all_goals subst nextState
  · apply Runs.cons (next := bootstrapped)
    · exact bootstrapStep
    · apply Runs.cons
      · exact dispatchStep
      · exact Runs.nil _
  · apply Runs.cons
    · exact failStep
    · exact Runs.nil _
  · apply Runs.cons
    · exact rotateStep
    · exact Runs.nil _
  · apply Runs.cons
    · exact rejectStep
    · exact Runs.nil _
  · apply Runs.cons (next := secondaryDispatched)
    · exact dispatchSecondaryStep
    · apply Runs.cons
      · exact completeSecondaryStep
      · exact Runs.nil _

theorem initialProjects {systemState} (initialState : System.behavior.Initial () systemState) :
    ∃ featureState, Feature.behavior.Initial () featureState ∧ Projects systemState featureState := by
  subst systemState
  exact ⟨Feature.WorkflowOwnership.initial, rfl, rfl⟩

theorem stepSimulates : StepSimulation System.behavior Feature.behavior Projects actionMap := by
  intro _ systemState featureState action nextSystemState related systemStep
  subst featureState
  exact ⟨project nextSystemState, mappedRun systemStep, rfl⟩

def soundSimulation : SafetySimulation System.behavior Feature.behavior where
  Relates := Projects
  mapAction := actionMap
  initial := fun _ initialState => initialProjects initialState
  step := stepSimulates

theorem mutationBreaksDeclaredSimulation :
    ¬StepSimulation System.mutatedBehavior Feature.behavior Projects actionMap := by
  intro simulation
  have related : Projects .ownerRotated Feature.WorkflowOwnership.beforeRotation := rfl
  have transition : System.mutatedBehavior.Step () .ownerRotated .completeStale .staleCompleted := by
    decide
  rcases simulation .ownerRotated Feature.WorkflowOwnership.beforeRotation .completeStale
      .staleCompleted related transition with ⟨nextFeatureState, run, projects⟩
  have finalState : nextFeatureState = Feature.WorkflowOwnership.staleCompletionFinal := by
    exact projects.symm
  subst nextFeatureState
  rcases run.firstStep with ⟨_, invalidStep⟩
  change _ ∈ Feature.WorkflowOwnership.next Feature.WorkflowOwnership.beforeRotation
    (.complete .primary) at invalidStep
  have noNext : Feature.WorkflowOwnership.next Feature.WorkflowOwnership.beforeRotation
      (.complete .primary) = [] := by decide
  rw [noNext] at invalidStep
  cases invalidStep

end Umpire3.Temporal.Refinement.WorkflowOwnership
