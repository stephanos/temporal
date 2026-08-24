import Temporal.Families.NexusCancellation.Targets.FirstOrder
import Temporal.Families.NexusCancellation.Targets.Veil.Sound
import Umpire3.Veil.Semantics

namespace Umpire3.Temporal.Veil.NexusCancellationFencing

open Umpire3.Temporal.Targets.NexusCancellationFencing

set_option maxHeartbeats 400000

abbrev SoundLifecycle := NexusCancellationSound.Lifecycle_IndT
abbrev SoundTaskStage := NexusCancellationSound.TaskStage_IndT
abbrev SoundEpoch := NexusCancellationSound.Epoch_IndT
abbrev SoundField :=
  NexusCancellationSound.FieldConcreteType SoundLifecycle SoundTaskStage SoundEpoch
abbrev SoundState := NexusCancellationSound.State SoundField
abbrev SoundTheory := NexusCancellationSound.Theory SoundLifecycle SoundTaskStage SoundEpoch
abbrev SoundLabel := NexusCancellationSound.Label SoundLifecycle SoundTaskStage SoundEpoch

private def soundStateTuple (state : SoundState) :
    SoundLifecycle × SoundTaskStage × SoundEpoch × SoundEpoch × SoundEpoch :=
  (state.lifecycle, state.task, state.ownerEpoch, state.workerEpoch, state.completionEpoch)

private theorem soundStateTuple_injective : Function.Injective soundStateTuple := by
  intro left right equal
  rcases left with ⟨leftLifecycle, leftTask, leftOwner, leftWorker, leftCompletion⟩
  rcases right with ⟨rightLifecycle, rightTask, rightOwner, rightWorker, rightCompletion⟩
  simp only [soundStateTuple, Prod.mk.injEq] at equal
  rcases equal with ⟨rfl, rfl, rfl, rfl, rfl⟩
  rfl

local instance : DecidableEq SoundState :=
  soundStateTuple_injective.decidableEq

local instance : NexusCancellationSound.Lifecycle_EnumClass SoundLifecycle :=
  NexusCancellationSound.instLifecycle_EnumClassLifecycle_IndT
local instance : NexusCancellationSound.TaskStage_EnumClass SoundTaskStage :=
  NexusCancellationSound.instTaskStage_EnumClassTaskStage_IndT
local instance : NexusCancellationSound.Epoch_EnumClass SoundEpoch :=
  NexusCancellationSound.instEpoch_EnumClassEpoch_IndT
local instance soundFieldRepresentation : (field : NexusCancellationSound.State.Label) →
    Veil.FieldRepresentation
      (NexusCancellationSound.State.Label.toDomain SoundLifecycle SoundTaskStage SoundEpoch field)
      (NexusCancellationSound.State.Label.toCodomain SoundLifecycle SoundTaskStage SoundEpoch field)
      (SoundField field) :=
  NexusCancellationSound.instFieldRepresentation SoundLifecycle SoundTaskStage SoundEpoch
local instance soundFieldLawful : (field : NexusCancellationSound.State.Label) →
    Veil.LawfulFieldRepresentation
      (NexusCancellationSound.State.Label.toDomain SoundLifecycle SoundTaskStage SoundEpoch field)
      (NexusCancellationSound.State.Label.toCodomain SoundLifecycle SoundTaskStage SoundEpoch field)
      (SoundField field)
      (soundFieldRepresentation field) :=
  NexusCancellationSound.instLawfulFieldRepresentation SoundLifecycle SoundTaskStage SoundEpoch
local instance : Inhabited SoundState :=
  NexusCancellationSound.instInhabitedStateFieldConcreteType

private def soundTheory : SoundTheory := {}

private def soundInitial (state : SoundState) : Prop :=
  NexusCancellationSound.Init SoundTheory SoundState SoundLifecycle SoundTaskStage SoundEpoch
    SoundField soundTheory state

private def soundNext (state : SoundState) (label : SoundLabel) (nextState : SoundState) : Prop :=
  NexusCancellationSound.Next SoundTheory SoundState SoundLifecycle SoundTaskStage SoundEpoch
    SoundField soundTheory state label nextState

private def soundProperty (state : SoundState) : Prop :=
  NexusCancellationSound.NexusCancellationWonExcludesSuccess
    (th := soundTheory) (st := state)

private def encodeLifecycle : SoundLifecycle → String
  | .Open => "open"
  | .CancellationAccepted => "cancellation-accepted"
  | .Cancelled => "cancelled"
  | .Succeeded => "succeeded"

