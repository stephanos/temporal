import Temporal.Families.NexusCancellation.Targets.Veil.SoundConcrete
import Temporal.Families.NexusCancellation.Targets.Veil.SoundSemantics

namespace Umpire3.Temporal.Veil.NexusCancellationFencing

open Umpire3.Temporal.Targets.NexusCancellationFencing

set_option maxHeartbeats 400000
abbrev ConcreteLifecycle := NexusCancellationSoundConcrete.Lifecycle_IndT
abbrev ConcreteTaskStage := NexusCancellationSoundConcrete.TaskStage_IndT
abbrev ConcreteEpoch := NexusCancellationSoundConcrete.Epoch_IndT
abbrev ConcreteField :=
  NexusCancellationSoundConcrete.FieldConcreteType ConcreteLifecycle ConcreteTaskStage ConcreteEpoch
abbrev ConcreteState := NexusCancellationSoundConcrete.State ConcreteField
abbrev ConcreteTheory :=
  NexusCancellationSoundConcrete.Theory ConcreteLifecycle ConcreteTaskStage ConcreteEpoch
abbrev ConcreteLabel :=
  NexusCancellationSoundConcrete.Label ConcreteLifecycle ConcreteTaskStage ConcreteEpoch

local instance : NexusCancellationSoundConcrete.Lifecycle_EnumClass ConcreteLifecycle :=
  NexusCancellationSoundConcrete.instLifecycle_EnumClassLifecycle_IndT
local instance : NexusCancellationSoundConcrete.TaskStage_EnumClass ConcreteTaskStage :=
  NexusCancellationSoundConcrete.instTaskStage_EnumClassTaskStage_IndT
local instance : NexusCancellationSoundConcrete.Epoch_EnumClass ConcreteEpoch :=
  NexusCancellationSoundConcrete.instEpoch_EnumClassEpoch_IndT
local instance concreteFieldRepresentation :
    (field : NexusCancellationSoundConcrete.State.Label) →
      Veil.FieldRepresentation
        (NexusCancellationSoundConcrete.State.Label.toDomain
          ConcreteLifecycle ConcreteTaskStage ConcreteEpoch field)
        (NexusCancellationSoundConcrete.State.Label.toCodomain
          ConcreteLifecycle ConcreteTaskStage ConcreteEpoch field)
        (ConcreteField field) :=
  NexusCancellationSoundConcrete.instFieldRepresentation
    ConcreteLifecycle ConcreteTaskStage ConcreteEpoch
local instance concreteFieldLawful :
    (field : NexusCancellationSoundConcrete.State.Label) →
      Veil.LawfulFieldRepresentation
        (NexusCancellationSoundConcrete.State.Label.toDomain
          ConcreteLifecycle ConcreteTaskStage ConcreteEpoch field)
        (NexusCancellationSoundConcrete.State.Label.toCodomain
          ConcreteLifecycle ConcreteTaskStage ConcreteEpoch field)
        (ConcreteField field)
        (concreteFieldRepresentation field) :=
  NexusCancellationSoundConcrete.instLawfulFieldRepresentation
    ConcreteLifecycle ConcreteTaskStage ConcreteEpoch
local instance : Inhabited ConcreteState :=
  NexusCancellationSoundConcrete.instInhabitedStateFieldConcreteType

def concreteTheory : ConcreteTheory := {}

def concreteSystem :=
  NexusCancellationSoundConcrete.enumerableTransitionSystem
    ConcreteLifecycle ConcreteTaskStage ConcreteEpoch concreteTheory

def concreteInitialState : ConcreteState := {
  lifecycle := .Open
  task := .Idle
  ownerEpoch := .Epoch0
  workerEpoch := .None
  completionEpoch := .None
}

private theorem concreteInitialStates_eq : concreteSystem.initStates = [concreteInitialState] := by
  rfl

def concreteNext (state : ConcreteState) (label : ConcreteLabel)
    (nextState : ConcreteState) : Prop :=
  (label, Veil.ExecutionOutcome.success nextState) ∈
    concreteSystem.tr concreteTheory state

def concreteInitial (state : ConcreteState) : Prop :=
  state ∈ concreteSystem.initStates

def concreteProperty (state : ConcreteState) : Prop :=
  NexusCancellationSoundConcrete.NexusCancellationWonExcludesSuccess
    (th := concreteTheory) (st := state)

