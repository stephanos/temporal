import Temporal.Product.CallbackReference
import Temporal.Product.CallbackResponse
import Temporal.Product.NexusActivityLink
import Temporal.Product.NexusClosure
import Temporal.Product.NexusTimeout
import Temporal.Product.SpeculativeTask
import Temporal.Product.Update
import Temporal.Product.WorkflowLineage
import Temporal.Product.WorkflowOwnership
import Temporal.Product.WorkflowProgress
import Temporal.Product.WorkflowRouting
import Temporal.Feature.UpdateLifecycle
import Temporal.System.MigratedFamilies
import Umpire3.Refinement

namespace Umpire3.Temporal.Refinement.MigratedFamilies

namespace WorkflowOwnership

namespace Feature

abbrev State := Product.WorkflowOwnership.State
abbrev Action := Product.WorkflowOwnership.Action
abbrev behavior : Behavior Unit := Behavior.ofTransitionSystem Product.WorkflowOwnership.model

end Feature

namespace System

abbrev State := System.MigratedFamilies.WorkflowOwnership.State
abbrev Action := System.MigratedFamilies.WorkflowOwnership.Action
abbrev behavior := System.MigratedFamilies.WorkflowOwnership.behavior
abbrev mutatedBehavior := System.MigratedFamilies.WorkflowOwnership.mutatedBehavior
abbrev next := System.MigratedFamilies.WorkflowOwnership.next
abbrev mutatedNext := System.MigratedFamilies.WorkflowOwnership.mutatedNext

end System

def bootstrapped : Feature.State :=
  Product.WorkflowOwnership.initial.setTaskEpoch .primary .first

def dispatched : Feature.State :=
  bootstrapped.setAttempt
    .primary .primary .first .dispatched

def failed : Feature.State := dispatched.setAttemptState .primary .failed

def rejected : Feature.State :=
  Product.WorkflowOwnership.beforeRotation.setAttemptState .primary .rejected

def project : System.State → Feature.State
  | .idle => Product.WorkflowOwnership.initial
  | .currentDispatched => dispatched
  | .currentFailed => failed
  | .ownerRotated => Product.WorkflowOwnership.beforeRotation
  | .staleRejected => rejected
  | .currentCompleted => Product.WorkflowOwnership.fencedFinal
  | .staleCompleted => Product.WorkflowOwnership.staleCompletionFinal

def Projects (systemState : System.State) (featureState : Feature.State) : Prop :=
  project systemState = featureState

def actionMap : System.Action → ActionEmission Feature.Action
  | .dispatchCurrent => .many [.bootstrap .primary .first, .dispatch .primary .primary .first]
  | .failCurrent => .one (.fail .primary)
  | .rotateOwner => .one (.rotate .primary .first .second)
  | .rejectStale => .one (.rejectStale .primary)
  | .completeCurrent => .many [.dispatch .secondary .primary .second, .complete .secondary]
  | .completeStale => .one (.complete .primary)

theorem bootstrapStep : Product.WorkflowOwnership.model.Step Product.WorkflowOwnership.initial
    (.bootstrap .primary .first) bootstrapped := by
  change bootstrapped ∈ Product.WorkflowOwnership.next Product.WorkflowOwnership.initial
    (.bootstrap .primary .first)
  decide

theorem dispatchStep : Product.WorkflowOwnership.model.Step bootstrapped
    (.dispatch .primary .primary .first) dispatched := by
  change dispatched ∈ Product.WorkflowOwnership.next bootstrapped
    (.dispatch .primary .primary .first)
  decide

theorem failStep : Product.WorkflowOwnership.model.Step dispatched
    (.fail .primary) failed := by
  change failed ∈ Product.WorkflowOwnership.next dispatched (.fail .primary)
  decide

theorem rotateStep : Product.WorkflowOwnership.model.Step failed
    (.rotate .primary .first .second) Product.WorkflowOwnership.beforeRotation := by
  change Product.WorkflowOwnership.beforeRotation ∈ Product.WorkflowOwnership.next failed
    (.rotate .primary .first .second)
  decide

theorem rejectStep : Product.WorkflowOwnership.model.Step Product.WorkflowOwnership.beforeRotation
    (.rejectStale .primary) rejected := by
  change rejected ∈ Product.WorkflowOwnership.next Product.WorkflowOwnership.beforeRotation
    (.rejectStale .primary)
  decide

def secondaryDispatched : Feature.State :=
  rejected.setAttempt .secondary .primary .second .dispatched

theorem dispatchSecondaryStep : Product.WorkflowOwnership.model.Step rejected
    (.dispatch .secondary .primary .second) secondaryDispatched := by
  change secondaryDispatched ∈ Product.WorkflowOwnership.next rejected
    (.dispatch .secondary .primary .second)
  decide

theorem completeSecondaryStep : Product.WorkflowOwnership.model.Step secondaryDispatched
    (.complete .secondary) Product.WorkflowOwnership.fencedFinal := by
  change Product.WorkflowOwnership.fencedFinal ∈ Product.WorkflowOwnership.next secondaryDispatched
    (.complete .secondary)
  decide

theorem mappedRun {state action nextState} (step : nextState ∈ System.next state action) :
    Runs Product.WorkflowOwnership.model (project state) (actionMap action).actions
      (project nextState) := by
  cases state <;> cases action <;> simp [System.next,
    Umpire3.Temporal.System.MigratedFamilies.WorkflowOwnership.next] at step
  all_goals subst nextState
  · apply Runs.cons (next := bootstrapped)
    · exact bootstrapStep
    · apply Runs.cons
      · exact dispatchStep
      · exact Runs.nil _
  · apply Runs.cons
    · exact failStep
    · exact Runs.nil _
  · apply Runs.cons
    · exact rotateStep
    · exact Runs.nil _
  · apply Runs.cons
    · exact rejectStep
    · exact Runs.nil _
  · apply Runs.cons (next := secondaryDispatched)
    · exact dispatchSecondaryStep
    · apply Runs.cons
      · exact completeSecondaryStep
      · exact Runs.nil _

