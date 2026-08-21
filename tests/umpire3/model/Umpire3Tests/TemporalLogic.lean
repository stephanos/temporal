import Temporal.System.TaskDeliveryProgress

namespace Umpire3Tests.TemporalLogic

open Umpire3
open Umpire3.Temporal.System.TaskDeliveryProgress

example (execution : InfiniteExecution behavior ()) (fairness : Fairness execution) :
    LeadsTo Unfinished Completed execution.state :=
  progressUnderFairness execution fairness

example : deliveryFairnessIdentifier ≠ recoveryFairnessIdentifier := by decide

end Umpire3Tests.TemporalLogic
