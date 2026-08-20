import Temporal.Product.Nexus
import Temporal.System.TaskDelivery

namespace Umpire3.Temporal.System.NexusTasks

structure EffectProgress where
  attempted : Bool
  applied : Bool
  committed : Bool
  persisted : Bool
  observed : Bool
  deriving DecidableEq, Inhabited, Repr

def noProgress : EffectProgress where
  attempted := false
  applied := false
  committed := false
  persisted := false
  observed := false

inductive TaskStage where
  | idle
  | dispatched
  | returned
  | acknowledged
  deriving BEq, DecidableEq, Inhabited, Repr

structure State where
  scheduled : Bool
  task : TaskStage
  attempt : Nat
  ownerEpoch : Nat
  ownerAvailable : Bool
  workerEpoch : Option Nat
  staleWorkerEpoch : Option Nat
  completionEpoch : Option Nat
  cancellation : EffectProgress
  success : EffectProgress
  visible : Temporal.Product.Nexus.State
  deriving DecidableEq, Inhabited, Repr

inductive Action where
  | ScheduleOperation
  | DispatchTask
  | WorkerReturnsSuccess
  | RequestCancellation
  | CommitCancellation
  | PersistSuccess
  | RetryTask
  | AcquireOwnership
  | CrashOwner
  | RecoverOwner
  | AckTask
  deriving DecidableEq, Inhabited, Repr

def initial : State where
  scheduled := false
  task := .idle
  attempt := 0
  ownerEpoch := 0
  ownerAvailable := true
  workerEpoch := none
  staleWorkerEpoch := none
  completionEpoch := none
  cancellation := noProgress
  success := noProgress
  visible := .active

structure TransitionResult (state : State) where
  nextState : State
  productActions : List Temporal.Product.Nexus.Command
  productRun : Runs Temporal.Product.Nexus.product state.visible productActions nextState.visible

def stutterResult (state nextState : State)
    (sameVisible : nextState.visible = state.visible) : TransitionResult state where
  nextState := nextState
  productActions := []
  productRun := by
    rw [sameVisible]
    exact Runs.nil (model := Temporal.Product.Nexus.product) state.visible

def productResult (state nextState : State)
    (action : Temporal.Product.Nexus.Command)
    (productStep : Temporal.Product.Nexus.product.Step state.visible action nextState.visible) :
    TransitionResult state where
  nextState := nextState
  productActions := [action]
  productRun := Runs.cons productStep
    (Runs.nil (model := Temporal.Product.Nexus.product) nextState.visible)

def returnedEpoch (state : State) : Option Nat :=
  match state.staleWorkerEpoch with
  | some epoch => some epoch
  | none => state.workerEpoch

def transitions (state : State) : Action → List (TransitionResult state)
  | .ScheduleOperation =>
      if state.scheduled = true then []
      else [stutterResult state { state with scheduled := true } rfl]
  | .DispatchTask =>
      if state.scheduled = true ∧ state.ownerAvailable = true ∧ state.task = .idle then
        [stutterResult state { state with
          task := .dispatched
          workerEpoch := some state.ownerEpoch } rfl]
      else []
  | .WorkerReturnsSuccess =>
      match returnedEpoch state with
      | some epoch =>
          if state.task = .dispatched then
            [stutterResult state { state with
              task := .returned
              completionEpoch := some epoch
              staleWorkerEpoch := none
              success := { state.success with attempted := true, applied := true } } rfl]
          else []
      | none => []
  | .RequestCancellation =>
      if active : state.visible = .active then
        [productResult state { state with
          cancellation := { state.cancellation with attempted := true, applied := true }
          visible := .cancellationAccepted } .acceptCancellation (by
            cases visible : state.visible <;> simp_all [Temporal.Product.Nexus.step])]
      else []
  | .CommitCancellation =>
      if cancellable : state.cancellation.applied = true ∧
          state.visible = .cancellationAccepted then
        [productResult state { state with
          cancellation := { state.cancellation with
            committed := true, persisted := true, observed := true }
          visible := .cancelled } .winCancellation (by
            cases visible : state.visible <;> simp_all [Temporal.Product.Nexus.step])]
      else []
  | .PersistSuccess =>
      if persistable : state.success.applied = true ∧
          Temporal.System.TaskDelivery.CurrentCompletion state.completionEpoch state.ownerEpoch ∧
          (state.visible = .active ∨ state.visible = .cancellationAccepted) then
        [productResult state { state with
          success := { state.success with
            committed := true, persisted := true, observed := true }
          visible := .succeeded } .completeSuccess (by
            cases visible : state.visible <;> simp_all [Temporal.Product.Nexus.step])]
      else []
  | .RetryTask =>
      if state.scheduled = true ∧ state.ownerAvailable = true then
        [stutterResult state { state with
          task := .dispatched
          attempt := state.attempt + 1
          workerEpoch := some state.ownerEpoch
          completionEpoch := none } rfl]
      else []
  | .AcquireOwnership =>
      let staleEpoch :=
        if state.task = .dispatched then state.workerEpoch else state.staleWorkerEpoch
      [stutterResult state { state with
        ownerEpoch := state.ownerEpoch + 1
        ownerAvailable := true
        staleWorkerEpoch := staleEpoch } rfl]
  | .CrashOwner =>
      if state.ownerAvailable = true then
        [stutterResult state { state with ownerAvailable := false } rfl]
      else []
  | .RecoverOwner =>
      if state.ownerAvailable = true then []
      else [stutterResult state { state with ownerAvailable := true } rfl]
  | .AckTask =>
      if state.task = .returned then
        [stutterResult state { state with
          task := .acknowledged
          success := { state.success with observed := true } } rfl]
      else []

