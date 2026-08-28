import Temporal.System.Nexus.Tests

namespace Temporal.System.Nexus.ImplementationLinkTests

open Umpire

/-!
This focused System test root stays below Feature. The checked correspondence and its forward
witness are exercised from `Temporal.ImplementationLinkTests.Nexus`, the exact composed-test root.
-/

example : Temporal.System.Nexus.target.kernel.steps
    Temporal.System.Nexus.queuedState
    Temporal.System.Nexus.dispatchAction =
    [Temporal.System.Nexus.dispatchedResult] := by
  native_decide

theorem named_system_authority_remains_available_for_correspondence :
    Temporal.System.Nexus.target.kernel.authoritativeInitial
      Temporal.System.Nexus.queuedSetup
      Temporal.System.Nexus.queuedState ∧
    Temporal.System.Nexus.target.kernel.authoritativeStep
      Temporal.System.Nexus.queuedState
      Temporal.System.Nexus.dispatchAction
      Temporal.System.Nexus.dispatchedResult ∧
    Temporal.System.Nexus.target.kernel.authoritativeStep
      Temporal.System.Nexus.runningState
      Temporal.System.Nexus.recordCancellationAction
      Temporal.System.Nexus.cancellationRecordedResult ∧
    Temporal.System.Nexus.target.kernel.authoritativeStep
      Temporal.System.Nexus.runningState
      Temporal.System.Nexus.recordCompletionAction
      Temporal.System.Nexus.completionRecordedResult := by
  exact ⟨Temporal.System.Nexus.target_queued_initial_authoritative,
    Temporal.System.Nexus.target_queued_dispatch_authoritative,
    Temporal.System.Nexus.target_running_cancellation_authoritative,
    Temporal.System.Nexus.target_running_completion_authoritative⟩

end Temporal.System.Nexus.ImplementationLinkTests
