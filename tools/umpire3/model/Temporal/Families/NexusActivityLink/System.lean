import Umpire3.ExecutableView

namespace Umpire3.Temporal.System.NexusActivityLink

inductive State where
  | empty
  | operationObserved
  | linked
  | oneSided
  deriving DecidableEq, Inhabited, Repr

inductive Action where
  | observeOperation
  | observeLinkedActivity
  | observeOneSidedActivity
  deriving DecidableEq, Inhabited, Repr

def initial : State := .empty
def actions : List Action := [.observeOperation, .observeLinkedActivity, .observeOneSidedActivity]

def next : State → Action → List State
  | .empty, .observeOperation => [.operationObserved]
  | .operationObserved, .observeLinkedActivity => [.linked]
  | _, _ => []

def mutatedNext : State → Action → List State
  | .operationObserved, .observeOneSidedActivity => [.oneSided]
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

end Umpire3.Temporal.System.NexusActivityLink
