import Temporal.Families.NexusClosure.Feature
import Temporal.Families.NexusClosure.System
import Umpire3.Refinement

namespace Umpire3.Temporal.Refinement.NexusClosure

namespace Feature

abbrev State := Feature.NexusClosure.State
abbrev Action := Feature.NexusClosure.Action
abbrev behavior : Behavior Unit := Behavior.ofTransitionSystem Feature.NexusClosure.model

end Feature

namespace System

abbrev State := System.NexusClosure.State
abbrev Action := System.NexusClosure.Action
abbrev behavior := System.NexusClosure.behavior
abbrev mutatedBehavior := System.NexusClosure.mutatedBehavior
abbrev next := System.NexusClosure.next

end System

def scheduled : Feature.State :=
  (Feature.NexusClosure.initial.setOperation .primary .scheduled).setCaller .primary true

def started : Feature.State := scheduled.setOperation .primary .started

def settled : Feature.State := started.setOperation .primary .succeeded

def project : System.State → Feature.State
  | .idle => Feature.NexusClosure.initial
  | .scheduled => scheduled
  | .started => started
  | .settled => settled
  | .closed => Feature.NexusClosure.permittedFinal
  | .closedWhileRunning => Feature.NexusClosure.unsafeFinal

def Projects (systemState : System.State) (featureState : Feature.State) : Prop :=
  project systemState = featureState

def actionMap : System.Action → ActionEmission Feature.Action
  | .schedule => .one (.schedule .primary)
  | .start => .one (.start .primary)
  | .settle => .one (.settle .primary .succeeded)
  | .close | .closeWhileRunning => .one (.closeWorkflow .completed)

theorem scheduleStep : Feature.NexusClosure.model.Step Feature.NexusClosure.initial
    (.schedule .primary) scheduled := by
  change scheduled ∈ Feature.NexusClosure.next Feature.NexusClosure.initial (.schedule .primary)
  decide

theorem startStep : Feature.NexusClosure.model.Step scheduled (.start .primary) started := by
  change started ∈ Feature.NexusClosure.next scheduled (.start .primary)
  decide

theorem settleStep : Feature.NexusClosure.model.Step started
    (.settle .primary .succeeded) settled := by
  change settled ∈ Feature.NexusClosure.next started (.settle .primary .succeeded)
  decide

theorem closeStep : Feature.NexusClosure.model.Step settled
    (.closeWorkflow .completed) Feature.NexusClosure.permittedFinal := by
  change Feature.NexusClosure.permittedFinal ∈ Feature.NexusClosure.next settled
    (.closeWorkflow .completed)
  decide

theorem mappedRun {state action nextState} (step : nextState ∈ System.next state action) :
    Runs Feature.NexusClosure.model (project state) (actionMap action).actions
      (project nextState) := by
  cases state <;> cases action <;> simp [System.next,
    Umpire3.Temporal.System.NexusClosure.next] at step
  all_goals subst nextState
  · exact Runs.cons scheduleStep (Runs.nil _)
  · exact Runs.cons startStep (Runs.nil _)
  · exact Runs.cons settleStep (Runs.nil _)
  · exact Runs.cons closeStep (Runs.nil _)

theorem initialProjects {systemState} (initialState : System.behavior.Initial () systemState) :
    ∃ featureState, Feature.behavior.Initial () featureState ∧ Projects systemState featureState := by
  subst systemState
  exact ⟨Feature.NexusClosure.initial, rfl, rfl⟩

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
  have transition : System.mutatedBehavior.Step () .started .closeWhileRunning
      .closedWhileRunning := by decide
  rcases simulation .started started .closeWhileRunning .closedWhileRunning rfl transition with
      ⟨nextFeatureState, run, projects⟩
  have finalState : nextFeatureState = Feature.NexusClosure.unsafeFinal := projects.symm
  subst nextFeatureState
  rcases run.firstStep with ⟨_, invalidStep⟩
  change _ ∈ Feature.NexusClosure.next started (.closeWorkflow .completed) at invalidStep
  have noNext : Feature.NexusClosure.next started (.closeWorkflow .completed) = [] := by decide
  rw [noNext] at invalidStep
  cases invalidStep

end Umpire3.Temporal.Refinement.NexusClosure
