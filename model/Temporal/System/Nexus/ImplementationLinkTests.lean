import Temporal.System.Nexus.ImplementationLink
import Temporal.System.Nexus.Tests

namespace Temporal.System.Nexus.ImplementationLinkTests

open Umpire
open Temporal.System.Nexus.ImplementationLink

example : checkedResult.isOk = true ∧ checked.hasCanonicalIdentity = true := by
  native_decide

example : checked.sourceTarget.id = Temporal.System.Nexus.targetId ∧
    checked.destinationTarget.id = Temporal.Feature.Nexus.Lifecycle.targetId ∧
    checked.declaration.capabilityMappings = [lifecycleCapabilityMapping] ∧
    checked.declaration.relationMappings = [] := by
  native_decide

example : Temporal.Feature.Nexus.Lifecycle.target.kernel.authoritativeInitial
    Temporal.Feature.Nexus.Lifecycle.scheduledSetup
    Temporal.Feature.Nexus.Lifecycle.scheduledState := by
  simpa [witness] using witness.initialForward Temporal.System.Nexus.queuedSetup
    Temporal.System.Nexus.queuedState
    Temporal.System.Nexus.target_queued_initial_authoritative

example : Temporal.Feature.Nexus.Lifecycle.target.kernel.authoritativeStep
    Temporal.Feature.Nexus.Lifecycle.startedState
    Temporal.Feature.Nexus.Lifecycle.cancelAction
    Temporal.Feature.Nexus.Lifecycle.canceledResult := by
  simpa [witness, Temporal.System.Nexus.cancellationRecordedResult,
    Temporal.Feature.Nexus.Lifecycle.canceledResult] using
    witness.stepForward Temporal.System.Nexus.runningState
    Temporal.System.Nexus.recordCancellationAction
    Temporal.System.Nexus.cancellationRecordedResult
    Temporal.System.Nexus.target_running_cancellation_authoritative

end Temporal.System.Nexus.ImplementationLinkTests
