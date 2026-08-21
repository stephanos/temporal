import Umpire3.Executable
import Umpire3.Property

namespace Umpire3.Temporal.Product.NexusProgress

inductive OperationState where
  | idle
  | pending
  | settled
  deriving BEq, DecidableEq, Inhabited, Repr

structure State where
  operation : OperationState
  handlerAvailable : Bool
  retryableFailureObserved : Bool
  waitAge : Nat
  deriving DecidableEq, Inhabited, Repr

def progressB (state : State) : Bool :=
  if state.operation == .pending && state.handlerAvailable then state.waitAge ≤ 1 else true

def progressReadyB (state : State) : Bool :=
  state.retryableFailureObserved && state.operation != .idle

def NexusOperationProgress (state : State) : Prop := progressB state = true

def ProgressQualified (state : State) : Prop :=
  progressReadyB state = true ∧ NexusOperationProgress state

instance (state : State) : Decidable (NexusOperationProgress state) := by
  unfold NexusOperationProgress
  infer_instance

instance (state : State) : Decidable (ProgressQualified state) := by
  unfold ProgressQualified
  infer_instance

inductive Action where
  | schedule
  | observeRetryableFailure
  | wait
  | settle
  deriving DecidableEq, Inhabited, Repr

def initial : State where
  operation := .idle
  handlerAvailable := false
  retryableFailureObserved := false
  waitAge := 0

def rawNext (state : State) : Action → List State
  | .schedule =>
      if state.operation != .idle then [] else [{ state with operation := .pending }]
  | .observeRetryableFailure =>
      if state.operation != .pending || state.retryableFailureObserved then []
      else [{ state with handlerAvailable := true, retryableFailureObserved := true }]
  | .wait =>
      if state.operation != .pending || !state.retryableFailureObserved then []
      else [{ state with waitAge := state.waitAge + 1 }]
  | .settle =>
      if state.operation != .pending then [] else [{ state with operation := .settled }]

def next (state : State) (action : Action) : List State :=
  (rawNext state action).filter progressB

def step (state : State) (action : Action) (nextState : State) : Prop :=
  nextState ∈ next state action

abbrev model : TransitionSystem where
  State := State
  Action := Action
  Initial := (· = initial)
  Step := step

def executable : ExecutableModel model where
  next := next
  next_iff := by intros; rfl

def actions : List Action := [.schedule, .observeRetryableFailure, .wait, .settle]

theorem action_mem (action : Action) : action ∈ actions := by cases action <;> simp [actions]

def bounded : BoundedModel model where
  toExecutableModel := executable
  initials := [initial]
  initial_iff := by intro state; simp
  actions := actions
  action_complete := by
    intro state action nextState _
    exact action_mem action

abbrev weakenedModel : TransitionSystem where
  State := State
  Action := Action
  Initial := (· = initial)
  Step := fun state action nextState => nextState ∈ rawNext state action

def weakenedExecutable : ExecutableModel weakenedModel where
  next := rawNext
  next_iff := by intros; rfl

def scheduled : State := { initial with operation := .pending }

def retrying : State := {
  scheduled with handlerAvailable := true, retryableFailureObserved := true
}

def waited : State := { retrying with waitAge := 1 }

def settledFinal : State := { waited with operation := .settled }

def stuckFinal : State := { waited with waitAge := 2 }

theorem initialProgress : NexusOperationProgress initial := by decide

theorem successorProgress {state action nextState}
    (transition : model.Step state action nextState) : NexusOperationProgress nextState := by
  exact (List.mem_filter.mp transition).2

theorem runsPreserveProgress {start actionHistory final}
    (run : Runs model start actionHistory final)
    (property : NexusOperationProgress start) : NexusOperationProgress final := by
  induction run with
  | nil => exact property
  | cons transition _ induction => exact induction (successorProgress transition)

theorem progressSafe : Safety model NexusOperationProgress := by
  intro state reachable
  rcases reachable with ⟨start, actionHistory, initialState, run⟩
  subst start
  exact runsPreserveProgress run initialProgress

theorem stuckAfterDeadlineMutationNegativeControl : ¬NexusOperationProgress stuckFinal := by decide

end Umpire3.Temporal.Product.NexusProgress
