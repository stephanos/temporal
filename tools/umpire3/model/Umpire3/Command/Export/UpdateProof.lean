import Temporal.Families.UpdateLifecycle.Experiment

def main : IO Unit := do
  IO.println (Umpire3.SemanticProofManifest.json
    Umpire3.Temporal.Experiments.UpdateLifecycle.proofManifest)
