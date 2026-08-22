import Umpire3.ExecutableView

namespace Umpire3.Temporal.System.TaskAck

namespace Protocol

inductive State where
  | idle
  | messageQueued
  | deliveryIssued
  | completionStored
  | completionStoredWithBacklog
  deriving DecidableEq, Inhabited, Repr

inductive Action where
  | enqueueMessage
  | issueDelivery
  | storeCompletion
  | storeCompletionWithoutRemovingBacklog
  deriving DecidableEq, Inhabited, Repr

def initial : State := .idle

def actions : List Action := [
  .enqueueMessage, .issueDelivery, .storeCompletion, .storeCompletionWithoutRemovingBacklog,
]

def next : State → Action → List State
  | .idle, .enqueueMessage => [.messageQueued]
  | .messageQueued, .issueDelivery => [.deliveryIssued]
  | .deliveryIssued, .storeCompletion => [.completionStored]
  | _, _ => []

def mutatedNext : State → Action → List State
  | .deliveryIssued, .storeCompletionWithoutRemovingBacklog => [.completionStoredWithBacklog]
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
def mutatedExecutable : ExecutableView mutatedBehavior := executableFor mutatedNext

theorem completionRun :
    Runs (behavior.at ()) .idle [.enqueueMessage, .issueDelivery, .storeCompletion]
      .completionStored := by
  exact Runs.cons (next := State.messageQueued) (by
      change State.messageQueued ∈ next .idle .enqueueMessage
      simp [next])
    (Runs.cons (next := State.deliveryIssued) (by
        change State.deliveryIssued ∈ next .messageQueued .issueDelivery
        simp [next])
      (Runs.cons (by
          change State.completionStored ∈ next .deliveryIssued .storeCompletion
          simp [next]) (Runs.nil _)))

theorem backlogRetentionMutationRun :
    Runs (mutatedBehavior.at ()) .idle
      [.enqueueMessage, .issueDelivery, .storeCompletionWithoutRemovingBacklog]
      .completionStoredWithBacklog := by
  exact Runs.cons (next := State.messageQueued) (by
      change State.messageQueued ∈ mutatedNext .idle .enqueueMessage
      simp [mutatedNext, next])
    (Runs.cons (next := State.deliveryIssued) (by
        change State.deliveryIssued ∈ mutatedNext .messageQueued .issueDelivery
        simp [mutatedNext, next])
      (Runs.cons (by
          change State.completionStoredWithBacklog ∈
            mutatedNext .deliveryIssued .storeCompletionWithoutRemovingBacklog
          simp [mutatedNext]) (Runs.nil _)))

end Protocol

namespace History

inductive State where
  | empty
  | scheduledObserved
  | startedObserved
  | completedObserved
  | completedObservedWithBacklog
  deriving DecidableEq, Inhabited, Repr

inductive Action where
  | observeScheduled
  | observeStarted
  | observeCompleted
  | observeCompletedWithoutRemovingBacklog
  deriving DecidableEq, Inhabited, Repr

def initial : State := .empty

def actions : List Action := [
  .observeScheduled, .observeStarted, .observeCompleted,
  .observeCompletedWithoutRemovingBacklog,
]

def next : State → Action → List State
  | .empty, .observeScheduled => [.scheduledObserved]
  | .scheduledObserved, .observeStarted => [.startedObserved]
  | .startedObserved, .observeCompleted => [.completedObserved]
  | _, _ => []

def mutatedNext : State → Action → List State
  | .startedObserved, .observeCompletedWithoutRemovingBacklog =>
      [.completedObservedWithBacklog]
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
def mutatedExecutable : ExecutableView mutatedBehavior := executableFor mutatedNext

theorem completionRun :
    Runs (behavior.at ()) .empty [.observeScheduled, .observeStarted, .observeCompleted]
      .completedObserved := by
  exact Runs.cons (next := State.scheduledObserved) (by
      change State.scheduledObserved ∈ next .empty .observeScheduled
      simp [next])
    (Runs.cons (next := State.startedObserved) (by
        change State.startedObserved ∈ next .scheduledObserved .observeStarted
        simp [next])
      (Runs.cons (by
          change State.completedObserved ∈ next .startedObserved .observeCompleted
          simp [next]) (Runs.nil _)))

theorem backlogRetentionMutationRun :
    Runs (mutatedBehavior.at ()) .empty
      [.observeScheduled, .observeStarted, .observeCompletedWithoutRemovingBacklog]
      .completedObservedWithBacklog := by
  exact Runs.cons (next := State.scheduledObserved) (by
      change State.scheduledObserved ∈ mutatedNext .empty .observeScheduled
      simp [mutatedNext, next])
    (Runs.cons (next := State.startedObserved) (by
        change State.startedObserved ∈ mutatedNext .scheduledObserved .observeStarted
        simp [mutatedNext, next])
      (Runs.cons (by
          change State.completedObservedWithBacklog ∈
            mutatedNext .startedObserved .observeCompletedWithoutRemovingBacklog
          simp [mutatedNext]) (Runs.nil _)))

end History

structure ExecutableViews where
  protocol : ExecutableView Protocol.behavior
  history : ExecutableView History.behavior

def executableViews : ExecutableViews where
  protocol := Protocol.executable
  history := History.executable

end Umpire3.Temporal.System.TaskAck
