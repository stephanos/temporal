import Batteries.Tactic.Lint
import Lake.CLI.Main
import ModelLint.Entity
import ModelLint.ImportGraph
import ModelLint.PackageModules
import Tools.LeanImportGraph.Metadata
import Tools.LeanSourceInventory
import Umpire.Command.Registry

/-! Whole-environment model linting beyond Lean's built-in declaration linters. -/

open Batteries.Tactic.Lint Lean Core
open System

namespace ModelLint

open ImportGraph

/-- `umpire-lint` replays what the child build wrote. The policy and the writing both belong to
`PackageModules`, so the exporter's quieter policy is a sibling of this one rather than a second
implementation of it. -/
private def replay (transcript : PackageModules.BuildTranscript) : IO Unit :=
  (PackageModules.replayed transcript).write

/-- The committed ledger of modules that build authoring records without the commands, read from
the model root the lint runs in. -/
private def handwrittenInventoryPath : System.FilePath := "HANDWRITTEN_INVENTORY.md"

/-- The lakefile `checkBuildRootsDrift` reads the declared roots from, relative to the model root
the lint runs in. -/
private def lakefilePath : System.FilePath := "lakefile.lean"

/-- The import-graph pass, and the owned source modules it discovered (none when discovery or a
later phase failed), which the declaration pass reads rather than discovering them again. -/
private unsafe def lintImportGraph : IO (Bool × Array Name) := do
  match ← PackageModules.load defaultPolicy PackageModules.liveEffects with
  | .error (.discovery reason) =>
      IO.eprintln (PackageModules.discoveryFailureMessage reason)
      pure (false, #[])
  | .error (.sources issues) =>
      for issue in issues do
        IO.eprintln (ImportGraph.InventoryIssue.render issue)
      pure (false, #[])
  | .error (.build transcript) =>
      replay transcript
      IO.eprintln PackageModules.buildFailureMessage
      pure (false, #[])
  | .error (.metadata issues) =>
      for issue in issues do
        IO.eprintln issue.render
      pure (false, #[])
  | .ok loaded =>
      replay loaded.transcript
      -- The hand-written ledger is an input of the lint: a module that builds authoring records
      -- without the commands is listed there with a destination, or reported here.
      let ledger ← IO.FS.readFile handwrittenInventoryPath
      let inventoryIssues := reconcile defaultPolicy loaded.sources loaded.modules ++
        reconcileHandwritten defaultPolicy (inventoriedModules ledger) loaded.modules
      for issue in inventoryIssues do
        IO.eprintln (ImportGraph.InventoryIssue.render issue)
      let violations := check defaultPolicy loaded.modules
      for violation in violations do
        IO.eprintln violation.render
      let unbuilt := checkUnbuilt defaultPolicy loaded.modules
      for issue in unbuilt do
        IO.eprintln issue.render
      let lakefileSource ← IO.FS.readFile lakefilePath
      let rootDrift := checkBuildRootsDrift defaultPolicy lakefileSource
      for line in rootDrift.render do
        IO.eprintln line
      -- The records' names may point into the mapped module data, so the regions stay reachable
      -- until every reader above has finished with them.
      let _loadedRegionCount := loaded.regions.size
      let discovered := loaded.sources.map (·.module)
      if inventoryIssues.isEmpty && violations.isEmpty && unbuilt.isEmpty && rootDrift.isEmpty then
        IO.println "-- Model import-graph linting passed."
        pure (true, discovered)
      else
        pure (false, discovered)

end ModelLint

private def lintModules : Array Name := #[`Shared, `Temporal.Lint, `Umpire.Lint]

private def enabledLinters : List Name := [
  `checkType,
  `impossibleInstance,
  `nonClassInstance,
  `simpComm,
  `simpNF,
  `synTaut,
  `unusedArguments,
  `unusedHavesSuffices
]

private def buildIfNeeded (module : Name) : IO Unit := do
  let olean ← findOLean module
  unless (← olean.pathExists) do
    let child ← IO.Process.spawn {
      cmd := (← IO.getEnv "LAKE").getD "lake"
      args := #["build", s!"+{module}"]
      stdin := .null
    }
    let exitCode ← child.wait
    if exitCode != 0 then
      throw <| IO.userError s!"failed to build lint module {module}"

private unsafe def lintModule (module : Name) : IO Bool := do
  initSearchPath (← findSysroot)
  buildIfNeeded module
  Lean.enableInitializersExecution
  let env ← importModules #[module, `Batteries.Tactic.Lint] {}
    (trustLevel := 1024) (loadExts := true)
  let context : Core.Context := {
    fileName := ""
    fileMap := default
    options := {}
  }
  let state : Core.State := { env }
  Prod.fst <$> (CoreM.toIO · context state) do
    let declarations ← getDeclsInPackage module.getRoot
    let linters ← getChecks (slow := true) (runOnly := some enabledLinters) (runAlways := none)
    let results ← lintCore declarations linters (inIO := true) (currentModule := module)
    if results.any (!·.2.isEmpty) then
      let formatted ← formatLinterResults results declarations (groupByFilename := true)
        s!"in {module}" (runSlowLinters := true) .medium linters.size (useErrorFormat := true)
      IO.print (← formatted.toString)
      pure false
    else
      IO.println s!"-- Batteries linting passed for {module}."
      pure true

/-- `feature-entity-uniqueness` reads declarations, which the import-graph rules cannot see and the
Batteries pass has no rule for, so it is its own pass over an imported environment's registry. The
environment imports every production feature module the source discovery found, so a module no
aggregate imports is read all the same. -/
private unsafe def lintFeatureEntities (discovered : Array Name) : IO Bool := do
  let roots := ModelLint.Entity.importRoots discovered
  if roots.isEmpty then
    IO.eprintln "[model-entity/feature-entity-uniqueness] no production feature module discovered"
    return false
  initSearchPath (← findSysroot)
  roots.forM buildIfNeeded
  Lean.enableInitializersExecution
  let env ← importModules (roots.map fun module => { module }) {}
    (trustLevel := 1024) (loadExts := true)
  let mut declarations : Array ModelLint.Entity.Declaration := #[]
  let mut unplaced : Array Name := #[]
  let declared := (Umpire.Command.Registry.entities env).map (fun entry =>
      (ModelLint.Entity.Kind.entity, entry.name, entry.declName)) ++
    (Umpire.Command.Registry.actions env).map fun entry =>
      (ModelLint.Entity.Kind.action, entry.name, entry.declName)
  for (kind, name, declName) in declared do
    match env.getModuleIdxFor? declName with
    | some index =>
        declarations := declarations.push
          { kind, name, module := env.header.moduleNames[index.toNat]! }
    | none => unplaced := unplaced.push declName
  -- A declaration no imported module owns is one the rule cannot place, so it fails rather than
  -- being left out of the comparison.
  for declName in unplaced do
    IO.eprintln s!"[model-entity/feature-entity-uniqueness] no declaring module for {declName}"
  let violations := ModelLint.Entity.check ModelLint.Entity.allowlist declarations
  for violation in violations do
    IO.eprintln violation.render
  if unplaced.isEmpty && violations.isEmpty then
    IO.println "-- Feature entity uniqueness linting passed."
    pure true
  else
    pure false

unsafe def main : IO UInt32 := do
  let (graphPassed, discovered) ← ModelLint.lintImportGraph
  let passed ← lintModules.mapM lintModule
  let entitiesPassed ← lintFeatureEntities discovered
  pure <| ModelLint.ImportGraph.exitCode graphPassed (passed.all id && entitiesPassed)
