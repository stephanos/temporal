import Temporal.Families.CallbackResponse.Feature
import Temporal.Families.CallbackResponse.System
import Umpire3.Refinement

namespace Umpire3.Temporal.Refinement.CallbackResponse

namespace Feature

abbrev State := Feature.CallbackResponse.State
abbrev Action := Feature.CallbackResponse.Action
abbrev behavior : Behavior Unit := Behavior.ofTransitionSystem Feature.CallbackResponse.model

end Feature

namespace System

abbrev State := System.CallbackResponse.State
abbrev Action := System.CallbackResponse.Action
abbrev behavior := System.CallbackResponse.behavior
abbrev mutatedBehavior := System.CallbackResponse.mutatedBehavior
abbrev next := System.CallbackResponse.next

end System

def registered : Feature.State :=
  Feature.CallbackResponse.initial.register .primary .primary .primary

def settled : Feature.State := registered.settle .primary .second true

def conflictedAfterSuccess : Feature.State :=
  Feature.CallbackResponse.consistentFinal.markConflict .primary

def project : System.State → Feature.State
  | .empty => Feature.CallbackResponse.initial
  | .registered => registered
  | .settled => settled
  | .responded => Feature.CallbackResponse.consistentFinal
  | .conflictingResponse => conflictedAfterSuccess

def Projects (systemState : System.State) (featureState : Feature.State) : Prop :=
  project systemState = featureState

def actionMap : System.Action → ActionEmission Feature.Action
  | .register => .one (.register .primary .primary .primary)
  | .settle => .one (.settleOperation .primary .second true)
  | .respond => .one (.recordResponse .primary .asyncSuccess .accepted .third)
  | .conflict => .one (.recordResponse .primary .failure .conflicting .third)

theorem registerStep : Feature.CallbackResponse.model.Step Feature.CallbackResponse.initial
    (.register .primary .primary .primary) registered := by
  change registered ∈ Feature.CallbackResponse.next Feature.CallbackResponse.initial
    (.register .primary .primary .primary)
  decide

theorem settleStep : Feature.CallbackResponse.model.Step registered
    (.settleOperation .primary .second true) settled := by
  change settled ∈ Feature.CallbackResponse.next registered
    (.settleOperation .primary .second true)
  decide

theorem responseStep : Feature.CallbackResponse.model.Step settled
    (.recordResponse .primary .asyncSuccess .accepted .third)
    Feature.CallbackResponse.consistentFinal := by
  change Feature.CallbackResponse.consistentFinal ∈ Feature.CallbackResponse.next settled
    (.recordResponse .primary .asyncSuccess .accepted .third)
  decide

theorem mappedRun {state action nextState} (step : nextState ∈ System.next state action) :
    Runs Feature.CallbackResponse.model (project state) (actionMap action).actions
      (project nextState) := by
  cases state <;> cases action <;> simp [System.next,
    Umpire3.Temporal.System.CallbackResponse.next] at step
  all_goals subst nextState
  · exact Runs.cons registerStep (Runs.nil _)
  · exact Runs.cons settleStep (Runs.nil _)
  · exact Runs.cons responseStep (Runs.nil _)

theorem initialProjects {systemState} (initialState : System.behavior.Initial () systemState) :
    ∃ featureState, Feature.behavior.Initial () featureState ∧ Projects systemState featureState := by
  subst systemState
  exact ⟨Feature.CallbackResponse.initial, rfl, rfl⟩

theorem stepSimulates : StepSimulation System.behavior Feature.behavior Projects actionMap := by
  intro _ systemState featureState action nextSystemState related systemStep
  subst featureState
  exact ⟨project nextSystemState, mappedRun systemStep, rfl⟩

def soundSimulation : SafetySimulation System.behavior Feature.behavior where
  Relates := Projects
  mapAction := actionMap
  initial := fun _ initialState => initialProjects initialState
  step := stepSimulates

theorem conflictViolatesResponseConsistency :
    ¬Feature.CallbackResponse.ResponseConsistency conflictedAfterSuccess := by decide

theorem mutationBreaksDeclaredSimulation :
    ¬StepSimulation System.mutatedBehavior Feature.behavior Projects actionMap := by
  intro simulation
  have transition : System.mutatedBehavior.Step () .responded .conflict .conflictingResponse := by
    decide
  rcases simulation .responded Feature.CallbackResponse.consistentFinal .conflict
      .conflictingResponse rfl transition with ⟨nextFeatureState, run, projects⟩
  have finalState : nextFeatureState = conflictedAfterSuccess := projects.symm
  subst nextFeatureState
  rcases run.firstStep with ⟨_, invalidStep⟩
  change _ ∈ Feature.CallbackResponse.next Feature.CallbackResponse.consistentFinal
    (.recordResponse .primary .failure .conflicting .third) at invalidStep
  have noNext : Feature.CallbackResponse.next Feature.CallbackResponse.consistentFinal
      (.recordResponse .primary .failure .conflicting .third) = [] := by decide
  rw [noNext] at invalidStep
  cases invalidStep

end Umpire3.Temporal.Refinement.CallbackResponse
