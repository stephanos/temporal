import Temporal.Product.NexusClosure

namespace Umpire3.Temporal.System.NexusClosure

inductive TaskStage where
  | idle
  | dispatched
  | returned
  deriving BEq, DecidableEq, Inhabited, Repr

structure State where
  visible : Umpire3.Temporal.Product.NexusClosure.State
  primaryTask : TaskStage
  secondaryTask : TaskStage
  primaryAttempt : Nat
  secondaryAttempt : Nat
  ownerEpoch : Nat
  ownerAvailable : Bool
  primaryWorkerEpoch : Option Nat
  secondaryWorkerEpoch : Option Nat
  primaryCompletionEpoch : Option Nat
  secondaryCompletionEpoch : Option Nat
  deriving DecidableEq, Inhabited, Repr

def State.task (state : State) : Umpire3.Temporal.Product.NexusClosure.OperationID → TaskStage
  | .primary => state.primaryTask
  | .secondary => state.secondaryTask

def State.setTask (state : State) : Umpire3.Temporal.Product.NexusClosure.OperationID → TaskStage → State
  | .primary, task => { state with primaryTask := task }
  | .secondary, task => { state with secondaryTask := task }

def State.attempt (state : State) : Umpire3.Temporal.Product.NexusClosure.OperationID → Nat
  | .primary => state.primaryAttempt
  | .secondary => state.secondaryAttempt

def State.setAttempt (state : State) : Umpire3.Temporal.Product.NexusClosure.OperationID → Nat → State
  | .primary, attempt => { state with primaryAttempt := attempt }
  | .secondary, attempt => { state with secondaryAttempt := attempt }

def State.workerEpoch (state : State) : Umpire3.Temporal.Product.NexusClosure.OperationID → Option Nat
  | .primary => state.primaryWorkerEpoch
  | .secondary => state.secondaryWorkerEpoch

def State.setWorkerEpoch (state : State) :
    Umpire3.Temporal.Product.NexusClosure.OperationID → Option Nat → State
  | .primary, epoch => { state with primaryWorkerEpoch := epoch }
  | .secondary, epoch => { state with secondaryWorkerEpoch := epoch }

def State.completionEpoch (state : State) :
    Umpire3.Temporal.Product.NexusClosure.OperationID → Option Nat
  | .primary => state.primaryCompletionEpoch
  | .secondary => state.secondaryCompletionEpoch

def State.setCompletionEpoch (state : State) :
    Umpire3.Temporal.Product.NexusClosure.OperationID → Option Nat → State
  | .primary, epoch => { state with primaryCompletionEpoch := epoch }
  | .secondary, epoch => { state with secondaryCompletionEpoch := epoch }

inductive Action where
  | scheduleOperation (operation : Umpire3.Temporal.Product.NexusClosure.OperationID)
  | dispatchTask (operation : Umpire3.Temporal.Product.NexusClosure.OperationID)
  | observeStart (operation : Umpire3.Temporal.Product.NexusClosure.OperationID)
  | workerReturns (operation : Umpire3.Temporal.Product.NexusClosure.OperationID)
  | persist (operation : Umpire3.Temporal.Product.NexusClosure.OperationID)
      (outcome : Umpire3.Temporal.Product.NexusClosure.OperationOutcome)
  | retryTask (operation : Umpire3.Temporal.Product.NexusClosure.OperationID)
  | acquireOwnership
  | crashOwner
  | recoverOwner
  | closeWorkflow (outcome : Umpire3.Temporal.Product.NexusClosure.WorkflowOutcome)
  deriving DecidableEq, Inhabited, Repr

def initial : State where
  visible := Umpire3.Temporal.Product.NexusClosure.initial
  primaryTask := .idle
  secondaryTask := .idle
  primaryAttempt := 0
  secondaryAttempt := 0
  ownerEpoch := 0
  ownerAvailable := true
  primaryWorkerEpoch := none
  secondaryWorkerEpoch := none
  primaryCompletionEpoch := none
  secondaryCompletionEpoch := none

structure TransitionResult (state : State) where
  nextState : State
  productActions : List Umpire3.Temporal.Product.NexusClosure.Action
  productRun : Runs Umpire3.Temporal.Product.NexusClosure.model
    state.visible productActions nextState.visible

def stutterResult (state nextState : State)
    (sameVisible : nextState.visible = state.visible) : TransitionResult state where
  nextState := nextState
  productActions := []
  productRun := by
    rw [sameVisible]
    exact Runs.nil (model := Umpire3.Temporal.Product.NexusClosure.model) state.visible

def liftProduct (state : State) (action : Umpire3.Temporal.Product.NexusClosure.Action)
    (decorate : Umpire3.Temporal.Product.NexusClosure.State → State)
    (projects : ∀ visible, (decorate visible).visible = visible) : List (TransitionResult state) :=
  (Umpire3.Temporal.Product.NexusClosure.next state.visible action).attach.map fun successor => {
    nextState := decorate successor.1
    productActions := [action]
    productRun := by
      rw [projects]
      exact Runs.cons
        ((Umpire3.Temporal.Product.NexusClosure.executable.next_iff _ _ _).mp successor.2)
        (Runs.nil (model := Umpire3.Temporal.Product.NexusClosure.model) successor.1)
  }

