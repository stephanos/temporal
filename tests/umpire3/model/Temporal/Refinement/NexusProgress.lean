import Temporal.Product.NexusProgress
import Temporal.System.NexusProgress
import Umpire3.Refinement

namespace Umpire3.Temporal.Refinement.NexusProgress

namespace Feature

abbrev State := Product.NexusProgress.State
abbrev Action := Product.NexusProgress.Action
abbrev behavior : Behavior Unit := Behavior.ofTransitionSystem Product.NexusProgress.model

end Feature

namespace System

abbrev State := System.NexusProgress.State
abbrev Action := System.NexusProgress.Action
abbrev behavior := System.NexusProgress.behavior
abbrev mutatedBehavior := System.NexusProgress.mutatedBehavior
abbrev next := System.NexusProgress.next

end System

def project : System.State → Feature.State
  | .idle => Product.NexusProgress.initial
  | .scheduled => Product.NexusProgress.scheduled
  | .retrying => Product.NexusProgress.retrying
  | .waited => Product.NexusProgress.waited
  | .settled => Product.NexusProgress.settledFinal
  | .stuckAfterDeadline => Product.NexusProgress.stuckFinal

def Projects (systemState : System.State) (featureState : Feature.State) : Prop :=
  project systemState = featureState

def actionMap : System.Action → ActionEmission Feature.Action
  | .schedule => .one .schedule
  | .failRetryably => .one .observeRetryableFailure
  | .elapseWithinDeadline | .exceedDeadline => .one .wait
  | .settle => .one .settle

theorem scheduleStep : Product.NexusProgress.model.Step Product.NexusProgress.initial
    .schedule Product.NexusProgress.scheduled := by
  change Product.NexusProgress.scheduled ∈ Product.NexusProgress.next
    Product.NexusProgress.initial .schedule
  decide

theorem retryableFailureStep : Product.NexusProgress.model.Step Product.NexusProgress.scheduled
    .observeRetryableFailure Product.NexusProgress.retrying := by
  change Product.NexusProgress.retrying ∈ Product.NexusProgress.next
    Product.NexusProgress.scheduled .observeRetryableFailure
  decide

theorem waitStep : Product.NexusProgress.model.Step Product.NexusProgress.retrying
    .wait Product.NexusProgress.waited := by
  change Product.NexusProgress.waited ∈ Product.NexusProgress.next
    Product.NexusProgress.retrying .wait
  decide

theorem settleStep : Product.NexusProgress.model.Step Product.NexusProgress.waited
    .settle Product.NexusProgress.settledFinal := by
  change Product.NexusProgress.settledFinal ∈ Product.NexusProgress.next
    Product.NexusProgress.waited .settle
  decide

theorem mappedRun {state action nextState} (step : nextState ∈ System.next state action) :
    Runs Product.NexusProgress.model (project state) (actionMap action).actions
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
  exact ⟨Product.NexusProgress.initial, rfl, rfl⟩

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
  rcases simulation .waited Product.NexusProgress.waited .exceedDeadline
      .stuckAfterDeadline rfl transition with ⟨nextFeatureState, run, projects⟩
  have finalState : nextFeatureState = Product.NexusProgress.stuckFinal := projects.symm
  subst nextFeatureState
  rcases run.firstStep with ⟨_, invalidStep⟩
  change _ ∈ Product.NexusProgress.next Product.NexusProgress.waited .wait at invalidStep
  have noNext : Product.NexusProgress.next Product.NexusProgress.waited .wait = [] := by decide
  rw [noNext] at invalidStep
  cases invalidStep

end Umpire3.Temporal.Refinement.NexusProgress
