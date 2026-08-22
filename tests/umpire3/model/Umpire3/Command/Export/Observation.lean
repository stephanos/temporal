import Temporal.Observation

open Umpire3.Observation

def main : IO Unit := do
  let some semanticHash ← IO.getEnv "UMPIRE3_SEMANTIC_HASH"
    | throw (IO.userError "UMPIRE3_SEMANTIC_HASH is required")
  let some catalogHash ← IO.getEnv "UMPIRE3_CATALOG_HASH"
    | throw (IO.userError "UMPIRE3_CATALOG_HASH is required")
  IO.println (catalogJson semanticHash catalogHash
    (Umpire3.Temporal.Observation.Nexus.programs ++
      Umpire3.Temporal.Observation.Families.programs)
    (Umpire3.Temporal.Observation.Nexus.fixtures ++
      Umpire3.Temporal.Observation.Families.fixtures))
