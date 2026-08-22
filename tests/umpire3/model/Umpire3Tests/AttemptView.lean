import Temporal.Families.NexusCancellation.Targets.Attempt

namespace Umpire3.Tests.AttemptView

open Umpire3.Temporal.System.NexusCancellationFencing
open Umpire3.Temporal.Targets.NexusCancellationFencing

example : soundAttemptArtifact.validFor soundFirstOrderArtifact := by decide

example : mutatedAttemptArtifact.validFor mutatedFirstOrderArtifact := by decide

example : soundAttemptArtifact.apply soundFirstOrderArtifact (encodeState staleReturned)
    "persist-success" .applied = none := by decide

example : soundAttemptArtifact.apply soundFirstOrderArtifact (encodeState staleReturned)
    "persist-success" .suppressed = some (encodeState staleReturned) := by decide

example : mutatedAttemptArtifact.apply mutatedFirstOrderArtifact (encodeState staleReturned)
    "persist-success" .applied = some (encodeState staleSuccess) := by decide

end Umpire3.Tests.AttemptView
