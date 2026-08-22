import Temporal.Families.NexusCancellation.Experiment

def main : IO Unit := do
  IO.println (Umpire3.SemanticProofManifest.json
    Umpire3.Temporal.Experiments.NexusCancellation.proofManifest)
