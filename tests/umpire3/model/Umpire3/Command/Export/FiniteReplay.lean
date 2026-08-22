import Temporal.Targets.FiniteReplay

def main : IO Unit := do
  let some semanticHash ← IO.getEnv "UMPIRE3_SEMANTIC_HASH"
    | throw (IO.userError "UMPIRE3_SEMANTIC_HASH is required")
  let some catalogHash ← IO.getEnv "UMPIRE3_CATALOG_HASH"
    | throw (IO.userError "UMPIRE3_CATALOG_HASH is required")
  let some targets := Umpire3.Temporal.Targets.FiniteReplay.targetsJson? semanticHash
    | throw (IO.userError "finite replay exploration did not produce checked exhaustive graphs")
  IO.println (Umpire3.finiteReplayCatalogJson semanticHash catalogHash targets)
