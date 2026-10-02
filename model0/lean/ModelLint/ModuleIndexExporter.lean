import Lake.Load.Workspace
import Lake.Config.InstallPath
import Lean.Util.Path
import ModelLint.ModuleIndex
import ModelLint.PackageModules

/-!
# The module index exporter's two decisions

`temporal-model-module-index` is the shared loader, the pure index and one write. What this module
adds is the two things neither of those can decide for it.

**Where it is.** The loader inventories whatever directory it runs in, and would happily inventory
a different Lake package and report its modules as ours. The exporter first loads the current
directory as a Lake root, without resolving dependencies or touching the toolchain, and refuses to
continue unless that root declares the `temporal-model` package and owns the executables the index
is about. A relocated checkout passes; another package, or a package that merely shares the name,
does not. This identifies the project's shape, not a security principal.

**What it writes.** The complete document is in memory before the first byte leaves, so a loader,
index or rendering failure writes nothing to the output stream. The final write is one call to an
injected writer, and if that call fails the exit code says so; what the operating system had
already accepted of the document is not ours to take back, which is why a caller reads the exit
code and not the length of what arrived.
-/

namespace ModelLint.ModuleIndexExporter

open Lean System
open ModelLint.ModuleIndex
open ModelLint.PackageModules

/-- The package the exporter indexes, as its lakefile declares it. -/
def packageName : Name := .mkSimple "temporal-model"

/-- The executables the root must own, each with the module it must name as its root. Owning the
linter, its tests and this exporter is what distinguishes the model package from another package
that borrowed its name. -/
def ownedExecutables : Array (Name × Name) := #[
  (.mkSimple "umpire-lint", `ModelLint),
  (.mkSimple "umpire-lint-tests", `ModelLint.ImportGraphTests),
  (.mkSimple "temporal-model-module-index", `ModelLint.ModuleIndexMain)
]

/-- The prefix of every diagnostic the exporter itself raises; loader and index diagnostics keep
their own. -/
def diagnosticPrefix (kind : String) : String := s!"[model-module-index/{kind}]"

private def rootFailure (message : String) : String :=
  s!"{diagnosticPrefix "root"} {message}"

/-- A name as a person would write it in the lakefile, without Lean's guillemets. -/
private def plain (name : Name) : String := name.toString (escape := false)

/-- What the exporter says when the final write did not complete. -/
def writeFailureMessage (failure : String) : String :=
  s!"{diagnosticPrefix "write"} the document could not be written in full; whatever the stream \
    already accepted is not a document: {failure}"

private def loadConfig (env : Lake.Env) (root : FilePath) : Lake.LoadConfig := {
  lakeEnv := env
  wsDir := root
  reconfigure := false
  updateDeps := false
  updateToolchain := false
}

/-- Check that the current directory is the model package, as a list of reasons it is not.

Only the root configuration is loaded: no dependency is resolved, no manifest is updated and no
toolchain is changed, and what Lake logged while loading is returned with the failure rather than
written by this function. -/
def preflightRoot : IO (Except (Array String) FilePath) := do
  let root ← Lean.realPathNormalized (← IO.currentDir)
  let (elan?, lean?, lake?) ← Lake.findInstall?
  let some lean := lean?
    | return .error #[rootFailure "no Lean installation is visible; run through `lake exe`"]
  let some lake := lake?
    | return .error #[rootFailure "no Lake installation is visible; run through `lake exe`"]
  let env ← match ← (Lake.Env.compute lake lean elan?).toBaseIO with
    | .ok env => pure env
    | .error message => return .error #[rootFailure s!"Lake environment: {message}"]
  let (workspace?, log) ← (Lake.loadWorkspaceRoot (loadConfig env root)).run? {}
  let some workspace := workspace?
    | return .error <| #[rootFailure s!"{root} is not a Lake package root"] ++
        log.entries.map fun entry => rootFailure entry.toString
  let mut failures : Array String := #[]
  let packageDir ← Lean.realPathNormalized workspace.root.dir
  unless packageDir == root do
    failures := failures.push
      (rootFailure s!"Lake resolved {root} to a package at {packageDir}")
  unless workspace.root.origName == packageName do
    failures := failures.push
      (rootFailure s!"{root} declares package `{plain workspace.root.origName}`, not \
        `{plain packageName}`")
  for (executable, moduleName) in ownedExecutables do
    match workspace.root.findLeanExe? executable with
    | none =>
        failures := failures.push
          (rootFailure s!"{root} declares no executable `{plain executable}`")
    | some exe =>
        unless exe.root.name == moduleName do
          failures := failures.push
            (rootFailure s!"{root} roots executable `{plain executable}` at \
              `{plain exe.root.name}`, not `{plain moduleName}`")
  return if failures.isEmpty then .ok root else .error failures

/-- Load, index and write, with every stream injected.

The loader's transcript follows the exporter's policy: a successful build says nothing, a failed one
says everything on the error stream and then why. Nothing reaches `writeOutput` until the whole
document exists. -/
def run (effects : Effects) (roots : IndexPolicy := defaultIndexPolicy)
    (writeOutput writeError : String → IO Unit) : IO UInt32 := do
  let root ← Lean.realPathNormalized (← IO.currentDir)
  match ← load ModelLint.ImportGraph.defaultPolicy effects with
  | .error (.discovery message) =>
      writeError (discoveryFailureMessage message ++ "\n")
      pure 1
  | .error (.sources issues) =>
      for issue in issues do
        writeError (ModelLint.ImportGraph.InventoryIssue.render issue ++ "\n")
      pure 1
  | .error (.build transcript) =>
      writeError (quieted transcript).stderr
      writeError (buildFailureMessage ++ "\n")
      pure 1
  | .error (.metadata issues) =>
      for issue in issues do
        writeError (issue.render ++ "\n")
      pure 1
  | .ok loaded =>
      writeError (quieted loaded.transcript).stderr
      let sources := relativizeSources root.toString loaded.sources
      match build ModelLint.ImportGraph.defaultPolicy roots sources loaded.modules with
      | .error issues =>
          for issue in issues do
            writeError (issue.render ++ "\n")
          pure 1
      | .ok index =>
          let document := render index
          -- The records' names may point into the mapped module data; the regions stay reachable
          -- until the document has been rendered from them.
          let _loadedRegionCount := loaded.regions.size
          try
            writeOutput document
            pure 0
          catch failure =>
            try writeError (writeFailureMessage failure.toString ++ "\n") catch _ => pure ()
            pure 1

/-- What the executable does: refuse a wrong root, otherwise export to the standard streams. -/
unsafe def main (args : List String) : IO UInt32 := do
  unless args.isEmpty do
    IO.eprintln "usage: temporal-model-module-index (no arguments; run from the model package root)"
    return 2
  match ← preflightRoot with
  | .error failures =>
      for failure in failures do
        IO.eprintln failure
      pure 1
  | .ok _ => run liveEffects defaultIndexPolicy IO.print IO.eprint

end ModelLint.ModuleIndexExporter