private def encodeTask : SoundTaskStage → String
  | .Idle => "idle"
  | .Dispatched => "dispatched"
  | .Returned => "returned"

private def encodeEpoch : SoundEpoch → String
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

@[simp] private theorem encodeLifecycle_eq_open (lifecycle : SoundLifecycle) :
    encodeLifecycle lifecycle = "open" ↔ lifecycle = .Open := by
  cases lifecycle <;> decide

@[simp] private theorem encodeTask_eq_idle (task : SoundTaskStage) :
    encodeTask task = "idle" ↔ task = .Idle := by
  cases task <;> decide

@[simp] private theorem encodeTask_eq_dispatched (task : SoundTaskStage) :
    encodeTask task = "dispatched" ↔ task = .Dispatched := by
  cases task <;> decide

@[simp] private theorem encodeLifecycle_eq_cancellationAccepted (lifecycle : SoundLifecycle) :
    encodeLifecycle lifecycle = "cancellation-accepted" ↔
      lifecycle = .CancellationAccepted := by
  cases lifecycle <;> decide

@[simp] private theorem encodeLifecycle_eq_cancelled (lifecycle : SoundLifecycle) :
    encodeLifecycle lifecycle = "cancelled" ↔ lifecycle = .Cancelled := by
  cases lifecycle <;> decide

@[simp] private theorem encodeTask_eq_returned (task : SoundTaskStage) :
    encodeTask task = "returned" ↔ task = .Returned := by
  cases task <;> decide

@[simp] private theorem encodeEpoch_eq_epoch0 (epoch : SoundEpoch) :
    encodeEpoch epoch = "epoch-0" ↔ epoch = .Epoch0 := by
  cases epoch <;> decide

@[simp] private theorem open_eq_encodeLifecycle (lifecycle : SoundLifecycle) :
    "open" = encodeLifecycle lifecycle ↔ .Open = lifecycle := by
  cases lifecycle <;> decide

@[simp] private theorem dispatched_eq_encodeTask (task : SoundTaskStage) :
    "dispatched" = encodeTask task ↔ .Dispatched = task := by
  cases task <;> decide

@[simp] private theorem cancellationAccepted_eq_encodeLifecycle (lifecycle : SoundLifecycle) :
    "cancellation-accepted" = encodeLifecycle lifecycle ↔
      .CancellationAccepted = lifecycle := by
  cases lifecycle <;> decide

@[simp] private theorem cancelled_eq_encodeLifecycle (lifecycle : SoundLifecycle) :
    "cancelled" = encodeLifecycle lifecycle ↔ .Cancelled = lifecycle := by
  cases lifecycle <;> decide

@[simp] private theorem succeeded_eq_encodeLifecycle (lifecycle : SoundLifecycle) :
    "succeeded" = encodeLifecycle lifecycle ↔ .Succeeded = lifecycle := by
  cases lifecycle <;> decide

@[simp] private theorem returned_eq_encodeTask (task : SoundTaskStage) :
    "returned" = encodeTask task ↔ .Returned = task := by
  cases task <;> decide

@[simp] private theorem epoch1_eq_encodeEpoch (epoch : SoundEpoch) :
    "epoch-1" = encodeEpoch epoch ↔ .Epoch1 = epoch := by
  cases epoch <;> decide

@[simp] private theorem encodeLifecycle_eq_encode (left right : SoundLifecycle) :
    encodeLifecycle left = encodeLifecycle right ↔ left = right :=
  encodeLifecycle_injective.eq_iff

@[simp] private theorem encodeTask_eq_encode (left right : SoundTaskStage) :
    encodeTask left = encodeTask right ↔ left = right :=
  encodeTask_injective.eq_iff

@[simp] private theorem encodeEpoch_eq_encode (left right : SoundEpoch) :
    encodeEpoch left = encodeEpoch right ↔ left = right :=
  encodeEpoch_injective.eq_iff

private def encodeSoundState (state : SoundState) : FirstOrderState where
  fields := [
    { field := "lifecycle", value := encodeLifecycle state.lifecycle },
    { field := "task", value := encodeTask state.task },
    { field := "owner-epoch", value := encodeEpoch state.ownerEpoch },
    { field := "worker-epoch", value := encodeEpoch state.workerEpoch },
    { field := "completion-epoch", value := encodeEpoch state.completionEpoch },
  ]

