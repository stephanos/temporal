import ModelLint.ImportGraph
import Tools.LeanImportGraphTests
import Tools.LeanSourceInventoryTests
import Tools.LeanImportGraph.Metadata

/-! Executable synthetic regressions for model import-graph policy and inventory checking. -/

open ModelLint.ImportGraph

private def moduleRecord (name : Lean.Name) (imports : Array Lean.Name := #[]) : ModuleRecord :=
  { name, imports }

private def sourceRecord
    (module : Lean.Name) (path := s!"{module}.lean") : SourceRecord :=
  { path, module }

private def requireEqual [BEq α] [Repr α] (label : String) (actual expected : α) : IO Unit :=
  unless actual == expected do
    throw <| IO.userError s!"{label}: expected {repr expected}, got {repr actual}"

private def requireViolation
    (label : String)
    (modules : Array ModuleRecord)
    (rule : Rule)
    (path : Array Lean.Name) : IO Unit := do
  let violations := check defaultPolicy modules
  requireEqual label violations.size 1
  let some violation := violations[0]?
    | throw <| IO.userError s!"{label}: missing violation"
  requireEqual s!"{label} rule" violation.rule rule
  requireEqual s!"{label} path" violation.path path

private def requireIncludedViolation
    (label : String)
    (modules : Array ModuleRecord)
    (rule : Rule)
    (path : Array Lean.Name) : IO Unit := do
  let some violation := (check defaultPolicy modules).find? fun violation =>
      violation.rule == rule && violation.path == path
    | throw <| IO.userError s!"{label}: missing {repr rule} violation with path {path}"
  requireEqual s!"{label} source" violation.source path[0]!
  requireEqual s!"{label} destination" violation.destination path.back!

private structure ForbiddenCase where
  label : String
  source : Lean.Name
  destination : Lean.Name
  rule : Rule

private def forbiddenCases : Array ForbiddenCase := #[
  { label := "Shared to Umpire", source := `Shared.Root, destination := `Umpire.Core,
    rule := .sharedIndependence },
  { label := "Shared to Temporal", source := `Shared.Root, destination := `Temporal.Feature.Root,
    rule := .sharedIndependence },
  { label := "Shared to Veil", source := `Shared.Root,
    destination := `Umpire.Verify.Veil.Core,
    rule := .sharedIndependence },
  { label := "Umpire to Temporal", source := `Umpire.Root, destination := `Temporal.Feature.Root,
    rule := .umpireIndependence },
  { label := "Umpire to Verify", source := `Umpire.Root,
    destination := `Temporal.Verify.Root,
    rule := .umpireIndependence },
  { label := "Veil implementation to Temporal", source := `Umpire.Verify.Veil.Core,
    destination := `Temporal.Feature.Root,
    rule := .umpireIndependence },
  { label := "Feature to System", source := `Temporal.Feature.Root,
    destination := `Temporal.System.Root,
    rule := .featureIsolation },
  { label := "Feature to Verify", source := `Temporal.Feature.Root,
    destination := `Temporal.Verify.Root,
    rule := .featureIsolation },
  { label := "Feature to Veil", source := `Temporal.Feature.Root,
    destination := `Umpire.Verify.Veil.Core,
    rule := .featureIsolation },
  { label := "System to Feature", source := `Temporal.System.Root,
    destination := `Temporal.Feature.Root,
    rule := .systemIsolation },
  { label := "Umpire to Veil", source := `Umpire.Root,
    destination := `Umpire.Verify.Veil.Core,
    rule := .verificationIsolation },
  { label := "System to Verify", source := `Temporal.System.Root,
    destination := `Temporal.Verify.Root,
    rule := .verificationIsolation },
  { label := "System to Veil", source := `Temporal.System.Root,
    destination := `Umpire.Verify.Veil.Core,
    rule := .verificationIsolation },
  { label := "Temporal to Verify", source := `Temporal.Root,
    destination := `Temporal.Verify.Root,
    rule := .verificationIsolation },
  { label := "Temporal to Veil", source := `Temporal.Root,
    destination := `Umpire.Verify.Veil.Core,
    rule := .verificationIsolation },
  { label := "model tests to Verify", source := `TemporalModelTests,
    destination := `Temporal.Verify.Root,
    rule := .verificationIsolation },
  { label := "model tests to Veil", source := `TemporalModelTests,
    destination := `Umpire.Verify.Veil.Core,
    rule := .verificationIsolation },
  { label := "tool to Verify", source := `Temporal.Tool.Inspect,
    destination := `Temporal.Verify.Root,
    rule := .verificationIsolation },
  { label := "tool to Veil", source := `Temporal.Tool.Inspect,
    destination := `Umpire.Verify.Veil.Core,
    rule := .verificationIsolation }
]

