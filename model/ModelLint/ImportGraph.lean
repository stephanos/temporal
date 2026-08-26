import Lean.Data.Name

/-!
Pure import-graph policy checking for the Temporal Lean model.

The checker works only with qualified Lean module names and authoritative direct imports supplied
by its caller. It classifies every first-party module through an explicit policy, reconciles source
inventory with loaded metadata, and reports all forbidden transitive paths. Breadth-first traversal
is cycle-safe; lexical neighbor ordering chooses one stable shortest path when several exist.
-/

namespace ModelLint.ImportGraph

/-- One first-party module and the qualified names it imports directly. -/
structure ModuleRecord where
  name : Lean.Name
  imports : Array Lean.Name
  deriving Repr, BEq

/-- A source discovered beneath the canonical model package root. -/
structure SourceRecord where
  path : String
  module : Lean.Name
  contained : Bool := true
  deriving Repr, BEq

/-- The policy class assigned to a first-party module. -/
inductive ModuleClass where
  | shared
  | umpire
  | umpireVeil
  | temporalFeature
  | temporalSystem
  | temporalVerify
  | temporalTool
  | temporal
  | modelTests
  | optInVerify
  | lintInfrastructure
  deriving Repr, BEq

/-- A qualified prefix and the class it assigns. Earlier entries take precedence. -/
structure Classifier where
  modulePrefix : Lean.Name
  moduleClass : ModuleClass
  exact : Bool := false
  deriving Repr, BEq

/-- Explicit module classes and exact reviewed exceptions for import-boundary checking. -/
structure Policy where
  classifiers : Array Classifier
  refinementConsumers : Array Lean.Name
  verifyConsumers : Array Lean.Name
  deriving Repr, BEq

/-- The import-boundary rules enforced by the checker. -/
inductive Rule where
  | sharedIndependence
  | umpireIndependence
  | featureIsolation
  | systemIsolation
  | verificationIsolation
  deriving Repr, BEq

/-- One forbidden reachability result and its selected shortest qualified path. -/
structure Violation where
  rule : Rule
  source : Lean.Name
  destination : Lean.Name
  path : Array Lean.Name
  deriving Repr, BEq

/-- A fail-closed discrepancy between owned sources and loaded module metadata. -/
inductive InventoryIssue where
  | escapingSource (path : String)
  | duplicateSource (module : Lean.Name) (paths : Array String)
  | duplicateMetadata (module : Lean.Name)
  | uncoveredSource (module : Lean.Name) (path : String)
  | unclassifiedModule (module : Lean.Name)
  | unknownFirstPartyImport (source imported : Lean.Name)
  deriving Repr, BEq

private def nameLess (left right : Lean.Name) : Bool :=
  left.toString < right.toString

private def classifierLess (left right : Classifier) : Bool :=
  left.modulePrefix.toString < right.modulePrefix.toString

private def matchesPrefix (modulePrefix name : Lean.Name) : Bool :=
  modulePrefix == name || modulePrefix.isPrefixOf name

/-- Classify a qualified module name, or return `none` when the policy does not own it. -/
def Policy.classify? (policy : Policy) (name : Lean.Name) : Option ModuleClass :=
  (policy.classifiers.find? fun classifier =>
    if classifier.exact then classifier.modulePrefix == name
    else matchesPrefix classifier.modulePrefix name).map (·.moduleClass)

/-- Whether a qualified module name belongs to the first-party policy. -/
def Policy.isFirstParty (policy : Policy) (name : Lean.Name) : Bool :=
  (policy.classify? name).isSome

/-- The closed import-boundary policy for the current and planned model module roots. -/
def defaultPolicy : Policy := {
  classifiers := #[
    { modulePrefix := `ModelLint, moduleClass := .lintInfrastructure },
    { modulePrefix := `Shared, moduleClass := .shared },
    { modulePrefix := `Temporal.Feature, moduleClass := .temporalFeature },
    { modulePrefix := `Temporal.System, moduleClass := .temporalSystem },
    { modulePrefix := `Temporal.Tool, moduleClass := .temporalTool },
    { modulePrefix := `Temporal.Verify, moduleClass := .temporalVerify },
    { modulePrefix := `Temporal, moduleClass := .temporal },
    { modulePrefix := `TemporalModelTests, moduleClass := .modelTests },
    { modulePrefix := `TemporalVeilTests, moduleClass := .optInVerify, exact := true },
    { modulePrefix := `TemporalVerify, moduleClass := .optInVerify, exact := true },
    { modulePrefix := `Umpire.Verify.Veil, moduleClass := .umpireVeil },
    { modulePrefix := `Umpire, moduleClass := .umpire },
    { modulePrefix := `UmpireTests, moduleClass := .modelTests }
  ],
  refinementConsumers := #[`Temporal.System.Nexus.Refinement],
  verifyConsumers := #[
    `Temporal.Feature.Nexus.CallerClosure.VeilTests,
    `Temporal.Tool.VerifyVeil,
    `TemporalVeilTests,
    `TemporalVerify
  ]
}

private def Rule.label : Rule → String
  | .sharedIndependence => "shared-independence"
  | .umpireIndependence => "umpire-independence"
  | .featureIsolation => "feature-isolation"
  | .systemIsolation => "system-isolation"
  | .verificationIsolation => "verification-isolation"

private def pathText (path : Array Lean.Name) : String :=
  " -> ".intercalate <| path.toList.map (·.toString)

