import Temporal.Families.NexusCancellation.Targets.FirstOrder
import Temporal.Families.NexusCancellation.Targets.Veil.Mutated
import Umpire3.Veil.Semantics

namespace Umpire3.Temporal.Veil.NexusCancellationFencing

open Umpire3.Temporal.Targets.NexusCancellationFencing

set_option maxHeartbeats 400000

abbrev MutatedLifecycle := NexusCancellationStaleCompletionGuardRemoved.Lifecycle_IndT
abbrev MutatedTaskStage := NexusCancellationStaleCompletionGuardRemoved.TaskStage_IndT
abbrev MutatedEpoch := NexusCancellationStaleCompletionGuardRemoved.Epoch_IndT
abbrev MutatedField :=
  NexusCancellationStaleCompletionGuardRemoved.FieldConcreteType MutatedLifecycle MutatedTaskStage MutatedEpoch
abbrev MutatedState := NexusCancellationStaleCompletionGuardRemoved.State MutatedField
abbrev MutatedTheory := NexusCancellationStaleCompletionGuardRemoved.Theory MutatedLifecycle MutatedTaskStage MutatedEpoch
abbrev MutatedLabel := NexusCancellationStaleCompletionGuardRemoved.Label MutatedLifecycle MutatedTaskStage MutatedEpoch

private def mutatedStateTuple (state : MutatedState) :
    MutatedLifecycle × MutatedTaskStage × MutatedEpoch × MutatedEpoch × MutatedEpoch :=
  (state.lifecycle, state.task, state.ownerEpoch, state.workerEpoch, state.completionEpoch)

private theorem mutatedStateTuple_injective : Function.Injective mutatedStateTuple := by
  intro left right equal
  rcases left with ⟨leftLifecycle, leftTask, leftOwner, leftWorker, leftCompletion⟩
  rcases right with ⟨rightLifecycle, rightTask, rightOwner, rightWorker, rightCompletion⟩
  simp only [mutatedStateTuple, Prod.mk.injEq] at equal
  rcases equal with ⟨rfl, rfl, rfl, rfl, rfl⟩
  rfl

local instance : DecidableEq MutatedState :=
  mutatedStateTuple_injective.decidableEq

local instance : NexusCancellationStaleCompletionGuardRemoved.Lifecycle_EnumClass MutatedLifecycle :=
  NexusCancellationStaleCompletionGuardRemoved.instLifecycle_EnumClassLifecycle_IndT
local instance : NexusCancellationStaleCompletionGuardRemoved.TaskStage_EnumClass MutatedTaskStage :=
  NexusCancellationStaleCompletionGuardRemoved.instTaskStage_EnumClassTaskStage_IndT
local instance : NexusCancellationStaleCompletionGuardRemoved.Epoch_EnumClass MutatedEpoch :=
  NexusCancellationStaleCompletionGuardRemoved.instEpoch_EnumClassEpoch_IndT
local instance mutatedFieldRepresentation : (field : NexusCancellationStaleCompletionGuardRemoved.State.Label) →
    Veil.FieldRepresentation
      (NexusCancellationStaleCompletionGuardRemoved.State.Label.toDomain MutatedLifecycle MutatedTaskStage MutatedEpoch field)
      (NexusCancellationStaleCompletionGuardRemoved.State.Label.toCodomain MutatedLifecycle MutatedTaskStage MutatedEpoch field)
      (MutatedField field) :=
  NexusCancellationStaleCompletionGuardRemoved.instFieldRepresentation MutatedLifecycle MutatedTaskStage MutatedEpoch
local instance mutatedFieldLawful : (field : NexusCancellationStaleCompletionGuardRemoved.State.Label) →
    Veil.LawfulFieldRepresentation
      (NexusCancellationStaleCompletionGuardRemoved.State.Label.toDomain MutatedLifecycle MutatedTaskStage MutatedEpoch field)
      (NexusCancellationStaleCompletionGuardRemoved.State.Label.toCodomain MutatedLifecycle MutatedTaskStage MutatedEpoch field)
      (MutatedField field)
      (mutatedFieldRepresentation field) :=
  NexusCancellationStaleCompletionGuardRemoved.instLawfulFieldRepresentation MutatedLifecycle MutatedTaskStage MutatedEpoch
local instance : Inhabited MutatedState :=
  NexusCancellationStaleCompletionGuardRemoved.instInhabitedStateFieldConcreteType

