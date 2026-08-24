import Temporal.Families.NexusCancellation.Targets.FirstOrder
import Temporal.Families.NexusCancellation.Targets.Veil.SoundTrusted
import Temporal.Families.NexusCancellation.Targets.Veil.SoundConcreteSemantics
import Umpire3.Veil.Semantics

namespace Umpire3.Temporal.Veil.NexusCancellationFencing

open Umpire3.Temporal.Targets.NexusCancellationFencing

set_option maxHeartbeats 400000

abbrev TrustedLifecycle := NexusCancellationSoundTrusted.Lifecycle_IndT
abbrev TrustedTaskStage := NexusCancellationSoundTrusted.TaskStage_IndT
abbrev TrustedEpoch := NexusCancellationSoundTrusted.Epoch_IndT
abbrev TrustedField :=
  NexusCancellationSoundTrusted.FieldConcreteType TrustedLifecycle TrustedTaskStage TrustedEpoch
abbrev TrustedState := NexusCancellationSoundTrusted.State TrustedField
abbrev TrustedTheory := NexusCancellationSoundTrusted.Theory TrustedLifecycle TrustedTaskStage TrustedEpoch
abbrev TrustedLabel := NexusCancellationSoundTrusted.Label TrustedLifecycle TrustedTaskStage TrustedEpoch

private def trustedStateTuple (state : TrustedState) :
    TrustedLifecycle × TrustedTaskStage × TrustedEpoch × TrustedEpoch × TrustedEpoch :=
  (state.lifecycle, state.task, state.ownerEpoch, state.workerEpoch, state.completionEpoch)

private theorem trustedStateTuple_injective : Function.Injective trustedStateTuple := by
  intro left right equal
  rcases left with ⟨leftLifecycle, leftTask, leftOwner, leftWorker, leftCompletion⟩
  rcases right with ⟨rightLifecycle, rightTask, rightOwner, rightWorker, rightCompletion⟩
  simp only [trustedStateTuple, Prod.mk.injEq] at equal
  rcases equal with ⟨rfl, rfl, rfl, rfl, rfl⟩
  rfl

local instance : DecidableEq TrustedState :=
  trustedStateTuple_injective.decidableEq

local instance : NexusCancellationSoundTrusted.Lifecycle_EnumClass TrustedLifecycle :=
  NexusCancellationSoundTrusted.instLifecycle_EnumClassLifecycle_IndT
local instance : NexusCancellationSoundTrusted.TaskStage_EnumClass TrustedTaskStage :=
  NexusCancellationSoundTrusted.instTaskStage_EnumClassTaskStage_IndT
local instance : NexusCancellationSoundTrusted.Epoch_EnumClass TrustedEpoch :=
  NexusCancellationSoundTrusted.instEpoch_EnumClassEpoch_IndT
local instance trustedFieldRepresentation : (field : NexusCancellationSoundTrusted.State.Label) →
    Veil.FieldRepresentation
      (NexusCancellationSoundTrusted.State.Label.toDomain TrustedLifecycle TrustedTaskStage TrustedEpoch field)
      (NexusCancellationSoundTrusted.State.Label.toCodomain TrustedLifecycle TrustedTaskStage TrustedEpoch field)
      (TrustedField field) :=
  NexusCancellationSoundTrusted.instFieldRepresentation TrustedLifecycle TrustedTaskStage TrustedEpoch
local instance trustedFieldLawful : (field : NexusCancellationSoundTrusted.State.Label) →
    Veil.LawfulFieldRepresentation
      (NexusCancellationSoundTrusted.State.Label.toDomain TrustedLifecycle TrustedTaskStage TrustedEpoch field)
      (NexusCancellationSoundTrusted.State.Label.toCodomain TrustedLifecycle TrustedTaskStage TrustedEpoch field)
      (TrustedField field)
      (trustedFieldRepresentation field) :=
  NexusCancellationSoundTrusted.instLawfulFieldRepresentation TrustedLifecycle TrustedTaskStage TrustedEpoch
local instance : Inhabited TrustedState :=
  NexusCancellationSoundTrusted.instInhabitedStateFieldConcreteType

