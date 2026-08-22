import Temporal.Families.WorkflowRoutingIsolation.LineageFeature
import Temporal.Families.WorkflowRoutingIsolation.LineageSystem
import Umpire3.Refinement

namespace Umpire3.Temporal.Refinement.WorkflowLineage

namespace Feature

abbrev State := Feature.WorkflowLineage.State
abbrev Action := Feature.WorkflowLineage.Action
abbrev behavior : Behavior Unit := Behavior.ofTransitionSystem Feature.WorkflowLineage.model

end Feature


namespace System

abbrev State := System.WorkflowLineage.State
abbrev Action := System.WorkflowLineage.Action
abbrev behavior := System.WorkflowLineage.behavior
abbrev mutatedBehavior := System.WorkflowLineage.mutatedBehavior
abbrev next := System.WorkflowLineage.next

end System

def project : System.State → Feature.State
  | .empty => Feature.WorkflowLineage.initial
  | .continuationObserved => Feature.WorkflowLineage.continuationFinal
  | .resetObserved => Feature.WorkflowLineage.resetFinal
  | .invalidContinuation => Feature.WorkflowLineage.invalidContinuationFinal

def Projects (systemState : System.State) (featureState : Feature.State) : Prop :=
  project systemState = featureState

def actionMap : System.Action → ActionEmission Feature.Action
  | .observeContinuation =>
      .one (.observe .secondary .continuation .primary .secondary .primary)
  | .observeReset => .one (.observe .secondary .reset .primary .primary .primary)
  | .observeInvalidContinuation =>
      .one (.observe .secondary .continuation .primary .primary .primary)

theorem continuationStep : Feature.WorkflowLineage.model.Step Feature.WorkflowLineage.initial
    (.observe .secondary .continuation .primary .secondary .primary)
    Feature.WorkflowLineage.continuationFinal := by
  change Feature.WorkflowLineage.continuationFinal ∈ Feature.WorkflowLineage.next
    Feature.WorkflowLineage.initial
    (.observe .secondary .continuation .primary .secondary .primary)
  decide

theorem resetStep : Feature.WorkflowLineage.model.Step Feature.WorkflowLineage.initial
    (.observe .secondary .reset .primary .primary .primary)
    Feature.WorkflowLineage.resetFinal := by
  change Feature.WorkflowLineage.resetFinal ∈ Feature.WorkflowLineage.next
    Feature.WorkflowLineage.initial (.observe .secondary .reset .primary .primary .primary)
  decide

theorem mappedRun {state action nextState} (step : nextState ∈ System.next state action) :
    Runs Feature.WorkflowLineage.model (project state) (actionMap action).actions
      (project nextState) := by
  cases state <;> cases action <;> simp [System.next,
    Umpire3.Temporal.System.WorkflowLineage.next] at step
  all_goals subst nextState
  · exact Runs.cons continuationStep (Runs.nil _)
  · exact Runs.cons resetStep (Runs.nil _)

theorem initialProjects {systemState} (initialState : System.behavior.Initial () systemState) :
    ∃ featureState, Feature.behavior.Initial () featureState ∧ Projects systemState featureState := by
  subst systemState
  exact ⟨Feature.WorkflowLineage.initial, rfl, rfl⟩

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
  have transition : System.mutatedBehavior.Step () .empty .observeInvalidContinuation
      .invalidContinuation := by decide
  rcases simulation .empty Feature.WorkflowLineage.initial .observeInvalidContinuation
      .invalidContinuation rfl transition with ⟨nextFeatureState, run, projects⟩
  have finalState : nextFeatureState = Feature.WorkflowLineage.invalidContinuationFinal :=
    projects.symm
  subst nextFeatureState
  rcases run.firstStep with ⟨_, invalidStep⟩
  change _ ∈ Feature.WorkflowLineage.next Feature.WorkflowLineage.initial
    (.observe .secondary .continuation .primary .primary .primary) at invalidStep
  have noNext : Feature.WorkflowLineage.next Feature.WorkflowLineage.initial
      (.observe .secondary .continuation .primary .primary .primary) = [] := by decide
  rw [noNext] at invalidStep
  cases invalidStep

end Umpire3.Temporal.Refinement.WorkflowLineage