def lifecycleToSound : ConcreteLifecycle → SoundLifecycle
  | .Open => .Open
  | .CancellationAccepted => .CancellationAccepted
  | .Cancelled => .Cancelled
  | .Succeeded => .Succeeded

def taskToSound : ConcreteTaskStage → SoundTaskStage
  | .Idle => .Idle
  | .Dispatched => .Dispatched
  | .Returned => .Returned

def epochToSound : ConcreteEpoch → SoundEpoch
  | .None => .None
  | .Epoch0 => .Epoch0
  | .Epoch1 => .Epoch1

def stateToSound (state : ConcreteState) : SoundState := {
  lifecycle := lifecycleToSound state.lifecycle
  task := taskToSound state.task
  ownerEpoch := epochToSound state.ownerEpoch
  workerEpoch := epochToSound state.workerEpoch
  completionEpoch := epochToSound state.completionEpoch
}

@[simp] private theorem stateToSound_lifecycle (state : ConcreteState) :
    (stateToSound state).lifecycle = lifecycleToSound state.lifecycle := rfl

@[simp] private theorem stateToSound_task (state : ConcreteState) :
    (stateToSound state).task = taskToSound state.task := rfl

@[simp] private theorem stateToSound_ownerEpoch (state : ConcreteState) :
    (stateToSound state).ownerEpoch = epochToSound state.ownerEpoch := rfl

@[simp] private theorem stateToSound_workerEpoch (state : ConcreteState) :
    (stateToSound state).workerEpoch = epochToSound state.workerEpoch := rfl

@[simp] private theorem stateToSound_completionEpoch (state : ConcreteState) :
    (stateToSound state).completionEpoch = epochToSound state.completionEpoch := rfl

private theorem lifecycleToSound_injective : Function.Injective lifecycleToSound := by
  intro left right equal
  cases left <;> cases right <;> simp_all [lifecycleToSound]

private theorem taskToSound_injective : Function.Injective taskToSound := by
  intro left right equal
  cases left <;> cases right <;> simp_all [taskToSound]

private theorem epochToSound_injective : Function.Injective epochToSound := by
  intro left right equal
  cases left <;> cases right <;> simp_all [epochToSound]

private theorem stateToSound_injective : Function.Injective stateToSound := by
  intro left right equal
  rcases left with ⟨leftLifecycle, leftTask, leftOwner, leftWorker, leftCompletion⟩
  rcases right with ⟨rightLifecycle, rightTask, rightOwner, rightWorker, rightCompletion⟩
  simp only [stateToSound, NexusCancellationSound.State.mk.injEq] at equal
  rcases equal with ⟨lifecycleEqual, taskEqual, ownerEqual, workerEqual, completionEqual⟩
  have := lifecycleToSound_injective lifecycleEqual
  have := taskToSound_injective taskEqual
  have := epochToSound_injective ownerEqual
  have := epochToSound_injective workerEqual
  have := epochToSound_injective completionEqual
  subst_vars
  rfl

def labelToSound : ConcreteLabel → SoundLabel
  | .DispatchTask => .DispatchTask
  | .RequestCancellation => .RequestCancellation
  | .AcquireOwnership => .AcquireOwnership
  | .CommitCancellation => .CommitCancellation
  | .WorkerReturnsSuccess => .WorkerReturnsSuccess
  | .PersistSuccess => .PersistSuccess

private theorem labelToSound_injective : Function.Injective labelToSound := by
  intro left right equal
  cases left <;> cases right <;> simp_all [labelToSound]

def encodeConcreteState (state : ConcreteState) : Umpire3.FirstOrderState :=
  soundSemanticRelation.encodeState (stateToSound state)

def encodeConcreteLabel (label : ConcreteLabel) : String :=
  soundSemanticRelation.encodeLabel (labelToSound label)

def concreteDispatchState (state : ConcreteState) : ConcreteState :=
  { state with task := .Dispatched, workerEpoch := state.ownerEpoch }

def concreteRequestCancellationState (state : ConcreteState) : ConcreteState :=
  { state with lifecycle := .CancellationAccepted }

def concreteAcquireOwnershipState (state : ConcreteState) : ConcreteState :=
  { state with ownerEpoch := .Epoch1 }

def concreteCommitCancellationState (state : ConcreteState) : ConcreteState :=
  { state with lifecycle := .Cancelled }