/-- Render one deterministic architecture diagnostic. -/
def Violation.render (violation : Violation) : String :=
  s!"[model-import-graph/{violation.rule.label}] forbidden qualified import path: \
    {pathText violation.path}"

/-- Render one deterministic inventory or metadata diagnostic. -/
def InventoryIssue.render : InventoryIssue → String
  | .escapingSource path =>
      s!"[model-import-graph/inventory] source escapes canonical model root: {path}"
  | .duplicateSource module paths =>
      s!"[model-import-graph/inventory] duplicate module identity {module}: \
        {", ".intercalate paths.toList}"
  | .duplicateMetadata module =>
      s!"[model-import-graph/metadata] duplicate module metadata: {module}"
  | .uncoveredSource module path =>
      s!"[model-import-graph/metadata] no loaded metadata for {module}: {path}"
  | .unclassifiedModule module =>
      s!"[model-import-graph/inventory] unclassified first-party module: {module}"
  | .unknownFirstPartyImport source imported =>
      s!"[model-import-graph/metadata] {source} imports unknown first-party module {imported}"

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

private def isTemporalClass : ModuleClass → Bool
  | .temporalFeature | .temporalSystem | .temporalVerify | .temporalTool | .temporal => true
  | _ => false

private def isVerifyClass : ModuleClass → Bool
  | .temporalVerify | .umpireVeil => true
  | _ => false

private def forbiddenRule?
    (policy : Policy)
    (source : Lean.Name)
    (sourceClass : ModuleClass)
    (destinationClass : ModuleClass) : Option Rule :=
  if sourceClass == .shared &&
      (destinationClass == .umpire || destinationClass == .umpireVeil ||
        isTemporalClass destinationClass) then
    some .sharedIndependence
  else if (sourceClass == .umpire || sourceClass == .umpireVeil) &&
      isTemporalClass destinationClass then
    some .umpireIndependence
  else if sourceClass == .temporalFeature && destinationClass == .temporalSystem then
    some .featureIsolation
  else if sourceClass == .temporalFeature && isVerifyClass destinationClass &&
      !policy.verifyConsumers.contains source then
    some .featureIsolation
  else if sourceClass == .temporalSystem && destinationClass == .temporalFeature &&
      !policy.refinementConsumers.contains source then
    some .systemIsolation
  else if (sourceClass == .temporalSystem || sourceClass == .temporalTool ||
      sourceClass == .temporal || sourceClass == .modelTests || sourceClass == .umpire) &&
      isVerifyClass destinationClass && !policy.verifyConsumers.contains source then
    some .verificationIsolation
  else
    none

private def violationLess (left right : Violation) : Bool :=
  let leftKey :=
    s!"{left.rule.label}\u0000{left.source}\u0000{left.destination}\u0000{pathText left.path}"
  let rightKey :=
    s!"{right.rule.label}\u0000{right.source}\u0000{right.destination}\u0000{pathText right.path}"
  leftKey < rightKey

/--
Return every forbidden transitive reachability result in deterministic order.

The caller must first reconcile inventory and metadata. Imports outside the first-party policy are
external leaves and are intentionally not traversed.
-/
def check (policy : Policy) (modules : Array ModuleRecord) : Array Violation := Id.run do
  let mut violations := #[]
  let modules := modules.qsort fun left right => nameLess left.name right.name
  for sourceRecord in modules do
    if let some sourceClass := policy.classify? sourceRecord.name then
      for (destination, path) in shortestPathsFrom modules sourceRecord.name do
        if let some destinationClass := policy.classify? destination then
          if let some rule := forbiddenRule? policy sourceRecord.name sourceClass destinationClass then
            violations := violations.push { rule, source := sourceRecord.name, destination, path }
  return violations.qsort violationLess

private def issueLess (left right : InventoryIssue) : Bool :=
  left.render < right.render

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

/--
Reconcile a canonical owned-source inventory with loaded direct-import metadata.

Every discrepancy is retained and sorted, so one lint run reports all independently actionable
inventory failures instead of stopping at the first one.
-/
def reconcile
    (policy : Policy)
    (sources : Array SourceRecord)
    (modules : Array ModuleRecord) : Array InventoryIssue := Id.run do
  let mut issues := duplicateSourceIssues sources ++ duplicateMetadataIssues modules
  for source in sources do
    unless source.contained do
      issues := issues.push (.escapingSource source.path)
    if (moduleRecord? modules source.module).isNone then
      issues := issues.push (.uncoveredSource source.module source.path)
    if (policy.classify? source.module).isNone then
      issues := issues.push (.unclassifiedModule source.module)
  for record in modules do
    if (policy.classify? record.name).isNone then
      issues := issues.push (.unclassifiedModule record.name)
    for imported in uniqueSortedNames record.imports do
      if policy.isFirstParty imported && (moduleRecord? modules imported).isNone then
        issues := issues.push (.unknownFirstPartyImport record.name imported)
  return (issues.qsort issueLess).foldl (init := #[]) fun unique issue =>
    if unique.contains issue then unique else unique.push issue

/-- Compose graph and declaration-linter success without allowing either result to mask the other. -/
def exitCode (graphPassed declarationLintersPassed : Bool) : UInt32 :=
  if graphPassed && declarationLintersPassed then 0 else 1

end ModelLint.ImportGraph
