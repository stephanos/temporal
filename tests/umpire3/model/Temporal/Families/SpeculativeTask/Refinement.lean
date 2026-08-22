import Temporal.Families.SpeculativeTask.Feature
import Temporal.Families.SpeculativeTask.System
import Umpire3.Refinement

namespace Umpire3.Temporal.Refinement.SpeculativeTask

namespace Feature

abbrev State := Feature.SpeculativeTask.State
abbrev Action := Feature.SpeculativeTask.Action
abbrev behavior : Behavior Unit := Behavior.ofTransitionSystem Feature.SpeculativeTask.model

end Feature

namespace System

abbrev State := System.SpeculativeTask.State
abbrev Action := System.SpeculativeTask.Action
abbrev behavior := System.SpeculativeTask.behavior
abbrev mutatedBehavior := System.SpeculativeTask.mutatedBehavior
abbrev next := System.SpeculativeTask.next

end System

def requested : Feature.State :=
  Feature.SpeculativeTask.initial.setUpdateState .primary .pending

def speculative : Feature.State := requested.setTask .primary .primary .speculative

def project : System.State → Feature.State
  | .empty => Feature.SpeculativeTask.initial
  | .updatePending => requested
  | .taskSpeculative => speculative
  | .taskCommitted => Feature.SpeculativeTask.committedFinal
  | .orphanedTask => Feature.SpeculativeTask.orphanedFinal

def Projects (systemState : System.State) (featureState : Feature.State) : Prop :=
  project systemState = featureState

def actionMap : System.Action → ActionEmission Feature.Action
  | .requestUpdate => .one (.requestUpdate .primary)
  | .createTask => .one (.create .primary .primary)
  | .commitTask => .one (.commit .primary)
  | .createOrphan => .one (.create .primary .secondary)

theorem requestStep : Feature.SpeculativeTask.model.Step Feature.SpeculativeTask.initial
    (.requestUpdate .primary) requested := by
  change requested ∈ Feature.SpeculativeTask.next Feature.SpeculativeTask.initial
    (.requestUpdate .primary)
  decide

theorem createStep : Feature.SpeculativeTask.model.Step requested
    (.create .primary .primary) speculative := by
  change speculative ∈ Feature.SpeculativeTask.next requested (.create .primary .primary)
  decide

theorem commitStep : Feature.SpeculativeTask.model.Step speculative
    (.commit .primary) Feature.SpeculativeTask.committedFinal := by
  change Feature.SpeculativeTask.committedFinal ∈ Feature.SpeculativeTask.next speculative
    (.commit .primary)
  decide

theorem mappedRun {state action nextState} (step : nextState ∈ System.next state action) :
    Runs Feature.SpeculativeTask.model (project state) (actionMap action).actions
      (project nextState) := by
  cases state <;> cases action <;> simp [System.next,
    Umpire3.Temporal.System.SpeculativeTask.next] at step
  all_goals subst nextState
  · exact Runs.cons requestStep (Runs.nil _)
  · exact Runs.cons createStep (Runs.nil _)
  · exact Runs.cons commitStep (Runs.nil _)

theorem initialProjects {systemState} (initialState : System.behavior.Initial () systemState) :
    ∃ featureState, Feature.behavior.Initial () featureState ∧ Projects systemState featureState := by
  subst systemState
  exact ⟨Feature.SpeculativeTask.initial, rfl, rfl⟩

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
  have transition : System.mutatedBehavior.Step () .empty .createOrphan .orphanedTask := by decide
  rcases simulation .empty Feature.SpeculativeTask.initial .createOrphan .orphanedTask rfl
      transition with ⟨nextFeatureState, run, projects⟩
  have finalState : nextFeatureState = Feature.SpeculativeTask.orphanedFinal := projects.symm
  subst nextFeatureState
  rcases run.firstStep with ⟨_, invalidStep⟩
  change _ ∈ Feature.SpeculativeTask.next Feature.SpeculativeTask.initial
    (.create .primary .secondary) at invalidStep
  have noNext : Feature.SpeculativeTask.next Feature.SpeculativeTask.initial
      (.create .primary .secondary) = [] := by decide
  rw [noNext] at invalidStep
  cases invalidStep

end Umpire3.Temporal.Refinement.SpeculativeTask