private theorem encodeSoundState_injective : Function.Injective encodeSoundState := by
  intro left right equal
  rcases left with ⟨leftLifecycle, leftTask, leftOwner, leftWorker, leftCompletion⟩
  rcases right with ⟨rightLifecycle, rightTask, rightOwner, rightWorker, rightCompletion⟩
  simp only [encodeSoundState, FirstOrderState.mk.injEq, List.cons.injEq,
    FirstOrderBinding.mk.injEq, true_and] at equal
  rcases equal with ⟨lifecycleEqual, taskEqual, ownerEqual, workerEqual, completionEqual⟩
  have := encodeLifecycle_injective lifecycleEqual
  have := encodeTask_injective taskEqual
  have := encodeEpoch_injective ownerEqual
  have := encodeEpoch_injective workerEqual
  have := encodeEpoch_injective completionEqual.1
  subst_vars
  rfl

private def encodeSoundLabel : SoundLabel → String
  | .DispatchTask => "dispatch-task"
  | .RequestCancellation => "request-cancellation"
  | .AcquireOwnership => "acquire-ownership"
  | .CommitCancellation => "commit-cancellation"
  | .WorkerReturnsSuccess => "worker-returns-success"
  | .PersistSuccess => "persist-success"

private theorem encodeSoundLabel_injective : Function.Injective encodeSoundLabel := by
  intro left right equal
  cases left <;> cases right <;> simp_all [encodeSoundLabel]

private theorem soundInitial_iff (state : SoundState) :
    soundInitial state ↔ soundFirstOrderArtifact.initial.eval (encodeSoundState state) = true := by
  rcases state with ⟨lifecycle, task, ownerEpoch, workerEpoch, completionEpoch⟩
  cases lifecycle <;> cases task <;> cases ownerEpoch <;> cases workerEpoch <;>
    cases completionEpoch <;>
    simp [soundInitial, NexusCancellationSound.Init,
      NexusCancellationSound.initializer.ext.tr] <;>
    decide

private theorem soundProperty_iff (state : SoundState) :
    soundProperty state ↔
      soundFirstOrderArtifact.invariant.eval (encodeSoundState state) = true := by
  rcases state with ⟨lifecycle, task, ownerEpoch, workerEpoch, completionEpoch⟩
  cases lifecycle <;> cases task <;> cases ownerEpoch <;> cases workerEpoch <;>
    cases completionEpoch <;>
    simp [soundProperty, NexusCancellationSound.NexusCancellationWonExcludesSuccess] <;>
    decide

private def dispatchState (state : SoundState) : SoundState :=
  { state with task := .Dispatched, workerEpoch := state.ownerEpoch }

private def requestCancellationState (state : SoundState) : SoundState :=
  { state with lifecycle := .CancellationAccepted }

private def acquireOwnershipState (state : SoundState) : SoundState :=
  { state with ownerEpoch := .Epoch1 }

private def commitCancellationState (state : SoundState) : SoundState :=
  { state with lifecycle := .Cancelled }

private def workerReturnsSuccessState (state : SoundState) : SoundState :=
  { state with task := .Returned, completionEpoch := state.workerEpoch }

private def persistSuccessState (state : SoundState) : SoundState :=
  { state with lifecycle := .Succeeded }

private theorem soundDispatch_iff (state nextState : SoundState) :
    soundNext state .DispatchTask nextState ↔
      state.task = .Idle ∧ state.lifecycle = .Open ∧ dispatchState state = nextState := by
  unfold soundNext NexusCancellationSound.Next NexusCancellationSound.NextAct
  rw [NexusCancellationSound.DispatchTask.ext.derived_eq]
  rfl

private theorem soundRequestCancellation_iff (state nextState : SoundState) :
    soundNext state .RequestCancellation nextState ↔
      state.lifecycle = .Open ∧ requestCancellationState state = nextState := by
  unfold soundNext NexusCancellationSound.Next NexusCancellationSound.NextAct
  rw [NexusCancellationSound.RequestCancellation.ext.derived_eq]
  rfl

private theorem soundAcquireOwnership_iff (state nextState : SoundState) :
    soundNext state .AcquireOwnership nextState ↔
      ((state.lifecycle = .Open ∨ state.lifecycle = .CancellationAccepted) ∨
          state.lifecycle = .Cancelled) ∧ state.ownerEpoch = .Epoch0 ∧
        acquireOwnershipState state = nextState := by
  unfold soundNext NexusCancellationSound.Next NexusCancellationSound.NextAct
  rw [NexusCancellationSound.AcquireOwnership.ext.derived_eq]
  rfl

