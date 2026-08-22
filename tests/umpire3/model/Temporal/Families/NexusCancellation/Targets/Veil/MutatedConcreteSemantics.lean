import Temporal.Families.NexusCancellation.Targets.Veil.MutatedConcrete
import Temporal.Families.NexusCancellation.Targets.Veil.MutatedSemantics

namespace Umpire3.Temporal.Veil.NexusCancellationFencing

open Umpire3.Temporal.Targets.NexusCancellationFencing

set_option maxHeartbeats 400000
abbrev MutatedConcreteLifecycle := NexusCancellationStaleCompletionGuardRemovedConcrete.Lifecycle_IndT
abbrev MutatedConcreteTaskStage := NexusCancellationStaleCompletionGuardRemovedConcrete.TaskStage_IndT
abbrev MutatedConcreteEpoch := NexusCancellationStaleCompletionGuardRemovedConcrete.Epoch_IndT
abbrev MutatedConcreteField :=
  NexusCancellationStaleCompletionGuardRemovedConcrete.FieldConcreteType MutatedConcreteLifecycle MutatedConcreteTaskStage MutatedConcreteEpoch
abbrev MutatedConcreteState := NexusCancellationStaleCompletionGuardRemovedConcrete.State MutatedConcreteField
abbrev MutatedConcreteTheoryType :=
  NexusCancellationStaleCompletionGuardRemovedConcrete.Theory MutatedConcreteLifecycle MutatedConcreteTaskStage MutatedConcreteEpoch
abbrev MutatedConcreteLabel :=
  NexusCancellationStaleCompletionGuardRemovedConcrete.Label MutatedConcreteLifecycle MutatedConcreteTaskStage MutatedConcreteEpoch

local instance : NexusCancellationStaleCompletionGuardRemovedConcrete.Lifecycle_EnumClass MutatedConcreteLifecycle :=
  NexusCancellationStaleCompletionGuardRemovedConcrete.instLifecycle_EnumClassLifecycle_IndT
local instance : NexusCancellationStaleCompletionGuardRemovedConcrete.TaskStage_EnumClass MutatedConcreteTaskStage :=
  NexusCancellationStaleCompletionGuardRemovedConcrete.instTaskStage_EnumClassTaskStage_IndT
local instance : NexusCancellationStaleCompletionGuardRemovedConcrete.Epoch_EnumClass MutatedConcreteEpoch :=
  NexusCancellationStaleCompletionGuardRemovedConcrete.instEpoch_EnumClassEpoch_IndT
local instance MutatedConcreteFieldRepresentation :
    (field : NexusCancellationStaleCompletionGuardRemovedConcrete.State.Label) →
      Veil.FieldRepresentation
        (NexusCancellationStaleCompletionGuardRemovedConcrete.State.Label.toDomain
          MutatedConcreteLifecycle MutatedConcreteTaskStage MutatedConcreteEpoch field)
        (NexusCancellationStaleCompletionGuardRemovedConcrete.State.Label.toCodomain
          MutatedConcreteLifecycle MutatedConcreteTaskStage MutatedConcreteEpoch field)
        (MutatedConcreteField field) :=
  NexusCancellationStaleCompletionGuardRemovedConcrete.instFieldRepresentation
    MutatedConcreteLifecycle MutatedConcreteTaskStage MutatedConcreteEpoch
local instance MutatedConcreteFieldLawful :
    (field : NexusCancellationStaleCompletionGuardRemovedConcrete.State.Label) →
      Veil.LawfulFieldRepresentation
        (NexusCancellationStaleCompletionGuardRemovedConcrete.State.Label.toDomain
          MutatedConcreteLifecycle MutatedConcreteTaskStage MutatedConcreteEpoch field)
        (NexusCancellationStaleCompletionGuardRemovedConcrete.State.Label.toCodomain
          MutatedConcreteLifecycle MutatedConcreteTaskStage MutatedConcreteEpoch field)
        (MutatedConcreteField field)
        (MutatedConcreteFieldRepresentation field) :=
  NexusCancellationStaleCompletionGuardRemovedConcrete.instLawfulFieldRepresentation
    MutatedConcreteLifecycle MutatedConcreteTaskStage MutatedConcreteEpoch
