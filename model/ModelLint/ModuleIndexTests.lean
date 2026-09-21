import ModelLint.ModuleIndex

/-!
# What the module index says, and what it refuses to say

These tests pin the pure projection: which rows come out of which graph, in which order, with which
bytes. They build synthetic graphs of real classified names, because `defaultPolicy` refuses a name
it cannot classify and a fixture on `Alpha` would test the refusal rather than the index.

Reachability is the part worth pinning carefully. A root reaches itself; it reaches through an
external module without that module becoming a row or an edge; two roots that reach one module both
appear on its row, once each, sorted; a module nobody reaches has empty root arrays rather than a
missing key.
-/

open Lean
open ModelLint.ImportGraph (defaultPolicy ModuleClass)
open ModelLint.ModuleIndex
open Tools.LeanImportGraph (ModuleRecord)
open Tools.LeanSourceInventory (SourceRecord)

namespace ModelLint.ModuleIndexTests

private def requireEqual [BEq α] [Repr α] (label : String) (actual expected : α) : IO Unit :=
  unless actual == expected do
    throw <| IO.userError s!"{label}: expected {repr expected}, got {repr actual}"

private def moduleRecord (name : Name) (imports : Array Name := #[]) : ModuleRecord :=
  { name, imports }

private def source (module : Name) (path := s!"{module}.lean") : SourceRecord :=
  { path, module }

/-- Sources for exactly the given modules, each at its conventional path. -/
private def sourcesFor (modules : Array ModuleRecord) : Array SourceRecord :=
  modules.map fun record => source record.name

private def noRoots : IndexPolicy := { publicFacades := #[], focusedTests := #[] }

private def requireIndex (label : String) (built : Except (Array Issue) Index) : IO Index :=
  match built with
  | .ok index => pure index
  | .error issues =>
      throw <| IO.userError s!"{label}: expected an index, got issues: \
        {issues.toList.map Issue.render}"

private def requireIssues
    (label : String) (built : Except (Array Issue) Index) (expected : Array Issue) : IO Unit :=
  match built with
  | .ok index => throw <| IO.userError s!"{label}: expected issues, got an index with \
      {index.modules.size} rows"
  | .error issues => requireEqual s!"{label} issues" issues expected

private def requireRow (label : String) (index : Index) (name : Name) : IO Row := do
  let some row := index.modules.find? (·.name == name)
    | throw <| IO.userError s!"{label}: no row for {name}"
  pure row

/-- Every constructor has its v1 spelling; the list is the whole inductive, so a class added to the
policy is a class added here. -/
private def testClassificationSpellings : IO Unit := do
  let spellings : Array (ModuleClass × String) := #[
    (.shared, "shared"),
    (.testpilot, "testpilot"),
    (.umpire, "umpire"),
    (.temporalShared, "temporal-shared"),
    (.temporalFeature, "temporal-feature"),
    (.temporalSystem, "temporal-system"),
    (.temporalTool, "temporal-tool"),
    (.temporal, "temporal"),
    (.modelTests, "model-tests"),
    (.lintInfrastructure, "lint-infrastructure")
  ]
  for (moduleClass, spelling) in spellings do
    requireEqual s!"spelling of {repr moduleClass}" (classificationName moduleClass) spelling
  requireEqual "distinct spellings" (spellings.map (·.2)).toList.eraseDups.length spellings.size

/-- A row carries the classification `defaultPolicy` assigns, through every class the tree uses. -/
private def testRowsCarryPolicyClassification : IO Unit := do
  let expected : Array (Name × String) := #[
    (`Shared.Probe, "shared"),
    (`Testpilot.Probe, "testpilot"),
    (`Umpire.Probe, "umpire"),
    (`Temporal.Shared.Probe, "temporal-shared"),
    (`Temporal.Feature.Probe, "temporal-feature"),
    (`Temporal.System.Probe, "temporal-system"),
    (`Temporal.Tool.Probe, "temporal-tool"),
    (`Temporal.Probe, "temporal"),
    (`UmpireTests.Probe, "model-tests"),
    (`ModelLint.Probe, "lint-infrastructure")
  ]
  let modules := expected.map fun (name, _) => moduleRecord name
  let index ← requireIndex "classified rows"
    (build defaultPolicy noRoots (sourcesFor modules) modules)
  requireEqual "row count" index.modules.size expected.size
  for (name, spelling) in expected do
    let row ← requireRow "classified rows" index name
    requireEqual s!"classification of {name}" row.classification spelling