private def mutatedTheory : MutatedTheory := {}

private def mutatedInitial (state : MutatedState) : Prop :=
  NexusCancellationStaleCompletionGuardRemoved.Init MutatedTheory MutatedState MutatedLifecycle MutatedTaskStage MutatedEpoch
    MutatedField mutatedTheory state

private def mutatedNext (state : MutatedState) (label : MutatedLabel) (nextState : MutatedState) : Prop :=
  NexusCancellationStaleCompletionGuardRemoved.Next MutatedTheory MutatedState MutatedLifecycle MutatedTaskStage MutatedEpoch
    MutatedField mutatedTheory state label nextState

private def mutatedProperty (state : MutatedState) : Prop :=
  NexusCancellationStaleCompletionGuardRemoved.NexusCancellationWonExcludesSuccess
    (th := mutatedTheory) (st := state)

private def encodeLifecycle : MutatedLifecycle → String
  | .Open => "open"
  | .CancellationAccepted => "cancellation-accepted"
  | .Cancelled => "cancelled"
  | .Succeeded => "succeeded"

private def encodeTask : MutatedTaskStage → String
  | .Idle => "idle"
  | .Dispatched => "dispatched"
  | .Returned => "returned"

private def encodeEpoch : MutatedEpoch → String
  | .None => "none"
  | .Epoch0 => "epoch-0"
  | .Epoch1 => "epoch-1"

private theorem encodeLifecycle_injective : Function.Injective encodeLifecycle := by
  intro left right equal
  cases left <;> cases right <;> simp_all [encodeLifecycle]

private theorem encodeTask_injective : Function.Injective encodeTask := by
  intro left right equal
  cases left <;> cases right <;> simp_all [encodeTask]

private theorem encodeEpoch_injective : Function.Injective encodeEpoch := by
  intro left right equal
  cases left <;> cases right <;> simp_all [encodeEpoch]

@[simp] private theorem encodeLifecycle_eq_open (lifecycle : MutatedLifecycle) :
    encodeLifecycle lifecycle = "open" ↔ lifecycle = .Open := by
  cases lifecycle <;> decide

@[simp] private theorem encodeTask_eq_idle (task : MutatedTaskStage) :
    encodeTask task = "idle" ↔ task = .Idle := by
  cases task <;> decide

@[simp] private theorem encodeTask_eq_dispatched (task : MutatedTaskStage) :
    encodeTask task = "dispatched" ↔ task = .Dispatched := by
  cases task <;> decide

@[simp] private theorem encodeLifecycle_eq_cancellationAccepted (lifecycle : MutatedLifecycle) :
    encodeLifecycle lifecycle = "cancellation-accepted" ↔
      lifecycle = .CancellationAccepted := by
  cases lifecycle <;> decide

@[simp] private theorem encodeLifecycle_eq_cancelled (lifecycle : MutatedLifecycle) :
    encodeLifecycle lifecycle = "cancelled" ↔ lifecycle = .Cancelled := by
  cases lifecycle <;> decide

@[simp] private theorem encodeTask_eq_returned (task : MutatedTaskStage) :
    encodeTask task = "returned" ↔ task = .Returned := by
  cases task <;> decide

@[simp] private theorem encodeEpoch_eq_epoch0 (epoch : MutatedEpoch) :
    encodeEpoch epoch = "epoch-0" ↔ epoch = .Epoch0 := by
  cases epoch <;> decide

@[simp] private theorem encodeEpoch_eq_none (epoch : MutatedEpoch) :
    encodeEpoch epoch = "none" ↔ epoch = .None := by
  cases epoch <;> decide

@[simp] private theorem open_eq_encodeLifecycle (lifecycle : MutatedLifecycle) :
    "open" = encodeLifecycle lifecycle ↔ .Open = lifecycle := by
  cases lifecycle <;> decide

@[simp] private theorem dispatched_eq_encodeTask (task : MutatedTaskStage) :
    "dispatched" = encodeTask task ↔ .Dispatched = task := by
  cases task <;> decide

@[simp] private theorem cancellationAccepted_eq_encodeLifecycle (lifecycle : MutatedLifecycle) :
    "cancellation-accepted" = encodeLifecycle lifecycle ↔
      .CancellationAccepted = lifecycle := by
  cases lifecycle <;> decide

