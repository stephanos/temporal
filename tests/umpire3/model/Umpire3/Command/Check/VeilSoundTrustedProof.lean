import Temporal.Families.NexusCancellation.Targets.Veil.TrustedBinding
import Umpire3.Veil.JobReceipt

def main (arguments : List String) : IO UInt32 := do
  let some binding := Umpire3.Temporal.Veil.NexusCancellationFencing.soundTrustedBindingExport
    | IO.eprintln "trusted Veil binding does not match its checked first-order export"
      return 2
  Umpire3.Veil.JobReceipt.run {
    binding
    invariantAxioms := resolved_veil_axioms% [
      NexusCancellationSoundTrusted.initializer_doesNotThrow,
      NexusCancellationSoundTrusted.initializer_NexusCancellationWonExcludesSuccess,
      NexusCancellationSoundTrusted.initializer_CanonicalReachableEnvelope,
      NexusCancellationSoundTrusted.DispatchTask_doesNotThrow,
      NexusCancellationSoundTrusted.DispatchTask_NexusCancellationWonExcludesSuccess,
      NexusCancellationSoundTrusted.DispatchTask_CanonicalReachableEnvelope,
      NexusCancellationSoundTrusted.RequestCancellation_doesNotThrow,
      NexusCancellationSoundTrusted.RequestCancellation_NexusCancellationWonExcludesSuccess,
      NexusCancellationSoundTrusted.RequestCancellation_CanonicalReachableEnvelope,
      NexusCancellationSoundTrusted.AcquireOwnership_doesNotThrow,
      NexusCancellationSoundTrusted.AcquireOwnership_NexusCancellationWonExcludesSuccess,
      NexusCancellationSoundTrusted.AcquireOwnership_CanonicalReachableEnvelope,
      NexusCancellationSoundTrusted.CommitCancellation_doesNotThrow,
      NexusCancellationSoundTrusted.CommitCancellation_NexusCancellationWonExcludesSuccess,
      NexusCancellationSoundTrusted.CommitCancellation_CanonicalReachableEnvelope,
      NexusCancellationSoundTrusted.WorkerReturnsSuccess_doesNotThrow,
      NexusCancellationSoundTrusted.WorkerReturnsSuccess_NexusCancellationWonExcludesSuccess,
      NexusCancellationSoundTrusted.WorkerReturnsSuccess_CanonicalReachableEnvelope,
      NexusCancellationSoundTrusted.PersistSuccess_doesNotThrow,
      NexusCancellationSoundTrusted.PersistSuccess_NexusCancellationWonExcludesSuccess,
      NexusCancellationSoundTrusted.PersistSuccess_CanonicalReachableEnvelope
    ]
  } arguments
