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
  | testpilot
  | umpire
  | temporalShared
  | temporalFeature
  | temporalSystem
  | temporalTool
  | temporal
  | modelTests
  | lintInfrastructure
  deriving Repr, BEq

/-- A qualified prefix and the class it assigns. Earlier entries take precedence. -/
structure Classifier where
  modulePrefix : Lean.Name
  moduleClass : ModuleClass
  exact : Bool := false
  deriving Repr, BEq

/-- Explicit module classes and exact reviewed import exceptions. -/
structure Policy where
  firstPartyRoots : Array Lean.Name
  classifiers : Array Classifier
  implementationLinkConsumers : Array Lean.Name
  testSupportNamespaces : Array Lean.Name
  testConsumerModules : Array Lean.Name
  /-- Exact production entry points whose full closure must remain free of Target elaboration. -/
  semanticRoots : Array Lean.Name
  deriving Repr, BEq

/-- The import-boundary rules enforced by the checker. -/
inductive Rule where
  | sharedIndependence
  | testpilotIndependence
  | umpireIndependence
  | modelIsolation
  | semanticModelIsolation
  | inventoryIsolation
  | outcomeClassificationIsolation
  | temporalSharedIsolation
  | featureIsolation
  | nexusExperimentalIsolation
  | systemIsolation
  | testSupportIsolation
  deriving Repr, BEq

/-- One forbidden reachability result and its selected shortest qualified path. -/
abbrev Violation := Tools.LeanImportGraph.Violation Rule

/-- A fail-closed discrepancy between owned sources and loaded module metadata. -/
abbrev InventoryIssue := Tools.LeanSourceInventory.InventoryIssue

private def matchesPrefix (modulePrefix name : Lean.Name) : Bool :=
  modulePrefix == name || modulePrefix.isPrefixOf name

/-- Classify a qualified module name, or return `none` when no explicit class is authorized. -/
def Policy.classify? (policy : Policy) (name : Lean.Name) : Option ModuleClass :=
  match policy.classifiers.find? fun classifier =>
      classifier.exact && classifier.modulePrefix == name with
  | some classifier => some classifier.moduleClass
  | none =>
      (policy.classifiers.find? fun classifier =>
        !classifier.exact && matchesPrefix classifier.modulePrefix name).map (·.moduleClass)

/-- Whether a qualified module name belongs to the first-party policy. -/
def Policy.isFirstParty (policy : Policy) (name : Lean.Name) : Bool :=
  policy.firstPartyRoots.any (matchesPrefix · name)

/-- The closed import-boundary policy for the current and planned model module roots. -/
def defaultPolicy : Policy := {
  firstPartyRoots := #[
    `ModelLint,
    `Shared,
    `Testpilot,
    `Temporal,
    `TemporalExperimentalTests,
    `TemporalModelTests,
    `Tools,
    `Umpire,
    `UmpireTests
  ],
  classifiers := #[
    { modulePrefix := `ModelLint, moduleClass := .lintInfrastructure },
    { modulePrefix := `Shared, moduleClass := .shared },
    { modulePrefix := `Testpilot.Tests, moduleClass := .modelTests },
    { modulePrefix := `Testpilot, moduleClass := .testpilot },
    { modulePrefix := `Temporal.Shared, moduleClass := .temporalShared },
    { modulePrefix := `Temporal.Feature, moduleClass := .temporalFeature },
    { modulePrefix := `Temporal.System, moduleClass := .temporalSystem },
    { modulePrefix := `Temporal.Tool, moduleClass := .temporalTool },
    { modulePrefix := `Temporal, moduleClass := .temporal },
    { modulePrefix := `TemporalExperimentalTests, moduleClass := .modelTests, exact := true },
    { modulePrefix := `TemporalModelTests, moduleClass := .modelTests },
    { modulePrefix := `Tools, moduleClass := .lintInfrastructure },
    { modulePrefix := `Umpire, moduleClass := .umpire },
    { modulePrefix := `UmpireTests, moduleClass := .modelTests }
  ],
  implementationLinkConsumers := #[`Temporal.System.Nexus.ImplementationLink],
  testSupportNamespaces := #[
    `Shared.Test,
    `Temporal.Shared.Test,
    `Umpire.Shared.Test
  ],
  testConsumerModules := #[
    `Temporal.Lint,
    `Temporal.Tool.GenerateTestsIOTestsMain,
    `Temporal.Tool.Goldens,
    `Umpire.Lint
  ],
  semanticRoots := #[
    `Umpire.Model.Check,
    `Umpire.Property,
    `Umpire.Property.Check,
    `Umpire.Property.Evaluate,
    `Umpire.Property.Correlated,
    `Umpire.Property.Correlated.Kernel,
    `Umpire.Property.Correlated.Reference,
    `Umpire.Scenario,
    `Umpire.Scenario.Check,
    `Umpire.Query,
    `Umpire.Query.Check,
    `Umpire.Search,
    `Umpire.Search.Types,
    `Umpire.Search.Branches,
    `Umpire.Artifact.Types,
    `Umpire.Artifact.Codecs,
    `Umpire.Artifact.Planning
  ]
}

