import Lentil.Basic
import Temporal.System.TaskDeliveryProgress

namespace Umpire3.TemporalAdapters.Lentil

def LeadsTo {State : Type} (trigger goal : State → Prop) : TLA.pred State :=
  TLA.leads_to (TLA.state_pred trigger) (TLA.state_pred goal)

theorem leadsTo_iff {State : Type} (trigger goal : State → Prop) (states : Nat → State) :
    Umpire3.LeadsTo trigger goal states ↔ LeadsTo trigger goal states 0 := by
  simp only [Umpire3.LeadsTo, Umpire3.EventuallyFrom, LeadsTo, TLA.leads_to,
    TLA.always, TLA.tla_implies, TLA.eventually, TLA.state_pred, Nat.zero_add]
  constructor
  · intro progresses offset triggered
    rcases progresses offset triggered with ⟨index, afterOffset, reached⟩
    exact ⟨index - offset, by simpa [Nat.add_sub_of_le afterOffset] using reached⟩
  · intro progresses start triggered
    rcases progresses start triggered with ⟨offset, reached⟩
    exact ⟨start + offset, Nat.le_add_right start offset, reached⟩

open Umpire3.Temporal.System.TaskDeliveryProgress in
theorem progressUnderFairnessLentil (execution : InfiniteExecution behavior ())
    (fairness : Fairness execution) :
    LeadsTo Unfinished Completed execution.state 0 :=
  (leadsTo_iff Unfinished Completed execution.state).mp
    (progressUnderFairness execution fairness)

end Umpire3.TemporalAdapters.Lentil
