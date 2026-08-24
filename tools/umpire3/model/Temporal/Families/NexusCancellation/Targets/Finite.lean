import Temporal.Families.NexusCancellation.System
import Umpire3.FiniteView

namespace Umpire3.Temporal.Targets.NexusCancellationFencing

abbrev SystemState := Umpire3.Temporal.System.NexusCancellationFencing.State
abbrev SystemAction := Umpire3.Temporal.System.NexusCancellationFencing.Action

def stateIdentity : StateIdentity SystemState where
  Code := SystemState
  codeDecidableEq := inferInstance
  encode := id
  encode_injective := fun _ _ equality => equality
  fingerprint := fun state => state.ownerEpoch
  encodedSize := fun _ => 48

def actionName : SystemAction → String
  | .dispatchTask => "dispatch-task"
  | .acceptCancellation => "request-cancellation"
  | .acquireOwnership => "acquire-ownership"
  | .commitCancellation => "commit-cancellation"
  | .returnSuccess => "worker-returns-success"
  | .persistSuccess => "persist-success"

def soundFiniteView : FiniteView
    Umpire3.Temporal.System.NexusCancellationFencing.behavior .smoke where
  executable := Umpire3.Temporal.System.NexusCancellationFencing.executable
  identity := stateIdentity
  actionDecidableEq := inferInstance
  actionName := actionName
  actionName_injective := by
    intro left right
    cases left <;> cases right <;> simp_all [actionName]

def mutatedFiniteView : FiniteView
    Umpire3.Temporal.System.NexusCancellationFencing.mutatedBehavior .smoke where
  executable := Umpire3.Temporal.System.NexusCancellationFencing.mutatedExecutable
  identity := stateIdentity
  actionDecidableEq := inferInstance
  actionName := actionName
  actionName_injective := by
    intro left right
    cases left <;> cases right <;> simp_all [actionName]

def noStaleCompletion (state : SystemState) : Bool :=
  if state.lifecycle = .succeeded then
    decide (state.completionEpoch = some state.ownerEpoch)
  else true

end Umpire3.Temporal.Targets.NexusCancellationFencing