@[simp] private theorem cancelled_eq_encodeLifecycle (lifecycle : MutatedLifecycle) :
    "cancelled" = encodeLifecycle lifecycle ↔ .Cancelled = lifecycle := by
  cases lifecycle <;> decide

@[simp] private theorem succeeded_eq_encodeLifecycle (lifecycle : MutatedLifecycle) :
    "succeeded" = encodeLifecycle lifecycle ↔ .Succeeded = lifecycle := by
  cases lifecycle <;> decide

@[simp] private theorem returned_eq_encodeTask (task : MutatedTaskStage) :
    "returned" = encodeTask task ↔ .Returned = task := by
  cases task <;> decide

@[simp] private theorem epoch1_eq_encodeEpoch (epoch : MutatedEpoch) :
    "epoch-1" = encodeEpoch epoch ↔ .Epoch1 = epoch := by
  cases epoch <;> decide

@[simp] private theorem encodeLifecycle_eq_encode (left right : MutatedLifecycle) :
    encodeLifecycle left = encodeLifecycle right ↔ left = right :=
  encodeLifecycle_injective.eq_iff

@[simp] private theorem encodeTask_eq_encode (left right : MutatedTaskStage) :
    encodeTask left = encodeTask right ↔ left = right :=
  encodeTask_injective.eq_iff

@[simp] private theorem encodeEpoch_eq_encode (left right : MutatedEpoch) :
    encodeEpoch left = encodeEpoch right ↔ left = right :=
  encodeEpoch_injective.eq_iff

private def encodeMutatedState (state : MutatedState) : FirstOrderState where
  fields := [
    { field := "lifecycle", value := encodeLifecycle state.lifecycle },
    { field := "task", value := encodeTask state.task },
    { field := "owner-epoch", value := encodeEpoch state.ownerEpoch },
    { field := "worker-epoch", value := encodeEpoch state.workerEpoch },
    { field := "completion-epoch", value := encodeEpoch state.completionEpoch },
  ]

private theorem encodeMutatedState_injective : Function.Injective encodeMutatedState := by
  intro left right equal
  rcases left with ⟨leftLifecycle, leftTask, leftOwner, leftWorker, leftCompletion⟩
  rcases right with ⟨rightLifecycle, rightTask, rightOwner, rightWorker, rightCompletion⟩
  simp only [encodeMutatedState, FirstOrderState.mk.injEq, List.cons.injEq,
    FirstOrderBinding.mk.injEq, true_and] at equal
  rcases equal with ⟨lifecycleEqual, taskEqual, ownerEqual, workerEqual, completionEqual⟩
  have := encodeLifecycle_injective lifecycleEqual
  have := encodeTask_injective taskEqual
  have := encodeEpoch_injective ownerEqual
  have := encodeEpoch_injective workerEqual
  have := encodeEpoch_injective completionEqual.1
  subst_vars
  rfl

private def encodeMutatedLabel : MutatedLabel → String
  | .DispatchTask => "dispatch-task"
  | .RequestCancellation => "request-cancellation"
  | .AcquireOwnership => "acquire-ownership"
  | .CommitCancellation => "commit-cancellation"
  | .WorkerReturnsSuccess => "worker-returns-success"
  | .PersistSuccess => "persist-success"

private theorem encodeMutatedLabel_injective : Function.Injective encodeMutatedLabel := by
  intro left right equal
  cases left <;> cases right <;> simp_all [encodeMutatedLabel]

private theorem mutatedInitial_iff (state : MutatedState) :
    mutatedInitial state ↔ mutatedFirstOrderArtifact.initial.eval (encodeMutatedState state) = true := by
  rcases state with ⟨lifecycle, task, ownerEpoch, workerEpoch, completionEpoch⟩
  cases lifecycle <;> cases task <;> cases ownerEpoch <;> cases workerEpoch <;>
    cases completionEpoch <;>
    simp [mutatedInitial, NexusCancellationStaleCompletionGuardRemoved.Init,
      NexusCancellationStaleCompletionGuardRemoved.initializer.ext.tr] <;>
    decide

private theorem mutatedProperty_iff (state : MutatedState) :
    mutatedProperty state ↔
      mutatedFirstOrderArtifact.invariant.eval (encodeMutatedState state) = true := by
  rcases state with ⟨lifecycle, task, ownerEpoch, workerEpoch, completionEpoch⟩
  cases lifecycle <;> cases task <;> cases ownerEpoch <;> cases workerEpoch <;>
    cases completionEpoch <;>
    simp [mutatedProperty, NexusCancellationStaleCompletionGuardRemoved.NexusCancellationWonExcludesSuccess] <;>
    decide

