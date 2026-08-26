import ModelLint.ImportGraph

/-!
Canonical, symlink-safe source inventory for the model import-graph linter.

This adapter walks a package root without entering build/runtime state. It resolves every visited
path before traversal, rejects containment escapes, and fails when a second logical directory maps
to an already visited canonical directory. The latter makes both directory aliases and symlink
cycles explicit instead of silently dropping a namespace from the inventory.
-/

namespace ModelLint.Inventory

open Lean ModelLint.ImportGraph System

private structure DirectoryVisit where
  canonical : FilePath
  logical : FilePath

private def excludedDirectories : Array String :=
  #[".git", ".lake", ".flow", "build", "dist", "runtime", "target", "tmp"]

private def canonicalPath (path : FilePath) : IO FilePath :=
  return (← IO.FS.realPath path).normalize

private def sameFilesystemObject (left right : FilePath) : IO Bool := do
  let child ← IO.Process.spawn {
    cmd := "sh"
    args := #["-c", "test \"$1\" -ef \"$2\"", "model-lint", left.toString, right.toString]
    stdin := .null
    stdout := .null
    stderr := .null
  }
  match ← child.wait with
  | 0 => pure true
  | 1 => pure false
  | status => throw <| IO.userError s!"filesystem identity check failed with status {status}"

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

private partial def scanDirectory
    (root directory : FilePath)
    (visited : Array DirectoryVisit) : IO (Array SourceRecord × Array DirectoryVisit) := do
  let canonicalDirectory ← canonicalPath directory
  unless containedBy root canonicalDirectory do
    throw <| IO.userError s!"directory symlink escapes canonical root: {directory} -> \
      {canonicalDirectory}"
  if let some previous := visited.find? (·.canonical == canonicalDirectory) then
    throw <| IO.userError s!"directory alias or cycle: {directory} -> {canonicalDirectory}; \
      already inventoried as {previous.logical}"
  for previous in visited do
    if previous.canonical.toString.toLower == canonicalDirectory.toString.toLower &&
        (← sameFilesystemObject previous.logical directory) then
      let previousType ← previous.logical.symlinkMetadata
      let directoryType ← directory.symlinkMetadata
      if previousType.type == .dir && directoryType.type == .dir &&
          previous.logical.toString.toLower == directory.toString.toLower then
        return (#[], visited)
      throw <| IO.userError s!"directory alias or cycle: {directory} -> {canonicalDirectory}; \
        already inventoried as {previous.logical}"
  let mut visited := visited.push { canonical := canonicalDirectory, logical := directory }
  let mut sources := #[]
  let entries := (← directory.readDir).qsort fun left right => left.fileName < right.fileName
  for entry in entries do
    unless excludedDirectories.contains entry.fileName do
      let metadata ← entry.path.symlinkMetadata
      let canonicalEntry ← canonicalPath entry.path
      unless containedBy root canonicalEntry do
        throw <| IO.userError s!"source symlink escapes canonical root: {entry.path} -> \
          {canonicalEntry}"
      let targetMetadata ← canonicalEntry.metadata
      if metadata.type == .dir || targetMetadata.type == .dir then
        let (nested, nestedVisited) ← scanDirectory root entry.path visited
        sources := sources ++ nested
        visited := nestedVisited
      else if targetMetadata.type == .file && entry.path.extension == some "lean" then
        sources := sources.push {
          path := entry.path.toString
          module := ← moduleNameForSource root entry.path
          contained := true
        }
  pure (sources, visited)

/-- Inventory all Lean sources below `sourceRoot`, rejecting path aliases, cycles, and escapes. -/
def scanSources (sourceRoot : FilePath) : IO (Array SourceRecord) := do
  let root ← canonicalPath sourceRoot
  let (sources, _) ← scanDirectory root root #[]
  pure <| sources.qsort fun left right => left.module.toString < right.module.toString

/-- Inventory the canonical current Lake package root used by the model lint executable. -/
def canonicalModelSources : IO (Array SourceRecord) := do
  let root ← canonicalPath (← IO.currentDir)
  unless (← (root / "lakefile.toml").pathExists) do
    throw <| IO.userError s!"canonical model root has no lakefile.toml: {root}"
  scanSources root

end ModelLint.Inventory