private def testDirectAndTransitiveRejections : IO Unit := do
  for testCase in forbiddenCases do
    requireViolation s!"{testCase.label} direct"
      #[moduleRecord testCase.source #[testCase.destination], moduleRecord testCase.destination]
      testCase.rule #[testCase.source, testCase.destination]
    requireViolation s!"{testCase.label} transitive"
      #[
        moduleRecord testCase.source #[`ModelLint.Bridge],
        moduleRecord `ModelLint.Bridge #[testCase.destination],
        moduleRecord testCase.destination
      ]
      testCase.rule #[testCase.source, `ModelLint.Bridge, testCase.destination]

private def testAllowedOrdinaryImports : IO Unit := do
  let modules := #[
    moduleRecord `Shared.Root #[`Std.Data.HashMap],
    moduleRecord `Umpire.Root #[`Shared.Root],
    moduleRecord `Temporal.Feature.Root #[`Umpire.Root],
    moduleRecord `Temporal.System.Root #[`Umpire.Root],
    moduleRecord `Temporal.Root #[`Temporal.Feature.Root, `Temporal.System.Root]
  ]
  requireEqual "allowed ordinary imports" (check defaultPolicy modules) #[]

private def testTestpilotIsolation : IO Unit := do
  requireEqual "Testpilot protocol has a distinct class"
    (defaultPolicy.classify? `Testpilot.Protocol)
    (some .testpilot)
  requireEqual "Testpilot tests take model-test precedence"
    (defaultPolicy.classify? `Testpilot.Tests.Protocol)
    (some .modelTests)
  let allowed := #[
    moduleRecord `Testpilot #[`Testpilot.Protocol],
    moduleRecord `Testpilot.Protocol #[`Protobuf]
  ]
  requireEqual "Testpilot protocol imports" (check defaultPolicy allowed) #[]
  for destination in #[`Umpire.Core, `Temporal.Feature.Root] do
    requireViolation s!"Testpilot to {destination} direct"
      #[moduleRecord `Testpilot.Protocol #[destination], moduleRecord destination]
      .testpilotIndependence #[`Testpilot.Protocol, destination]
    requireViolation s!"Testpilot to {destination} transitive"
      #[
        moduleRecord `Testpilot.Protocol #[`ModelLint.Bridge],
        moduleRecord `ModelLint.Bridge #[destination],
        moduleRecord destination
      ]
      .testpilotIndependence #[`Testpilot.Protocol, `ModelLint.Bridge, destination]
  let sources := #[
    sourceRecord `Testpilot.Protocol,
    sourceRecord `Temporal.ImplementationLinkTests.UnclassifiedBridge,
    sourceRecord `Umpire.Core
  ]
  let modules := #[
    moduleRecord `Testpilot.Protocol #[`Temporal.ImplementationLinkTests.UnclassifiedBridge],
    moduleRecord `Temporal.ImplementationLinkTests.UnclassifiedBridge #[`Umpire.Core],
    moduleRecord `Umpire.Core
  ]
  requireEqual "unclassified Testpilot bridge"
    (reconcile defaultPolicy sources modules)
    #[.unclassifiedModule `Temporal.ImplementationLinkTests.UnclassifiedBridge]
  requireViolation "unclassified Testpilot bridge preserves forbidden path" modules
    .testpilotIndependence
    #[`Testpilot.Protocol, `Temporal.ImplementationLinkTests.UnclassifiedBridge, `Umpire.Core]

