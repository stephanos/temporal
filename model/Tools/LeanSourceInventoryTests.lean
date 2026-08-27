import Tools.LeanSourceInventory

/-! Executable regressions for the reusable Lean source-inventory interface. -/

namespace Tools.LeanSourceInventoryTests

open System
open Tools.LeanImportGraph
open Tools.LeanSourceInventory

private def moduleRecord (name : Lean.Name) (imports : Array Lean.Name := #[]) : ModuleRecord :=
  { name, imports }

private def requireEqual [BEq α] [Repr α] (label : String) (actual expected : α) : IO Unit :=
  unless actual == expected do
    throw <| IO.userError s!"{label}: expected {repr expected}, got {repr actual}"

private def requireFailureContaining (label needle : String) (action : IO α) : IO Unit := do
  let failure? ← try
    action *> pure none
  catch error =>
    pure <| some (toString error)
  let some failure := failure?
    | throw <| IO.userError s!"{label}: expected failure containing {needle}"
  unless failure.contains needle do
    throw <| IO.userError s!"{label}: expected failure containing {needle}, got {failure}"

private def runProcess (command : String) (args : Array String) : IO Unit := do
  discard <| IO.Process.run { cmd := command, args }

private def isOwned (name : Lean.Name) : Bool :=
  `Owned == name || (`Owned).isPrefixOf name

private def inventoryPolicy : InventoryPolicy := {
  isFirstParty := isOwned
  isClassified := isOwned
}

private def testReconciliation : IO Unit := do
  requireEqual "generic duplicate source identity"
    (reconcile inventoryPolicy
      #[
        { path := "Owned/Root.lean", module := `Owned.Root },
        { path := "Alias/Root.lean", module := `Owned.Root }
      ]
      #[moduleRecord `Owned.Root])
    #[.duplicateSource `Owned.Root #["Alias/Root.lean", "Owned/Root.lean"]]
  requireEqual "generic duplicate metadata"
    (reconcile inventoryPolicy #[{ path := "Owned/Root.lean", module := `Owned.Root }]
      #[moduleRecord `Owned.Root, moduleRecord `Owned.Root])
    #[.duplicateMetadata `Owned.Root]
  requireEqual "generic uncovered source"
    (reconcile inventoryPolicy #[{ path := "Owned/Root.lean", module := `Owned.Root }] #[])
    #[.uncoveredSource `Owned.Root "Owned/Root.lean"]

private def testFilesystemInventory : IO Unit := do
  IO.FS.withTempDir fun root => do
    let runtime := root / "runtime"
    IO.FS.createDir runtime
    IO.FS.writeFile (runtime / "Root.lean") "def runtimeSource := true\n"
    let sources ← scanSources root
    requireEqual "generic inventory retains caller-owned directory names"
      (sources.map (·.module)) #[`runtime.Root]
    let excluded ← scanSources root #["runtime"]
    requireEqual "caller-owned inventory exclusions" excluded #[]
  IO.FS.withTempDir fun root => do
    let canonical := root / ".lake"
    IO.FS.createDir canonical
    IO.FS.writeFile (canonical / "Root.lean") "def canonical := true\n"
    runProcess "ln" #["-s", canonical.toString, (root / "Alias").toString]
    requireFailureContaining "excluded directory alias" "source path alias"
      (scanSources root #[".lake"])
  IO.FS.withTempDir fun root => do
    let directory := root / "Cycle"
    IO.FS.createDir directory
    runProcess "ln" #["-s", root.toString, (directory / "Loop").toString]
    requireFailureContaining "directory cycle" "source path alias" (scanSources root)
  IO.FS.withTempDir fun root =>
    IO.FS.withTempDir fun outside => do
      IO.FS.writeFile (outside / "Escape.lean") "def escape := true\n"
      runProcess "ln" #["-s", outside.toString, (root / "Outside").toString]
      requireFailureContaining "escaping directory" "escapes canonical root" (scanSources root)
  IO.FS.withTempDir fun root => do
    let canonical := root / "Canonical.lean"
    IO.FS.writeFile canonical "def canonical := true\n"
    runProcess "ln" #["-s", canonical.toString, (root / "Alias.lean").toString]
    requireFailureContaining "source alias" "source path alias" (scanSources root)
  IO.FS.withTempDir fun root => do
    let canonical := root / "Case.lean"
    let caseVariant := root / "case.lean"
    IO.FS.writeFile canonical "def sameBytes := true\n"
    IO.FS.writeFile caseVariant "def sameBytes := true\n"
    let entries ← root.readDir
    if entries.any (·.fileName == "Case.lean") && entries.any (·.fileName == "case.lean") then
      let sources ← scanSources root
      requireEqual "distinct case variants retained" (sources.map (·.module)) #[`Case, `case]

/-- Exercise reconciliation and filesystem behavior through the generic inventory interface. -/
def run : IO Unit := do
  testReconciliation
  testFilesystemInventory

end Tools.LeanSourceInventoryTests
