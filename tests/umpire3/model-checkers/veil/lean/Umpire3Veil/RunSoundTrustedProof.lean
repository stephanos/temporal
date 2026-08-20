import Umpire3Veil.Generated.NexusCancellationSoundTrusted
import Umpire3Veil.JobReceipt

def main (arguments : List String) : IO UInt32 :=
  Umpire3Veil.JobReceipt.run Umpire3Veil.Generated.NexusCancellationSoundEvidence arguments
