import Temporal.Product.NexusTimeout

namespace Umpire3.Temporal.System.NexusTimeout

structure State where
  visible : Umpire3.Temporal.Product.NexusTimeout.State
  sourceSequence : Nat
  duplicateDeliveries : Nat
  deriving DecidableEq, Inhabited, Repr

inductive Action where
  | observeConfiguration (operation : Umpire3.Temporal.Product.NexusTimeout.OperationID)
  | observeTimeout (operation : Umpire3.Temporal.Product.NexusTimeout.OperationID)
      (evidence : Umpire3.Temporal.Product.NexusTimeout.EvidenceID)
      (kind : Umpire3.Temporal.Product.NexusTimeout.TimeoutKind)
      (message : Umpire3.Temporal.Product.NexusTimeout.TimeoutMessage)
  | redeliver
  deriving DecidableEq, Inhabited, Repr

def initial : State where
  visible := Umpire3.Temporal.Product.NexusTimeout.initial
  sourceSequence := 0
  duplicateDeliveries := 0

structure TransitionResult (state : State) where
  nextState : State
  productActions : List Umpire3.Temporal.Product.NexusTimeout.Action
  productRun : Runs Umpire3.Temporal.Product.NexusTimeout.model
    state.visible productActions nextState.visible

def stutterResult (state nextState : State)
    (sameVisible : nextState.visible = state.visible) : TransitionResult state where
  nextState := nextState
  productActions := []
  productRun := by
    rw [sameVisible]
    exact Runs.nil (model := Umpire3.Temporal.Product.NexusTimeout.model) state.visible

def liftProduct (state : State) (action : Umpire3.Temporal.Product.NexusTimeout.Action) :
    List (TransitionResult state) :=
  (Umpire3.Temporal.Product.NexusTimeout.next state.visible action).attach.map fun successor => {
    nextState := { state with visible := successor.1, sourceSequence := state.sourceSequence + 1 }
    productActions := [action]
    productRun := Runs.cons
      ((Umpire3.Temporal.Product.NexusTimeout.executable.next_iff _ _ _).mp successor.2)
      (Runs.nil (model := Umpire3.Temporal.Product.NexusTimeout.model) successor.1)
  }

def transitions (state : State) : Action → List (TransitionResult state)
  | .observeConfiguration operation =>
      liftProduct state (.configure operation)
  | .observeTimeout operation evidence kind message =>
      liftProduct state (.recordTimeout operation evidence kind message)
  | .redeliver =>
      [stutterResult state { state with duplicateDeliveries := state.duplicateDeliveries + 1 } rfl]

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

def actions : List Action :=
  [.observeConfiguration .primary, .observeConfiguration .secondary, .redeliver] ++
    [Umpire3.Temporal.Product.NexusTimeout.OperationID.primary,
      Umpire3.Temporal.Product.NexusTimeout.OperationID.secondary].flatMap fun operation =>
      [Umpire3.Temporal.Product.NexusTimeout.EvidenceID.primary,
        Umpire3.Temporal.Product.NexusTimeout.EvidenceID.secondary].flatMap fun evidence =>
        [Umpire3.Temporal.Product.NexusTimeout.TimeoutKind.startToClose,
          Umpire3.Temporal.Product.NexusTimeout.TimeoutKind.scheduleToClose,
          Umpire3.Temporal.Product.NexusTimeout.TimeoutKind.unspecified].flatMap fun kind =>
          [Umpire3.Temporal.Product.NexusTimeout.TimeoutMessage.operationTimedOut,
            Umpire3.Temporal.Product.NexusTimeout.TimeoutMessage.unrelatedFailure].map fun message =>
            .observeTimeout operation evidence kind message

def bounded : BoundedModel system where
  toExecutableModel := executable
  initials := [initial]
  initial_iff := by intro state; simp
  actions := actions
  action_complete := by
    intro state action nextState _
    cases action with
    | observeConfiguration operation => cases operation <;> simp [actions]
    | observeTimeout operation evidence kind message =>
        cases operation <;> cases evidence <;> cases kind <;> cases message <;> simp [actions]
    | redeliver => simp [actions]

def TimeoutSemantics (state : State) : Prop :=
  Umpire3.Temporal.Product.NexusTimeout.TimeoutSemantics state.visible

instance (state : State) : Decidable (TimeoutSemantics state) :=
  Umpire3.Temporal.Product.NexusTimeout.instDecidableTimeoutSemantics state.visible

theorem stepPreservesTimeoutSemantics {state action nextState}
    (property : TimeoutSemantics state) (transition : system.Step state action nextState) :
    TimeoutSemantics nextState := by
  rcases transition with ⟨result, _, rfl⟩
  exact Umpire3.Temporal.Product.NexusTimeout.runsPreserveTimeoutSemantics
    result.productRun property

def permittedFinal : State := {
  initial with
  visible := Umpire3.Temporal.Product.NexusTimeout.permittedFinal
  sourceSequence := 2
}

end Umpire3.Temporal.System.NexusTimeout
