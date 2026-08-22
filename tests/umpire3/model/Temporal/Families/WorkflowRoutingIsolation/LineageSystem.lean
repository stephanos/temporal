import Umpire3.ExecutableView

namespace Umpire3.Temporal.System.WorkflowLineage

inductive State where
  | empty
  | continuationObserved
  | resetObserved
  | invalidContinuation
  deriving DecidableEq, Inhabited, Repr

inductive Action where
  | observeContinuation
  | observeReset
  | observeInvalidContinuation
  deriving DecidableEq, Inhabited, Repr

def initial : State := .empty
def actions : List Action := [.observeContinuation, .observeReset, .observeInvalidContinuation]

def next : State → Action → List State
  | .empty, .observeContinuation => [.continuationObserved]
  | .empty, .observeReset => [.resetObserved]
  | _, _ => []

def mutatedNext : State → Action → List State
  | .empty, .observeInvalidContinuation => [.invalidContinuation]
  | state, action => next state action

def successorsWith (step : State → Action → List State) (state : State) : List (Action × State) :=
  actions.flatMap fun action => (step state action).map fun nextState => (action, nextState)

abbrev behavior : Behavior Unit where
  State := fun _ => State
  Action := fun _ => Action
  Initial := fun _ state => state = initial
  Step := fun _ state action nextState => nextState ∈ next state action

abbrev mutatedBehavior : Behavior Unit where
  State := fun _ => State
  Action := fun _ => Action
  Initial := fun _ state => state = initial
  Step := fun _ state action nextState => nextState ∈ mutatedNext state action

def executableFor (step : State → Action → List State) : ExecutableView ({
    State := fun _ => State
    Action := fun _ => Action
    Initial := fun _ state => state = initial
    Step := fun _ state action nextState => nextState ∈ step state action
  } : Behavior Unit) where
  initials := fun _ => [initial]
  successors := fun _ => successorsWith step
  initials_exact := by intro _ state; exact List.mem_singleton
  successors_exact := by
    intro _ state action nextState
    cases state <;> cases action <;> simp [successorsWith, actions]

def executable : ExecutableView behavior := executableFor next
def mutatedExecutable : ExecutableView mutatedBehavior :=
  executableFor mutatedNext

end Umpire3.Temporal.System.WorkflowLineage