private def dispatchState (state : MutatedState) : MutatedState :=
  { state with task := .Dispatched, workerEpoch := state.ownerEpoch }

private def requestCancellationState (state : MutatedState) : MutatedState :=
  { state with lifecycle := .CancellationAccepted }

private def acquireOwnershipState (state : MutatedState) : MutatedState :=
  { state with ownerEpoch := .Epoch1 }

private def commitCancellationState (state : MutatedState) : MutatedState :=
  { state with lifecycle := .Cancelled }

private def workerReturnsSuccessState (state : MutatedState) : MutatedState :=
  { state with task := .Returned, completionEpoch := state.workerEpoch }

private def persistSuccessState (state : MutatedState) : MutatedState :=
  { state with lifecycle := .Succeeded }

private theorem mutatedDispatch_iff (state nextState : MutatedState) :
    mutatedNext state .DispatchTask nextState ↔
      state.task = .Idle ∧ state.lifecycle = .Open ∧ dispatchState state = nextState := by
  unfold mutatedNext NexusCancellationStaleCompletionGuardRemoved.Next NexusCancellationStaleCompletionGuardRemoved.NextAct
  rw [NexusCancellationStaleCompletionGuardRemoved.DispatchTask.ext.derived_eq]
  rfl

private theorem mutatedRequestCancellation_iff (state nextState : MutatedState) :
    mutatedNext state .RequestCancellation nextState ↔
      state.lifecycle = .Open ∧ requestCancellationState state = nextState := by
  unfold mutatedNext NexusCancellationStaleCompletionGuardRemoved.Next NexusCancellationStaleCompletionGuardRemoved.NextAct
  rw [NexusCancellationStaleCompletionGuardRemoved.RequestCancellation.ext.derived_eq]
  rfl

private theorem mutatedAcquireOwnership_iff (state nextState : MutatedState) :
    mutatedNext state .AcquireOwnership nextState ↔
      ((state.lifecycle = .Open ∨ state.lifecycle = .CancellationAccepted) ∨
          state.lifecycle = .Cancelled) ∧ state.ownerEpoch = .Epoch0 ∧
        acquireOwnershipState state = nextState := by
  unfold mutatedNext NexusCancellationStaleCompletionGuardRemoved.Next NexusCancellationStaleCompletionGuardRemoved.NextAct
  rw [NexusCancellationStaleCompletionGuardRemoved.AcquireOwnership.ext.derived_eq]
  rfl

private theorem mutatedCommitCancellation_iff (state nextState : MutatedState) :
    mutatedNext state .CommitCancellation nextState ↔
      state.lifecycle = .CancellationAccepted ∧ commitCancellationState state = nextState := by
  unfold mutatedNext NexusCancellationStaleCompletionGuardRemoved.Next NexusCancellationStaleCompletionGuardRemoved.NextAct
  rw [NexusCancellationStaleCompletionGuardRemoved.CommitCancellation.ext.derived_eq]
  rfl

private theorem mutatedWorkerReturnsSuccess_iff (state nextState : MutatedState) :
    mutatedNext state .WorkerReturnsSuccess nextState ↔
      state.task = .Dispatched ∧ workerReturnsSuccessState state = nextState := by
  unfold mutatedNext NexusCancellationStaleCompletionGuardRemoved.Next NexusCancellationStaleCompletionGuardRemoved.NextAct
  rw [NexusCancellationStaleCompletionGuardRemoved.WorkerReturnsSuccess.ext.derived_eq]
  rfl

private theorem mutatedPersistSuccess_iff (state nextState : MutatedState) :
    mutatedNext state .PersistSuccess nextState ↔
      state.task = .Returned ∧ state.completionEpoch ≠ .None ∧
        persistSuccessState state = nextState := by
  unfold mutatedNext NexusCancellationStaleCompletionGuardRemoved.Next NexusCancellationStaleCompletionGuardRemoved.NextAct
  rw [NexusCancellationStaleCompletionGuardRemoved.PersistSuccess.ext.derived_eq]
  simp only [NexusCancellationStaleCompletionGuardRemoved.PersistSuccess.ext.tr, persistSuccessState]
  rfl