theorem initialProjects {systemState} (initialState : System.behavior.Initial () systemState) :
    ∃ featureState, Feature.behavior.Initial () featureState ∧ Projects systemState featureState := by
  subst systemState
  exact ⟨Product.WorkflowOwnership.initial, rfl, rfl⟩

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
  have related : Projects .ownerRotated Product.WorkflowOwnership.beforeRotation := rfl
  have transition : System.mutatedBehavior.Step () .ownerRotated .completeStale .staleCompleted := by
    decide
  rcases simulation .ownerRotated Product.WorkflowOwnership.beforeRotation .completeStale
      .staleCompleted related transition with ⟨nextFeatureState, run, projects⟩
  have finalState : nextFeatureState = Product.WorkflowOwnership.staleCompletionFinal := by
    exact projects.symm
  subst nextFeatureState
  rcases run.firstStep with ⟨_, invalidStep⟩
  change _ ∈ Product.WorkflowOwnership.next Product.WorkflowOwnership.beforeRotation
    (.complete .primary) at invalidStep
  have noNext : Product.WorkflowOwnership.next Product.WorkflowOwnership.beforeRotation
      (.complete .primary) = [] := by decide
  rw [noNext] at invalidStep
  cases invalidStep

end WorkflowOwnership

namespace WorkflowLineage

namespace Feature

abbrev State := Product.WorkflowLineage.State
abbrev Action := Product.WorkflowLineage.Action
abbrev behavior : Behavior Unit := Behavior.ofTransitionSystem Product.WorkflowLineage.model

end Feature


namespace System

abbrev State := System.MigratedFamilies.WorkflowLineage.State
abbrev Action := System.MigratedFamilies.WorkflowLineage.Action
abbrev behavior := System.MigratedFamilies.WorkflowLineage.behavior
abbrev mutatedBehavior := System.MigratedFamilies.WorkflowLineage.mutatedBehavior
abbrev next := System.MigratedFamilies.WorkflowLineage.next

end System

def project : System.State → Feature.State
  | .empty => Product.WorkflowLineage.initial
  | .continuationObserved => Product.WorkflowLineage.continuationFinal
  | .resetObserved => Product.WorkflowLineage.resetFinal
  | .invalidContinuation => Product.WorkflowLineage.invalidContinuationFinal

def Projects (systemState : System.State) (featureState : Feature.State) : Prop :=
  project systemState = featureState

def actionMap : System.Action → ActionEmission Feature.Action
  | .observeContinuation =>
      .one (.observe .secondary .continuation .primary .secondary .primary)
  | .observeReset => .one (.observe .secondary .reset .primary .primary .primary)
  | .observeInvalidContinuation =>
      .one (.observe .secondary .continuation .primary .primary .primary)

theorem continuationStep : Product.WorkflowLineage.model.Step Product.WorkflowLineage.initial
    (.observe .secondary .continuation .primary .secondary .primary)
    Product.WorkflowLineage.continuationFinal := by
  change Product.WorkflowLineage.continuationFinal ∈ Product.WorkflowLineage.next
    Product.WorkflowLineage.initial
    (.observe .secondary .continuation .primary .secondary .primary)
  decide

theorem resetStep : Product.WorkflowLineage.model.Step Product.WorkflowLineage.initial
    (.observe .secondary .reset .primary .primary .primary)
    Product.WorkflowLineage.resetFinal := by
  change Product.WorkflowLineage.resetFinal ∈ Product.WorkflowLineage.next
    Product.WorkflowLineage.initial (.observe .secondary .reset .primary .primary .primary)
  decide

theorem mappedRun {state action nextState} (step : nextState ∈ System.next state action) :
    Runs Product.WorkflowLineage.model (project state) (actionMap action).actions
      (project nextState) := by
  cases state <;> cases action <;> simp [System.next,
    Umpire3.Temporal.System.MigratedFamilies.WorkflowLineage.next] at step
  all_goals subst nextState
  · exact Runs.cons continuationStep (Runs.nil _)
  · exact Runs.cons resetStep (Runs.nil _)

theorem initialProjects {systemState} (initialState : System.behavior.Initial () systemState) :
    ∃ featureState, Feature.behavior.Initial () featureState ∧ Projects systemState featureState := by
  subst systemState
  exact ⟨Product.WorkflowLineage.initial, rfl, rfl⟩

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
  have transition : System.mutatedBehavior.Step () .empty .observeInvalidContinuation
      .invalidContinuation := by decide
  rcases simulation .empty Product.WorkflowLineage.initial .observeInvalidContinuation
      .invalidContinuation rfl transition with ⟨nextFeatureState, run, projects⟩
  have finalState : nextFeatureState = Product.WorkflowLineage.invalidContinuationFinal :=
    projects.symm
  subst nextFeatureState
  rcases run.firstStep with ⟨_, invalidStep⟩
  change _ ∈ Product.WorkflowLineage.next Product.WorkflowLineage.initial
    (.observe .secondary .continuation .primary .primary .primary) at invalidStep
  have noNext : Product.WorkflowLineage.next Product.WorkflowLineage.initial
      (.observe .secondary .continuation .primary .primary .primary) = [] := by decide
  rw [noNext] at invalidStep
  cases invalidStep

end WorkflowLineage

namespace WorkflowRouting

namespace Feature

abbrev State := Product.WorkflowRouting.State
abbrev Action := Product.WorkflowRouting.Action
abbrev behavior : Behavior Unit := Behavior.ofTransitionSystem Product.WorkflowRouting.model

end Feature

namespace System

abbrev State := System.MigratedFamilies.WorkflowRouting.State
abbrev Action := System.MigratedFamilies.WorkflowRouting.Action
abbrev behavior := System.MigratedFamilies.WorkflowRouting.behavior
abbrev mutatedBehavior := System.MigratedFamilies.WorkflowRouting.mutatedBehavior
abbrev next := System.MigratedFamilies.WorkflowRouting.next

end System

def taskRouted : Feature.State :=
  Product.WorkflowRouting.initial.setTaskRoute .primary .primary

