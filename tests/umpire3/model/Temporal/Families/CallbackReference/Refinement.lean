import Temporal.Families.CallbackReference.Feature
import Temporal.Families.CallbackReference.System
import Umpire3.Refinement

namespace Umpire3.Temporal.Refinement.CallbackReference

namespace Feature

abbrev State := Feature.CallbackReference.State
abbrev Action := Feature.CallbackReference.Action
abbrev behavior : Behavior Unit := Behavior.ofTransitionSystem Feature.CallbackReference.model

end Feature

namespace System

abbrev State := System.CallbackReference.State
abbrev Action := System.CallbackReference.Action
abbrev behavior := System.CallbackReference.behavior
abbrev mutatedBehavior := System.CallbackReference.mutatedBehavior
abbrev next := System.CallbackReference.next

end System

def attachment : Feature.State := Feature.CallbackReference.initial.setAttachment
  .primary .primary .event .workflowStarted .first false

def project : System.State → Feature.State
  | .empty => Feature.CallbackReference.initial
  | .attachmentObserved => attachment
  | .matchingOperationObserved => Feature.CallbackReference.matchingFinal
  | .wrongOperationObserved => Feature.CallbackReference.wrongReferenceFinal

def Projects (systemState : System.State) (featureState : Feature.State) : Prop :=
  project systemState = featureState

def actionMap : System.Action → ActionEmission Feature.Action
  | .observeAttachment =>
      .one (.observeAttachment .primary .primary .event .workflowStarted .first false)
  | .observeMatchingOperation =>
      .one (.observeOperationStart .primary .primary .primary .event .workflowStarted .second false)
  | .observeWrongOperation =>
      .one (.observeOperationStart .primary .primary .secondary .request .optionsUpdated .second false)

theorem attachmentStep : Feature.CallbackReference.model.Step Feature.CallbackReference.initial
    (.observeAttachment .primary .primary .event .workflowStarted .first false) attachment := by
  change attachment ∈ Feature.CallbackReference.next Feature.CallbackReference.initial
    (.observeAttachment .primary .primary .event .workflowStarted .first false)
  decide

theorem matchingOperationStep : Feature.CallbackReference.model.Step attachment
    (.observeOperationStart .primary .primary .primary .event .workflowStarted .second false)
    Feature.CallbackReference.matchingFinal := by
  change Feature.CallbackReference.matchingFinal ∈ Feature.CallbackReference.next attachment
    (.observeOperationStart .primary .primary .primary .event .workflowStarted .second false)
  decide

theorem mappedRun {state action nextState} (step : nextState ∈ System.next state action) :
    Runs Feature.CallbackReference.model (project state) (actionMap action).actions
      (project nextState) := by
  cases state <;> cases action <;> simp [System.next,
    Umpire3.Temporal.System.CallbackReference.next] at step
  all_goals subst nextState
  · exact Runs.cons attachmentStep (Runs.nil _)
  · exact Runs.cons matchingOperationStep (Runs.nil _)

theorem initialProjects {systemState} (initialState : System.behavior.Initial () systemState) :
    ∃ featureState, Feature.behavior.Initial () featureState ∧ Projects systemState featureState := by
  subst systemState
  exact ⟨Feature.CallbackReference.initial, rfl, rfl⟩

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
  have transition : System.mutatedBehavior.Step () .attachmentObserved .observeWrongOperation
      .wrongOperationObserved := by decide
  rcases simulation .attachmentObserved attachment .observeWrongOperation .wrongOperationObserved
      rfl transition with ⟨nextFeatureState, run, projects⟩
  have finalState : nextFeatureState = Feature.CallbackReference.wrongReferenceFinal := projects.symm
  subst nextFeatureState
  rcases run.firstStep with ⟨_, invalidStep⟩
  change _ ∈ Feature.CallbackReference.next attachment
    (.observeOperationStart .primary .primary .secondary .request .optionsUpdated .second false)
      at invalidStep
  have noNext : Feature.CallbackReference.next attachment
      (.observeOperationStart .primary .primary .secondary .request .optionsUpdated .second false) =
      [] := by decide
  rw [noNext] at invalidStep
  cases invalidStep

end Umpire3.Temporal.Refinement.CallbackReference