private def Rule.label : Rule → String
  | .sharedIndependence => "shared-independence"
  | .testpilotIndependence => "testpilot-independence"
  | .umpireIndependence => "umpire-independence"
  | .modelIsolation => "model-isolation"
  | .semanticModelIsolation => "semantic-model-isolation"
  | .inventoryIsolation => "inventory-isolation"
  | .outcomeClassificationIsolation => "outcome-classification-isolation"
  | .temporalSharedIsolation => "temporal-shared-isolation"
  | .featureIsolation => "feature-isolation"
  | .nexusExperimentalIsolation => "nexus-experimental-isolation"
  | .systemIsolation => "system-isolation"
  | .testSupportIsolation => "test-support-isolation"

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
  | .temporalShared | .temporalFeature | .temporalSystem
  | .temporalTool | .temporal => true
  | _ => false

private def nameHasComponent (name : Lean.Name) (component : String) : Bool :=
  match name with
  | .anonymous => false
  | .str parent value => value == component || nameHasComponent parent component
  | .num parent _ => nameHasComponent parent component

private def nameEndsWithTests : Lean.Name → Bool
  | .str _ component => component.endsWith "Tests"
  | _ => false

private def Policy.isTestSupportModule (policy : Policy) (name : Lean.Name) : Bool :=
  policy.testSupportNamespaces.any (matchesPrefix · name)

private def Policy.isTestConsumer
    (policy : Policy) (name : Lean.Name) (moduleClass : ModuleClass) : Bool :=
  moduleClass == .modelTests || policy.testConsumerModules.contains name || nameHasComponent name "Tests" ||
    (name != `Temporal.Tool.GenerateTests && nameEndsWithTests name)

private def Policy.isProductionModule
    (policy : Policy) (name : Lean.Name) (moduleClass : ModuleClass) : Bool :=
  !policy.isTestSupportModule name && !policy.isTestConsumer name moduleClass

private def isAllowedTemporalSharedDestination : ModuleClass → Bool
  | .shared | .umpire | .temporalShared => true
  | _ => false

private def isModelModule (name : Lean.Name) : Bool :=
  matchesPrefix `Umpire.Model name

private def isModelForbiddenDestination (name : Lean.Name) : Bool :=
  #[
    `Umpire.Query,
    `Umpire.Search,
    `Umpire.Artifact,
    `Umpire.Runtime,
    `Temporal
  ].any (matchesPrefix · name)

private def forbiddenRule?
    (policy : Policy)
    (source : Lean.Name)
    (sourceClass : ModuleClass)
    (destination : Lean.Name)
    (destinationClass : ModuleClass) : Option Rule :=
  if matchesPrefix `Umpire source &&
      policy.isProductionModule source sourceClass &&
      !matchesPrefix `Umpire.Inventory source &&
      matchesPrefix `Umpire.Inventory destination then
    some .inventoryIsolation
  else if source == `Temporal.Feature.Nexus &&
      matchesPrefix `Temporal.Feature.Nexus.Experimental destination then
    some .nexusExperimentalIsolation
  else if isModelModule source && isModelForbiddenDestination destination then
    some .modelIsolation
  else if sourceClass == .shared &&
      (destinationClass == .umpire || isTemporalClass destinationClass) then
    some .sharedIndependence
  else if sourceClass == .testpilot &&
      (destinationClass == .umpire || isTemporalClass destinationClass) then
    some .testpilotIndependence
  else if sourceClass == .umpire && isTemporalClass destinationClass then
    some .umpireIndependence
  else if sourceClass == .temporalShared &&
      (policy.isTestSupportModule destination ||
        !isAllowedTemporalSharedDestination destinationClass) then
    some .temporalSharedIsolation
  else if sourceClass == .temporalFeature && destinationClass == .temporalSystem then
    some .featureIsolation
  else if sourceClass == .temporalSystem && destinationClass == .temporalFeature &&
      !policy.implementationLinkConsumers.contains source then
    some .systemIsolation
  else if policy.isProductionModule source sourceClass &&
      policy.isTestSupportModule destination then
    some .testSupportIsolation
  else
    none

/--
Return every forbidden transitive reachability result in deterministic order.

For the owned-only inventory projection:
The caller must first reconcile inventory and metadata. Imports outside the first-party policy are
external leaves and are intentionally not traversed.

Complete checking supplies reachable external metadata as well, so external wrappers participate
in the same traversal. Missing records still expose their endpoint to the policy.
-/
def check (policy : Policy) (modules : Array ModuleRecord) : Array Violation :=
  Tools.LeanImportGraph.check (fun source destination =>
    if source == `Umpire.OutcomeClassification && !matchesPrefix `Init destination then
      some .outcomeClassificationIsolation
    else if policy.semanticRoots.contains source &&
        (matchesPrefix `Umpire.Model.Elab destination || destination == `Lean.Elab.Term) then
      some .semanticModelIsolation
    else
      match policy.classify? source, policy.classify? destination with
      | some sourceClass, some destinationClass =>
          forbiddenRule? policy source sourceClass destination destinationClass
      | _, _ => none) modules policy.isFirstParty

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

Reachable external records need no first-party class. Owned source classification and missing
owned metadata checks still apply, including owned imports reached through external wrappers.
-/
def reconcile
    (policy : Policy)
    (sources : Array SourceRecord)
    (modules : Array ModuleRecord) : Array InventoryIssue :=
  let inventoryPolicy := { policy.inventoryPolicy with
    isClassified := fun name => (policy.classify? name).isSome ||
      (!policy.isFirstParty name && !sources.any (·.module == name))
  }
  Tools.LeanSourceInventory.reconcile inventoryPolicy sources modules

/-- Compose graph and declaration-linter success without allowing either result to mask the other. -/
def exitCode (graphPassed declarationLintersPassed : Bool) : UInt32 :=
  if graphPassed && declarationLintersPassed then 0 else 1

end ModelLint.ImportGraph
