import Temporal.Product.WorkflowRouting

namespace Umpire3.Temporal.System.WorkflowRouting

namespace Product

abbrev State := Umpire3.Temporal.Product.WorkflowRouting.State
abbrev Action := Umpire3.Temporal.Product.WorkflowRouting.Action
abbrev TaskID := Umpire3.Temporal.Product.WorkflowRouting.TaskID
abbrev PollerID := Umpire3.Temporal.Product.WorkflowRouting.PollerID
abbrev RouteID := Umpire3.Temporal.Product.WorkflowRouting.RouteID
abbrev AttemptID := Umpire3.Temporal.Product.WorkflowRouting.AttemptID
abbrev model := Umpire3.Temporal.Product.WorkflowRouting.model
def initial := Umpire3.Temporal.Product.WorkflowRouting.initial
def next := Umpire3.Temporal.Product.WorkflowRouting.next
def executable := Umpire3.Temporal.Product.WorkflowRouting.executable
def actions := Umpire3.Temporal.Product.WorkflowRouting.actions
def attemptIDs := Umpire3.Temporal.Product.WorkflowRouting.attemptIDs
theorem action_mem (action : Action) : action ∈ actions :=
  Umpire3.Temporal.Product.WorkflowRouting.action_mem action
theorem attempt_mem (attempt : AttemptID) : attempt ∈ attemptIDs :=
  Umpire3.Temporal.Product.WorkflowRouting.attempt_mem attempt
def matchingFinal := Umpire3.Temporal.Product.WorkflowRouting.matchingFinal

end Product

structure State where
  visible : Product.State
  routeObservations : Nat
  reservationDeliveries : Nat
  deriving DecidableEq, Inhabited, Repr

inductive Action where
  | observeTaskRoute (task : Product.TaskID) (route : Product.RouteID)
  | observePollerRoute (poller : Product.PollerID) (route : Product.RouteID)
  | observeReservation (attempt : Product.AttemptID) (task : Product.TaskID)
      (poller : Product.PollerID)
  | redeliverReservation (attempt : Product.AttemptID)
  deriving DecidableEq, Inhabited, Repr

def initial : State where
  visible := Product.initial
  routeObservations := 0
  reservationDeliveries := 0

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
    (reservation : Bool) : List (TransitionResult state) :=
  (Product.next state.visible action).attach.map fun successor => {
    nextState := if reservation then
      { state with visible := successor.1, reservationDeliveries := state.reservationDeliveries + 1 }
    else
      { state with visible := successor.1, routeObservations := state.routeObservations + 1 }
    productActions := [action]
    productRun := by
      split <;>
        exact Runs.cons ((Product.executable.next_iff _ _ _).mp successor.2)
          (Runs.nil (model := Product.model) successor.1)
  }

def transitions (state : State) : Action → List (TransitionResult state)
  | .observeTaskRoute task route => liftProduct state (.assignTask task route) false
  | .observePollerRoute poller route => liftProduct state (.registerPoller poller route) false
  | .observeReservation attempt task poller =>
      liftProduct state (.reserve attempt task poller) true
  | .redeliverReservation attempt =>
      if state.visible.attemptObserved attempt then
        [stutterResult state
          { state with reservationDeliveries := state.reservationDeliveries + 1 } rfl]
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
  | .assignTask task route => .observeTaskRoute task route
  | .registerPoller poller route => .observePollerRoute poller route
  | .reserve attempt task poller => .observeReservation attempt task poller

def actions : List Action := Product.actions.map fromProduct ++
  Product.attemptIDs.map Action.redeliverReservation

theorem action_mem (action : Action) : action ∈ actions := by
  cases action with
  | observeTaskRoute task route =>
      apply List.mem_append_left
      exact List.mem_map.mpr ⟨.assignTask task route, Product.action_mem _, rfl⟩
  | observePollerRoute poller route =>
      apply List.mem_append_left
      exact List.mem_map.mpr ⟨.registerPoller poller route, Product.action_mem _, rfl⟩
  | observeReservation attempt task poller =>
      apply List.mem_append_left
      exact List.mem_map.mpr ⟨.reserve attempt task poller, Product.action_mem _, rfl⟩
  | redeliverReservation attempt =>
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

def matchingFinal : State := {
  initial with
  visible := Product.matchingFinal
  routeObservations := 2
  reservationDeliveries := 2
}

end Umpire3.Temporal.System.WorkflowRouting
