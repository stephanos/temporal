import Umpire3.Executable
import Umpire3.Property

namespace Umpire3.Temporal.Product.Nexus

inductive State where
  | active
  | cancellationAccepted
  | succeeded
  | cancelled
  deriving DecidableEq, Inhabited, Repr

inductive Command where
  | acceptCancellation
  | winCancellation
  | completeSuccess
  deriving DecidableEq, Inhabited, Repr

def initial : State := .active

def cancellationAccepted : State := .cancellationAccepted

def succeeded : State := .succeeded

def cancelled : State := .cancelled

def step : State → Command → State → Prop
  | .active, .acceptCancellation, .cancellationAccepted => True
  | .active, .completeSuccess, .succeeded => True
  | .cancellationAccepted, .completeSuccess, .succeeded => True
  | .cancellationAccepted, .winCancellation, .cancelled => True
  | _, _, _ => False

abbrev product : TransitionSystem where
  State := State
  Action := Command
  Initial := fun state => state = initial
  Step := step

def Terminal : State → Prop
  | .succeeded | .cancelled => True
  | _ => False

def CancellationAccepted : State → Prop
  | .cancellationAccepted | .cancelled => True
  | _ => False

def CancellationWon : State → Prop
  | .cancelled => True
  | _ => False

def next : State → Command → List State
  | .active, .acceptCancellation => [.cancellationAccepted]
  | .active, .completeSuccess => [.succeeded]
  | .cancellationAccepted, .completeSuccess => [.succeeded]
  | .cancellationAccepted, .winCancellation => [.cancelled]
  | _, _ => []

theorem next_iff (state action nextState) :
    nextState ∈ next state action ↔ product.Step state action nextState := by
  cases state <;> cases action <;> cases nextState <;> simp [next, step]

def executable : ExecutableModel product where
  next := next
  next_iff := next_iff

def bounded : BoundedModel product where
  toExecutableModel := executable
  initials := [initial]
  initial_iff := by
    intro state
    cases state <;> decide
  actions := [.acceptCancellation, .winCancellation, .completeSuccess]
  action_complete := by
    intro state action nextState _
    cases action <;> simp

theorem terminal_stable {state action nextState}
    (terminal : Terminal state) (transition : product.Step state action nextState) :
    state = nextState := by
  cases state <;> cases action <;> cases nextState <;> simp [Terminal, step] at terminal transition ⊢

theorem cancellation_won_is_terminal {state} (won : CancellationWon state) : Terminal state := by
  cases state <;> simp [CancellationWon, Terminal] at won ⊢

theorem cancellation_won_excludes_success {state} (won : CancellationWon state) :
    state ≠ succeeded := by
  cases state <;> simp [CancellationWon, succeeded] at won ⊢

end Umpire3.Temporal.Product.Nexus
