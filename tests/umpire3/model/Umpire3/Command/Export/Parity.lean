import Temporal.Parity

def main : IO Unit := do
  let some semanticHash ← IO.getEnv "UMPIRE3_SEMANTIC_HASH"
    | throw (IO.userError "UMPIRE3_SEMANTIC_HASH is required")
  let some catalogHash ← IO.getEnv "UMPIRE3_CATALOG_HASH"
    | throw (IO.userError "UMPIRE3_CATALOG_HASH is required")
  let some dependencyHash ← IO.getEnv "UMPIRE3_DEPENDENCY_HASH"
    | throw (IO.userError "UMPIRE3_DEPENDENCY_HASH is required")
  IO.println (Umpire3.Temporal.Parity.json semanticHash dependencyHash catalogHash)