def transitions (state : State) : Action → List (TransitionResult state)
  | .scheduleOperation operation =>
      liftProduct state (.schedule operation) (fun visible => { state with visible }) (by intros; rfl)
  | .dispatchTask operation =>
      if state.ownerAvailable && state.task operation == .idle &&
          state.visible.operation operation == .scheduled then
        let nextState := ((state.setTask operation .dispatched).setWorkerEpoch operation
          (some state.ownerEpoch)).setCompletionEpoch operation none
        [stutterResult state nextState (by cases operation <;> rfl)]
      else []
  | .observeStart operation =>
      if (state.visible.operation operation).terminal then
        [stutterResult state state rfl]
      else
        liftProduct state (.start operation) (fun visible => { state with visible }) (by intros; rfl)
  | .workerReturns operation =>
      if state.task operation == .dispatched then
        match state.workerEpoch operation with
        | some epoch =>
            let nextState := (state.setTask operation .returned).setCompletionEpoch operation (some epoch)
            [stutterResult state nextState (by cases operation <;> rfl)]
        | none => []
      else []
  | .persist operation outcome =>
      if state.task operation == .returned &&
          state.completionEpoch operation == some state.ownerEpoch then
        liftProduct state (.settle operation outcome) (fun visible => { state with visible }) (by intros; rfl)
      else []
  | .retryTask operation =>
      if state.ownerAvailable && !state.visible.workflow.terminal then
        let nextState := (((state.setTask operation .dispatched).setAttempt operation
          (state.attempt operation + 1)).setWorkerEpoch operation
          (some state.ownerEpoch)).setCompletionEpoch operation none
        [stutterResult state nextState (by cases operation <;> rfl)]
      else []
  | .acquireOwnership =>
      [stutterResult state { state with ownerEpoch := state.ownerEpoch + 1, ownerAvailable := true } rfl]
  | .crashOwner =>
      if state.ownerAvailable then
        [stutterResult state { state with ownerAvailable := false } rfl]
      else []
  | .recoverOwner =>
      if state.ownerAvailable then []
      else [stutterResult state { state with ownerAvailable := true } rfl]
  | .closeWorkflow outcome =>
      liftProduct state (.closeWorkflow outcome) (fun visible => { state with visible }) (by intros; rfl)

def next (state : State) (action : Action) : List State :=
  (transitions state action).map TransitionResult.nextState

def step (state : State) (action : Action) (nextState : State) : Prop :=
  ∃ result, result ∈ transitions state action ∧ result.nextState = nextState

abbrev system : TransitionSystem where
  State := State
  Action := Action
  Initial := (· = initial)
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

def actions : List Action := [
  .scheduleOperation .primary,
  .scheduleOperation .secondary,
  .dispatchTask .primary,
  .dispatchTask .secondary,
  .observeStart .primary,
  .observeStart .secondary,
  .workerReturns .primary,
  .workerReturns .secondary,
  .persist .primary .succeeded,
  .persist .primary .failed,
  .persist .primary .cancelled,
  .persist .primary .timedOut,
  .persist .primary .terminated,
  .persist .primary .rejected,
  .persist .secondary .succeeded,
  .persist .secondary .failed,
  .persist .secondary .cancelled,
  .persist .secondary .timedOut,
  .persist .secondary .terminated,
  .persist .secondary .rejected,
  .retryTask .primary,
  .retryTask .secondary,
  .acquireOwnership,
  .crashOwner,
  .recoverOwner,
  .closeWorkflow .completed,
  .closeWorkflow .failed,
  .closeWorkflow .cancelled,
  .closeWorkflow .terminated,
  .closeWorkflow .timedOut,
]

def bounded : BoundedModel system where
  toExecutableModel := executable
  initials := [initial]
  initial_iff := by intro state; simp
  actions := actions
  action_complete := by
    intro state action nextState _
    cases action with
    | scheduleOperation operation => cases operation <;> simp [actions]
    | dispatchTask operation => cases operation <;> simp [actions]
    | observeStart operation => cases operation <;> simp [actions]
    | workerReturns operation => cases operation <;> simp [actions]
    | persist operation outcome => cases operation <;> cases outcome <;> simp [actions]
    | retryTask operation => cases operation <;> simp [actions]
    | acquireOwnership => simp [actions]
    | crashOwner => simp [actions]
    | recoverOwner => simp [actions]
    | closeWorkflow outcome => cases outcome <;> simp [actions]

def Closure (state : State) : Prop :=
  Umpire3.Temporal.Product.NexusClosure.Closure state.visible

instance (state : State) : Decidable (Closure state) :=
  Umpire3.Temporal.Product.NexusClosure.instDecidableClosure state.visible

theorem initialClosure : Closure initial := Umpire3.Temporal.Product.NexusClosure.initialClosure

theorem stepPreservesClosure {state action nextState}
    (property : Closure state) (transition : system.Step state action nextState) :
    Closure nextState := by
  rcases transition with ⟨result, _, rfl⟩
  exact Umpire3.Temporal.Product.NexusClosure.runsPreserveClosure result.productRun property

theorem runsPreserveClosure {start actionHistory final}
    (run : Runs system start actionHistory final) (property : Closure start) : Closure final := by
  induction run with
  | nil => exact property
  | cons transition _ induction => exact induction (stepPreservesClosure property transition)

theorem closureSafe : Safety system Closure := by
  intro state reachable
  rcases reachable with ⟨start, actionHistory, initialState, run⟩
  subst start
  exact runsPreserveClosure run initialClosure

def ordinaryFinal : State := {
  initial with
  visible := Umpire3.Temporal.Product.NexusClosure.permittedFinal
  primaryTask := .returned
  primaryWorkerEpoch := some 0
  primaryCompletionEpoch := some 0
}

def completionBeforeStartFinal : State := ordinaryFinal

end Umpire3.Temporal.System.NexusClosure