local instance : Inhabited MutatedConcreteState :=
  NexusCancellationStaleCompletionGuardRemovedConcrete.instInhabitedStateFieldConcreteType

def mutatedConcreteTheory : MutatedConcreteTheoryType := {}

def MutatedConcreteSystem :=
  NexusCancellationStaleCompletionGuardRemovedConcrete.enumerableTransitionSystem
    MutatedConcreteLifecycle MutatedConcreteTaskStage MutatedConcreteEpoch mutatedConcreteTheory

def MutatedConcreteInitialState : MutatedConcreteState := {
  lifecycle := .Open
  task := .Idle
  ownerEpoch := .Epoch0
  workerEpoch := .None
  completionEpoch := .None
}

private theorem MutatedConcreteInitialStates_eq : MutatedConcreteSystem.initStates = [MutatedConcreteInitialState] := by
  rfl

def MutatedConcreteNext (state : MutatedConcreteState) (label : MutatedConcreteLabel)
    (nextState : MutatedConcreteState) : Prop :=
  (label, Veil.ExecutionOutcome.success nextState) ∈
    MutatedConcreteSystem.tr mutatedConcreteTheory state

def MutatedConcreteInitial (state : MutatedConcreteState) : Prop :=
  state ∈ MutatedConcreteSystem.initStates

def MutatedConcreteProperty (state : MutatedConcreteState) : Prop :=
  NexusCancellationStaleCompletionGuardRemovedConcrete.NexusCancellationWonExcludesSuccess
    (th := mutatedConcreteTheory) (st := state)

def lifecycleToMutated : MutatedConcreteLifecycle → MutatedLifecycle
  | .Open => .Open
  | .CancellationAccepted => .CancellationAccepted
  | .Cancelled => .Cancelled
  | .Succeeded => .Succeeded

def taskToMutated : MutatedConcreteTaskStage → MutatedTaskStage
  | .Idle => .Idle
  | .Dispatched => .Dispatched
  | .Returned => .Returned

def epochToMutated : MutatedConcreteEpoch → MutatedEpoch
  | .None => .None
  | .Epoch0 => .Epoch0
  | .Epoch1 => .Epoch1

def stateToMutated (state : MutatedConcreteState) : MutatedState := {
  lifecycle := lifecycleToMutated state.lifecycle
  task := taskToMutated state.task
  ownerEpoch := epochToMutated state.ownerEpoch
  workerEpoch := epochToMutated state.workerEpoch
  completionEpoch := epochToMutated state.completionEpoch
}

@[simp] private theorem stateToMutated_lifecycle (state : MutatedConcreteState) :
    (stateToMutated state).lifecycle = lifecycleToMutated state.lifecycle := rfl

@[simp] private theorem stateToMutated_task (state : MutatedConcreteState) :
    (stateToMutated state).task = taskToMutated state.task := rfl

@[simp] private theorem stateToMutated_ownerEpoch (state : MutatedConcreteState) :
    (stateToMutated state).ownerEpoch = epochToMutated state.ownerEpoch := rfl

@[simp] private theorem stateToMutated_workerEpoch (state : MutatedConcreteState) :
    (stateToMutated state).workerEpoch = epochToMutated state.workerEpoch := rfl

@[simp] private theorem stateToMutated_completionEpoch (state : MutatedConcreteState) :
    (stateToMutated state).completionEpoch = epochToMutated state.completionEpoch := rfl

private theorem lifecycleToMutated_injective : Function.Injective lifecycleToMutated := by
  intro left right equal
  cases left <;> cases right <;> simp_all [lifecycleToMutated]

private theorem taskToMutated_injective : Function.Injective taskToMutated := by
  intro left right equal
  cases left <;> cases right <;> simp_all [taskToMutated]

private theorem epochToMutated_injective : Function.Injective epochToMutated := by
  intro left right equal
  cases left <;> cases right <;> simp_all [epochToMutated]

