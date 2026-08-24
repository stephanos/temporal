import Temporal.Families.NexusCancellation.Targets.FirstOrder
import Temporal.Families.NexusCancellation.Targets.Veil.SoundConcrete
import Temporal.Families.NexusCancellation.Targets.Veil.SoundTrusted
import Temporal.Families.NexusCancellation.Targets.Veil.TrustedSemantics
import Umpire3.Veil.Binding

namespace Umpire3.Temporal.Veil.NexusCancellationFencing

open Umpire3.Temporal.Targets.NexusCancellationFencing

def soundTrustedBinding : Umpire3.Veil.ResolvedBinding :=
  resolved_veil_binding% soundFirstOrderView NexusCancellationSoundTrusted
    NexusCancellationSoundConcrete soundTrustedSemanticBinding
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
    veil_trust trusted

def soundTrustedBindingExport : Option Umpire3.Veil.BindingExport :=
  soundFirstOrderExport.bind (Umpire3.Veil.BindingExport.of soundTrustedBinding)

end Umpire3.Temporal.Veil.NexusCancellationFencing
