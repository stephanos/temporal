import Tools.LeanSourceInventory

/-!
Pure import-graph policy checking for the Temporal Lean model.

The checker owns model-specific classification, exact exceptions, inventory reconciliation, and
diagnostic language. It delegates reusable cycle-safe traversal and deterministic shortest-path
selection to `Tools.LeanImportGraph`.
-/

namespace ModelLint.ImportGraph

open Tools.LeanSourceInventory

/-- One first-party module and the qualified names it imports directly. -/
abbrev ModuleRecord := Tools.LeanImportGraph.ModuleRecord

/-- A source discovered beneath the canonical model package root. -/
abbrev SourceRecord := Tools.LeanSourceInventory.SourceRecord

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
  firstPartyRoots : Array Lean.Name
  classifiers : Array Classifier
  refinementConsumers : Array Lean.Name
  verifyConsumers : Array Lean.Name
  deriving Repr, BEq

/-- The import-boundary rules enforced by the checker. -/
inductive Rule where
  | sharedIndependence
  | umpireIndependence
  | targetIsolation
  | featureIsolation
  | systemIsolation
  | verificationIsolation
  deriving Repr, BEq

/-- One forbidden reachability result and its selected shortest qualified path. -/
abbrev Violation := Tools.LeanImportGraph.Violation Rule

/-- A fail-closed discrepancy between owned sources and loaded module metadata. -/
abbrev InventoryIssue := Tools.LeanSourceInventory.InventoryIssue

private def matchesPrefix (modulePrefix name : Lean.Name) : Bool :=
  modulePrefix == name || modulePrefix.isPrefixOf name

/-- Classify a qualified module name, or return `none` when the policy does not own it. -/
def Policy.classify? (policy : Policy) (name : Lean.Name) : Option ModuleClass :=
  (policy.classifiers.find? fun classifier =>
    if classifier.exact then classifier.modulePrefix == name
    else matchesPrefix classifier.modulePrefix name).map (·.moduleClass)

/-- Whether a qualified module name belongs to the first-party policy. -/
def Policy.isFirstParty (policy : Policy) (name : Lean.Name) : Bool :=
  policy.firstPartyRoots.any (matchesPrefix · name)

/-- The closed import-boundary policy for the current and planned model module roots. -/
def defaultPolicy : Policy := {
  firstPartyRoots := #[
    `ModelLint,
    `Shared,
    `Temporal,
    `TemporalExperimentalTests,
    `TemporalModelTests,
    `TemporalVeilTests,
    `TemporalVerify,
    `Tools,
    `Umpire,
    `UmpireTests
  ],
  classifiers := #[
    { modulePrefix := `ModelLint, moduleClass := .lintInfrastructure },
    { modulePrefix := `Shared, moduleClass := .shared },
    { modulePrefix := `Temporal.Feature, moduleClass := .temporalFeature },
    { modulePrefix := `Temporal.System, moduleClass := .temporalSystem },
    { modulePrefix := `Temporal.Tool, moduleClass := .temporalTool },
    { modulePrefix := `Temporal.Verify, moduleClass := .temporalVerify },
    { modulePrefix := `Temporal, moduleClass := .temporal },
    { modulePrefix := `TemporalExperimentalTests, moduleClass := .modelTests, exact := true },
    { modulePrefix := `TemporalModelTests, moduleClass := .modelTests },
    { modulePrefix := `TemporalVeilTests, moduleClass := .optInVerify, exact := true },
    { modulePrefix := `TemporalVerify, moduleClass := .optInVerify, exact := true },
    { modulePrefix := `Tools, moduleClass := .lintInfrastructure },
    { modulePrefix := `Umpire.Verify.Veil, moduleClass := .umpireVeil },
    { modulePrefix := `Umpire, moduleClass := .umpire },
    { modulePrefix := `UmpireTests, moduleClass := .modelTests }
  ],
  refinementConsumers := #[`Temporal.System.Nexus.Refinement],
  verifyConsumers := #[
    `Temporal.Feature.Nexus.Experimental.CallerClosure.VeilTests,
    `Temporal.Tool.VerifyVeil,
    `TemporalVeilTests,
    `TemporalVerify
  ]
}

private def Rule.label : Rule → String
  | .sharedIndependence => "shared-independence"
  | .umpireIndependence => "umpire-independence"
  | .targetIsolation => "target-isolation"
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

private def isTemporalClass : ModuleClass → Bool
  | .temporalFeature | .temporalSystem | .temporalVerify | .temporalTool | .temporal => true
  | _ => false

private def isVerifyClass : ModuleClass → Bool
  | .temporalVerify | .umpireVeil => true
  | _ => false

private def isTargetModule (name : Lean.Name) : Bool :=
  matchesPrefix `Umpire.Target name

private def isTargetForbiddenDestination (name : Lean.Name) : Bool :=
  #[
    `Umpire.Query,
    `Umpire.Planning,
    `Umpire.Artifact,
    `Umpire.Runtime,
    `Umpire.Verify,
    `Temporal
  ].any (matchesPrefix · name)

private def forbiddenRule?
    (policy : Policy)
    (source : Lean.Name)
    (sourceClass : ModuleClass)
    (destination : Lean.Name)
    (destinationClass : ModuleClass) : Option Rule :=
  if isTargetModule source && isTargetForbiddenDestination destination then
    some .targetIsolation
  else if sourceClass == .shared &&
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

/--
Return every forbidden transitive reachability result in deterministic order.

The caller must first reconcile inventory and metadata. Imports outside the first-party policy are
external leaves and are intentionally not traversed.
-/
def check (policy : Policy) (modules : Array ModuleRecord) : Array Violation :=
  Tools.LeanImportGraph.check (fun source destination =>
    match policy.classify? source, policy.classify? destination with
    | some sourceClass, some destinationClass =>
        forbiddenRule? policy source sourceClass destination destinationClass
    | _, _ => none) modules

private def Policy.inventoryPolicy (policy : Policy) : InventoryPolicy := {
  isFirstParty := policy.isFirstParty
  isClassified := fun module => (policy.classify? module).isSome
}

/-- Validate source classification and qualified identity before invoking Lake. -/
def validateSources (policy : Policy) (sources : Array SourceRecord) : Array InventoryIssue :=
  Tools.LeanSourceInventory.validateSources policy.inventoryPolicy sources

/--
Reconcile a canonical owned-source inventory with loaded direct-import metadata.

Every discrepancy is retained and sorted, so one lint run reports all independently actionable
inventory failures instead of stopping at the first one.
-/
def reconcile
    (policy : Policy)
    (sources : Array SourceRecord)
    (modules : Array ModuleRecord) : Array InventoryIssue :=
  Tools.LeanSourceInventory.reconcile policy.inventoryPolicy sources modules

/-- Compose graph and declaration-linter success without allowing either result to mask the other. -/
def exitCode (graphPassed declarationLintersPassed : Bool) : UInt32 :=
  if graphPassed && declarationLintersPassed then 0 else 1

end ModelLint.ImportGraph