private theorem stateToMutated_injective : Function.Injective stateToMutated := by
  intro left right equal
  rcases left with ⟨leftLifecycle, leftTask, leftOwner, leftWorker, leftCompletion⟩
  rcases right with ⟨rightLifecycle, rightTask, rightOwner, rightWorker, rightCompletion⟩
  simp only [stateToMutated, NexusCancellationStaleCompletionGuardRemoved.State.mk.injEq] at equal
  rcases equal with ⟨lifecycleEqual, taskEqual, ownerEqual, workerEqual, completionEqual⟩
  have := lifecycleToMutated_injective lifecycleEqual
  have := taskToMutated_injective taskEqual
  have := epochToMutated_injective ownerEqual
  have := epochToMutated_injective workerEqual
  have := epochToMutated_injective completionEqual
  subst_vars
  rfl

def labelToMutated : MutatedConcreteLabel → MutatedLabel
  | .DispatchTask => .DispatchTask
  | .RequestCancellation => .RequestCancellation
  | .AcquireOwnership => .AcquireOwnership
  | .CommitCancellation => .CommitCancellation
  | .WorkerReturnsSuccess => .WorkerReturnsSuccess
  | .PersistSuccess => .PersistSuccess

private theorem labelToMutated_injective : Function.Injective labelToMutated := by
  intro left right equal
  cases left <;> cases right <;> simp_all [labelToMutated]

def encodeMutatedConcreteState (state : MutatedConcreteState) : Umpire3.FirstOrderState :=
  mutatedSemanticRelation.encodeState (stateToMutated state)

def encodeMutatedConcreteLabel (label : MutatedConcreteLabel) : String :=
  mutatedSemanticRelation.encodeLabel (labelToMutated label)

def MutatedConcreteDispatchState (state : MutatedConcreteState) : MutatedConcreteState :=
  { state with task := .Dispatched, workerEpoch := state.ownerEpoch }

def MutatedConcreteRequestCancellationState (state : MutatedConcreteState) : MutatedConcreteState :=
  { state with lifecycle := .CancellationAccepted }

def MutatedConcreteAcquireOwnershipState (state : MutatedConcreteState) : MutatedConcreteState :=
  { state with ownerEpoch := .Epoch1 }

def MutatedConcreteCommitCancellationState (state : MutatedConcreteState) : MutatedConcreteState :=
  { state with lifecycle := .Cancelled }

def MutatedConcreteWorkerReturnsSuccessState (state : MutatedConcreteState) : MutatedConcreteState :=
  { state with task := .Returned, completionEpoch := state.workerEpoch }

def MutatedConcretePersistSuccessState (state : MutatedConcreteState) : MutatedConcreteState :=
  { state with lifecycle := .Succeeded }

@[simp] private theorem stateToMutated_dispatchState (state : MutatedConcreteState) :
    { stateToMutated state with task := .Dispatched, workerEpoch := (stateToMutated state).ownerEpoch } =
      stateToMutated (MutatedConcreteDispatchState state) := rfl

@[simp] private theorem stateToMutated_requestCancellationState (state : MutatedConcreteState) :
    { stateToMutated state with lifecycle := .CancellationAccepted } =
      stateToMutated (MutatedConcreteRequestCancellationState state) := rfl

@[simp] private theorem stateToMutated_acquireOwnershipState (state : MutatedConcreteState) :
    { stateToMutated state with ownerEpoch := .Epoch1 } =
      stateToMutated (MutatedConcreteAcquireOwnershipState state) := rfl

@[simp] private theorem stateToMutated_commitCancellationState (state : MutatedConcreteState) :
    { stateToMutated state with lifecycle := .Cancelled } =
      stateToMutated (MutatedConcreteCommitCancellationState state) := rfl

@[simp] private theorem stateToMutated_workerReturnsSuccessState (state : MutatedConcreteState) :
    { stateToMutated state with task := .Returned, completionEpoch := (stateToMutated state).workerEpoch } =
      stateToMutated (MutatedConcreteWorkerReturnsSuccessState state) := rfl

@[simp] private theorem stateToMutated_persistSuccessState (state : MutatedConcreteState) :
    { stateToMutated state with lifecycle := .Succeeded } =
      stateToMutated (MutatedConcretePersistSuccessState state) := rfl