private def testOrdinaryNexusFacadeIsolation : IO Unit := do
  requireViolation "ordinary Nexus facade to Experimental direct"
    #[
      moduleRecord `Temporal.Feature.Nexus #[
        `Temporal.Feature.Nexus.Experimental.AutoClose
      ],
      moduleRecord `Temporal.Feature.Nexus.Experimental.AutoClose
    ]
    .nexusExperimentalIsolation
    #[`Temporal.Feature.Nexus, `Temporal.Feature.Nexus.Experimental.AutoClose]
  requireIncludedViolation "ordinary Nexus facade to Experimental transitive"
    #[
      moduleRecord `Temporal.Feature.Nexus #[`Temporal.Feature.Nexus.Operations],
      moduleRecord `Temporal.Feature.Nexus.Operations #[
        `Temporal.Feature.Nexus.Experimental.VariationSpace
      ],
      moduleRecord `Temporal.Feature.Nexus.Experimental.VariationSpace
    ]
    .nexusExperimentalIsolation
    #[
      `Temporal.Feature.Nexus,
      `Temporal.Feature.Nexus.Operations,
      `Temporal.Feature.Nexus.Experimental.VariationSpace
    ]
  let explicitExperimentalImports := #[
    moduleRecord `Temporal.Feature.Nexus.Experimental.VariationSpace #[
      `Temporal.Feature.Nexus.Experimental.AutoClose
    ],
    moduleRecord `Temporal.Feature.Nexus.Experimental.AutoClose
  ]
  requireEqual "explicit Experimental entry points remain usable"
    (check defaultPolicy explicitExperimentalImports) #[]

private def testTemporalSharedIsolation : IO Unit := do
  requireEqual "Temporal.Shared has a distinct class"
    (defaultPolicy.classify? `Temporal.Shared.Construction)
    (some .temporalShared)
  let allowed := #[
    moduleRecord `Temporal.Shared.Construction #[
      `Shared.Root,
      `Temporal.Shared.Foundation,
      `Umpire.Shared
    ],
    moduleRecord `Temporal.Shared.Foundation,
    moduleRecord `Umpire.Shared #[`Umpire.Core],
    moduleRecord `Umpire.Core,
    moduleRecord `Shared.Root
  ]
  requireEqual "Temporal.Shared lower-layer imports" (check defaultPolicy allowed) #[]
  for destination in #[
    `Temporal.API.Proto,
    `Temporal.Feature.Root,
    `Temporal.System.Root,
    `Temporal.Tool.Inspect,
    `Temporal.Verify.Root,
    `Umpire.Verify.Veil.Core,
    `TemporalModelTests
  ] do
    requireViolation s!"Temporal.Shared to {destination} direct"
      #[moduleRecord `Temporal.Shared.Construction #[destination], moduleRecord destination]
      .temporalSharedIsolation #[`Temporal.Shared.Construction, destination]
    requireIncludedViolation s!"Temporal.Shared to {destination} transitive"
      #[
        moduleRecord `Temporal.Shared.Construction #[`ModelLint.Bridge],
        moduleRecord `ModelLint.Bridge #[destination],
        moduleRecord destination
      ]
      .temporalSharedIsolation #[`Temporal.Shared.Construction, `ModelLint.Bridge, destination]
  for destination in defaultPolicy.testSupportNamespaces do
    requireViolation s!"Temporal.Shared to {destination} direct"
      #[moduleRecord `Temporal.Shared.Construction #[destination], moduleRecord destination]
      .temporalSharedIsolation #[`Temporal.Shared.Construction, destination]
    requireIncludedViolation s!"Temporal.Shared to {destination} transitive"
      #[
        moduleRecord `Temporal.Shared.Construction #[`Umpire.PolicyTests],
        moduleRecord `Umpire.PolicyTests #[destination],
        moduleRecord destination
      ]
      .temporalSharedIsolation #[`Temporal.Shared.Construction, `Umpire.PolicyTests, destination]

private def testTestSupportIsolation : IO Unit := do
  for destination in defaultPolicy.testSupportNamespaces do
    requireViolation s!"production to {destination} direct"
      #[moduleRecord `Temporal.Feature.Root #[destination], moduleRecord destination]
      .testSupportIsolation #[`Temporal.Feature.Root, destination]
    requireViolation s!"production to {destination} transitive"
      #[
        moduleRecord `Temporal.Feature.Root #[`Temporal.Feature.PolicyTests],
        moduleRecord `Temporal.Feature.PolicyTests #[destination],
        moduleRecord destination
      ]
      .testSupportIsolation
      #[`Temporal.Feature.Root, `Temporal.Feature.PolicyTests, destination]
  requireViolation "Shared production to Shared test support"
    #[moduleRecord `Shared.Root #[`Shared.Test], moduleRecord `Shared.Test]
    .testSupportIsolation #[`Shared.Root, `Shared.Test]
  for source in #[`Umpire.Shared, `Temporal.Tool.GenerateTests] do
    requireViolation s!"{source} to Umpire test support"
      #[moduleRecord source #[`Umpire.Shared.Test], moduleRecord `Umpire.Shared.Test]
      .testSupportIsolation #[source, `Umpire.Shared.Test]
  let allowed := #[
    moduleRecord `Umpire.Model.Tests.Fixtures #[`Umpire.Shared.Test],
    moduleRecord `Umpire.Model.Tests.Validation #[`Umpire.Model.Tests.Fixtures],
    moduleRecord `Umpire.ModelTests #[`Umpire.Model.Tests.Validation],
    moduleRecord `UmpireTests #[`Umpire.ModelTests],
    moduleRecord `Umpire.Lint #[`UmpireTests],
    moduleRecord `Temporal.Feature.Nexus.LifecycleTests #[`Umpire.Shared.Test],
    moduleRecord `Temporal.Tool.GenerateTestsTests #[`Umpire.Shared.Test],
    moduleRecord `Temporal.Tool.GenerateTestsIOTestsMain #[
      `Temporal.Tool.GenerateTestsTests
    ],
    moduleRecord `Umpire.Shared.Test
  ]
  requireEqual "test consumers may reach test support" (check defaultPolicy allowed) #[]

private def testTargetIsolation : IO Unit := do
  let allowed := #[
    moduleRecord `Umpire.Model.Tests.Validation #[`Umpire.Model],
    moduleRecord `Umpire.Model #[`Umpire.Model.Types],
    moduleRecord `Umpire.Model.Types #[`Umpire.Core],
    moduleRecord `Umpire.Core
  ]
  requireEqual "Target-owned imports" (check defaultPolicy allowed) #[]
  let destinations := #[
    `Umpire.Query.Language,
    `Umpire.Planning.Engine,
    `Umpire.Artifact,
    `Umpire.Runtime.Driver,
    `Umpire.Verify.Core,
    `Temporal.Feature.Nexus.Lifecycle
  ]
  for source in #[`Umpire.Model, `Umpire.Model.Tests.Validation] do
    for destination in destinations do
      requireViolation s!"{source} to {destination} direct"
        #[moduleRecord source #[destination], moduleRecord destination]
        .modelIsolation #[source, destination]
      requireViolation s!"{source} to {destination} transitive"
        #[
          moduleRecord source #[`ModelLint.Bridge],
          moduleRecord `ModelLint.Bridge #[destination],
          moduleRecord destination
        ]
        .modelIsolation #[source, `ModelLint.Bridge, destination]

