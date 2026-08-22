import Temporal.Families.UpdateLifecycle.Feature
import Temporal.Families.UpdateLifecycle.System
import Umpire3.Refinement

namespace Umpire3.Temporal.Refinement.UpdateLifecycle

namespace Feature

abbrev State := Umpire3.Temporal.Feature.UpdateLifecycle.State
abbrev Action := Umpire3.Temporal.Feature.UpdateLifecycle.Action
abbrev behavior := Umpire3.Temporal.Feature.UpdateLifecycle.behavior
abbrev next := Umpire3.Temporal.Feature.UpdateLifecycle.next

end Feature

namespace System

abbrev State := System.UpdateLifecycle.State
abbrev Action := System.UpdateLifecycle.Action
abbrev behavior := System.UpdateLifecycle.behavior
abbrev mutatedBehavior := System.UpdateLifecycle.mutatedBehavior
abbrev next := System.UpdateLifecycle.next

end System

def project : System.State → Feature.State
  | .idle => .idle
  | .requested | .taskDispatched => .requested
  | .accepted => .accepted
  | .historyRecorded | .workflowTaskCompleted => .historyRecorded
  | .completed => .completed
  | .completedWithoutHistory => .completedWithoutHistory

def Projects (systemState : System.State) (featureState : Feature.State) : Prop :=
  project systemState = featureState

def actionMap : System.Action → ActionEmission Feature.Action
  | .start => .one .request
  | .dispatchTask | .completeWorkflowTask => .stutter
  | .accept => .one .accept
  | .recordHistory => .one .recordHistory
  | .complete => .one .complete
  | .completeWithoutHistory => .one .completeWithoutHistory

theorem requestStep : Feature.behavior.Step () .idle .request .requested := by decide
theorem acceptStep : Feature.behavior.Step () .requested .accept .accepted := by decide
theorem historyStep : Feature.behavior.Step () .accepted .recordHistory .historyRecorded := by decide
theorem completeStep : Feature.behavior.Step () .historyRecorded .complete .completed := by decide

theorem mappedRun {state action nextState} (step : nextState ∈ System.next state action) :
    Runs (Feature.behavior.at ()) (project state) (actionMap action).actions
      (project nextState) := by
  cases state <;> cases action <;> simp [System.next,
    Umpire3.Temporal.System.UpdateLifecycle.next] at step
  all_goals subst nextState
  · exact Runs.cons requestStep (Runs.nil _)
  · exact Runs.nil _
  · exact Runs.cons acceptStep (Runs.nil _)
  · exact Runs.cons historyStep (Runs.nil _)
  · exact Runs.nil _
  · exact Runs.cons completeStep (Runs.nil _)

theorem initialProjects {systemState} (initialState : System.behavior.Initial () systemState) :
    ∃ featureState, Feature.behavior.Initial () featureState ∧ Projects systemState featureState := by
  subst systemState
  exact ⟨Umpire3.Temporal.Feature.UpdateLifecycle.State.idle, rfl, rfl⟩

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
  have transition : System.mutatedBehavior.Step () .accepted .completeWithoutHistory
      .completedWithoutHistory := by decide
  rcases simulation .accepted .accepted .completeWithoutHistory .completedWithoutHistory rfl
      transition with ⟨nextFeatureState, run, projects⟩
  have finalState : nextFeatureState =
      Umpire3.Temporal.Feature.UpdateLifecycle.State.completedWithoutHistory := projects.symm
  subst nextFeatureState
  rcases run.firstStep with ⟨_, invalidStep⟩
  change _ ∈ Feature.next .accepted .completeWithoutHistory at invalidStep
  have noNext : Feature.next .accepted .completeWithoutHistory = [] := by decide
  rw [noNext] at invalidStep
  cases invalidStep

end Umpire3.Temporal.Refinement.UpdateLifecycle
