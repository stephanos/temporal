import Umpire3.Registration

namespace Umpire3.Temporal.System.TaskDelivery

abbrev CurrentCompletionOf {Epoch : Type} (completionEpoch : Option Epoch) (ownerEpoch : Epoch) : Prop :=
  completionEpoch = some ownerEpoch

abbrev CurrentCompletion (completionEpoch : Option Nat) (ownerEpoch : Nat) : Prop :=
  CurrentCompletionOf completionEpoch ownerEpoch

theorem stale_completion_is_not_current {Epoch : Type} {completionEpoch ownerEpoch : Epoch}
    (stale : completionEpoch ≠ ownerEpoch) :
    ¬CurrentCompletionOf (some completionEpoch) ownerEpoch := by
  simp [CurrentCompletionOf, stale]

def CurrentCompletionOnly : Prop :=
  ∀ (Epoch : Type) (completionEpoch ownerEpoch : Epoch), completionEpoch ≠ ownerEpoch →
    ¬CurrentCompletionOf (some completionEpoch) ownerEpoch

theorem current_completion_only : CurrentCompletionOnly := by
  intro Epoch completionEpoch ownerEpoch stale
  exact stale_completion_is_not_current stale

def guarantee : Umpire3.Guarantee :=
  registered_guarantee% "task-delivery.current-completion-only" current_completion_only

abbrev Requirement := Umpire3.Requirement guarantee

end Umpire3.Temporal.System.TaskDelivery
