import Tools.LeanImportGraph
import Lean.Environment
import Lean.Util.Path

/-!
Load actual compiled import metadata for architecture checks. Region ownership travels with the
records because their names may point into the mapped module data.
-/

namespace Tools.LeanImportGraph.Metadata

open Lean

/-- Read compiled module records, retaining their compacted regions for the caller's traversal.
Only reachable imports are loaded, once per qualified name. Owned imports absent from `roots`
remain missing so a stale compiled file cannot repair an incomplete source inventory. -/
unsafe def load
    (roots : Array Name)
    (isOwned : Name → Bool := fun _ => false) : IO (Array ModuleRecord × Array CompactedRegion) := do
  initSearchPath (← findSysroot)
  let mut records := #[]
  let mut regions := #[]
  let mut queue := roots
  let mut visited : Std.HashSet Name := {}
  let mut index := 0
  while index < queue.size do
    let name := queue[index]!
    index := index + 1
    if visited.contains name then continue
    visited := visited.insert name
    if isOwned name && !roots.contains name then continue
    let olean ← findOLean name
    let (metadata, region) ← readModuleData olean
    regions := regions.push region
    let imports := metadata.imports.map (·.module)
    records := records.push { name, imports }
    queue := queue ++ imports
  pure (records, regions)

end Tools.LeanImportGraph.Metadata
