import Temporal.Families.NexusActivityLink.Feature
import Temporal.Families.NexusActivityLink.System
import Umpire3.Refinement

namespace Umpire3.Temporal.Refinement.NexusActivityLink

namespace Feature

abbrev State := Feature.NexusActivityLink.State
abbrev Action := Feature.NexusActivityLink.Action
abbrev behavior : Behavior Unit := Behavior.ofTransitionSystem Feature.NexusActivityLink.model

end Feature

namespace System

abbrev State := System.NexusActivityLink.State
abbrev Action := System.NexusActivityLink.Action
abbrev behavior := System.NexusActivityLink.behavior
abbrev mutatedBehavior := System.NexusActivityLink.mutatedBehavior
abbrev next := System.NexusActivityLink.next

end System

def operationObserved : Feature.State :=
  (Feature.NexusActivityLink.initial.setOperationObserved .primary true).setForward
    .primary (some .primary)

def project : System.State → Feature.State
  | .empty => Feature.NexusActivityLink.initial
  | .operationObserved => operationObserved
  | .linked => Feature.NexusActivityLink.matchingFinal
  | .oneSided => Feature.NexusActivityLink.oneSidedFinal

def Projects (systemState : System.State) (featureState : Feature.State) : Prop :=
  project systemState = featureState

def actionMap : System.Action → ActionEmission Feature.Action
  | .observeOperation => .one (.observeOperation .primary (some .primary))
  | .observeLinkedActivity => .one (.observeActivity .primary (some .primary))
  | .observeOneSidedActivity => .one (.observeActivity .primary none)

theorem operationStep : Feature.NexusActivityLink.model.Step Feature.NexusActivityLink.initial
    (.observeOperation .primary (some .primary)) operationObserved := by
  change operationObserved ∈ Feature.NexusActivityLink.next Feature.NexusActivityLink.initial
    (.observeOperation .primary (some .primary))
  decide

theorem activityStep : Feature.NexusActivityLink.model.Step operationObserved
    (.observeActivity .primary (some .primary)) Feature.NexusActivityLink.matchingFinal := by
  change Feature.NexusActivityLink.matchingFinal ∈ Feature.NexusActivityLink.next
    operationObserved (.observeActivity .primary (some .primary))
  decide

theorem mappedRun {state action nextState} (step : nextState ∈ System.next state action) :
    Runs Feature.NexusActivityLink.model (project state) (actionMap action).actions
      (project nextState) := by
  cases state <;> cases action <;> simp [System.next,
    Umpire3.Temporal.System.NexusActivityLink.next] at step
  all_goals subst nextState
  · exact Runs.cons operationStep (Runs.nil _)
  · exact Runs.cons activityStep (Runs.nil _)

theorem initialProjects {systemState} (initialState : System.behavior.Initial () systemState) :
    ∃ featureState, Feature.behavior.Initial () featureState ∧ Projects systemState featureState := by
  subst systemState
  exact ⟨Feature.NexusActivityLink.initial, rfl, rfl⟩

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
  have transition : System.mutatedBehavior.Step () .operationObserved .observeOneSidedActivity
      .oneSided := by decide
  rcases simulation .operationObserved operationObserved .observeOneSidedActivity .oneSided rfl
      transition with ⟨nextFeatureState, run, projects⟩
  have finalState : nextFeatureState = Feature.NexusActivityLink.oneSidedFinal := projects.symm
  subst nextFeatureState
  rcases run.firstStep with ⟨_, invalidStep⟩
  change _ ∈ Feature.NexusActivityLink.next operationObserved
    (.observeActivity .primary none) at invalidStep
  have noNext : Feature.NexusActivityLink.next operationObserved
      (.observeActivity .primary none) = [] := by decide
  rw [noNext] at invalidStep
  cases invalidStep

end Umpire3.Temporal.Refinement.NexusActivityLink