private def trustedTheory : TrustedTheory := {}

private def trustedInitial (state : TrustedState) : Prop :=
  NexusCancellationSoundTrusted.Init TrustedTheory TrustedState TrustedLifecycle TrustedTaskStage TrustedEpoch
    TrustedField trustedTheory state

private def trustedNext (state : TrustedState) (label : TrustedLabel) (nextState : TrustedState) : Prop :=
  NexusCancellationSoundTrusted.Next TrustedTheory TrustedState TrustedLifecycle TrustedTaskStage TrustedEpoch
    TrustedField trustedTheory state label nextState

private def trustedProperty (state : TrustedState) : Prop :=
  NexusCancellationSoundTrusted.NexusCancellationWonExcludesSuccess
    (th := trustedTheory) (st := state)

private def encodeLifecycle : TrustedLifecycle → String
  | .Open => "open"
  | .CancellationAccepted => "cancellation-accepted"
  | .Cancelled => "cancelled"
  | .Succeeded => "succeeded"

private def encodeTask : TrustedTaskStage → String
  | .Idle => "idle"
  | .Dispatched => "dispatched"
  | .Returned => "returned"

private def encodeEpoch : TrustedEpoch → String
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

@[simp] private theorem encodeLifecycle_eq_open (lifecycle : TrustedLifecycle) :
    encodeLifecycle lifecycle = "open" ↔ lifecycle = .Open := by
  cases lifecycle <;> decide

@[simp] private theorem encodeTask_eq_idle (task : TrustedTaskStage) :
    encodeTask task = "idle" ↔ task = .Idle := by
  cases task <;> decide

@[simp] private theorem encodeTask_eq_dispatched (task : TrustedTaskStage) :
    encodeTask task = "dispatched" ↔ task = .Dispatched := by
  cases task <;> decide

@[simp] private theorem encodeLifecycle_eq_cancellationAccepted (lifecycle : TrustedLifecycle) :
    encodeLifecycle lifecycle = "cancellation-accepted" ↔
      lifecycle = .CancellationAccepted := by
  cases lifecycle <;> decide

@[simp] private theorem encodeLifecycle_eq_cancelled (lifecycle : TrustedLifecycle) :
    encodeLifecycle lifecycle = "cancelled" ↔ lifecycle = .Cancelled := by
  cases lifecycle <;> decide

@[simp] private theorem encodeTask_eq_returned (task : TrustedTaskStage) :
    encodeTask task = "returned" ↔ task = .Returned := by
  cases task <;> decide

@[simp] private theorem encodeEpoch_eq_epoch0 (epoch : TrustedEpoch) :
    encodeEpoch epoch = "epoch-0" ↔ epoch = .Epoch0 := by
  cases epoch <;> decide

@[simp] private theorem open_eq_encodeLifecycle (lifecycle : TrustedLifecycle) :
    "open" = encodeLifecycle lifecycle ↔ .Open = lifecycle := by
  cases lifecycle <;> decide

@[simp] private theorem dispatched_eq_encodeTask (task : TrustedTaskStage) :
    "dispatched" = encodeTask task ↔ .Dispatched = task := by
  cases task <;> decide

@[simp] private theorem cancellationAccepted_eq_encodeLifecycle (lifecycle : TrustedLifecycle) :
    "cancellation-accepted" = encodeLifecycle lifecycle ↔
      .CancellationAccepted = lifecycle := by
  cases lifecycle <;> decide

@[simp] private theorem cancelled_eq_encodeLifecycle (lifecycle : TrustedLifecycle) :
    "cancelled" = encodeLifecycle lifecycle ↔ .Cancelled = lifecycle := by
  cases lifecycle <;> decide

@[simp] private theorem succeeded_eq_encodeLifecycle (lifecycle : TrustedLifecycle) :
    "succeeded" = encodeLifecycle lifecycle ↔ .Succeeded = lifecycle := by
  cases lifecycle <;> decide

@[simp] private theorem returned_eq_encodeTask (task : TrustedTaskStage) :
    "returned" = encodeTask task ↔ .Returned = task := by
  cases task <;> decide

