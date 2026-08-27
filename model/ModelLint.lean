import Batteries.Tactic.Lint
import Lake.CLI.Main
import ModelLint.ImportGraph
import Tools.LeanSourceInventory

/-! Whole-environment model linting beyond Lean's built-in declaration linters. -/

open Batteries.Tactic.Lint Lean Core
open System

namespace ModelLint

open ImportGraph

private def buildOwnedSources (sources : Array SourceRecord) : IO Unit := do
  -- The running executable already proves `ModelLint` current; Lake refreshes every other root.
  let ordinaryModules := sources.filterMap fun source =>
    if source.module == `ModelLint || source.module == `ModelLint.ImportGraphTests then none
    else some s!"+{source.module}"
  let args := #["build", "modelLintTests"] ++ ordinaryModules
  let child ← IO.Process.spawn {
    cmd := (← IO.getEnv "LAKE").getD "lake"
    args
    stdin := .null
  }
  let exitCode ← child.wait
  if exitCode != 0 then
    throw <| IO.userError "Lake failed to make every owned model source current"

private unsafe def loadModuleRecords
    (sources : Array SourceRecord) : IO (Array ModuleRecord × Array CompactedRegion) := do
  initSearchPath (← findSysroot)
  let mut records := #[]
  let mut regions := #[]
  for source in sources do
    let olean ← findOLean source.module
    let (metadata, region) ← readModuleData olean
    regions := regions.push region
    records := records.push {
      name := source.module
      imports := metadata.imports.map (·.module)
    }
  pure (records, regions)

private def captureStep (category : String) (action : IO α) : IO (Except String α) := do
  try
    pure <| .ok (← action)
  catch error =>
    pure <| .error s!"[model-import-graph/{category}] {error}"

private def excludedSourceDirectories : Array String :=
  #[".git", ".lake", ".flow", "build", "dist", "runtime", "target", "tmp"]

private unsafe def lintImportGraph : IO Bool := do
  match ← captureStep "inventory"
      (Tools.LeanSourceInventory.canonicalPackageSources excludedSourceDirectories) with
  | .error error => IO.eprintln error; pure false
  | .ok sources =>
    let sourceIssues := validateSources defaultPolicy sources
    if !sourceIssues.isEmpty then
      for issue in sourceIssues do
        IO.eprintln issue.render
      pure false
    else
      match ← captureStep "build" (buildOwnedSources sources) with
      | .error error => IO.eprintln error; pure false
      | .ok _ =>
        match ← captureStep "metadata" (loadModuleRecords sources) with
        | .error error => IO.eprintln error; pure false
        | .ok (modules, regions) =>
          let inventoryIssues := reconcile defaultPolicy sources modules
          for issue in inventoryIssues do
            IO.eprintln issue.render
          let violations := check defaultPolicy modules
          for violation in violations do
            IO.eprintln violation.render
          let _loadedRegionCount := regions.size
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