private theorem soundCommitCancellation_iff (state nextState : SoundState) :
    soundNext state .CommitCancellation nextState ↔
      state.lifecycle = .CancellationAccepted ∧ commitCancellationState state = nextState := by
  unfold soundNext NexusCancellationSound.Next NexusCancellationSound.NextAct
  rw [NexusCancellationSound.CommitCancellation.ext.derived_eq]
  rfl

private theorem soundWorkerReturnsSuccess_iff (state nextState : SoundState) :
    soundNext state .WorkerReturnsSuccess nextState ↔
      state.task = .Dispatched ∧ workerReturnsSuccessState state = nextState := by
  unfold soundNext NexusCancellationSound.Next NexusCancellationSound.NextAct
  rw [NexusCancellationSound.WorkerReturnsSuccess.ext.derived_eq]
  rfl

private theorem soundPersistSuccess_iff (state nextState : SoundState) :
    soundNext state .PersistSuccess nextState ↔
      state.task = .Returned ∧ state.completionEpoch = state.ownerEpoch ∧
        (state.lifecycle = .Open ∨ state.lifecycle = .CancellationAccepted) ∧
          persistSuccessState state = nextState := by
  unfold soundNext NexusCancellationSound.Next NexusCancellationSound.NextAct
  rw [NexusCancellationSound.PersistSuccess.ext.derived_eq]
  simp only [NexusCancellationSound.PersistSuccess.ext.tr, persistSuccessState]
  rfl

private theorem firstOrderDispatch_iff (state nextState : SoundState) :
    soundFirstOrderArtifact.next (encodeSoundState state) "dispatch-task" =
        some (encodeSoundState nextState) ↔
      state.task = .Idle ∧ state.lifecycle = .Open ∧ dispatchState state = nextState := by
  rcases state with ⟨lifecycle, task, ownerEpoch, workerEpoch, completionEpoch⟩
  rcases nextState with ⟨nextLifecycle, nextTask, nextOwner, nextWorker, nextCompletion⟩
  cases task <;> cases lifecycle <;>
    simp [encodeSoundState, soundFirstOrderArtifact, firstOrderArtifact,
      baseFirstOrderActions, field, value, equalFieldValue, equalFields, all,
      Umpire3.Temporal.Targets.NexusCancellationFencing.any, setValue, copyValue,
      FirstOrderArtifact.next, FirstOrderAction.apply, FirstOrderFormula.eval,
      FirstOrderTerm.eval, FirstOrderState.read, FirstOrderState.write,
      applyFirstOrderUpdates, dispatchState]

private theorem firstOrderRequestCancellation_iff (state nextState : SoundState) :
    soundFirstOrderArtifact.next (encodeSoundState state) "request-cancellation" =
        some (encodeSoundState nextState) ↔
      state.lifecycle = .Open ∧ requestCancellationState state = nextState := by
  rcases state with ⟨lifecycle, task, ownerEpoch, workerEpoch, completionEpoch⟩
  rcases nextState with ⟨nextLifecycle, nextTask, nextOwner, nextWorker, nextCompletion⟩
  cases lifecycle <;>
    simp [encodeSoundState, soundFirstOrderArtifact, firstOrderArtifact,
      baseFirstOrderActions, field, value, equalFieldValue, equalFields, all,
      Umpire3.Temporal.Targets.NexusCancellationFencing.any, setValue, copyValue,
      FirstOrderArtifact.next, FirstOrderAction.apply, FirstOrderFormula.eval,
      FirstOrderTerm.eval, FirstOrderState.read, FirstOrderState.write,
      applyFirstOrderUpdates, requestCancellationState]

private theorem firstOrderAcquireOwnership_iff (state nextState : SoundState) :
    soundFirstOrderArtifact.next (encodeSoundState state) "acquire-ownership" =
        some (encodeSoundState nextState) ↔
      ((state.lifecycle = .Open ∨ state.lifecycle = .CancellationAccepted) ∨
          state.lifecycle = .Cancelled) ∧ state.ownerEpoch = .Epoch0 ∧
        acquireOwnershipState state = nextState := by
  rcases state with ⟨lifecycle, task, ownerEpoch, workerEpoch, completionEpoch⟩
  rcases nextState with ⟨nextLifecycle, nextTask, nextOwner, nextWorker, nextCompletion⟩
  cases lifecycle <;> cases ownerEpoch <;>
    simp [encodeSoundState, soundFirstOrderArtifact, firstOrderArtifact,
      baseFirstOrderActions, field, value, equalFieldValue, equalFields, all,
      Umpire3.Temporal.Targets.NexusCancellationFencing.any, setValue, copyValue,
      FirstOrderArtifact.next, FirstOrderAction.apply, FirstOrderFormula.eval,
      FirstOrderTerm.eval, FirstOrderState.read, FirstOrderState.write,
      applyFirstOrderUpdates, acquireOwnershipState]

