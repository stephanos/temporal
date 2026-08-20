import Umpire3.Executable
import Umpire3.Property

namespace Umpire3.Temporal.Product.Update

inductive State where
  | idle
  | requested
  | accepted
  | completed
  deriving DecidableEq, Inhabited, Repr

inductive Command where
  | request
  | accept
  | complete
  deriving DecidableEq, Inhabited, Repr

def initial : State := .idle
def requested : State := .requested
def accepted : State := .accepted
def completed : State := .completed

def step : State → Command → State → Prop
  | .idle, .request, .requested => True
  | .requested, .accept, .accepted => True
  | .accepted, .complete, .completed => True
  | _, _, _ => False

abbrev product : TransitionSystem where
  State := State
  Action := Command
  Initial := fun state => state = initial
  Step := step

def next : State → Command → List State
  | .idle, .request => [.requested]
  | .requested, .accept => [.accepted]
  | .accepted, .complete => [.completed]
  | _, _ => []

theorem next_iff (state action nextState) :
    nextState ∈ next state action ↔ product.Step state action nextState := by
  cases state <;> cases action <;> cases nextState <;> simp [next, step]

def executable : ExecutableModel product where
  next := next
  next_iff := next_iff

theorem completed_stable {action nextState} (transition : product.Step completed action nextState) :
    nextState = completed := by
  cases action <;> cases nextState <;> simp [completed, step] at transition ⊢

end Umpire3.Temporal.Product.Update
