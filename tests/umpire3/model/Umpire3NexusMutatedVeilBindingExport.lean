import Temporal.Veil.NexusCancellationFencing.MutatedBinding

def main : IO Unit := do
  let some sourceDigest ← IO.getEnv "UMPIRE3_SEMANTIC_HASH"
    | throw (IO.userError "UMPIRE3_SEMANTIC_HASH is required")
  let some firstOrderSemanticHash ← IO.getEnv "UMPIRE3_DEPENDENCY_HASH"
    | throw (IO.userError "UMPIRE3_DEPENDENCY_HASH is required")
  let some exported := Umpire3.Temporal.Veil.NexusCancellationFencing.mutatedBindingExport
    | throw (IO.userError "mutated Veil binding does not match its checked first-order export")
  IO.println (exported.json firstOrderSemanticHash sourceDigest)
