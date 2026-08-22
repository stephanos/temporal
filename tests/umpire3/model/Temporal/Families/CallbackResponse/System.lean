import Umpire3.ExecutableView

namespace Umpire3.Temporal.System.CallbackResponse

inductive State where
  | empty
  | registered
  | settled
  | responded
  | conflictingResponse
  deriving DecidableEq, Inhabited, Repr

inductive Action where
  | register
  | settle
  | respond
  | conflict
  deriving DecidableEq, Inhabited, Repr

def initial : State := .empty
def actions : List Action := [.register, .settle, .respond, .conflict]

def next : State → Action → List State
  | .empty, .register => [.registered]
  | .registered, .settle => [.settled]
  | .settled, .respond => [.responded]
  | _, _ => []

def mutatedNext : State → Action → List State
  | .responded, .conflict => [.conflictingResponse]
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

end Umpire3.Temporal.System.CallbackResponse