def matchingPollerRegistered : Feature.State :=
  taskRouted.setPollerRoute .primary .primary

def crossingPollerRegistered : Feature.State :=
  taskRouted.setPollerRoute .primary .secondary

def project : System.State → Feature.State
  | .empty => Product.WorkflowRouting.initial
  | .taskRouted => taskRouted
  | .matchingPollerRegistered => matchingPollerRegistered
  | .crossingPollerRegistered => crossingPollerRegistered
  | .matchingReservation => Product.WorkflowRouting.matchingFinal
  | .crossingReservation => Product.WorkflowRouting.crossingFinal

def Projects (systemState : System.State) (featureState : Feature.State) : Prop :=
  project systemState = featureState

def actionMap : System.Action → ActionEmission Feature.Action
  | .assignTask => .one (.assignTask .primary .primary)
  | .registerMatchingPoller => .one (.registerPoller .primary .primary)
  | .registerCrossingPoller => .one (.registerPoller .primary .secondary)
  | .reserveMatching | .reserveCrossing => .one (.reserve .primary .primary .primary)

theorem assignTaskStep : Product.WorkflowRouting.model.Step Product.WorkflowRouting.initial
    (.assignTask .primary .primary) taskRouted := by
  change taskRouted ∈ Product.WorkflowRouting.next Product.WorkflowRouting.initial
    (.assignTask .primary .primary)
  decide

theorem matchingPollerStep : Product.WorkflowRouting.model.Step taskRouted
    (.registerPoller .primary .primary) matchingPollerRegistered := by
  change matchingPollerRegistered ∈ Product.WorkflowRouting.next taskRouted
    (.registerPoller .primary .primary)
  decide

theorem crossingPollerStep : Product.WorkflowRouting.model.Step taskRouted
    (.registerPoller .primary .secondary) crossingPollerRegistered := by
  change crossingPollerRegistered ∈ Product.WorkflowRouting.next taskRouted
    (.registerPoller .primary .secondary)
  decide

theorem matchingReservationStep : Product.WorkflowRouting.model.Step matchingPollerRegistered
    (.reserve .primary .primary .primary) Product.WorkflowRouting.matchingFinal := by
  change Product.WorkflowRouting.matchingFinal ∈ Product.WorkflowRouting.next
    matchingPollerRegistered (.reserve .primary .primary .primary)
  decide

theorem mappedRun {state action nextState} (step : nextState ∈ System.next state action) :
    Runs Product.WorkflowRouting.model (project state) (actionMap action).actions
      (project nextState) := by
  cases state <;> cases action <;> simp [System.next,
    Umpire3.Temporal.System.MigratedFamilies.WorkflowRouting.next] at step
  all_goals subst nextState
  · exact Runs.cons assignTaskStep (Runs.nil _)
  · exact Runs.cons matchingPollerStep (Runs.nil _)
  · exact Runs.cons crossingPollerStep (Runs.nil _)
  · exact Runs.cons matchingReservationStep (Runs.nil _)

theorem initialProjects {systemState} (initialState : System.behavior.Initial () systemState) :
    ∃ featureState, Feature.behavior.Initial () featureState ∧ Projects systemState featureState := by
  subst systemState
  exact ⟨Product.WorkflowRouting.initial, rfl, rfl⟩

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
  have finalState : nextFeatureState = Product.WorkflowRouting.crossingFinal := projects.symm
  subst nextFeatureState
  rcases run.firstStep with ⟨_, invalidStep⟩
  change _ ∈ Product.WorkflowRouting.next crossingPollerRegistered
    (.reserve .primary .primary .primary) at invalidStep
  have noNext : Product.WorkflowRouting.next crossingPollerRegistered
      (.reserve .primary .primary .primary) = [] := by decide
  rw [noNext] at invalidStep
  cases invalidStep

end WorkflowRouting

namespace SpeculativeTask

namespace Feature

abbrev State := Product.SpeculativeTask.State
abbrev Action := Product.SpeculativeTask.Action
abbrev behavior : Behavior Unit := Behavior.ofTransitionSystem Product.SpeculativeTask.model

end Feature

namespace System

abbrev State := System.MigratedFamilies.SpeculativeTask.State
abbrev Action := System.MigratedFamilies.SpeculativeTask.Action
abbrev behavior := System.MigratedFamilies.SpeculativeTask.behavior
abbrev mutatedBehavior := System.MigratedFamilies.SpeculativeTask.mutatedBehavior
abbrev next := System.MigratedFamilies.SpeculativeTask.next

end System

def requested : Feature.State :=
  Product.SpeculativeTask.initial.setUpdateState .primary .pending

def speculative : Feature.State := requested.setTask .primary .primary .speculative

def project : System.State → Feature.State
  | .empty => Product.SpeculativeTask.initial
  | .updatePending => requested
  | .taskSpeculative => speculative
  | .taskCommitted => Product.SpeculativeTask.committedFinal
  | .orphanedTask => Product.SpeculativeTask.orphanedFinal

def Projects (systemState : System.State) (featureState : Feature.State) : Prop :=
  project systemState = featureState

def actionMap : System.Action → ActionEmission Feature.Action
  | .requestUpdate => .one (.requestUpdate .primary)
  | .createTask => .one (.create .primary .primary)
  | .commitTask => .one (.commit .primary)
  | .createOrphan => .one (.create .primary .secondary)

theorem requestStep : Product.SpeculativeTask.model.Step Product.SpeculativeTask.initial
    (.requestUpdate .primary) requested := by
  change requested ∈ Product.SpeculativeTask.next Product.SpeculativeTask.initial
    (.requestUpdate .primary)
  decide

theorem createStep : Product.SpeculativeTask.model.Step requested
    (.create .primary .primary) speculative := by
  change speculative ∈ Product.SpeculativeTask.next requested (.create .primary .primary)
  decide

theorem commitStep : Product.SpeculativeTask.model.Step speculative
    (.commit .primary) Product.SpeculativeTask.committedFinal := by
  change Product.SpeculativeTask.committedFinal ∈ Product.SpeculativeTask.next speculative
    (.commit .primary)
  decide