/-- Every configured root is a classified first-party module and, in the checkout the tests run
from, an actual source file. No filename heuristic can supply a root; this is where one that
disappears from the tree is caught before the exporter refuses it. -/
private def testConfiguredRootsExist : IO Unit := do
  let roots := defaultIndexPolicy.publicFacades ++ defaultIndexPolicy.focusedTests
  requireEqual "facade root count" defaultIndexPolicy.publicFacades.size 35
  requireEqual "test root count" defaultIndexPolicy.focusedTests.size 13
  requireEqual "roots are distinct" roots.toList.eraseDups.length roots.size
  for root in roots do
    requireEqual s!"{root} is first party" (defaultPolicy.isFirstParty root) true
    requireEqual s!"{root} classifies" (defaultPolicy.classify? root).isSome true
    let path := System.FilePath.mk (root.toString.replace "." "/") |>.addExtension "lean"
    unless (← path.pathExists) do
      throw <| IO.userError s!"configured root {root} has no source at {path}"
  -- Under a synthetic tree holding exactly the roots, each root reaches itself and nothing else.
  let modules := roots.map fun root => moduleRecord root
  let index ← requireIndex "roots alone"
    (build defaultPolicy defaultIndexPolicy (sourcesFor modules) modules)
  requireEqual "roots alone row count" index.modules.size roots.size
  for facade in defaultIndexPolicy.publicFacades do
    let row ← requireRow "roots alone" index facade
    requireEqual s!"{facade} reaches itself" row.publicFacades #[facade]
    requireEqual s!"{facade} is no test" row.focusedTests #[]
  for test in defaultIndexPolicy.focusedTests do
    let row ← requireRow "roots alone" index test
    requireEqual s!"{test} reaches itself" row.focusedTests #[test]
    requireEqual s!"{test} is no facade" row.publicFacades #[]

/--
The reachability shapes, on one graph:

```
Umpire.Facade ─┬─> Umpire.Left ──┐
               └─> Umpire.Right ─┴─> Umpire.Deep      (diamond: Deep is credited once)
Umpire.Other  ───> Umpire.Deep                        (a second facade on the same module)
UmpireTests.Suite ─> Umpire.Left                      (a test root; Deep inherits it)
Umpire.Island                                          (disconnected: nothing reaches it)
```
-/
private def testReachabilityShapes : IO Unit := do
  let modules := #[
    moduleRecord `Umpire.Facade #[`Umpire.Right, `Umpire.Left],
    moduleRecord `Umpire.Left #[`Umpire.Deep],
    moduleRecord `Umpire.Right #[`Umpire.Deep],
    moduleRecord `Umpire.Deep,
    moduleRecord `Umpire.Other #[`Umpire.Deep],
    moduleRecord `UmpireTests.Suite #[`Umpire.Left],
    moduleRecord `Umpire.Island
  ]
  let roots : IndexPolicy := {
    publicFacades := #[`Umpire.Other, `Umpire.Facade]
    focusedTests := #[`UmpireTests.Suite]
  }
  let index ← requireIndex "shapes" (build defaultPolicy roots (sourcesFor modules) modules)
  requireEqual "rows sorted by name" (index.modules.map (·.name)) #[
    `Umpire.Deep, `Umpire.Facade, `Umpire.Island, `Umpire.Left, `Umpire.Other, `Umpire.Right,
    `UmpireTests.Suite
  ]
  let facade ← requireRow "shapes" index `Umpire.Facade
  requireEqual "facade direct imports sorted" facade.directDependencies
    #[`Umpire.Left, `Umpire.Right]
  requireEqual "facade has no importer" facade.reverseDependencies #[]
  requireEqual "facade reaches itself" facade.publicFacades #[`Umpire.Facade]
  requireEqual "facade is reached by no test" facade.focusedTests #[]
  let deep ← requireRow "shapes" index `Umpire.Deep
  requireEqual "deep imports nothing" deep.directDependencies #[]
  requireEqual "deep importers sorted" deep.reverseDependencies
    #[`Umpire.Left, `Umpire.Other, `Umpire.Right]
  requireEqual "diamond and multi-root credit each facade once, sorted" deep.publicFacades
    #[`Umpire.Facade, `Umpire.Other]
  requireEqual "deep inherits the test root" deep.focusedTests #[`UmpireTests.Suite]
  let left ← requireRow "shapes" index `Umpire.Left
  requireEqual "left importers" left.reverseDependencies #[`Umpire.Facade, `UmpireTests.Suite]
  requireEqual "left facades" left.publicFacades #[`Umpire.Facade]
  requireEqual "left tests" left.focusedTests #[`UmpireTests.Suite]
  let island ← requireRow "shapes" index `Umpire.Island
  requireEqual "island imports nothing" island.directDependencies #[]
  requireEqual "island has no importer" island.reverseDependencies #[]
  requireEqual "island is reached by no facade" island.publicFacades #[]
  requireEqual "island is reached by no test" island.focusedTests #[]
  let suite ← requireRow "shapes" index `UmpireTests.Suite
  requireEqual "test root reaches itself" suite.focusedTests #[`UmpireTests.Suite]
  requireEqual "test root is no facade" suite.publicFacades #[]