@[simp] private theorem epoch1_eq_encodeEpoch (epoch : TrustedEpoch) :
    "epoch-1" = encodeEpoch epoch ↔ .Epoch1 = epoch := by
  cases epoch <;> decide

@[simp] private theorem encodeLifecycle_eq_encode (left right : TrustedLifecycle) :
    encodeLifecycle left = encodeLifecycle right ↔ left = right :=
  encodeLifecycle_injective.eq_iff

@[simp] private theorem encodeTask_eq_encode (left right : TrustedTaskStage) :
    encodeTask left = encodeTask right ↔ left = right :=
  encodeTask_injective.eq_iff

@[simp] private theorem encodeEpoch_eq_encode (left right : TrustedEpoch) :
    encodeEpoch left = encodeEpoch right ↔ left = right :=
  encodeEpoch_injective.eq_iff

private def encodeTrustedState (state : TrustedState) : FirstOrderState where
  fields := [
    { field := "lifecycle", value := encodeLifecycle state.lifecycle },
    { field := "task", value := encodeTask state.task },
    { field := "owner-epoch", value := encodeEpoch state.ownerEpoch },
    { field := "worker-epoch", value := encodeEpoch state.workerEpoch },
    { field := "completion-epoch", value := encodeEpoch state.completionEpoch },
  ]

private theorem encodeTrustedState_injective : Function.Injective encodeTrustedState := by
  intro left right equal
  rcases left with ⟨leftLifecycle, leftTask, leftOwner, leftWorker, leftCompletion⟩
  rcases right with ⟨rightLifecycle, rightTask, rightOwner, rightWorker, rightCompletion⟩
  simp only [encodeTrustedState, FirstOrderState.mk.injEq, List.cons.injEq,
    FirstOrderBinding.mk.injEq, true_and] at equal
  rcases equal with ⟨lifecycleEqual, taskEqual, ownerEqual, workerEqual, completionEqual⟩
  have := encodeLifecycle_injective lifecycleEqual
  have := encodeTask_injective taskEqual
  have := encodeEpoch_injective ownerEqual
  have := encodeEpoch_injective workerEqual
  have := encodeEpoch_injective completionEqual.1
  subst_vars
  rfl

private def encodeTrustedLabel : TrustedLabel → String
  | .DispatchTask => "dispatch-task"
  | .RequestCancellation => "request-cancellation"
  | .AcquireOwnership => "acquire-ownership"
  | .CommitCancellation => "commit-cancellation"
  | .WorkerReturnsSuccess => "worker-returns-success"
  | .PersistSuccess => "persist-success"

private theorem encodeTrustedLabel_injective : Function.Injective encodeTrustedLabel := by
  intro left right equal
  cases left <;> cases right <;> simp_all [encodeTrustedLabel]

private theorem trustedInitial_iff (state : TrustedState) :
    trustedInitial state ↔ soundFirstOrderArtifact.initial.eval (encodeTrustedState state) = true := by
  rcases state with ⟨lifecycle, task, ownerEpoch, workerEpoch, completionEpoch⟩
  cases lifecycle <;> cases task <;> cases ownerEpoch <;> cases workerEpoch <;>
    cases completionEpoch <;>
    simp [trustedInitial, NexusCancellationSoundTrusted.Init,
      NexusCancellationSoundTrusted.initializer.ext.tr] <;>
    decide

private theorem trustedProperty_iff (state : TrustedState) :
    trustedProperty state ↔
      soundFirstOrderArtifact.invariant.eval (encodeTrustedState state) = true := by
  rcases state with ⟨lifecycle, task, ownerEpoch, workerEpoch, completionEpoch⟩
  cases lifecycle <;> cases task <;> cases ownerEpoch <;> cases workerEpoch <;>
    cases completionEpoch <;>
    simp [trustedProperty, NexusCancellationSoundTrusted.NexusCancellationWonExcludesSuccess] <;>
    decide

private def dispatchState (state : TrustedState) : TrustedState :=
  { state with task := .Dispatched, workerEpoch := state.ownerEpoch }

private def requestCancellationState (state : TrustedState) : TrustedState :=
  { state with lifecycle := .CancellationAccepted }

