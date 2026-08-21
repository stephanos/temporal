import Temporal.NexusCancellationFencing.World
import Umpire3.ExecutableView

namespace Umpire3.Temporal.System.NexusCancellationFencing

abbrev World := Umpire3.Temporal.NexusCancellationFencing.World

inductive Lifecycle where
  | open
  | cancellationAccepted
  | cancelled
  | succeeded
  deriving DecidableEq, Inhabited, Repr

inductive TaskStage where
  | idle
  | dispatched
  | returned
  deriving DecidableEq, Inhabited, Repr

structure State where
  lifecycle : Lifecycle
  task : TaskStage
  ownerEpoch : Nat
  workerEpoch : Option Nat
  completionEpoch : Option Nat
  deriving DecidableEq, Inhabited, Repr

inductive Action where
  | dispatchTask
  | acceptCancellation
  | acquireOwnership
  | commitCancellation
  | returnSuccess
  | persistSuccess
  deriving DecidableEq, Inhabited, Repr

def initial : State where
  lifecycle := .open
  task := .idle
  ownerEpoch := 0
  workerEpoch := none
  completionEpoch := none

def actions : List Action := [
  .dispatchTask,
  .acceptCancellation,
  .acquireOwnership,
  .commitCancellation,
  .returnSuccess,
  .persistSuccess,
]

def next (world : World) (state : State) : Action → List State
  | .dispatchTask =>
      if state.task = .idle ∧ state.lifecycle = .open then
        [{ state with task := .dispatched, workerEpoch := some state.ownerEpoch }]
      else []
  | .acceptCancellation =>
      if state.lifecycle = .open then
        [{ state with lifecycle := .cancellationAccepted }]
      else []
  | .acquireOwnership =>
      if (state.lifecycle = .open ∨ state.lifecycle = .cancellationAccepted ∨
          state.lifecycle = .cancelled) ∧
          state.ownerEpoch < world.maxOwnerEpoch then
        [{ state with ownerEpoch := state.ownerEpoch + 1 }]
      else []
  | .commitCancellation =>
      if state.lifecycle = .cancellationAccepted then
        [{ state with lifecycle := .cancelled }]
      else []
  | .returnSuccess =>
      if state.task = .dispatched then
        [{ state with task := .returned, completionEpoch := state.workerEpoch }]
      else []
  | .persistSuccess =>
      if state.task = .returned ∧ state.completionEpoch = some state.ownerEpoch ∧
          (state.lifecycle = .open ∨ state.lifecycle = .cancellationAccepted) then
        [{ state with lifecycle := .succeeded }]
      else []

def successors (world : World) (state : State) : List (Action × State) :=
  actions.flatMap fun action => (next world state action).map fun nextState => (action, nextState)

abbrev behavior : Behavior World where
  State := fun _ => State
  Action := fun _ => Action
  Initial := fun _ state => state = initial
  Step := fun world state action nextState => nextState ∈ next world state action

theorem successors_iff (world state action nextState) :
    (action, nextState) ∈ successors world state ↔ nextState ∈ next world state action := by
  cases action <;> simp [successors, actions]

def executable : ExecutableView behavior where
  initials := fun _ => [initial]
  successors := successors
  initials_exact := by
    intro world state
    cases world
    exact List.mem_singleton
  successors_exact := by
    intro world state action nextState
    exact successors_iff world state action nextState

def mutatedNext (world : World) (state : State) : Action → List State
  | .persistSuccess =>
      if state.task = .returned ∧ state.completionEpoch ≠ none then
        [{ state with lifecycle := .succeeded }]
      else []
  | action => next world state action

def mutatedSuccessors (world : World) (state : State) : List (Action × State) :=
  actions.flatMap fun action =>
    (mutatedNext world state action).map fun nextState => (action, nextState)

abbrev mutatedBehavior : Behavior World where
  State := fun _ => State
  Action := fun _ => Action
  Initial := fun _ state => state = initial
  Step := fun world state action nextState => nextState ∈ mutatedNext world state action

theorem mutatedSuccessors_iff (world state action nextState) :
    (action, nextState) ∈ mutatedSuccessors world state ↔
      nextState ∈ mutatedNext world state action := by
  cases action <;> simp [mutatedSuccessors, actions]

def mutatedExecutable : ExecutableView mutatedBehavior where
  initials := fun _ => [initial]
  successors := mutatedSuccessors
  initials_exact := by
    intro world state
    cases world
    exact List.mem_singleton
  successors_exact := by
    intro world state action nextState
    exact mutatedSuccessors_iff world state action nextState

def afterDispatch : State :=
  { initial with task := .dispatched, workerEpoch := some 0 }

def afterCancellationAccepted : State :=
  { afterDispatch with lifecycle := .cancellationAccepted }

def afterOwnershipChange : State :=
  { afterCancellationAccepted with ownerEpoch := 1 }

def afterCancellationCommit : State :=
  { afterOwnershipChange with lifecycle := .cancelled }

def staleReturned : State :=
  { afterCancellationCommit with task := .returned, completionEpoch := some 0 }

def staleSuccess : State :=
  { staleReturned with lifecycle := .succeeded }

def mutatedCounterexampleActions : List Action := [
  .dispatchTask,
  .acceptCancellation,
  .acquireOwnership,
  .commitCancellation,
  .returnSuccess,
  .persistSuccess,
]

theorem mutatedCounterexample : Runs (mutatedBehavior.at .smoke)
    initial mutatedCounterexampleActions staleSuccess := by
  apply Runs.cons (next := afterDispatch) (by
    simp [Behavior.at, mutatedBehavior, mutatedNext, next, initial, afterDispatch])
  apply Runs.cons (next := afterCancellationAccepted) (by
    simp [Behavior.at, mutatedBehavior, mutatedNext, next, afterDispatch,
      afterCancellationAccepted, initial])
  apply Runs.cons (next := afterOwnershipChange) (by
    simp [Behavior.at, mutatedBehavior, mutatedNext, next, afterCancellationAccepted,
      afterOwnershipChange, afterDispatch, initial,
      Umpire3.Temporal.NexusCancellationFencing.World.maxOwnerEpoch])
  apply Runs.cons (next := afterCancellationCommit) (by
    simp [Behavior.at, mutatedBehavior, mutatedNext, next, afterOwnershipChange,
      afterCancellationCommit, afterCancellationAccepted, afterDispatch])
  apply Runs.cons (next := staleReturned) (by
    simp [Behavior.at, mutatedBehavior, mutatedNext, next, afterCancellationCommit,
      staleReturned, afterOwnershipChange, afterCancellationAccepted, afterDispatch])
  apply Runs.cons (next := staleSuccess) (by
    simp [Behavior.at, mutatedBehavior, mutatedNext, staleReturned, staleSuccess,
      afterCancellationCommit, afterOwnershipChange, afterCancellationAccepted, afterDispatch])
  exact Runs.nil (model := mutatedBehavior.at .smoke) staleSuccess

end Umpire3.Temporal.System.NexusCancellationFencing