private def testSemanticTargetIsolation : IO Unit := do
  let roots := #[
    `Umpire.Model.Check,
    `Umpire.Property.Language,
    `Umpire.Property.Check,
    `Umpire.Property.Trace,
    `Umpire.Property.Evaluation,
    `Umpire.Property.Scoped,
    `Umpire.Property.Scoped.Kernel,
    `Umpire.Property.Scoped.Reference,
    `Umpire.Behavior.Language,
    `Umpire.Query.Language,
    `Umpire.Planning,
    `Umpire.Planning.Types,
    `Umpire.Planning.Engine,
    `Umpire.Planning.CaseAnalysis,
    `Umpire.Artifact.Types,
    `Umpire.Artifact.Codecs,
    `Umpire.Artifact.Planning
  ]
  for root in roots do
    for destination in #[`Umpire.Model.Elab, `Lean.Elab.Term] do
      let direct := check defaultPolicy #[moduleRecord root #[destination]]
      requireEqual s!"{root} rejects missing-record endpoint {destination}"
        (direct.map (·.path)) #[#[root, destination]]
      for bridge in #[`ModelLint.Bridge, `External.Wrapper] do
        let modules := #[
          moduleRecord root #[bridge],
          moduleRecord bridge #[destination],
          moduleRecord destination
        ]
        requireEqual s!"{root} rejects {bridge} to {destination}"
          ((check defaultPolicy modules).map (·.path)) #[#[root, bridge, destination]]
  let allowed := #[
    moduleRecord `Umpire.Property.Evaluation #[`Umpire.Model.Check],
    moduleRecord `Umpire.Model.Check #[`External.Pure],
    moduleRecord `External.Pure #[`Lean.Data.Json],
    moduleRecord `Lean.Data.Json,
    moduleRecord `Umpire.Model #[`Umpire.Model.Types],
    moduleRecord `Umpire.Model.Types #[`Umpire.Model.Elab],
    moduleRecord `Umpire.Model.Elab #[`Lean.Elab.Term],
    moduleRecord `Lean.Elab.Term,
    moduleRecord `Umpire.Property.Authoring #[`Umpire.Model],
    moduleRecord `Umpire.Property #[`Umpire.Property.Authoring],
    moduleRecord `Umpire.Property.Tests.Fixtures #[`Umpire.Property],
    moduleRecord `Umpire.Behavior.Authoring #[`Umpire.Model],
    moduleRecord `Umpire.Behavior #[`Umpire.Behavior.Authoring],
    moduleRecord `Umpire.Query.Authoring #[`Umpire.Model],
    moduleRecord `Umpire.Query #[`Umpire.Query.Authoring]
  ]
  requireEqual "pure imports and explicit authoring remain allowed"
    (check defaultPolicy allowed) #[]
  let cyclic := #[
    moduleRecord `Umpire.Planning.Engine #[`External.Zed, `External.Alpha],
    moduleRecord `External.Zed #[`Lean.Elab.Term],
    moduleRecord `External.Alpha #[`External.Zed, `Umpire.Planning.Engine, `Lean.Elab.Term]
  ]
  let expected := #[#[`Umpire.Planning.Engine, `External.Alpha, `Lean.Elab.Term]]
  requireEqual "external cycles retain stable shortest path"
    ((check defaultPolicy cyclic).map (·.path)) expected
  requireEqual "metadata order does not select the path"
    ((check defaultPolicy cyclic.reverse).map (·.path)) expected

