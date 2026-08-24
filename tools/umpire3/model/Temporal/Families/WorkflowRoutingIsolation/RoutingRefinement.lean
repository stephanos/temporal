import Temporal.Families.WorkflowRoutingIsolation.RoutingFeature
import Temporal.Families.WorkflowRoutingIsolation.RoutingSystem
import Umpire3.Refinement

namespace Umpire3.Temporal.Refinement.WorkflowRouting

namespace Feature

abbrev State := Feature.WorkflowRouting.State
abbrev Action := Feature.WorkflowRouting.Action
abbrev behavior : Behavior Unit := Behavior.ofTransitionSystem Feature.WorkflowRouting.model

end Feature

namespace System

abbrev State := System.WorkflowRouting.State
abbrev Action := System.WorkflowRouting.Action
abbrev behavior := System.WorkflowRouting.behavior
abbrev mutatedBehavior := System.WorkflowRouting.mutatedBehavior
abbrev next := System.WorkflowRouting.next

end System

def taskRouted : Feature.State :=
  Feature.WorkflowRouting.initial.setTaskRoute .primary .primary

def matchingPollerRegistered : Feature.State :=
  taskRouted.setPollerRoute .primary .primary

def crossingPollerRegistered : Feature.State :=
  taskRouted.setPollerRoute .primary .secondary

def project : System.State → Feature.State
  | .empty => Feature.WorkflowRouting.initial
  | .taskRouted => taskRouted
  | .matchingPollerRegistered => matchingPollerRegistered
  | .crossingPollerRegistered => crossingPollerRegistered
  | .matchingReservation => Feature.WorkflowRouting.matchingFinal
  | .crossingReservation => Feature.WorkflowRouting.crossingFinal

def Projects (systemState : System.State) (featureState : Feature.State) : Prop :=
  project systemState = featureState

def actionMap : System.Action → ActionEmission Feature.Action
  | .assignTask => .one (.assignTask .primary .primary)
  | .registerMatchingPoller => .one (.registerPoller .primary .primary)
  | .registerCrossingPoller => .one (.registerPoller .primary .secondary)
  | .reserveMatching | .reserveCrossing => .one (.reserve .primary .primary .primary)

theorem assignTaskStep : Feature.WorkflowRouting.model.Step Feature.WorkflowRouting.initial
    (.assignTask .primary .primary) taskRouted := by
  change taskRouted ∈ Feature.WorkflowRouting.next Feature.WorkflowRouting.initial
    (.assignTask .primary .primary)
  decide

theorem matchingPollerStep : Feature.WorkflowRouting.model.Step taskRouted
    (.registerPoller .primary .primary) matchingPollerRegistered := by
  change matchingPollerRegistered ∈ Feature.WorkflowRouting.next taskRouted
    (.registerPoller .primary .primary)
  decide

theorem crossingPollerStep : Feature.WorkflowRouting.model.Step taskRouted
    (.registerPoller .primary .secondary) crossingPollerRegistered := by
  change crossingPollerRegistered ∈ Feature.WorkflowRouting.next taskRouted
    (.registerPoller .primary .secondary)
  decide

theorem matchingReservationStep : Feature.WorkflowRouting.model.Step matchingPollerRegistered
    (.reserve .primary .primary .primary) Feature.WorkflowRouting.matchingFinal := by
  change Feature.WorkflowRouting.matchingFinal ∈ Feature.WorkflowRouting.next
    matchingPollerRegistered (.reserve .primary .primary .primary)
  decide

theorem mappedRun {state action nextState} (step : nextState ∈ System.next state action) :
    Runs Feature.WorkflowRouting.model (project state) (actionMap action).actions
      (project nextState) := by
  cases state <;> cases action <;> simp [System.next,
    Umpire3.Temporal.System.WorkflowRouting.next] at step
  all_goals subst nextState
  · exact Runs.cons assignTaskStep (Runs.nil _)
  · exact Runs.cons matchingPollerStep (Runs.nil _)
  · exact Runs.cons crossingPollerStep (Runs.nil _)
  · exact Runs.cons matchingReservationStep (Runs.nil _)

theorem initialProjects {systemState} (initialState : System.behavior.Initial () systemState) :
    ∃ featureState, Feature.behavior.Initial () featureState ∧ Projects systemState featureState := by
  subst systemState
  exact ⟨Feature.WorkflowRouting.initial, rfl, rfl⟩

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
  have transition : System.mutatedBehavior.Step () .crossingPollerRegistered .reserveCrossing
      .crossingReservation := by decide
  rcases simulation .crossingPollerRegistered crossingPollerRegistered .reserveCrossing
      .crossingReservation rfl transition with ⟨nextFeatureState, run, projects⟩
  have finalState : nextFeatureState = Feature.WorkflowRouting.crossingFinal := projects.symm
  subst nextFeatureState
  rcases run.firstStep with ⟨_, invalidStep⟩
  change _ ∈ Feature.WorkflowRouting.next crossingPollerRegistered
    (.reserve .primary .primary .primary) at invalidStep
  have noNext : Feature.WorkflowRouting.next crossingPollerRegistered
      (.reserve .primary .primary .primary) = [] := by decide
  rw [noNext] at invalidStep
  cases invalidStep

end Umpire3.Temporal.Refinement.WorkflowRouting