private def acquireOwnershipState (state : TrustedState) : TrustedState :=
  { state with ownerEpoch := .Epoch1 }

private def commitCancellationState (state : TrustedState) : TrustedState :=
  { state with lifecycle := .Cancelled }

private def workerReturnsSuccessState (state : TrustedState) : TrustedState :=
  { state with task := .Returned, completionEpoch := state.workerEpoch }

private def persistSuccessState (state : TrustedState) : TrustedState :=
  { state with lifecycle := .Succeeded }

private theorem trustedDispatch_iff (state nextState : TrustedState) :
    trustedNext state .DispatchTask nextState ↔
      state.task = .Idle ∧ state.lifecycle = .Open ∧ dispatchState state = nextState := by
  unfold trustedNext NexusCancellationSoundTrusted.Next NexusCancellationSoundTrusted.NextAct
  rw [NexusCancellationSoundTrusted.DispatchTask.ext.derived_eq]
  rfl

private theorem trustedRequestCancellation_iff (state nextState : TrustedState) :
    trustedNext state .RequestCancellation nextState ↔
      state.lifecycle = .Open ∧ requestCancellationState state = nextState := by
  unfold trustedNext NexusCancellationSoundTrusted.Next NexusCancellationSoundTrusted.NextAct
  rw [NexusCancellationSoundTrusted.RequestCancellation.ext.derived_eq]
  rfl

private theorem trustedAcquireOwnership_iff (state nextState : TrustedState) :
    trustedNext state .AcquireOwnership nextState ↔
      ((state.lifecycle = .Open ∨ state.lifecycle = .CancellationAccepted) ∨
          state.lifecycle = .Cancelled) ∧ state.ownerEpoch = .Epoch0 ∧
        acquireOwnershipState state = nextState := by
  unfold trustedNext NexusCancellationSoundTrusted.Next NexusCancellationSoundTrusted.NextAct
  rw [NexusCancellationSoundTrusted.AcquireOwnership.ext.derived_eq]
  rfl

private theorem trustedCommitCancellation_iff (state nextState : TrustedState) :
    trustedNext state .CommitCancellation nextState ↔
      state.lifecycle = .CancellationAccepted ∧ commitCancellationState state = nextState := by
  unfold trustedNext NexusCancellationSoundTrusted.Next NexusCancellationSoundTrusted.NextAct
  rw [NexusCancellationSoundTrusted.CommitCancellation.ext.derived_eq]
  rfl

private theorem trustedWorkerReturnsSuccess_iff (state nextState : TrustedState) :
    trustedNext state .WorkerReturnsSuccess nextState ↔
      state.task = .Dispatched ∧ workerReturnsSuccessState state = nextState := by
  unfold trustedNext NexusCancellationSoundTrusted.Next NexusCancellationSoundTrusted.NextAct
  rw [NexusCancellationSoundTrusted.WorkerReturnsSuccess.ext.derived_eq]
  rfl

private theorem trustedPersistSuccess_iff (state nextState : TrustedState) :
    trustedNext state .PersistSuccess nextState ↔
      state.task = .Returned ∧ state.completionEpoch = state.ownerEpoch ∧
        (state.lifecycle = .Open ∨ state.lifecycle = .CancellationAccepted) ∧
          persistSuccessState state = nextState := by
  unfold trustedNext NexusCancellationSoundTrusted.Next NexusCancellationSoundTrusted.NextAct
  rw [NexusCancellationSoundTrusted.PersistSuccess.ext.derived_eq]
  simp only [NexusCancellationSoundTrusted.PersistSuccess.ext.tr, persistSuccessState]
  rfl

private theorem firstOrderDispatch_iff (state nextState : TrustedState) :
    soundFirstOrderArtifact.next (encodeTrustedState state) "dispatch-task" =
        some (encodeTrustedState nextState) ↔
      state.task = .Idle ∧ state.lifecycle = .Open ∧ dispatchState state = nextState := by
  rcases state with ⟨lifecycle, task, ownerEpoch, workerEpoch, completionEpoch⟩
  rcases nextState with ⟨nextLifecycle, nextTask, nextOwner, nextWorker, nextCompletion⟩
  cases task <;> cases lifecycle <;>
    simp [encodeTrustedState, soundFirstOrderArtifact, firstOrderArtifact,
      baseFirstOrderActions, field, value, equalFieldValue, equalFields, all,
      Umpire3.Temporal.Targets.NexusCancellationFencing.any, setValue, copyValue,
      FirstOrderArtifact.next, FirstOrderAction.apply, FirstOrderFormula.eval,
      FirstOrderTerm.eval, FirstOrderState.read, FirstOrderState.write,
      applyFirstOrderUpdates, dispatchState]

