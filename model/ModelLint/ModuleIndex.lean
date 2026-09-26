import ModelLint.ImportGraph
import Lean.Data.Json

/-!
# The module impact index

A pure projection of the reconciled first-party module graph into one closed, deterministic JSON
document (`temporal-model-module-index/v1`): one row per owned source with its classification, its
direct and reverse first-party imports, and which configured public facades and focused test roots
reach it. It is a navigation aid for a reader about to change a module, and nothing more: no
behavior fingerprint, Definition ID or semantic catalog claim is derived from it, and a facade or
test root that reaches a module says that compiling the module affects that root, not that a suite
ran.

Everything here is pure. The loader (`ModelLint.PackageModules`) supplies the sources and the
compiled import metadata, external modules included; this module validates them, refuses anything
it would otherwise have to guess about, and renders. What is rejected is rejected whole: a duplicate
identity, an unknown root or a cycle yields issues and no index, never a row that quietly dropped
the problem.
-/

namespace ModelLint.ModuleIndex

open Lean
open Tools.LeanImportGraph (ModuleRecord)
open Tools.LeanSourceInventory (SourceRecord)
open ModelLint.ImportGraph (Policy ModuleClass)

/-- The format every emitted document names. -/
def formatVersion : String := "temporal-model-module-index/v1"

/-- The roots whose reachability the index projects. They are explicit reviewed policy: a module
becomes a facade or a test root by being listed here, never by its file name. -/
structure IndexPolicy where
  publicFacades : Array Name
  focusedTests : Array Name
  deriving Repr, BEq

/-- The v1 roots. Every entry is a module the tree carries; a root that goes missing is reported by
`build` as `unknown-root` rather than silently dropped, so this list moves only by review. -/
def defaultIndexPolicy : IndexPolicy := {
  publicFacades := #[
    `Shared,
    `Temporal,
    `Temporal.API,
    `Temporal.Case.Syntax,
    `Temporal.DynamicConfig,
    `Temporal.Feature,
    `Temporal.Feature.Nexus,
    `Temporal.System,
    `Temporal.System.Configuration,
    `Temporal.Testpilot,
    `Testpilot,
    `Testpilot.Authoring,
    `Testpilot.ProtoJSON,
    `Testpilot.Protocol,
    `Umpire,
    `Umpire.Artifact,
    `Umpire.Case,
    `Umpire.Case.Compiler,
    `Umpire.Command,
    `Umpire.Core,
    `Umpire.Evidence,
    `Umpire.Exploration,
    `Umpire.ImplementationLink,
    `Umpire.Inventory,
    `Umpire.Json,
    `Umpire.KnownGap,
    `Umpire.Model,
    `Umpire.Model.Check,
    `Umpire.OutcomeClassification,
    `Umpire.Promotion,
    `Umpire.Property,
    `Umpire.Query,
    `Umpire.Scenario,
    `Umpire.Search,
    `Umpire.Variations
  ]
  focusedTests := #[
    `ModelLint.ImportGraphTests,
    `Temporal.Tool.InventoryMainTests,
    `Temporal.Tool.InventoryMakeTestsMain,
    `Temporal.Tool.InventoryTests,
    `TemporalModelTests,
    `Testpilot.Tests,
    `Testpilot.Tests.ProtoJSONMain,
    `Umpire.Case.CorrelatedTests,
    `Umpire.Evidence.Tests,
    `Umpire.Model.CheckImportTests,
    `Umpire.Property.Tests.Correlated,
    `Umpire.Search.SemanticsImportTests,
    `UmpireTests
  ]
}

/-- The v1 spelling of each policy class. The match is exhaustive, so a class added to the policy
without a spelling here does not compile rather than rendering as something invented. -/
def classificationName : ModuleClass → String
  | .shared => "shared"
  | .testpilot => "testpilot"
  | .umpire => "umpire"
  | .temporalShared => "temporal-shared"
  | .temporalFeature => "temporal-feature"
  | .temporalSystem => "temporal-system"
  | .temporalTool => "temporal-tool"
  | .temporal => "temporal"
  | .modelTests => "model-tests"
  | .lintInfrastructure => "lint-infrastructure"

/-- One reason the inputs are not an index. Each carries the identity it is about and nothing
else, so two issues about one module sort together whatever wording they carry. -/
inductive Issue where
  | duplicateSource (module : Name) (paths : Array String)
  | duplicateMetadata (module : Name)
  | uncoveredSource (module : Name) (path : String)
  | missingSource (module : Name)
  | unclassifiedModule (module : Name)
  | unknownFirstPartyImport (source imported : Name)
  | unsafePath (module : Name) (path : String)
  | unknownRoot (root : Name)
  | cycle (path : Array Name)
  deriving Repr, BEq

