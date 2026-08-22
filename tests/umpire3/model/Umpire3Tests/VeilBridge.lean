import Temporal.Families.NexusCancellation.Targets.Veil.Binding
import Temporal.Families.NexusCancellation.Targets.Veil.MutatedBinding
import Temporal.Families.NexusCancellation.Targets.Veil.SoundConcrete
import Temporal.Families.NexusCancellation.Targets.Veil.TrustedBinding

namespace Umpire3.Tests.VeilBridge

open Umpire3.Temporal.Targets.NexusCancellationFencing
open Umpire3.Temporal.Veil.NexusCancellationFencing

example : soundBinding.view.artifact = soundFirstOrderArtifact := rfl

example : soundBinding.semantic.declaration =
    "Umpire3.Temporal.Veil.NexusCancellationFencing.soundSemanticBinding" := rfl

example : soundBinding.semantic.axioms = ["propext", "Classical.choice", "Quot.sound"] := rfl

example : mutatedBinding.semantic.declaration =
    "Umpire3.Temporal.Veil.NexusCancellationFencing.mutatedSemanticBinding" := rfl

example : mutatedBinding.semantic.axioms = ["propext", "Classical.choice", "Quot.sound"] := rfl

example : soundTrustedBinding.moduleName = "NexusCancellationSoundTrusted" := rfl

example : soundTrustedBinding.semantic.declaration =
    "Umpire3.Temporal.Veil.NexusCancellationFencing.soundTrustedSemanticBinding" := rfl

example : soundTrustedBinding.semantic.axioms =
    ["propext", "Classical.choice", "Quot.sound"] := rfl

example : (resolved_veil_binding% soundFirstOrderView NexusCancellationSound
    NexusCancellationSoundConcrete soundSemanticBinding
    veil_actions [
      "dispatch-task" => DispatchTask,
      "request-cancellation" => RequestCancellation,
      "acquire-ownership" => AcquireOwnership,
      "commit-cancellation" => CommitCancellation,
      "worker-returns-success" => WorkerReturnsSuccess,
      "persist-success" => PersistSuccess
    ]
    veil_fields [
      "lifecycle" => lifecycle,
      "task" => task,
      "owner-epoch" => ownerEpoch,
      "worker-epoch" => workerEpoch,
      "completion-epoch" => completionEpoch
    ]
    veil_enums [
      "lifecycle" => Lifecycle [
        "open" => Open,
        "cancellation-accepted" => CancellationAccepted,
        "cancelled" => Cancelled,
        "succeeded" => Succeeded
      ],
      "task-stage" => TaskStage [
        "idle" => Idle,
        "dispatched" => Dispatched,
        "returned" => Returned
      ],
      "epoch" => Epoch [
        "none" => None,
        "epoch-0" => Epoch0,
        "epoch-1" => Epoch1
      ]
    ]
    veil_property "nexus.cancellation.won-excludes-success" =>
      NexusCancellationWonExcludesSuccess
    veil_trust reconstructed).actionLabels = soundBinding.actionLabels := rfl

/--
error: Veil binding cannot resolve declaration NexusCancellationMissing.State
-/
#guard_msgs in
#check resolved_veil_binding% soundFirstOrderView NexusCancellationMissing
  NexusCancellationSoundConcrete soundSemanticBinding
  veil_actions [
    "dispatch-task" => DispatchTask,
    "request-cancellation" => RequestCancellation,
    "acquire-ownership" => AcquireOwnership,
    "commit-cancellation" => CommitCancellation,
    "worker-returns-success" => WorkerReturnsSuccess,
    "persist-success" => PersistSuccess
  ]
  veil_fields [
    "lifecycle" => lifecycle,
    "task" => task,
    "owner-epoch" => ownerEpoch,
    "worker-epoch" => workerEpoch,
    "completion-epoch" => completionEpoch
  ]
  veil_enums [
    "lifecycle" => Lifecycle [
      "open" => Open,
      "cancellation-accepted" => CancellationAccepted,
      "cancelled" => Cancelled,
      "succeeded" => Succeeded
    ],
    "task-stage" => TaskStage [
      "idle" => Idle,
      "dispatched" => Dispatched,
      "returned" => Returned
    ],
    "epoch" => Epoch [
      "none" => None,
      "epoch-0" => Epoch0,
      "epoch-1" => Epoch1
    ]
  ]
  veil_property "nexus.cancellation.won-excludes-success" =>
    NexusCancellationWonExcludesSuccess
  veil_trust reconstructed

example : soundBinding.actionLabels = [
    ("dispatch-task", "DispatchTask"),
    ("request-cancellation", "RequestCancellation"),
    ("acquire-ownership", "AcquireOwnership"),
    ("commit-cancellation", "CommitCancellation"),
    ("worker-returns-success", "WorkerReturnsSuccess"),
    ("persist-success", "PersistSuccess"),
  ] := rfl

#check NexusCancellationSoundConcrete.modelCheckerResult

end Umpire3.Tests.VeilBridge