private theorem firstOrderDispatch_iff (state nextState : MutatedState) :
    mutatedFirstOrderArtifact.next (encodeMutatedState state) "dispatch-task" =
        some (encodeMutatedState nextState) ↔
      state.task = .Idle ∧ state.lifecycle = .Open ∧ dispatchState state = nextState := by
  rcases state with ⟨lifecycle, task, ownerEpoch, workerEpoch, completionEpoch⟩
  rcases nextState with ⟨nextLifecycle, nextTask, nextOwner, nextWorker, nextCompletion⟩
  cases task <;> cases lifecycle <;>
    simp [encodeMutatedState, mutatedFirstOrderArtifact, firstOrderArtifact,
      baseFirstOrderActions, field, value, equalFieldValue, all,
      Umpire3.Temporal.Targets.NexusCancellationFencing.any, setValue, copyValue,
      FirstOrderArtifact.next, FirstOrderAction.apply, FirstOrderFormula.eval,
      FirstOrderTerm.eval, FirstOrderState.read, FirstOrderState.write,
      applyFirstOrderUpdates, dispatchState]

private theorem firstOrderRequestCancellation_iff (state nextState : MutatedState) :
    mutatedFirstOrderArtifact.next (encodeMutatedState state) "request-cancellation" =
        some (encodeMutatedState nextState) ↔
      state.lifecycle = .Open ∧ requestCancellationState state = nextState := by
  rcases state with ⟨lifecycle, task, ownerEpoch, workerEpoch, completionEpoch⟩
  rcases nextState with ⟨nextLifecycle, nextTask, nextOwner, nextWorker, nextCompletion⟩
  cases lifecycle <;>
    simp [encodeMutatedState, mutatedFirstOrderArtifact, firstOrderArtifact,
      baseFirstOrderActions, field, value, equalFieldValue, all,
      Umpire3.Temporal.Targets.NexusCancellationFencing.any, setValue, copyValue,
      FirstOrderArtifact.next, FirstOrderAction.apply, FirstOrderFormula.eval,
      FirstOrderTerm.eval, FirstOrderState.read, FirstOrderState.write,
      applyFirstOrderUpdates, requestCancellationState]

private theorem firstOrderAcquireOwnership_iff (state nextState : MutatedState) :
    mutatedFirstOrderArtifact.next (encodeMutatedState state) "acquire-ownership" =
        some (encodeMutatedState nextState) ↔
      ((state.lifecycle = .Open ∨ state.lifecycle = .CancellationAccepted) ∨
          state.lifecycle = .Cancelled) ∧ state.ownerEpoch = .Epoch0 ∧
        acquireOwnershipState state = nextState := by
  rcases state with ⟨lifecycle, task, ownerEpoch, workerEpoch, completionEpoch⟩
  rcases nextState with ⟨nextLifecycle, nextTask, nextOwner, nextWorker, nextCompletion⟩
  cases lifecycle <;> cases ownerEpoch <;>
    simp [encodeMutatedState, mutatedFirstOrderArtifact, firstOrderArtifact,
      baseFirstOrderActions, field, value, equalFieldValue, all,
      Umpire3.Temporal.Targets.NexusCancellationFencing.any, setValue, copyValue,
      FirstOrderArtifact.next, FirstOrderAction.apply, FirstOrderFormula.eval,
      FirstOrderTerm.eval, FirstOrderState.read, FirstOrderState.write,
      applyFirstOrderUpdates, acquireOwnershipState]

private theorem firstOrderCommitCancellation_iff (state nextState : MutatedState) :
    mutatedFirstOrderArtifact.next (encodeMutatedState state) "commit-cancellation" =
        some (encodeMutatedState nextState) ↔
      state.lifecycle = .CancellationAccepted ∧ commitCancellationState state = nextState := by
  rcases state with ⟨lifecycle, task, ownerEpoch, workerEpoch, completionEpoch⟩
  rcases nextState with ⟨nextLifecycle, nextTask, nextOwner, nextWorker, nextCompletion⟩
  cases lifecycle <;>
    simp [encodeMutatedState, mutatedFirstOrderArtifact, firstOrderArtifact,
      baseFirstOrderActions, field, value, equalFieldValue, all,
      Umpire3.Temporal.Targets.NexusCancellationFencing.any, setValue, copyValue,
      FirstOrderArtifact.next, FirstOrderAction.apply, FirstOrderFormula.eval,
      FirstOrderTerm.eval, FirstOrderState.read, FirstOrderState.write,
      applyFirstOrderUpdates, commitCancellationState]

