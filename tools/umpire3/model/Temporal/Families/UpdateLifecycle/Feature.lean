import Umpire3.ExecutableView
import Umpire3.Execution
import Umpire3.Property

namespace Umpire3.Temporal.Feature.UpdateLifecycle

inductive State where
  | idle
  | requested
  | accepted
  | historyRecorded
  | completed
  | completedWithoutHistory
  deriving DecidableEq, Inhabited, Repr

inductive Action where
  | request
  | accept
  | recordHistory
  | complete
  | completeWithoutHistory
  deriving DecidableEq, Inhabited, Repr

def initial : State := .idle
def actions : List Action := [.request, .accept, .recordHistory, .complete, .completeWithoutHistory]

def next : State → Action → List State
  | .idle, .request => [.requested]
  | .requested, .accept => [.accepted]
  | .accepted, .recordHistory => [.historyRecorded]
  | .historyRecorded, .complete => [.completed]
  | _, _ => []

def successors (state : State) : List (Action × State) :=
  actions.flatMap fun action => (next state action).map fun nextState => (action, nextState)

abbrev behavior : Behavior Unit where
  State := fun _ => State
  Action := fun _ => Action
  Initial := fun _ state => state = initial
  Step := fun _ state action nextState => nextState ∈ next state action

def executable : ExecutableView behavior where
  initials := fun _ => [initial]
  successors := fun _ => successors
  initials_exact := by intro _ state; exact List.mem_singleton
  successors_exact := by
    intro _ state action nextState
    cases state <;> cases action <;> simp [successors, actions, next]

def HistoryBacked : State → Prop
  | .completedWithoutHistory => False
  | _ => True

theorem initialHistoryBacked : HistoryBacked initial := by trivial

theorem stepPreservesHistoryBacked {state action nextState}
    (_ : HistoryBacked state) (step : behavior.Step () state action nextState) :
    HistoryBacked nextState := by
  cases state <;> cases action <;> simp [next] at step
  all_goals subst nextState <;> trivial

theorem runsPreserveHistoryBacked {start actions final}
    (run : Runs (behavior.at ()) start actions final) (safe : HistoryBacked start) :
    HistoryBacked final := by
  induction run with
  | nil => exact safe
  | cons step _ induction => exact induction (stepPreservesHistoryBacked safe step)

theorem historyBackedSafe :
    ∀ state, Behavior.Reachable behavior () state → HistoryBacked state := by
  intro state reachable
  rcases reachable with ⟨start, actionHistory, initialState, run⟩
  subst start
  exact runsPreserveHistoryBacked run initialHistoryBacked

theorem requestStep : behavior.Step () .idle .request .requested := by
  change State.requested ∈ next .idle .request
  decide

theorem acceptStep : behavior.Step () .requested .accept .accepted := by
  change State.accepted ∈ next .requested .accept
  decide

theorem historyStep : behavior.Step () .accepted .recordHistory .historyRecorded := by
  change State.historyRecorded ∈ next .accepted .recordHistory
  decide

theorem completeStep : behavior.Step () .historyRecorded .complete .completed := by
  change State.completed ∈ next .historyRecorded .complete
  decide

theorem completedReachable : Behavior.Reachable behavior () .completed := by
  refine ⟨initial, [.request, .accept, .recordHistory, .complete], rfl, ?_⟩
  apply Runs.cons (next := State.requested) requestStep
  apply Runs.cons (next := State.accepted) acceptStep
  apply Runs.cons (next := State.historyRecorded) historyStep
  apply Runs.cons (next := State.completed) completeStep
  exact Runs.nil _

theorem completionWithoutHistoryMutationNegativeControl :
    ¬HistoryBacked .completedWithoutHistory := by simp [HistoryBacked]

end Umpire3.Temporal.Feature.UpdateLifecycle
