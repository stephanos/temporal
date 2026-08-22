import Umpire3.ExecutableView
import Temporal.Mechanisms.TaskDelivery

namespace Umpire3.Temporal.System.UpdateLifecycle

inductive State where
  | idle
  | requested
  | taskDispatched
  | accepted
  | historyRecorded
  | workflowTaskCompleted
  | completed
  | completedWithoutHistory
  deriving DecidableEq, Inhabited, Repr

inductive Action where
  | start
  | dispatchTask
  | accept
  | recordHistory
  | completeWorkflowTask
  | complete
  | completeWithoutHistory
  deriving DecidableEq, Inhabited, Repr

def initial : State := .idle
def actions : List Action := [
  .start, .dispatchTask, .accept, .recordHistory, .completeWorkflowTask, .complete,
  .completeWithoutHistory,
]

def next : State → Action → List State
  | .idle, .start => [.requested]
  | .requested, .dispatchTask => [.taskDispatched]
  | .taskDispatched, .accept => [.accepted]
  | .accepted, .recordHistory => [.historyRecorded]
  | .historyRecorded, .completeWorkflowTask => [.workflowTaskCompleted]
  | .workflowTaskCompleted, .complete => [.completed]
  | _, _ => []

def mutatedNext : State → Action → List State
  | .accepted, .completeWithoutHistory => [.completedWithoutHistory]
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

def deliveryRequirement : Temporal.Mechanisms.TaskDelivery.Requirement where
  consumer := "Temporal.System.UpdateLifecycle"
  proof := Temporal.Mechanisms.TaskDelivery.guarantee.proof

end Umpire3.Temporal.System.UpdateLifecycle
