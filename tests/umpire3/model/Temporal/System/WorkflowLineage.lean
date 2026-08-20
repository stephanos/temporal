import Temporal.Product.WorkflowLineage

namespace Umpire3.Temporal.System.WorkflowLineage

namespace Product

abbrev State := Umpire3.Temporal.Product.WorkflowLineage.State
abbrev Action := Umpire3.Temporal.Product.WorkflowLineage.Action
abbrev RunID := Umpire3.Temporal.Product.WorkflowLineage.RunID
abbrev LineageKind := Umpire3.Temporal.Product.WorkflowLineage.LineageKind
abbrev model := Umpire3.Temporal.Product.WorkflowLineage.model
def initial := Umpire3.Temporal.Product.WorkflowLineage.initial
def next := Umpire3.Temporal.Product.WorkflowLineage.next
def executable := Umpire3.Temporal.Product.WorkflowLineage.executable
def actions := Umpire3.Temporal.Product.WorkflowLineage.actions
def runIDs := Umpire3.Temporal.Product.WorkflowLineage.runIDs
theorem action_mem (action : Action) : action ∈ actions :=
  Umpire3.Temporal.Product.WorkflowLineage.action_mem action
theorem run_mem (run : RunID) : run ∈ runIDs :=
  Umpire3.Temporal.Product.WorkflowLineage.run_mem run
def continuationFinal := Umpire3.Temporal.Product.WorkflowLineage.continuationFinal

end Product

structure State where
  visible : Product.State
  deliveries : Nat
  deriving DecidableEq, Inhabited, Repr

inductive Action where
  | observe (run : Product.RunID) (kind : Product.LineageKind) (predecessor : Product.RunID)
      (original : Product.RunID) (first : Product.RunID)
  | redeliver (run : Product.RunID)
  deriving DecidableEq, Inhabited, Repr

def initial : State where
  visible := Product.initial
  deliveries := 0

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

def liftProduct (state : State) (action : Product.Action) : List (TransitionResult state) :=
  (Product.next state.visible action).attach.map fun successor => {
    nextState := { state with visible := successor.1, deliveries := state.deliveries + 1 }
    productActions := [action]
    productRun := Runs.cons ((Product.executable.next_iff _ _ _).mp successor.2)
      (Runs.nil (model := Product.model) successor.1)
  }

def transitions (state : State) : Action → List (TransitionResult state)
  | .observe run kind predecessor original first =>
      liftProduct state (.observe run kind predecessor original first)
  | .redeliver run =>
      if (state.visible.evidence run).observed then
        [stutterResult state { state with deliveries := state.deliveries + 1 } rfl]
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
  | .observe run kind predecessor original first => .observe run kind predecessor original first

def actions : List Action := Product.actions.map fromProduct ++ Product.runIDs.map Action.redeliver

theorem action_mem (action : Action) : action ∈ actions := by
  cases action with
  | observe run kind predecessor original first =>
      apply List.mem_append_left
      exact List.mem_map.mpr
        ⟨.observe run kind predecessor original first, Product.action_mem _, rfl⟩
  | redeliver run =>
      apply List.mem_append_right
      exact List.mem_map.mpr ⟨run, Product.run_mem run, rfl⟩

def bounded : BoundedModel system where
  toExecutableModel := executable
  initials := [initial]
  initial_iff := by intro state; simp
  actions := actions
  action_complete := by
    intro state action nextState _
    exact action_mem action

def continuationFinal : State := {
  initial with
  visible := Product.continuationFinal
  deliveries := 2
}

end Umpire3.Temporal.System.WorkflowLineage