/-- An external module on the path is walked through for reachability, but it is no row and no
edge: the facade's direct imports name only first-party modules, the module past the bridge has
no first-party importer, and the document mentions the external name nowhere. -/
private def testExternalBridge : IO Unit := do
  let modules := #[
    moduleRecord `Umpire.Facade #[`Lean.Data.Json, `Umpire.Near],
    moduleRecord `Lean.Data.Json #[`Umpire.Past],
    moduleRecord `Umpire.Near,
    moduleRecord `Umpire.Past
  ]
  let sources := #[source `Umpire.Facade, source `Umpire.Near, source `Umpire.Past]
  let roots : IndexPolicy := { publicFacades := #[`Umpire.Facade], focusedTests := #[] }
  let index ← requireIndex "bridge" (build defaultPolicy roots sources modules)
  requireEqual "no external row" (index.modules.map (·.name))
    #[`Umpire.Facade, `Umpire.Near, `Umpire.Past]
  let facade ← requireRow "bridge" index `Umpire.Facade
  requireEqual "no external edge" facade.directDependencies #[`Umpire.Near]
  let past ← requireRow "bridge" index `Umpire.Past
  requireEqual "reached through the bridge" past.publicFacades #[`Umpire.Facade]
  requireEqual "no importer through the bridge" past.reverseDependencies #[]
  requireEqual "external name absent from the document"
    ((render index).splitOn "Lean.Data.Json").length 1