theorem mappedRun {state action nextState} (step : nextState ∈ System.next state action) :
    Runs Product.SpeculativeTask.model (project state) (actionMap action).actions
      (project nextState) := by
  cases state <;> cases action <;> simp [System.next,
    Umpire3.Temporal.System.MigratedFamilies.SpeculativeTask.next] at step
  all_goals subst nextState
  · exact Runs.cons requestStep (Runs.nil _)
  · exact Runs.cons createStep (Runs.nil _)
  · exact Runs.cons commitStep (Runs.nil _)

theorem initialProjects {systemState} (initialState : System.behavior.Initial () systemState) :
    ∃ featureState, Feature.behavior.Initial () featureState ∧ Projects systemState featureState := by
  subst systemState
  exact ⟨Product.SpeculativeTask.initial, rfl, rfl⟩

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
  rcases simulation .empty Product.SpeculativeTask.initial .createOrphan .orphanedTask rfl
      transition with ⟨nextFeatureState, run, projects⟩
  have finalState : nextFeatureState = Product.SpeculativeTask.orphanedFinal := projects.symm
  subst nextFeatureState
  rcases run.firstStep with ⟨_, invalidStep⟩
  change _ ∈ Product.SpeculativeTask.next Product.SpeculativeTask.initial
    (.create .primary .secondary) at invalidStep
  have noNext : Product.SpeculativeTask.next Product.SpeculativeTask.initial
      (.create .primary .secondary) = [] := by decide
  rw [noNext] at invalidStep
  cases invalidStep

end SpeculativeTask

namespace CallbackReference

namespace Feature

abbrev State := Product.CallbackReference.State
abbrev Action := Product.CallbackReference.Action
abbrev behavior : Behavior Unit := Behavior.ofTransitionSystem Product.CallbackReference.model

end Feature

namespace System

abbrev State := System.MigratedFamilies.CallbackReference.State
abbrev Action := System.MigratedFamilies.CallbackReference.Action
abbrev behavior := System.MigratedFamilies.CallbackReference.behavior
abbrev mutatedBehavior := System.MigratedFamilies.CallbackReference.mutatedBehavior
abbrev next := System.MigratedFamilies.CallbackReference.next

end System

def attachment : Feature.State := Product.CallbackReference.initial.setAttachment
  .primary .primary .event .workflowStarted .first false

def project : System.State → Feature.State
  | .empty => Product.CallbackReference.initial
  | .attachmentObserved => attachment
  | .matchingOperationObserved => Product.CallbackReference.matchingFinal
  | .wrongOperationObserved => Product.CallbackReference.wrongReferenceFinal

def Projects (systemState : System.State) (featureState : Feature.State) : Prop :=
  project systemState = featureState

def actionMap : System.Action → ActionEmission Feature.Action
  | .observeAttachment =>
      .one (.observeAttachment .primary .primary .event .workflowStarted .first false)
  | .observeMatchingOperation =>
      .one (.observeOperationStart .primary .primary .primary .event .workflowStarted .second false)
  | .observeWrongOperation =>
      .one (.observeOperationStart .primary .primary .secondary .request .optionsUpdated .second false)

theorem attachmentStep : Product.CallbackReference.model.Step Product.CallbackReference.initial
    (.observeAttachment .primary .primary .event .workflowStarted .first false) attachment := by
  change attachment ∈ Product.CallbackReference.next Product.CallbackReference.initial
    (.observeAttachment .primary .primary .event .workflowStarted .first false)
  decide

theorem matchingOperationStep : Product.CallbackReference.model.Step attachment
    (.observeOperationStart .primary .primary .primary .event .workflowStarted .second false)
    Product.CallbackReference.matchingFinal := by
  change Product.CallbackReference.matchingFinal ∈ Product.CallbackReference.next attachment
    (.observeOperationStart .primary .primary .primary .event .workflowStarted .second false)
  decide

theorem mappedRun {state action nextState} (step : nextState ∈ System.next state action) :
    Runs Product.CallbackReference.model (project state) (actionMap action).actions
      (project nextState) := by
  cases state <;> cases action <;> simp [System.next,
    Umpire3.Temporal.System.MigratedFamilies.CallbackReference.next] at step
  all_goals subst nextState
  · exact Runs.cons attachmentStep (Runs.nil _)
  · exact Runs.cons matchingOperationStep (Runs.nil _)

theorem initialProjects {systemState} (initialState : System.behavior.Initial () systemState) :
    ∃ featureState, Feature.behavior.Initial () featureState ∧ Projects systemState featureState := by
  subst systemState
  exact ⟨Product.CallbackReference.initial, rfl, rfl⟩

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
  have finalState : nextFeatureState = Product.CallbackReference.wrongReferenceFinal := projects.symm
  subst nextFeatureState
  rcases run.firstStep with ⟨_, invalidStep⟩
  change _ ∈ Product.CallbackReference.next attachment
    (.observeOperationStart .primary .primary .secondary .request .optionsUpdated .second false)
      at invalidStep
  have noNext : Product.CallbackReference.next attachment
      (.observeOperationStart .primary .primary .secondary .request .optionsUpdated .second false) =
      [] := by decide
  rw [noNext] at invalidStep
  cases invalidStep

end CallbackReference

namespace CallbackResponse

namespace Feature

abbrev State := Product.CallbackResponse.State
abbrev Action := Product.CallbackResponse.Action
abbrev behavior : Behavior Unit := Behavior.ofTransitionSystem Product.CallbackResponse.model

end Feature

namespace System

abbrev State := System.MigratedFamilies.CallbackResponse.State
abbrev Action := System.MigratedFamilies.CallbackResponse.Action
abbrev behavior := System.MigratedFamilies.CallbackResponse.behavior
abbrev mutatedBehavior := System.MigratedFamilies.CallbackResponse.mutatedBehavior
abbrev next := System.MigratedFamilies.CallbackResponse.next

end System

def registered : Feature.State :=
  Product.CallbackResponse.initial.register .primary .primary .primary

def settled : Feature.State := registered.settle .primary .second true