private theorem firstOrderCommitCancellation_iff (state nextState : SoundState) :
    soundFirstOrderArtifact.next (encodeSoundState state) "commit-cancellation" =
        some (encodeSoundState nextState) ↔
      state.lifecycle = .CancellationAccepted ∧ commitCancellationState state = nextState := by
  rcases state with ⟨lifecycle, task, ownerEpoch, workerEpoch, completionEpoch⟩
  rcases nextState with ⟨nextLifecycle, nextTask, nextOwner, nextWorker, nextCompletion⟩
  cases lifecycle <;>
    simp [encodeSoundState, soundFirstOrderArtifact, firstOrderArtifact,
      baseFirstOrderActions, field, value, equalFieldValue, equalFields, all,
      Umpire3.Temporal.Targets.NexusCancellationFencing.any, setValue, copyValue,
      FirstOrderArtifact.next, FirstOrderAction.apply, FirstOrderFormula.eval,
      FirstOrderTerm.eval, FirstOrderState.read, FirstOrderState.write,
      applyFirstOrderUpdates, commitCancellationState]

private theorem firstOrderWorkerReturnsSuccess_iff (state nextState : SoundState) :
    soundFirstOrderArtifact.next (encodeSoundState state) "worker-returns-success" =
        some (encodeSoundState nextState) ↔
      state.task = .Dispatched ∧ workerReturnsSuccessState state = nextState := by
  rcases state with ⟨lifecycle, task, ownerEpoch, workerEpoch, completionEpoch⟩
  rcases nextState with ⟨nextLifecycle, nextTask, nextOwner, nextWorker, nextCompletion⟩
  cases task <;>
    simp [encodeSoundState, soundFirstOrderArtifact, firstOrderArtifact,
      baseFirstOrderActions, field, value, equalFieldValue, equalFields, all,
      Umpire3.Temporal.Targets.NexusCancellationFencing.any, setValue, copyValue,
      FirstOrderArtifact.next, FirstOrderAction.apply, FirstOrderFormula.eval,
      FirstOrderTerm.eval, FirstOrderState.read, FirstOrderState.write,
      applyFirstOrderUpdates, workerReturnsSuccessState]

private theorem firstOrderPersistSuccess_iff (state nextState : SoundState) :
    soundFirstOrderArtifact.next (encodeSoundState state) "persist-success" =
        some (encodeSoundState nextState) ↔
      state.task = .Returned ∧ state.completionEpoch = state.ownerEpoch ∧
        (state.lifecycle = .Open ∨ state.lifecycle = .CancellationAccepted) ∧
          persistSuccessState state = nextState := by
  rcases state with ⟨lifecycle, task, ownerEpoch, workerEpoch, completionEpoch⟩
  rcases nextState with ⟨nextLifecycle, nextTask, nextOwner, nextWorker, nextCompletion⟩
  cases task <;> cases completionEpoch <;> cases ownerEpoch <;> cases lifecycle <;>
    simp [encodeSoundState, soundFirstOrderArtifact, firstOrderArtifact,
      baseFirstOrderActions, field, value, equalFieldValue, equalFields, all,
      Umpire3.Temporal.Targets.NexusCancellationFencing.any, setValue, copyValue,
      FirstOrderArtifact.next, FirstOrderAction.apply, FirstOrderFormula.eval,
      FirstOrderTerm.eval, FirstOrderState.read, FirstOrderState.write,
      applyFirstOrderUpdates, persistSuccessState]