private def testSemanticInventoryIsolation : IO Unit := do
  for source in #[
    `Umpire,
    `Umpire.NewHelper,
    `Umpire.SemanticInventoryHelper,
    `Umpire.Planning.Engine,
    `Umpire.Observation.Evaluation.Types,
    `Umpire.Observation.Verdict,
    `Umpire.Artifact.Runtime,
    `Umpire.Artifact.Result,
    `Umpire.ImplementationLink.Application,
    `Umpire.Verify.Veil.Core
  ] do
    for destination in #[`Umpire.SemanticInventory, `Umpire.SemanticInventory.Types] do
      let direct := check defaultPolicy #[moduleRecord source #[destination]]
      requireEqual s!"{source} rejects inventory without endpoint metadata"
        (direct.map (·.render))
        #[s!"[model-import-graph/semantic-inventory-isolation] forbidden qualified import path: \
          {source} -> {destination}"]
      for bridge in #[
        `Umpire.NewHelper.Bridge,
        `Umpire.Artifact.Result,
        `Umpire.ImplementationLink,
        `ModelLint.Bridge,
        `External.Wrapper,
        `Umpire.Planning.Tests.KnownGaps,
        `Umpire.Shared.Test
      ] do
        if source == bridge then continue
        let modules := #[
          moduleRecord source #[bridge],
          moduleRecord bridge #[destination],
          moduleRecord destination
        ]
        requireEqual s!"{source} rejects inventory through {bridge}"
          ((check defaultPolicy modules).any (·.path == #[source, bridge, destination])) true
  let allowed := #[
    moduleRecord `Umpire.SemanticInventory #[
      `Umpire.SemanticInventory.Types, `Umpire.SemanticInventory.KnownGaps,
      `Umpire.Planning.Engine, `Umpire.Artifact.Runtime, `Umpire.Artifact.Result,
      `Umpire.Observation.Verdict, `Umpire.ImplementationLink.Application
    ],
    moduleRecord `Umpire.SemanticInventory.Types #[`Umpire.OutcomeClassification],
    moduleRecord `Umpire.SemanticInventory.KnownGaps #[`Umpire.KnownGap],
    moduleRecord `Umpire.Planning.Tests.KnownGaps #[`Umpire.SemanticInventory.KnownGaps],
    moduleRecord `Umpire.InventoryTests #[`Umpire.SemanticInventory],
    moduleRecord `Umpire.SemanticInventory.Tests.PlanningRuntime #[`Umpire.SemanticInventory],
    moduleRecord `Umpire.Shared.Test #[`Umpire.SemanticInventory],
    moduleRecord `UmpireTests #[`Umpire.Planning.Tests.KnownGaps],
    moduleRecord `Umpire.Lint #[`UmpireTests],
    moduleRecord `Temporal.Tool.SemanticInventory #[`Umpire.SemanticInventory],
    moduleRecord `Umpire.Planning.Engine #[`Umpire.OutcomeClassification],
    moduleRecord `Umpire.Artifact.Runtime #[`Umpire.OutcomeClassification],
    moduleRecord `Umpire.Artifact.Result #[`Umpire.KnownGap],
    moduleRecord `Umpire.Observation.Verdict #[`Umpire.OutcomeClassification],
    moduleRecord `Umpire.ImplementationLink.Application #[`Umpire.OutcomeClassification],
    moduleRecord `Umpire.KnownGap,
    moduleRecord `Umpire.OutcomeClassification #[`Init.Data.List.Basic]
  ]
  requireEqual "inventory consumes owners and dedicated tests consume inventory"
    (check defaultPolicy allowed) #[]

