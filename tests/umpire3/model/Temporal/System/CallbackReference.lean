import Temporal.Product.CallbackReference

namespace Umpire3.Temporal.System.CallbackReference

namespace Product

abbrev State := Umpire3.Temporal.Product.CallbackReference.State
abbrev Action := Umpire3.Temporal.Product.CallbackReference.Action
abbrev CallbackID := Umpire3.Temporal.Product.CallbackReference.CallbackID
abbrev OperationID := Umpire3.Temporal.Product.CallbackReference.OperationID
abbrev HandlerID := Umpire3.Temporal.Product.CallbackReference.HandlerID
abbrev ReferenceKind := Umpire3.Temporal.Product.CallbackReference.ReferenceKind
abbrev ReferenceValue := Umpire3.Temporal.Product.CallbackReference.ReferenceValue
abbrev Position := Umpire3.Temporal.Product.CallbackReference.Position
abbrev model := Umpire3.Temporal.Product.CallbackReference.model
def initial := Umpire3.Temporal.Product.CallbackReference.initial
def next := Umpire3.Temporal.Product.CallbackReference.next
def executable := Umpire3.Temporal.Product.CallbackReference.executable
def actions := Umpire3.Temporal.Product.CallbackReference.actions
def callbackIDs := Umpire3.Temporal.Product.CallbackReference.callbackIDs
theorem action_mem (action : Action) : action ∈ actions :=
  Umpire3.Temporal.Product.CallbackReference.action_mem action
theorem callback_mem (callback : CallbackID) : callback ∈ callbackIDs :=
  Umpire3.Temporal.Product.CallbackReference.callback_mem callback
def matchingFinal := Umpire3.Temporal.Product.CallbackReference.matchingFinal

end Product

structure State where
  visible : Product.State
  attachmentDeliveries : Nat
  operationDeliveries : Nat
  deriving DecidableEq, Inhabited, Repr

inductive Action where
  | observeAttachment (callback : Product.CallbackID) (handler : Product.HandlerID)
      (kind : Product.ReferenceKind) (value : Product.ReferenceValue)
      (position : Product.Position) (malformed : Bool)
  | observeOperationStart (callback : Product.CallbackID) (operation : Product.OperationID)
      (handler : Product.HandlerID) (kind : Product.ReferenceKind) (value : Product.ReferenceValue)
      (position : Product.Position) (malformed : Bool)
  | redeliverAttachment (callback : Product.CallbackID)
  deriving DecidableEq, Inhabited, Repr

def initial : State where
  visible := Product.initial
  attachmentDeliveries := 0
  operationDeliveries := 0

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
    (attachmentDelivery : Bool) : List (TransitionResult state) :=
  (Product.next state.visible action).attach.map fun successor => {
    nextState := if attachmentDelivery then
      { state with visible := successor.1, attachmentDeliveries := state.attachmentDeliveries + 1 }
    else
      { state with visible := successor.1, operationDeliveries := state.operationDeliveries + 1 }
    productActions := [action]
    productRun := by
      split <;>
        exact Runs.cons ((Product.executable.next_iff _ _ _).mp successor.2)
          (Runs.nil (model := Product.model) successor.1)
  }

def transitions (state : State) : Action → List (TransitionResult state)
  | .observeAttachment callback handler kind value position malformed =>
      liftProduct state (.observeAttachment callback handler kind value position malformed) true
  | .observeOperationStart callback operation handler kind value position malformed =>
      liftProduct state
        (.observeOperationStart callback operation handler kind value position malformed) false
  | .redeliverAttachment callback =>
      if state.visible.attachmentObserved callback then
        [stutterResult state { state with attachmentDeliveries := state.attachmentDeliveries + 1 } rfl]
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
  | .observeAttachment callback handler kind value position malformed =>
      .observeAttachment callback handler kind value position malformed
  | .observeOperationStart callback operation handler kind value position malformed =>
      .observeOperationStart callback operation handler kind value position malformed

def actions : List Action := Product.actions.map fromProduct ++
  Product.callbackIDs.map Action.redeliverAttachment

theorem action_mem (action : Action) : action ∈ actions := by
  cases action with
  | observeAttachment callback handler kind value position malformed =>
      apply List.mem_append_left
      exact List.mem_map.mpr ⟨.observeAttachment callback handler kind value position malformed,
        Product.action_mem _, rfl⟩
  | observeOperationStart callback operation handler kind value position malformed =>
      apply List.mem_append_left
      exact List.mem_map.mpr
        ⟨.observeOperationStart callback operation handler kind value position malformed,
          Product.action_mem _, rfl⟩
  | redeliverAttachment callback =>
      apply List.mem_append_right
      exact List.mem_map.mpr ⟨callback, Product.callback_mem callback, rfl⟩

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
  attachmentDeliveries := 2
  operationDeliveries := 1
}

end Umpire3.Temporal.System.CallbackReference