private def diagnosticPrefix (kind : String) : String := s!"[model-module-index/{kind}]"

def Issue.render : Issue → String
  | .duplicateSource module paths =>
      s!"{diagnosticPrefix "duplicate-source"} {module}: {", ".intercalate paths.toList}"
  | .duplicateMetadata module => s!"{diagnosticPrefix "duplicate-metadata"} {module}"
  | .uncoveredSource module path =>
      s!"{diagnosticPrefix "uncovered-source"} no loaded metadata for {module}: {path}"
  | .missingSource module =>
      s!"{diagnosticPrefix "missing-source"} first-party metadata without an owned source: {module}"
  | .unclassifiedModule module =>
      s!"{diagnosticPrefix "unclassified-module"} unclassified first-party module: {module}"
  | .unknownFirstPartyImport source imported =>
      s!"{diagnosticPrefix "unknown-import"} {source} imports unknown first-party module {imported}"
  | .unsafePath module path =>
      s!"{diagnosticPrefix "unsafe-path"} {module}: {path}"
  | .unknownRoot root => s!"{diagnosticPrefix "unknown-root"} configured root is not a first-party module: {root}"
  | .cycle path =>
      s!"{diagnosticPrefix "cycle"} {" -> ".intercalate (path.toList.map (·.toString))}"

private def issueKey : Issue → String
  | .duplicateSource module paths =>
      s!"duplicate-source\u0000{module}\u0000{"\u0000".intercalate paths.toList}"
  | .duplicateMetadata module => s!"duplicate-metadata\u0000{module}"
  | .uncoveredSource module path => s!"uncovered-source\u0000{module}\u0000{path}"
  | .missingSource module => s!"missing-source\u0000{module}"
  | .unclassifiedModule module => s!"unclassified-module\u0000{module}"
  | .unknownFirstPartyImport source imported => s!"unknown-import\u0000{source}\u0000{imported}"
  | .unsafePath module path => s!"unsafe-path\u0000{module}\u0000{path}"
  | .unknownRoot root => s!"unknown-root\u0000{root}"
  | .cycle path => s!"cycle\u0000{"\u0000".intercalate (path.toList.map (·.toString))}"

/-- One emitted row. Every array is sorted and free of repeats, and every array is present, empty
or not: the document is closed, and a reader never asks whether a missing key meant "none". -/
structure Row where
  name : Name
  sourcePath : String
  classification : String
  directDependencies : Array Name
  reverseDependencies : Array Name
  publicFacades : Array Name
  focusedTests : Array Name
  deriving Repr, BEq

structure Index where
  format : String := formatVersion
  modules : Array Row
  deriving Repr, BEq

private def nameLess (left right : Name) : Bool := left.toString < right.toString

