import Batteries.Tactic.Lint
import Lake.CLI.Main
import ModelLint.ImportGraph
import ModelLint.PackageModules
import Tools.LeanImportGraph.Metadata
import Tools.LeanSourceInventory

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

private unsafe def lintImportGraph : IO Bool := do
  match ← PackageModules.load defaultPolicy PackageModules.liveEffects with
  | .error (.discovery message) =>
      IO.eprintln (PackageModules.discoveryFailureMessage message)
      pure false
  | .error (.sources issues) =>
      for issue in issues do
        IO.eprintln (ImportGraph.InventoryIssue.render issue)
      pure false
  | .error (.build transcript) =>
      replay transcript
      IO.eprintln PackageModules.buildFailureMessage
      pure false
  | .error (.metadata issues) =>
      for issue in issues do
        IO.eprintln issue.render
      pure false
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
      -- The records' names may point into the mapped module data, so the regions stay reachable
      -- until every reader above has finished with them.
      let _loadedRegionCount := loaded.regions.size
      if inventoryIssues.isEmpty && violations.isEmpty then
        IO.println "-- Model import-graph linting passed."
        pure true
      else
        pure false

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

unsafe def main : IO UInt32 := do
  let graphPassed ← ModelLint.lintImportGraph
  let passed ← lintModules.mapM lintModule
  pure <| ModelLint.ImportGraph.exitCode graphPassed (passed.all id)
