import Temporal.Families.NexusTimeout.Feature
import Temporal.Families.NexusTimeout.System
import Umpire3.Refinement

namespace Umpire3.Temporal.Refinement.NexusTimeout

namespace Feature

abbrev State := Feature.NexusTimeout.State
abbrev Action := Feature.NexusTimeout.Action
abbrev behavior : Behavior Unit := Behavior.ofTransitionSystem Feature.NexusTimeout.model

end Feature

namespace System

abbrev State := System.NexusTimeout.State
abbrev Action := System.NexusTimeout.Action
abbrev behavior := System.NexusTimeout.behavior
abbrev mutatedBehavior := System.NexusTimeout.mutatedBehavior
abbrev next := System.NexusTimeout.next

end System

def configured : Feature.State :=
  Feature.NexusTimeout.initial.setConfigured .primary true

def project : System.State → Feature.State
  | .idle => Feature.NexusTimeout.initial
  | .configured => configured
  | .timedOut => Feature.NexusTimeout.permittedFinal
  | .malformedTimeout => Feature.NexusTimeout.unsafeInvalidFinal

def Projects (systemState : System.State) (featureState : Feature.State) : Prop :=
  project systemState = featureState

def actionMap : System.Action → ActionEmission Feature.Action
  | .configure => .one (.configure .primary)
  | .recordTimeout =>
      .one (.recordTimeout .primary .primary .startToClose .operationTimedOut)
  | .recordMalformedTimeout =>
      .one (.recordTimeout .primary .primary .unspecified .unrelatedFailure)

theorem configureStep : Feature.NexusTimeout.model.Step Feature.NexusTimeout.initial
    (.configure .primary) configured := by
  change configured ∈ Feature.NexusTimeout.next Feature.NexusTimeout.initial (.configure .primary)
  decide

theorem timeoutStep : Feature.NexusTimeout.model.Step configured
    (.recordTimeout .primary .primary .startToClose .operationTimedOut)
    Feature.NexusTimeout.permittedFinal := by
  change Feature.NexusTimeout.permittedFinal ∈ Feature.NexusTimeout.next configured
    (.recordTimeout .primary .primary .startToClose .operationTimedOut)
  decide

theorem mappedRun {state action nextState} (step : nextState ∈ System.next state action) :
    Runs Feature.NexusTimeout.model (project state) (actionMap action).actions
      (project nextState) := by
  cases state <;> cases action <;> simp [System.next,
    Umpire3.Temporal.System.NexusTimeout.next] at step
  all_goals subst nextState
  · exact Runs.cons configureStep (Runs.nil _)
  · exact Runs.cons timeoutStep (Runs.nil _)

theorem initialProjects {systemState} (initialState : System.behavior.Initial () systemState) :
    ∃ featureState, Feature.behavior.Initial () featureState ∧ Projects systemState featureState := by
  subst systemState
  exact ⟨Feature.NexusTimeout.initial, rfl, rfl⟩

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
  have transition : System.mutatedBehavior.Step () .configured .recordMalformedTimeout
      .malformedTimeout := by decide
  rcases simulation .configured configured .recordMalformedTimeout .malformedTimeout rfl transition
      with ⟨nextFeatureState, run, projects⟩
  have finalState : nextFeatureState = Feature.NexusTimeout.unsafeInvalidFinal := projects.symm
  subst nextFeatureState
  rcases run.firstStep with ⟨_, invalidStep⟩
  change _ ∈ Feature.NexusTimeout.next configured
    (.recordTimeout .primary .primary .unspecified .unrelatedFailure) at invalidStep
  have noNext : Feature.NexusTimeout.next configured
      (.recordTimeout .primary .primary .unspecified .unrelatedFailure) = [] := by decide
  rw [noNext] at invalidStep
  cases invalidStep

end Umpire3.Temporal.Refinement.NexusTimeout
