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

example : System.TaskDelivery.guarantee.Claim :=
  System.NexusTasks.nexusDeliveryRequirement.proof

example : System.TaskDelivery.guarantee.Claim := updateDeliveryRequirement.proof

end Umpire3.Temporal.Update.Tests
