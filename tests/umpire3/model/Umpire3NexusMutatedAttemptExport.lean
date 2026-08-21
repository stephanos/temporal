import Temporal.Targets.NexusCancellationFencingAttempt

def main : IO Unit := do
  let some semanticHash ← IO.getEnv "UMPIRE3_SEMANTIC_HASH"
    | throw (IO.userError "UMPIRE3_SEMANTIC_HASH is required")
  let some firstOrderSemanticHash ← IO.getEnv "UMPIRE3_DEPENDENCY_HASH"
    | throw (IO.userError "UMPIRE3_DEPENDENCY_HASH is required")
  let exported := Umpire3.Temporal.Targets.NexusCancellationFencing.mutatedAttemptExport
  IO.println (exported.json semanticHash firstOrderSemanticHash)
