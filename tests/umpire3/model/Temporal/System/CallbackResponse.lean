import Temporal.Product.CallbackResponse

namespace Umpire3.Temporal.System.CallbackResponse

namespace Product

abbrev State := Umpire3.Temporal.Product.CallbackResponse.State
abbrev Action := Umpire3.Temporal.Product.CallbackResponse.Action
abbrev CallbackID := Umpire3.Temporal.Product.CallbackResponse.CallbackID
abbrev OperationID := Umpire3.Temporal.Product.CallbackResponse.OperationID
abbrev DeliveryID := Umpire3.Temporal.Product.CallbackResponse.DeliveryID
abbrev ResponseKind := Umpire3.Temporal.Product.CallbackResponse.ResponseKind
abbrev Fingerprint := Umpire3.Temporal.Product.CallbackResponse.Fingerprint
abbrev Position := Umpire3.Temporal.Product.CallbackResponse.Position
abbrev model := Umpire3.Temporal.Product.CallbackResponse.model
def initial := Umpire3.Temporal.Product.CallbackResponse.initial
def next := Umpire3.Temporal.Product.CallbackResponse.next
def executable := Umpire3.Temporal.Product.CallbackResponse.executable
def actions := Umpire3.Temporal.Product.CallbackResponse.actions
def deliveryIDs := Umpire3.Temporal.Product.CallbackResponse.deliveryIDs
theorem action_mem (action : Action) : action ∈ actions :=
  Umpire3.Temporal.Product.CallbackResponse.action_mem action
theorem delivery_mem (delivery : DeliveryID) : delivery ∈ deliveryIDs :=
  Umpire3.Temporal.Product.CallbackResponse.delivery_mem delivery
def consistentFinal := Umpire3.Temporal.Product.CallbackResponse.consistentFinal

end Product

structure State where
  visible : Product.State
  registrationDeliveries : Nat
  responseDeliveries : Nat
  deriving DecidableEq, Inhabited, Repr

inductive Action where
  | observeRegistration (delivery : Product.DeliveryID) (callback : Product.CallbackID)
      (operation : Product.OperationID)
  | observeSettlement (operation : Product.OperationID) (position : Product.Position)
      (terminal : Bool)
  | observeResponse (delivery : Product.DeliveryID) (kind : Product.ResponseKind)
      (fingerprint : Product.Fingerprint) (position : Product.Position)
  | redeliverResponse (delivery : Product.DeliveryID)
  deriving DecidableEq, Inhabited, Repr

def initial : State where
  visible := Product.initial
  registrationDeliveries := 0
  responseDeliveries := 0

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
    (registrationDelivery : Bool) : List (TransitionResult state) :=
  (Product.next state.visible action).attach.map fun successor => {
    nextState := if registrationDelivery then
      { state with visible := successor.1, registrationDeliveries := state.registrationDeliveries + 1 }
    else
      { state with visible := successor.1, responseDeliveries := state.responseDeliveries + 1 }
    productActions := [action]
    productRun := by
      split <;>
        exact Runs.cons ((Product.executable.next_iff _ _ _).mp successor.2)
          (Runs.nil (model := Product.model) successor.1)
  }

def transitions (state : State) : Action → List (TransitionResult state)
  | .observeRegistration delivery callback operation =>
      liftProduct state (.register delivery callback operation) true
  | .observeSettlement operation position terminal =>
      liftProduct state (.settleOperation operation position terminal) false
  | .observeResponse delivery kind fingerprint position =>
      liftProduct state (.recordResponse delivery kind fingerprint position) false
  | .redeliverResponse delivery =>
      if (state.visible.acceptedResponse delivery).isSome then
        [stutterResult state { state with responseDeliveries := state.responseDeliveries + 1 } rfl]
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
  | .register delivery callback operation => .observeRegistration delivery callback operation
  | .settleOperation operation position terminal => .observeSettlement operation position terminal
  | .recordResponse delivery kind fingerprint position =>
      .observeResponse delivery kind fingerprint position

def actions : List Action := Product.actions.map fromProduct ++
  Product.deliveryIDs.map Action.redeliverResponse

theorem action_mem (action : Action) : action ∈ actions := by
  cases action with
  | observeRegistration delivery callback operation =>
      apply List.mem_append_left
      exact List.mem_map.mpr ⟨.register delivery callback operation, Product.action_mem _, rfl⟩
  | observeSettlement operation position terminal =>
      apply List.mem_append_left
      exact List.mem_map.mpr ⟨.settleOperation operation position terminal, Product.action_mem _, rfl⟩
  | observeResponse delivery kind fingerprint position =>
      apply List.mem_append_left
      exact List.mem_map.mpr
        ⟨.recordResponse delivery kind fingerprint position, Product.action_mem _, rfl⟩
  | redeliverResponse delivery =>
      apply List.mem_append_right
      exact List.mem_map.mpr ⟨delivery, Product.delivery_mem delivery, rfl⟩

def bounded : BoundedModel system where
  toExecutableModel := executable
  initials := [initial]
  initial_iff := by intro state; simp
  actions := actions
  action_complete := by
    intro state action nextState _
    exact action_mem action

def consistentFinal : State := {
  initial with
  visible := Product.consistentFinal
  registrationDeliveries := 1
  responseDeliveries := 3
}

end Umpire3.Temporal.System.CallbackResponse
