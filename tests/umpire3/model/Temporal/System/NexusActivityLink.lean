import Temporal.Product.NexusActivityLink

namespace Umpire3.Temporal.System.NexusActivityLink

structure State where
  visible : Umpire3.Temporal.Product.NexusActivityLink.State
  operationDeliveries : Nat
  activityDeliveries : Nat
  deriving DecidableEq, Inhabited, Repr

inductive Action where
  | observeOperation (operation : Umpire3.Temporal.Product.NexusActivityLink.OperationID)
      (activity : Option Umpire3.Temporal.Product.NexusActivityLink.ActivityID)
  | observeActivity (activity : Umpire3.Temporal.Product.NexusActivityLink.ActivityID)
      (operation : Option Umpire3.Temporal.Product.NexusActivityLink.OperationID)
  | redeliverOperation (operation : Umpire3.Temporal.Product.NexusActivityLink.OperationID)
  deriving DecidableEq, Inhabited, Repr

def initial : State where
  visible := Umpire3.Temporal.Product.NexusActivityLink.initial
  operationDeliveries := 0
  activityDeliveries := 0

structure TransitionResult (state : State) where
  nextState : State
  productActions : List Umpire3.Temporal.Product.NexusActivityLink.Action
  productRun : Runs Umpire3.Temporal.Product.NexusActivityLink.model
    state.visible productActions nextState.visible

def stutterResult (state nextState : State)
    (sameVisible : nextState.visible = state.visible) : TransitionResult state where
  nextState := nextState
  productActions := []
  productRun := by
    rw [sameVisible]
    exact Runs.nil (model := Umpire3.Temporal.Product.NexusActivityLink.model) state.visible

def liftProduct (state : State) (action : Umpire3.Temporal.Product.NexusActivityLink.Action)
    (operationDelivery : Bool) : List (TransitionResult state) :=
  (Umpire3.Temporal.Product.NexusActivityLink.next state.visible action).attach.map fun successor => {
    nextState := if operationDelivery then
      { state with visible := successor.1, operationDeliveries := state.operationDeliveries + 1 }
    else
      { state with visible := successor.1, activityDeliveries := state.activityDeliveries + 1 }
    productActions := [action]
    productRun := by
      split <;>
        exact Runs.cons
          ((Umpire3.Temporal.Product.NexusActivityLink.executable.next_iff _ _ _).mp successor.2)
          (Runs.nil (model := Umpire3.Temporal.Product.NexusActivityLink.model) successor.1)
  }

def transitions (state : State) : Action → List (TransitionResult state)
  | .observeOperation operation activity =>
      liftProduct state (.observeOperation operation activity) true
  | .observeActivity activity operation =>
      liftProduct state (.observeActivity activity operation) false
  | .redeliverOperation operation =>
      if state.visible.operationObserved operation then
        [stutterResult state { state with operationDeliveries := state.operationDeliveries + 1 } rfl]
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

def activities := Umpire3.Temporal.Product.NexusActivityLink.activities
def operations := Umpire3.Temporal.Product.NexusActivityLink.operations

def actions : List Action :=
  (Umpire3.Temporal.Product.NexusActivityLink.operationIDs.flatMap fun operation =>
    (activities.map fun activity => Action.observeOperation operation activity) ++
      [Action.redeliverOperation operation]) ++
  (Umpire3.Temporal.Product.NexusActivityLink.activityIDs.flatMap fun activity =>
    operations.map fun operation => Action.observeActivity activity operation)

def bounded : BoundedModel system where
  toExecutableModel := executable
  initials := [initial]
  initial_iff := by intro state; simp
  actions := actions
  action_complete := by
    intro state action nextState _
    cases action with
    | observeOperation operation activity =>
        cases operation <;> cases activity with
        | none => simp [actions, activities, Umpire3.Temporal.Product.NexusActivityLink.activities,
            Umpire3.Temporal.Product.NexusActivityLink.operationIDs]
        | some activity =>
            cases activity <;> simp [actions, activities,
              Umpire3.Temporal.Product.NexusActivityLink.activities,
              Umpire3.Temporal.Product.NexusActivityLink.operationIDs]
    | observeActivity activity operation =>
        cases activity <;> cases operation with
        | none => simp [actions, operations, Umpire3.Temporal.Product.NexusActivityLink.operations,
            Umpire3.Temporal.Product.NexusActivityLink.activityIDs]
        | some operation =>
            cases operation <;> simp [actions, operations,
              Umpire3.Temporal.Product.NexusActivityLink.operations,
              Umpire3.Temporal.Product.NexusActivityLink.activityIDs]
    | redeliverOperation operation =>
        cases operation <;> simp [actions, Umpire3.Temporal.Product.NexusActivityLink.operationIDs]

def matchingFinal : State := {
  initial with
  visible := Umpire3.Temporal.Product.NexusActivityLink.matchingFinal
  operationDeliveries := 2
  activityDeliveries := 1
}

end Umpire3.Temporal.System.NexusActivityLink