private theorem firstOrderWorkerReturnsSuccess_iff (state nextState : MutatedState) :
    mutatedFirstOrderArtifact.next (encodeMutatedState state) "worker-returns-success" =
        some (encodeMutatedState nextState) ↔
      state.task = .Dispatched ∧ workerReturnsSuccessState state = nextState := by
  rcases state with ⟨lifecycle, task, ownerEpoch, workerEpoch, completionEpoch⟩
  rcases nextState with ⟨nextLifecycle, nextTask, nextOwner, nextWorker, nextCompletion⟩
  cases task <;>
    simp [encodeMutatedState, mutatedFirstOrderArtifact, firstOrderArtifact,
      baseFirstOrderActions, field, value, equalFieldValue, all,
      Umpire3.Temporal.Targets.NexusCancellationFencing.any, setValue, copyValue,
      FirstOrderArtifact.next, FirstOrderAction.apply, FirstOrderFormula.eval,
      FirstOrderTerm.eval, FirstOrderState.read, FirstOrderState.write,
      applyFirstOrderUpdates, workerReturnsSuccessState]

private theorem firstOrderPersistSuccess_iff (state nextState : MutatedState) :
    mutatedFirstOrderArtifact.next (encodeMutatedState state) "persist-success" =
        some (encodeMutatedState nextState) ↔
      state.task = .Returned ∧ state.completionEpoch ≠ .None ∧
        persistSuccessState state = nextState := by
  rcases state with ⟨lifecycle, task, ownerEpoch, workerEpoch, completionEpoch⟩
  rcases nextState with ⟨nextLifecycle, nextTask, nextOwner, nextWorker, nextCompletion⟩
  cases task <;> cases completionEpoch <;>
    simp [encodeMutatedState, mutatedFirstOrderArtifact, firstOrderArtifact,
      baseFirstOrderActions, field, value, equalFieldValue, all,
      Umpire3.Temporal.Targets.NexusCancellationFencing.any, setValue, copyValue,
      FirstOrderArtifact.next, FirstOrderAction.apply, FirstOrderFormula.eval,
      FirstOrderTerm.eval, FirstOrderState.read, FirstOrderState.write,
      applyFirstOrderUpdates, persistSuccessState]

private theorem mutatedNext_iff (state : MutatedState) (label : MutatedLabel)
    (nextState : MutatedState) :
    mutatedNext state label nextState ↔
      mutatedFirstOrderArtifact.next (encodeMutatedState state) (encodeMutatedLabel label) =
        some (encodeMutatedState nextState) := by
  cases label
  · exact (mutatedDispatch_iff state nextState).trans (firstOrderDispatch_iff state nextState).symm
  · exact (mutatedRequestCancellation_iff state nextState).trans
      (firstOrderRequestCancellation_iff state nextState).symm
  · exact (mutatedAcquireOwnership_iff state nextState).trans
      (firstOrderAcquireOwnership_iff state nextState).symm
  · exact (mutatedCommitCancellation_iff state nextState).trans
      (firstOrderCommitCancellation_iff state nextState).symm
  · exact (mutatedWorkerReturnsSuccess_iff state nextState).trans
      (firstOrderWorkerReturnsSuccess_iff state nextState).symm
  · exact (mutatedPersistSuccess_iff state nextState).trans
      (firstOrderPersistSuccess_iff state nextState).symm

def mutatedSemanticRelation : Umpire3.Veil.SemanticRelation mutatedFirstOrderArtifact where
  State := MutatedState
  Label := MutatedLabel
  initial := mutatedInitial
  next := mutatedNext
  property := mutatedProperty
  encodeState := encodeMutatedState
  encodeLabel := encodeMutatedLabel
  initial_iff := mutatedInitial_iff
  next_iff := mutatedNext_iff
  property_iff := mutatedProperty_iff
  state_injective := encodeMutatedState_injective
  label_injective := encodeMutatedLabel_injective
  label_total := by intro label; cases label <;> decide
  label_complete := by
    intro identifier member
    simp [mutatedFirstOrderArtifact, firstOrderArtifact, baseFirstOrderActions,
      FirstOrderArtifact.actionIdentifiers] at member
    rcases member with rfl | rfl | rfl | rfl | rfl | rfl
    · exact ⟨.DispatchTask, rfl⟩
    · exact ⟨.RequestCancellation, rfl⟩
    · exact ⟨.AcquireOwnership, rfl⟩
    · exact ⟨.CommitCancellation, rfl⟩
    · exact ⟨.WorkerReturnsSuccess, rfl⟩
    · exact ⟨.PersistSuccess, rfl⟩

