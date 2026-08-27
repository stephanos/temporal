import Temporal.System.Nexus.Core

namespace Temporal.System.Nexus.Tests

open Umpire
open Temporal.System.Nexus

example : step .queued .dispatch = some .running ∧
    step .running .recordCancellation = some .cancellationRecorded ∧
    step .running .recordCompletion = some .completionRecorded := by
  exact ⟨rfl, rfl, rfl⟩

example : (checkTarget targetAuthoring).isOk = true ∧
    target.requiredCapabilities = [lifecycleCapabilityId] ∧
    target.providers.map CapabilityProvider.id = [lifecycleProviderId] ∧
    target.connectors = [] := by
  native_decide

example : target.kernel.initialStates queuedSetup = [queuedState] ∧
    target.kernel.initialStates runningSetup = [runningState] ∧
    target.kernel.steps queuedState dispatchAction = [dispatchedResult] ∧
    target.kernel.steps runningState recordCancellationAction = [cancellationRecordedResult] ∧
    target.kernel.steps runningState recordCompletionAction = [completionRecordedResult] := by
  native_decide

end Temporal.System.Nexus.Tests
