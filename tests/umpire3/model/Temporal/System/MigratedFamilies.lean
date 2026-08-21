import Umpire3.ExecutableView
import Temporal.System.TaskDelivery

namespace Umpire3.Temporal.System.MigratedFamilies

namespace WorkflowOwnership

inductive State where
  | idle
  | currentDispatched
  | currentFailed
  | ownerRotated
  | staleRejected
  | currentCompleted
  | staleCompleted
  deriving DecidableEq, Inhabited, Repr

inductive Action where
  | dispatchCurrent
  | failCurrent
  | rotateOwner
  | rejectStale
  | completeCurrent
  | completeStale
  deriving DecidableEq, Inhabited, Repr

def initial : State := .idle

def actions : List Action := [
  .dispatchCurrent, .failCurrent, .rotateOwner, .rejectStale, .completeCurrent, .completeStale,
]

def next : State → Action → List State
  | .idle, .dispatchCurrent => [.currentDispatched]
  | .currentDispatched, .failCurrent => [.currentFailed]
  | .currentFailed, .rotateOwner => [.ownerRotated]
  | .ownerRotated, .rejectStale => [.staleRejected]
  | .staleRejected, .completeCurrent => [.currentCompleted]
  | _, _ => []

def mutatedNext : State → Action → List State
  | .ownerRotated, .completeStale => [.staleCompleted]
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

def deliveryRequirement : Temporal.System.TaskDelivery.Requirement where
  consumer := "Temporal.System.MigratedFamilies.WorkflowOwnership"
  proof := Temporal.System.TaskDelivery.guarantee.proof

end WorkflowOwnership

namespace WorkflowLineage

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

end WorkflowLineage

namespace WorkflowRouting

inductive State where
  | empty
  | taskRouted
  | matchingPollerRegistered
  | crossingPollerRegistered
  | matchingReservation
  | crossingReservation
  deriving DecidableEq, Inhabited, Repr

inductive Action where
  | assignTask
  | registerMatchingPoller
  | registerCrossingPoller
  | reserveMatching
  | reserveCrossing
  deriving DecidableEq, Inhabited, Repr

def initial : State := .empty
def actions : List Action := [
  .assignTask, .registerMatchingPoller, .registerCrossingPoller, .reserveMatching, .reserveCrossing,
]

def next : State → Action → List State
  | .empty, .assignTask => [.taskRouted]
  | .taskRouted, .registerMatchingPoller => [.matchingPollerRegistered]
  | .taskRouted, .registerCrossingPoller => [.crossingPollerRegistered]
  | .matchingPollerRegistered, .reserveMatching => [.matchingReservation]
  | _, _ => []

def mutatedNext : State → Action → List State
  | .crossingPollerRegistered, .reserveCrossing => [.crossingReservation]
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

end WorkflowRouting

namespace SpeculativeTask

inductive State where
  | empty
  | updatePending
  | taskSpeculative
  | taskCommitted
  | orphanedTask
  deriving DecidableEq, Inhabited, Repr

inductive Action where
  | requestUpdate
  | createTask
  | commitTask
  | createOrphan
  deriving DecidableEq, Inhabited, Repr

def initial : State := .empty
def actions : List Action := [.requestUpdate, .createTask, .commitTask, .createOrphan]

def next : State → Action → List State
  | .empty, .requestUpdate => [.updatePending]
  | .updatePending, .createTask => [.taskSpeculative]
  | .taskSpeculative, .commitTask => [.taskCommitted]
  | _, _ => []

def mutatedNext : State → Action → List State
  | .empty, .createOrphan => [.orphanedTask]
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

end SpeculativeTask

namespace CallbackReference

inductive State where
  | empty
  | attachmentObserved
  | matchingOperationObserved
  | wrongOperationObserved
  deriving DecidableEq, Inhabited, Repr

inductive Action where
  | observeAttachment
  | observeMatchingOperation
  | observeWrongOperation
  deriving DecidableEq, Inhabited, Repr

def initial : State := .empty
def actions : List Action := [.observeAttachment, .observeMatchingOperation, .observeWrongOperation]

def next : State → Action → List State
  | .empty, .observeAttachment => [.attachmentObserved]
  | .attachmentObserved, .observeMatchingOperation => [.matchingOperationObserved]
  | _, _ => []

def mutatedNext : State → Action → List State
  | .attachmentObserved, .observeWrongOperation => [.wrongOperationObserved]
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

end CallbackReference

namespace CallbackResponse

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

end CallbackResponse

namespace NexusTimeout

inductive State where
  | idle
  | configured
  | timedOut
  | malformedTimeout
  deriving DecidableEq, Inhabited, Repr

inductive Action where
  | configure
  | recordTimeout
  | recordMalformedTimeout
  deriving DecidableEq, Inhabited, Repr

def initial : State := .idle
def actions : List Action := [.configure, .recordTimeout, .recordMalformedTimeout]

def next : State → Action → List State
  | .idle, .configure => [.configured]
  | .configured, .recordTimeout => [.timedOut]
  | _, _ => []

def mutatedNext : State → Action → List State
  | .configured, .recordMalformedTimeout => [.malformedTimeout]
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

end NexusTimeout

namespace NexusClosure

inductive State where
  | idle
  | scheduled
  | started
  | settled
  | closed
  | closedWhileRunning
  deriving DecidableEq, Inhabited, Repr

inductive Action where
  | schedule
  | start
  | settle
  | close
  | closeWhileRunning
  deriving DecidableEq, Inhabited, Repr

def initial : State := .idle
def actions : List Action := [.schedule, .start, .settle, .close, .closeWhileRunning]

def next : State → Action → List State
  | .idle, .schedule => [.scheduled]
  | .scheduled, .start => [.started]
  | .started, .settle => [.settled]
  | .settled, .close => [.closed]
  | _, _ => []

def mutatedNext : State → Action → List State
  | .started, .closeWhileRunning => [.closedWhileRunning]
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

end NexusClosure

namespace NexusActivityLink

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

end NexusActivityLink

namespace WorkflowProgress

inductive State where
  | idle
  | queued
  | workerAvailable
  | waited
  | dispatched
  | completed
  | starved
  | wrongEntityCompleted
  deriving DecidableEq, Inhabited, Repr

inductive Action where
  | enqueue
  | makeWorkerAvailable
  | wait
  | dispatch
  | complete
  | waitAgain
  | completeWrongEntity
  deriving DecidableEq, Inhabited, Repr

def initial : State := .idle
def actions : List Action := [
  .enqueue, .makeWorkerAvailable, .wait, .dispatch, .complete, .waitAgain, .completeWrongEntity,
]

def next : State → Action → List State
  | .idle, .enqueue => [.queued]
  | .queued, .makeWorkerAvailable => [.workerAvailable]
  | .workerAvailable, .wait => [.waited]
  | .waited, .dispatch => [.dispatched]
  | .dispatched, .complete => [.completed]
  | _, _ => []

def mutatedNext : State → Action → List State
  | .waited, .waitAgain => [.starved]
  | .dispatched, .completeWrongEntity => [.wrongEntityCompleted]
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

end WorkflowProgress

namespace UpdateLifecycle

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

def deliveryRequirement : Temporal.System.TaskDelivery.Requirement where
  consumer := "Temporal.System.MigratedFamilies.UpdateLifecycle"
  proof := Temporal.System.TaskDelivery.guarantee.proof

end UpdateLifecycle

end Umpire3.Temporal.System.MigratedFamilies