def conflictedAfterSuccess : Feature.State :=
  Product.CallbackResponse.consistentFinal.markConflict .primary

def project : System.State → Feature.State
  | .empty => Product.CallbackResponse.initial
  | .registered => registered
  | .settled => settled
  | .responded => Product.CallbackResponse.consistentFinal
  | .conflictingResponse => conflictedAfterSuccess

def Projects (systemState : System.State) (featureState : Feature.State) : Prop :=
  project systemState = featureState

def actionMap : System.Action → ActionEmission Feature.Action
  | .register => .one (.register .primary .primary .primary)
  | .settle => .one (.settleOperation .primary .second true)
  | .respond => .one (.recordResponse .primary .asyncSuccess .accepted .third)
  | .conflict => .one (.recordResponse .primary .failure .conflicting .third)

theorem registerStep : Product.CallbackResponse.model.Step Product.CallbackResponse.initial
    (.register .primary .primary .primary) registered := by
  change registered ∈ Product.CallbackResponse.next Product.CallbackResponse.initial
    (.register .primary .primary .primary)
  decide

theorem settleStep : Product.CallbackResponse.model.Step registered
    (.settleOperation .primary .second true) settled := by
  change settled ∈ Product.CallbackResponse.next registered
    (.settleOperation .primary .second true)
  decide

theorem responseStep : Product.CallbackResponse.model.Step settled
    (.recordResponse .primary .asyncSuccess .accepted .third)
    Product.CallbackResponse.consistentFinal := by
  change Product.CallbackResponse.consistentFinal ∈ Product.CallbackResponse.next settled
    (.recordResponse .primary .asyncSuccess .accepted .third)
  decide

theorem mappedRun {state action nextState} (step : nextState ∈ System.next state action) :
    Runs Product.CallbackResponse.model (project state) (actionMap action).actions
      (project nextState) := by
  cases state <;> cases action <;> simp [System.next,
    Umpire3.Temporal.System.MigratedFamilies.CallbackResponse.next] at step
  all_goals subst nextState
  · exact Runs.cons registerStep (Runs.nil _)
  · exact Runs.cons settleStep (Runs.nil _)
  · exact Runs.cons responseStep (Runs.nil _)

theorem initialProjects {systemState} (initialState : System.behavior.Initial () systemState) :
    ∃ featureState, Feature.behavior.Initial () featureState ∧ Projects systemState featureState := by
  subst systemState
  exact ⟨Product.CallbackResponse.initial, rfl, rfl⟩

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
    ¬Product.CallbackResponse.ResponseConsistency conflictedAfterSuccess := by decide

theorem mutationBreaksDeclaredSimulation :
    ¬StepSimulation System.mutatedBehavior Feature.behavior Projects actionMap := by
  intro simulation
  have transition : System.mutatedBehavior.Step () .responded .conflict .conflictingResponse := by
    decide
  rcases simulation .responded Product.CallbackResponse.consistentFinal .conflict
      .conflictingResponse rfl transition with ⟨nextFeatureState, run, projects⟩
  have finalState : nextFeatureState = conflictedAfterSuccess := projects.symm
  subst nextFeatureState
  rcases run.firstStep with ⟨_, invalidStep⟩
  change _ ∈ Product.CallbackResponse.next Product.CallbackResponse.consistentFinal
    (.recordResponse .primary .failure .conflicting .third) at invalidStep
  have noNext : Product.CallbackResponse.next Product.CallbackResponse.consistentFinal
      (.recordResponse .primary .failure .conflicting .third) = [] := by decide
  rw [noNext] at invalidStep
  cases invalidStep

end CallbackResponse

namespace NexusTimeout

namespace Feature

abbrev State := Product.NexusTimeout.State
abbrev Action := Product.NexusTimeout.Action
abbrev behavior : Behavior Unit := Behavior.ofTransitionSystem Product.NexusTimeout.model

end Feature

namespace System

abbrev State := System.MigratedFamilies.NexusTimeout.State
abbrev Action := System.MigratedFamilies.NexusTimeout.Action
abbrev behavior := System.MigratedFamilies.NexusTimeout.behavior
abbrev mutatedBehavior := System.MigratedFamilies.NexusTimeout.mutatedBehavior
abbrev next := System.MigratedFamilies.NexusTimeout.next

end System

def configured : Feature.State :=
  Product.NexusTimeout.initial.setConfigured .primary true

def project : System.State → Feature.State
  | .idle => Product.NexusTimeout.initial
  | .configured => configured
  | .timedOut => Product.NexusTimeout.permittedFinal
  | .malformedTimeout => Product.NexusTimeout.unsafeInvalidFinal

def Projects (systemState : System.State) (featureState : Feature.State) : Prop :=
  project systemState = featureState

def actionMap : System.Action → ActionEmission Feature.Action
  | .configure => .one (.configure .primary)
  | .recordTimeout =>
      .one (.recordTimeout .primary .primary .startToClose .operationTimedOut)
  | .recordMalformedTimeout =>
      .one (.recordTimeout .primary .primary .unspecified .unrelatedFailure)

theorem configureStep : Product.NexusTimeout.model.Step Product.NexusTimeout.initial
    (.configure .primary) configured := by
  change configured ∈ Product.NexusTimeout.next Product.NexusTimeout.initial (.configure .primary)
  decide

theorem timeoutStep : Product.NexusTimeout.model.Step configured
    (.recordTimeout .primary .primary .startToClose .operationTimedOut)
    Product.NexusTimeout.permittedFinal := by
  change Product.NexusTimeout.permittedFinal ∈ Product.NexusTimeout.next configured
    (.recordTimeout .primary .primary .startToClose .operationTimedOut)
  decide

theorem mappedRun {state action nextState} (step : nextState ∈ System.next state action) :
    Runs Product.NexusTimeout.model (project state) (actionMap action).actions
      (project nextState) := by
  cases state <;> cases action <;> simp [System.next,
    Umpire3.Temporal.System.MigratedFamilies.NexusTimeout.next] at step
  all_goals subst nextState
  · exact Runs.cons configureStep (Runs.nil _)
  · exact Runs.cons timeoutStep (Runs.nil _)