def MutatedConcreteOutcomes (state : MutatedConcreteState) (label : MutatedConcreteLabel) :
    List (Veil.ExecutionOutcome Int MutatedConcreteState) :=
  Veil.Extract.extractAllOutcomes
    (NexusCancellationStaleCompletionGuardRemovedConcrete.NextAct.extracted
      MutatedConcreteTheoryType MutatedConcreteState
      MutatedConcreteLifecycle MutatedConcreteTaskStage MutatedConcreteEpoch label)
    mutatedConcreteTheory state

theorem MutatedConcreteSystemNext_iff (state nextState : MutatedConcreteState) (label : MutatedConcreteLabel) :
    MutatedConcreteNext state label nextState ↔
      Veil.ExecutionOutcome.success nextState ∈ MutatedConcreteOutcomes state label := by
  cases label <;>
    simp [MutatedConcreteNext, MutatedConcreteSystem, MutatedConcreteOutcomes,
      NexusCancellationStaleCompletionGuardRemovedConcrete.enumerableTransitionSystem] <;>
    intro <;> exact Veil.Enumeration.complete _

theorem MutatedConcreteDispatchOutcomes_eq (state : MutatedConcreteState) :
    MutatedConcreteOutcomes state .DispatchTask =
      if state.task = .Idle ∧ state.lifecycle = .Open then
        [.success (MutatedConcreteDispatchState state)]
      else [] := by
  rcases state with ⟨lifecycle, task, ownerEpoch, workerEpoch, completionEpoch⟩
  cases lifecycle <;> cases task <;> rfl

theorem MutatedConcreteRequestCancellationOutcomes_eq (state : MutatedConcreteState) :
    MutatedConcreteOutcomes state .RequestCancellation =
      if state.lifecycle = .Open then
        [.success (MutatedConcreteRequestCancellationState state)]
      else [] := by
  rcases state with ⟨lifecycle, task, ownerEpoch, workerEpoch, completionEpoch⟩
  cases lifecycle <;> rfl

theorem MutatedConcreteAcquireOwnershipOutcomes_eq (state : MutatedConcreteState) :
    MutatedConcreteOutcomes state .AcquireOwnership =
      if (((state.lifecycle = .Open ∨ state.lifecycle = .CancellationAccepted) ∨
          state.lifecycle = .Cancelled) ∧ state.ownerEpoch = .Epoch0) then
        [.success (MutatedConcreteAcquireOwnershipState state)]
      else [] := by
  rcases state with ⟨lifecycle, task, ownerEpoch, workerEpoch, completionEpoch⟩
  cases lifecycle <;> cases ownerEpoch <;> rfl

theorem MutatedConcreteCommitCancellationOutcomes_eq (state : MutatedConcreteState) :
    MutatedConcreteOutcomes state .CommitCancellation =
      if state.lifecycle = .CancellationAccepted then
        [.success (MutatedConcreteCommitCancellationState state)]
      else [] := by
  rcases state with ⟨lifecycle, task, ownerEpoch, workerEpoch, completionEpoch⟩
  cases lifecycle <;> rfl

theorem MutatedConcreteWorkerReturnsSuccessOutcomes_eq (state : MutatedConcreteState) :
    MutatedConcreteOutcomes state .WorkerReturnsSuccess =
      if state.task = .Dispatched then
        [.success (MutatedConcreteWorkerReturnsSuccessState state)]
      else [] := by
  rcases state with ⟨lifecycle, task, ownerEpoch, workerEpoch, completionEpoch⟩
  cases task <;> rfl

theorem MutatedConcretePersistSuccessOutcomes_eq (state : MutatedConcreteState) :
    MutatedConcreteOutcomes state .PersistSuccess =
      if state.task = .Returned ∧ state.completionEpoch ≠ .None then
        [.success (MutatedConcretePersistSuccessState state)]
      else [] := by
  rcases state with ⟨lifecycle, task, ownerEpoch, workerEpoch, completionEpoch⟩
  cases task <;> cases completionEpoch <;> rfl