private theorem firstOrderRequestCancellation_iff (state nextState : TrustedState) :
    soundFirstOrderArtifact.next (encodeTrustedState state) "request-cancellation" =
        some (encodeTrustedState nextState) ↔
      state.lifecycle = .Open ∧ requestCancellationState state = nextState := by
  rcases state with ⟨lifecycle, task, ownerEpoch, workerEpoch, completionEpoch⟩
  rcases nextState with ⟨nextLifecycle, nextTask, nextOwner, nextWorker, nextCompletion⟩
  cases lifecycle <;>
    simp [encodeTrustedState, soundFirstOrderArtifact, firstOrderArtifact,
      baseFirstOrderActions, field, value, equalFieldValue, equalFields, all,
      Umpire3.Temporal.Targets.NexusCancellationFencing.any, setValue, copyValue,
      FirstOrderArtifact.next, FirstOrderAction.apply, FirstOrderFormula.eval,
      FirstOrderTerm.eval, FirstOrderState.read, FirstOrderState.write,
      applyFirstOrderUpdates, requestCancellationState]

private theorem firstOrderAcquireOwnership_iff (state nextState : TrustedState) :
    soundFirstOrderArtifact.next (encodeTrustedState state) "acquire-ownership" =
        some (encodeTrustedState nextState) ↔
      ((state.lifecycle = .Open ∨ state.lifecycle = .CancellationAccepted) ∨
          state.lifecycle = .Cancelled) ∧ state.ownerEpoch = .Epoch0 ∧
        acquireOwnershipState state = nextState := by
  rcases state with ⟨lifecycle, task, ownerEpoch, workerEpoch, completionEpoch⟩
  rcases nextState with ⟨nextLifecycle, nextTask, nextOwner, nextWorker, nextCompletion⟩
  cases lifecycle <;> cases ownerEpoch <;>
    simp [encodeTrustedState, soundFirstOrderArtifact, firstOrderArtifact,
      baseFirstOrderActions, field, value, equalFieldValue, equalFields, all,
      Umpire3.Temporal.Targets.NexusCancellationFencing.any, setValue, copyValue,
      FirstOrderArtifact.next, FirstOrderAction.apply, FirstOrderFormula.eval,
      FirstOrderTerm.eval, FirstOrderState.read, FirstOrderState.write,
      applyFirstOrderUpdates, acquireOwnershipState]

private theorem firstOrderCommitCancellation_iff (state nextState : TrustedState) :
    soundFirstOrderArtifact.next (encodeTrustedState state) "commit-cancellation" =
        some (encodeTrustedState nextState) ↔
      state.lifecycle = .CancellationAccepted ∧ commitCancellationState state = nextState := by
  rcases state with ⟨lifecycle, task, ownerEpoch, workerEpoch, completionEpoch⟩
  rcases nextState with ⟨nextLifecycle, nextTask, nextOwner, nextWorker, nextCompletion⟩
  cases lifecycle <;>
    simp [encodeTrustedState, soundFirstOrderArtifact, firstOrderArtifact,
      baseFirstOrderActions, field, value, equalFieldValue, equalFields, all,
      Umpire3.Temporal.Targets.NexusCancellationFencing.any, setValue, copyValue,
      FirstOrderArtifact.next, FirstOrderAction.apply, FirstOrderFormula.eval,
      FirstOrderTerm.eval, FirstOrderState.read, FirstOrderState.write,
      applyFirstOrderUpdates, commitCancellationState]

