import Temporal.Product.WorkflowProgress

namespace Umpire3.Temporal.System.WorkflowProgress

namespace Product

abbrev State := Umpire3.Temporal.Product.WorkflowProgress.State
abbrev Action := Umpire3.Temporal.Product.WorkflowProgress.Action
abbrev TaskID := Umpire3.Temporal.Product.WorkflowProgress.TaskID
abbrev EntityID := Umpire3.Temporal.Product.WorkflowProgress.EntityID
abbrev WorkerID := Umpire3.Temporal.Product.WorkflowProgress.WorkerID
abbrev model := Umpire3.Temporal.Product.WorkflowProgress.model
def initial := Umpire3.Temporal.Product.WorkflowProgress.initial
def next := Umpire3.Temporal.Product.WorkflowProgress.next
def executable := Umpire3.Temporal.Product.WorkflowProgress.executable
def actions := Umpire3.Temporal.Product.WorkflowProgress.actions
def taskIDs := Umpire3.Temporal.Product.WorkflowProgress.taskIDs
theorem action_mem (action : Action) : action ∈ actions :=
  Umpire3.Temporal.Product.WorkflowProgress.action_mem action
theorem task_mem (task : TaskID) : task ∈ taskIDs :=
  Umpire3.Temporal.Product.WorkflowProgress.task_mem task
def progressedFinal := Umpire3.Temporal.Product.WorkflowProgress.progressedFinal

end Product

structure State where
  visible : Product.State
  lifecycleObservations : Nat
  completionDeliveries : Nat
  deriving DecidableEq, Inhabited, Repr

inductive Action where
  | observeEnqueue (task : Product.TaskID) (entity : Product.EntityID)
  | observeWorkerAvailability (worker : Product.WorkerID)
  | observeWait (task : Product.TaskID)
  | observeDispatch (task : Product.TaskID) (worker : Product.WorkerID)
  | observeCompletion (task : Product.TaskID) (entity : Product.EntityID)
  | redeliverCompletion (task : Product.TaskID)
  deriving DecidableEq, Inhabited, Repr

def initial : State where
  visible := Product.initial
  lifecycleObservations := 0
  completionDeliveries := 0

structure TransitionResult (state : State) where
  nextState : State
  productActions : List Product.Action
  productRun : Runs Product.model state.visible productActions nextState.visible

def stutterResult (state nextState : State)
    (sameVisible : nextState.visible = state.visible) : TransitionResult state where
  nextState
  productActions := []
  productRun := by
    rw [sameVisible]
    exact Runs.nil (model := Product.model) state.visible

def liftProduct (state : State) (action : Product.Action)
    (completion : Bool) : List (TransitionResult state) :=
  (Product.next state.visible action).attach.map fun successor => {
    nextState := if completion then
      { state with visible := successor.1, completionDeliveries := state.completionDeliveries + 1 }
    else
      { state with visible := successor.1, lifecycleObservations := state.lifecycleObservations + 1 }
    productActions := [action]
    productRun := by
      split <;>
        exact Runs.cons ((Product.executable.next_iff _ _ _).mp successor.2)
          (Runs.nil (model := Product.model) successor.1)
  }

def transitions (state : State) : Action → List (TransitionResult state)
  | .observeEnqueue task entity => liftProduct state (.enqueue task entity) false
  | .observeWorkerAvailability worker => liftProduct state (.makeWorkerAvailable worker) false
  | .observeWait task => liftProduct state (.wait task) false
  | .observeDispatch task worker => liftProduct state (.dispatch task worker) false
  | .observeCompletion task entity => liftProduct state (.complete task entity) true
  | .redeliverCompletion task =>
      if state.visible.taskState task == .completed then
        [stutterResult state
          { state with completionDeliveries := state.completionDeliveries + 1 } rfl]
      else []

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

def fromProduct : Product.Action → Action
  | .enqueue task entity => .observeEnqueue task entity
  | .makeWorkerAvailable worker => .observeWorkerAvailability worker
  | .wait task => .observeWait task
  | .dispatch task worker => .observeDispatch task worker
  | .complete task entity => .observeCompletion task entity

def actions : List Action := Product.actions.map fromProduct ++
  Product.taskIDs.map Action.redeliverCompletion

theorem action_mem (action : Action) : action ∈ actions := by
  cases action with
  | observeEnqueue task entity =>
      apply List.mem_append_left
      exact List.mem_map.mpr ⟨.enqueue task entity, Product.action_mem _, rfl⟩
  | observeWorkerAvailability worker =>
      apply List.mem_append_left
      exact List.mem_map.mpr ⟨.makeWorkerAvailable worker, Product.action_mem _, rfl⟩
  | observeWait task =>
      apply List.mem_append_left
      exact List.mem_map.mpr ⟨.wait task, Product.action_mem _, rfl⟩
  | observeDispatch task worker =>
      apply List.mem_append_left
      exact List.mem_map.mpr ⟨.dispatch task worker, Product.action_mem _, rfl⟩
  | observeCompletion task entity =>
      apply List.mem_append_left
      exact List.mem_map.mpr ⟨.complete task entity, Product.action_mem _, rfl⟩
  | redeliverCompletion task =>
      apply List.mem_append_right
      exact List.mem_map.mpr ⟨task, Product.task_mem task, rfl⟩

def bounded : BoundedModel system where
  toExecutableModel := executable
  initials := [initial]
  initial_iff := by intro state; simp
  actions := actions
  action_complete := by
    intro state action nextState _
    exact action_mem action

def progressedFinal : State := {
  initial with visible := Product.progressedFinal, lifecycleObservations := 4, completionDeliveries := 2
}

end Umpire3.Temporal.System.WorkflowProgress