theorem MutatedConcreteNext_mutatedNext_iff (state : MutatedConcreteState) (label : MutatedConcreteLabel)
    (nextState : MutatedConcreteState) :
    MutatedConcreteNext state label nextState ↔
      mutatedSemanticRelation.next (stateToMutated state) (labelToMutated label)
        (stateToMutated nextState) := by
  rw [MutatedConcreteSystemNext_iff]
  cases label <;> simp only [labelToMutated]
  · rw [MutatedConcreteDispatchOutcomes_eq, mutatedSemanticRelation_dispatch_iff]
    rcases state with ⟨lifecycle, task, ownerEpoch, workerEpoch, completionEpoch⟩
    cases task <;> cases lifecycle <;> simp [taskToMutated, lifecycleToMutated]
    constructor
    · intro equal
      subst nextState
      rfl
    · intro equal
      apply stateToMutated_injective
      simpa [stateToMutated, MutatedConcreteDispatchState] using equal.symm
  · rw [MutatedConcreteRequestCancellationOutcomes_eq,
      mutatedSemanticRelation_requestCancellation_iff]
    rcases state with ⟨lifecycle, task, ownerEpoch, workerEpoch, completionEpoch⟩
    cases lifecycle <;> simp [lifecycleToMutated]
    constructor
    · intro equal
      subst nextState
      rfl
    · intro equal
      apply stateToMutated_injective
      simpa [stateToMutated, MutatedConcreteRequestCancellationState] using equal.symm
  · rw [MutatedConcreteAcquireOwnershipOutcomes_eq,
      mutatedSemanticRelation_acquireOwnership_iff]
    rcases state with ⟨lifecycle, task, ownerEpoch, workerEpoch, completionEpoch⟩
    cases lifecycle <;> cases ownerEpoch <;> simp [lifecycleToMutated, epochToMutated]
    all_goals
      constructor
      · intro equal
        subst nextState
        rfl
      · intro equal
        apply stateToMutated_injective
        simpa [stateToMutated, MutatedConcreteAcquireOwnershipState] using equal.symm
  · rw [MutatedConcreteCommitCancellationOutcomes_eq,
      mutatedSemanticRelation_commitCancellation_iff]
    rcases state with ⟨lifecycle, task, ownerEpoch, workerEpoch, completionEpoch⟩
    cases lifecycle <;> simp [lifecycleToMutated]
    constructor
    · intro equal
      subst nextState
      rfl
    · intro equal
      apply stateToMutated_injective
      simpa [stateToMutated, MutatedConcreteCommitCancellationState] using equal.symm
  · rw [MutatedConcreteWorkerReturnsSuccessOutcomes_eq,
      mutatedSemanticRelation_workerReturnsSuccess_iff]
    rcases state with ⟨lifecycle, task, ownerEpoch, workerEpoch, completionEpoch⟩
    cases task <;> simp [taskToMutated]
    constructor
    · intro equal
      subst nextState
      rfl
    · intro equal
      apply stateToMutated_injective
      simpa [stateToMutated, MutatedConcreteWorkerReturnsSuccessState] using equal.symm
  · rw [MutatedConcretePersistSuccessOutcomes_eq, mutatedSemanticRelation_persistSuccess_iff]
    rcases state with ⟨lifecycle, task, ownerEpoch, workerEpoch, completionEpoch⟩
    cases task <;> cases completionEpoch <;>
      simp [taskToMutated, epochToMutated]
    all_goals
      constructor
      · intro equal
        subst nextState
        rfl
      · intro equal
        apply stateToMutated_injective
        simpa [stateToMutated, MutatedConcretePersistSuccessState] using equal.symm

theorem MutatedConcreteInitial_mutatedInitial_iff (state : MutatedConcreteState) :
    MutatedConcreteInitial state ↔ mutatedSemanticRelation.initial (stateToMutated state) := by
  unfold MutatedConcreteInitial
  rw [MutatedConcreteInitialStates_eq]
  rw [mutatedSemanticRelation_initialState_iff]
  simp only [List.mem_singleton]
  exact stateToMutated_injective.eq_iff.symm

theorem MutatedConcreteProperty_mutatedProperty_iff (state : MutatedConcreteState) :
    MutatedConcreteProperty state ↔ mutatedSemanticRelation.property (stateToMutated state) := by
  rw [mutatedSemanticRelation_propertyState_iff]
  unfold MutatedConcreteProperty NexusCancellationStaleCompletionGuardRemovedConcrete.NexusCancellationWonExcludesSuccess
  rcases state with ⟨lifecycle, task, ownerEpoch, workerEpoch, completionEpoch⟩
  cases lifecycle <;> cases task <;> cases ownerEpoch <;> cases workerEpoch <;>
    cases completionEpoch <;>
    decide