theorem initialProjects {systemState} (initialState : System.behavior.Initial () systemState) :
    ∃ featureState, Feature.behavior.Initial () featureState ∧ Projects systemState featureState := by
  subst systemState
  exact ⟨Product.NexusTimeout.initial, rfl, rfl⟩

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
  have finalState : nextFeatureState = Product.NexusTimeout.unsafeInvalidFinal := projects.symm
  subst nextFeatureState
  rcases run.firstStep with ⟨_, invalidStep⟩
  change _ ∈ Product.NexusTimeout.next configured
    (.recordTimeout .primary .primary .unspecified .unrelatedFailure) at invalidStep
  have noNext : Product.NexusTimeout.next configured
      (.recordTimeout .primary .primary .unspecified .unrelatedFailure) = [] := by decide
  rw [noNext] at invalidStep
  cases invalidStep

end NexusTimeout

namespace NexusClosure

namespace Feature

abbrev State := Product.NexusClosure.State
abbrev Action := Product.NexusClosure.Action
abbrev behavior : Behavior Unit := Behavior.ofTransitionSystem Product.NexusClosure.model

end Feature

namespace System

abbrev State := System.MigratedFamilies.NexusClosure.State
abbrev Action := System.MigratedFamilies.NexusClosure.Action
abbrev behavior := System.MigratedFamilies.NexusClosure.behavior
abbrev mutatedBehavior := System.MigratedFamilies.NexusClosure.mutatedBehavior
abbrev next := System.MigratedFamilies.NexusClosure.next

end System

def scheduled : Feature.State :=
  (Product.NexusClosure.initial.setOperation .primary .scheduled).setCaller .primary true

def started : Feature.State := scheduled.setOperation .primary .started

def settled : Feature.State := started.setOperation .primary .succeeded

def project : System.State → Feature.State
  | .idle => Product.NexusClosure.initial
  | .scheduled => scheduled
  | .started => started
  | .settled => settled
  | .closed => Product.NexusClosure.permittedFinal
  | .closedWhileRunning => Product.NexusClosure.unsafeFinal

def Projects (systemState : System.State) (featureState : Feature.State) : Prop :=
  project systemState = featureState

def actionMap : System.Action → ActionEmission Feature.Action
  | .schedule => .one (.schedule .primary)
  | .start => .one (.start .primary)
  | .settle => .one (.settle .primary .succeeded)
  | .close | .closeWhileRunning => .one (.closeWorkflow .completed)

theorem scheduleStep : Product.NexusClosure.model.Step Product.NexusClosure.initial
    (.schedule .primary) scheduled := by
  change scheduled ∈ Product.NexusClosure.next Product.NexusClosure.initial (.schedule .primary)
  decide

theorem startStep : Product.NexusClosure.model.Step scheduled (.start .primary) started := by
  change started ∈ Product.NexusClosure.next scheduled (.start .primary)
  decide

theorem settleStep : Product.NexusClosure.model.Step started
    (.settle .primary .succeeded) settled := by
  change settled ∈ Product.NexusClosure.next started (.settle .primary .succeeded)
  decide

theorem closeStep : Product.NexusClosure.model.Step settled
    (.closeWorkflow .completed) Product.NexusClosure.permittedFinal := by
  change Product.NexusClosure.permittedFinal ∈ Product.NexusClosure.next settled
    (.closeWorkflow .completed)
  decide

theorem mappedRun {state action nextState} (step : nextState ∈ System.next state action) :
    Runs Product.NexusClosure.model (project state) (actionMap action).actions
      (project nextState) := by
  cases state <;> cases action <;> simp [System.next,
    Umpire3.Temporal.System.MigratedFamilies.NexusClosure.next] at step
  all_goals subst nextState
  · exact Runs.cons scheduleStep (Runs.nil _)
  · exact Runs.cons startStep (Runs.nil _)
  · exact Runs.cons settleStep (Runs.nil _)
  · exact Runs.cons closeStep (Runs.nil _)

theorem initialProjects {systemState} (initialState : System.behavior.Initial () systemState) :
    ∃ featureState, Feature.behavior.Initial () featureState ∧ Projects systemState featureState := by
  subst systemState
  exact ⟨Product.NexusClosure.initial, rfl, rfl⟩

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
  have finalState : nextFeatureState = Product.NexusClosure.unsafeFinal := projects.symm
  subst nextFeatureState
  rcases run.firstStep with ⟨_, invalidStep⟩
  change _ ∈ Product.NexusClosure.next started (.closeWorkflow .completed) at invalidStep
  have noNext : Product.NexusClosure.next started (.closeWorkflow .completed) = [] := by decide
  rw [noNext] at invalidStep
  cases invalidStep

end NexusClosure

namespace NexusActivityLink

namespace Feature

abbrev State := Product.NexusActivityLink.State
abbrev Action := Product.NexusActivityLink.Action
abbrev behavior : Behavior Unit := Behavior.ofTransitionSystem Product.NexusActivityLink.model

end Feature

namespace System

abbrev State := System.MigratedFamilies.NexusActivityLink.State
abbrev Action := System.MigratedFamilies.NexusActivityLink.Action
abbrev behavior := System.MigratedFamilies.NexusActivityLink.behavior
abbrev mutatedBehavior := System.MigratedFamilies.NexusActivityLink.mutatedBehavior
abbrev next := System.MigratedFamilies.NexusActivityLink.next

end System

def operationObserved : Feature.State :=
  (Product.NexusActivityLink.initial.setOperationObserved .primary true).setForward
    .primary (some .primary)

def project : System.State → Feature.State
  | .empty => Product.NexusActivityLink.initial
  | .operationObserved => operationObserved
  | .linked => Product.NexusActivityLink.matchingFinal
  | .oneSided => Product.NexusActivityLink.oneSidedFinal

def Projects (systemState : System.State) (featureState : Feature.State) : Prop :=
  project systemState = featureState

