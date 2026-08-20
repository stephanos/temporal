import Temporal.Targets.NexusCancellationFencingFirstOrder

def main : IO Unit := do
  let some semanticHash ← IO.getEnv "UMPIRE3_SEMANTIC_HASH"
    | throw (IO.userError "UMPIRE3_SEMANTIC_HASH is required")
  let some exported := Umpire3.Temporal.Targets.NexusCancellationFencing.mutatedFirstOrderExport
    | throw (IO.userError "mutated first-order search did not produce a checked exhaustive certificate")
  IO.println (exported.json semanticHash)