theorem MutatedConcreteInitial_iff (state : MutatedConcreteState) :
    MutatedConcreteInitial state ↔
      Umpire3.Temporal.Targets.NexusCancellationFencing.mutatedFirstOrderArtifact.initial.eval
        (encodeMutatedConcreteState state) = true :=
  (MutatedConcreteInitial_mutatedInitial_iff state).trans
    (mutatedSemanticRelation.initial_iff (stateToMutated state))

theorem MutatedConcreteNext_iff (state : MutatedConcreteState) (label : MutatedConcreteLabel)
    (nextState : MutatedConcreteState) :
    MutatedConcreteNext state label nextState ↔
      Umpire3.Temporal.Targets.NexusCancellationFencing.mutatedFirstOrderArtifact.next
          (encodeMutatedConcreteState state) (encodeMutatedConcreteLabel label) =
        some (encodeMutatedConcreteState nextState) :=
  (MutatedConcreteNext_mutatedNext_iff state label nextState).trans
    (mutatedSemanticRelation.next_iff
      (stateToMutated state) (labelToMutated label) (stateToMutated nextState))

theorem MutatedConcreteProperty_iff (state : MutatedConcreteState) :
    MutatedConcreteProperty state ↔
      Umpire3.Temporal.Targets.NexusCancellationFencing.mutatedFirstOrderArtifact.invariant.eval
        (encodeMutatedConcreteState state) = true :=
  (MutatedConcreteProperty_mutatedProperty_iff state).trans
    (mutatedSemanticRelation.property_iff (stateToMutated state))

def mutatedConcreteSemanticRelation :
    Umpire3.Veil.SemanticRelation
      Umpire3.Temporal.Targets.NexusCancellationFencing.mutatedFirstOrderArtifact where
  State := MutatedConcreteState
  Label := MutatedConcreteLabel
  initial := MutatedConcreteInitial
  next := MutatedConcreteNext
  property := MutatedConcreteProperty
  encodeState := encodeMutatedConcreteState
  encodeLabel := encodeMutatedConcreteLabel
  initial_iff := MutatedConcreteInitial_iff
  next_iff := MutatedConcreteNext_iff
  property_iff := MutatedConcreteProperty_iff
  state_injective := by
    intro left right equal
    apply stateToMutated_injective
    apply mutatedSemanticRelation.state_injective
    simpa [encodeMutatedConcreteState] using equal
  label_injective := by
    intro left right equal
    apply labelToMutated_injective
    apply mutatedSemanticRelation.label_injective
    simpa [encodeMutatedConcreteLabel] using equal
  label_total := by
    intro label
    simpa [encodeMutatedConcreteLabel] using mutatedSemanticRelation.label_total (labelToMutated label)
  label_complete := by
    intro identifier member
    obtain ⟨label, equal⟩ := mutatedSemanticRelation.label_complete identifier member
    cases label
    · exact ⟨.DispatchTask, by simpa [encodeMutatedConcreteLabel, labelToMutated] using equal⟩
    · exact ⟨.RequestCancellation, by simpa [encodeMutatedConcreteLabel, labelToMutated] using equal⟩
    · exact ⟨.AcquireOwnership, by simpa [encodeMutatedConcreteLabel, labelToMutated] using equal⟩
    · exact ⟨.CommitCancellation, by simpa [encodeMutatedConcreteLabel, labelToMutated] using equal⟩
    · exact ⟨.WorkerReturnsSuccess, by simpa [encodeMutatedConcreteLabel, labelToMutated] using equal⟩
    · exact ⟨.PersistSuccess, by simpa [encodeMutatedConcreteLabel, labelToMutated] using equal⟩

def mutatedSemanticBinding :
    Umpire3.Veil.SemanticBinding
      Umpire3.Temporal.Targets.NexusCancellationFencing.mutatedFirstOrderArtifact where
  symbolic := mutatedSemanticRelation
  concrete := mutatedConcreteSemanticRelation

end Umpire3.Temporal.Veil.NexusCancellationFencing
