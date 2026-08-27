import Tools.LeanImportGraph

/-!
Canonical Lean source discovery and import-metadata reconciliation.

This module inventories a package root without entering build/runtime state. It resolves every
visited path before traversal, rejects containment escapes, and fails when a second logical
directory maps to an already visited canonical directory. Its pure reconciliation interface checks
source identities against caller-supplied module metadata and ownership/classification predicates.
-/

namespace Tools.LeanSourceInventory

open Lean System
open Tools.LeanImportGraph

/-- One qualified Lean source discovered beneath a package root. -/
structure SourceRecord where
  path : String
  module : Lean.Name
  contained : Bool := true
  deriving Repr, BEq

/-- Caller-owned tests for first-party identity and policy classification. -/
structure InventoryPolicy where
  isFirstParty : Lean.Name → Bool
  isClassified : Lean.Name → Bool

/-- A fail-closed discrepancy between owned sources and loaded module metadata. -/
inductive InventoryIssue where
  | escapingSource (path : String)
  | duplicateSource (module : Lean.Name) (paths : Array String)
  | duplicateMetadata (module : Lean.Name)
  | uncoveredSource (module : Lean.Name) (path : String)
  | unclassifiedModule (module : Lean.Name)
  | unknownFirstPartyImport (source imported : Lean.Name)
  deriving Repr, BEq

private structure DirectoryVisit where
  canonical : FilePath
  logical : FilePath

private def nameLess (left right : Lean.Name) : Bool :=
  left.toString < right.toString

private def uniqueSortedNames (names : Array Lean.Name) : Array Lean.Name :=
  (names.qsort nameLess).foldl (init := #[]) fun result name =>
    if result.contains name then result else result.push name

private def moduleRecord? (modules : Array ModuleRecord) (name : Lean.Name) : Option ModuleRecord :=
  modules.find? (·.name == name)

private def issueKey : InventoryIssue → String
  | .escapingSource path => s!"escaping-source\u0000{path}"
  | .duplicateSource module paths =>
      s!"duplicate-source\u0000{module}\u0000{String.intercalate "\u0000" paths.toList}"
  | .duplicateMetadata module => s!"duplicate-metadata\u0000{module}"
  | .uncoveredSource module path => s!"uncovered-source\u0000{module}\u0000{path}"
  | .unclassifiedModule module => s!"unclassified-module\u0000{module}"
  | .unknownFirstPartyImport source imported =>
      s!"unknown-first-party-import\u0000{source}\u0000{imported}"

private def issueLess (left right : InventoryIssue) : Bool :=
  issueKey left < issueKey right

private def duplicateSourceIssues (sources : Array SourceRecord) : Array InventoryIssue := Id.run do
  let mut issues := #[]
  let moduleNames := uniqueSortedNames <| sources.map (·.module)
  for module in moduleNames do
    let paths := (sources.filter (·.module == module)).map (·.path) |>.qsort (· < ·)
    if paths.size > 1 then
      issues := issues.push (.duplicateSource module paths)
  return issues

private def duplicateMetadataIssues (modules : Array ModuleRecord) : Array InventoryIssue := Id.run do
  let mut issues := #[]
  let moduleNames := uniqueSortedNames <| modules.map (·.name)
  for module in moduleNames do
    if (modules.filter (·.name == module)).size > 1 then
      issues := issues.push (.duplicateMetadata module)
  return issues

/-- Validate source containment, classification, and qualified identity before loading metadata. -/
def validateSources
    (policy : InventoryPolicy) (sources : Array SourceRecord) : Array InventoryIssue := Id.run do
  let mut issues := duplicateSourceIssues sources
  for source in sources do
    unless source.contained do
      issues := issues.push (.escapingSource source.path)
    unless policy.isClassified source.module do
      issues := issues.push (.unclassifiedModule source.module)
  return (issues.qsort issueLess).foldl (init := #[]) fun unique issue =>
    if unique.contains issue then unique else unique.push issue

/--
Reconcile an owned-source inventory with loaded direct-import metadata.

Every discrepancy is retained and sorted, so one check reports all independently actionable
inventory failures instead of stopping at the first one.
-/
def reconcile
    (policy : InventoryPolicy)
    (sources : Array SourceRecord)
    (modules : Array ModuleRecord) : Array InventoryIssue := Id.run do
  let mut issues := validateSources policy sources ++ duplicateMetadataIssues modules
  for source in sources do
    if (moduleRecord? modules source.module).isNone then
      issues := issues.push (.uncoveredSource source.module source.path)
  for record in modules do
    unless policy.isClassified record.name do
      issues := issues.push (.unclassifiedModule record.name)
    for imported in uniqueSortedNames record.imports do
      if policy.isFirstParty imported && (moduleRecord? modules imported).isNone then
        issues := issues.push (.unknownFirstPartyImport record.name imported)
  return (issues.qsort issueLess).foldl (init := #[]) fun unique issue =>
    if unique.contains issue then unique else unique.push issue

private def canonicalPath (path : FilePath) : IO FilePath :=
  return (← IO.FS.realPath path).normalize

private def sameFilesystemObject (left right : FilePath) : IO Bool := do
  let child ← IO.Process.spawn {
    cmd := "sh"
    args := #["-c", "test \"$1\" -ef \"$2\"", "lean-source-inventory", left.toString,
      right.toString]
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
    (excludedDirectories : Array String)
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
        let (nested, nestedVisited) ← scanDirectory root entry.path excludedDirectories visited
        sources := sources ++ nested
        visited := nestedVisited
      else if targetMetadata.type == .file && entry.path.extension == some "lean" then
        sources := sources.push {
          path := entry.path.toString
          module := ← moduleNameForSource root entry.path
          contained := true
        }
  pure (sources, visited)

/-- Inventory Lean sources below `sourceRoot`, rejecting path aliases, cycles, and escapes. -/
def scanSources
    (sourceRoot : FilePath) (excludedDirectories : Array String := #[]) : IO (Array SourceRecord) := do
  let root ← canonicalPath sourceRoot
  let (sources, _) ← scanDirectory root root excludedDirectories #[]
  pure <| sources.qsort fun left right => left.module.toString < right.module.toString

/-- Inventory the canonical current Lake package root used by a tooling executable. -/
def canonicalPackageSources (excludedDirectories : Array String := #[]) : IO (Array SourceRecord) := do
  let root ← canonicalPath (← IO.currentDir)
  unless (← (root / "lakefile.toml").pathExists) do
    throw <| IO.userError s!"canonical package root has no lakefile.toml: {root}"
  scanSources root excludedDirectories

end Tools.LeanSourceInventory
