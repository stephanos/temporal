import Temporal.Experiments.NexusCancellation

def main : IO Unit := do
  let some semanticHash ← IO.getEnv "UMPIRE3_SEMANTIC_HASH"
    | throw (IO.userError "UMPIRE3_SEMANTIC_HASH is required")
  IO.println (Umpire3.Temporal.Experiments.NexusCancellation.proofManifest.json semanticHash)