def concreteWorkerReturnsSuccessState (state : ConcreteState) : ConcreteState :=
  { state with task := .Returned, completionEpoch := state.workerEpoch }

def concretePersistSuccessState (state : ConcreteState) : ConcreteState :=
  { state with lifecycle := .Succeeded }

@[simp] private theorem stateToSound_dispatchState (state : ConcreteState) :
    { stateToSound state with task := .Dispatched, workerEpoch := (stateToSound state).ownerEpoch } =
      stateToSound (concreteDispatchState state) := rfl

@[simp] private theorem stateToSound_requestCancellationState (state : ConcreteState) :
    { stateToSound state with lifecycle := .CancellationAccepted } =
      stateToSound (concreteRequestCancellationState state) := rfl

@[simp] private theorem stateToSound_acquireOwnershipState (state : ConcreteState) :
    { stateToSound state with ownerEpoch := .Epoch1 } =
      stateToSound (concreteAcquireOwnershipState state) := rfl

@[simp] private theorem stateToSound_commitCancellationState (state : ConcreteState) :
    { stateToSound state with lifecycle := .Cancelled } =
      stateToSound (concreteCommitCancellationState state) := rfl

@[simp] private theorem stateToSound_workerReturnsSuccessState (state : ConcreteState) :
    { stateToSound state with task := .Returned, completionEpoch := (stateToSound state).workerEpoch } =
      stateToSound (concreteWorkerReturnsSuccessState state) := rfl

@[simp] private theorem stateToSound_persistSuccessState (state : ConcreteState) :
    { stateToSound state with lifecycle := .Succeeded } =
      stateToSound (concretePersistSuccessState state) := rfl

def concreteOutcomes (state : ConcreteState) (label : ConcreteLabel) :
    List (Veil.ExecutionOutcome Int ConcreteState) :=
  Veil.Extract.extractAllOutcomes
    (NexusCancellationSoundConcrete.NextAct.extracted ConcreteTheory ConcreteState
      ConcreteLifecycle ConcreteTaskStage ConcreteEpoch label)
    concreteTheory state

theorem concreteSystemNext_iff (state nextState : ConcreteState) (label : ConcreteLabel) :
    concreteNext state label nextState ↔
      Veil.ExecutionOutcome.success nextState ∈ concreteOutcomes state label := by
  cases label <;>
    simp [concreteNext, concreteSystem, concreteOutcomes,
      NexusCancellationSoundConcrete.enumerableTransitionSystem] <;>
    intro <;> exact Veil.Enumeration.complete _

theorem concreteDispatchOutcomes_eq (state : ConcreteState) :
    concreteOutcomes state .DispatchTask =
      if state.task = .Idle ∧ state.lifecycle = .Open then
        [.success (concreteDispatchState state)]
      else [] := by
  rcases state with ⟨lifecycle, task, ownerEpoch, workerEpoch, completionEpoch⟩
  cases lifecycle <;> cases task <;> rfl

theorem concreteRequestCancellationOutcomes_eq (state : ConcreteState) :
    concreteOutcomes state .RequestCancellation =
      if state.lifecycle = .Open then
        [.success (concreteRequestCancellationState state)]
      else [] := by
  rcases state with ⟨lifecycle, task, ownerEpoch, workerEpoch, completionEpoch⟩
  cases lifecycle <;> rfl

theorem concreteAcquireOwnershipOutcomes_eq (state : ConcreteState) :
    concreteOutcomes state .AcquireOwnership =
      if (((state.lifecycle = .Open ∨ state.lifecycle = .CancellationAccepted) ∨
          state.lifecycle = .Cancelled) ∧ state.ownerEpoch = .Epoch0) then
        [.success (concreteAcquireOwnershipState state)]
      else [] := by
  rcases state with ⟨lifecycle, task, ownerEpoch, workerEpoch, completionEpoch⟩
  cases lifecycle <;> cases ownerEpoch <;> rfl

theorem concreteCommitCancellationOutcomes_eq (state : ConcreteState) :
    concreteOutcomes state .CommitCancellation =
      if state.lifecycle = .CancellationAccepted then
        [.success (concreteCommitCancellationState state)]
      else [] := by
  rcases state with ⟨lifecycle, task, ownerEpoch, workerEpoch, completionEpoch⟩
  cases lifecycle <;> rfl

