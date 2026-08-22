import Temporal.Families.WorkflowProgress.Feature
import Temporal.Families.WorkflowProgress.System
import Umpire3.Refinement

namespace Umpire3.Temporal.Refinement.WorkflowProgress

namespace Feature

abbrev State := Feature.WorkflowProgress.State
abbrev Action := Feature.WorkflowProgress.Action
abbrev behavior : Behavior Unit := Behavior.ofTransitionSystem Feature.WorkflowProgress.model

end Feature

namespace System

abbrev State := System.WorkflowProgress.State
abbrev Action := System.WorkflowProgress.Action
abbrev behavior := System.WorkflowProgress.behavior
abbrev mutatedBehavior := System.WorkflowProgress.mutatedBehavior
abbrev next := System.WorkflowProgress.next

end System

def queued : Feature.State :=
  (Feature.WorkflowProgress.initial.enqueue .primary .primary).setEntityState .primary .pending

def dispatched : Feature.State := Feature.WorkflowProgress.waited.dispatch .primary .primary

def wrongEntityAfterWait : Feature.State := dispatched.complete .primary .secondary

def project : System.State → Feature.State
  | .idle => Feature.WorkflowProgress.initial
  | .queued => queued
  | .workerAvailable => Feature.WorkflowProgress.queuedAvailable
  | .waited => Feature.WorkflowProgress.waited
  | .dispatched => dispatched
  | .completed => Feature.WorkflowProgress.progressedFinal
  | .starved => Feature.WorkflowProgress.starvedFinal
  | .wrongEntityCompleted => wrongEntityAfterWait

def Projects (systemState : System.State) (featureState : Feature.State) : Prop :=
  project systemState = featureState

def actionMap : System.Action → ActionEmission Feature.Action
  | .enqueue => .one (.enqueue .primary .primary)
  | .makeWorkerAvailable => .one (.makeWorkerAvailable .primary)
  | .wait | .waitAgain => .one (.wait .primary)
  | .dispatch => .one (.dispatch .primary .primary)
  | .complete => .one (.complete .primary .primary)
  | .completeWrongEntity => .one (.complete .primary .secondary)

theorem enqueueStep : Feature.WorkflowProgress.model.Step Feature.WorkflowProgress.initial
    (.enqueue .primary .primary) queued := by
  change queued ∈ Feature.WorkflowProgress.next Feature.WorkflowProgress.initial
    (.enqueue .primary .primary)
  decide

theorem workerStep : Feature.WorkflowProgress.model.Step queued
    (.makeWorkerAvailable .primary) Feature.WorkflowProgress.queuedAvailable := by
  change Feature.WorkflowProgress.queuedAvailable ∈ Feature.WorkflowProgress.next queued
    (.makeWorkerAvailable .primary)
  decide

theorem waitStep : Feature.WorkflowProgress.model.Step Feature.WorkflowProgress.queuedAvailable
    (.wait .primary) Feature.WorkflowProgress.waited := by
  change Feature.WorkflowProgress.waited ∈ Feature.WorkflowProgress.next
    Feature.WorkflowProgress.queuedAvailable (.wait .primary)
  decide

theorem dispatchStep : Feature.WorkflowProgress.model.Step Feature.WorkflowProgress.waited
    (.dispatch .primary .primary) dispatched := by
  change dispatched ∈ Feature.WorkflowProgress.next Feature.WorkflowProgress.waited
    (.dispatch .primary .primary)
  decide

theorem completeStep : Feature.WorkflowProgress.model.Step dispatched
    (.complete .primary .primary) Feature.WorkflowProgress.progressedFinal := by
  change Feature.WorkflowProgress.progressedFinal ∈ Feature.WorkflowProgress.next dispatched
    (.complete .primary .primary)
  decide

theorem mappedRun {state action nextState} (step : nextState ∈ System.next state action) :
    Runs Feature.WorkflowProgress.model (project state) (actionMap action).actions
      (project nextState) := by
  cases state <;> cases action <;> simp [System.next,
    Umpire3.Temporal.System.WorkflowProgress.next] at step
  all_goals subst nextState
  · exact Runs.cons enqueueStep (Runs.nil _)
  · exact Runs.cons workerStep (Runs.nil _)
  · exact Runs.cons waitStep (Runs.nil _)
  · exact Runs.cons dispatchStep (Runs.nil _)
  · exact Runs.cons completeStep (Runs.nil _)

theorem initialProjects {systemState} (initialState : System.behavior.Initial () systemState) :
    ∃ featureState, Feature.behavior.Initial () featureState ∧ Projects systemState featureState := by
  subst systemState
  exact ⟨Feature.WorkflowProgress.initial, rfl, rfl⟩

theorem stepSimulates : StepSimulation System.behavior Feature.behavior Projects actionMap := by
  intro _ systemState featureState action nextSystemState related systemStep
  subst featureState
  exact ⟨project nextSystemState, mappedRun systemStep, rfl⟩

def soundSimulation : SafetySimulation System.behavior Feature.behavior where
  Relates := Projects
  mapAction := actionMap
  initial := fun _ initialState => initialProjects initialState
  step := stepSimulates

theorem starvationMutationBreaksDeclaredSimulation :
    ¬StepSimulation System.mutatedBehavior Feature.behavior Projects actionMap := by
  intro simulation
  have transition : System.mutatedBehavior.Step () .waited .waitAgain .starved := by decide
  rcases simulation .waited Feature.WorkflowProgress.waited .waitAgain .starved rfl transition
      with ⟨nextFeatureState, run, projects⟩
  have finalState : nextFeatureState = Feature.WorkflowProgress.starvedFinal := projects.symm
  subst nextFeatureState
  rcases run.firstStep with ⟨_, invalidStep⟩
  change _ ∈ Feature.WorkflowProgress.next Feature.WorkflowProgress.waited (.wait .primary)
    at invalidStep
  have noNext : Feature.WorkflowProgress.next Feature.WorkflowProgress.waited (.wait .primary) =
      [] := by decide
  rw [noNext] at invalidStep
  cases invalidStep

theorem entityMutationBreaksDeclaredSimulation :
    ¬StepSimulation System.mutatedBehavior Feature.behavior Projects actionMap := by
  intro simulation
  have transition : System.mutatedBehavior.Step () .dispatched .completeWrongEntity
      .wrongEntityCompleted := by decide
  rcases simulation .dispatched dispatched .completeWrongEntity .wrongEntityCompleted rfl transition
      with ⟨nextFeatureState, run, projects⟩
  have finalState : nextFeatureState = wrongEntityAfterWait := projects.symm
  subst nextFeatureState
  rcases run.firstStep with ⟨_, invalidStep⟩
  change _ ∈ Feature.WorkflowProgress.next dispatched (.complete .primary .secondary)
    at invalidStep
  have noNext : Feature.WorkflowProgress.next dispatched (.complete .primary .secondary) = [] := by
    decide
  rw [noNext] at invalidStep
  cases invalidStep

end Umpire3.Temporal.Refinement.WorkflowProgress
