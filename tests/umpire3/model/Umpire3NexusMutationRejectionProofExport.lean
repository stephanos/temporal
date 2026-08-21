import Temporal.Mutations.NexusCancellationFencing

def main : IO Unit := do
  IO.println (Umpire3.SemanticProofManifest.json
    Umpire3.Temporal.Mutations.NexusCancellationFencing.refinementManifest)