def actionMap : System.Action → ActionEmission Feature.Action
  | .observeOperation => .one (.observeOperation .primary (some .primary))
  | .observeLinkedActivity => .one (.observeActivity .primary (some .primary))
  | .observeOneSidedActivity => .one (.observeActivity .primary none)

theorem operationStep : Product.NexusActivityLink.model.Step Product.NexusActivityLink.initial
    (.observeOperation .primary (some .primary)) operationObserved := by
  change operationObserved ∈ Product.NexusActivityLink.next Product.NexusActivityLink.initial
    (.observeOperation .primary (some .primary))
  decide

theorem activityStep : Product.NexusActivityLink.model.Step operationObserved
    (.observeActivity .primary (some .primary)) Product.NexusActivityLink.matchingFinal := by
  change Product.NexusActivityLink.matchingFinal ∈ Product.NexusActivityLink.next
    operationObserved (.observeActivity .primary (some .primary))
  decide

theorem mappedRun {state action nextState} (step : nextState ∈ System.next state action) :
    Runs Product.NexusActivityLink.model (project state) (actionMap action).actions
      (project nextState) := by
  cases state <;> cases action <;> simp [System.next,
    Umpire3.Temporal.System.MigratedFamilies.NexusActivityLink.next] at step
  all_goals subst nextState
  · exact Runs.cons operationStep (Runs.nil _)
  · exact Runs.cons activityStep (Runs.nil _)

theorem initialProjects {systemState} (initialState : System.behavior.Initial () systemState) :
    ∃ featureState, Feature.behavior.Initial () featureState ∧ Projects systemState featureState := by
  subst systemState
  exact ⟨Product.NexusActivityLink.initial, rfl, rfl⟩

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
  have finalState : nextFeatureState = Product.NexusActivityLink.oneSidedFinal := projects.symm
  subst nextFeatureState
  rcases run.firstStep with ⟨_, invalidStep⟩
  change _ ∈ Product.NexusActivityLink.next operationObserved
    (.observeActivity .primary none) at invalidStep
  have noNext : Product.NexusActivityLink.next operationObserved
      (.observeActivity .primary none) = [] := by decide
  rw [noNext] at invalidStep
  cases invalidStep

end NexusActivityLink

namespace WorkflowProgress

namespace Feature

abbrev State := Product.WorkflowProgress.State
abbrev Action := Product.WorkflowProgress.Action
abbrev behavior : Behavior Unit := Behavior.ofTransitionSystem Product.WorkflowProgress.model

end Feature

namespace System

abbrev State := System.MigratedFamilies.WorkflowProgress.State
abbrev Action := System.MigratedFamilies.WorkflowProgress.Action
abbrev behavior := System.MigratedFamilies.WorkflowProgress.behavior
abbrev mutatedBehavior := System.MigratedFamilies.WorkflowProgress.mutatedBehavior
abbrev next := System.MigratedFamilies.WorkflowProgress.next

end System

def queued : Feature.State :=
  (Product.WorkflowProgress.initial.enqueue .primary .primary).setEntityState .primary .pending

def dispatched : Feature.State := Product.WorkflowProgress.waited.dispatch .primary .primary

def wrongEntityAfterWait : Feature.State := dispatched.complete .primary .secondary

def project : System.State → Feature.State
  | .idle => Product.WorkflowProgress.initial
  | .queued => queued
  | .workerAvailable => Product.WorkflowProgress.queuedAvailable
  | .waited => Product.WorkflowProgress.waited
  | .dispatched => dispatched
  | .completed => Product.WorkflowProgress.progressedFinal
  | .starved => Product.WorkflowProgress.starvedFinal
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

theorem enqueueStep : Product.WorkflowProgress.model.Step Product.WorkflowProgress.initial
    (.enqueue .primary .primary) queued := by
  change queued ∈ Product.WorkflowProgress.next Product.WorkflowProgress.initial
    (.enqueue .primary .primary)
  decide

theorem workerStep : Product.WorkflowProgress.model.Step queued
    (.makeWorkerAvailable .primary) Product.WorkflowProgress.queuedAvailable := by
  change Product.WorkflowProgress.queuedAvailable ∈ Product.WorkflowProgress.next queued
    (.makeWorkerAvailable .primary)
  decide

theorem waitStep : Product.WorkflowProgress.model.Step Product.WorkflowProgress.queuedAvailable
    (.wait .primary) Product.WorkflowProgress.waited := by
  change Product.WorkflowProgress.waited ∈ Product.WorkflowProgress.next
    Product.WorkflowProgress.queuedAvailable (.wait .primary)
  decide

theorem dispatchStep : Product.WorkflowProgress.model.Step Product.WorkflowProgress.waited
    (.dispatch .primary .primary) dispatched := by
  change dispatched ∈ Product.WorkflowProgress.next Product.WorkflowProgress.waited
    (.dispatch .primary .primary)
  decide

theorem completeStep : Product.WorkflowProgress.model.Step dispatched
    (.complete .primary .primary) Product.WorkflowProgress.progressedFinal := by
  change Product.WorkflowProgress.progressedFinal ∈ Product.WorkflowProgress.next dispatched
    (.complete .primary .primary)
  decide

theorem mappedRun {state action nextState} (step : nextState ∈ System.next state action) :
    Runs Product.WorkflowProgress.model (project state) (actionMap action).actions
      (project nextState) := by
  cases state <;> cases action <;> simp [System.next,
    Umpire3.Temporal.System.MigratedFamilies.WorkflowProgress.next] at step
  all_goals subst nextState
  · exact Runs.cons enqueueStep (Runs.nil _)
  · exact Runs.cons workerStep (Runs.nil _)
  · exact Runs.cons waitStep (Runs.nil _)
  · exact Runs.cons dispatchStep (Runs.nil _)
  · exact Runs.cons completeStep (Runs.nil _)