theorem mutatedSemanticRelation_dispatch_iff (state nextState : MutatedState) :
    mutatedSemanticRelation.next state .DispatchTask nextState ↔
      state.task = .Idle ∧ state.lifecycle = .Open ∧
        { state with task := .Dispatched, workerEpoch := state.ownerEpoch } = nextState := by
  simpa [mutatedSemanticRelation, dispatchState] using mutatedDispatch_iff state nextState

theorem mutatedSemanticRelation_requestCancellation_iff (state nextState : MutatedState) :
    mutatedSemanticRelation.next state .RequestCancellation nextState ↔
      state.lifecycle = .Open ∧ { state with lifecycle := .CancellationAccepted } = nextState := by
  simpa [mutatedSemanticRelation, requestCancellationState] using
    mutatedRequestCancellation_iff state nextState

theorem mutatedSemanticRelation_acquireOwnership_iff (state nextState : MutatedState) :
    mutatedSemanticRelation.next state .AcquireOwnership nextState ↔
      ((state.lifecycle = .Open ∨ state.lifecycle = .CancellationAccepted) ∨
          state.lifecycle = .Cancelled) ∧ state.ownerEpoch = .Epoch0 ∧
        { state with ownerEpoch := .Epoch1 } = nextState := by
  simpa [mutatedSemanticRelation, acquireOwnershipState] using
    mutatedAcquireOwnership_iff state nextState

theorem mutatedSemanticRelation_commitCancellation_iff (state nextState : MutatedState) :
    mutatedSemanticRelation.next state .CommitCancellation nextState ↔
      state.lifecycle = .CancellationAccepted ∧
        { state with lifecycle := .Cancelled } = nextState := by
  simpa [mutatedSemanticRelation, commitCancellationState] using
    mutatedCommitCancellation_iff state nextState

theorem mutatedSemanticRelation_workerReturnsSuccess_iff (state nextState : MutatedState) :
    mutatedSemanticRelation.next state .WorkerReturnsSuccess nextState ↔
      state.task = .Dispatched ∧
        { state with task := .Returned, completionEpoch := state.workerEpoch } = nextState := by
  simpa [mutatedSemanticRelation, workerReturnsSuccessState] using
    mutatedWorkerReturnsSuccess_iff state nextState

theorem mutatedSemanticRelation_persistSuccess_iff (state nextState : MutatedState) :
    mutatedSemanticRelation.next state .PersistSuccess nextState ↔
      state.task = .Returned ∧ state.completionEpoch ≠ .None ∧
        { state with lifecycle := .Succeeded } = nextState := by
  simpa [mutatedSemanticRelation, persistSuccessState] using mutatedPersistSuccess_iff state nextState

theorem mutatedSemanticRelation_initialState_iff (state : MutatedState) :
    mutatedSemanticRelation.initial state ↔
      state = {
        lifecycle := .Open
        task := .Idle
        ownerEpoch := .Epoch0
        workerEpoch := .None
        completionEpoch := .None
      } := by
  rcases state with ⟨lifecycle, task, ownerEpoch, workerEpoch, completionEpoch⟩
  cases lifecycle <;> cases task <;> cases ownerEpoch <;> cases workerEpoch <;>
    cases completionEpoch <;>
    simp [mutatedSemanticRelation, mutatedInitial, NexusCancellationStaleCompletionGuardRemoved.Init,
      NexusCancellationStaleCompletionGuardRemoved.initializer.ext.tr] <;>
    decide

theorem mutatedSemanticRelation_propertyState_iff (state : MutatedState) :
    mutatedSemanticRelation.property state ↔
      state.lifecycle ≠ .Succeeded ∨ state.completionEpoch = state.ownerEpoch := by
  unfold mutatedSemanticRelation mutatedProperty
  unfold NexusCancellationStaleCompletionGuardRemoved.NexusCancellationWonExcludesSuccess
  rfl

end Umpire3.Temporal.Veil.NexusCancellationFencing
