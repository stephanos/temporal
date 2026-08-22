import Temporal.Families.NexusProgress.Feature
import Temporal.Families.NexusProgress.System
import Umpire3.Refinement

namespace Umpire3.Temporal.Refinement.NexusProgress

namespace Feature

abbrev State := Feature.NexusProgress.State
abbrev Action := Feature.NexusProgress.Action
abbrev behavior : Behavior Unit := Behavior.ofTransitionSystem Feature.NexusProgress.model

end Feature

namespace System

abbrev State := System.NexusProgress.State
abbrev Action := System.NexusProgress.Action
abbrev behavior := System.NexusProgress.behavior
abbrev mutatedBehavior := System.NexusProgress.mutatedBehavior
abbrev next := System.NexusProgress.next

end System

def project : System.State → Feature.State
  | .idle => Feature.NexusProgress.initial
  | .scheduled => Feature.NexusProgress.scheduled
  | .retrying => Feature.NexusProgress.retrying
  | .waited => Feature.NexusProgress.waited
  | .settled => Feature.NexusProgress.settledFinal
  | .stuckAfterDeadline => Feature.NexusProgress.stuckFinal

def Projects (systemState : System.State) (featureState : Feature.State) : Prop :=
  project systemState = featureState

def actionMap : System.Action → ActionEmission Feature.Action
  | .schedule => .one .schedule
  | .failRetryably => .one .observeRetryableFailure
  | .elapseWithinDeadline | .exceedDeadline => .one .wait
  | .settle => .one .settle

theorem scheduleStep : Feature.NexusProgress.model.Step Feature.NexusProgress.initial
    .schedule Feature.NexusProgress.scheduled := by
  change Feature.NexusProgress.scheduled ∈ Feature.NexusProgress.next
    Feature.NexusProgress.initial .schedule
  decide

theorem retryableFailureStep : Feature.NexusProgress.model.Step Feature.NexusProgress.scheduled
    .observeRetryableFailure Feature.NexusProgress.retrying := by
  change Feature.NexusProgress.retrying ∈ Feature.NexusProgress.next
    Feature.NexusProgress.scheduled .observeRetryableFailure
  decide

theorem waitStep : Feature.NexusProgress.model.Step Feature.NexusProgress.retrying
    .wait Feature.NexusProgress.waited := by
  change Feature.NexusProgress.waited ∈ Feature.NexusProgress.next
    Feature.NexusProgress.retrying .wait
  decide

theorem settleStep : Feature.NexusProgress.model.Step Feature.NexusProgress.waited
    .settle Feature.NexusProgress.settledFinal := by
  change Feature.NexusProgress.settledFinal ∈ Feature.NexusProgress.next
    Feature.NexusProgress.waited .settle
  decide

theorem mappedRun {state action nextState} (step : nextState ∈ System.next state action) :
    Runs Feature.NexusProgress.model (project state) (actionMap action).actions
      (project nextState) := by
  cases state <;> cases action <;> simp [System.next,
    Umpire3.Temporal.System.NexusProgress.next] at step
  all_goals subst nextState
  · exact Runs.cons scheduleStep (Runs.nil _)
  · exact Runs.cons retryableFailureStep (Runs.nil _)
  · exact Runs.cons waitStep (Runs.nil _)
  · exact Runs.cons settleStep (Runs.nil _)

theorem initialProjects {systemState} (initialState : System.behavior.Initial () systemState) :
    ∃ featureState, Feature.behavior.Initial () featureState ∧ Projects systemState featureState := by
  subst systemState
  exact ⟨Feature.NexusProgress.initial, rfl, rfl⟩

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
  have transition : System.mutatedBehavior.Step () .waited .exceedDeadline
      .stuckAfterDeadline := by decide
  rcases simulation .waited Feature.NexusProgress.waited .exceedDeadline
      .stuckAfterDeadline rfl transition with ⟨nextFeatureState, run, projects⟩
  have finalState : nextFeatureState = Feature.NexusProgress.stuckFinal := projects.symm
  subst nextFeatureState
  rcases run.firstStep with ⟨_, invalidStep⟩
  change _ ∈ Feature.NexusProgress.next Feature.NexusProgress.waited .wait at invalidStep
  have noNext : Feature.NexusProgress.next Feature.NexusProgress.waited .wait = [] := by decide
  rw [noNext] at invalidStep
  cases invalidStep

end Umpire3.Temporal.Refinement.NexusProgress
