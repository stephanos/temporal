import Temporal.System.NexusTasks
import Temporal.System.TaskDelivery
import Temporal.System.MigratedFamilies

namespace Umpire3.Tests.Registration

example : Umpire3.Temporal.System.TaskDelivery.guarantee.Claim :=
  Umpire3.Temporal.System.NexusTasks.nexusDeliveryRequirement.proof

example : Umpire3.Temporal.System.TaskDelivery.guarantee.Claim :=
  Umpire3.Temporal.System.MigratedFamilies.WorkflowOwnership.deliveryRequirement.proof

theorem weakenedClaim : True := by trivial

def notATheorem : Nat := 1

/--
error: registered declaration must be a theorem
-/
#guard_msgs in
#check resolved_theorem% notATheorem

/--
error: Unknown constant `missingDeclaration`
-/
#guard_msgs in
#check resolved_declaration% missingDeclaration

def weakenedProvider : Umpire3.Guarantee :=
  registered_guarantee% "task-delivery.current-completion-only" weakenedClaim

/--
error: Type mismatch
  Temporal.System.TaskDelivery.guarantee.proof
has type
  Temporal.System.TaskDelivery.guarantee.Claim
but is expected to have type
  weakenedProvider.Claim
-/
#guard_msgs in
def incompatibleConsumer : Umpire3.Requirement weakenedProvider where
  consumer := "Temporal.System.NexusTasks"
  proof := Umpire3.Temporal.System.TaskDelivery.guarantee.proof

end Umpire3.Tests.Registration