private def uniqueSortedNames (names : Array Name) : Array Name :=
  (names.qsort nameLess).foldl (init := #[]) fun result name =>
    if result.back? == some name then result else result.push name

private def uniqueSortedIssues (issues : Array Issue) : Array Issue :=
  (issues.qsort fun left right => issueKey left < issueKey right).foldl (init := #[])
    fun result issue => if result.back? == some issue then result else result.push issue

/-! ### Paths

A source path is emitted relative to the package root with forward slashes, so two checkouts of one
tree produce one document. A backslash spelling and a leading `./` are the same path said
differently and normalize; a path that leaves the root, names a drive, starts at `/` or contains an
empty segment is not a spelling of an owned source and is refused. -/

private def hasDriveLetter (path : String) : Bool :=
  match path.toList with
  | letter :: ':' :: _ => letter.isAlpha
  | _ => false

private partial def stripDotPrefix (path : String) : String :=
  if path.startsWith "./" then stripDotPrefix (path.drop 2).copy else path

/-- The canonical spelling of an owned source path, or `none` for one that cannot be owned. -/
def normalizePath? (path : String) : Option String :=
  let forward := String.map (fun character => if character == '\\' then '/' else character) path
  let stripped := stripDotPrefix forward
  if stripped.isEmpty || stripped.startsWith "/" || hasDriveLetter stripped then none
  else
    let segments := stripped.splitOn "/"
    if segments.any (fun segment => segment.isEmpty || segment == "." || segment == "..") then none
    else some stripped

/-- The loader reports each source at its canonical absolute path; the document names it relative
to the package root. A path the root does not contain is left as it is, so `build` refuses it as
unsafe rather than this function guessing where it belongs. -/
def relativizeSources (root : String) (sources : Array SourceRecord) : Array SourceRecord :=
  let forward (path : String) : String :=
    String.map (fun character => if character == '\\' then '/' else character) path
  let rootPrefix :=
    let root := forward root
    if root.endsWith "/" then root else root ++ "/"
  sources.map fun source =>
    let path := forward source.path
    if rootPrefix.isPrefixOf path then { source with path := (path.drop rootPrefix.length).copy }
    else source

/-! ### Building the index -/

private structure Validated where
  rows : Array (Name × String × ModuleClass)
  imports : Std.HashMap Name (Array Name)
  firstParty : Std.HashSet Name

/-- Validate the inputs, refusing every discrepancy rather than the first, and refusing them all
rather than indexing around one. -/
private def validate (policy : Policy) (sources : Array SourceRecord)
    (modules : Array ModuleRecord) : Except (Array Issue) Validated := Id.run do
  let mut issues : Array Issue := #[]
  -- Sources: one path per module, every path an owned spelling, every module classified.
  let mut sourcePaths : Std.HashMap Name String := {}
  for module in uniqueSortedNames (sources.map (·.module)) do
    let paths := (sources.filter (·.module == module)).map (·.path) |>.qsort (· < ·)
    if paths.size > 1 then
      issues := issues.push (.duplicateSource module paths)
    else if let some path := paths[0]? then
      match normalizePath? path with
      | some normalized => sourcePaths := sourcePaths.insert module normalized
      | none => issues := issues.push (.unsafePath module path)
    if (policy.classify? module).isNone then
      issues := issues.push (.unclassifiedModule module)
  -- Metadata: one record per module, every first-party import known.
  let mut imports : Std.HashMap Name (Array Name) := {}
  let mut recorded : Std.HashSet Name := {}
  for module in uniqueSortedNames (modules.map (·.name)) do
    if (modules.filter (·.name == module)).size > 1 then
      issues := issues.push (.duplicateMetadata module)
    recorded := recorded.insert module
  for record in modules do
    -- A compiled header lists a module once per import modifier it was imported under (`public
    -- import` and `meta import` of one module are two entries), so a repeated name is the
    -- toolchain's spelling of one edge and normalizes; only the identities are held to one each.
    let directImports := uniqueSortedNames record.imports
    for imported in directImports do
      if policy.isFirstParty imported && !recorded.contains imported then
        issues := issues.push (.unknownFirstPartyImport record.name imported)
    imports := imports.insert record.name directImports
    if policy.isFirstParty record.name then
      if (policy.classify? record.name).isNone then
        issues := issues.push (.unclassifiedModule record.name)
      unless sourcePaths.contains record.name || sources.any (·.module == record.name) do
        issues := issues.push (.missingSource record.name)
  for source in sources do
    unless recorded.contains source.module do
      issues := issues.push (.uncoveredSource source.module source.path)
  let sortedIssues := uniqueSortedIssues issues
  if !sortedIssues.isEmpty then return .error sortedIssues
  let mut rows : Array (Name × String × ModuleClass) := #[]
  let mut firstParty : Std.HashSet Name := {}
  for source in sources do
    if let some path := sourcePaths[source.module]? then
      if let some moduleClass := policy.classify? source.module then
        rows := rows.push (source.module, path, moduleClass)
        firstParty := firstParty.insert source.module
  return .ok { rows, imports, firstParty }

/-- The first cycle a depth-first walk of the whole graph meets, as the path that closes it. Lean
cannot compile one, so meeting one means the metadata is not what a build produced. -/
private def findCycle (imports : Std.HashMap Name (Array Name)) (starts : Array Name) :
    Option (Array Name) := Id.run do
  let mut finished : Std.HashSet Name := {}
  for start in starts do
    if finished.contains start then continue
    -- An explicit stack of (module, next import index) keeps the walk iterative; the path is the
    -- stack's modules, so a back edge to one of them is a cycle read straight off it.
    let mut stack : Array (Name × Nat) := #[(start, 0)]
    let mut onPath : Std.HashSet Name := {start}
    while !stack.isEmpty do
      let (current, index) := stack.back!
      let directImports := imports.getD current #[]
      if let some imported := directImports[index]? then
        stack := stack.pop.push (current, index + 1)
        if onPath.contains imported then
          let path := (stack.map (·.1)).toList.dropWhile (· != imported)
          return some (path.toArray.push imported)
        unless finished.contains imported do
          stack := stack.push (imported, 0)
          onPath := onPath.insert imported
      else
        stack := stack.pop
        onPath := onPath.erase current
        finished := finished.insert current
  return none

/-- Every module a root reaches, itself included, over the whole validated graph: an external
module on the way is walked through, because compiling what it imports still affects the root. -/
private def reachableFrom (imports : Std.HashMap Name (Array Name)) (root : Name) : Array Name :=
  Id.run do
    let mut visited : Std.HashSet Name := {root}
    let mut pending : Array Name := #[root]
    let mut reached : Array Name := #[]
    while let some current := pending.back? do
      pending := pending.pop
      reached := reached.push current
      for imported in imports.getD current #[] do
        unless visited.contains imported do
          visited := visited.insert imported
          pending := pending.push imported
    return reached

/-- Build the index or report why the inputs are not one. -/
def build (policy : Policy) (roots : IndexPolicy) (sources : Array SourceRecord)
    (modules : Array ModuleRecord) : Except (Array Issue) Index := do
  let validated ← validate policy sources modules
  let mut issues : Array Issue := #[]
  for root in roots.publicFacades ++ roots.focusedTests do
    unless validated.firstParty.contains root do
      issues := issues.push (.unknownRoot root)
  if let some path := findCycle validated.imports (modules.map (·.name)) then
    issues := issues.push (.cycle path)
  let sortedIssues := uniqueSortedIssues issues
  if !sortedIssues.isEmpty then throw sortedIssues
  -- One reverse adjacency for the whole graph, built once from the direct first-party edges.
  let mut reverse : Std.HashMap Name (Array Name) := {}
  for (name, _, _) in validated.rows do
    for imported in validated.imports.getD name #[] do
      if validated.firstParty.contains imported then
        reverse := reverse.insert imported ((reverse.getD imported #[]).push name)
  -- Each root's descendants, credited to the modules it reaches: one walk per root, never one per
  -- pair of modules.
  let mut facades : Std.HashMap Name (Array Name) := {}
  for root in roots.publicFacades do
    for reached in reachableFrom validated.imports root do
      facades := facades.insert reached ((facades.getD reached #[]).push root)
  let mut tests : Std.HashMap Name (Array Name) := {}
  for root in roots.focusedTests do
    for reached in reachableFrom validated.imports root do
      tests := tests.insert reached ((tests.getD reached #[]).push root)
  let rows := validated.rows.map fun (name, sourcePath, moduleClass) => {
    name
    sourcePath
    classification := classificationName moduleClass
    directDependencies :=
      (validated.imports.getD name #[]).filter validated.firstParty.contains
    reverseDependencies := uniqueSortedNames (reverse.getD name #[])
    publicFacades := uniqueSortedNames (facades.getD name #[])
    focusedTests := uniqueSortedNames (tests.getD name #[])
    : Row }
  pure { modules := rows.qsort fun left right => nameLess left.name right.name }

/-! ### Rendering

The document is written by hand rather than through a generic object, because a generic JSON object
orders keys its own way and the closed field order is part of the contract. Strings are escaped by
Lean's own JSON string rendering. -/

private def jsonString (value : String) : String := (Json.str value).compress

private def jsonNames (names : Array Name) : String :=
  "[" ++ ",".intercalate (names.toList.map fun name => jsonString name.toString) ++ "]"

private def renderRow (row : Row) : String :=
  "{\"name\":" ++ jsonString row.name.toString ++
    ",\"sourcePath\":" ++ jsonString row.sourcePath ++
    ",\"classification\":" ++ jsonString row.classification ++
    ",\"directDependencies\":" ++ jsonNames row.directDependencies ++
    ",\"reverseDependencies\":" ++ jsonNames row.reverseDependencies ++
    ",\"publicFacades\":" ++ jsonNames row.publicFacades ++
    ",\"focusedTests\":" ++ jsonNames row.focusedTests ++ "}"

/-- The complete document: one compact JSON object and one terminal LF. -/
def render (index : Index) : String :=
  "{\"format\":" ++ jsonString index.format ++ ",\"modules\":[" ++
    ",".intercalate (index.modules.toList.map renderRow) ++ "]}\n"

end ModelLint.ModuleIndex
