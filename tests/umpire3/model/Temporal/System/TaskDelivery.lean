namespace Umpire3.Temporal.System.TaskDelivery

abbrev CurrentCompletion (completionEpoch : Option Nat) (ownerEpoch : Nat) : Prop :=
  completionEpoch = some ownerEpoch

theorem stale_completion_is_not_current {completionEpoch ownerEpoch : Nat}
    (stale : completionEpoch ≠ ownerEpoch) :
    ¬CurrentCompletion (some completionEpoch) ownerEpoch := by
  simp [CurrentCompletion, stale]

structure Guarantee where
  identifier : String
  statementHash : String
  deriving DecidableEq, Repr

structure Requirement where
  provider : String
  statementHash : String
  deriving DecidableEq, Repr

def guarantee : Guarantee where
  identifier := "task-delivery.current-completion-only"
  statementHash := "sha256:31e3bc0f50ed17ad15f1848d8fd3cc753e18c21729d606f10fc10d5b71d9bc93"

def Compatible (provider : Guarantee) (consumer : Requirement) : Prop :=
  consumer.provider = provider.identifier ∧ consumer.statementHash = provider.statementHash

end Umpire3.Temporal.System.TaskDelivery
