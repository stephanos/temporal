import Batteries.Tactic.Lint
import Lake.CLI.Main
import ModelLint.ImportGraph

/-! Whole-environment model linting beyond Lean's built-in declaration linters. -/

open Batteries.Tactic.Lint Lean Core
open System

namespace ModelLint

open ImportGraph

private def excludedInventoryDirectories : Array String :=
  #[".git", ".lake", ".flow", "build", "dist", "runtime", "target", "tmp"]

private def containedBy (root path : FilePath) : Bool :=
  let rootText := root.normalize.toString
  let pathText := path.normalize.toString
  let rootPrefix :=
    if rootText.endsWith FilePath.pathSeparator.toString then rootText
    else rootText ++ FilePath.pathSeparator.toString
  pathText == rootText || rootPrefix.isPrefixOf pathText

private def moduleNameForSource (root source : FilePath) : IO Name := do
  let rootText := root.normalize.toString
  let sourceText := source.normalize.toString
  let rootPrefix :=
    if rootText.endsWith FilePath.pathSeparator.toString then rootText
    else rootText ++ FilePath.pathSeparator.toString
  unless rootPrefix.isPrefixOf sourceText do
    throw <| IO.userError s!"source path is not beneath canonical root: {source}"
  let relative := FilePath.mk (sourceText.drop rootPrefix.length).copy |>.withExtension ""
  let module := relative.components.foldl Name.mkStr Name.anonymous
  if module.isAnonymous then
    throw <| IO.userError s!"source path has no qualified module identity: {source}"
  pure module

private partial def scanSourceDirectory
    (root directory : FilePath)
    (visited : Array String) : IO (Array SourceRecord × Array String) := do
  let canonicalDirectory ← realPathNormalized directory
  unless containedBy root canonicalDirectory do
    throw <| IO.userError s!"directory symlink escapes canonical root: {directory} -> \
      {canonicalDirectory}"
  if visited.contains canonicalDirectory.toString then
    return (#[], visited)
  let mut visited := visited.push canonicalDirectory.toString
  let mut sources := #[]
  let entries := (← directory.readDir).qsort fun left right => left.fileName < right.fileName
  for entry in entries do
    unless excludedInventoryDirectories.contains entry.fileName do
      let metadata ← entry.path.symlinkMetadata
      let canonicalEntry ← realPathNormalized entry.path
      unless containedBy root canonicalEntry do
        throw <| IO.userError s!"source symlink escapes canonical root: {entry.path} -> \
          {canonicalEntry}"
      let targetMetadata ← canonicalEntry.metadata
      if metadata.type == .dir || targetMetadata.type == .dir then
        let (nested, nestedVisited) ← scanSourceDirectory root entry.path visited
        sources := sources ++ nested
        visited := nestedVisited
      else if targetMetadata.type == .file && entry.path.extension == some "lean" then
        sources := sources.push {
          path := entry.path.toString
          module := ← moduleNameForSource root entry.path
          contained := true
        }
  pure (sources, visited)

private def inventorySources : IO (Array SourceRecord) := do
  let root ← realPathNormalized (← IO.currentDir)
  unless (← (root / "lakefile.toml").pathExists) do
    throw <| IO.userError s!"canonical model root has no lakefile.toml: {root}"
  let (sources, _) ← scanSourceDirectory root root #[]
  pure <| sources.qsort fun left right => left.module.toString < right.module.toString

private def buildOwnedSources (sources : Array SourceRecord) : IO Unit := do
  let ordinaryModules := sources.filterMap fun source =>
    if source.module == `ModelLint || source.module == `ModelLint.ImportGraphTests then none
    else some s!"+{source.module}"
  let args := #["build", "modelLint", "modelLintTests"] ++ ordinaryModules
  let child ← IO.Process.spawn {
    cmd := (← IO.getEnv "LAKE").getD "lake"
    args
    stdin := .null
  }
  let exitCode ← child.wait
  if exitCode != 0 then
    throw <| IO.userError "Lake failed to make every owned model source current"

private unsafe def loadModuleRecords (sources : Array SourceRecord) : IO (Array ModuleRecord) := do
  initSearchPath (← findSysroot)
  Lean.enableInitializersExecution
  let modules := sources.map (·.module)
  let env ← importModules modules {} (trustLevel := 1024) (loadExts := true)
  let mut metadataByName : Std.HashMap Name ModuleData := {}
  for i in [0:env.header.moduleNames.size] do
    let name := env.header.moduleNames[i]!
    if metadataByName.contains name then
      throw <| IO.userError s!"duplicate loaded module metadata: {name}"
    metadataByName := metadataByName.insert name env.header.moduleData[i]!
  let mut records := #[]
  for source in sources do
    let some metadata := metadataByName[source.module]?
      | throw <| IO.userError s!"missing loaded metadata for {source.module} ({source.path})"
    records := records.push {
      name := source.module
      imports := metadata.imports.map (·.module)
    }
  pure records

private def captureStep (category : String) (action : IO α) : IO (Except String α) := do
  try
    pure <| .ok (← action)
  catch error =>
    pure <| .error s!"[model-import-graph/{category}] {error}"

private unsafe def lintImportGraph : IO Bool := do
  match ← captureStep "inventory" inventorySources with
  | .error error => IO.eprintln error; pure false
  | .ok sources =>
    match ← captureStep "build" (buildOwnedSources sources) with
    | .error error => IO.eprintln error; pure false
    | .ok _ =>
      match ← captureStep "metadata" (loadModuleRecords sources) with
      | .error error => IO.eprintln error; pure false
      | .ok modules =>
        let inventoryIssues := reconcile defaultPolicy sources modules
        for issue in inventoryIssues do
          IO.eprintln issue.render
        let violations := check defaultPolicy modules
        for violation in violations do
          IO.eprintln violation.render
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