private theorem firstOrderWorkerReturnsSuccess_iff (state nextState : TrustedState) :
    soundFirstOrderArtifact.next (encodeTrustedState state) "worker-returns-success" =
        some (encodeTrustedState nextState) ↔
      state.task = .Dispatched ∧ workerReturnsSuccessState state = nextState := by
  rcases state with ⟨lifecycle, task, ownerEpoch, workerEpoch, completionEpoch⟩
  rcases nextState with ⟨nextLifecycle, nextTask, nextOwner, nextWorker, nextCompletion⟩
  cases task <;>
    simp [encodeTrustedState, soundFirstOrderArtifact, firstOrderArtifact,
      baseFirstOrderActions, field, value, equalFieldValue, equalFields, all,
      Umpire3.Temporal.Targets.NexusCancellationFencing.any, setValue, copyValue,
      FirstOrderArtifact.next, FirstOrderAction.apply, FirstOrderFormula.eval,
      FirstOrderTerm.eval, FirstOrderState.read, FirstOrderState.write,
      applyFirstOrderUpdates, workerReturnsSuccessState]

private theorem firstOrderPersistSuccess_iff (state nextState : TrustedState) :
    soundFirstOrderArtifact.next (encodeTrustedState state) "persist-success" =
        some (encodeTrustedState nextState) ↔
      state.task = .Returned ∧ state.completionEpoch = state.ownerEpoch ∧
        (state.lifecycle = .Open ∨ state.lifecycle = .CancellationAccepted) ∧
          persistSuccessState state = nextState := by
  rcases state with ⟨lifecycle, task, ownerEpoch, workerEpoch, completionEpoch⟩
  rcases nextState with ⟨nextLifecycle, nextTask, nextOwner, nextWorker, nextCompletion⟩
  cases task <;> cases completionEpoch <;> cases ownerEpoch <;> cases lifecycle <;>
    simp [encodeTrustedState, soundFirstOrderArtifact, firstOrderArtifact,
      baseFirstOrderActions, field, value, equalFieldValue, equalFields, all,
      Umpire3.Temporal.Targets.NexusCancellationFencing.any, setValue, copyValue,
      FirstOrderArtifact.next, FirstOrderAction.apply, FirstOrderFormula.eval,
      FirstOrderTerm.eval, FirstOrderState.read, FirstOrderState.write,
      applyFirstOrderUpdates, persistSuccessState]

private theorem trustedNext_iff (state : TrustedState) (label : TrustedLabel)
    (nextState : TrustedState) :
    trustedNext state label nextState ↔
      soundFirstOrderArtifact.next (encodeTrustedState state) (encodeTrustedLabel label) =
        some (encodeTrustedState nextState) := by
  cases label
  · exact (trustedDispatch_iff state nextState).trans (firstOrderDispatch_iff state nextState).symm
  · exact (trustedRequestCancellation_iff state nextState).trans
      (firstOrderRequestCancellation_iff state nextState).symm
  · exact (trustedAcquireOwnership_iff state nextState).trans
      (firstOrderAcquireOwnership_iff state nextState).symm
  · exact (trustedCommitCancellation_iff state nextState).trans
      (firstOrderCommitCancellation_iff state nextState).symm
  · exact (trustedWorkerReturnsSuccess_iff state nextState).trans
      (firstOrderWorkerReturnsSuccess_iff state nextState).symm
  · exact (trustedPersistSuccess_iff state nextState).trans
      (firstOrderPersistSuccess_iff state nextState).symm

def trustedSemanticRelation : Umpire3.Veil.SemanticRelation soundFirstOrderArtifact where
  State := TrustedState
  Label := TrustedLabel
  initial := trustedInitial
  next := trustedNext
  property := trustedProperty
  encodeState := encodeTrustedState
  encodeLabel := encodeTrustedLabel
  initial_iff := trustedInitial_iff
  next_iff := trustedNext_iff
  property_iff := trustedProperty_iff
  state_injective := encodeTrustedState_injective
  label_injective := encodeTrustedLabel_injective
  label_total := by intro label; cases label <;> decide
  label_complete := by
    intro identifier member
    simp [soundFirstOrderArtifact, firstOrderArtifact, baseFirstOrderActions,
      FirstOrderArtifact.actionIdentifiers] at member
    rcases member with rfl | rfl | rfl | rfl | rfl | rfl
    · exact ⟨.DispatchTask, rfl⟩
    · exact ⟨.RequestCancellation, rfl⟩
    · exact ⟨.AcquireOwnership, rfl⟩
    · exact ⟨.CommitCancellation, rfl⟩
    · exact ⟨.WorkerReturnsSuccess, rfl⟩
    · exact ⟨.PersistSuccess, rfl⟩