theorem initialProjects {systemState} (initialState : System.behavior.Initial () systemState) :
    ∃ featureState, Feature.behavior.Initial () featureState ∧ Projects systemState featureState := by
  subst systemState
  exact ⟨Product.WorkflowProgress.initial, rfl, rfl⟩

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
  rcases simulation .waited Product.WorkflowProgress.waited .waitAgain .starved rfl transition
      with ⟨nextFeatureState, run, projects⟩
  have finalState : nextFeatureState = Product.WorkflowProgress.starvedFinal := projects.symm
  subst nextFeatureState
  rcases run.firstStep with ⟨_, invalidStep⟩
  change _ ∈ Product.WorkflowProgress.next Product.WorkflowProgress.waited (.wait .primary)
    at invalidStep
  have noNext : Product.WorkflowProgress.next Product.WorkflowProgress.waited (.wait .primary) =
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
  change _ ∈ Product.WorkflowProgress.next dispatched (.complete .primary .secondary)
    at invalidStep
  have noNext : Product.WorkflowProgress.next dispatched (.complete .primary .secondary) = [] := by
    decide
  rw [noNext] at invalidStep
  cases invalidStep

end WorkflowProgress

namespace UpdateLifecycle

namespace Feature

abbrev State := Umpire3.Temporal.Feature.UpdateLifecycle.State
abbrev Action := Umpire3.Temporal.Feature.UpdateLifecycle.Action
abbrev behavior := Umpire3.Temporal.Feature.UpdateLifecycle.behavior
abbrev next := Umpire3.Temporal.Feature.UpdateLifecycle.next

end Feature

namespace System

abbrev State := System.MigratedFamilies.UpdateLifecycle.State
abbrev Action := System.MigratedFamilies.UpdateLifecycle.Action
abbrev behavior := System.MigratedFamilies.UpdateLifecycle.behavior
abbrev mutatedBehavior := System.MigratedFamilies.UpdateLifecycle.mutatedBehavior
abbrev next := System.MigratedFamilies.UpdateLifecycle.next

end System

def project : System.State → Feature.State
  | .idle => .idle
  | .requested | .taskDispatched => .requested
  | .accepted => .accepted
  | .historyRecorded | .workflowTaskCompleted => .historyRecorded
  | .completed => .completed
  | .completedWithoutHistory => .completedWithoutHistory

def Projects (systemState : System.State) (featureState : Feature.State) : Prop :=
  project systemState = featureState

def actionMap : System.Action → ActionEmission Feature.Action
  | .start => .one .request
  | .dispatchTask | .completeWorkflowTask => .stutter
  | .accept => .one .accept
  | .recordHistory => .one .recordHistory
  | .complete => .one .complete
  | .completeWithoutHistory => .one .completeWithoutHistory

theorem requestStep : Feature.behavior.Step () .idle .request .requested := by decide
theorem acceptStep : Feature.behavior.Step () .requested .accept .accepted := by decide
theorem historyStep : Feature.behavior.Step () .accepted .recordHistory .historyRecorded := by decide
theorem completeStep : Feature.behavior.Step () .historyRecorded .complete .completed := by decide

theorem mappedRun {state action nextState} (step : nextState ∈ System.next state action) :
    Runs (Feature.behavior.at ()) (project state) (actionMap action).actions
      (project nextState) := by
  cases state <;> cases action <;> simp [System.next,
    Umpire3.Temporal.System.MigratedFamilies.UpdateLifecycle.next] at step
  all_goals subst nextState
  · exact Runs.cons requestStep (Runs.nil _)
  · exact Runs.nil _
  · exact Runs.cons acceptStep (Runs.nil _)
  · exact Runs.cons historyStep (Runs.nil _)
  · exact Runs.nil _
  · exact Runs.cons completeStep (Runs.nil _)

theorem initialProjects {systemState} (initialState : System.behavior.Initial () systemState) :
    ∃ featureState, Feature.behavior.Initial () featureState ∧ Projects systemState featureState := by
  subst systemState
  exact ⟨Umpire3.Temporal.Feature.UpdateLifecycle.State.idle, rfl, rfl⟩

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
  have transition : System.mutatedBehavior.Step () .accepted .completeWithoutHistory
      .completedWithoutHistory := by decide
  rcases simulation .accepted .accepted .completeWithoutHistory .completedWithoutHistory rfl
      transition with ⟨nextFeatureState, run, projects⟩
  have finalState : nextFeatureState =
      Umpire3.Temporal.Feature.UpdateLifecycle.State.completedWithoutHistory := projects.symm
  subst nextFeatureState
  rcases run.firstStep with ⟨_, invalidStep⟩
  change _ ∈ Feature.next .accepted .completeWithoutHistory at invalidStep
  have noNext : Feature.next .accepted .completeWithoutHistory = [] := by decide
  rw [noNext] at invalidStep
  cases invalidStep

end UpdateLifecycle

namespace RoutingIsolation

structure ExecutableViews where
  lineage : ExecutableView WorkflowLineage.System.behavior
  routing : ExecutableView WorkflowRouting.System.behavior

def executableViews : ExecutableViews where
  lineage := Umpire3.Temporal.System.MigratedFamilies.WorkflowLineage.executable
  routing := Umpire3.Temporal.System.MigratedFamilies.WorkflowRouting.executable

structure Simulations where
  lineage : SafetySimulation WorkflowLineage.System.behavior WorkflowLineage.Feature.behavior
  routing : SafetySimulation WorkflowRouting.System.behavior WorkflowRouting.Feature.behavior

def soundSimulations : Simulations where
  lineage := WorkflowLineage.soundSimulation
  routing := WorkflowRouting.soundSimulation

theorem mutationsBreakDeclaredSimulations :
    (¬StepSimulation WorkflowLineage.System.mutatedBehavior WorkflowLineage.Feature.behavior
      WorkflowLineage.Projects WorkflowLineage.actionMap) ∧
    (¬StepSimulation WorkflowRouting.System.mutatedBehavior WorkflowRouting.Feature.behavior
      WorkflowRouting.Projects WorkflowRouting.actionMap) :=
  ⟨WorkflowLineage.mutationBreaksDeclaredSimulation,
    WorkflowRouting.mutationBreaksDeclaredSimulation⟩

end RoutingIsolation

end Umpire3.Temporal.Refinement.MigratedFamilies