theorem concreteWorkerReturnsSuccessOutcomes_eq (state : ConcreteState) :
    concreteOutcomes state .WorkerReturnsSuccess =
      if state.task = .Dispatched then
        [.success (concreteWorkerReturnsSuccessState state)]
      else [] := by
  rcases state with ⟨lifecycle, task, ownerEpoch, workerEpoch, completionEpoch⟩
  cases task <;> rfl

theorem concretePersistSuccessOutcomes_eq (state : ConcreteState) :
    concreteOutcomes state .PersistSuccess =
      if (state.task = .Returned ∧ state.completionEpoch = state.ownerEpoch) ∧
          (state.lifecycle = .Open ∨ state.lifecycle = .CancellationAccepted) then
        [.success (concretePersistSuccessState state)]
      else [] := by
  rcases state with ⟨lifecycle, task, ownerEpoch, workerEpoch, completionEpoch⟩
  cases task <;> cases completionEpoch <;> cases ownerEpoch <;> cases lifecycle <;> rfl

theorem concreteNext_soundNext_iff (state : ConcreteState) (label : ConcreteLabel)
    (nextState : ConcreteState) :
    concreteNext state label nextState ↔
      soundSemanticRelation.next (stateToSound state) (labelToSound label)
        (stateToSound nextState) := by
  rw [concreteSystemNext_iff]
  cases label <;> simp only [labelToSound]
  · rw [concreteDispatchOutcomes_eq, soundSemanticRelation_dispatch_iff]
    rcases state with ⟨lifecycle, task, ownerEpoch, workerEpoch, completionEpoch⟩
    cases task <;> cases lifecycle <;> simp [taskToSound, lifecycleToSound]
    constructor
    · intro equal
      subst nextState
      rfl
    · intro equal
      apply stateToSound_injective
      simpa [stateToSound, concreteDispatchState] using equal.symm
  · rw [concreteRequestCancellationOutcomes_eq,
      soundSemanticRelation_requestCancellation_iff]
    rcases state with ⟨lifecycle, task, ownerEpoch, workerEpoch, completionEpoch⟩
    cases lifecycle <;> simp [lifecycleToSound]
    constructor
    · intro equal
      subst nextState
      rfl
    · intro equal
      apply stateToSound_injective
      simpa [stateToSound, concreteRequestCancellationState] using equal.symm
  · rw [concreteAcquireOwnershipOutcomes_eq,
      soundSemanticRelation_acquireOwnership_iff]
    rcases state with ⟨lifecycle, task, ownerEpoch, workerEpoch, completionEpoch⟩
    cases lifecycle <;> cases ownerEpoch <;> simp [lifecycleToSound, epochToSound]
    all_goals
      constructor
      · intro equal
        subst nextState
        rfl
      · intro equal
        apply stateToSound_injective
        simpa [stateToSound, concreteAcquireOwnershipState] using equal.symm
  · rw [concreteCommitCancellationOutcomes_eq,
      soundSemanticRelation_commitCancellation_iff]
    rcases state with ⟨lifecycle, task, ownerEpoch, workerEpoch, completionEpoch⟩
    cases lifecycle <;> simp [lifecycleToSound]
    constructor
    · intro equal
      subst nextState
      rfl
    · intro equal
      apply stateToSound_injective
      simpa [stateToSound, concreteCommitCancellationState] using equal.symm
  · rw [concreteWorkerReturnsSuccessOutcomes_eq,
      soundSemanticRelation_workerReturnsSuccess_iff]
    rcases state with ⟨lifecycle, task, ownerEpoch, workerEpoch, completionEpoch⟩
    cases task <;> simp [taskToSound]
    constructor
    · intro equal
      subst nextState
      rfl
    · intro equal
      apply stateToSound_injective
      simpa [stateToSound, concreteWorkerReturnsSuccessState] using equal.symm
  · rw [concretePersistSuccessOutcomes_eq, soundSemanticRelation_persistSuccess_iff]
    rcases state with ⟨lifecycle, task, ownerEpoch, workerEpoch, completionEpoch⟩
    cases task <;> cases completionEpoch <;> cases ownerEpoch <;> cases lifecycle <;>
      simp [taskToSound, lifecycleToSound, epochToSound]
    all_goals
      constructor
      · intro equal
        subst nextState
        rfl
      · intro equal
        apply stateToSound_injective
        simpa [stateToSound, concretePersistSuccessState] using equal.symm

