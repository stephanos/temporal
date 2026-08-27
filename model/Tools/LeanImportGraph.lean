import Lean.Data.Name

/-!
Reusable deterministic dependency-graph checking.

Callers provide qualified module records and a small rule set. The linter reports every forbidden
transitive dependency, using cycle-safe breadth-first traversal and lexical tie-breaking to select
one stable shortest path per source and destination. It performs no filesystem or process I/O.
-/

namespace Tools.LeanImportGraph

/-- One module and the qualified names it imports directly. -/
structure ModuleRecord where
  name : Lean.Name
  imports : Array Lean.Name
  deriving Repr, BEq

/-- Caller-owned dependency policy and stable rule identity. -/
structure RuleSet (Rule : Type) where
  forbidden? : Lean.Name → Lean.Name → Option Rule
  ruleKey : Rule → String

/-- One forbidden reachability result and its selected shortest qualified path. -/
structure Violation (Rule : Type) where
  rule : Rule
  source : Lean.Name
  destination : Lean.Name
  path : Array Lean.Name
  deriving Repr, BEq

private def nameLess (left right : Lean.Name) : Bool :=
  left.toString < right.toString

private def pathText (path : Array Lean.Name) : String :=
  " -> ".intercalate <| path.toList.map (·.toString)

private def moduleRecord? (modules : Array ModuleRecord) (name : Lean.Name) : Option ModuleRecord :=
  modules.find? (·.name == name)

private def uniqueSortedNames (names : Array Lean.Name) : Array Lean.Name :=
  (names.qsort nameLess).foldl (init := #[]) fun result name =>
    if result.contains name then result else result.push name

private def ownedImports (modules : Array ModuleRecord) (record : ModuleRecord) : Array Lean.Name :=
  uniqueSortedNames <| record.imports.filter fun imported => (moduleRecord? modules imported).isSome

private def shortestPathsFrom
    (modules : Array ModuleRecord) (source : Lean.Name) : Array (Lean.Name × Array Lean.Name) := Id.run do
  let mut queue : Array (Array Lean.Name) := #[#[source]]
  let mut visited : Array Lean.Name := #[source]
  let mut paths : Array (Lean.Name × Array Lean.Name) := #[]
  let mut cursor := 0
  while cursor < queue.size do
    let path := queue[cursor]!
    cursor := cursor + 1
    let current := path.back!
    if current != source then
      paths := paths.push (current, path)
    if let some record := moduleRecord? modules current then
      for imported in ownedImports modules record do
        unless visited.contains imported do
          visited := visited.push imported
          queue := queue.push (path.push imported)
  return paths

private def violationLess
    (ruleSet : RuleSet Rule) (left right : Violation Rule) : Bool :=
  let leftKey :=
    s!"{ruleSet.ruleKey left.rule}\u0000{left.source}\u0000{left.destination}\u0000{pathText left.path}"
  let rightKey :=
    s!"{ruleSet.ruleKey right.rule}\u0000{right.source}\u0000{right.destination}\u0000{pathText right.path}"
  leftKey < rightKey

/--
Return every forbidden transitive dependency in deterministic order.

Imports absent from `modules` are external leaves and are intentionally not traversed. Inventory
completeness and module classification remain caller-owned concerns.
-/
def check (ruleSet : RuleSet Rule) (modules : Array ModuleRecord) : Array (Violation Rule) := Id.run do
  let mut violations := #[]
  let modules := modules.qsort fun left right => nameLess left.name right.name
  for sourceRecord in modules do
    for (destination, path) in shortestPathsFrom modules sourceRecord.name do
      if let some rule := ruleSet.forbidden? sourceRecord.name destination then
        violations := violations.push {
          rule
          source := sourceRecord.name
          destination
          path
        }
  return violations.qsort (violationLess ruleSet)

end Tools.LeanImportGraph
