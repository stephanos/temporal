import Lean.Data.Name

/-!
Reusable deterministic dependency-graph checking.

Callers provide qualified module records and a forbidden-dependency predicate. The checker reports
every forbidden transitive dependency, using cycle-safe breadth-first traversal and lexical
tie-breaking to select one stable shortest path per source and destination. It performs no
filesystem or process I/O.
-/

namespace Tools.LeanImportGraph

/-- One module and the qualified names it imports directly. -/
structure ModuleRecord where
  name : Lean.Name
  imports : Array Lean.Name
  deriving Repr, BEq

/-- One forbidden reachability result and its selected shortest qualified path. -/
structure Violation (Rule : Type) where
  rule : Rule
  source : Lean.Name
  destination : Lean.Name
  path : Array Lean.Name
  deriving Repr, BEq

private def nameLess (left right : Lean.Name) : Bool :=
  left.toString < right.toString

private def moduleRecord? (modules : Array ModuleRecord) (name : Lean.Name) : Option ModuleRecord :=
  modules.find? (·.name == name)

private def uniqueSortedNames (names : Array Lean.Name) : Array Lean.Name :=
  (names.qsort nameLess).foldl (init := #[]) fun result name =>
    if result.back? == some name then result else result.push name

private def ownedImports (modules : Array ModuleRecord) (record : ModuleRecord) : Array Lean.Name :=
  uniqueSortedNames <| record.imports.filter fun imported => (moduleRecord? modules imported).isSome

private def shortestPathsFrom
    (modules : Array ModuleRecord) (source : Lean.Name) : Array (Lean.Name × Array Lean.Name) := Id.run do
  let mut queue : Std.Queue (Array Lean.Name) :=
    (Std.Queue.empty : Std.Queue (Array Lean.Name)).enqueue #[source]
  let mut visited : Array Lean.Name := #[source]
  let mut paths : Array (Lean.Name × Array Lean.Name) := #[]
  while let some (path, remaining) := queue.dequeue? do
    queue := remaining
    let current := path.back!
    if current != source then
      paths := paths.push (current, path)
    if let some record := moduleRecord? modules current then
      for imported in ownedImports modules record do
        unless visited.contains imported do
          visited := visited.push imported
          queue := queue.enqueue (path.push imported)
  return paths

private def violationLess (left right : Violation Rule) : Bool :=
  let leftKey := s!"{left.source}\u0000{left.destination}"
  let rightKey := s!"{right.source}\u0000{right.destination}"
  leftKey < rightKey

/--
Return every forbidden transitive dependency in deterministic order.

Imports absent from `modules` are external leaves and are intentionally not traversed. Inventory
completeness and module classification remain caller-owned concerns.
-/
def check
    (forbidden? : Lean.Name → Lean.Name → Option Rule)
    (modules : Array ModuleRecord) : Array (Violation Rule) := Id.run do
  let mut violations := #[]
  let modules := modules.qsort fun left right => nameLess left.name right.name
  for sourceRecord in modules do
    for (destination, path) in shortestPathsFrom modules sourceRecord.name do
      if let some rule := forbidden? sourceRecord.name destination then
        violations := violations.push {
          rule
          source := sourceRecord.name
          destination
          path
        }
  return violations.qsort violationLess

end Tools.LeanImportGraph
