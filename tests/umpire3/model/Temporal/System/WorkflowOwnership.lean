import Temporal.Product.WorkflowOwnership

namespace Umpire3.Temporal.System.WorkflowOwnership

namespace Product

abbrev State := Umpire3.Temporal.Product.WorkflowOwnership.State
abbrev Action := Umpire3.Temporal.Product.WorkflowOwnership.Action
abbrev TaskID := Umpire3.Temporal.Product.WorkflowOwnership.TaskID
abbrev AttemptID := Umpire3.Temporal.Product.WorkflowOwnership.AttemptID
abbrev Epoch := Umpire3.Temporal.Product.WorkflowOwnership.Epoch
abbrev model := Umpire3.Temporal.Product.WorkflowOwnership.model
def initial := Umpire3.Temporal.Product.WorkflowOwnership.initial
def next := Umpire3.Temporal.Product.WorkflowOwnership.next
def executable := Umpire3.Temporal.Product.WorkflowOwnership.executable
def actions := Umpire3.Temporal.Product.WorkflowOwnership.actions
def attemptIDs := Umpire3.Temporal.Product.WorkflowOwnership.attemptIDs
theorem action_mem (action : Action) : action ∈ actions :=
  Umpire3.Temporal.Product.WorkflowOwnership.action_mem action
theorem attempt_mem (attempt : AttemptID) : attempt ∈ attemptIDs :=
  Umpire3.Temporal.Product.WorkflowOwnership.attempt_mem attempt
def fencedFinal := Umpire3.Temporal.Product.WorkflowOwnership.fencedFinal

end Product

structure State where
  visible : Product.State
  ownershipObservations : Nat
  completionDeliveries : Nat
  deriving DecidableEq, Inhabited, Repr

inductive Action where
  | observeBootstrap (task : Product.TaskID) (epoch : Product.Epoch)
  | observeDispatch (attempt : Product.AttemptID) (task : Product.TaskID)
      (epoch : Product.Epoch)
  | observeFailure (attempt : Product.AttemptID)
  | observeRotation (task : Product.TaskID) (previous : Product.Epoch)
      (current : Product.Epoch)
  | observeStaleRejection (attempt : Product.AttemptID)
  | observeCompletion (attempt : Product.AttemptID)
  | redeliverCompletion (attempt : Product.AttemptID)
  deriving DecidableEq, Inhabited, Repr

def initial : State where
  visible := Product.initial
  ownershipObservations := 0
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
      { state with visible := successor.1, ownershipObservations := state.ownershipObservations + 1 }
    productActions := [action]
    productRun := by
      split <;>
        exact Runs.cons ((Product.executable.next_iff _ _ _).mp successor.2)
          (Runs.nil (model := Product.model) successor.1)
  }

def transitions (state : State) : Action → List (TransitionResult state)
  | .observeBootstrap task epoch => liftProduct state (.bootstrap task epoch) false
  | .observeDispatch attempt task epoch => liftProduct state (.dispatch attempt task epoch) false
  | .observeFailure attempt => liftProduct state (.fail attempt) false
  | .observeRotation task previous current => liftProduct state (.rotate task previous current) false
  | .observeStaleRejection attempt => liftProduct state (.rejectStale attempt) false
  | .observeCompletion attempt => liftProduct state (.complete attempt) true
  | .redeliverCompletion attempt =>
      if state.visible.attemptState attempt == .completed then
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
  | .bootstrap task epoch => .observeBootstrap task epoch
  | .dispatch attempt task epoch => .observeDispatch attempt task epoch
  | .fail attempt => .observeFailure attempt
  | .rotate task previous current => .observeRotation task previous current
  | .rejectStale attempt => .observeStaleRejection attempt
  | .complete attempt => .observeCompletion attempt

def actions : List Action := Product.actions.map fromProduct ++
  Product.attemptIDs.map Action.redeliverCompletion

theorem action_mem (action : Action) : action ∈ actions := by
  cases action with
  | observeBootstrap task epoch =>
      apply List.mem_append_left
      exact List.mem_map.mpr ⟨.bootstrap task epoch, Product.action_mem _, rfl⟩
  | observeDispatch attempt task epoch =>
      apply List.mem_append_left
      exact List.mem_map.mpr ⟨.dispatch attempt task epoch, Product.action_mem _, rfl⟩
  | observeFailure attempt =>
      apply List.mem_append_left
      exact List.mem_map.mpr ⟨.fail attempt, Product.action_mem _, rfl⟩
  | observeRotation task previous current =>
      apply List.mem_append_left
      exact List.mem_map.mpr ⟨.rotate task previous current, Product.action_mem _, rfl⟩
  | observeStaleRejection attempt =>
      apply List.mem_append_left
      exact List.mem_map.mpr ⟨.rejectStale attempt, Product.action_mem _, rfl⟩
  | observeCompletion attempt =>
      apply List.mem_append_left
      exact List.mem_map.mpr ⟨.complete attempt, Product.action_mem _, rfl⟩
  | redeliverCompletion attempt =>
      apply List.mem_append_right
      exact List.mem_map.mpr ⟨attempt, Product.attempt_mem attempt, rfl⟩

def bounded : BoundedModel system where
  toExecutableModel := executable
  initials := [initial]
  initial_iff := by intro state; simp
  actions := actions
  action_complete := by
    intro state action nextState _
    exact action_mem action

def fencedFinal : State := {
  initial with
  visible := Product.fencedFinal
  ownershipObservations := 6
  completionDeliveries := 2
}

end Umpire3.Temporal.System.WorkflowOwnership
