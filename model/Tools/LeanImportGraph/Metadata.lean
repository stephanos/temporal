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
remain missing so a stale compiled file cannot repair an incomplete source inventory.

A module whose metadata cannot be read is recorded as `(name, message)` and the traversal continues
with the modules reachable independently of it. Its own imports are *not* queued: a descendant
reached only through it was never examined, and queueing it would let the caller claim to have
examined a module it never opened. A caller that wants all-or-nothing reads the issues and discards
the records, which is what the one in `ModelLint.PackageModules` does. -/
unsafe def load
    (roots : Array Name)
    (isOwned : Name → Bool := fun _ => false) :
    IO (Array ModuleRecord × Array CompactedRegion × Array (Name × String)) := do
  initSearchPath (← findSysroot)
  let mut records := #[]
  let mut regions := #[]
  let mut issues := #[]
  let mut queue := roots
  let mut visited : Std.HashSet Name := {}
  let mut index := 0
  while index < queue.size do
    let name := queue[index]!
    index := index + 1
    if visited.contains name then continue
    visited := visited.insert name
    if isOwned name && !roots.contains name then continue
    let read ← try
        let olean ← findOLean name
        pure (Except.ok (← readModuleData olean))
      catch error => pure (Except.error (toString error))
    match read with
    | .error message => issues := issues.push (name, message)
    | .ok (metadata, region) =>
      regions := regions.push region
      let imports := metadata.imports.map (·.module)
      records := records.push { name, imports }
      queue := queue ++ imports
  pure (records, regions, issues)

end Tools.LeanImportGraph.Metadata