theorem concreteInitial_soundInitial_iff (state : ConcreteState) :
    concreteInitial state ↔ soundSemanticRelation.initial (stateToSound state) := by
  unfold concreteInitial
  rw [concreteInitialStates_eq]
  rw [soundSemanticRelation_initialState_iff]
  simp only [List.mem_singleton]
  exact stateToSound_injective.eq_iff.symm

theorem concreteProperty_soundProperty_iff (state : ConcreteState) :
    concreteProperty state ↔ soundSemanticRelation.property (stateToSound state) := by
  rw [soundSemanticRelation_propertyState_iff]
  unfold concreteProperty NexusCancellationSoundConcrete.NexusCancellationWonExcludesSuccess
  rcases state with ⟨lifecycle, task, ownerEpoch, workerEpoch, completionEpoch⟩
  cases lifecycle <;> cases task <;> cases ownerEpoch <;> cases workerEpoch <;>
    cases completionEpoch <;>
    decide

theorem concreteInitial_iff (state : ConcreteState) :
    concreteInitial state ↔
      Umpire3.Temporal.Targets.NexusCancellationFencing.soundFirstOrderArtifact.initial.eval
        (encodeConcreteState state) = true :=
  (concreteInitial_soundInitial_iff state).trans
    (soundSemanticRelation.initial_iff (stateToSound state))

theorem concreteNext_iff (state : ConcreteState) (label : ConcreteLabel)
    (nextState : ConcreteState) :
    concreteNext state label nextState ↔
      Umpire3.Temporal.Targets.NexusCancellationFencing.soundFirstOrderArtifact.next
          (encodeConcreteState state) (encodeConcreteLabel label) =
        some (encodeConcreteState nextState) :=
  (concreteNext_soundNext_iff state label nextState).trans
    (soundSemanticRelation.next_iff
      (stateToSound state) (labelToSound label) (stateToSound nextState))

theorem concreteProperty_iff (state : ConcreteState) :
    concreteProperty state ↔
      Umpire3.Temporal.Targets.NexusCancellationFencing.soundFirstOrderArtifact.invariant.eval
        (encodeConcreteState state) = true :=
  (concreteProperty_soundProperty_iff state).trans
    (soundSemanticRelation.property_iff (stateToSound state))

def soundConcreteSemanticRelation :
    Umpire3.Veil.SemanticRelation
      Umpire3.Temporal.Targets.NexusCancellationFencing.soundFirstOrderArtifact where
  State := ConcreteState
  Label := ConcreteLabel
  initial := concreteInitial
  next := concreteNext
  property := concreteProperty
  encodeState := encodeConcreteState
  encodeLabel := encodeConcreteLabel
  initial_iff := concreteInitial_iff
  next_iff := concreteNext_iff
  property_iff := concreteProperty_iff
  state_injective := by
    intro left right equal
    apply stateToSound_injective
    apply soundSemanticRelation.state_injective
    simpa [encodeConcreteState] using equal
  label_injective := by
    intro left right equal
    apply labelToSound_injective
    apply soundSemanticRelation.label_injective
    simpa [encodeConcreteLabel] using equal
  label_total := by
    intro label
    simpa [encodeConcreteLabel] using soundSemanticRelation.label_total (labelToSound label)
  label_complete := by
    intro identifier member
    obtain ⟨label, equal⟩ := soundSemanticRelation.label_complete identifier member
    cases label
    · exact ⟨.DispatchTask, by simpa [encodeConcreteLabel, labelToSound] using equal⟩
    · exact ⟨.RequestCancellation, by simpa [encodeConcreteLabel, labelToSound] using equal⟩
    · exact ⟨.AcquireOwnership, by simpa [encodeConcreteLabel, labelToSound] using equal⟩
    · exact ⟨.CommitCancellation, by simpa [encodeConcreteLabel, labelToSound] using equal⟩
    · exact ⟨.WorkerReturnsSuccess, by simpa [encodeConcreteLabel, labelToSound] using equal⟩
    · exact ⟨.PersistSuccess, by simpa [encodeConcreteLabel, labelToSound] using equal⟩

def soundSemanticBinding :
    Umpire3.Veil.SemanticBinding
      Umpire3.Temporal.Targets.NexusCancellationFencing.soundFirstOrderArtifact where
  symbolic := soundSemanticRelation
  concrete := soundConcreteSemanticRelation

end Umpire3.Temporal.Veil.NexusCancellationFencing