private def testOutcomeClassificationIsolation : IO Unit := do
  let source := `Umpire.OutcomeClassification
  for destination in #[
    `Umpire, `Umpire.Core, `Umpire.KnownGap, `Umpire.Planning.Engine,
    `Umpire.Artifact.Runtime, `Umpire.Artifact.Result,
    `Umpire.Observation.Evaluation.Types, `Umpire.Observation.Verdict,
    `Umpire.ImplementationLink.Application,
    `Umpire.SemanticInventory, `Umpire.SemanticInventory.Types,
    `Umpire.OutcomeClassification.Helper, `Shared.Root,
    `Temporal.Feature.Root, `Temporal.Tool.SemanticInventory, `ModelLint,
    `Lean, `Std, `Batteries, `External.Wrapper, `InitExtra
  ] do
    requireEqual s!"neutral classification rejects {destination} without endpoint metadata"
      ((check defaultPolicy #[moduleRecord source #[destination]]).map (·.render))
      #[s!"[model-import-graph/outcome-classification-isolation] forbidden qualified import path: \
        {source} -> {destination}"]
    for bridge in #[`Init.Data.List.Basic, `External.Bridge, `Umpire.PolicyTests] do
      let modules := #[
        moduleRecord source #[bridge],
        moduleRecord bridge #[destination],
        moduleRecord destination
      ]
      requireEqual s!"neutral classification rejects {destination} through {bridge}"
        ((check defaultPolicy modules).any (·.path == #[source, bridge, destination])) true
  let allowed := #[
    moduleRecord source #[`Init, `Init.Data.List.Basic],
    moduleRecord `Init #[`Init.Prelude],
    moduleRecord `Init.Data.List.Basic #[`Init.Prelude],
    moduleRecord `Init.Prelude,
    moduleRecord `Umpire.ImportTests #[source, `Umpire.Core],
    moduleRecord `Umpire.Core
  ]
  requireEqual "neutral classification uses only Init foundation; import tests remain consumers"
    (check defaultPolicy allowed) #[]

private def testInventoryBoundaryPaths : IO Unit := do
  for (source, destination) in #[
    (`Umpire.NewFacade, `Umpire.SemanticInventory.Types),
    (`Umpire.OutcomeClassification, `Umpire.Core)
  ] do
    let modules := #[
      moduleRecord source #[`External.Zed, `External.Long, `External.Alpha],
      moduleRecord `External.Zed #[destination],
      moduleRecord `External.Long #[`External.Zed],
      moduleRecord `External.Alpha #[source, `External.Zed, destination],
      moduleRecord destination
    ]
    let paths := fun records => (check defaultPolicy records).filterMap fun violation =>
      if violation.source == source && violation.destination == destination then
        some violation.path
      else none
    let expected := #[#[source, `External.Alpha, destination]]
    requireEqual s!"{source} cycle-safe shortest path" (paths modules) expected
    requireEqual s!"{source} metadata-order independent path" (paths modules.reverse) expected
    let direct := modules.map fun record =>
      if record.name == source then { record with imports := record.imports.push destination }
      else record
    requireEqual s!"{source} direct path wins" (paths direct) #[#[source, destination]]

