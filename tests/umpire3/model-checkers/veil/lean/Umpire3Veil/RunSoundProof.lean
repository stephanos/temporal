import Umpire3Veil.Generated.NexusCancellationSound
import Umpire3Veil.JobReceipt

def main (arguments : List String) : IO UInt32 :=
  Umpire3Veil.JobReceipt.run Umpire3Veil.Generated.NexusCancellationSoundEvidence arguments