/-- Each way the inputs can fail to be an index, alone. -/
private def testEachRejection : IO Unit := do
  let one := #[moduleRecord `Umpire.Core]
  requireIssues "duplicate source"
    (build defaultPolicy noRoots
      #[source `Umpire.Core "Umpire/Core.lean", source `Umpire.Core "Umpire/Core2.lean"] one)
    #[.duplicateSource `Umpire.Core #["Umpire/Core.lean", "Umpire/Core2.lean"]]
  requireIssues "duplicate metadata"
    (build defaultPolicy noRoots (sourcesFor one) #[moduleRecord `Umpire.Core, moduleRecord `Umpire.Core])
    #[.duplicateMetadata `Umpire.Core]
  requireIssues "uncovered source"
    (build defaultPolicy noRoots #[source `Umpire.Core, source `Umpire.Extra] one)
    #[.uncoveredSource `Umpire.Extra "Umpire.Extra.lean"]
  requireIssues "missing source"
    (build defaultPolicy noRoots (sourcesFor one) #[moduleRecord `Umpire.Core, moduleRecord `Umpire.Extra])
    #[.missingSource `Umpire.Extra]
  requireIssues "unclassified module"
    (build defaultPolicy noRoots #[source `Alpha] #[moduleRecord `Alpha])
    #[.unclassifiedModule `Alpha]
  requireIssues "unknown first-party import"
    (build defaultPolicy noRoots (sourcesFor one) #[moduleRecord `Umpire.Core #[`Umpire.Missing]])
    #[.unknownFirstPartyImport `Umpire.Core `Umpire.Missing]
  -- A compiled header lists a module once per import modifier (`public import` and `meta import`
  -- of one module are two entries); that is one edge, first-party or external.
  let twice := #[
    moduleRecord `Umpire.Core #[`Umpire.Deep, `Init, `Umpire.Deep, `Init],
    moduleRecord `Umpire.Deep,
    moduleRecord `Init
  ]
  let index ← requireIndex "import listed under two modifiers"
    (build defaultPolicy noRoots #[source `Umpire.Core, source `Umpire.Deep] twice)
  requireEqual "import listed under two modifiers rows" (index.modules.map (·.name))
    #[`Umpire.Core, `Umpire.Deep]
  let core ← requireRow "import listed under two modifiers" index `Umpire.Core
  requireEqual "one edge" core.directDependencies #[`Umpire.Deep]
  let deep ← requireRow "import listed under two modifiers" index `Umpire.Deep
  requireEqual "one reverse edge" deep.reverseDependencies #[`Umpire.Core]
  for path in #["../Umpire/Core.lean", "/Umpire/Core.lean", "C:\\Umpire\\Core.lean",
      "Umpire//Core.lean", "Umpire/./Core.lean", "", "./"] do
    requireIssues s!"unsafe path {repr path}"
      (build defaultPolicy noRoots #[source `Umpire.Core path] one)
      #[.unsafePath `Umpire.Core path]
  requireIssues "unknown roots"
    (build defaultPolicy
      { publicFacades := #[`Umpire.Absent], focusedTests := #[`UmpireTests.Absent] }
      (sourcesFor one) one)
    #[.unknownRoot `Umpire.Absent, .unknownRoot `UmpireTests.Absent]
  -- An external root is unknown too: a root is a first-party module or it is no root.
  requireIssues "external root"
    (build defaultPolicy { publicFacades := #[`Lean], focusedTests := #[] } (sourcesFor one)
      (one.push (moduleRecord `Lean)))
    #[.unknownRoot `Lean]
  let cycle := #[
    moduleRecord `Umpire.First #[`Umpire.Second],
    moduleRecord `Umpire.Second #[`Umpire.Third],
    moduleRecord `Umpire.Third #[`Umpire.Second]
  ]
  requireIssues "cycle"
    (build defaultPolicy noRoots (sourcesFor cycle) cycle)
    #[.cycle #[`Umpire.Second, `Umpire.Third, `Umpire.Second]]
  requireIssues "self cycle"
    (build defaultPolicy noRoots #[source `Umpire.Core] #[moduleRecord `Umpire.Core #[`Umpire.Core]])
    #[.cycle #[`Umpire.Core, `Umpire.Core]]

/-- Several discrepancies at once come back together, sorted, and none of them yields a row. -/
private def testAllIssuesTogether : IO Unit := do
  let built := build defaultPolicy
    { publicFacades := #[`Umpire.Absent], focusedTests := #[] }
    #[source `Umpire.Core "../Umpire/Core.lean", source `Beta, source `Umpire.Orphan]
    #[
      moduleRecord `Umpire.Core #[`Umpire.Missing, `Umpire.Gone],
      moduleRecord `Beta,
      moduleRecord `Umpire.Ghost
    ]
  requireIssues "all issues together" built #[
    .missingSource `Umpire.Ghost,
    .unclassifiedModule `Beta,
    .uncoveredSource `Umpire.Orphan "Umpire.Orphan.lean",
    .unknownFirstPartyImport `Umpire.Core `Umpire.Gone,
    .unknownFirstPartyImport `Umpire.Core `Umpire.Missing,
    .unsafePath `Umpire.Core "../Umpire/Core.lean"
  ]
  -- Root and cycle findings wait for clean inputs, so the two phases never report on each other's
  -- guesses; once the inputs are clean, they too come back together and sorted.
  let cycle := #[moduleRecord `Umpire.Core #[`Umpire.Deep], moduleRecord `Umpire.Deep #[`Umpire.Core]]
  requireIssues "root and cycle together"
    (build defaultPolicy { publicFacades := #[`Umpire.Absent], focusedTests := #[] }
      (sourcesFor cycle) cycle)
    #[.cycle #[`Umpire.Core, `Umpire.Deep, `Umpire.Core], .unknownRoot `Umpire.Absent]

/-- Every issue renders under its own diagnostic prefix. -/
private def testIssueRendering : IO Unit := do
  let renderings : Array (Issue × String) := #[
    (.duplicateSource `Umpire.Core #["a.lean", "b.lean"],
      "[model-module-index/duplicate-source] Umpire.Core: a.lean, b.lean"),
    (.duplicateMetadata `Umpire.Core, "[model-module-index/duplicate-metadata] Umpire.Core"),
    (.uncoveredSource `Umpire.Core "a.lean",
      "[model-module-index/uncovered-source] no loaded metadata for Umpire.Core: a.lean"),
    (.missingSource `Umpire.Core,
      "[model-module-index/missing-source] first-party metadata without an owned source: \
        Umpire.Core"),
    (.unclassifiedModule `Alpha,
      "[model-module-index/unclassified-module] unclassified first-party module: Alpha"),
    (.unknownFirstPartyImport `Umpire.Core `Umpire.Gone,
      "[model-module-index/unknown-import] Umpire.Core imports unknown first-party module \
        Umpire.Gone"),
    (.unsafePath `Umpire.Core "../a.lean", "[model-module-index/unsafe-path] Umpire.Core: ../a.lean"),
    (.unknownRoot `Umpire.Absent,
      "[model-module-index/unknown-root] configured root is not a first-party module: \
        Umpire.Absent"),
    (.cycle #[`Umpire.Core, `Umpire.Deep, `Umpire.Core],
      "[model-module-index/cycle] Umpire.Core -> Umpire.Deep -> Umpire.Core")
  ]
  for (issue, expected) in renderings do
    requireEqual s!"rendering of {repr issue}" issue.render expected

/-- The loader's canonical absolute paths become root-relative; a path outside the root is left
alone for `build` to refuse rather than guessed at. -/
private def testRelativizeSources : IO Unit := do
  let relativized := relativizeSources "/checkout/model" #[
    source `Umpire.Core "/checkout/model/Umpire/Core.lean",
    source `Umpire.Deep "/checkout/model/Umpire/Deep.lean",
    source `Umpire.Stray "/elsewhere/Umpire/Stray.lean"
  ]
  requireEqual "relativized paths" (relativized.map (·.path))
    #["Umpire/Core.lean", "Umpire/Deep.lean", "/elsewhere/Umpire/Stray.lean"]
  requireEqual "root with trailing separator"
    ((relativizeSources "/checkout/model/" #[source `Umpire.Core "/checkout/model/Umpire/Core.lean"]).map
      (·.path))
    #["Umpire/Core.lean"]
  requireEqual "backslash root and path"
    ((relativizeSources "C:\\checkout\\model" #[source `Umpire.Core "C:\\checkout\\model\\Umpire\\Core.lean"]).map
      (·.path))
    #["Umpire/Core.lean"]
  requireEqual "a sibling directory sharing the root's prefix is outside it"
    ((relativizeSources "/checkout/model" #[source `Umpire.Core "/checkout/model2/Umpire/Core.lean"]).map
      (·.path))
    #["/checkout/model2/Umpire/Core.lean"]
  requireIssues "stray source refused"
    (build defaultPolicy noRoots
      (relativizeSources "/checkout/model" #[source `Umpire.Core "/elsewhere/Umpire/Core.lean"])
      #[moduleRecord `Umpire.Core])
    #[.unsafePath `Umpire.Core "/elsewhere/Umpire/Core.lean"]

/-- Path spellings that name one owned source normalize to one emitted path. -/
private def testPathNormalization : IO Unit := do
  for spelling in #["Umpire/Core.lean", "./Umpire/Core.lean", ".\\Umpire\\Core.lean",
      "Umpire\\Core.lean", "././Umpire/Core.lean"] do
    requireEqual s!"normalized {repr spelling}" (normalizePath? spelling) (some "Umpire/Core.lean")
  for spelling in #["", "./", "/Umpire/Core.lean", "\\Umpire\\Core.lean", "c:/Umpire/Core.lean",
      "Umpire/../Core.lean", "Umpire//Core.lean", "Umpire/./Core.lean", ".", "..", "Umpire/"] do
    requireEqual s!"refused {repr spelling}" (normalizePath? spelling) none
  -- A single-letter first segment is a directory, not a drive.
  requireEqual "single-letter directory" (normalizePath? "c/Core.lean") (some "c/Core.lean")

/-- Reordering sources, records and imports, and respelling paths, changes no byte. -/
private def testPermutationsAreByteIdentical : IO Unit := do
  let roots : IndexPolicy := {
    publicFacades := #[`Umpire.Facade]
    focusedTests := #[`UmpireTests.Suite]
  }
  let canonical := build defaultPolicy roots
    #[
      source `Umpire.Facade "Umpire/Facade.lean",
      source `Umpire.Deep "Umpire/Deep.lean",
      source `Umpire.Left "Umpire/Left.lean",
      source `UmpireTests.Suite "UmpireTests/Suite.lean"
    ]
    #[
      moduleRecord `Umpire.Facade #[`Umpire.Left, `Umpire.Deep],
      moduleRecord `Umpire.Left #[`Umpire.Deep],
      moduleRecord `Umpire.Deep,
      moduleRecord `UmpireTests.Suite #[`Umpire.Facade]
    ]
  let permuted := build defaultPolicy
    { publicFacades := #[`Umpire.Facade], focusedTests := #[`UmpireTests.Suite] }
    #[
      source `UmpireTests.Suite ".\\UmpireTests\\Suite.lean",
      source `Umpire.Left "./Umpire/Left.lean",
      source `Umpire.Deep "Umpire\\Deep.lean",
      source `Umpire.Facade "./Umpire/Facade.lean"
    ]
    #[
      moduleRecord `UmpireTests.Suite #[`Umpire.Facade],
      moduleRecord `Umpire.Deep,
      moduleRecord `Umpire.Left #[`Umpire.Deep],
      moduleRecord `Umpire.Facade #[`Umpire.Deep, `Umpire.Left]
    ]
  let canonicalIndex ← requireIndex "canonical" canonical
  let permutedIndex ← requireIndex "permuted" permuted
  requireEqual "permuted index" permutedIndex canonicalIndex
  requireEqual "permuted bytes" (render permutedIndex) (render canonicalIndex)

/-- The exact bytes: closed field order, every array present, compact, one terminal LF. -/
private def testExactBytes : IO Unit := do
  let roots : IndexPolicy := {
    publicFacades := #[`Umpire.Facade]
    focusedTests := #[`UmpireTests.Suite]
  }
  let index ← requireIndex "bytes" (build defaultPolicy roots
    #[
      source `Umpire.Facade "Umpire/Facade.lean",
      source `Umpire.Deep "Umpire/Deep.lean",
      source `Umpire.Island "Umpire/Island.lean",
      source `UmpireTests.Suite "UmpireTests/Suite.lean"
    ]
    #[
      moduleRecord `Umpire.Facade #[`Umpire.Deep, `Lean.Data.Json],
      moduleRecord `Umpire.Deep,
      moduleRecord `Umpire.Island,
      moduleRecord `UmpireTests.Suite #[`Umpire.Deep],
      moduleRecord `Lean.Data.Json
    ])
  let expected := "{\"format\":\"temporal-model-module-index/v1\",\"modules\":[" ++
    "{\"name\":\"Umpire.Deep\",\"sourcePath\":\"Umpire/Deep.lean\",\"classification\":\"umpire\"," ++
      "\"directDependencies\":[],\"reverseDependencies\":[\"Umpire.Facade\",\"UmpireTests.Suite\"]," ++
      "\"publicFacades\":[\"Umpire.Facade\"],\"focusedTests\":[\"UmpireTests.Suite\"]}," ++
    "{\"name\":\"Umpire.Facade\",\"sourcePath\":\"Umpire/Facade.lean\",\"classification\":\"umpire\"," ++
      "\"directDependencies\":[\"Umpire.Deep\"],\"reverseDependencies\":[]," ++
      "\"publicFacades\":[\"Umpire.Facade\"],\"focusedTests\":[]}," ++
    "{\"name\":\"Umpire.Island\",\"sourcePath\":\"Umpire/Island.lean\",\"classification\":\"umpire\"," ++
      "\"directDependencies\":[],\"reverseDependencies\":[]," ++
      "\"publicFacades\":[],\"focusedTests\":[]}," ++
    "{\"name\":\"UmpireTests.Suite\",\"sourcePath\":\"UmpireTests/Suite.lean\"," ++
      "\"classification\":\"model-tests\",\"directDependencies\":[\"Umpire.Deep\"]," ++
      "\"reverseDependencies\":[],\"publicFacades\":[],\"focusedTests\":[\"UmpireTests.Suite\"]}" ++
    "]}\n"
  requireEqual "exact bytes" (render index) expected
  requireEqual "one terminal LF" ((render index).endsWith "]}\n") true
  requireEqual "empty index bytes" (render { modules := #[] })
    "{\"format\":\"temporal-model-module-index/v1\",\"modules\":[]}\n"
  -- Every byte that JSON must escape goes through the same escaping as any artifact string.
  requireEqual "escaped path" (render { modules := #[{
    name := `Umpire.Core
    sourcePath := "Umpire/\"quoted\"\\.lean"
    classification := "umpire"
    directDependencies := #[]
    reverseDependencies := #[]
    publicFacades := #[]
    focusedTests := #[]
  }] })
    ("{\"format\":\"temporal-model-module-index/v1\",\"modules\":[{\"name\":\"Umpire.Core\"," ++
      "\"sourcePath\":\"Umpire/\\\"quoted\\\"\\\\.lean\",\"classification\":\"umpire\"," ++
      "\"directDependencies\":[],\"reverseDependencies\":[],\"publicFacades\":[]," ++
      "\"focusedTests\":[]}]}\n")