private def testExternalMetadataReconciliation : IO Unit := do
  let sources := #[sourceRecord `Umpire.Property.Language]
  let modules := #[
    moduleRecord `Umpire.Property.Language #[`External.Wrapper],
    moduleRecord `External.Wrapper #[`Umpire.Model.Missing]
  ]
  requireEqual "external metadata keeps unknown owned imports visible"
    (reconcile defaultPolicy sources modules)
    #[.unknownFirstPartyImport `External.Wrapper `Umpire.Model.Missing]
  requireEqual "external metadata cannot supply a missing owned record"
    (reconcile defaultPolicy (sources.push (sourceRecord `Umpire.Model.Missing)) modules)
    #[
      .uncoveredSource `Umpire.Model.Missing "Umpire.Model.Missing.lean",
      .unknownFirstPartyImport `External.Wrapper `Umpire.Model.Missing
    ]

private def testExactImplementationLinkExceptions : IO Unit := do
  let allowed := #[
    moduleRecord `Temporal.System.Nexus.ImplementationLink #[
      `Temporal.Feature.Nexus.Root,
      `Temporal.System.Nexus.Core
    ],
    moduleRecord `Temporal.ImplementationLinkTests.Nexus #[
      `Temporal.Feature.Nexus.Root,
      `Temporal.System.Nexus.ImplementationLink
    ],
    moduleRecord `Temporal.System.Nexus.Core,
    moduleRecord `Temporal.Feature.Nexus.Root
  ]
  requireEqual "exact Implementation Link composition" (check defaultPolicy allowed) #[]
  requireEqual "composed test has a distinct exact class"
    (defaultPolicy.classify? `Temporal.ImplementationLinkTests.Nexus)
    (some .temporalImplementationLinkTest)
  requireEqual "composed test is not base System"
    (defaultPolicy.classify? `Temporal.ImplementationLinkTests.Nexus ==
      some .temporalSystem)
    false
  for nearMiss in #[
    `Temporal.System.Nexus.ImplementationLink.Extra,
    `Temporal.System.Nexus.ImplementationLinkSibling,
    `Temporal.System.Nexus.ImplementationLinkTests,
    `Temporal.System.Nexus.Other
  ] do
    requireViolation s!"Implementation Link System near miss {nearMiss}"
      #[
        moduleRecord nearMiss #[`Temporal.Feature.Nexus.Root],
        moduleRecord `Temporal.Feature.Nexus.Root
      ]
      .systemIsolation
      #[nearMiss, `Temporal.Feature.Nexus.Root]
  for nearMiss in #[
    `Temporal.ImplementationLinkTests.Nexus.Extra,
    `Temporal.ImplementationLinkTests.NexusExtra,
    `Temporal.ImplementationLinkTests.Other
  ] do
    requireEqual s!"composed-test near miss {nearMiss}"
      (reconcile defaultPolicy #[sourceRecord nearMiss] #[moduleRecord nearMiss])
      #[.unclassifiedModule nearMiss]

private def testExactVerifyExceptions : IO Unit := do
  let verifyDestinations := #[`Temporal.Verify.Nexus.Root, `Umpire.Verify.Veil.Core]
  for consumer in defaultPolicy.verifyConsumers do
    let modules := #[
      moduleRecord consumer verifyDestinations,
      moduleRecord verifyDestinations[0]!,
      moduleRecord verifyDestinations[1]!
    ]
    requireEqual s!"exact verification consumer {consumer}" (check defaultPolicy modules) #[]
  for nearMiss in #[`Temporal.Tool.VerifyVeil.Extra] do
    let modules := #[moduleRecord nearMiss #[`Umpire.Verify.Veil.Core],
      moduleRecord `Umpire.Verify.Veil.Core]
    let violations := check defaultPolicy modules
    requireEqual s!"verification near miss {nearMiss}" violations.size 1
  for unclassifiedNearMiss in #[`TemporalVerify.Extra, `TemporalVeilTests.Extra] do
    let sources := #[sourceRecord unclassifiedNearMiss]
    let modules := #[moduleRecord unclassifiedNearMiss]
    requireEqual s!"aggregate near miss {unclassifiedNearMiss}"
      (reconcile defaultPolicy sources modules)
      #[.unclassifiedModule unclassifiedNearMiss]

private def testModelInventoryPolicy : IO Unit := do
  requireEqual "experimental test aggregate classified"
    (reconcile defaultPolicy #[sourceRecord `TemporalExperimentalTests]
      #[moduleRecord `TemporalExperimentalTests])
    #[]
  requireEqual "unclassified source"
    (reconcile defaultPolicy #[sourceRecord `Unknown.Root] #[moduleRecord `Unknown.Root])
    #[.unclassifiedModule `Unknown.Root]
  requireEqual "unknown first-party import"
    (reconcile defaultPolicy #[sourceRecord `Temporal.Root]
      #[moduleRecord `Temporal.Root #[`Temporal.Future]])
    #[.unknownFirstPartyImport `Temporal.Root `Temporal.Future]
  requireEqual "unknown near-miss aggregate import"
    (reconcile defaultPolicy #[sourceRecord `Temporal.Root]
      #[moduleRecord `Temporal.Root #[`TemporalVerify.Extra]])
    #[.unknownFirstPartyImport `Temporal.Root `TemporalVerify.Extra]

private def testExternalLeaves : IO Unit := do
  let sources := #[sourceRecord `Shared.Root]
  let modules := #[moduleRecord `Shared.Root #[`Lean.Data.Name, `Std.Data.HashMap]]
  requireEqual "external inventory leaves" (reconcile defaultPolicy sources modules) #[]
  requireEqual "external graph leaves" (check defaultPolicy modules) #[]

private def testStableShortestPath : IO Unit := do
  let modules := #[
    moduleRecord `Temporal.Feature.Root #[`ModelLint.BridgeB, `ModelLint.BridgeA],
    moduleRecord `ModelLint.BridgeA #[`Temporal.System.Target],
    moduleRecord `ModelLint.BridgeB #[`Temporal.System.Target],
    moduleRecord `Temporal.System.Target
  ]
  requireViolation "equal shortest paths" modules .featureIsolation
    #[`Temporal.Feature.Root, `ModelLint.BridgeA, `Temporal.System.Target]

private def testMultipleFindings : IO Unit := do
  let modules := #[
    moduleRecord `Temporal.Feature.Root #[`Temporal.System.Zed, `Temporal.System.Alpha],
    moduleRecord `Temporal.System.Alpha,
    moduleRecord `Temporal.System.Zed
  ]
  let violations := check defaultPolicy modules
  requireEqual "multiple findings count" violations.size 2
  requireEqual "multiple findings order" (violations.map (·.destination))
    #[`Temporal.System.Alpha, `Temporal.System.Zed]

private def testCyclesTerminate : IO Unit := do
  let modules := #[
    moduleRecord `Temporal.Feature.Root #[`ModelLint.BridgeA],
    moduleRecord `ModelLint.BridgeA #[`ModelLint.BridgeB],
    moduleRecord `ModelLint.BridgeB #[`ModelLint.BridgeA, `Temporal.System.Target],
    moduleRecord `Temporal.System.Target
  ]
  requireViolation "cycle-safe traversal" modules .featureIsolation
    #[`Temporal.Feature.Root, `ModelLint.BridgeA, `ModelLint.BridgeB, `Temporal.System.Target]