theorem trustedSemanticRelation_dispatch_iff (state nextState : TrustedState) :
    trustedSemanticRelation.next state .DispatchTask nextState ↔
      state.task = .Idle ∧ state.lifecycle = .Open ∧
        { state with task := .Dispatched, workerEpoch := state.ownerEpoch } = nextState := by
  simpa [trustedSemanticRelation, dispatchState] using trustedDispatch_iff state nextState

theorem trustedSemanticRelation_requestCancellation_iff (state nextState : TrustedState) :
    trustedSemanticRelation.next state .RequestCancellation nextState ↔
      state.lifecycle = .Open ∧ { state with lifecycle := .CancellationAccepted } = nextState := by
  simpa [trustedSemanticRelation, requestCancellationState] using
    trustedRequestCancellation_iff state nextState

theorem trustedSemanticRelation_acquireOwnership_iff (state nextState : TrustedState) :
    trustedSemanticRelation.next state .AcquireOwnership nextState ↔
      ((state.lifecycle = .Open ∨ state.lifecycle = .CancellationAccepted) ∨
          state.lifecycle = .Cancelled) ∧ state.ownerEpoch = .Epoch0 ∧
        { state with ownerEpoch := .Epoch1 } = nextState := by
  simpa [trustedSemanticRelation, acquireOwnershipState] using
    trustedAcquireOwnership_iff state nextState

theorem trustedSemanticRelation_commitCancellation_iff (state nextState : TrustedState) :
    trustedSemanticRelation.next state .CommitCancellation nextState ↔
      state.lifecycle = .CancellationAccepted ∧
        { state with lifecycle := .Cancelled } = nextState := by
  simpa [trustedSemanticRelation, commitCancellationState] using
    trustedCommitCancellation_iff state nextState

theorem trustedSemanticRelation_workerReturnsSuccess_iff (state nextState : TrustedState) :
    trustedSemanticRelation.next state .WorkerReturnsSuccess nextState ↔
      state.task = .Dispatched ∧
        { state with task := .Returned, completionEpoch := state.workerEpoch } = nextState := by
  simpa [trustedSemanticRelation, workerReturnsSuccessState] using
    trustedWorkerReturnsSuccess_iff state nextState

theorem trustedSemanticRelation_persistSuccess_iff (state nextState : TrustedState) :
    trustedSemanticRelation.next state .PersistSuccess nextState ↔
      state.task = .Returned ∧ state.completionEpoch = state.ownerEpoch ∧
        (state.lifecycle = .Open ∨ state.lifecycle = .CancellationAccepted) ∧
          { state with lifecycle := .Succeeded } = nextState := by
  simpa [trustedSemanticRelation, persistSuccessState] using trustedPersistSuccess_iff state nextState

theorem trustedSemanticRelation_initialState_iff (state : TrustedState) :
    trustedSemanticRelation.initial state ↔
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
    simp [trustedSemanticRelation, trustedInitial, NexusCancellationSoundTrusted.Init,
      NexusCancellationSoundTrusted.initializer.ext.tr] <;>
    decide

theorem trustedSemanticRelation_propertyState_iff (state : TrustedState) :
    trustedSemanticRelation.property state ↔
      state.lifecycle ≠ .Succeeded ∨ state.completionEpoch = state.ownerEpoch := by
  unfold trustedSemanticRelation trustedProperty
  unfold NexusCancellationSoundTrusted.NexusCancellationWonExcludesSuccess
  rfl

def soundTrustedSemanticBinding :
    Umpire3.Veil.SemanticBinding soundFirstOrderArtifact where
  symbolic := trustedSemanticRelation
  concrete := soundConcreteSemanticRelation

end Umpire3.Temporal.Veil.NexusCancellationFencing
