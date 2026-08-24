import Temporal.Families.NexusCancellation.World
import Umpire3.ExecutableView

namespace Umpire3.Temporal.Feature.NexusCancellationFencing

abbrev World := Umpire3.Temporal.NexusCancellationFencing.World

inductive State where
  | active
  | cancellationAccepted
  | cancelled
  | succeeded
  deriving DecidableEq, Inhabited, Repr

inductive Action where
  | acceptCancellation
  | winCancellation
  | completeSuccess
  deriving DecidableEq, Inhabited, Repr

def initial : State := .active

def successors : State → List (Action × State)
  | .active => [(.acceptCancellation, .cancellationAccepted), (.completeSuccess, .succeeded)]
  | .cancellationAccepted => [(.winCancellation, .cancelled), (.completeSuccess, .succeeded)]
  | .cancelled | .succeeded => []

abbrev behavior : Behavior World where
  State := fun _ => State
  Action := fun _ => Action
  Initial := fun _ state => state = initial
  Step := fun _ state action nextState => (action, nextState) ∈ successors state

def executable : ExecutableView behavior where
  initials := fun _ => [initial]
  successors := fun _ => successors
  initials_exact := by
    intro world state
    cases world
    exact List.mem_singleton
  successors_exact := by
    intro _ _ _ _
    rfl

theorem acceptCancellationStep (world) :
    behavior.Step world .active .acceptCancellation .cancellationAccepted := by
  change (.acceptCancellation, .cancellationAccepted) ∈ successors .active
  decide

theorem winCancellationStep (world) :
    behavior.Step world .cancellationAccepted .winCancellation .cancelled := by
  change (.winCancellation, .cancelled) ∈ successors .cancellationAccepted
  decide

theorem completeActiveStep (world) :
    behavior.Step world .active .completeSuccess .succeeded := by
  change (.completeSuccess, .succeeded) ∈ successors .active
  decide

theorem completeAcceptedStep (world) :
    behavior.Step world .cancellationAccepted .completeSuccess .succeeded := by
  change (.completeSuccess, .succeeded) ∈ successors .cancellationAccepted
  decide

def Terminal : State → Prop
  | .cancelled | .succeeded => True
  | _ => False

theorem terminalStable {world state action nextState}
    (terminal : Terminal state)
    (step : behavior.Step world state action nextState) : False := by
  cases state <;> simp [Terminal, successors] at terminal step

theorem cancelledCannotComplete {world finalState}
    (run : Runs (behavior.at world) .cancelled [.completeSuccess] finalState) : False := by
  rcases run.firstStep with ⟨_, step⟩
  exact terminalStable (world := world) (by trivial) step

def CancellationWon : State → Prop
  | .cancelled => True
  | _ => False

theorem cancellation_won_excludes_success {state} (won : CancellationWon state) :
    state ≠ .succeeded := by
  cases state <;> simp [CancellationWon] at won ⊢

end Umpire3.Temporal.Feature.NexusCancellationFencing