private def controlledViolations : Array Violation :=
  check defaultPolicy #[
    moduleRecord `Shared.Root #[`ModelLint.Bridge],
    moduleRecord `ModelLint.Bridge #[`Umpire.Core],
    moduleRecord `Umpire.Core
  ]

private unsafe def testExternalMetadataClosure : IO Unit := do
  let (modules, regions) ← Tools.LeanImportGraph.Metadata.load #[`Lean.Elab.Command]
  requireEqual "actual external wrapper loads transitive Term metadata"
    (modules.any (·.name == `Lean.Elab.Term)) true
  let root := moduleRecord `Umpire.Property.Language #[`Lean.Elab.Command]
  let violations := check defaultPolicy (modules.push root)
  requireEqual "actual external wrapper is rejected"
    (violations.map (·.destination)) #[`Lean.Elab.Term]
  requireEqual "actual external closure contains unique records"
    (modules.map (·.name)).size
    ((modules.map (·.name)).foldl (init := ({} : Std.HashSet Lean.Name))
      fun names name => names.insert name).size
  let (incomplete, incompleteRegions) ← Tools.LeanImportGraph.Metadata.load
    #[`Lean.Elab.Command] (· == `Lean.Elab.Term)
  requireEqual "cached metadata cannot fill an uninventoried owned import"
    (incomplete.any (·.name == `Lean.Elab.Term)) false
  requireEqual "missing owned metadata still exposes the forbidden endpoint"
    ((check defaultPolicy (incomplete.push root)).map (·.destination)) #[`Lean.Elab.Term]
  let missingFailed ← try
    let _ ← Tools.LeanImportGraph.Metadata.load #[`ModelLint.MissingOwnedMetadata]
    pure false
  catch _ => pure true
  requireEqual "missing source metadata fails loading" missingFailed true
  let _loadedRegionCount := regions.size + incompleteRegions.size

private unsafe def runSyntheticSuite : IO UInt32 := do
  Tools.LeanImportGraphTests.run
  Tools.LeanSourceInventoryTests.run
  testAllowedOrdinaryImports
  testTestpilotIsolation
  testOrdinaryNexusFacadeIsolation
  testTemporalSharedIsolation
  testTestSupportIsolation
  testTargetIsolation
  testSemanticTargetIsolation
  testSemanticInventoryIsolation
  testOutcomeClassificationIsolation
  testInventoryBoundaryPaths
  testExternalMetadataReconciliation
  testExternalMetadataClosure
  testDirectAndTransitiveRejections
  testExactImplementationLinkExceptions
  testExactVerifyExceptions
  testModelInventoryPolicy
  testExternalLeaves
  testStableShortestPath
  testMultipleFindings
  testCyclesTerminate
  IO.println "-- Model import-graph synthetic tests passed."
  pure 0

private def runControlledViolation : IO UInt32 := do
  for violation in controlledViolations do
    IO.eprintln violation.render
  pure <| exitCode controlledViolations.isEmpty true

unsafe def main (args : List String) : IO UInt32 :=
  match args with
  | [] => runSyntheticSuite
  | ["--controlled-violation"] => runControlledViolation
  | _ => do
      IO.eprintln "usage: modelLintTests [--controlled-violation]"
      pure 2
