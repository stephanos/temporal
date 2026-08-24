import Temporal.Families.NexusCancellation.Targets.Veil.Binding
import Umpire3.Veil.JobReceipt

def main (arguments : List String) : IO UInt32 := do
  let some binding := Umpire3.Temporal.Veil.NexusCancellationFencing.soundBindingExport
    | IO.eprintln "sound Veil binding does not match its checked first-order export"
      return 2
  Umpire3.Veil.JobReceipt.run {
    binding
    invariantAxioms := resolved_veil_axioms% [
      NexusCancellationSound.initializer_doesNotThrow,
      NexusCancellationSound.initializer_NexusCancellationWonExcludesSuccess,
      NexusCancellationSound.initializer_CanonicalReachableEnvelope,
      NexusCancellationSound.DispatchTask_doesNotThrow,
      NexusCancellationSound.DispatchTask_NexusCancellationWonExcludesSuccess,
      NexusCancellationSound.DispatchTask_CanonicalReachableEnvelope,
      NexusCancellationSound.RequestCancellation_doesNotThrow,
      NexusCancellationSound.RequestCancellation_NexusCancellationWonExcludesSuccess,
      NexusCancellationSound.RequestCancellation_CanonicalReachableEnvelope,
      NexusCancellationSound.AcquireOwnership_doesNotThrow,
      NexusCancellationSound.AcquireOwnership_NexusCancellationWonExcludesSuccess,
      NexusCancellationSound.AcquireOwnership_CanonicalReachableEnvelope,
      NexusCancellationSound.CommitCancellation_doesNotThrow,
      NexusCancellationSound.CommitCancellation_NexusCancellationWonExcludesSuccess,
      NexusCancellationSound.CommitCancellation_CanonicalReachableEnvelope,
      NexusCancellationSound.WorkerReturnsSuccess_doesNotThrow,
      NexusCancellationSound.WorkerReturnsSuccess_NexusCancellationWonExcludesSuccess,
      NexusCancellationSound.WorkerReturnsSuccess_CanonicalReachableEnvelope,
      NexusCancellationSound.PersistSuccess_doesNotThrow,
      NexusCancellationSound.PersistSuccess_NexusCancellationWonExcludesSuccess,
      NexusCancellationSound.PersistSuccess_CanonicalReachableEnvelope
    ]
  } arguments