private theorem soundNext_iff (state : SoundState) (label : SoundLabel)
    (nextState : SoundState) :
    soundNext state label nextState ↔
      soundFirstOrderArtifact.next (encodeSoundState state) (encodeSoundLabel label) =
        some (encodeSoundState nextState) := by
  cases label
  · exact (soundDispatch_iff state nextState).trans (firstOrderDispatch_iff state nextState).symm
  · exact (soundRequestCancellation_iff state nextState).trans
      (firstOrderRequestCancellation_iff state nextState).symm
  · exact (soundAcquireOwnership_iff state nextState).trans
      (firstOrderAcquireOwnership_iff state nextState).symm
  · exact (soundCommitCancellation_iff state nextState).trans
      (firstOrderCommitCancellation_iff state nextState).symm
  · exact (soundWorkerReturnsSuccess_iff state nextState).trans
      (firstOrderWorkerReturnsSuccess_iff state nextState).symm
  · exact (soundPersistSuccess_iff state nextState).trans
      (firstOrderPersistSuccess_iff state nextState).symm

def soundSemanticRelation : Umpire3.Veil.SemanticRelation soundFirstOrderArtifact where
  State := SoundState
  Label := SoundLabel
  initial := soundInitial
  next := soundNext
  property := soundProperty
  encodeState := encodeSoundState
  encodeLabel := encodeSoundLabel
  initial_iff := soundInitial_iff
  next_iff := soundNext_iff
  property_iff := soundProperty_iff
  state_injective := encodeSoundState_injective
  label_injective := encodeSoundLabel_injective
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

theorem soundSemanticRelation_dispatch_iff (state nextState : SoundState) :
    soundSemanticRelation.next state .DispatchTask nextState ↔
      state.task = .Idle ∧ state.lifecycle = .Open ∧
        { state with task := .Dispatched, workerEpoch := state.ownerEpoch } = nextState := by
  simpa [soundSemanticRelation, dispatchState] using soundDispatch_iff state nextState

theorem soundSemanticRelation_requestCancellation_iff (state nextState : SoundState) :
    soundSemanticRelation.next state .RequestCancellation nextState ↔
      state.lifecycle = .Open ∧ { state with lifecycle := .CancellationAccepted } = nextState := by
  simpa [soundSemanticRelation, requestCancellationState] using
    soundRequestCancellation_iff state nextState

theorem soundSemanticRelation_acquireOwnership_iff (state nextState : SoundState) :
    soundSemanticRelation.next state .AcquireOwnership nextState ↔
      ((state.lifecycle = .Open ∨ state.lifecycle = .CancellationAccepted) ∨
          state.lifecycle = .Cancelled) ∧ state.ownerEpoch = .Epoch0 ∧
        { state with ownerEpoch := .Epoch1 } = nextState := by
  simpa [soundSemanticRelation, acquireOwnershipState] using
    soundAcquireOwnership_iff state nextState

theorem soundSemanticRelation_commitCancellation_iff (state nextState : SoundState) :
    soundSemanticRelation.next state .CommitCancellation nextState ↔
      state.lifecycle = .CancellationAccepted ∧
        { state with lifecycle := .Cancelled } = nextState := by
  simpa [soundSemanticRelation, commitCancellationState] using
    soundCommitCancellation_iff state nextState

theorem soundSemanticRelation_workerReturnsSuccess_iff (state nextState : SoundState) :
    soundSemanticRelation.next state .WorkerReturnsSuccess nextState ↔
      state.task = .Dispatched ∧
        { state with task := .Returned, completionEpoch := state.workerEpoch } = nextState := by
  simpa [soundSemanticRelation, workerReturnsSuccessState] using
    soundWorkerReturnsSuccess_iff state nextState

theorem soundSemanticRelation_persistSuccess_iff (state nextState : SoundState) :
    soundSemanticRelation.next state .PersistSuccess nextState ↔
      state.task = .Returned ∧ state.completionEpoch = state.ownerEpoch ∧
        (state.lifecycle = .Open ∨ state.lifecycle = .CancellationAccepted) ∧
          { state with lifecycle := .Succeeded } = nextState := by
  simpa [soundSemanticRelation, persistSuccessState] using soundPersistSuccess_iff state nextState

theorem soundSemanticRelation_initialState_iff (state : SoundState) :
    soundSemanticRelation.initial state ↔
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
    simp [soundSemanticRelation, soundInitial, NexusCancellationSound.Init,
      NexusCancellationSound.initializer.ext.tr] <;>
    decide

theorem soundSemanticRelation_propertyState_iff (state : SoundState) :
    soundSemanticRelation.property state ↔
      state.lifecycle ≠ .Succeeded ∨ state.completionEpoch = state.ownerEpoch := by
  unfold soundSemanticRelation soundProperty
  unfold NexusCancellationSound.NexusCancellationWonExcludesSuccess
  rfl

end Umpire3.Temporal.Veil.NexusCancellationFencing
