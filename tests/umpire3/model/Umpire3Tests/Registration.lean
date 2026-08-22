import Temporal.Mechanisms.TaskDelivery
import Temporal.Families

namespace Umpire3.Tests.Registration

example : Umpire3.Temporal.Mechanisms.TaskDelivery.guarantee.Claim :=
  Umpire3.Temporal.System.UpdateLifecycle.deliveryRequirement.proof

example : Umpire3.Temporal.Mechanisms.TaskDelivery.guarantee.Claim :=
  Umpire3.Temporal.System.WorkflowOwnership.deliveryRequirement.proof

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
  Temporal.Mechanisms.TaskDelivery.guarantee.proof
has type
  Temporal.Mechanisms.TaskDelivery.guarantee.Claim
but is expected to have type
  weakenedProvider.Claim
-/
#guard_msgs in
def incompatibleConsumer : Umpire3.Requirement weakenedProvider where
  consumer := "Temporal.System.UpdateLifecycle"
  proof := Umpire3.Temporal.Mechanisms.TaskDelivery.guarantee.proof

end Umpire3.Tests.Registration