def next (state : State) (action : Action) : List State :=
  (transitions state action).map TransitionResult.nextState

def step (state : State) (action : Action) (nextState : State) : Prop :=
  ∃ result, result ∈ transitions state action ∧ result.nextState = nextState

abbrev system : TransitionSystem where
  State := State
  Action := Action
  Initial := fun state => state = initial
  Step := step

theorem next_iff (state action nextState) :
    nextState ∈ next state action ↔ system.Step state action nextState := by
  constructor
  · intro member
    rcases List.mem_map.mp member with ⟨result, resultMember, equality⟩
    exact ⟨result, resultMember, equality⟩
  · rintro ⟨result, resultMember, rfl⟩
    exact List.mem_map.mpr ⟨result, resultMember, rfl⟩

def executable : ExecutableModel system where
  next := next
  next_iff := next_iff

def bounded : BoundedModel system where
  toExecutableModel := executable
  initials := [initial]
  initial_iff := by simp [system]
  actions := [
    .ScheduleOperation,
    .DispatchTask,
    .WorkerReturnsSuccess,
    .RequestCancellation,
    .CommitCancellation,
    .PersistSuccess,
    .RetryTask,
    .AcquireOwnership,
    .CrashOwner,
    .RecoverOwner,
    .AckTask,
  ]
  action_complete := by
    intro state action nextState _
    cases action <;> simp

def assumptions : List Assumption := [
  { identifier := "persistence-commit-atomicity",
    statementHash := "sha256:61db93cf9c68dfc18c4d379f71dc5b03ae45854bb70c94c1b390e30ba52e50ad" },
  { identifier := "task-at-least-once-delivery",
    statementHash := "sha256:4bb260aba066d97231a70090468b1c6bd1ad4bba387742a54f8110a484270415" },
  { identifier := Temporal.System.TaskDelivery.guarantee.identifier,
    statementHash := Temporal.System.TaskDelivery.guarantee.statementHash },
]

def nexusDeliveryRequirement : Temporal.System.TaskDelivery.Requirement where
  provider := Temporal.System.TaskDelivery.guarantee.identifier
  statementHash := Temporal.System.TaskDelivery.guarantee.statementHash

def StutteringAction (action : Action) : Prop :=
  ∀ state result, result ∈ transitions state action → result.productActions = []

def unsafeNext (state : State) (action : Action) : List State :=
  match action with
  | .PersistSuccess =>
      if state.success.applied then
        [{ state with
          success := { state.success with
            committed := true, persisted := true, observed := true }
          visible := .succeeded }]
      else []
  | _ => next state action

def unsafeStep (state : State) (action : Action) (nextState : State) : Prop :=
  nextState ∈ unsafeNext state action

abbrev unsafeSystem : TransitionSystem where
  State := State
  Action := Action
  Initial := fun state => state = initial
  Step := unsafeStep

def unsafeExecutable : ExecutableModel unsafeSystem where
  next := unsafeNext
  next_iff := by
    intro state action nextState
    rfl

def unsafeBounded : BoundedModel unsafeSystem where
  toExecutableModel := unsafeExecutable
  initials := [initial]
  initial_iff := by simp [unsafeSystem]
  actions := bounded.actions
  action_complete := by
    intro state action nextState _
    cases action <;> simp [bounded]

end Umpire3.Temporal.System.NexusTasks