/-- Roughly ten times the tree, as a layered acyclic graph with fan-out and fan-in, indexed and
rendered from every configured root count in bounded time. An all-pairs path table would be a few
million entries here; one walk per root is a few hundred thousand steps. -/
private def testTenfoldFixtureBuildsQuickly : IO Unit := do
  let count := 3600
  let name (index : Nat) : Name := (`Umpire.Synthetic).str s!"M{index}"
  let mut modules : Array ModuleRecord := #[]
  for index in [0:count] do
    let imports := #[index + 1, index + 7, index + 31, index + 199].filterMap fun target =>
      if target < count then some (name target) else none
    modules := modules.push (moduleRecord (name index) imports)
  let sources := modules.map fun record =>
    source record.name s!"Umpire/Synthetic/{record.name.components.getLast!}.lean"
  let roots : IndexPolicy := {
    publicFacades := (Array.range 35).map fun index => name (index * 3)
    focusedTests := (Array.range 13).map fun index => name (index * 5 + 1)
  }
  let started ← IO.monoMsNow
  let index ← requireIndex "tenfold" (build defaultPolicy roots sources modules)
  let bytes := render index
  let elapsed := (← IO.monoMsNow) - started
  requireEqual "tenfold row count" index.modules.size count
  requireEqual "tenfold terminal LF" (bytes.endsWith "]}\n") true
  let first ← requireRow "tenfold" index (name 0)
  requireEqual "tenfold first row facades" first.publicFacades #[name 0]
  let last ← requireRow "tenfold" index (name (count - 1))
  requireEqual "tenfold last row reached by every root" last.publicFacades.size 35
  requireEqual "tenfold last row reached by every test" last.focusedTests.size 13
  requireEqual "tenfold last row importers" last.reverseDependencies
    #[name (count - 200), name (count - 32), name (count - 8), name (count - 2)]
  IO.println s!"-- Module index: {count} synthetic modules indexed and rendered in {elapsed} ms."
  unless elapsed < 20000 do
    throw <| IO.userError s!"tenfold fixture took {elapsed} ms; expected well under 20 s"

def run : IO Unit := do
  testClassificationSpellings
  testRowsCarryPolicyClassification
  testConfiguredRootsExist
  testReachabilityShapes
  testExternalBridge
  testEachRejection
  testAllIssuesTogether
  testIssueRendering
  testRelativizeSources
  testPathNormalization
  testPermutationsAreByteIdentical
  testExactBytes
  testTenfoldFixtureBuildsQuickly
  IO.println "-- Module index synthetic tests passed."

end ModelLint.ModuleIndexTests
