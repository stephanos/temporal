namespace Umpire3.Temporal.NexusCancellationFencing

inductive World where
  | smoke
  deriving DecidableEq, Inhabited, Repr

def World.maxOwnerEpoch : World → Nat
  | .smoke => 1

end Umpire3.Temporal.NexusCancellationFencing
