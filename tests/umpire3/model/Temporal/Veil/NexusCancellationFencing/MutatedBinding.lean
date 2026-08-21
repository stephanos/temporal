import Temporal.Targets.NexusCancellationFencingFirstOrder
import Temporal.Veil.NexusCancellationFencing.Mutated
import Temporal.Veil.NexusCancellationFencing.MutatedConcrete
import Temporal.Veil.NexusCancellationFencing.MutatedConcreteSemantics
import Umpire3.Veil.Binding

namespace Umpire3.Temporal.Veil.NexusCancellationFencing

open Umpire3.Temporal.Targets.NexusCancellationFencing

def mutatedBinding : Umpire3.Veil.ResolvedBinding :=
  resolved_veil_binding% mutatedFirstOrderView NexusCancellationStaleCompletionGuardRemoved
    NexusCancellationStaleCompletionGuardRemovedConcrete mutatedSemanticBinding
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

def mutatedBindingExport : Option Umpire3.Veil.BindingExport :=
  mutatedFirstOrderExport.bind (Umpire3.Veil.BindingExport.of mutatedBinding)

end Umpire3.Temporal.Veil.NexusCancellationFencing
