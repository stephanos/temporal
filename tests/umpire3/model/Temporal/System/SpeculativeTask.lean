import Temporal.Product.SpeculativeTask

namespace Umpire3.Temporal.System.SpeculativeTask

namespace Product

abbrev State := Umpire3.Temporal.Product.SpeculativeTask.State
abbrev Action := Umpire3.Temporal.Product.SpeculativeTask.Action
abbrev TaskID := Umpire3.Temporal.Product.SpeculativeTask.TaskID
abbrev UpdateID := Umpire3.Temporal.Product.SpeculativeTask.UpdateID
abbrev model := Umpire3.Temporal.Product.SpeculativeTask.model
def initial := Umpire3.Temporal.Product.SpeculativeTask.initial
def next := Umpire3.Temporal.Product.SpeculativeTask.next
def executable := Umpire3.Temporal.Product.SpeculativeTask.executable
def actions := Umpire3.Temporal.Product.SpeculativeTask.actions
def taskIDs := Umpire3.Temporal.Product.SpeculativeTask.taskIDs
theorem action_mem (action : Action) : action ∈ actions :=
  Umpire3.Temporal.Product.SpeculativeTask.action_mem action
theorem task_mem (task : TaskID) : task ∈ taskIDs :=
  Umpire3.Temporal.Product.SpeculativeTask.task_mem task
def committedFinal := Umpire3.Temporal.Product.SpeculativeTask.committedFinal

end Product

structure State where
  visible : Product.State
  updateObservations : Nat
  taskDeliveries : Nat
  deriving DecidableEq, Inhabited, Repr

inductive Action where
  | observeUpdateRequest (update : Product.UpdateID)
  | observeSpeculativeCreation (task : Product.TaskID) (update : Product.UpdateID)
  | observeCommit (task : Product.TaskID)
  | redeliverCommit (task : Product.TaskID)
  deriving DecidableEq, Inhabited, Repr

def initial : State where
  visible := Product.initial
  updateObservations := 0
  taskDeliveries := 0

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
    (taskDelivery : Bool) : List (TransitionResult state) :=
  (Product.next state.visible action).attach.map fun successor => {
    nextState := if taskDelivery then
      { state with visible := successor.1, taskDeliveries := state.taskDeliveries + 1 }
    else
      { state with visible := successor.1, updateObservations := state.updateObservations + 1 }
    productActions := [action]
    productRun := by
      split <;>
        exact Runs.cons ((Product.executable.next_iff _ _ _).mp successor.2)
          (Runs.nil (model := Product.model) successor.1)
  }

def transitions (state : State) : Action → List (TransitionResult state)
  | .observeUpdateRequest update => liftProduct state (.requestUpdate update) false
  | .observeSpeculativeCreation task update => liftProduct state (.create task update) true
  | .observeCommit task => liftProduct state (.commit task) true
  | .redeliverCommit task =>
      if state.visible.taskState task == .committed then
        [stutterResult state { state with taskDeliveries := state.taskDeliveries + 1 } rfl]
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
  | .requestUpdate update => .observeUpdateRequest update
  | .create task update => .observeSpeculativeCreation task update
  | .commit task => .observeCommit task

def actions : List Action := Product.actions.map fromProduct ++
  Product.taskIDs.map Action.redeliverCommit

theorem action_mem (action : Action) : action ∈ actions := by
  cases action with
  | observeUpdateRequest update =>
      apply List.mem_append_left
      exact List.mem_map.mpr ⟨.requestUpdate update, Product.action_mem _, rfl⟩
  | observeSpeculativeCreation task update =>
      apply List.mem_append_left
      exact List.mem_map.mpr ⟨.create task update, Product.action_mem _, rfl⟩
  | observeCommit task =>
      apply List.mem_append_left
      exact List.mem_map.mpr ⟨.commit task, Product.action_mem _, rfl⟩
  | redeliverCommit task =>
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

def committedFinal : State := {
  initial with visible := Product.committedFinal, updateObservations := 1, taskDeliveries := 3
}

end Umpire3.Temporal.System.SpeculativeTask
