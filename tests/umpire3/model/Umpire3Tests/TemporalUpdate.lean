import Temporal.Refinement.UpdateTasks
import Temporal.System.NexusTasks

namespace Umpire3.Temporal.Update.Tests

open Product.Update
open System.UpdateTasks

example : product.Step initial .request requested := by
  simp [Product.Update.step, Product.Update.initial, Product.Update.requested]

example : product.Step requested .accept accepted := by
  simp [Product.Update.step, Product.Update.requested, Product.Update.accepted]

example : product.Step accepted .complete completed := by
  simp [Product.Update.step, Product.Update.accepted, Product.Update.completed]

example : Refinement system product := updateTasksRefinesProduct

example : System.TaskDelivery.Compatible System.TaskDelivery.guarantee
    System.NexusTasks.nexusDeliveryRequirement := by
  simp [System.TaskDelivery.Compatible, System.NexusTasks.nexusDeliveryRequirement]

example : System.TaskDelivery.Compatible System.TaskDelivery.guarantee updateDeliveryRequirement := by
  simp [System.TaskDelivery.Compatible, updateDeliveryRequirement]

end Umpire3.Temporal.Update.Tests
